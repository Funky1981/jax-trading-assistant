package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/marketdata"
	"jax-trading-assistant/services/jax-market/internal/config"
	"jax-trading-assistant/services/jax-market/internal/ingester"
)

// Metrics holds ingestion statistics
type Metrics struct {
	TotalIngestions    int64     `json:"total_ingestions"`
	SuccessfulIngests  int64     `json:"successful_ingests"`
	FailedIngests      int64     `json:"failed_ingests"`
	LastIngestTime     time.Time `json:"last_ingest_time"`
	LastIngestDuration string    `json:"last_ingest_duration"`
	SymbolCount        int       `json:"symbol_count"`
	Uptime             string    `json:"uptime"`
}

var (
	metrics   Metrics
	startTime = time.Now()
)

func main() {
	var configPath string
	var httpPort string
	flag.StringVar(&configPath, "config", "config/jax-market.json", "Path to configuration file")
	flag.StringVar(&httpPort, "port", "8095", "HTTP server port")
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
	metrics.SymbolCount = len(cfg.Symbols)

	// Initialize ingester with metrics callback
	ing := ingester.New(mdClient, db.DB, cfg)
	ing.SetMetricsCallback(func(success, failed int, duration time.Duration) {
		atomic.AddInt64(&metrics.TotalIngestions, 1)
		atomic.AddInt64(&metrics.SuccessfulIngests, int64(success))
		atomic.AddInt64(&metrics.FailedIngests, int64(failed))
		metrics.LastIngestTime = time.Now()
		metrics.LastIngestDuration = duration.String()
	})

	// Start HTTP server for health checks and metrics
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/metrics", handleMetrics)

	server := &http.Server{
		Addr:    ":" + httpPort,
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Printf("HTTP server listening on :%s", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start ingestion scheduler
	go ing.Start(ctx)

	log.Printf("jax-market started (interval: %ds)", cfg.IngestInterval)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Printf("shutting down...")
	cancel()

	// Shutdown HTTP server gracefully
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	time.Sleep(2 * time.Second)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "jax-market",
		"uptime":  time.Since(startTime).String(),
	})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Update uptime
	currentMetrics := Metrics{
		TotalIngestions:    atomic.LoadInt64(&metrics.TotalIngestions),
		SuccessfulIngests:  atomic.LoadInt64(&metrics.SuccessfulIngests),
		FailedIngests:      atomic.LoadInt64(&metrics.FailedIngests),
		LastIngestTime:     metrics.LastIngestTime,
		LastIngestDuration: metrics.LastIngestDuration,
		SymbolCount:        metrics.SymbolCount,
		Uptime:             time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(currentMetrics)
}
