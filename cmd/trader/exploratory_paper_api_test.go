package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/workflow"
	"jax-trading-assistant/libs/auth"
)

type exploratoryExitDecisionStoreStub struct {
	record       exploratorypaper.LifecycleRecord
	persistCalls int
	persisted    *exploratorypaper.ApprovalSnapshot
	persistErr   error
}

func (s *exploratoryExitDecisionStoreStub) PersistExitDecision(_ context.Context, _ string, approval exploratorypaper.ApprovalSnapshot) error {
	s.persistCalls++
	s.persisted = &approval
	return s.persistErr
}

func (s *exploratoryExitDecisionStoreStub) Get(_ context.Context, _ string) (exploratorypaper.LifecycleRecord, error) {
	return s.record, nil
}

func TestExploratoryExitDecisionHandlerUsesAuthenticatedJWT(t *testing.T) {
	tests := []struct {
		name               string
		withJWT            bool
		jwtActor           string
		confirmationActor  string
		spoofedHeader      string
		action             string
		state              workflow.State
		mode               string
		wantStatus         int
		wantPersistedCalls int
	}{
		{name: "matching authenticated user is accepted", withJWT: true, jwtActor: "user-1", confirmationActor: "user-1", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "paper", wantStatus: http.StatusOK, wantPersistedCalls: 1},
		{name: "spoofed header cannot override JWT user", withJWT: true, jwtActor: "user-1", confirmationActor: "user-1", spoofedHeader: "attacker", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "paper", wantStatus: http.StatusOK, wantPersistedCalls: 1},
		{name: "confirmation for another user is rejected", withJWT: true, jwtActor: "user-1", confirmationActor: "user-2", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "paper", wantStatus: http.StatusForbidden, wantPersistedCalls: 0},
		{name: "header without JWT is rejected", withJWT: false, confirmationActor: "attacker", spoofedHeader: "attacker", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "paper", wantStatus: http.StatusForbidden, wantPersistedCalls: 0},
		{name: "no JWT and no header is rejected", withJWT: false, confirmationActor: "anonymous", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "paper", wantStatus: http.StatusForbidden, wantPersistedCalls: 0},
		{name: "non-PAPER runtime is rejected", withJWT: true, jwtActor: "user-1", confirmationActor: "user-1", action: "exit-approval", state: workflow.StatePaperIntentCreated, mode: "dev", wantStatus: http.StatusConflict, wantPersistedCalls: 0},
		{name: "matching authenticated rejection is persisted", withJWT: true, jwtActor: "user-1", confirmationActor: "user-1", action: "exit-rejection", state: workflow.StateHumanRejected, mode: "paper", wantStatus: http.StatusOK, wantPersistedCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JAX_RUNTIME_MODE", tt.mode)
			store := &exploratoryExitDecisionStoreStub{record: exploratorypaper.LifecycleRecord{Position: exploratorypaper.Position{PositionID: "position-1"}}}
			snapshot := exploratorypaper.ApprovalSnapshot{
				Workflow: workflow.Workflow{State: tt.state, Confirmation: &workflow.Confirmation{Actor: tt.confirmationActor}},
			}
			body, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/exploratory-paper/positions/position-1/"+tt.action, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tt.spoofedHeader != "" {
				req.Header.Set("X-User-ID", tt.spoofedHeader)
			}
			rec := httptest.NewRecorder()
			base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handleExploratoryExitDecision(w, r, store, "position-1", tt.action)
			})
			var handler http.Handler = base
			if tt.withJWT {
				manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("test-secret"), Expiry: time.Hour})
				if err != nil {
					t.Fatal(err)
				}
				token, err := manager.GenerateToken(tt.jwtActor, tt.jwtActor+"@example.test", "operator")
				if err != nil {
					t.Fatal(err)
				}
				req.Header.Set("Authorization", "Bearer "+token)
				handler = manager.Middleware(handler)
			}
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if store.persistCalls != tt.wantPersistedCalls {
				t.Fatalf("persist calls = %d, want %d", store.persistCalls, tt.wantPersistedCalls)
			}
			if tt.wantPersistedCalls == 1 && store.persisted.Workflow.State != tt.state {
				t.Fatalf("persisted state = %q, want %q", store.persisted.Workflow.State, tt.state)
			}
			if tt.spoofedHeader != "" && tt.wantPersistedCalls == 1 && store.persisted.Workflow.Confirmation.Actor != tt.jwtActor {
				t.Fatalf("persisted actor = %q, want authenticated JWT actor %q", store.persisted.Workflow.Confirmation.Actor, tt.jwtActor)
			}
		})
	}
}

func TestExploratoryExitDecisionHandlerTreatsMissingJWTClaimsAsForbidden(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	store := &exploratoryExitDecisionStoreStub{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exploratory-paper/positions/position-1/exit-approval", bytes.NewReader([]byte(`{"workflow":{"state":"PAPER_INTENT_CREATED","confirmation":{"actor":"attacker"}}}`)))
	req.Header.Set("X-User-ID", "attacker")
	rec := httptest.NewRecorder()
	handleExploratoryExitDecision(rec, req, store, "position-1", "exit-approval")
	if rec.Code != http.StatusForbidden || store.persistCalls != 0 {
		t.Fatalf("status=%d persistCalls=%d, want forbidden and no persistence", rec.Code, store.persistCalls)
	}
}
