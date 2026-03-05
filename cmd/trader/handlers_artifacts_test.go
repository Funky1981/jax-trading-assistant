package main

import (
	"context"
	"encoding/json"
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
	out := make([]*artifacts.Artifact, 0, len(f.artifacts))
	for _, art := range f.artifacts {
		state := f.approvals[art.ID].State
		if state == artifacts.StateApproved || state == artifacts.StateActive {
			out = append(out, art)
		}
	}
	return out, nil
}

func (f *fakeArtifactStore) ListArtifacts(ctx context.Context, stateFilter string) ([]*artifacts.Artifact, error) {
	if stateFilter == "" {
		return append([]*artifacts.Artifact(nil), f.artifacts...), nil
	}
	target := artifacts.ApprovalState(stateFilter)
	out := make([]*artifacts.Artifact, 0, len(f.artifacts))
	for _, art := range f.artifacts {
		if f.approvals[art.ID].State == target {
			out = append(out, art)
		}
	}
	return out, nil
}

func (f *fakeArtifactStore) GetApproval(ctx context.Context, artifactID uuid.UUID) (*artifacts.Approval, error) {
	ap, ok := f.approvals[artifactID]
	if !ok {
		return nil, context.Canceled
	}
	return ap, nil
}

func (f *fakeArtifactStore) GetArtifactByID(ctx context.Context, id uuid.UUID) (*artifacts.Artifact, error) {
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
	ap := f.approvals[artifactID]
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
	f.reports = append(f.reports, report)
	return nil
}

func (f *fakeArtifactStore) RecordValidationOutcome(ctx context.Context, artifactID, runID uuid.UUID, passed bool, reportURI string) error {
	ap := f.approvals[artifactID]
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
