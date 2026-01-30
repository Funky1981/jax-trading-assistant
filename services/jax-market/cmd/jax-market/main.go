package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/services/jax-market/internal/config"
	"jax-trading-assistant/services/jax-market/internal/ingester"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config/jax-market.json", "Path to configuration file")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize database connection
	dbConfig := database.DefaultConfig()
	dbConfig.DSN = cfg.DatabaseDSN

	db, err := database.Connect(ctx, dbConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Printf("database connected")

	// Initialize market data client
	mdConfig := &marketdata.Config{
		Providers: []marketdata.ProviderConfig{},
		Cache: marketdata.CacheConfig{
			Enabled:  cfg.Cache.Enabled,
			RedisURL: cfg.Cache.RedisURL,
			TTL:      time.Duration(cfg.Cache.TTL) * time.Second,
		},
		Symbols: cfg.Symbols,
	}

	// Add configured providers
	if cfg.Polygon.Enabled {
		mdConfig.Providers = append(mdConfig.Providers, marketdata.ProviderConfig{
			Name:     marketdata.ProviderPolygon,
			APIKey:   cfg.Polygon.APIKey,
			Tier:     cfg.Polygon.Tier,
			Priority: 1,
			Enabled:  true,
		})
	}

	if cfg.Alpaca.Enabled {
		mdConfig.Providers = append(mdConfig.Providers, marketdata.ProviderConfig{
			Name:      marketdata.ProviderAlpaca,
			APIKey:    cfg.Alpaca.APIKey,
			APISecret: cfg.Alpaca.APISecret,
			Tier:      cfg.Alpaca.Tier,
			Priority:  2,
			Enabled:   true,
		})
	}

	if cfg.IB.Enabled {
		mdConfig.Providers = append(mdConfig.Providers, marketdata.ProviderConfig{
			Name:       marketdata.ProviderIB,
			IBHost:     cfg.IB.Host,
			IBPort:     cfg.IB.Port,
			IBClientID: cfg.IB.ClientID,
			Priority:   1, // Highest priority if enabled (real-time data)
			Enabled:    true,
		})
	}

	mdClient, err := marketdata.NewClient(mdConfig)
	if err != nil {
		log.Fatalf("failed to create market data client: %v", err)
	}
	defer mdClient.Close()

	log.Printf("market data client initialized with %d symbol(s)", len(cfg.Symbols))

	// Initialize ingester
	ing := ingester.New(mdClient, db.DB, cfg)

	// Start ingestion scheduler
	go ing.Start(ctx)

	log.Printf("jax-market started (interval: %ds)", cfg.IngestInterval)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("shutting down...")
	cancel()
	time.Sleep(2 * time.Second)
}
