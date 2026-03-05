package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/internal/domain/artifacts"

	"github.com/google/uuid"
)

type fakeArtifactStore struct {
	artifacts []*artifacts.Artifact
	approvals map[uuid.UUID]*artifacts.Approval
	reports   []*artifacts.ValidationReport

	listApprovedCalls int
	listArtifactsCalls int

	errListApproved            error
	errListArtifacts           error
	errGetApproval             error
	errGetApprovals            error
	errGetArtifactByID         error
	errCreateValidationReport  error
	errRecordValidationOutcome error
	errUpdateApprovalState     error
}

func newFakeArtifactStore(arts ...*artifacts.Artifact) *fakeArtifactStore {
	store := &fakeArtifactStore{
		artifacts: arts,
		approvals: make(map[uuid.UUID]*artifacts.Approval, len(arts)),
	}
	for _, art := range arts {
		store.approvals[art.ID] = artifacts.NewApproval(art.ID, "tester")
	}
	return store
}

func (f *fakeArtifactStore) ListApprovedArtifacts(ctx context.Context) ([]*artifacts.Artifact, error) {
	f.listApprovedCalls++
	if f.errListApproved != nil {
		return nil, f.errListApproved
	}
	out := make([]*artifacts.Artifact, 0, len(f.artifacts))
	for _, art := range f.artifacts {
		approval := f.approvals[art.ID]
		if approval == nil {
			continue
		}
		state := approval.State
		if state == artifacts.StateApproved || state == artifacts.StateActive {
			out = append(out, art)
		}
	}
	return out, nil
}

func (f *fakeArtifactStore) ListArtifacts(ctx context.Context, stateFilter string) ([]*artifacts.Artifact, error) {
	f.listArtifactsCalls++
	if f.errListArtifacts != nil {
		return nil, f.errListArtifacts
	}
	if stateFilter == "" {
		return append([]*artifacts.Artifact(nil), f.artifacts...), nil
	}
	target := artifacts.ApprovalState(stateFilter)
	out := make([]*artifacts.Artifact, 0, len(f.artifacts))
	for _, art := range f.artifacts {
		approval := f.approvals[art.ID]
		if approval != nil && approval.State == target {
			out = append(out, art)
		}
	}
	return out, nil
}

func (f *fakeArtifactStore) GetApproval(ctx context.Context, artifactID uuid.UUID) (*artifacts.Approval, error) {
	if f.errGetApproval != nil {
		return nil, f.errGetApproval
	}
	ap, ok := f.approvals[artifactID]
	if !ok {
		return nil, context.Canceled
	}
	return ap, nil
}

func (f *fakeArtifactStore) GetApprovals(ctx context.Context, artifactIDs []uuid.UUID) (map[uuid.UUID]*artifacts.Approval, error) {
	if f.errGetApprovals != nil {
		return nil, f.errGetApprovals
	}
	out := make(map[uuid.UUID]*artifacts.Approval, len(artifactIDs))
	for _, artifactID := range artifactIDs {
		if ap, ok := f.approvals[artifactID]; ok {
			out[artifactID] = ap
		}
	}
	return out, nil
}

func (f *fakeArtifactStore) GetArtifactByID(ctx context.Context, id uuid.UUID) (*artifacts.Artifact, error) {
	if f.errGetArtifactByID != nil {
		return nil, f.errGetArtifactByID
	}
	for _, art := range f.artifacts {
		if art.ID == id {
			return art, nil
		}
	}
	return nil, context.Canceled
}

func (f *fakeArtifactStore) CreateArtifact(ctx context.Context, artifact *artifacts.Artifact) error {
	f.artifacts = append(f.artifacts, artifact)
	return nil
}

func (f *fakeArtifactStore) CreateApproval(ctx context.Context, approval *artifacts.Approval) error {
	f.approvals[approval.ArtifactID] = approval
	return nil
}

func (f *fakeArtifactStore) UpdateApprovalState(ctx context.Context, artifactID uuid.UUID, toState artifacts.ApprovalState, promotedBy, reason string) error {
	if f.errUpdateApprovalState != nil {
		return f.errUpdateApprovalState
	}
	ap := f.approvals[artifactID]
	if ap == nil {
		return context.Canceled
	}
	now := time.Now().UTC()
	prev := ap.State
	ap.PreviousState = &prev
	ap.State = toState
	ap.StateChangedBy = promotedBy
	ap.StateChangeReason = reason
	ap.StateChangedAt = now
	ap.UpdatedAt = now
	return nil
}

func (f *fakeArtifactStore) CreateValidationReport(ctx context.Context, report *artifacts.ValidationReport) error {
	if f.errCreateValidationReport != nil {
		return f.errCreateValidationReport
	}
	f.reports = append(f.reports, report)
	return nil
}

func (f *fakeArtifactStore) RecordValidationOutcome(ctx context.Context, artifactID, runID uuid.UUID, passed bool, reportURI string) error {
	if f.errRecordValidationOutcome != nil {
		return f.errRecordValidationOutcome
	}
	ap := f.approvals[artifactID]
	if ap == nil {
		return context.Canceled
	}
	ap.ValidationRunID = &runID
	ap.ValidationPassed = passed
	ap.ValidationReportURI = reportURI
	return nil
}

func makeTestArtifact(t *testing.T, name string) *artifacts.Artifact {
	t.Helper()

	createdAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	art := &artifacts.Artifact{
		ID:            uuid.New(),
		ArtifactID:    "strat_" + name + "_2026-03-01T12:00:00Z",
		SchemaVersion: "1.0.0",
		Strategy: artifacts.StrategyInfo{
			Name:    name,
			Version: "1.0.0",
			Params: map[string]any{
				"period": 14,
			},
		},
		DataWindow: &artifacts.DataWindow{
			From:    createdAt.Add(-24 * time.Hour),
			To:      createdAt,
			Symbols: []string{"AAPL"},
		},
		Validation: &artifacts.ValidationInfo{
			BacktestRunID:   uuid.New(),
			DeterminismSeed: 42,
			Metrics: map[string]any{
				"total_trades":     20,
				"win_rate":         0.65,
				"total_return_pct": 10.0,
				"max_drawdown":     0.05,
				"sharpe_ratio":     1.4,
				"profit_factor":    1.8,
			},
		},
		RiskProfile: artifacts.RiskProfile{
			MaxPositionPct:    0.05,
			MaxDailyLoss:      0.02,
			AllowedOrderTypes: []string{"LMT"},
		},
		CreatedBy: "tester",
		CreatedAt: createdAt,
	}
	hash, err := art.ComputeHash()
	if err != nil {
		t.Fatalf("compute hash: %v", err)
	}
	art.Hash = hash
	return art
}

func TestHandleListArtifacts_DefaultReturnsAllStates(t *testing.T) {
	draft := makeTestArtifact(t, "draft")
	approved := makeTestArtifact(t, "approved")
	store := newFakeArtifactStore(draft, approved)
	store.approvals[approved.ID].State = artifacts.StateApproved

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got []ArtifactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(got))
	}
}

func TestHandleValidateArtifact_PromotesDraftOnSuccessfulValidation(t *testing.T) {
	art := makeTestArtifact(t, "valid")
	store := newFakeArtifactStore(art)

	h := NewArtifactHandlers(store)
	h.now = func() time.Time { return time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC) }
	h.runReplayGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate2",
			"testRunId":   "replay-run-1",
			"status":      "completed",
			"artifactUri": "/reports/gate2/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	h.runPromotionGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate3",
			"testRunId":   "gate-run-1",
			"status":      "completed",
			"artifactUri": "/reports/gate3/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.approvals[art.ID].State != artifacts.StateValidated {
		t.Fatalf("expected state VALIDATED, got %s", store.approvals[art.ID].State)
	}
	if !store.approvals[art.ID].ValidationPassed {
		t.Fatal("expected validation_passed to be true")
	}
	if len(store.reports) != 1 {
		t.Fatalf("expected 1 validation report, got %d", len(store.reports))
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["new_state"] != "VALIDATED" {
		t.Fatalf("expected new_state VALIDATED, got %v", body["new_state"])
	}
	if body["passed"] != true {
		t.Fatalf("expected passed=true, got %v", body["passed"])
	}
	replayRun, ok := body["replayRun"].(map[string]any)
	if !ok {
		t.Fatalf("expected replayRun object, got %T", body["replayRun"])
	}
	if replayRun["testRunId"] != "replay-run-1" {
		t.Fatalf("expected replayRun testRunId replay-run-1, got %v", replayRun["testRunId"])
	}
	gateRun, ok := body["gateRun"].(map[string]any)
	if !ok {
		t.Fatalf("expected gateRun object, got %T", body["gateRun"])
	}
	if gateRun["testRunId"] != "gate-run-1" {
		t.Fatalf("expected gateRun testRunId gate-run-1, got %v", gateRun["testRunId"])
	}
	if store.approvals[art.ID].ValidationReportURI != "/reports/gate3/2026-03-03/summary.md" {
		t.Fatalf("expected validation report URI from gate run, got %q", store.approvals[art.ID].ValidationReportURI)
	}
}

func TestHandleValidateArtifact_DoesNotPromoteFailedArtifact(t *testing.T) {
	art := makeTestArtifact(t, "tampered")
	art.Hash = "deadbeef"
	store := newFakeArtifactStore(art)

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.approvals[art.ID].State != artifacts.StateDraft {
		t.Fatalf("expected state DRAFT, got %s", store.approvals[art.ID].State)
	}
	if store.approvals[art.ID].ValidationPassed {
		t.Fatal("expected validation_passed to remain false")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["new_state"] != "DRAFT" {
		t.Fatalf("expected new_state DRAFT, got %v", body["new_state"])
	}
	if body["passed"] != false {
		t.Fatalf("expected passed=false, got %v", body["passed"])
	}
}

func TestHandleValidateArtifact_DoesNotPromoteWhenGate3Fails(t *testing.T) {
	art := makeTestArtifact(t, "gate-fail")
	store := newFakeArtifactStore(art)

	h := NewArtifactHandlers(store)
	h.runReplayGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate2",
			"testRunId":   "replay-run-2",
			"status":      "completed",
			"artifactUri": "/reports/gate2/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	h.runPromotionGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate3",
			"testRunId":   "gate-run-2",
			"status":      "failed",
			"artifactUri": "/reports/gate3/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "failed",
			},
		}
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.approvals[art.ID].State != artifacts.StateDraft {
		t.Fatalf("expected state DRAFT, got %s", store.approvals[art.ID].State)
	}
	if store.approvals[art.ID].ValidationPassed {
		t.Fatal("expected validation_passed to be false")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["passed"] != false {
		t.Fatalf("expected passed=false, got %v", body["passed"])
	}
}

func TestHandleValidateArtifact_DoesNotPromoteWhenReplayGateFails(t *testing.T) {
	art := makeTestArtifact(t, "replay-fail")
	store := newFakeArtifactStore(art)

	h := NewArtifactHandlers(store)
	h.runReplayGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate2",
			"testRunId":   "replay-run-3",
			"status":      "failed",
			"artifactUri": "/reports/gate2/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "failed",
			},
		}
	}
	h.runPromotionGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"gate":        "Gate3",
			"testRunId":   "gate-run-3",
			"status":      "completed",
			"artifactUri": "/reports/gate3/2026-03-03/summary.md",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.approvals[art.ID].State != artifacts.StateDraft {
		t.Fatalf("expected state DRAFT, got %s", store.approvals[art.ID].State)
	}
	if store.approvals[art.ID].ValidationPassed {
		t.Fatal("expected validation_passed to be false")
	}
}

func TestHandleListArtifacts_FilterApprovedUsesApprovedQuery(t *testing.T) {
	draft := makeTestArtifact(t, "draft-filter")
	approved := makeTestArtifact(t, "approved-filter")
	store := newFakeArtifactStore(draft, approved)
	store.approvals[approved.ID].State = artifacts.StateApproved

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts?state=approved", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.listApprovedCalls != 1 {
		t.Fatalf("expected ListApprovedArtifacts to be called once, got %d", store.listApprovedCalls)
	}
	if store.listArtifactsCalls != 0 {
		t.Fatalf("expected ListArtifacts not to be called, got %d", store.listArtifactsCalls)
	}

	var got []ArtifactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 approved artifact, got %d", len(got))
	}
	if got[0].ID != approved.ID.String() {
		t.Fatalf("expected approved artifact %s, got %s", approved.ID, got[0].ID)
	}
}

func TestHandleListArtifacts_FilterStateUsesStateQuery(t *testing.T) {
	draft := makeTestArtifact(t, "draft-state")
	validated := makeTestArtifact(t, "validated-state")
	store := newFakeArtifactStore(draft, validated)
	store.approvals[validated.ID].State = artifacts.StateValidated

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/artifacts?state=VALIDATED", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.listArtifactsCalls != 1 {
		t.Fatalf("expected ListArtifacts to be called once, got %d", store.listArtifactsCalls)
	}
	if store.listApprovedCalls != 0 {
		t.Fatalf("expected ListApprovedArtifacts not to be called, got %d", store.listApprovedCalls)
	}

	var got []ArtifactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 validated artifact, got %d", len(got))
	}
	if got[0].State != string(artifacts.StateValidated) {
		t.Fatalf("expected state VALIDATED, got %s", got[0].State)
	}
}

func TestHandlePromoteArtifact_InvalidTransitionReturnsBadRequest(t *testing.T) {
	art := makeTestArtifact(t, "invalid-transition")
	store := newFakeArtifactStore(art)
	store.approvals[art.ID].State = artifacts.StateDraft
	store.errUpdateApprovalState = errors.New("invalid state transition")

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/artifacts/"+art.ID.String()+"/promote",
		mustJSONBody(t, map[string]any{
			"to_state":    string(artifacts.StateActive),
			"promoted_by": "reviewer",
			"reason":      "attempt invalid transition",
		}),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandlePromoteArtifact_ReturnsServerErrorWhenArtifactLookupFails(t *testing.T) {
	art := makeTestArtifact(t, "lookup-fail")
	store := newFakeArtifactStore(art)
	store.approvals[art.ID].State = artifacts.StateValidated
	store.errGetArtifactByID = errors.New("artifact lookup failed")

	h := NewArtifactHandlers(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/artifacts/"+art.ID.String()+"/promote",
		mustJSONBody(t, map[string]any{
			"to_state":    string(artifacts.StateApproved),
			"promoted_by": "reviewer",
			"reason":      "approve artifact",
		}),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidateArtifact_ReturnsServerErrorWhenReportPersistenceFails(t *testing.T) {
	art := makeTestArtifact(t, "report-persist-fail")
	store := newFakeArtifactStore(art)
	store.errCreateValidationReport = errors.New("report write failed")

	h := NewArtifactHandlers(store)
	h.runReplayGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"status": "completed",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	h.runPromotionGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"status": "completed",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(store.reports) != 0 {
		t.Fatalf("expected no reports persisted, got %d", len(store.reports))
	}
}

func TestHandleValidateArtifact_ReturnsServerErrorWhenOutcomePersistenceFails(t *testing.T) {
	art := makeTestArtifact(t, "outcome-persist-fail")
	store := newFakeArtifactStore(art)
	store.errRecordValidationOutcome = errors.New("outcome write failed")

	h := NewArtifactHandlers(store)
	h.runReplayGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"status": "completed",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	h.runPromotionGate = func(ctx context.Context) map[string]any {
		return map[string]any{
			"status": "completed",
			"summary": map[string]any{
				"status": "passed",
			},
		}
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/artifacts/"+art.ID.String()+"/validate", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if len(store.reports) != 1 {
		t.Fatalf("expected validation report to be created before failure, got %d", len(store.reports))
	}
}

func mustJSONBody(t *testing.T, payload map[string]any) io.ReadCloser {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json payload: %v", err)
	}
	return io.NopCloser(bytes.NewReader(raw))
}
