package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"jax-trading-assistant/libs/contracts/domain"
)

func TestHTTPTradeExecutor_ExecuteSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/execute" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"id": "ord-123",
			"symbol": "AAPL",
			"type": "limit",
			"side": "buy",
			"quantity": 100,
			"price": 150.25,
			"time_in_force": "day",
			"status": "submitted",
			"submitted_at": "2026-02-13T10:30:00Z",
			"filled_quantity": 0
		}]`))
	}))
	defer server.Close()

	client := NewHTTPTradeExecutor(server.URL)
	signal := domain.Signal{
		ID:         "sig-123",
		Symbol:     "AAPL",
		Type:       "buy",
		EntryPrice: 150.25,
	}

	orders, err := client.ExecuteSignal(context.Background(), signal)
	if err != nil {
		t.Fatalf("ExecuteSignal failed: %v", err)
	}

	if len(orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(orders))
	}
	if orders[0].Symbol != "AAPL" {
		t.Errorf("Expected symbol AAPL, got %s", orders[0].Symbol)
	}
}

func TestHTTPTradeExecutor_GetPositions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/positions" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"symbol": "AAPL",
			"quantity": 100,
			"avg_entry_price": 150.25,
			"current_price": 152.50,
			"unrealized_pl": 225.00,
			"realized_pl": 0.00,
			"opened_at": "2026-02-10T09:30:00Z",
			"last_updated": "2026-02-13T10:30:00Z"
		}]`))
	}))
	defer server.Close()

	client := NewHTTPTradeExecutor(server.URL)
	positions, err := client.GetPositions(context.Background(), "acc-123")
	if err != nil {
		t.Fatalf("GetPositions failed: %v", err)
	}

	if len(positions) != 1 {
		t.Errorf("Expected 1 position, got %d", len(positions))
	}
}

func TestHTTPTradeExecutor_GetPortfolio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/portfolio" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"account_id": "acc-123",
			"cash": 50000.00,
			"buying_power": 200000.00,
			"positions": [],
			"total_value": 50000.00,
			"total_pl": 0.00,
			"last_updated": "2026-02-13T10:30:00Z"
		}`))
	}))
	defer server.Close()

	client := NewHTTPTradeExecutor(server.URL)
	portfolio, err := client.GetPortfolio(context.Background(), "acc-123")
	if err != nil {
		t.Fatalf("GetPortfolio failed: %v", err)
	}

	if portfolio.AccountID != "acc-123" {
		t.Errorf("Expected account acc-123, got %s", portfolio.AccountID)
	}
	if portfolio.Cash != 50000.00 {
		t.Errorf("Expected cash 50000.00, got %f", portfolio.Cash)
	}
}
