package edgestability_test

import (
	"testing"
	"time"

	"jax-trading-assistant/libs/edgestability"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func makeOutcome(stratID string, pnl, retFrac float64) edgestability.Outcome {
	return edgestability.Outcome{
		TradeID:    "T",
		Symbol:     "AAPL",
		StrategyID: stratID,
		PnL:        pnl,
		ReturnFrac: retFrac,
		ClosedAt:   time.Now(),
	}
}

// ─── RecordOutcome / metrics ──────────────────────────────────────────────────

func TestEmptyStrategySnapshotZero(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 10})
	// Record one outcome to create the series, then reset to check zero snap.
	m.RecordOutcome(makeOutcome("s1", 10, 0.01))
	m.Reset("s1")

	if _, err := m.Snapshot("s1"); err == nil {
		t.Fatal("expected error for unknown strategy after reset, got nil")
	}
}

func TestWinRateCalculation(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 10})

	// 3 wins, 2 losses → 60%
	for i := range 5 {
		pnl := 10.0
		ret := 0.01
		if i < 2 {
			pnl = -10.0
			ret = -0.01
		}
		m.RecordOutcome(makeOutcome("wrstrat", pnl, ret))
	}

	snap, err := m.Snapshot("wrstrat")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.WinRate < 0.59 || snap.WinRate > 0.61 {
		t.Errorf("WinRate: got %.4f, want ~0.60", snap.WinRate)
	}
	if snap.TradesInWindow != 5 {
		t.Errorf("TradesInWindow: got %d, want 5", snap.TradesInWindow)
	}
}

func TestRollingWindowEvictsOldOutcomes(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 3})

	// Record 5 outcomes; window should keep only last 3.
	for range 5 {
		m.RecordOutcome(makeOutcome("win", 10, 0.01))
	}

	snap, _ := m.Snapshot("win")
	if snap.TradesInWindow != 3 {
		t.Errorf("TradesInWindow: got %d, want 3 (window size)", snap.TradesInWindow)
	}
}

// ─── Sharpe ───────────────────────────────────────────────────────────────────

func TestSharpePositiveForConsistentGains(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 50})

	// Uniform 1% returns → zero std dev, but with slight noise to avoid /0.
	returns := []float64{0.01, 0.011, 0.009, 0.010, 0.012, 0.010, 0.011, 0.010, 0.009, 0.010}
	for _, r := range returns {
		m.RecordOutcome(makeOutcome("sharp", r*100, r))
	}

	snap, _ := m.Snapshot("sharp")
	if snap.SharpeRatio <= 0 {
		t.Errorf("expected positive Sharpe for consistent gains, got %.2f", snap.SharpeRatio)
	}
}

// ─── Drift alerts ─────────────────────────────────────────────────────────────

func TestSharpeDecayAlert(t *testing.T) {
	m := edgestability.New(edgestability.Config{
		WindowSize:     10,
		BaselineSharpe: 1.5,
		MinSharpe:      0.5,
	})

	// All losses → Sharpe will be strongly negative.
	for range 10 {
		m.RecordOutcome(makeOutcome("decay", -5, -0.005))
	}

	snap, _ := m.Snapshot("decay")
	if !snap.IsDrifting {
		t.Fatal("expected IsDrifting=true, got false")
	}
	found := false
	for _, a := range snap.Alerts {
		if a.Code == edgestability.AlertSharpeDecay {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AlertSharpeDecay in %v", snap.Alerts)
	}
	if snap.DecayScore <= 0 {
		t.Errorf("expected positive DecayScore, got %.2f", snap.DecayScore)
	}
}

func TestLowWinRateAlert(t *testing.T) {
	m := edgestability.New(edgestability.Config{
		WindowSize: 10,
		MinWinRate: 0.40,
	})

	// 10 losses, 0 wins → win rate = 0%.
	for range 10 {
		m.RecordOutcome(makeOutcome("lose", -5, -0.005))
	}

	snap, _ := m.Snapshot("lose")
	found := false
	for _, a := range snap.Alerts {
		if a.Code == edgestability.AlertLowWinRate {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AlertLowWinRate in %v", snap.Alerts)
	}
}

func TestDrawdownBreakAlert(t *testing.T) {
	m := edgestability.New(edgestability.Config{
		WindowSize:  10,
		MaxDrawdown: 0.10,
	})

	// First build equity then lose 20%.
	m.RecordOutcome(makeOutcome("dd", 1000, 0.10))
	for range 5 {
		m.RecordOutcome(makeOutcome("dd", -50, -0.05))
	}

	snap, _ := m.Snapshot("dd")
	found := false
	for _, a := range snap.Alerts {
		if a.Code == edgestability.AlertDrawdownBreak {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected AlertDrawdownBreak in %v", snap.Alerts)
	}
}

// ─── SnapshotAll ──────────────────────────────────────────────────────────────

func TestSnapshotAllReturnsBothStrategies(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 5})

	m.RecordOutcome(makeOutcome("s1", 10, 0.01))
	m.RecordOutcome(makeOutcome("s2", -5, -0.01))

	all := m.SnapshotAll()
	if len(all) != 2 {
		t.Errorf("SnapshotAll: got %d, want 2", len(all))
	}
}

// ─── Reset ────────────────────────────────────────────────────────────────────

func TestResetClearsSeries(t *testing.T) {
	m := edgestability.New(edgestability.Config{WindowSize: 5})
	m.RecordOutcome(makeOutcome("clear", 10, 0.01))
	m.Reset("clear")

	if _, err := m.Snapshot("clear"); err == nil {
		t.Fatal("expected error after reset, got nil")
	}
}

// ─── Snapshot on unknown strategy ─────────────────────────────────────────────

func TestSnapshotUnknownStrategyReturnsError(t *testing.T) {
	m := edgestability.New(edgestability.Config{})
	if _, err := m.Snapshot("nope"); err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
}
