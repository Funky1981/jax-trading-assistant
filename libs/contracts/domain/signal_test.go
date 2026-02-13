package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSignal_JSONMarshaling(t *testing.T) {
	signal := Signal{
		ID:         "sig-123",
		Symbol:     "AAPL",
		Timestamp:  time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		Type:       "buy",
		Confidence: 0.85,
		EntryPrice: 150.25,
		StopLoss:   145.00,
		TakeProfit: []float64{155.00, 160.00, 165.00},
		Reason:     "Strong uptrend with RSI oversold",
		StrategyID: "rsi-momentum",
		Indicators: map[string]interface{}{
			"rsi":  35.5,
			"macd": 1.25,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(signal)
	if err != nil {
		t.Fatalf("Failed to marshal signal: %v", err)
	}

	// Unmarshal back
	var decoded Signal
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal signal: %v", err)
	}

	// Verify fields
	if decoded.ID != signal.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, signal.ID)
	}
	if decoded.Symbol != signal.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", decoded.Symbol, signal.Symbol)
	}
	if decoded.Type != signal.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, signal.Type)
	}
	if decoded.Confidence != signal.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", decoded.Confidence, signal.Confidence)
	}
	if len(decoded.TakeProfit) != len(signal.TakeProfit) {
		t.Errorf("TakeProfit length mismatch: got %d, want %d", len(decoded.TakeProfit), len(signal.TakeProfit))
	}
}

func TestSignal_ZeroValue(t *testing.T) {
	var signal Signal

	if signal.ID != "" {
		t.Errorf("Zero value ID should be empty string")
	}
	if signal.Confidence != 0.0 {
		t.Errorf("Zero value Confidence should be 0.0")
	}
	if signal.Indicators != nil {
		t.Errorf("Zero value Indicators should be nil")
	}
}

func TestSignal_PartialData(t *testing.T) {
	jsonData := `{
		"id": "sig-456",
		"symbol": "TSLA",
		"timestamp": "2026-02-13T10:30:00Z",
		"type": "sell",
		"confidence": 0.75
	}`

	var signal Signal
	if err := json.Unmarshal([]byte(jsonData), &signal); err != nil {
		t.Fatalf("Failed to unmarshal partial signal: %v", err)
	}

	if signal.ID != "sig-456" {
		t.Errorf("ID mismatch: got %s, want sig-456", signal.ID)
	}
	if signal.EntryPrice != 0.0 {
		t.Errorf("EntryPrice should be 0.0 for missing field")
	}
	if signal.Reason != "" {
		t.Errorf("Reason should be empty for missing field")
	}
}
