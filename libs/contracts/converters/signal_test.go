package converters

import (
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/domain"
	"jax-trading-assistant/libs/strategies"
)

func TestSignalToDomain(t *testing.T) {
	stratSig := strategies.Signal{
		Type:       strategies.SignalBuy,
		Symbol:     "AAPL",
		Timestamp:  time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		Confidence: 0.85,
		EntryPrice: 150.25,
		StopLoss:   145.00,
		TakeProfit: []float64{155.00, 160.00},
		Reason:     "Strong uptrend",
		Indicators: map[string]interface{}{"rsi": 35.5},
	}

	domainSig := SignalToDomain("rsi-momentum", stratSig)

	if domainSig.Symbol != stratSig.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", domainSig.Symbol, stratSig.Symbol)
	}
	if domainSig.Type != "buy" {
		t.Errorf("Type mismatch: got %s, want buy", domainSig.Type)
	}
	if domainSig.Confidence != stratSig.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", domainSig.Confidence, stratSig.Confidence)
	}
	if domainSig.StrategyID != "rsi-momentum" {
		t.Errorf("StrategyID mismatch: got %s, want rsi-momentum", domainSig.StrategyID)
	}
	if len(domainSig.TakeProfit) != len(stratSig.TakeProfit) {
		t.Errorf("TakeProfit length mismatch: got %d, want %d", len(domainSig.TakeProfit), len(stratSig.TakeProfit))
	}
}

func TestSignalFromDomain(t *testing.T) {
	domainSig := domain.Signal{
		ID:         "sig-123",
		Symbol:     "TSLA",
		Timestamp:  time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		Type:       "sell",
		Confidence: 0.75,
		EntryPrice: 200.00,
		StopLoss:   205.00,
		TakeProfit: []float64{195.00, 190.00},
		Reason:     "Bearish divergence",
		StrategyID: "macd-cross",
		Indicators: map[string]interface{}{"macd": -1.5},
	}

	stratSig := SignalFromDomain(domainSig)

	if stratSig.Symbol != domainSig.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", stratSig.Symbol, domainSig.Symbol)
	}
	if string(stratSig.Type) != domainSig.Type {
		t.Errorf("Type mismatch: got %s, want %s", stratSig.Type, domainSig.Type)
	}
	if stratSig.Confidence != domainSig.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", stratSig.Confidence, domainSig.Confidence)
	}
}

func TestSignalRoundTrip(t *testing.T) {
	original := strategies.Signal{
		Type:       strategies.SignalBuy,
		Symbol:     "AAPL",
		Timestamp:  time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		Confidence: 0.85,
		EntryPrice: 150.25,
		StopLoss:   145.00,
		TakeProfit: []float64{155.00, 160.00},
		Reason:     "Test",
		Indicators: map[string]interface{}{"test": 1.0},
	}

	// Convert to domain and back
	domainSig := SignalToDomain("test-strategy", original)
	converted := SignalFromDomain(domainSig)

	// Verify round-trip preserves data
	if converted.Symbol != original.Symbol {
		t.Errorf("Round-trip Symbol mismatch")
	}
	if converted.Type != original.Type {
		t.Errorf("Round-trip Type mismatch")
	}
	if converted.Confidence != original.Confidence {
		t.Errorf("Round-trip Confidence mismatch")
	}
}
