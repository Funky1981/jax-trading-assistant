package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/portfoliorisk"
)

func TestStateMachineBindsRiskAndRejectsInvalidJumps(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	risk := acceptedRiskDecision(t, now)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: risk, Now: now, IdempotencyKey: "create-1"})
	if err != nil {
		t.Fatal(err)
	}
	if workflow.State != StateRiskAccepted || workflow.RiskDecisionID != risk.DecisionID || workflow.RecommendationID != risk.RecommendationID {
		t.Fatalf("workflow did not bind risk decision: %#v", workflow)
	}
	if _, err := store.transition(TransitionRequest{WorkflowID: workflow.WorkflowID, Action: ActionHumanApprove, Actor: "system", ActorRole: ActorSystem, IdempotencyKey: "jump-1", Now: now}, StateHumanApproved, ActorSystem); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("state jump error = %v, want invalid transition", err)
	}
	workflow, err = store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "request-1", Now: now})
	if err != nil || workflow.State != StateAwaitingHumanConfirmation {
		t.Fatalf("request confirmation = %#v, %v", workflow, err)
	}
	if _, err := store.Create(ctx, CreateRequest{RiskDecision: rejectedRiskDecision(risk), Now: now, IdempotencyKey: "reject-1"}); !errors.Is(err, ErrRiskRejected) {
		t.Fatalf("rejected risk error = %v, want risk rejection", err)
	}
}

func TestStateMachineAuditIsAtomicAndRetriesAreIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "create-atomic"})
	if err != nil {
		t.Fatal(err)
	}
	store.FailNextAuditPersistence()
	if _, err := store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "request-atomic", Now: now}); err == nil {
		t.Fatal("expected audit failure")
	}
	unchanged, err := store.Get(ctx, workflow.WorkflowID)
	if err != nil || unchanged.State != StateRiskAccepted || unchanged.Revision != 1 {
		t.Fatalf("audit failure advanced state: %#v, %v", unchanged, err)
	}
	request := TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "request-atomic", Now: now}
	first, err := store.RequestHumanConfirmation(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RequestHumanConfirmation(ctx, request)
	if err != nil || first.Revision != second.Revision || first.State != second.State {
		t.Fatalf("idempotent retry changed result: %#v %#v %v", first, second, err)
	}
	events, err := store.Events(ctx, workflow.WorkflowID)
	if err != nil || len(events) != 2 || events[1].Sequence != 2 {
		t.Fatalf("audit events = %#v, %v", events, err)
	}
	replayed, err := store.Replay(ctx, workflow.WorkflowID)
	if err != nil || replayed.State != StateAwaitingHumanConfirmation {
		t.Fatalf("replay = %#v, %v", replayed, err)
	}
}

func TestStateMachineRejectsNonUTCTimestamps(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.FixedZone("BST", 3600))
	_, err := NewStore().Create(context.Background(), CreateRequest{RiskDecision: acceptedRiskDecision(t, now.UTC()), Now: now, IdempotencyKey: "non-utc"})
	if !errors.Is(err, ErrInvalidTimestamp) {
		t.Fatalf("non-UTC timestamp error = %v, want invalid timestamp", err)
	}
}

func acceptedRiskDecision(t *testing.T, now time.Time) portfoliorisk.RiskDecision {
	t.Helper()
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: "synthetic-phase10", AsOf: now, CapturedAt: now,
		Provider: "phase10-fixture", Currency: "USD", Synthetic: true,
		Cash: portfoliorisk.KnownNumber(10000, "fixture"), Equity: portfoliorisk.KnownNumber(10000, "fixture"),
		Positions: []portfoliorisk.Position{{
			InstrumentID: "MSFT", InstrumentResolved: true, Currency: "USD", SignedQuantity: 1,
			Price: portfoliorisk.KnownNumber(100, "fixture"), MarketValue: portfoliorisk.KnownNumber(100, "fixture"),
			CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: now,
			PriceSource: "phase10-fixture", Provenance: []string{"fixture:phase10"},
		}},
		ValuationBasis: "frozen-fixture", Provenance: []string{"fixture:phase10"},
	})
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := portfoliorisk.BuildRiskPolicy(portfoliorisk.RiskPolicy{
		Version: "phase10-policy-1", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(1),
		MaximumPositionValue: portfoliorisk.Limit(2000), MaximumConcentration: portfoliorisk.Limit(.5),
		MaximumGrossExposure: portfoliorisk.Limit(5000), MaximumNetExposure: portfoliorisk.Limit(5000),
	})
	if err != nil {
		t.Fatal(err)
	}
	riskAllocation := 0.01
	leverage := 1.0
	decision := portfoliorisk.EvaluateRecommendation(portfoliorisk.RecommendationRiskInput{
		RecommendationID: "recommendation-phase10", InstrumentID: "AAPL", Currency: "USD",
		SignedMarketValue: 1000, RiskAllocation: &riskAllocation, RequestedLeverage: &leverage,
		ExecutionAuthority: "NONE", QuantResultIDs: []string{"quant:phase10"},
	}, snapshot, analytics, policy, now, time.Hour)
	if decision.Outcome != portfoliorisk.DecisionAccept {
		t.Fatalf("fixture risk outcome = %s, reasons=%v", decision.Outcome, decision.ReasonCodes)
	}
	return decision
}

func rejectedRiskDecision(accepted portfoliorisk.RiskDecision) portfoliorisk.RiskDecision {
	accepted.Outcome = portfoliorisk.DecisionReject
	accepted.ReasonCodes = []portfoliorisk.ReasonCode{portfoliorisk.ReasonRejectStalePortfolio}
	accepted.Explanations = []string{"fixture rejection"}
	// A rejected decision is rejected by workflow before it can be persisted.
	return accepted
}
