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
	"syscall"
	"time"

	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/services/jax-signal-generator/internal/config"
	"jax-trading-assistant/services/jax-signal-generator/internal/generator"
	"jax-trading-assistant/services/jax-signal-generator/internal/orchestrator"
)

type Metrics struct {
	TotalRuns        int64     `json:"total_runs"`
	SignalsGenerated int64     `json:"signals_generated"`
	FailedRuns       int64     `json:"failed_runs"`
	LastRunTime      time.Time `json:"last_run_time"`
	Uptime           string    `json:"uptime"`
}

var (
	metrics   Metrics
	startTime = time.Now()
)

func main() {
	configPath := flag.String("config", "/app/config/jax-signal-generator.json", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to database
	dbConfig := database.DefaultConfig()
	dbConfig.DSN = cfg.DatabaseDSN

	db, err := database.Connect(context.Background(), dbConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("database connected")

	// Create strategy registry
	registry := strategies.NewRegistry()

	// Register all strategies
	rsi := strategies.NewRSIMomentumStrategy()
	registry.Register(rsi, rsi.GetMetadata())

	macd := strategies.NewMACDCrossoverStrategy()
	registry.Register(macd, macd.GetMetadata())

	ma := strategies.NewMACrossoverStrategy()
	registry.Register(ma, ma.GetMetadata())

	log.Printf("registered %d strategies", len(registry.List()))

	// Create signal generator (use the underlying sql.DB)
	gen := generator.New(db.DB, registry, cfg.Symbols, metricsCallback).
		WithConfidenceThreshold(cfg.ConfidenceThreshold)

	// Add orchestrator integration if enabled
	if cfg.OrchestrationEnabled && cfg.OrchestratorURL != "" {
		orchClient := orchestrator.NewClient(cfg.OrchestratorURL)
		gen = gen.WithOrchestrator(orchClient)
		log.Printf("orchestration enabled (URL: %s, threshold: %.2f)", cfg.OrchestratorURL, cfg.ConfidenceThreshold)
	} else {
		log.Println("orchestration disabled")
	}

	// Start HTTP server
	go startHTTPServer()

	// Run generator on schedule
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(time.Duration(cfg.GenerateInterval) * time.Second)
	defer ticker.Stop()

	// Run immediately on startup
	log.Println("running initial signal generation...")
	if err := gen.Generate(ctx); err != nil {
		log.Printf("failed initial generation: %v", err)
		metrics.FailedRuns++
	}

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("signal generator started (interval: %ds)", cfg.GenerateInterval)

	for {
		select {
		case <-ticker.C:
			log.Println("starting scheduled signal generation...")
			if err := gen.Generate(ctx); err != nil {
				log.Printf("failed signal generation: %v", err)
				metrics.FailedRuns++
			}
		case <-sigChan:
			log.Println("shutdown signal received, exiting...")
			return
		}
	}
}

func metricsCallback(generated int, failed int, duration time.Duration) {
	metrics.TotalRuns++
	metrics.SignalsGenerated += int64(generated)
	if failed > 0 {
		metrics.FailedRuns++
	}
	metrics.LastRunTime = time.Now()
	log.Printf("generation complete: %d signals generated, %d  failed (duration: %v)", generated, failed, duration)
}

func startHTTPServer() {
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/metrics/prometheus", handlePrometheusMetrics)

	log.Println("HTTP server listening on :8096")
	if err := http.ListenAndServe(":8096", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"service": "jax-signal-generator",
		"status":  "healthy",
		"uptime":  time.Since(startTime).String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics.Uptime = time.Since(startTime).String()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	uptime := time.Since(startTime).Seconds()

	fmt.Fprintf(w, "# HELP jax_signal_generator_runs_total Total generator runs\n")
	fmt.Fprintf(w, "# TYPE jax_signal_generator_runs_total counter\n")
	fmt.Fprintf(w, "jax_signal_generator_runs_total %d\n", metrics.TotalRuns)

	fmt.Fprintf(w, "# HELP jax_signal_generator_signals_generated_total Total signals generated\n")
	fmt.Fprintf(w, "# TYPE jax_signal_generator_signals_generated_total counter\n")
	fmt.Fprintf(w, "jax_signal_generator_signals_generated_total %d\n", metrics.SignalsGenerated)

	fmt.Fprintf(w, "# HELP jax_signal_generator_failed_runs_total Failed generator runs\n")
	fmt.Fprintf(w, "# TYPE jax_signal_generator_failed_runs_total counter\n")
	fmt.Fprintf(w, "jax_signal_generator_failed_runs_total %d\n", metrics.FailedRuns)

	fmt.Fprintf(w, "# HELP jax_signal_generator_last_run_timestamp_seconds Last run timestamp\n")
	fmt.Fprintf(w, "# TYPE jax_signal_generator_last_run_timestamp_seconds gauge\n")
	if !metrics.LastRunTime.IsZero() {
		fmt.Fprintf(w, "jax_signal_generator_last_run_timestamp_seconds %d\n", metrics.LastRunTime.Unix())
	} else {
		fmt.Fprintf(w, "jax_signal_generator_last_run_timestamp_seconds 0\n")
	}

	fmt.Fprintf(w, "# HELP jax_signal_generator_uptime_seconds Service uptime\n")
	fmt.Fprintf(w, "# TYPE jax_signal_generator_uptime_seconds gauge\n")
	fmt.Fprintf(w, "jax_signal_generator_uptime_seconds %.0f\n", uptime)
}
