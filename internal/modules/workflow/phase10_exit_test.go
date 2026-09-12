package workflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestPhase10ExitHarness demonstrates the exact Phase-10 gate with frozen
// synthetic Phase-06/09 artifacts. It proves workflow mechanics only; it is
// not a trading-performance or profitability claim.
func TestPhase10ExitHarness(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	risk := acceptedRiskDecision(t, now)
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: risk, Now: now, IdempotencyKey: "exit-create"})
	if err != nil || workflow.WorkflowID == "" || workflow.State != StateRiskAccepted {
		t.Fatalf("create risk-bound workflow = %#v, %v", workflow, err)
	}
	if _, err := store.Create(ctx, CreateRequest{RiskDecision: rejectedRiskDecision(risk), Now: now, IdempotencyKey: "exit-rejected-risk"}); !errors.Is(err, ErrRiskRejected) {
		t.Fatalf("risk reject path = %v", err)
	}
	workflow, err = store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "exit-await", Now: now})
	if err != nil || workflow.State != StateAwaitingHumanConfirmation {
		t.Fatalf("awaiting-human transition = %#v, %v", workflow, err)
	}
	if workflow.Confirmation != nil {
		t.Fatal("workflow inferred approval before human action")
	}
	confirmation, err := NewConfirmation(workflow, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("operator-1", ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.Confirm(ctx, HumanConfirmationRequest{WorkflowID: workflow.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: ActorHuman, IdempotencyKey: "exit-approve", Now: now})
	if err != nil || workflow.State != StateHumanApproved || workflow.Confirmation == nil {
		t.Fatalf("explicit human approval = %#v, %v", workflow, err)
	}
	workflow, intent, err := store.CreatePaperIntent(ctx, PaperIntentRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "exit-intent", Now: now})
	if err != nil || workflow.State != StatePaperIntentCreated {
		t.Fatalf("paper intent transition = %#v %#v %v", workflow, intent, err)
	}
	if err := intent.Validate(); err != nil || intent.ExecutionStatus != ExecutionStatusNotExecuted || !intent.PaperOnly || intent.BrokerExecutionAllowed || intent.PortfolioMutation {
		t.Fatalf("paper intent execution boundary = %#v, %v", intent, err)
	}
	if len(store.BreakerEvents(ctx, "unused")) != 0 || store.AnyBreakerTripped() {
		t.Fatal("unexpected breaker or hidden execution state")
	}
	replayed, err := store.Replay(ctx, workflow.WorkflowID)
	if err != nil || replayed.State != StatePaperIntentCreated || replayed.PaperIntentID != intent.IntentID {
		t.Fatalf("audit replay = %#v, %v", replayed, err)
	}
	report := store.Health(ctx, now, HealthDependencies{PersistenceKnown: true, PersistenceOK: true, AuditKnown: true, AuditOK: true, SafetyKnown: true})
	if report.Status != HealthHealthy || report.Safety.AllowLiveTrading || report.Safety.BrokerExecutionAllowed || report.Safety.ExecutionWorkerEnabled || report.Safety.BrokerExecutionAuthority != BrokerExecutionAuthorityNone || report.Safety.MaximumLeverage != "1x" {
		t.Fatalf("operator status = %#v", report)
	}

	// The paper-intent path intentionally has no call site for a broker,
	// execution worker, order, trade or fill. The contract's safety fields are
	// the negative proof at this phase boundary.
	var _ = risk.Outcome
}
