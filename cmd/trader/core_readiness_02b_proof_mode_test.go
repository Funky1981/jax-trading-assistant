package main

import (
	"testing"

	"jax-trading-assistant/libs/runtimepolicy"
)

func setValid02BProofEnvironment(t *testing.T) Config {
	t.Helper()
	for name, value := range map[string]string{
		coreReadiness02BProofModeEnv:            "true",
		"DATABASE_URL":                          "postgresql://jax:jax@127.0.0.1/jax_paper02r_test_core02b",
		"JAX_RUNTIME_MODE":                      "PAPER",
		"IB_BRIDGE_URL":                         "",
		"ALPACA_API_KEY":                        "proof-key",
		"ALPACA_API_SECRET":                     "proof-secret",
		"POLYGON_API_KEY":                       "",
		"POLYGON_ENABLED":                       "false",
		"JAX_MARKET_DATA_ALLOWED_SOURCES":       "alpaca",
		"JAX_MARKET_DATA_TIMEFRAME":             "1h",
		"JAX_MARKET_DATA_QUOTE_MAX_AGE":         "60s",
		"JAX_MARKET_DATA_LATEST_CANDLE_MAX_AGE": "90m",
		"JAX_MARKET_DATA_CANDLE_LOOKBACK":       "7d",
		"WORLD_MONITOR_PULL_ENABLED":            "true",
		"WORLD_MONITOR_EVENTS_URL":              "http://127.0.0.1:8082/api/v1/jax/events",
		"BROKER_EXECUTION_ALLOWED":              "false",
		"ALLOW_LIVE_TRADING":                    "false",
		"EXECUTION_ENABLED":                     "false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED":  "false",
		"EXECUTION_AUTHORITY":                   "NONE",
		"MAX_LEVERAGE":                          "1",
		"PAPER_ACCOUNT_ID":                      "b33647ee-2e29-4aa0-8a79-f642a9c8c220",
	} {
		t.Setenv(name, value)
	}
	return Config{
		DatabaseURL: "postgresql://jax:jax@127.0.0.1/jax_paper02r_test_core02b",
		RuntimeMode: runtimepolicy.ModePaper,
		WorkerGates: workerStartupGates{MarketIngesterEnabled: true},
	}
}

func TestValidateCoreReadiness02BProofModeAcceptsOnlyIsolatedAlpacaPaperRuntime(t *testing.T) {
	cfg := setValid02BProofEnvironment(t)
	if err := validateCoreReadiness02BProofMode(cfg); err != nil {
		t.Fatalf("valid isolated proof configuration rejected: %v", err)
	}
}

func TestValidateCoreReadiness02BProofModeFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, *Config)
	}{
		{name: "normal database", mutate: func(_ *testing.T, cfg *Config) { cfg.DatabaseURL = "postgresql://jax:jax@127.0.0.1/jax" }},
		{name: "non-paper mode", mutate: func(_ *testing.T, cfg *Config) { cfg.RuntimeMode = runtimepolicy.ModeResearch }},
		{name: "bridge configured", mutate: func(_ *testing.T, cfg *Config) { cfg.IBBridgeURL = "http://ib-bridge:8092" }},
		{name: "execution enabled", mutate: func(t *testing.T, _ *Config) { t.Setenv("EXECUTION_ENABLED", "true") }},
		{name: "broker authority enabled", mutate: func(t *testing.T, _ *Config) { t.Setenv("BROKER_EXECUTION_ALLOWED", "true") }},
		{name: "strategy generation worker enabled", mutate: func(_ *testing.T, cfg *Config) { cfg.WorkerGates.TradeWatcherEnabled = true }},
		{name: "provider allowlist broadened", mutate: func(t *testing.T, _ *Config) { t.Setenv("JAX_MARKET_DATA_ALLOWED_SOURCES", "alpaca,polygon") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := setValid02BProofEnvironment(t)
			tt.mutate(t, &cfg)
			if err := validateCoreReadiness02BProofMode(cfg); err == nil {
				t.Fatal("unsafe proof configuration unexpectedly accepted")
			}
		})
	}
}

func TestRequireApprovedStrategiesForStartupAllowsOnlyEmptyPaperProofRegistry(t *testing.T) {
	if err := requireApprovedStrategiesForStartup(runtimepolicy.ModePaper, 0, true); err != nil {
		t.Fatalf("empty registry was rejected in guarded proof mode: %v", err)
	}
	for _, tt := range []struct {
		name      string
		mode      runtimepolicy.Mode
		loaded    int
		proofMode bool
	}{
		{name: "normal PAPER startup", mode: runtimepolicy.ModePaper, loaded: 0, proofMode: false},
		{name: "proof mode loaded a strategy", mode: runtimepolicy.ModePaper, loaded: 1, proofMode: true},
		{name: "proof mode outside PAPER", mode: runtimepolicy.ModeResearch, loaded: 0, proofMode: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := requireApprovedStrategiesForStartup(tt.mode, tt.loaded, tt.proofMode); err == nil {
				t.Fatal("unsafe/normal empty strategy registry unexpectedly accepted")
			}
		})
	}
}
