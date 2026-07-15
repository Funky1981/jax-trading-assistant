package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeWorldMonitorIngestService struct {
	receipt worldMonitorResearchReceipt
	err     error
	seen    worldMonitorResearchTrigger
}

func (f *fakeWorldMonitorIngestService) Ingest(_ context.Context, trigger worldMonitorResearchTrigger) (worldMonitorResearchReceipt, error) {
	f.seen = trigger
	return f.receipt, f.err
}

type fakeWorldMonitorPromoteService struct {
	result worldMonitorPromotionResult
	err    error
	limit  int
}

func (f *fakeWorldMonitorPromoteService) PromotePending(_ context.Context, limit int) (worldMonitorPromotionResult, error) {
	f.limit = limit
	return f.result, f.err
}

type fakeWorldMonitorStatusService struct {
	status worldMonitorResearchStatus
	err    error
}

func (f *fakeWorldMonitorStatusService) Status(_ context.Context) (worldMonitorResearchStatus, error) {
	return f.status, f.err
}

type fakeWorldMonitorInboxListService struct {
	result worldMonitorResearchInboxList
	err    error
	filter worldMonitorResearchInboxFilter
}

func (f *fakeWorldMonitorInboxListService) List(_ context.Context, filter worldMonitorResearchInboxFilter) (worldMonitorResearchInboxList, error) {
	f.filter = filter
	return f.result, f.err
}

func TestWorldMonitorResearchHandler_AcceptsValidTrigger(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	fake := &fakeWorldMonitorIngestService{
		receipt: worldMonitorResearchReceipt{InboxID: "inbox-1", EventID: "event-1", Status: "new"},
	}
	restore := replaceWorldMonitorIngestServiceFactory(fake)
	defer restore()

	res := performWorldMonitorResearchRequest(t, http.MethodPost, validWorldMonitorResearchTrigger(now))

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusAccepted, res.Body.String())
	}
	if fake.seen.SourceEventID == "" {
		t.Fatal("expected handler to pass decoded trigger to service")
	}
}

func TestWorldMonitorResearchHandler_ReturnsDuplicateAsOK(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	fake := &fakeWorldMonitorIngestService{
		receipt: worldMonitorResearchReceipt{InboxID: "inbox-1", EventID: "event-1", Status: "new", Duplicate: true},
	}
	restore := replaceWorldMonitorIngestServiceFactory(fake)
	defer restore()

	res := performWorldMonitorResearchRequest(t, http.MethodPost, validWorldMonitorResearchTrigger(now))

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}
}

func TestWorldMonitorResearchHandler_ReturnsRejectedAsUnprocessable(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	fake := &fakeWorldMonitorIngestService{
		receipt: worldMonitorResearchReceipt{Status: "rejected", RejectionReason: "source_urls are required"},
	}
	restore := replaceWorldMonitorIngestServiceFactory(fake)
	defer restore()

	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceURLs = nil
	res := performWorldMonitorResearchRequest(t, http.MethodPost, trigger)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusUnprocessableEntity, res.Body.String())
	}
}

func TestWorldMonitorResearchHandler_RejectsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/research/events/world-monitor", nil)
	res := httptest.NewRecorder()

	worldMonitorResearchIngestHandler(nil)(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestWorldMonitorResearchHandler_RejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/research/events/world-monitor", bytes.NewBufferString("{"))
	res := httptest.NewRecorder()

	worldMonitorResearchIngestHandler(nil)(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestWorldMonitorResearchHandler_RejectsTradeLanguage(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	fake := &fakeWorldMonitorIngestService{
		receipt: worldMonitorResearchReceipt{Status: "rejected", RejectionReason: "payload contains trade instruction language"},
	}
	restore := replaceWorldMonitorIngestServiceFactory(fake)
	defer restore()

	trigger := validWorldMonitorResearchTrigger(now)
	trigger.Headline = "Buy QQQ now"
	res := performWorldMonitorResearchRequest(t, http.MethodPost, trigger)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusUnprocessableEntity, res.Body.String())
	}
}

func TestWorldMonitorOpportunityPromoteHandler_RunsPromoter(t *testing.T) {
	candidateID := "00000000-0000-0000-0000-0000000000aa"
	fake := &fakeWorldMonitorPromoteService{
		result: worldMonitorPromotionResult{
			PromotedCount:       1,
			BlockedSkippedCount: 1,
			Skipped:             1,
			Promoted: []worldMonitorPromotedOpportunity{{
				InboxID:     "inbox-1",
				SignalID:    "00000000-0000-0000-0000-0000000000bb",
				CandidateID: candidateID,
				Symbol:      "QQQ",
				Route:       "approval_required",
			}},
			Outcomes: []worldMonitorPromotionOutcome{
				{InboxID: "inbox-0", Symbol: "XLE", Status: "skipped", ReasonCode: "no_enabled_strategy_instance", Reason: "No compatible enabled ETF strategy instance is configured for XLE."},
				{InboxID: "inbox-1", Symbol: "QQQ", Status: "promoted", ReasonCode: "candidate_created", CandidateID: candidateID},
			},
		},
	}
	restore := replaceWorldMonitorPromoteServiceFactory(fake)
	defer restore()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/research/events/world-monitor/promote", nil)
	res := httptest.NewRecorder()

	worldMonitorOpportunityPromoteHandler(nil)(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if fake.limit != 10 {
		t.Fatalf("limit = %d, want 10", fake.limit)
	}
	if !strings.Contains(res.Body.String(), candidateID) {
		t.Fatalf("response missing candidate id: %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"promotedCount":1`) || !strings.Contains(res.Body.String(), "no_enabled_strategy_instance") {
		t.Fatalf("response missing structured counts/reason: %s", res.Body.String())
	}
}

func TestWorldMonitorResearchStatusHandler_ReturnsLatestMonitorState(t *testing.T) {
	receivedAt := time.Date(2026, 6, 12, 10, 30, 0, 0, time.UTC)
	fake := &fakeWorldMonitorStatusService{
		status: worldMonitorResearchStatus{
			Connected:         true,
			LastReceivedAt:    &receivedAt,
			LastSourceEventID: "monitor-event-1",
			LastStatus:        "candidate_created",
			LastHeadline:      "Softer inflation supports growth ETF review",
			LastSymbols:       []string{"QQQ"},
			LastCandidateID:   "candidate-1",
			Counts: worldMonitorResearchStatusCounts{
				Total:             3,
				Pending:           1,
				CandidatesCreated: 1,
				Rejected:          1,
			},
			CheckedAt: receivedAt,
		},
	}
	restore := replaceWorldMonitorStatusServiceFactory(fake)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/research/events/world-monitor/status", nil)
	res := httptest.NewRecorder()

	worldMonitorResearchStatusHandler(nil)(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "monitor-event-1") || !strings.Contains(res.Body.String(), "candidate_created") {
		t.Fatalf("response missing latest monitor state: %s", res.Body.String())
	}
}

func TestWorldMonitorResearchInboxHandler_ReturnsAcceptedAndRejectedRows(t *testing.T) {
	receivedAt := time.Date(2026, 6, 12, 11, 0, 0, 0, time.UTC)
	fake := &fakeWorldMonitorInboxListService{
		result: worldMonitorResearchInboxList{
			Items: []worldMonitorResearchInboxItem{
				{
					ID:                   "inbox-accepted",
					Source:               "world-monitor",
					SourceEventID:        "accepted-1",
					WorldMonitorEventID:  "accepted-1",
					Status:               "candidate_created",
					EventType:            "macro_rates",
					Headline:             "Accepted monitor item",
					SourceURLs:           []string{"https://example.com/accepted"},
					SourceCount:          1,
					EventTime:            receivedAt,
					ReceivedAt:           receivedAt,
					PossibleAffectedETFs: []string{"QQQ"},
					Severity:             "high",
					SourceTier:           "tier2",
					Confidence:           0.8,
					ConfidenceReasons:    []string{"trusted source"},
					MappingReason:        "mapped to QQQ",
					CandidateID:          "candidate-1",
					RawPayload:           map[string]any{"fixture": true},
				},
				{
					ID:              "inbox-rejected",
					Source:          "world-monitor",
					SourceEventID:   "rejected-1",
					Status:          "rejected",
					RejectionReason: "source_urls are required",
					EventType:       "macro_rates",
					Headline:        "Rejected monitor item",
					EventTime:       receivedAt,
					ReceivedAt:      receivedAt,
					Severity:        "high",
					SourceTier:      "tier2",
					MappingReason:   "rejected before mapping",
					RawPayload:      map[string]any{"bad": true},
				},
			},
			Total:     2,
			CheckedAt: receivedAt,
		},
	}
	restore := replaceWorldMonitorInboxListServiceFactory(fake)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/research/events/world-monitor/inbox?status=rejected&limit=25", nil)
	res := httptest.NewRecorder()

	worldMonitorResearchInboxHandler(nil)(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", res.Code, http.StatusOK, res.Body.String())
	}
	if fake.filter.Status != "rejected" || fake.filter.Limit != 25 {
		t.Fatalf("filter = %+v, want rejected limit 25", fake.filter)
	}
	if !strings.Contains(res.Body.String(), "Accepted monitor item") || !strings.Contains(res.Body.String(), "source_urls are required") {
		t.Fatalf("response missing monitor inbox audit data: %s", res.Body.String())
	}
}

func performWorldMonitorResearchRequest(t *testing.T, method string, trigger worldMonitorResearchTrigger) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(trigger)
	if err != nil {
		t.Fatalf("marshal trigger: %v", err)
	}
	req := httptest.NewRequest(method, "/api/v1/research/events/world-monitor", bytes.NewReader(body))
	res := httptest.NewRecorder()
	worldMonitorResearchIngestHandler(nil)(res, req)
	return res
}

func replaceWorldMonitorIngestServiceFactory(fake *fakeWorldMonitorIngestService) func() {
	original := newWorldMonitorResearchIngestService
	newWorldMonitorResearchIngestService = func(_ *pgxpool.Pool) worldMonitorResearchIngestService {
		return fake
	}
	return func() {
		newWorldMonitorResearchIngestService = original
	}
}

func replaceWorldMonitorPromoteServiceFactory(fake *fakeWorldMonitorPromoteService) func() {
	original := newWorldMonitorOpportunityPromoteService
	newWorldMonitorOpportunityPromoteService = func(_ *pgxpool.Pool) worldMonitorOpportunityPromoteService {
		return fake
	}
	return func() {
		newWorldMonitorOpportunityPromoteService = original
	}
}

func replaceWorldMonitorStatusServiceFactory(fake *fakeWorldMonitorStatusService) func() {
	original := newWorldMonitorResearchStatusService
	newWorldMonitorResearchStatusService = func(_ *pgxpool.Pool) worldMonitorResearchStatusService {
		return fake
	}
	return func() {
		newWorldMonitorResearchStatusService = original
	}
}

func replaceWorldMonitorInboxListServiceFactory(fake *fakeWorldMonitorInboxListService) func() {
	original := newWorldMonitorResearchInboxListService
	newWorldMonitorResearchInboxListService = func(_ *pgxpool.Pool) worldMonitorResearchInboxListService {
		return fake
	}
	return func() {
		newWorldMonitorResearchInboxListService = original
	}
}
