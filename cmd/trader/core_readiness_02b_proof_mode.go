package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/libs/runtimepolicy"
)

const coreReadiness02BProofModeEnv = "CORE_READINESS_02B_PROOF_MODE"

func coreReadiness02BProofModeEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(envStr(coreReadiness02BProofModeEnv, "false")), "true")
}

// validateCoreReadiness02BProofMode permits only the isolated, Alpaca-only
// technical proof runtime. It is not a production strategy or execution mode.
func validateCoreReadiness02BProofMode(cfg Config) error {
	if !coreReadiness02BProofModeEnabled() {
		return fmt.Errorf("%s must be true", coreReadiness02BProofModeEnv)
	}
	if cfg.RuntimeMode != runtimepolicy.ModePaper {
		return fmt.Errorf("proof mode requires JAX_RUNTIME_MODE=PAPER")
	}
	dsn, err := url.Parse(strings.TrimSpace(cfg.DatabaseURL))
	if err != nil || dsn.Scheme != "postgres" && dsn.Scheme != "postgresql" {
		return fmt.Errorf("proof mode requires a PostgreSQL disposable database URL")
	}
	databaseName := strings.TrimPrefix(dsn.Path, "/")
	if databaseName != "jax_paper02r_test" && !strings.HasPrefix(databaseName, "jax_paper02r_test_") {
		return fmt.Errorf("proof mode refuses databases outside the jax_paper02r_test namespace")
	}
	if strings.TrimSpace(cfg.IBBridgeURL) != "" {
		return fmt.Errorf("proof mode requires IB_BRIDGE_URL to be empty")
	}
	providers := marketDataProviderConfigs(cfg.IBBridgeURL)
	if len(providers) != 1 || providers[0].Name != marketdata.ProviderAlpaca || !providers[0].Enabled {
		return fmt.Errorf("proof mode requires Alpaca as the only enabled market-data provider")
	}
	if cfg.ExecutionEnabled || !envIsFalse("EXECUTION_ENABLED") || !envIsFalse("BROKER_EXECUTION_ALLOWED") || !envIsFalse("ALLOW_LIVE_TRADING") ||
		!envIsFalse("EXECUTION_INSTRUCTION_WORKER_ENABLED") || !strings.EqualFold(strings.TrimSpace(envStr("EXECUTION_AUTHORITY", "")), "NONE") {
		return fmt.Errorf("proof mode requires PAPER-only execution authority NONE with broker/live execution disabled")
	}
	leverage, err := strconv.ParseFloat(strings.TrimSpace(envStr("MAX_LEVERAGE", "")), 64)
	if err != nil || leverage <= 0 || leverage > 1 {
		return fmt.Errorf("proof mode requires explicit MAX_LEVERAGE in (0,1]")
	}
	if cfg.WorkerGates.OpportunityScannerEnabled || cfg.WorkerGates.TradeWatcherEnabled || cfg.WorkerGates.MobileNotificationDispatcherEnabled || !cfg.WorkerGates.MarketIngesterEnabled {
		return fmt.Errorf("proof mode requires scanner, trade-watcher, and notification workers off and the bounded market ingester on")
	}
	policy, err := marketDataSafetyPolicyFromEnv()
	if err != nil {
		return fmt.Errorf("proof mode market-data policy invalid: %w", err)
	}
	if len(policy.AllowedSources) != 1 || policy.AllowedSources[0] != "alpaca" || policy.Timeframe != "1h" ||
		policy.QuoteMaxAge != time.Minute || policy.LatestCompletedCandleMaxAge != 90*time.Minute || policy.CandleHistoryLookback != 7*24*time.Hour {
		return fmt.Errorf("proof mode requires the reviewed Alpaca-only 1h/60s/90m/7d temporal policy")
	}
	if strings.TrimSpace(envStr("ALPACA_API_KEY", "")) == "" || strings.TrimSpace(envStr("ALPACA_API_SECRET", "")) == "" {
		return fmt.Errorf("proof mode requires the existing Alpaca credentials")
	}
	pullConfig, err := loadWorldMonitorPullConfig(os.LookupEnv)
	if err != nil {
		return fmt.Errorf("proof mode requires the existing genuine World Monitor pull provider: %w", err)
	}
	if !pullConfig.Enabled {
		return fmt.Errorf("proof mode requires WORLD_MONITOR_PULL_ENABLED=true")
	}
	if strings.TrimSpace(envStr("PAPER_ACCOUNT_ID", "")) == "" {
		return fmt.Errorf("proof mode requires a dedicated disposable PAPER_ACCOUNT_ID")
	}
	return nil
}

func envIsFalse(name string) bool {
	return strings.EqualFold(strings.TrimSpace(envStr(name, "")), "false")
}

func requireApprovedStrategiesForStartup(mode runtimepolicy.Mode, loaded int, proofMode bool) error {
	if !proofMode {
		return requireApprovedStrategies(mode, loaded)
	}
	if mode != runtimepolicy.ModePaper {
		return fmt.Errorf("02B proof mode is valid only in PAPER mode")
	}
	if loaded != 0 {
		return fmt.Errorf("02B proof mode refuses approved strategy artifacts")
	}
	return nil
}
