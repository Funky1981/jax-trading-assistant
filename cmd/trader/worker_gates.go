package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type workerStartupGates struct {
	OpportunityScannerEnabled           bool
	TradeWatcherEnabled                 bool
	MarketIngesterEnabled               bool
	MobileNotificationDispatcherEnabled bool
}

func loadWorkerStartupGates() (workerStartupGates, error) {
	opportunityScanner, err := parseWorkerEnabledEnv("OPPORTUNITY_SCANNER_WORKER_ENABLED", true)
	if err != nil {
		return workerStartupGates{}, err
	}
	tradeWatcher, err := parseWorkerEnabledEnv("TRADE_WATCHER_WORKER_ENABLED", true)
	if err != nil {
		return workerStartupGates{}, err
	}
	marketIngester, err := parseWorkerEnabledEnv("MARKET_INGESTER_WORKER_ENABLED", true)
	if err != nil {
		return workerStartupGates{}, err
	}
	mobileNotifications, err := parseWorkerEnabledEnv("MOBILE_NOTIFICATION_WORKER_ENABLED", true)
	if err != nil {
		return workerStartupGates{}, err
	}

	return workerStartupGates{
		OpportunityScannerEnabled:           opportunityScanner,
		TradeWatcherEnabled:                 tradeWatcher,
		MarketIngesterEnabled:               marketIngester,
		MobileNotificationDispatcherEnabled: mobileNotifications,
	}, nil
}

func parseWorkerEnabledEnv(name string, defaultValue bool) (bool, error) {
	raw, present := os.LookupEnv(name)
	if !present || strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
}

func (g workerStartupGates) readiness() map[string]any {
	return map[string]any{
		"opportunityScanner":           g.OpportunityScannerEnabled,
		"tradeWatcher":                 g.TradeWatcherEnabled,
		"marketIngester":               g.MarketIngesterEnabled,
		"mobileNotificationDispatcher": g.MobileNotificationDispatcherEnabled,
	}
}

func (g workerStartupGates) logStartupState() {
	log.Printf("worker gates: opportunity_scanner=%t trade_watcher=%t market_ingester=%t mobile_notification_dispatcher=%t", g.OpportunityScannerEnabled, g.TradeWatcherEnabled, g.MarketIngesterEnabled, g.MobileNotificationDispatcherEnabled)
}

func launchWorkerIfEnabled(enabled bool, name string, launch func()) {
	if !enabled {
		log.Printf("%s disabled by startup gate", name)
		return
	}
	launch()
}

func runOpportunityScannerOnce(ctx context.Context, pool *pgxpool.Pool, enabled bool) (opportunityScannerResult, error) {
	if !enabled {
		return opportunityScannerResult{Disabled: true}, nil
	}
	return newOpportunityScanner(pool).ScanOnce(ctx)
}
