package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemMarketDataStatusHandlerReturnsBridgeStatus(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/market-data-status", nil)
	rec := httptest.NewRecorder()
	systemMarketDataStatusHandler(mt).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Connected      bool   `json:"connected"`
		MarketDataMode string `json:"marketDataMode"`
		PaperTrading   bool   `json:"paperTrading"`
		CheckedAt      string `json:"checkedAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if !payload.Connected {
		t.Fatal("expected connected=true")
	}
	if payload.MarketDataMode != "delayed" {
		t.Fatalf("marketDataMode = %q, want delayed", payload.MarketDataMode)
	}
	if !payload.PaperTrading {
		t.Fatal("expected paperTrading=true")
	}
	if payload.CheckedAt == "" {
		t.Fatal("expected checkedAt to be populated")
	}
}

func TestSystemMarketDataStatusHandlerReturnsServiceUnavailableWhenBridgeFails(t *testing.T) {
	mt := &marketTools{
		httpClient:  http.DefaultClient,
		ibBridgeURL: "http://127.0.0.1:1",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/market-data-status", nil)
	rec := httptest.NewRecorder()
	systemMarketDataStatusHandler(mt).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSystemDiagnosticsHandlerReturnsMonitorAndMarketStatus(t *testing.T) {
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connected":        false,
			"market_data_mode": "delayed",
			"paper_trading":    true,
		})
	}))
	defer bridge.Close()

	mt := &marketTools{
		httpClient:  bridge.Client(),
		ibBridgeURL: bridge.URL,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
	rec := httptest.NewRecorder()
	systemDiagnosticsHandler(nil, mt).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		WorldMonitor struct {
			Connected bool `json:"connected"`
			Counts    struct {
				Total int `json:"total"`
			} `json:"counts"`
		} `json:"worldMonitor"`
		MarketData struct {
			Connected      bool   `json:"connected"`
			MarketDataMode string `json:"marketDataMode"`
			PaperTrading   bool   `json:"paperTrading"`
		} `json:"marketData"`
		CheckedAt string `json:"checkedAt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.WorldMonitor.Connected {
		t.Fatal("expected empty world monitor status to report connected=false")
	}
	if payload.MarketData.MarketDataMode != "delayed" {
		t.Fatalf("marketDataMode = %q, want delayed", payload.MarketData.MarketDataMode)
	}
	if !payload.MarketData.PaperTrading {
		t.Fatal("expected paperTrading=true")
	}
	if payload.CheckedAt == "" {
		t.Fatal("expected checkedAt")
	}
}
