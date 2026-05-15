package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"jax-trading-assistant/libs/runtimepolicy"
)

func TestEvaluateTraderReadinessFailsWhenPaperExecutionBrokerIsNotReady(t *testing.T) {
	t.Parallel()

	cfg := Config{
		RuntimeMode:      runtimepolicy.ModePaper,
		ExecutionEnabled: true,
		IBBridgeURL:      "http://ib-bridge:8092",
	}

	statusCode, payload := evaluateTraderReadiness(
		context.Background(),
		cfg,
		1,
		func(context.Context) error { return nil },
		func(context.Context) (map[string]any, error) {
			return map[string]any{
				"ok":       false,
				"endpoint": "http://ib-bridge:8092/ready",
			}, errors.New("ib bridge quote readiness failed")
		},
	)

	if statusCode != http.StatusServiceUnavailable {
		t.Fatalf("statusCode = %d, want %d", statusCode, http.StatusServiceUnavailable)
	}
	if ready, _ := payload["ready"].(bool); ready {
		t.Fatalf("expected trader readiness to fail, payload=%v", payload)
	}

	checks, ok := payload["checks"].(map[string]any)
	if !ok {
		t.Fatalf("checks missing from payload: %v", payload)
	}
	broker, ok := checks["broker"].(map[string]any)
	if !ok {
		t.Fatalf("broker check missing from payload: %v", payload)
	}
	if required, _ := broker["required"].(bool); !required {
		t.Fatalf("expected broker readiness to be required in paper execution mode: %v", broker)
	}
}

func TestApplyPaperRuntimeProbesOverridesHistoricalReadyState(t *testing.T) {
	t.Parallel()

	summary := map[string]any{
		"status": "ready",
		"ready":  true,
	}
	probes := map[string]map[string]any{
		"trader": {
			"ok":         false,
			"required":   true,
			"statusCode": http.StatusServiceUnavailable,
		},
		"ibBridge": {
			"ok":         true,
			"required":   true,
			"statusCode": http.StatusOK,
		},
		"research": {
			"ok":         true,
			"required":   true,
			"statusCode": http.StatusOK,
		},
	}

	applyPaperRuntimeProbes(summary, probes)

	if ready, _ := summary["ready"].(bool); ready {
		t.Fatalf("expected failed runtime probes to force ready=false, summary=%v", summary)
	}
	if status, _ := summary["status"].(string); status != "not_ready" {
		t.Fatalf("status = %q, want not_ready", status)
	}
	if failures, _ := summary["runtimeProbeFailures"].(int); failures != 1 {
		t.Fatalf("runtimeProbeFailures = %v, want 1", summary["runtimeProbeFailures"])
	}
}

func TestCollectPaperRuntimeProbesCapturesStatusCodesAndBodies(t *testing.T) {
	trader := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "strategies unavailable", http.StatusServiceUnavailable)
	}))
	defer trader.Close()

	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready","connected":true,"quote_ready":true}`))
	}))
	defer bridge.Close()

	research := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer research.Close()

	t.Setenv("JAX_TRADER_READY_URL", trader.URL+"/ready")
	t.Setenv("IB_BRIDGE_READY_URL", bridge.URL+"/ready")
	t.Setenv("JAX_RESEARCH_READY_URL", research.URL+"/ready")

	probes := collectPaperRuntimeProbes(context.Background(), http.DefaultClient)
	traderProbe, ok := probes["trader"]
	if !ok {
		t.Fatalf("expected trader probe, got %v", probes)
	}
	if probeOK, _ := traderProbe["ok"].(bool); probeOK {
		t.Fatalf("expected trader probe to fail, probe=%v", traderProbe)
	}
	if statusCode, _ := traderProbe["statusCode"].(int); statusCode != http.StatusServiceUnavailable {
		t.Fatalf("statusCode = %v, want %d", traderProbe["statusCode"], http.StatusServiceUnavailable)
	}
}

func TestETFPhase1ReadinessRequiresValidationUATAndSignoff(t *testing.T) {
	readiness := etfPhase1ReadinessEvidence()
	if ready, _ := readiness["ready"].(bool); ready {
		t.Fatalf("ETF readiness should not be ready without validation and sign-off evidence: %v", readiness)
	}
	if status, _ := readiness["status"].(string); status != "not_ready" {
		t.Fatalf("status = %q, want not_ready", status)
	}
	if workflow, _ := readiness["entryWorkflow"].(string); workflow != "candidate_approval_only" {
		t.Fatalf("entryWorkflow = %q, want candidate_approval_only", workflow)
	}
	signoffs, ok := readiness["signoffs"].(map[string]bool)
	if !ok {
		t.Fatalf("signoffs missing from readiness: %v", readiness)
	}
	if signoffs["engineering"] || signoffs["operations"] || signoffs["tradingRisk"] {
		t.Fatalf("expected signoffs to be false by default, got %v", signoffs)
	}
}

func TestETFPhase1ReadinessCanPassWithExplicitEvidence(t *testing.T) {
	t.Setenv("ETF_PHASE1_AUTOMATED_VALIDATION", "passed")
	t.Setenv("ETF_PHASE1_OPERATOR_UAT", "passed")
	t.Setenv("ETF_PHASE1_PAPER_PILOT_SIGNOFF", "passed")
	t.Setenv("ETF_PHASE1_ENGINEERING_SIGNOFF", "true")
	t.Setenv("ETF_PHASE1_OPERATIONS_SIGNOFF", "true")
	t.Setenv("ETF_PHASE1_TRADING_RISK_SIGNOFF", "true")

	readiness := etfPhase1ReadinessEvidence()
	if ready, _ := readiness["ready"].(bool); !ready {
		t.Fatalf("ETF readiness should be ready with explicit validation, UAT, pilot, and sign-off evidence: %v", readiness)
	}
	if status, _ := readiness["status"].(string); status != "ready" {
		t.Fatalf("status = %q, want ready", status)
	}
	if version, _ := readiness["catalogVersion"].(string); version == "" {
		t.Fatalf("catalogVersion missing from readiness: %v", readiness)
	}
}
