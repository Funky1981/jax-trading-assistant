package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrokerOrdersPostBlocksManualETFEntry(t *testing.T) {
	t.Setenv("TRADING_PILOT_MODE", "true")
	t.Setenv("TRADING_PILOT_ALLOWED_ROLES", "unauthenticated")

	bridgeCalls := 0
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalls++
		switch r.URL.Path {
		case "/ready":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"connected":true,"paper_trading":true,"market_data_mode":"real-time"}`))
		default:
			http.Error(w, "manual ETF order reached bridge", http.StatusInternalServerError)
		}
	}))
	defer bridge.Close()

	mt := &marketTools{
		httpClient:  bridge.Client(),
		ibBridgeURL: bridge.URL,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/broker/orders", strings.NewReader(`{"symbol":"SPY","action":"BUY","quantity":1,"order_type":"MKT"}`))
	rec := httptest.NewRecorder()
	brokerOrdersHandler(true, mt, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "manual ETF entry orders must use the approval workflow") {
		t.Fatalf("expected manual ETF rejection reason, got %s", rec.Body.String())
	}
	if bridgeCalls != 1 {
		t.Fatalf("expected only readiness probe to hit bridge, got %d bridge calls", bridgeCalls)
	}
}

func TestBrokerBracketPostBlocksManualETFEntry(t *testing.T) {
	t.Setenv("TRADING_PILOT_MODE", "true")
	t.Setenv("TRADING_PILOT_ALLOWED_ROLES", "unauthenticated")

	bridgeCalls := 0
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bridgeCalls++
		switch r.URL.Path {
		case "/ready":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"connected":true,"paper_trading":true,"market_data_mode":"real-time"}`))
		default:
			http.Error(w, "manual ETF bracket reached bridge", http.StatusInternalServerError)
		}
	}))
	defer bridge.Close()

	mt := &marketTools{
		httpClient:  bridge.Client(),
		ibBridgeURL: bridge.URL,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/broker/orders/bracket", strings.NewReader(`{"symbol":"SPY","action":"BUY","quantity":1,"entry_order_type":"MKT","stop_loss":650}`))
	rec := httptest.NewRecorder()
	brokerOrderDetailHandler(true, mt, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "manual ETF entry orders must use the approval workflow") {
		t.Fatalf("expected manual ETF rejection reason, got %s", rec.Body.String())
	}
	if bridgeCalls != 1 {
		t.Fatalf("expected only readiness probe to hit bridge, got %d bridge calls", bridgeCalls)
	}
}
