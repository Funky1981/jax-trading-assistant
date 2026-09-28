package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
	"jax-trading-assistant/libs/auth"

	"github.com/google/uuid"
)

type candidateDecisionServiceProbe struct {
	decisions []approvalsmod.ApprovalRequest
}

func (s *candidateDecisionServiceProbe) Decide(_ context.Context, request approvalsmod.ApprovalRequest) (*approvalsmod.Approval, error) {
	s.decisions = append(s.decisions, request)
	return &approvalsmod.Approval{ID: uuid.New(), CandidateID: request.CandidateID, Decision: request.Decision, ApprovedBy: request.ApprovedBy, DecidedAt: time.Now().UTC()}, nil
}

func (*candidateDecisionServiceProbe) GetByCandidate(context.Context, uuid.UUID) (*approvalsmod.ApprovalDetail, error) {
	return nil, context.Canceled
}

func TestCandidateDecisionHandlerUsesValidatedJWTActorAndIgnoresSpoofHeader(t *testing.T) {
	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("core-readiness-auth-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.GenerateToken("user-alice", "alice", "operator")
	if err != nil {
		t.Fatal(err)
	}
	service := &candidateDecisionServiceProbe{}
	candidateID := uuid.New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleApprovalDecision(w, r, service, candidateID, approvalsmod.DecisionRejected)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+candidateID.String()+"/reject", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-ID", "user-bob")
	rec := httptest.NewRecorder()
	manager.Middleware(handler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if len(service.decisions) != 1 || service.decisions[0].ApprovedBy != "user-alice" || service.decisions[0].Decision != approvalsmod.DecisionRejected {
		t.Fatalf("persisted decision = %+v, want rejected by authenticated user-alice", service.decisions)
	}
}

func TestCandidateDecisionHandlerWithoutJWTClaimsReturnsForbiddenBeforePersistence(t *testing.T) {
	service := &candidateDecisionServiceProbe{}
	candidateID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+candidateID.String()+"/approve", nil)
	req.Header.Set("X-User-ID", "user-spoof")
	rec := httptest.NewRecorder()
	handleApprovalDecision(rec, req, service, candidateID, approvalsmod.DecisionApproved)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", rec.Code, rec.Body.String())
	}
	if len(service.decisions) != 0 {
		t.Fatalf("unauthenticated request persisted %d decisions", len(service.decisions))
	}
}
