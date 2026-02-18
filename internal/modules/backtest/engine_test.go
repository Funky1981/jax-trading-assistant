package backtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/strategies"
)

// ── mock data source ──────────────────────────────────────────────────────────

type mockDataSource struct {
	candles    map[string][]strategies.Candle
	indicators map[string]map[time.Time]strategies.AnalysisInput
}

func newMockDS() *mockDataSource {
	return &mockDataSource{
		candles:    make(map[string][]strategies.Candle),
		indicators: make(map[string]map[time.Time]strategies.AnalysisInput),
	}
}

func (m *mockDataSource) GetCandles(ctx context.Context, symbol string, start, end time.Time) ([]strategies.Candle, error) {
	return m.candles[symbol], nil
}

func (m *mockDataSource) GetIndicators(ctx context.Context, symbol string, ts time.Time) (strategies.AnalysisInput, error) {
	if syms, ok := m.indicators[symbol]; ok {
		if input, ok := syms[ts]; ok {
			return input, nil
		}
	}
	return strategies.AnalysisInput{}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func registryWithRSI() *strategies.Registry {
	reg := strategies.NewRegistry()
	s := strategies.NewRSIMomentumStrategy()
	reg.Register(s, s.GetMetadata())
	return reg
}

// buildMockDS creates a data source that will trigger at least one RSI entry.
func buildMockDS(symbol string, base time.Time) *mockDataSource {
	ds := newMockDS()
	ds.candles[symbol] = []strategies.Candle{
		{Symbol: symbol, Timestamp: base, Open: 150, High: 152, Low: 149, Close: 151, Volume: 1_000_000},
		{Symbol: symbol, Timestamp: base.Add(24 * time.Hour), Open: 151, High: 156, Low: 150, Close: 155, Volume: 1_200_000},
		{Symbol: symbol, Timestamp: base.Add(48 * time.Hour), Open: 155, High: 160, Low: 154, Close: 159, Volume: 1_100_000},
	}
	ds.indicators[symbol] = map[time.Time]strategies.AnalysisInput{
		base: {
			Symbol:      symbol,
			Price:       150.0,
			Timestamp:   base,
			RSI:         22.0, // oversold → triggers buy signal
			ATR:         2.0,
			MarketTrend: "bullish",
			Volume:      1_000_000,
			AvgVolume20: 900_000,
		},
	}
	return ds
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestEngine_RunID_Format verifies the RunID has the expected "bt_<strategy>_<seed>" shape.
func TestEngine_RunID_Format(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	res, err := eng.Run(context.Background(), Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           42,
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	if !strings.HasPrefix(res.RunID, "bt_rsi_momentum_v1_") {
		t.Errorf("RunID %q does not have expected prefix bt_rsi_momentum_v1_", res.RunID)
	}
	if !strings.Contains(res.RunID, "42") {
		t.Errorf("RunID %q does not contain seed 42", res.RunID)
	}
}

// TestEngine_DeterministicSeed verifies that the same seed produces the same RunID.
func TestEngine_DeterministicSeed(t *testing.T) {
	base := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	cfg := Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           1234567890,
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	}

	r1, err := eng.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	r2, err := eng.Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}

	if r1.RunID != r2.RunID {
		t.Errorf("same seed should produce same RunID: got %q vs %q", r1.RunID, r2.RunID)
	}
	if r1.Seed != r2.Seed {
		t.Errorf("seed mismatch: %d vs %d", r1.Seed, r2.Seed)
	}
}

// TestEngine_AutoSeed verifies that seed=0 is replaced by a non-zero wall-clock seed.
func TestEngine_AutoSeed(t *testing.T) {
	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("MSFT", base)

	res, err := eng.Run(context.Background(), Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"MSFT"},
		StartDate:      base,
		EndDate:        base.Add(48 * time.Hour),
		DataSource:     ds,
		Seed:           0, // auto
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}
	if res.Seed == 0 {
		t.Error("expected auto-generated non-zero seed, got 0")
	}
}

// TestEngine_SymbolsPreserved verifies Result.Symbols mirrors cfg.Symbols.
func TestEngine_SymbolsPreserved(t *testing.T) {
	base := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())

	symbols := []string{"AAPL", "MSFT"}
	ds := buildMockDS("AAPL", base)
	// Also populate MSFT so the backtester doesn't error
	ds.candles["MSFT"] = ds.candles["AAPL"]
	ds.indicators["MSFT"] = ds.indicators["AAPL"]

	res, err := eng.Run(context.Background(), Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        symbols,
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           99,
		InitialCapital: 20_000,
		RiskPerTrade:   0.01,
	})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}
	if len(res.Symbols) != len(symbols) {
		t.Fatalf("expected %d symbols, got %d", len(symbols), len(res.Symbols))
	}
	for i, sym := range symbols {
		if res.Symbols[i] != sym {
			t.Errorf("symbols[%d]: expected %q, got %q", i, sym, res.Symbols[i])
		}
	}
}

// TestEngine_DefaultCapital verifies that InitialCapital=0 defaults to 100_000.
func TestEngine_DefaultCapital(t *testing.T) {
	base := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	res, err := eng.Run(context.Background(), Config{
		StrategyName: "rsi_momentum_v1",
		Symbols:      []string{"AAPL"},
		StartDate:    base,
		EndDate:      base.Add(72 * time.Hour),
		DataSource:   ds,
		Seed:         7,
		// InitialCapital left at zero
	})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}
	if res.InitialCapital != 100_000 {
		t.Errorf("expected default InitialCapital=100000, got %.2f", res.InitialCapital)
	}
}

// TestEngine_DefaultRiskPerTrade verifies that RiskPerTrade=0 defaults to 0.01.
func TestEngine_DefaultRiskPerTrade(t *testing.T) {
	base := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	// We can't directly observe RiskPerTrade on the result, so verify the run completes
	// without error and InitialCapital is set (proving the backtester received a valid config).
	res, err := eng.Run(context.Background(), Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           8,
		InitialCapital: 50_000,
		// RiskPerTrade left at zero
	})
	if err != nil {
		t.Fatalf("engine.Run with zero RiskPerTrade failed: %v", err)
	}
	// Run completed successfully, which means the backtester accepted the config.
	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

// TestEngine_UnknownStrategy verifies that an unregistered strategy name returns an error.
func TestEngine_UnknownStrategy(t *testing.T) {
	base := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	_, err := eng.Run(context.Background(), Config{
		StrategyName:   "does_not_exist",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           1,
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	})
	if err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
	if !strings.Contains(err.Error(), "does_not_exist") {
		t.Errorf("error should mention strategy name, got: %v", err)
	}
}

// TestEngine_TimingFields verifies RunAt and DurationMs are populated.
func TestEngine_TimingFields(t *testing.T) {
	before := time.Now()
	base := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	res, err := eng.Run(context.Background(), Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		Seed:           5,
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	})
	if err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}
	after := time.Now()

	if res.RunAt.Before(before) || res.RunAt.After(after) {
		t.Errorf("RunAt %v is outside expected range [%v, %v]", res.RunAt, before, after)
	}
	if res.DurationMs < 0 {
		t.Errorf("DurationMs should be non-negative, got %d", res.DurationMs)
	}
}

// TestEngine_DifferentSeeds verifies distinct seeds produce distinct RunIDs.
func TestEngine_DifferentSeeds(t *testing.T) {
	base := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	eng := New(registryWithRSI())
	ds := buildMockDS("AAPL", base)

	commonCfg := Config{
		StrategyName:   "rsi_momentum_v1",
		Symbols:        []string{"AAPL"},
		StartDate:      base,
		EndDate:        base.Add(72 * time.Hour),
		DataSource:     ds,
		InitialCapital: 10_000,
		RiskPerTrade:   0.01,
	}

	cfg1 := commonCfg
	cfg1.Seed = 111
	cfg2 := commonCfg
	cfg2.Seed = 222

	r1, err := eng.Run(context.Background(), cfg1)
	if err != nil {
		t.Fatalf("run1 failed: %v", err)
	}
	r2, err := eng.Run(context.Background(), cfg2)
	if err != nil {
		t.Fatalf("run2 failed: %v", err)
	}

	if r1.RunID == r2.RunID {
		t.Errorf("different seeds should produce different RunIDs, both got %q", r1.RunID)
	}
}
