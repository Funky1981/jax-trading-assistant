package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIScannerHandlerGetDefaultState(t *testing.T) {
	globalAIScannerStore.reset()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/scanner", nil)
	rec := httptest.NewRecorder()
	aiScannerHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload aiScannerState
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode scanner payload: %v", err)
	}
	if payload.IntervalSeconds <= 0 {
		t.Fatalf("expected positive intervalSeconds, got %d", payload.IntervalSeconds)
	}
	if payload.Sentiment.Mode == "" {
		t.Fatal("expected sentiment mode to be set")
	}
}

func TestAIScannerHandlerRejectsInvalidPayload(t *testing.T) {
	globalAIScannerStore.reset()

	bad := aiScannerState{
		Enabled:           true,
		AssetScope:        "etf",
		Symbols:           []string{"SPY"},
		UniversePreset:    "etf-core",
		IntervalSeconds:   0,
		MinimumConfidence: 2,
		Sentiment: aiScannerSentimentState{
			Enabled:                  true,
			SourceScope:              "news",
			Window:                   "24h",
			Threshold:                2,
			MinimumSourceCount:       0,
			SourceTrustWeightingMode: "",
			Mode:                     "",
		},
		Channels: aiScannerChannels{InApp: true},
		Policy: aiScannerPolicy{RequiresHumanApproval: true},
	}
	body, _ := json.Marshal(bad)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/ai/scanner", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	aiScannerHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("minimumConfidence")) {
		t.Fatalf("expected validation details in body, got %s", rec.Body.String())
	}
}

func TestAIScannerHandlerPersistsValidatedState(t *testing.T) {
	globalAIScannerStore.reset()

	payload := defaultAIScannerState()
	payload.Enabled = false
	payload.IntervalSeconds = 120
	payload.MinimumConfidence = 0.55
	payload.Sentiment.Enabled = true
	payload.Sentiment.Mode = "rank_boost"
	body, _ := json.Marshal(payload)

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/ai/scanner", bytes.NewReader(body))
	putRec := httptest.NewRecorder()
	aiScannerHandler().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("put status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/ai/scanner", nil)
	getRec := httptest.NewRecorder()
	aiScannerHandler().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, body = %s", getRec.Code, getRec.Body.String())
	}

	var stored aiScannerState
	if err := json.Unmarshal(getRec.Body.Bytes(), &stored); err != nil {
		t.Fatalf("decode stored scanner payload: %v", err)
	}
	if stored.Enabled {
		t.Fatal("expected scanner enabled=false after update")
	}
	if stored.Status != "disabled" {
		t.Fatalf("expected disabled status after disabling scanner, got %q", stored.Status)
	}
	if stored.NextScanAt == nil || stored.LastScanCompletedAt == nil {
		t.Fatal("expected scan timestamps to be set after update")
	}
}

func TestAIOverviewHandlerReturnsCoherentModelWithoutDB(t *testing.T) {
	globalAIScannerStore.reset()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ai/overview", nil)
	rec := httptest.NewRecorder()
	aiOverviewHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload aiOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode overview payload: %v", err)
	}
	if payload.Scanner.AssetScope == "" {
		t.Fatal("expected scanner in overview payload")
	}
	if _, ok := payload.OpportunityCounts["signalsPending"]; !ok {
		t.Fatalf("expected signalsPending count, got %+v", payload.OpportunityCounts)
	}
	if _, ok := payload.PolicySummary["requiresHumanApproval"]; !ok {
		t.Fatalf("expected policy summary fields, got %+v", payload.PolicySummary)
	}
}
