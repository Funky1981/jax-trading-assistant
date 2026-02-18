package signalgenerator

import (
	"context"
	"testing"

	"jax-trading-assistant/libs/strategies"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestNew verifies the constructor creates a valid instance
func TestNew(t *testing.T) {
	registry := strategies.NewRegistry()

	// Create mock pool (nil is acceptable for constructor test)
	var pool *pgxpool.Pool

	gen := New(pool, registry)

	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
	if gen.registry != registry {
		t.Error("registry not set correctly")
	}
}

// TestCalculateRSI tests RSI calculation with known values
func TestCalculateRSI(t *testing.T) {
	tests := []struct {
		name     string
		candles  []candle
		period   int
		expected float64
		delta    float64
	}{
		{
			name: "insufficient data",
			candles: []candle{
				{close: 100}, {close: 101},
			},
			period:   14,
			expected: 50.0, // Neutral
			delta:    0.01,
		},
		{
			name: "all gains",
			candles: func() []candle {
				c := make([]candle, 20)
				for i := range c {
					c[i].close = 100 + float64(i)
				}
				return c
			}(),
			period:   14,
			expected: 100.0,
			delta:    0.01,
		},
		{
			name: "mixed movements",
			candles: []candle{
				{close: 100.0}, {close: 101.0}, {close: 100.5}, {close: 102.0},
				{close: 101.5}, {close: 103.0}, {close: 102.5}, {close: 104.0},
				{close: 103.5}, {close: 105.0}, {close: 104.5}, {close: 106.0},
				{close: 105.5}, {close: 107.0}, {close: 106.5}, {close: 108.0},
			},
			period:   14,
			expected: 70.0, // Approximate - mostly uptrend
			delta:    10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateRSI(tt.candles, tt.period)
			if absFloat(result-tt.expected) > tt.delta {
				t.Errorf("calculateRSI() = %.2f, want %.2f (±%.2f)", result, tt.expected, tt.delta)
			}
		})
	}
}

// TestCalculateSMA tests simple moving average
func TestCalculateSMA(t *testing.T) {
	candles := []candle{
		{close: 10.0},
		{close: 20.0},
		{close: 30.0},
		{close: 40.0},
		{close: 50.0},
	}

	tests := []struct {
		name     string
		period   int
		expected float64
	}{
		{"SMA3", 3, 40.0}, // (30+40+50)/3
		{"SMA5", 5, 30.0}, // (10+20+30+40+50)/5
		{"SMA1", 1, 50.0}, // Just last value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSMA(candles, tt.period)
			if absFloat(result-tt.expected) > 0.01 {
				t.Errorf("calculateSMA(%d) = %.2f, want %.2f", tt.period, result, tt.expected)
			}
		})
	}
}

// TestCalculateATR tests Average True Range calculation
func TestCalculateATR(t *testing.T) {
	candles := []candle{
		{high: 105, low: 95, close: 100},
		{high: 110, low: 100, close: 108},
		{high: 112, low: 106, close: 110},
		{high: 115, low: 109, close: 113},
		{high: 116, low: 111, close: 114},
	}

	result := calculateATR(candles, 3)

	// ATR should be positive and reasonable
	if result <= 0 {
		t.Errorf("calculateATR() = %.2f, want positive value", result)
	}
	if result > 20 {
		t.Errorf("calculateATR() = %.2f, unexpectedly high", result)
	}
}

// TestCalculateBollingerBands tests Bollinger Bands calculation
func TestCalculateBollingerBands(t *testing.T) {
	candles := make([]candle, 20)
	for i := range candles {
		candles[i].close = 100.0 + float64(i%5) // Oscillating pattern
	}

	bb := calculateBollingerBands(candles, 20, 2.0)

	// Middle band should be the SMA
	expectedMiddle := calculateSMA(candles, 20)
	if absFloat(bb.Middle-expectedMiddle) > 0.01 {
		t.Errorf("Bollinger Middle = %.2f, want %.2f", bb.Middle, expectedMiddle)
	}

	// Upper should be above middle
	if bb.Upper <= bb.Middle {
		t.Errorf("Upper band (%.2f) should be > Middle (%.2f)", bb.Upper, bb.Middle)
	}

	// Lower should be below middle
	if bb.Lower >= bb.Middle {
		t.Errorf("Lower band (%.2f) should be < Middle (%.2f)", bb.Lower, bb.Middle)
	}

	// Bands should be symmetric
	upperDiff := bb.Upper - bb.Middle
	lowerDiff := bb.Middle - bb.Lower
	if absFloat(upperDiff-lowerDiff) > 0.01 {
		t.Errorf("Bands not symmetric: upper diff %.2f, lower diff %.2f", upperDiff, lowerDiff)
	}
}

// TestDetermineTrend tests market trend determination
func TestDetermineTrend(t *testing.T) {
	tests := []struct {
		name     string
		input    strategies.AnalysisInput
		expected string
	}{
		{
			name: "bullish trend",
			input: strategies.AnalysisInput{
				SMA20:  110.0,
				SMA50:  105.0,
				SMA200: 100.0,
			},
			expected: "bullish",
		},
		{
			name: "bearish trend",
			input: strategies.AnalysisInput{
				SMA20:  100.0,
				SMA50:  105.0,
				SMA200: 110.0,
			},
			expected: "bearish",
		},
		{
			name: "neutral trend",
			input: strategies.AnalysisInput{
				SMA20:  105.0,
				SMA50:  110.0,
				SMA200: 100.0,
			},
			expected: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineTrend(tt.input)
			if result != tt.expected {
				t.Errorf("determineTrend() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestHealthCheck tests the health check with mock dependencies
func TestHealthCheck(t *testing.T) {
	ctx := context.Background()
	registry := strategies.NewRegistry()

	// Register a test strategy
	testStrategy := strategies.NewRSIMomentumStrategy()
	registry.Register(testStrategy, testStrategy.GetMetadata())

	// Test with nil pool should fail
	genNilDB := New(nil, registry)
	err := genNilDB.Health(ctx)
	if err == nil {
		t.Error("expected health check to fail with nil pool")
	}
	if !contains(err.Error(), "database") {
		t.Errorf("expected database error, got: %v", err)
	}

	// Test with empty registry should fail (but requires valid db, so just verify constructor)
	emptyRegistry := strategies.NewRegistry()
	genEmpty := New(nil, emptyRegistry)
	if genEmpty == nil {
		t.Error("expected non-nil generator with empty registry")
	}
	// Note: Can't test empty registry health check without real database connection
	// The Health() method checks db first, which would fail with nil pool
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCalculateAvgVolume(t *testing.T) {
	candles := []candle{
		{volume: 1000},
		{volume: 2000},
		{volume: 3000},
		{volume: 4000},
		{volume: 5000},
	}

	tests := []struct {
		name     string
		period   int
		expected int64
	}{
		{"avg 3", 3, 4000}, // (3000+4000+5000)/3
		{"avg 5", 5, 3000}, // (1000+2000+3000+4000+5000)/5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAvgVolume(candles, tt.period)
			if result != tt.expected {
				t.Errorf("calculateAvgVolume(%d) = %d, want %d", tt.period, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkCalculateRSI(b *testing.B) {
	candles := make([]candle, 250)
	for i := range candles {
		candles[i].close = 100.0 + float64(i%20)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateRSI(candles, 14)
	}
}

func BenchmarkCalculateSMA(b *testing.B) {
	candles := make([]candle, 250)
	for i := range candles {
		candles[i].close = 100.0 + float64(i%20)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateSMA(candles, 50)
	}
}

func BenchmarkDetermineTrend(b *testing.B) {
	input := strategies.AnalysisInput{
		SMA20:  110.0,
		SMA50:  105.0,
		SMA200: 100.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		determineTrend(input)
	}
}
