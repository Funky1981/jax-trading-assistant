package utcp

import (
	"strings"
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

func TestValidateRuntimeProviderPolicy_ResearchRejectsUnknownTruthPathProvider(t *testing.T) {
	cfg := ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "market-data", Transport: "http", Endpoint: "http://localhost:8100/tools", DataSourceType: "unknown"},
		},
	}
	err := ValidateRuntimeProviderPolicy(runtimepolicy.ModeResearch, cfg)
	if err == nil {
		t.Fatalf("expected research mode to reject unknown truth-path provider")
	}
	if !strings.Contains(err.Error(), "truth-path providers") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRuntimeProviderPolicy_ResearchAllowsRealTruthPathProvider(t *testing.T) {
	cfg := ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "market-data", Transport: "http", Endpoint: "http://localhost:8100/tools", DataSourceType: "real"},
			{ID: "memory", Transport: "http", Endpoint: "http://localhost:8091/tools", DataSourceType: "real"},
		},
	}
	if err := ValidateRuntimeProviderPolicy(runtimepolicy.ModeResearch, cfg); err != nil {
		t.Fatalf("research mode should allow real truth-path providers: %v", err)
	}
}

func TestValidateRuntimeProviderPolicy_PaperRejectsUnknownBrokerProvider(t *testing.T) {
	cfg := ProvidersConfig{
		Providers: []ProviderConfig{
			{ID: "broker", Transport: "http", Endpoint: "http://localhost:8092/tools", DataSourceType: "unknown"},
		},
	}
	err := ValidateRuntimeProviderPolicy(runtimepolicy.ModePaper, cfg)
	if err == nil {
		t.Fatalf("expected paper mode to reject unknown broker provider")
	}
	if !strings.Contains(err.Error(), "truth-path providers") {
		t.Fatalf("unexpected error: %v", err)
	}
}
