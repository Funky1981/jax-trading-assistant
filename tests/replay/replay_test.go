package replay

import (
	"context"
	"testing"

	"jax-trading-assistant/libs/strategies"
)

func TestLoadFixture(t *testing.T) {
	tests := []struct {
		name        string
		fixtureName string
		wantErr     bool
	}{
		{
			name:        "load aapl rally fixture",
			fixtureName: "aapl-rally",
			wantErr:     false,
		},
		{
			name:        "load msft consolidation fixture",
			fixtureName: "msft-consolidation",
			wantErr:     false,
		},
		{
			name:        "load tsla volatility fixture",
			fixtureName: "tsla-volatility",
			wantErr:     false,
		},
		{
			name:        "load with .json extension",
			fixtureName: "aapl-rally.json",
			wantErr:     false,
		},
		{
			name:        "load nonexistent fixture",
			fixtureName: "nonexistent-fixture",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture, err := LoadFixture(tt.fixtureName)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFixture() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if fixture == nil {
					t.Error("LoadFixture() returned nil fixture")
					return
				}
				if len(fixture.MarketData) == 0 {
					t.Error("LoadFixture() returned fixture with no market data")
				}
			}
		})
	}
}

func TestLoadFixtureValidation(t *testing.T) {
	fixture, err := LoadFixture("aapl-rally")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	// Validate fixture structure
	if fixture.Name != "aapl-rally" {
		t.Errorf("Expected name 'aapl-rally', got '%s'", fixture.Name)
	}

	if fixture.Timestamp.IsZero() {
		t.Error("Fixture timestamp is zero")
	}

	if len(fixture.MarketData) == 0 {
		t.Fatal("Fixture has no market data")
	}

	// Validate AAPL data
	aapl, ok := fixture.MarketData["AAPL"]
	if !ok {
		t.Fatal("AAPL data not found in fixture")
	}

	if aapl.Price <= 0 {
		t.Errorf("Invalid price: %f", aapl.Price)
	}

	if aapl.Volume <= 0 {
		t.Errorf("Invalid volume: %d", aapl.Volume)
	}

	// Validate indicators
	if aapl.Indicators.RSI < 0 || aapl.Indicators.RSI > 100 {
		t.Errorf("Invalid RSI: %f", aapl.Indicators.RSI)
	}

	if aapl.Indicators.ATR <= 0 {
		t.Errorf("Invalid ATR: %f", aapl.Indicators.ATR)
	}
}

func TestReplayStrategy(t *testing.T) {
	fixture, err := LoadFixture("aapl-rally")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	// Simple strategy function for testing
	strategyFunc := func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error) {
		// Simple RSI-based strategy
		signal := strategies.Signal{
			Symbol:     input.Symbol,
			Timestamp:  input.Timestamp,
			Type:       strategies.SignalHold,
			Confidence: 0.0,
			Indicators: make(map[string]interface{}),
		}

		if input.RSI > 70 {
			signal.Type = strategies.SignalSell
			signal.Confidence = 0.7
			signal.EntryPrice = input.Price
			signal.StopLoss = input.Price + (2.0 * input.ATR)
			signal.TakeProfit = []float64{input.Price - (2.0 * input.ATR)}
			signal.Reason = "RSI overbought"
		} else if input.RSI < 30 {
			signal.Type = strategies.SignalBuy
			signal.Confidence = 0.7
			signal.EntryPrice = input.Price
			signal.StopLoss = input.Price - (2.0 * input.ATR)
			signal.TakeProfit = []float64{input.Price + (2.0 * input.ATR)}
			signal.Reason = "RSI oversold"
		} else if input.RSI > 60 && input.RSI <= 70 {
			signal.Type = strategies.SignalBuy
			signal.Confidence = 0.65
			signal.EntryPrice = input.Price
			signal.StopLoss = input.Price - (1.5 * input.ATR)
			signal.TakeProfit = []float64{input.Price + (2.0 * input.ATR)}
			signal.Reason = "RSI bullish momentum"
		}

		return signal, nil
	}

	result, err := ReplayStrategy(context.Background(), strategyFunc, fixture)
	if err != nil {
		t.Fatalf("ReplayStrategy() error = %v", err)
	}

	if result.Signal.Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", result.Signal.Symbol)
	}

	if result.Signal.Type == "" {
		t.Error("Signal type is empty")
	}

	// Duration should be non-negative (may be very small)
	if result.Duration < 0 {
		t.Error("Duration is negative")
	}

	// Verify metadata
	if result.Metadata["fixture_name"] != "aapl-rally" {
		t.Errorf("Expected fixture_name 'aapl-rally', got %v", result.Metadata["fixture_name"])
	}
}

func TestDeterminism(t *testing.T) {
	fixtures := []string{"aapl-rally", "msft-consolidation", "tsla-volatility"}

	for _, fixtureName := range fixtures {
		t.Run(fixtureName, func(t *testing.T) {
			fixture, err := LoadFixture(fixtureName)
			if err != nil {
				t.Fatalf("Failed to load fixture: %v", err)
			}

			// Deterministic strategy function
			strategyFunc := func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error) {
				signal := strategies.Signal{
					Symbol:     input.Symbol,
					Timestamp:  input.Timestamp,
					Type:       strategies.SignalHold,
					Confidence: 0.5,
					Indicators: make(map[string]interface{}),
				}

				// Deterministic logic based on RSI and moving averages
				if input.RSI > 70 {
					signal.Type = strategies.SignalSell
					signal.Confidence = 0.75
				} else if input.RSI < 30 {
					signal.Type = strategies.SignalBuy
					signal.Confidence = 0.75
				} else if input.SMA20 > input.SMA50 && input.SMA50 > input.SMA200 {
					signal.Type = strategies.SignalBuy
					signal.Confidence = 0.65
				} else if input.SMA20 < input.SMA50 && input.SMA50 < input.SMA200 {
					signal.Type = strategies.SignalSell
					signal.Confidence = 0.65
				}

				// Calculate price levels
				signal.EntryPrice = input.Price
				if signal.Type == strategies.SignalBuy {
					signal.StopLoss = input.Price - (2.0 * input.ATR)
					signal.TakeProfit = []float64{
						input.Price + (2.0 * input.ATR),
						input.Price + (3.0 * input.ATR),
					}
				} else if signal.Type == strategies.SignalSell {
					signal.StopLoss = input.Price + (2.0 * input.ATR)
					signal.TakeProfit = []float64{
						input.Price - (2.0 * input.ATR),
						input.Price - (3.0 * input.ATR),
					}
				}

				signal.Reason = "Test deterministic strategy"
				return signal, nil
			}

			// Run 10 times and verify determinism
			err = VerifyDeterminism(context.Background(), strategyFunc, fixture, 10)
			if err != nil {
				t.Errorf("Determinism verification failed: %v", err)
			}
		})
	}
}

func TestVerifyDeterminismInvalidRuns(t *testing.T) {
	fixture, err := LoadFixture("aapl-rally")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	strategyFunc := func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error) {
		return strategies.Signal{}, nil
	}

	// Test with invalid runs parameter
	err = VerifyDeterminism(context.Background(), strategyFunc, fixture, 1)
	if err == nil {
		t.Error("Expected error for runs < 2, got nil")
	}
}

func TestInferMarketTrend(t *testing.T) {
	tests := []struct {
		name     string
		snapshot MarketSnapshot
		want     string
	}{
		{
			name: "bullish trend",
			snapshot: MarketSnapshot{
				Indicators: IndicatorSnapshot{
					SMA20:  110.0,
					SMA50:  105.0,
					SMA200: 100.0,
				},
			},
			want: "bullish",
		},
		{
			name: "bearish trend",
			snapshot: MarketSnapshot{
				Indicators: IndicatorSnapshot{
					SMA20:  100.0,
					SMA50:  105.0,
					SMA200: 110.0,
				},
			},
			want: "bearish",
		},
		{
			name: "neutral trend",
			snapshot: MarketSnapshot{
				Indicators: IndicatorSnapshot{
					SMA20:  105.0,
					SMA50:  100.0,
					SMA200: 110.0,
				},
			},
			want: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMarketTrend(tt.snapshot)
			if got != tt.want {
				t.Errorf("inferMarketTrend() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplayWithRealStrategy(t *testing.T) {
	fixture, err := LoadFixture("tsla-volatility")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	// Test with RSI momentum strategy
	strategyFunc := func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error) {
		strategy := strategies.NewRSIMomentumStrategy()
		return strategy.Analyze(ctx, input)
	}

	result, err := ReplayStrategy(context.Background(), strategyFunc, fixture)
	if err != nil {
		t.Fatalf("ReplayStrategy() with RSI strategy error = %v", err)
	}

	// TSLA volatility fixture has RSI=78.5, which should trigger a sell signal
	if result.Signal.Type != strategies.SignalSell {
		t.Errorf("Expected sell signal for RSI=78.5 (overbought), got %s", result.Signal.Type)
	}
}

func TestReplayWithMAStrategy(t *testing.T) {
	fixture, err := LoadFixture("aapl-rally")
	if err != nil {
		t.Fatalf("Failed to load fixture: %v", err)
	}

	// Test with MA crossover strategy
	strategyFunc := func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error) {
		strategy := strategies.NewMACrossoverStrategy()
		return strategy.Analyze(ctx, input)
	}

	result, err := ReplayStrategy(context.Background(), strategyFunc, fixture)
	if err != nil {
		t.Fatalf("ReplayStrategy() with MA strategy error = %v", err)
	}

	// AAPL rally has SMA20 > SMA50 > SMA200 and price > SMA20, should be buy
	if result.Signal.Type != strategies.SignalBuy {
		t.Errorf("Expected buy signal for golden cross, got %s", result.Signal.Type)
	}
}
