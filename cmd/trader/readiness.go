package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/instruments"
	"jax-trading-assistant/internal/trader/signalgenerator"
	"jax-trading-assistant/libs/runtimepolicy"
	"jax-trading-assistant/libs/strategies"
)

func handleReady(cfg Config, sigGen *signalgenerator.InProcessSignalGenerator, registry *strategies.Registry, mt *marketTools) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strategyCount := 0
		if registry != nil {
			strategyCount = len(registry.List())
		}

		statusCode, payload := evaluateTraderReadiness(
			r.Context(),
			cfg,
			strategyCount,
			func(ctx context.Context) error {
				if sigGen == nil {
					return errors.New("signal generator unavailable")
				}
				return sigGen.Health(ctx)
			},
			func(ctx context.Context) (map[string]any, error) {
				return probeIBBridgeReadiness(ctx, mt)
			},
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("handleReady encode: %v", err)
		}
	}
}

func evaluateTraderReadiness(
	ctx context.Context,
	cfg Config,
	strategyCount int,
	signalHealth func(context.Context) error,
	brokerCheck func(context.Context) (map[string]any, error),
) (int, map[string]any) {
	ready := true
	checks := map[string]any{}

	strategyCheck := map[string]any{
		"ok":    strategyCount > 0,
		"count": strategyCount,
	}
	if strategyCount == 0 {
		ready = false
		strategyCheck["error"] = "no approved strategies loaded"
	}
	checks["strategies"] = strategyCheck

	signalCheck := map[string]any{}
	if signalHealth == nil {
		ready = false
		signalCheck["ok"] = false
		signalCheck["error"] = "signal health checker unavailable"
	} else if err := signalHealth(ctx); err != nil {
		ready = false
		signalCheck["ok"] = false
		signalCheck["error"] = err.Error()
	} else {
		signalCheck["ok"] = true
	}
	checks["signalGenerator"] = signalCheck

	brokerRequired := cfg.RuntimeMode == runtimepolicy.ModePaper && cfg.ExecutionEnabled
	brokerStatus := map[string]any{
		"required": brokerRequired,
	}
	if brokerRequired {
		if brokerCheck == nil {
			ready = false
			brokerStatus["ok"] = false
			brokerStatus["error"] = "broker readiness checker unavailable"
		} else {
			probe, err := brokerCheck(ctx)
			for key, value := range probe {
				brokerStatus[key] = value
			}
			if err != nil {
				ready = false
				brokerStatus["ok"] = false
				if _, exists := brokerStatus["error"]; !exists {
					brokerStatus["error"] = err.Error()
				}
			} else if ok, _ := brokerStatus["ok"].(bool); !ok {
				ready = false
				if _, exists := brokerStatus["error"]; !exists {
					brokerStatus["error"] = "broker readiness failed"
				}
			}
		}
	} else {
		brokerStatus["ok"] = true
		brokerStatus["skipped"] = true
	}
	checks["broker"] = brokerStatus

	payload := map[string]any{
		"service":          "jax-trader",
		"version":          version,
		"status":           "not_ready",
		"ready":            false,
		"runtimeMode":      string(cfg.RuntimeMode),
		"executionEnabled": cfg.ExecutionEnabled,
		"strategiesLoaded": strategyCount,
		"checkedAt":        time.Now().UTC(),
		"uptime":           time.Since(startTime).String(),
		"checks":           checks,
	}
	if ready {
		payload["status"] = "ready"
		payload["ready"] = true
		return http.StatusOK, payload
	}
	return http.StatusServiceUnavailable, payload
}

func requireApprovedStrategies(mode runtimepolicy.Mode, loaded int) error {
	if loaded > 0 {
		return nil
	}
	if mode == runtimepolicy.ModePaper || mode == runtimepolicy.ModeLive {
		return fmt.Errorf("no approved strategies loaded; trader refuses to start in %s mode", mode)
	}
	return nil
}

func requireEventProviders(mode runtimepolicy.Mode, hasPolygon, hasFinnhub bool) error {
	if hasPolygon || hasFinnhub {
		return nil
	}
	if mode.EnforceStrictProviderPolicy() {
		return fmt.Errorf("enabled event-dependent strategies require POLYGON_API_KEY or FINNHUB_API_KEY in %s mode", mode)
	}
	return nil
}

func probeIBBridgeReadiness(ctx context.Context, mt *marketTools) (map[string]any, error) {
	result := map[string]any{
		"ok":       false,
		"required": true,
	}
	if mt == nil || strings.TrimSpace(mt.ibBridgeURL) == "" {
		result["error"] = "ib bridge url unavailable"
		return result, errors.New("ib bridge url unavailable")
	}

	endpoint := strings.TrimRight(mt.ibBridgeURL, "/") + "/ready"
	result["endpoint"] = endpoint

	bridgeStatus, statusCode, err := mt.getIBBridgeReadiness(ctx)
	if statusCode > 0 {
		result["statusCode"] = statusCode
	}
	if bridgeStatus != nil {
		result["connected"] = bridgeStatus.Connected
		result["marketDataMode"] = bridgeStatus.MarketDataMode
		result["paperTrading"] = bridgeStatus.PaperTrading
		result["quoteReady"] = bridgeStatus.QuoteReady
		if strings.TrimSpace(bridgeStatus.QuoteSymbol) != "" {
			result["quoteSymbol"] = bridgeStatus.QuoteSymbol
		}
		if strings.TrimSpace(bridgeStatus.QuoteError) != "" {
			result["quoteError"] = bridgeStatus.QuoteError
		}
	}
	if err != nil {
		result["error"] = err.Error()
		return result, err
	}

	result["ok"] = true
	return result, nil
}

func collectPaperRuntimeProbes(ctx context.Context, client *http.Client) map[string]map[string]any {
	if client == nil {
		client = http.DefaultClient
	}
	return map[string]map[string]any{
		"trader":   probeReadinessEndpoint(ctx, client, "trader", traderReadinessURL(), true),
		"ibBridge": probeReadinessEndpoint(ctx, client, "ibBridge", ibBridgeReadinessURL(), true),
		"research": probeReadinessEndpoint(ctx, client, "research", researchReadinessURL(), true),
	}
}

func applyPaperRuntimeProbes(summary map[string]any, probes map[string]map[string]any) {
	summary["runtimeProbes"] = probes

	failures := 0
	for _, probe := range probes {
		required, _ := probe["required"].(bool)
		ok, _ := probe["ok"].(bool)
		if required && !ok {
			failures++
		}
	}
	summary["runtimeProbeFailures"] = failures
	if failures > 0 {
		summary["ready"] = false
		summary["status"] = "not_ready"
	}
}

func etfPhase1ReadinessEvidence() map[string]any {
	catalog, err := instruments.LoadDefaultCatalog()
	catalogLoaded := err == nil
	automatedValidation := strings.EqualFold(strings.TrimSpace(os.Getenv("ETF_PHASE1_AUTOMATED_VALIDATION")), "passed")
	operatorUAT := strings.EqualFold(strings.TrimSpace(os.Getenv("ETF_PHASE1_OPERATOR_UAT")), "passed")
	paperPilotSignoff := strings.EqualFold(strings.TrimSpace(os.Getenv("ETF_PHASE1_PAPER_PILOT_SIGNOFF")), "passed")
	signoffs := map[string]bool{
		"engineering": envBoolDefault("ETF_PHASE1_ENGINEERING_SIGNOFF", false),
		"operations":  envBoolDefault("ETF_PHASE1_OPERATIONS_SIGNOFF", false),
		"tradingRisk": envBoolDefault("ETF_PHASE1_TRADING_RISK_SIGNOFF", false),
	}
	stages := []map[string]any{
		{"name": "implementation_complete", "passed": true},
		{"name": "automated_validation_green", "passed": automatedValidation, "evidence": "ETF_PHASE1_AUTOMATED_VALIDATION=passed"},
		{"name": "operator_uat_passed", "passed": operatorUAT, "evidence": "ETF_PHASE1_OPERATOR_UAT=passed"},
		{"name": "limited_paper_pilot_signed_off", "passed": paperPilotSignoff, "evidence": "ETF_PHASE1_PAPER_PILOT_SIGNOFF=passed"},
	}
	ready := catalogLoaded && automatedValidation && operatorUAT && paperPilotSignoff &&
		signoffs["engineering"] && signoffs["operations"] && signoffs["tradingRisk"]

	out := map[string]any{
		"status":               "not_ready",
		"ready":                ready,
		"catalogLoaded":        catalogLoaded,
		"entryWorkflow":        "candidate_approval_only",
		"paperOnly":            true,
		"manualEntriesBlocked": true,
		"liveTradingBlocked":   true,
		"stages":               stages,
		"signoffs":             signoffs,
	}
	if ready {
		out["status"] = "ready"
	}
	if err != nil {
		out["error"] = err.Error()
		return out
	}
	out["catalogVersion"] = catalog.Version
	out["catalogHash"] = catalog.Hash()
	out["allowlistCount"] = len(catalog.ETFList())
	out["policy"] = map[string]any{
		"quoteFreshnessSeconds": catalog.Policy.QuoteFreshnessSeconds,
		"maxSpreadBps":          catalog.Policy.MaxSpreadBps,
		"minBidSize":            catalog.Policy.MinBidSize,
		"minAskSize":            catalog.Policy.MinAskSize,
		"regularSession":        catalog.Policy.RegularSessionStart + "-" + catalog.Policy.RegularSessionEnd,
		"sessionTimezone":       catalog.Policy.SessionTimezone,
		"requireStopLoss":       catalog.Policy.RequireStopLoss,
		"requireFlattenByClose": catalog.Policy.RequireFlattenByClose,
	}
	return out
}

func probeReadinessEndpoint(ctx context.Context, client *http.Client, name, endpoint string, required bool) map[string]any {
	result := map[string]any{
		"name":       name,
		"url":        endpoint,
		"required":   required,
		"ok":         false,
		"statusCode": 0,
	}
	if !required {
		result["ok"] = true
		result["skipped"] = true
		return result
	}
	if strings.TrimSpace(endpoint) == "" {
		result["error"] = fmt.Sprintf("%s readiness url unavailable", name)
		return result
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	resp, err := client.Do(req)
	if err != nil {
		result["error"] = err.Error()
		return result
	}
	defer resp.Body.Close()

	result["statusCode"] = resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result["error"] = err.Error()
		return result
	}

	trimmedBody := strings.TrimSpace(string(body))
	if trimmedBody != "" {
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err == nil {
			result["response"] = decoded
			if resp.StatusCode != http.StatusOK {
				if message, ok := decoded["error"].(string); ok && strings.TrimSpace(message) != "" {
					result["error"] = message
				}
			}
		} else {
			result["body"] = trimmedBody
		}
	}

	if resp.StatusCode == http.StatusOK {
		result["ok"] = true
		return result
	}

	if _, exists := result["error"]; !exists {
		if trimmedBody != "" {
			result["error"] = trimmedBody
		} else {
			result["error"] = fmt.Sprintf("%s readiness returned %d", name, resp.StatusCode)
		}
	}
	return result
}

func traderReadinessURL() string {
	if raw := strings.TrimSpace(os.Getenv("JAX_TRADER_READY_URL")); raw != "" {
		return raw
	}
	return "http://localhost:" + envStr("PORT", "8100") + "/ready"
}

func ibBridgeReadinessURL() string {
	if raw := strings.TrimSpace(os.Getenv("IB_BRIDGE_READY_URL")); raw != "" {
		return raw
	}
	return strings.TrimRight(envStr("IB_BRIDGE_URL", "http://localhost:8092"), "/") + "/ready"
}

func researchReadinessURL() string {
	if raw := strings.TrimSpace(os.Getenv("JAX_RESEARCH_READY_URL")); raw != "" {
		return raw
	}
	return strings.TrimRight(envStr("JAX_ORCHESTRATOR_URL", "http://localhost:8091"), "/") + "/ready"
}
