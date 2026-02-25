package utcp

import (
	"testing"

	"jax-trading-assistant/libs/runtimepolicy"
)

func TestValidateRuntimeProviderPolicy_StrictModeRejectsSynthetic(t *testing.T) {
	cfg := ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "market-data", Transport: "http", Endpoint: "http://localhost:8100/tools", DataSourceType: "real"},
			{ID: "backtest", Transport: "local", DataSourceType: "synthetic", SyntheticReason: "fixture"},
		},
	}
	if err := ValidateRuntimeProviderPolicy(runtimepolicy.ModePaper, cfg); err == nil {
		t.Fatalf("expected strict mode to reject synthetic provider")
	}
}

func TestValidateRuntimeProviderPolicy_DevAllowsSynthetic(t *testing.T) {
	cfg := ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "backtest", Transport: "local", DataSourceType: "synthetic", SyntheticReason: "fixture"},
		},
	}
	if err := ValidateRuntimeProviderPolicy(runtimepolicy.ModeDev, cfg); err != nil {
		t.Fatalf("dev mode should allow synthetic provider: %v", err)
	}
}
