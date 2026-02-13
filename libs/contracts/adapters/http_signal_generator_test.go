package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPSignalGenerator_GenerateSignals(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/signals/generate" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Unexpected method: %s", r.Method)
		}

		// Return mock signals
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"id": "sig-123",
			"symbol": "AAPL",
			"timestamp": "2026-02-13T10:30:00Z",
			"type": "buy",
			"confidence": 0.85,
			"entry_price": 150.25,
			"stop_loss": 145.00,
			"take_profit": [155.00, 160.00],
			"reason": "Strong uptrend",
			"strategy_id": "rsi-momentum",
			"indicators": {"rsi": 35.5}
		}]`))
	}))
	defer server.Close()

	client := NewHTTPSignalGenerator(server.URL)
	signals, err := client.GenerateSignals(context.Background(), []string{"AAPL"})
	if err != nil {
		t.Fatalf("GenerateSignals failed: %v", err)
	}

	if len(signals) != 1 {
		t.Errorf("Expected 1 signal, got %d", len(signals))
	}
	if signals[0].Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", signals[0].Symbol)
	}
}

func TestHTTPSignalGenerator_Health(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer server.Close()

	client := NewHTTPSignalGenerator(server.URL)
	err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

func TestHTTPSignalGenerator_HealthUnhealthy(t *testing.T) {
	// Mock HTTP server returning unhealthy
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "unhealthy"}`))
	}))
	defer server.Close()

	client := NewHTTPSignalGenerator(server.URL)
	err := client.Health(context.Background())
	if err == nil {
		t.Fatal("Expected health check to fail")
	}
}

func TestHTTPSignalGenerator_GetSignalHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/signals" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		symbol := r.URL.Query().Get("symbol")
		if symbol != "TSLA" {
			t.Errorf("Expected symbol TSLA, got %s", symbol)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"id": "sig-456",
			"symbol": "TSLA",
			"timestamp": "2026-02-13T09:30:00Z",
			"type": "sell",
			"confidence": 0.75,
			"entry_price": 200.00,
			"stop_loss": 205.00,
			"take_profit": [195.00, 190.00],
			"reason": "Bearish divergence",
			"strategy_id": "macd-cross",
			"indicators": {}
		}]`))
	}))
	defer server.Close()

	client := NewHTTPSignalGenerator(server.URL)
	signals, err := client.GetSignalHistory(context.Background(), "TSLA", 10)
	if err != nil {
		t.Fatalf("GetSignalHistory failed: %v", err)
	}

	if len(signals) != 1 {
		t.Errorf("Expected 1 signal, got %d", len(signals))
	}
}
