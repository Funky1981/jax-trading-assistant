package main

import (
	"context"
	"testing"
)

func TestLoadWorkerStartupGatesDefaultToCurrentBehaviour(t *testing.T) {
	for _, name := range []string{
		"OPPORTUNITY_SCANNER_WORKER_ENABLED",
		"TRADE_WATCHER_WORKER_ENABLED",
		"MARKET_INGESTER_WORKER_ENABLED",
		"MOBILE_NOTIFICATION_WORKER_ENABLED",
	} {
		t.Setenv(name, "")
	}

	gates, err := loadWorkerStartupGates()
	if err != nil {
		t.Fatalf("load default worker gates: %v", err)
	}
	if !gates.OpportunityScannerEnabled || !gates.TradeWatcherEnabled || !gates.MarketIngesterEnabled || !gates.MobileNotificationDispatcherEnabled {
		t.Fatalf("default worker gates changed existing behaviour: %+v", gates)
	}
}

func TestLoadWorkerStartupGatesExplicitFalseDisablesWorkers(t *testing.T) {
	t.Setenv("OPPORTUNITY_SCANNER_WORKER_ENABLED", "false")
	t.Setenv("TRADE_WATCHER_WORKER_ENABLED", "false")
	t.Setenv("MARKET_INGESTER_WORKER_ENABLED", "false")
	t.Setenv("MOBILE_NOTIFICATION_WORKER_ENABLED", "false")

	gates, err := loadWorkerStartupGates()
	if err != nil {
		t.Fatalf("load explicit worker gates: %v", err)
	}
	if gates.OpportunityScannerEnabled || gates.TradeWatcherEnabled || gates.MarketIngesterEnabled || gates.MobileNotificationDispatcherEnabled {
		t.Fatalf("explicit false did not disable every worker: %+v", gates)
	}
}

func TestLoadWorkerStartupGatesRejectInvalidBoolean(t *testing.T) {
	t.Setenv("MARKET_INGESTER_WORKER_ENABLED", "sometimes")

	if _, err := loadWorkerStartupGates(); err == nil {
		t.Fatal("expected invalid worker boolean to fail closed")
	}
}

func TestDisabledOpportunityScannerDoesNotRunScanOnce(t *testing.T) {
	result, err := runOpportunityScannerOnce(context.Background(), nil, false)
	if err != nil {
		t.Fatalf("disabled scanner returned error: %v", err)
	}
	if !result.Disabled {
		t.Fatalf("disabled scanner result = %+v, want Disabled=true", result)
	}
}

func TestWorkerStartupGateSuppressesRequestedWorkers(t *testing.T) {
	for _, name := range []string{
		"opportunity scanner",
		"trade watcher",
		"market ingester",
		"mobile notification dispatcher",
	} {
		launched := false
		launchWorkerIfEnabled(false, name, func() { launched = true })
		if launched {
			t.Fatalf("%s launch callback ran while disabled", name)
		}
	}
}

func TestWorkerStartupGateLaunchesEnabledWorker(t *testing.T) {
	launched := false
	launchWorkerIfEnabled(true, "test worker", func() { launched = true })
	if !launched {
		t.Fatal("enabled worker launch callback did not run")
	}
}

func TestWorkerReadinessReportsConfiguredGates(t *testing.T) {
	gates := workerStartupGates{
		OpportunityScannerEnabled:           false,
		TradeWatcherEnabled:                 false,
		MarketIngesterEnabled:               false,
		MobileNotificationDispatcherEnabled: false,
	}
	readiness := gates.readiness()
	for name, value := range readiness {
		if enabled, ok := value.(bool); !ok || enabled {
			t.Fatalf("worker %s readiness = %#v, want false", name, value)
		}
	}
}
