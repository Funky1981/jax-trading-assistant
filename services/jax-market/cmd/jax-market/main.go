package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	StaleQuotes        int       `json:"stale_quotes"`
	LastStaleCheck     time.Time `json:"last_stale_check"`
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
	ing.SetMetricsCallback(func(success, failed int, duration time.Duration, staleCount int) {
		atomic.AddInt64(&metrics.TotalIngestions, 1)
		atomic.AddInt64(&metrics.SuccessfulIngests, int64(success))
		atomic.AddInt64(&metrics.FailedIngests, int64(failed))
		metrics.LastIngestTime = time.Now()
		metrics.LastIngestDuration = duration.String()
		metrics.StaleQuotes = staleCount
		metrics.LastStaleCheck = time.Now()
	})

	// Start HTTP server for health checks and metrics
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/metrics/prometheus", handlePrometheusMetrics)

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
		StaleQuotes:        metrics.StaleQuotes,
		LastStaleCheck:     metrics.LastStaleCheck,
		Uptime:             time.Since(startTime).String(),
	}

	json.NewEncoder(w).Encode(currentMetrics)
}

func handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	uptime := time.Since(startTime).Seconds()

	fmt.Fprintf(w, "# HELP jax_market_ingestions_total Total ingestion runs\n")
	fmt.Fprintf(w, "# TYPE jax_market_ingestions_total counter\n")
	fmt.Fprintf(w, "jax_market_ingestions_total %d\n", atomic.LoadInt64(&metrics.TotalIngestions))

	fmt.Fprintf(w, "# HELP jax_market_ingestions_success_total Successful ingestions\n")
	fmt.Fprintf(w, "# TYPE jax_market_ingestions_success_total counter\n")
	fmt.Fprintf(w, "jax_market_ingestions_success_total %d\n", atomic.LoadInt64(&metrics.SuccessfulIngests))

	fmt.Fprintf(w, "# HELP jax_market_ingestions_failed_total Failed ingestions\n")
	fmt.Fprintf(w, "# TYPE jax_market_ingestions_failed_total counter\n")
	fmt.Fprintf(w, "jax_market_ingestions_failed_total %d\n", atomic.LoadInt64(&metrics.FailedIngests))

	fmt.Fprintf(w, "# HELP jax_market_last_ingest_duration_seconds Last ingest duration\n")
	fmt.Fprintf(w, "# TYPE jax_market_last_ingest_duration_seconds gauge\n")
	if metrics.LastIngestDuration != "" {
		if d, err := time.ParseDuration(metrics.LastIngestDuration); err == nil {
			fmt.Fprintf(w, "jax_market_last_ingest_duration_seconds %.2f\n", d.Seconds())
		} else {
			fmt.Fprintf(w, "jax_market_last_ingest_duration_seconds 0\n")
		}
	} else {
		fmt.Fprintf(w, "jax_market_last_ingest_duration_seconds 0\n")
	}

	fmt.Fprintf(w, "# HELP jax_market_stale_quotes Stale quote count\n")
	fmt.Fprintf(w, "# TYPE jax_market_stale_quotes gauge\n")
	fmt.Fprintf(w, "jax_market_stale_quotes %d\n", metrics.StaleQuotes)

	fmt.Fprintf(w, "# HELP jax_market_uptime_seconds Service uptime\n")
	fmt.Fprintf(w, "# TYPE jax_market_uptime_seconds gauge\n")
	fmt.Fprintf(w, "jax_market_uptime_seconds %.0f\n", uptime)
}
