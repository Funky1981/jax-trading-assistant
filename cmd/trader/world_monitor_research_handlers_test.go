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
			Promoted: []worldMonitorPromotedOpportunity{{
				InboxID:     "inbox-1",
				SignalID:    "00000000-0000-0000-0000-0000000000bb",
				CandidateID: candidateID,
				Symbol:      "QQQ",
				Route:       "approval_required",
			}},
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
