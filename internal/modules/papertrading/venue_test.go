package papertrading

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
)

func TestPaperVenueRequiresBoundApprovedIntentAndAvoidsLookAhead(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	wf, intent := approvedPaperArtifacts(t, now)
	contract := DefaultPaperCapabilityContract()
	costs := DefaultCostModel()
	venue, err := NewPaperVenue(contract, costs)
	if err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: contract, CostModel: costs, InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: OrderMarket, CreatedAt: now, IdempotencyKey: "paper-order-1"})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != OrderNew || !order.ActivatesAt.After(now) {
		t.Fatalf("order activation = %#v", order)
	}
	before := MarketTick{TickID: "before", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(500 * time.Millisecond), ReceivedAt: now.Add(500 * time.Millisecond), Session: SessionOpen, Source: "fixture"}
	fills, err := venue.ProcessTick(before)
	if err != nil || len(fills) != 0 {
		t.Fatalf("pre-activation fill = %#v, %v", fills, err)
	}
	after := before
	after.TickID = "after"
	after.Timestamp = now.Add(2 * time.Second)
	after.ReceivedAt = after.Timestamp
	fills, err = venue.ProcessTick(after)
	if err != nil || len(fills) != 1 {
		t.Fatalf("activation fill = %#v, %v", fills, err)
	}
	if fills[0].FilledAt.Before(order.ActivatesAt) || fills[0].Price <= 0 || fills[0].Costs.ModelID != costs.ModelID {
		t.Fatalf("invalid fill provenance = %#v", fills[0])
	}
}

func TestPaperVenueRejectsLiveUnknownStaleAndUnsupportedPaths(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	wf, intent := approvedPaperArtifacts(t, now)
	contract := DefaultPaperCapabilityContract()
	costs := DefaultCostModel()
	venue, err := NewPaperVenue(contract, costs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := venue.Submit(CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: CapabilityContract{VenueID: contract.VenueID, ContractVersion: CapabilityContractVersion, Environment: EnvironmentLive}, CostModel: costs, InstrumentID: "AAPL", Quantity: 1, ReferencePrice: 100, OrderType: OrderMarket, CreatedAt: now, IdempotencyKey: "live"}); err == nil {
		t.Fatal("live order accepted")
	}
	stale := MarketTick{TickID: "stale", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 1, Timestamp: now.Add(-2 * time.Minute), ReceivedAt: now, Session: SessionOpen, Source: "fixture"}
	if _, err := venue.ProcessTick(stale); err != ErrStaleMarketData {
		t.Fatalf("stale tick error = %v", err)
	}
	unknown := stale
	unknown.TickID = "unknown"
	unknown.Timestamp = now.Add(time.Second)
	unknown.ReceivedAt = unknown.Timestamp
	unknown.Session = SessionUnknown
	if _, err := venue.ProcessTick(unknown); err != ErrMarketUnknown {
		t.Fatal("unknown session accepted")
	}
}

func TestPaperVenueConsumesLiquidityAndSupportsCancellation(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	wf, intent := approvedPaperArtifacts(t, now)
	contract := DefaultPaperCapabilityContract()
	costs := DefaultCostModel()
	venue, err := NewPaperVenue(contract, costs)
	if err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: contract, CostModel: costs, InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: OrderMarket, CreatedAt: now, IdempotencyKey: "partial-order"})
	if err != nil {
		t.Fatal(err)
	}
	closed := MarketTick{TickID: "closed", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 4, Timestamp: now.Add(2 * time.Second), ReceivedAt: now.Add(2 * time.Second), Session: SessionClosed, Source: "fixture"}
	if fills, err := venue.ProcessTick(closed); err != nil || len(fills) != 0 {
		t.Fatalf("closed-session fill = %#v, %v", fills, err)
	}
	first := closed
	first.TickID, first.Session = "partial-1", SessionOpen
	first.Timestamp, first.ReceivedAt = now.Add(3*time.Second), now.Add(3*time.Second)
	fills, err := venue.ProcessTick(first)
	if err != nil || len(fills) != 1 || fills[0].Quantity != 4 {
		t.Fatalf("first partial fill = %#v, %v", fills, err)
	}
	second := first
	second.TickID = "partial-2"
	second.Timestamp, second.ReceivedAt = now.Add(4*time.Second), now.Add(4*time.Second)
	fills, err = venue.ProcessTick(second)
	if err != nil || len(fills) != 1 || fills[0].Quantity != 4 {
		t.Fatalf("second partial fill limited by available liquidity = %#v, %v", fills, err)
	}
	if got := venue.Snapshot().Orders[order.OrderID]; got.Status != OrderPartiallyFilled || got.FilledQuantity != 8 || got.RemainingQuantity != 2 {
		t.Fatalf("partial order state = %#v", got)
	}
	if _, err := venue.Cancel(CancelRequest{OrderID: order.OrderID, IdempotencyKey: "cancel-partial", Now: now.Add(5 * time.Second)}); err != nil {
		t.Fatal(err)
	}
	third := second
	third.TickID = "after-cancel"
	third.Timestamp, third.ReceivedAt = now.Add(6*time.Second), now.Add(6*time.Second)
	if fills, err := venue.ProcessTick(third); err != nil || len(fills) != 0 {
		t.Fatalf("cancelled order filled = %#v, %v", fills, err)
	}
}

func approvedPaperArtifacts(t *testing.T, now time.Time) (workflow.Workflow, workflow.PaperIntent) {
	t.Helper()
	store := workflow.NewStore()
	risk := acceptedRiskForPaper(t, now)
	wf, err := store.Create(context.Background(), workflow.CreateRequest{RiskDecision: risk, Now: now, IdempotencyKey: "paper-create"})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.RequestHumanConfirmation(context.Background(), workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "paper-await", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := workflow.NewConfirmation(wf, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("human", workflow.ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: "human", ActorRole: workflow.ActorHuman, IdempotencyKey: "paper-approve", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wf, intent, err := store.CreatePaperIntent(context.Background(), workflow.PaperIntentRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "paper-intent", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return wf, intent
}

func acceptedRiskForPaper(t *testing.T, now time.Time) portfoliorisk.RiskDecision {
	t.Helper()
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: "paper-fixture", AsOf: now, CapturedAt: now, Provider: "phase11-fixture", Currency: "USD", Synthetic: true,
		Cash: portfoliorisk.KnownNumber(10000, "fixture"), Equity: portfoliorisk.KnownNumber(10000, "fixture"),
		Positions:      []portfoliorisk.Position{{InstrumentID: "MSFT", InstrumentResolved: true, Currency: "USD", SignedQuantity: 1, Price: portfoliorisk.KnownNumber(100, "fixture"), MarketValue: portfoliorisk.KnownNumber(100, "fixture"), CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: now, PriceSource: "fixture", Provenance: []string{"fixture"}}},
		ValuationBasis: "frozen", Provenance: []string{"fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := portfoliorisk.BuildRiskPolicy(portfoliorisk.RiskPolicy{Version: "paper-policy", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(1), MaximumPositionValue: portfoliorisk.Limit(2000), MaximumConcentration: portfoliorisk.Limit(.5), MaximumGrossExposure: portfoliorisk.Limit(5000), MaximumNetExposure: portfoliorisk.Limit(5000)})
	if err != nil {
		t.Fatal(err)
	}
	riskAllocation, leverage := .01, 1.0
	decision := portfoliorisk.EvaluateRecommendation(portfoliorisk.RecommendationRiskInput{RecommendationID: "paper-recommendation", InstrumentID: "AAPL", Currency: "USD", SignedMarketValue: 1000, RiskAllocation: &riskAllocation, RequestedLeverage: &leverage, ExecutionAuthority: "NONE", QuantResultIDs: []string{"quant:paper"}}, snapshot, analytics, policy, now, time.Hour)
	if decision.Outcome != portfoliorisk.DecisionAccept {
		t.Fatalf("paper fixture risk outcome = %s, reasons=%v", decision.Outcome, decision.ReasonCodes)
	}
	return decision
}
