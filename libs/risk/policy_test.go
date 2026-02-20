package risk_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"jax-trading-assistant/libs/risk"
)

// ─── Policy loading ───────────────────────────────────────────────────────────

func TestDefaultPolicyIsValid(t *testing.T) {
	p := risk.DefaultPolicy()
	if p == nil {
		t.Fatal("DefaultPolicy returned nil")
	}
	if p.Position.MaxRiskPerTrade <= 0 {
		t.Errorf("expected MaxRiskPerTrade > 0, got %.4f", p.Position.MaxRiskPerTrade)
	}
	if p.Portfolio.MaxPositions <= 0 {
		t.Errorf("expected MaxPositions > 0, got %d", p.Portfolio.MaxPositions)
	}
	if p.Version == "" {
		t.Error("expected non-empty Version")
	}
}

func TestLoadPolicyFromFile(t *testing.T) {
	doc := map[string]interface{}{
		"portfolio_constraints": map[string]interface{}{
			"max_position_size":       25000.0,
			"max_positions":           5,
			"max_sector_exposure":     0.25,
			"max_correlated_exposure": 0.35,
			"max_portfolio_risk":      0.10,
			"max_drawdown":            0.15,
			"min_account_size":        5000.0,
		},
		"position_limits": map[string]interface{}{
			"max_risk_per_trade": 0.01,
			"min_risk_per_trade": 0.003,
			"max_leverage":       1.5,
			"min_stop_distance":  0.005,
			"max_stop_distance":  0.08,
		},
		"sizing_model": "fixed_fractional",
	}

	f, err := os.CreateTemp(t.TempDir(), "risk-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(doc); err != nil {
		t.Fatal(err)
	}
	f.Close()

	p, err := risk.LoadPolicy(f.Name())
	if err != nil {
		t.Fatalf("LoadPolicy failed: %v", err)
	}
	if p.Portfolio.MaxPositions != 5 {
		t.Errorf("expected MaxPositions=5, got %d", p.Portfolio.MaxPositions)
	}
	if p.Position.MaxRiskPerTrade != 0.01 {
		t.Errorf("expected MaxRiskPerTrade=0.01, got %.4f", p.Position.MaxRiskPerTrade)
	}
	if p.LoadedFrom != f.Name() {
		t.Errorf("LoadedFrom mismatch: %s", p.LoadedFrom)
	}
}

func TestLoadPolicyMissingFile(t *testing.T) {
	// Missing file → fall back to defaults without error
	p, err := risk.LoadPolicy(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if p == nil {
		t.Fatal("expected default policy, got nil")
	}
}

func TestLoadPolicyEmptyPath(t *testing.T) {
	p, err := risk.LoadPolicy("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected default policy")
	}
}

func TestLoadPolicyInvalidJSON(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "bad-*.json")
	f.WriteString("{not valid json")
	f.Close()
	_, err := risk.LoadPolicy(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ─── CheckSignal ──────────────────────────────────────────────────────────────

func TestCheckSignalPassesWithinLimits(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy())
	// 2% stop, $25 500 position on $100k account → risk = 3*(25500/150)/100000 = 0.51% → above MinRiskPerTrade (0.50%)
	vs := e.CheckSignal(risk.SignalInput{
		Symbol:        "AAPL",
		EntryPrice:    150.00,
		StopLoss:      147.00, // 2% stop — within [1%, 10%]
		AccountEquity: 100_000,
		PositionValue: 25_500,
	})
	if !vs.IsEmpty() {
		t.Errorf("expected no violations, got: %v", vs)
	}
}

func TestCheckSignalStopTooTight(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy())
	// stop distance = 0.3% → below 1% minimum
	vs := e.CheckSignal(risk.SignalInput{
		Symbol:        "SPY",
		EntryPrice:    500.00,
		StopLoss:      498.50,
		AccountEquity: 100_000,
		PositionValue: 50_000,
	})
	if vs.IsEmpty() {
		t.Fatal("expected STOP_TOO_TIGHT violation")
	}
	if vs[0].Code != risk.ViolationStopTooTight {
		t.Errorf("expected STOP_TOO_TIGHT, got %s", vs[0].Code)
	}
}

func TestCheckSignalStopTooWide(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy())
	// stop distance = 15% → above 10% maximum
	vs := e.CheckSignal(risk.SignalInput{
		Symbol:        "TSLA",
		EntryPrice:    200.00,
		StopLoss:      170.00,
		AccountEquity: 100_000,
		PositionValue: 10_000,
	})
	if vs.IsEmpty() {
		t.Fatal("expected STOP_TOO_WIDE violation")
	}
	found := false
	for _, v := range vs {
		if v.Code == risk.ViolationStopTooWide {
			found = true
		}
	}
	if !found {
		t.Errorf("expected STOP_TOO_WIDE in violations: %v", vs)
	}
}

func TestCheckSignalPositionTooLarge(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy()) // MaxPositionSize = 50_000
	vs := e.CheckSignal(risk.SignalInput{
		Symbol:        "NVDA",
		EntryPrice:    900.00,
		StopLoss:      855.00, // 5% stop — valid
		AccountEquity: 500_000,
		PositionValue: 75_000, // > 50_000 limit
	})
	if vs.IsEmpty() {
		t.Fatal("expected POSITION_VALUE_TOO_LARGE violation")
	}
	found := false
	for _, v := range vs {
		if v.Code == risk.ViolationPositionTooLarge {
			found = true
		}
	}
	if !found {
		t.Errorf("expected POSITION_VALUE_TOO_LARGE in violations: %v", vs)
	}
}

// ─── CheckPortfolio ───────────────────────────────────────────────────────────

func TestCheckPortfolioPasses(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy())
	vs := e.CheckPortfolio(risk.PortfolioState{
		NetLiquidation:  50_000,
		OpenPositions:   3,
		DailyLossDollar: 500,
		CurrentDrawdown: 0.05,
	})
	if !vs.IsEmpty() {
		t.Errorf("expected no violations, got: %v", vs)
	}
}

func TestCheckPortfolioAccountTooSmall(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy()) // MinAccountSize = 10_000
	vs := e.CheckPortfolio(risk.PortfolioState{
		NetLiquidation: 8_000,
		OpenPositions:  0,
	})
	if vs.IsEmpty() {
		t.Fatal("expected ACCOUNT_TOO_SMALL violation")
	}
	if vs[0].Code != risk.ViolationAccountTooSmall {
		t.Errorf("expected ACCOUNT_TOO_SMALL, got %s", vs[0].Code)
	}
}

func TestCheckPortfolioTooManyPositions(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy()) // MaxPositions = 10
	vs := e.CheckPortfolio(risk.PortfolioState{
		NetLiquidation: 100_000,
		OpenPositions:  10, // at limit → blocked
	})
	if vs.IsEmpty() {
		t.Fatal("expected TOO_MANY_OPEN_POSITIONS violation")
	}
}

func TestCheckPortfolioDrawdownHalt(t *testing.T) {
	e := risk.NewEnforcer(risk.DefaultPolicy()) // MaxDrawdown = 0.20
	vs := e.CheckPortfolio(risk.PortfolioState{
		NetLiquidation:  100_000,
		OpenPositions:   2,
		CurrentDrawdown: 0.21,
	})
	found := false
	for _, v := range vs {
		if v.Code == risk.ViolationDrawdownHalt {
			found = true
		}
	}
	if !found {
		t.Errorf("expected DRAWDOWN_HALT in violations %v", vs)
	}
}

func TestViolationsError(t *testing.T) {
	vs := risk.Violations{
		{Code: risk.ViolationStopTooTight, Message: "too tight", Limit: 0.01, Observed: 0.003},
		{Code: risk.ViolationRiskTooHigh, Message: "too risky", Limit: 0.02, Observed: 0.05},
	}
	msg := vs.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
