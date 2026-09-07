package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBreakerBlocksApprovalAndRequiresOperatorReset(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "create-breaker"})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "request-breaker", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetBreaker(ctx, BreakerRequest{Name: "workflow-kill", Tripped: true, Reason: "audit unavailable", Actor: "system", ActorRole: ActorSystem, IdempotencyKey: "trip-breaker", Now: now}); err != nil {
		t.Fatal(err)
	}
	confirmation, err := NewConfirmation(workflow, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, _ = confirmation.WithDecision("operator-1", ConfirmationApprove)
	if _, err := store.Confirm(ctx, HumanConfirmationRequest{WorkflowID: workflow.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: ActorHuman, IdempotencyKey: "approve-breaker", Now: now}); !errors.Is(err, ErrBreakerActive) {
		t.Fatalf("approval while breaker active = %v, want breaker active", err)
	}
	if _, err := store.SetBreaker(ctx, BreakerRequest{Name: "workflow-kill", Tripped: false, Reason: "cleared", Actor: "research-agent", ActorRole: ActorResearcher, IdempotencyKey: "reset-forged", Now: now}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("researcher reset = %v, want unauthorized", err)
	}
	if _, err := store.SetBreaker(ctx, BreakerRequest{Name: "workflow-kill", Tripped: false, Reason: "cleared", Actor: "operator-1", ActorRole: ActorOperator, IdempotencyKey: "reset-breaker", Now: now}); err != nil {
		t.Fatal(err)
	}
	if store.AnyBreakerTripped() {
		t.Fatal("breaker remained active after operator reset")
	}
	if len(store.BreakerEvents(ctx, "workflow-kill")) != 2 {
		t.Fatal("breaker activation/reset was not auditable")
	}
}
