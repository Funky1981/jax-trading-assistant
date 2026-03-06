package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTradingPilotStatusHandlerDefaultsToReadOnlyWithoutAuth(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connected":        true,
			"market_data_mode": "delayed",
			"paper_trading":    true,
		})
	}))
	defer bridge.Close()

	mt := &marketTools{
		httpClient:  bridge.Client(),
		ibBridgeURL: bridge.URL,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trading/pilot-status", nil)
	rec := httptest.NewRecorder()
	tradingPilotStatusHandler(false, mt).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload tradingPilotStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.ReadOnly {
		t.Fatal("expected readOnly=true when auth is disabled")
	}
	if payload.CanTrade {
		t.Fatal("expected canTrade=false when auth is disabled")
	}
	if payload.OperatorAccess {
		t.Fatal("expected operatorAccess=false when auth is disabled")
	}
	if !payload.BrokerConnected || !payload.PaperTrading {
		t.Fatal("expected broker connection and paper trading to reflect bridge health")
	}
}

func TestBrokerOrdersPostBlockedWhenPilotIsReadOnly(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "should not reach bridge", http.StatusInternalServerError)
	}))
	defer bridge.Close()

	mt := &marketTools{
		httpClient:  bridge.Client(),
		ibBridgeURL: bridge.URL,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/broker/orders", strings.NewReader(`{"symbol":"AAPL"}`))
	rec := httptest.NewRecorder()
	brokerOrdersHandler(false, mt, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "read-only") {
		t.Fatalf("expected read-only response, got %s", rec.Body.String())
	}
}
