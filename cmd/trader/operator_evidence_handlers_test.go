package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOperatorEvidenceOverviewReportsSafePaperRuntimeWithoutInventedActivity(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("ALLOW_LIVE_TRADING", "false")
	t.Setenv("EXECUTION_ENABLED", "false")
	t.Setenv("EXECUTION_INSTRUCTION_WORKER_ENABLED", "false")
	t.Setenv("BROKER_EXECUTION_ALLOWED", "false")
	t.Setenv("MAX_LEVERAGE", "1")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/operator-evidence/overview", nil)
	res := httptest.NewRecorder()
	operatorEvidenceOverviewHandler(nil)(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	for _, want := range []string{`"runtimeMode":"paper"`, `"allowLiveTrading":false`, `"executionEnabled":false`, `"maximumLeverage":1`, `"genuineEvents":0`} {
		if !strings.Contains(res.Body.String(), want) {
			t.Fatalf("body missing %s: %s", want, res.Body.String())
		}
	}
}

func TestOperatorEvidenceEndpointsAreReadOnly(t *testing.T) {
	for _, handler := range []http.HandlerFunc{
		operatorEvidenceOverviewHandler(nil),
		operatorCandidateEvidenceHandler(nil),
		worldMonitorResearchInboxHandler(nil),
	} {
		res := httptest.NewRecorder()
		handler(res, httptest.NewRequest(http.MethodPost, "/api/v1/operator-evidence/candidates/00000000-0000-0000-0000-000000000000", nil))
		if res.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d, want 405", res.Code)
		}
	}
}
