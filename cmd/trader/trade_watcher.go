package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	signalgenerator "jax-trading-assistant/internal/trader/signalgenerator"
)

// tradeWatcherConfig holds tunables for the always-on watcher.
type tradeWatcherConfig struct {
	// ScanInterval is how often the watcher polls enabled instances.
	ScanInterval time.Duration
	// ExpiryInterval is how often stale candidates are expired.
	ExpiryInterval time.Duration
}

func defaultWatcherConfig() tradeWatcherConfig {
	return tradeWatcherConfig{
		ScanInterval:   5 * time.Minute,
		ExpiryInterval: 1 * time.Minute,
	}
}

// startTradeWatcher launches the background goroutine that continuously
// evaluates enabled strategy instances and generates candidate trades.
// It runs until ctx is cancelled.
func startTradeWatcher(ctx context.Context, pool *pgxpool.Pool, sigGen *signalgenerator.InProcessSignalGenerator) {
	cfg := defaultWatcherConfig()
	store := candidatesmod.NewStore(pool)
	svc := candidatesmod.NewService(store)
	scheduler := newInstanceScheduler(cfg.ScanInterval)

	expireTicker := time.NewTicker(cfg.ExpiryInterval)
	scanTicker := time.NewTicker(cfg.ScanInterval)
	defer expireTicker.Stop()
	defer scanTicker.Stop()

	log.Printf("trade watcher started (scan every %s, expiry check every %s)",
		cfg.ScanInterval, cfg.ExpiryInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("trade watcher stopped")
			return

		case <-expireTicker.C:
			if err := svc.ExpireStale(ctx); err != nil {
				log.Printf("trade watcher: expire stale error: %v", err)
			}

		case <-scanTicker.C:
			// Skip if global kill switch is active.
			if checkKillSwitch(ctx, pool) {
				log.Println("trade watcher: kill switch active, skipping scan")
				continue
			}
			instances, err := loadEnabledInstances(ctx, pool)
			if err != nil {
				log.Printf("trade watcher: loadEnabledInstances error: %v", err)
				continue
			}
			for _, inst := range instances {
				if !scheduler.due(inst.ID) {
					continue
				}
				scanInstance(ctx, svc, sigGen, inst)
				scheduler.mark(inst.ID)
				publishEvent("watcher.scan", map[string]any{
					"instanceId":   inst.ID,
					"instanceName": inst.Name,
					"scannedAt":    time.Now().UTC(),
				})
			}
		}
	}
}
