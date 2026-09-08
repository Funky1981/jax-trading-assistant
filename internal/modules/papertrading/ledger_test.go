package papertrading

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestPaperLedgerDerivesCashPositionsAndRealizedPnLFromFills(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	ledger, err := NewPaperLedger("paper-account", "USD", 1000)
	if err != nil {
		t.Fatal(err)
	}
	buy := testFill("buy-fill", "LONG", 4, 100, .20, now)
	account, err := ledger.ApplyFill(buy)
	if err != nil {
		t.Fatal(err)
	}
	if account.Cash != 599.80 || account.Positions["AAPL"].Quantity != 4 || account.Positions["AAPL"].AverageCost != 100 || account.Fees != .20 {
		t.Fatalf("buy ledger = %#v", account)
	}
	sell := testFill("sell-fill", "SHORT", 2, 110, .10, now.Add(time.Second))
	account, err = ledger.ApplyFill(sell)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(account.Cash-819.70) > 1e-9 || account.Positions["AAPL"].Quantity != 2 || math.Abs(account.RealizedPnL-20) > 1e-9 || math.Abs(account.Fees-.30) > 1e-9 {
		t.Fatalf("sell ledger = %#v", account)
	}
	if _, err := ledger.ApplyFill(sell); err != nil {
		t.Fatalf("duplicate fill was not idempotent: %v", err)
	}
	if len(ledger.Snapshot().Events) != 2 {
		t.Fatal("duplicate fill created another ledger event")
	}
}

func TestPaperLedgerRejectsCapitalShortAndUnknownPriceViolations(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	ledger, err := NewPaperLedger("paper-account", "USD", 100)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ApplyFill(testFill("too-large", "LONG", 2, 100, 0, now)); !errors.Is(err, ErrInsufficientCapital) {
		t.Fatalf("capital error = %v", err)
	}
	if _, err := ledger.ApplyFill(testFill("short-first", "SHORT", 1, 100, 0, now)); !errors.Is(err, ErrInsufficientCapital) {
		t.Fatalf("short-first error = %v", err)
	}
	if _, err := ledger.ApplyFill(testFill("small-buy", "LONG", 1, 50, 0, now)); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.MarkToMarket(map[string]float64{"MSFT": 100}); !errors.Is(err, ErrInvalidMarketData) {
		t.Fatalf("missing price error = %v", err)
	}
}

func TestRestorePaperLedgerRejectsStateThatDoesNotMatchEvents(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	ledger, err := NewPaperLedger("paper-account", "USD", 1000)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.ApplyFill(testFill("restore", "LONG", 1, 100, 0, now)); err != nil {
		t.Fatal(err)
	}
	tampered := ledger.Snapshot()
	tampered.Cash = 1
	if _, err := RestorePaperLedger(tampered); err == nil {
		t.Fatal("tampered ledger restored")
	}
}

func testFill(id, direction string, quantity, price, fee float64, at time.Time) PaperFill {
	fill := PaperFill{ContractVersion: FillContractVersion, FillID: id, OrderID: "order-" + id, PaperIntentID: "intent-" + id, WorkflowID: "workflow-" + id, InstrumentID: "AAPL", Direction: direction, Quantity: quantity, Price: price, FilledAt: at, TickID: "tick-" + id, Costs: CostBreakdown{ModelID: "cost-v1", Commission: fee, ExecutedPrice: price}}
	fill.FillID = fillIdentity(fill)
	return fill
}
