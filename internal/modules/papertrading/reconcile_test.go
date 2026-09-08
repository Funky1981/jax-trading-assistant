package papertrading

import (
	"testing"
	"time"
)

func TestReconcileAcceptsValidVenueAndLedgerSnapshot(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	venue, order := filledVenue(t, now)
	ledger, err := NewPaperLedger("paper-account", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	for _, fill := range venue.Snapshot().Fills {
		if _, err := ledger.ApplyFill(fill); err != nil {
			t.Fatal(err)
		}
	}
	result := Reconcile(ReconciliationInput{VenueSnapshot: venue.Snapshot(), Account: ledger.Snapshot()}, now)
	if result.Status != ReconciliationClean || len(result.ReasonCodes) != 0 || result.Identity == "" {
		t.Fatalf("valid reconciliation = %#v", result)
	}
	if venue.Snapshot().Orders[order.OrderID].Status != OrderFilled {
		t.Fatal("fixture order did not fill")
	}
}

func TestReconcileRequiresRecoveryForCorruptState(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	venue, _ := filledVenue(t, now)
	ledger, err := NewPaperLedger("paper-account", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	for _, fill := range venue.Snapshot().Fills {
		if _, err := ledger.ApplyFill(fill); err != nil {
			t.Fatal(err)
		}
	}
	corrupt := ledger.Snapshot()
	corrupt.Cash += 1
	result := Reconcile(ReconciliationInput{VenueSnapshot: venue.Snapshot(), Account: corrupt}, now)
	if result.Status != ReconciliationRequired || !containsReason(result.ReasonCodes, ReasonLedgerMismatch) {
		t.Fatalf("corrupt reconciliation = %#v", result)
	}
	market := MarketTick{TickID: "stale-reference", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 1, Timestamp: now.Add(-2 * time.Minute), ReceivedAt: now, Session: SessionOpen, Source: "fixture"}
	result = Reconcile(ReconciliationInput{VenueSnapshot: venue.Snapshot(), Account: ledger.Snapshot(), MarketTick: &market}, now)
	if !containsReason(result.ReasonCodes, ReasonInvalidMarketData) {
		t.Fatalf("stale market reconciliation = %#v", result)
	}
}

func containsReason(reasons []ReasonCode, wanted ReasonCode) bool {
	for _, reason := range reasons {
		if reason == wanted {
			return true
		}
	}
	return false
}

func filledVenue(t *testing.T, now time.Time) (*PaperVenue, PaperOrder) {
	t.Helper()
	wf, intent := approvedPaperArtifacts(t, now)
	contract, costs := DefaultPaperCapabilityContract(), DefaultCostModel()
	venue, err := NewPaperVenue(contract, costs)
	if err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: contract, CostModel: costs, InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: OrderMarket, CreatedAt: now, IdempotencyKey: "reconcile-order"})
	if err != nil {
		t.Fatal(err)
	}
	tick := MarketTick{TickID: "reconcile-tick", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(2 * time.Second), ReceivedAt: now.Add(2 * time.Second), Session: SessionOpen, Source: "fixture"}
	if _, err := venue.ProcessTick(tick); err != nil {
		t.Fatal(err)
	}
	return venue, order
}
