package papertrading

import (
	"testing"
	"time"
)

// TestPhase11ExitHarness demonstrates the paper-execution capability only.
// It is accelerated synthetic evidence and is not a profitability claim or a
// real-time six-month soak.
func TestPhase11ExitHarness(t *testing.T) {
	start := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	wf, intent := approvedPaperArtifacts(t, start)
	contract, costs := DefaultPaperCapabilityContract(), DefaultCostModel()
	venue, err := NewPaperVenue(contract, costs)
	if err != nil {
		t.Fatal(err)
	}
	request := CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: contract, CostModel: costs, InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: OrderMarket, CreatedAt: start, IdempotencyKey: "exit-order"}
	order, err := venue.Submit(request)
	if err != nil {
		t.Fatal(err)
	}
	replayedOrder, err := venue.Submit(request)
	if err != nil || replayedOrder.OrderID != order.OrderID {
		t.Fatalf("duplicate order delivery = %#v, %v", replayedOrder, err)
	}
	first := MarketTick{TickID: "exit-first", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 4, Timestamp: start.Add(2 * time.Second), ReceivedAt: start.Add(2 * time.Second), Session: SessionOpen, Source: "frozen-fixture"}
	second := first
	second.TickID, second.AvailableQuantity = "exit-second", 6
	second.Timestamp, second.ReceivedAt = start.Add(3*time.Second), start.Add(3*time.Second)
	if fills, err := venue.ProcessTick(first); err != nil || len(fills) != 1 || fills[0].Quantity != 4 {
		t.Fatalf("exit first partial fill = %#v, %v", fills, err)
	}
	fills, err := venue.ProcessTick(second)
	if err != nil || len(fills) != 1 || fills[0].Quantity != 6 {
		t.Fatalf("exit final fill = %#v, %v", fills, err)
	}
	account, err := NewPaperLedger("phase11-paper-fixture", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	for _, fill := range venue.Snapshot().Fills {
		if _, err := account.ApplyFill(fill); err != nil {
			t.Fatal(err)
		}
	}
	ledgerState := account.Snapshot()
	if ledgerState.Cash <= 0 || ledgerState.Positions["AAPL"].Quantity != 10 || len(ledgerState.Events) != 2 {
		t.Fatalf("exit ledger = %#v", ledgerState)
	}
	clean := Reconcile(ReconciliationInput{VenueSnapshot: venue.Snapshot(), Account: ledgerState}, second.ReceivedAt)
	if clean.Status != ReconciliationClean || len(clean.ReasonCodes) != 0 {
		t.Fatalf("exit reconciliation = %#v", clean)
	}
	corrupt := ledgerState
	corrupt.Cash++
	if result := Reconcile(ReconciliationInput{VenueSnapshot: venue.Snapshot(), Account: corrupt}, second.ReceivedAt); result.Status != ReconciliationRequired {
		t.Fatalf("corrupt state reconciled cleanly: %#v", result)
	}
	lastFill := venue.Snapshot().Fills[fills[0].FillID]
	if _, err := Attribute(AttributionInput{RecommendationID: intent.RecommendationID, RiskDecisionID: intent.RiskDecisionID, PaperIntentID: intent.IntentID, OrderID: order.OrderID, FillID: lastFill.FillID, Direction: intent.Direction, RecommendationEntry: 100, RiskEntry: 100, ActivationReference: 100, RecommendationQty: 10, RiskQty: 10, ExitPrice: 105, Fill: lastFill}); err != nil {
		t.Fatal(err)
	}
	venueRestarted, err := RestorePaperVenue(venue.Snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if duplicate, err := venueRestarted.ProcessTick(second); err != nil || len(duplicate) != 0 {
		t.Fatalf("restart duplicated fill = %#v, %v", duplicate, err)
	}
	ledgerRestarted, err := RestorePaperLedger(ledgerState)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledgerRestarted.ApplyFill(lastFill); err != nil {
		t.Fatal(err)
	}
	if len(ledgerRestarted.Snapshot().Events) != len(ledgerState.Events) {
		t.Fatal("restart duplicated ledger event")
	}
	if _, err := venue.ProcessTickWithSafety(first, true); err != ErrPaperExecutionBlocked {
		t.Fatalf("breaker did not block paper processing: %v", err)
	}
	soak, err := NewSoakRun(DefaultSoakProtocol(), start, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := soak.Record(SoakObservation{At: start.Add(90 * 24 * time.Hour), Recommendations: 1, EligiblePaperIntents: 1, Orders: 1, Fills: 2, PartialFills: 1, Crashes: 1}); err != nil {
		t.Fatal(err)
	}
	if err := soak.Complete(start.Add(180 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if soak.EvidenceClass != AcceleratedSyntheticSoak || soak.ActualForwardDuration != 0 {
		t.Fatalf("soak evidence mislabelled: %#v", soak)
	}
	if contract.Environment != EnvironmentPaper || contract.ExecutionAuthority != ExecutionAuthorityNone || intent.BrokerExecutionAllowed || intent.PortfolioMutation {
		t.Fatal("paper/live safety boundary changed")
	}
}
