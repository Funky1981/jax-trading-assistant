package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvaluateETFPhase1Eligibility(t *testing.T) {
	policy := defaultETFInstrumentPolicy()

	approved := evaluateETFPhase1Eligibility(policy, "SPY", "paper")
	if !approved.Allowed || !approved.IsETF {
		t.Fatalf("expected SPY to be allowed ETF, got %+v", approved)
	}

	excluded := evaluateETFPhase1Eligibility(policy, "SQQQ", "paper")
	if excluded.Allowed {
		t.Fatalf("expected SQQQ to be excluded, got %+v", excluded)
	}
	if excluded.ReasonCode != "etf_class_excluded" {
		t.Fatalf("unexpected reason code %q", excluded.ReasonCode)
	}

	unknown := evaluateETFPhase1Eligibility(policy, "AAPL", "paper")
	if !unknown.Allowed || unknown.IsETF {
		t.Fatalf("expected AAPL to pass as non-ETF entry, got %+v", unknown)
	}
}

func TestTradingETFPolicyHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trading/etf-policy", nil)
	rec := httptest.NewRecorder()

	tradingETFPolicyHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var payload tradingETFPolicyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(payload.ApprovedETFs) == 0 {
		t.Fatal("expected approved ETF list")
	}
	if len(payload.ExcludedETFs) == 0 {
		t.Fatal("expected excluded ETF list")
	}
}
