package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"jax-trading-assistant/internal/domain/artifacts"
	artifactsModule "jax-trading-assistant/internal/modules/artifacts"
	"jax-trading-assistant/internal/modules/execution"
	"jax-trading-assistant/internal/trader/signalgenerator"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
	startTime = time.Now()
)

type Config struct {
	DatabaseURL      string
	Port             string
	IBBridgeURL      string
	ExecutionEnabled bool
	// Execution risk parameters
	MaxRiskPerTrade     float64
	MinPositionSize     int
	MaxPositionSize     int
	MaxPositionValuePct float64
	MaxOpenPositions    int
	MaxDailyLoss        float64
	DefaultOrderType    string
}

func main() {
	// Parse flags
	configFlag := flag.String("config", "", "path to config file (optional, env vars take precedence)")
	flag.Parse()

	if *configFlag != "" {
		log.Printf("config flag provided: %s (note: environment variables take precedence)", *configFlag)
	}

	// Load configuration from environment
	cfg := loadConfig()

	log.Printf("starting jax-trader v%s (built: %s)", version, buildTime)
	log.Printf("database: %s", maskDSN(cfg.DatabaseURL))
	log.Printf("port: %s", cfg.Port)

	// Initialize database connection pool
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to create database pool: %v", err)
	}
	defer dbPool.Close()

	// Test database connectivity
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection established")

	// Initialize strategy registry
	registry := strategies.NewRegistry()

	// ADR-0012 Phase 4: Load strategies from APPROVED artifacts only
	// This implements the artifact-based promotion gate
	artifactLoader := artifactsModule.NewLoader(dbPool, registry)
	if err := artifactLoader.LoadApprovedStrategies(ctx); err != nil {
		log.Fatalf("failed to load approved strategy artifacts: %v", err)
	}

	log.Printf("registered %d strategies: %v", len(registry.List()), registry.List())

	// ADR-0012 Phase 4: Create artifact management API
	artifactStore := artifacts.NewStore(dbPool)
	artifactHandlers := NewArtifactHandlers(artifactStore)

	// Create in-process signal generator
	sigGen := signalgenerator.New(dbPool, registry)
	log.Println("in-process signal generator initialized")

	// Initialize execution service (optional - only if enabled and IB Bridge available)
	var execService *execution.Service
	if cfg.ExecutionEnabled && cfg.IBBridgeURL != "" {
		// Create IB Bridge client
		ibClient := execution.NewIBClient(cfg.IBBridgeURL)
		log.Printf("IB Bridge client connected to %s", cfg.IBBridgeURL)

		// Create risk parameters
		riskParams := execution.RiskParameters{
			MaxRiskPerTrade:     cfg.MaxRiskPerTrade,
			MinPositionSize:     cfg.MinPositionSize,
			MaxPositionSize:     cfg.MaxPositionSize,
			MaxPositionValuePct: cfg.MaxPositionValuePct,
			MaxOpenPositions:    cfg.MaxOpenPositions,
			MaxDailyLoss:        cfg.MaxDailyLoss,
		}

		// Create execution engine
		engine := execution.NewEngine(riskParams)

		// Create SQL DB wrapper for trade store (pgx v5 doesn't work with database/sql interface)
		sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			log.Printf("warning: failed to create SQL DB for execution: %v (execution disabled)", err)
		} else {
			tradeStore := execution.NewPostgresTradeStore(sqlDB)

			// Create execution service
			execService = execution.NewService(engine, ibClient, tradeStore, cfg.DefaultOrderType, riskParams)
			log.Println("execution service initialized")
			log.Printf("  IB Bridge: %s", cfg.IBBridgeURL)
			log.Printf("  max risk per trade: %.2f%%", cfg.MaxRiskPerTrade*100)
			log.Printf("  max position value: %.2f%%", cfg.MaxPositionValuePct*100)
			log.Printf("  max open positions: %d", cfg.MaxOpenPositions)
			log.Printf("  order type: %s", cfg.DefaultOrderType)
		}
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// ADR-0012 Phase 6: launch in-process replacements for removed microservices.
	// startMarketIngester replaces jax-market; startFrontendAPIServer replaces jax-api.
	go startMarketIngester(ctx, dbPool)
	go startFrontendAPIServer(ctx, dbPool, registry)

	// Health check endpoint
	mux.HandleFunc("/health", handleHealth(sigGen))

	// ADR-0012 Phase 4: Artifact promotion workflow API
	artifactHandlers.RegisterRoutes(mux)

	// Signal generation endpoints (compatible with jax-signal-generator API)
	mux.HandleFunc("/api/v1/signals/generate", handleGenerateSignals(sigGen))
	mux.HandleFunc("/api/v1/signals", handleGetSignals(sigGen))

	// NOTE: /orchestrate removed from trader — handled by cmd/research (jax-research).

	// Execution endpoint (compatible with jax-trade-executor API)
	if execService != nil {
		mux.HandleFunc("/api/v1/execute", handleExecute(execService))
	}

	// Metrics endpoint
	mux.HandleFunc("/metrics", handleMetrics())

	// Server configuration
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("HTTP server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("shutdown signal received, gracefully stopping...")
	ctxCancel() // signal background goroutines to stop

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("jax-trader stopped")
}

func loadConfig() Config {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		IBBridgeURL: os.Getenv("IB_BRIDGE_URL"),
	}

	// Parse execution settings
	if os.Getenv("EXECUTION_ENABLED") == "true" {
		cfg.ExecutionEnabled = true
	}

	// Parse execution risk parameters with defaults
	cfg.MaxRiskPerTrade = parseFloatEnv("MAX_RISK_PER_TRADE", 0.01) // 1% default
	cfg.MinPositionSize = parseIntEnv("MIN_POSITION_SIZE", 1)
	cfg.MaxPositionSize = parseIntEnv("MAX_POSITION_SIZE", 1000)
	cfg.MaxPositionValuePct = parseFloatEnv("MAX_POSITION_VALUE_PCT", 0.20) // 20% default
	cfg.MaxOpenPositions = parseIntEnv("MAX_OPEN_POSITIONS", 5)
	cfg.MaxDailyLoss = parseFloatEnv("MAX_DAILY_LOSS", 1000.0)
	cfg.DefaultOrderType = os.Getenv("DEFAULT_ORDER_TYPE")
	if cfg.DefaultOrderType == "" {
		cfg.DefaultOrderType = "LMT"
	}

	// Set defaults
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgresql://jax:jax@localhost:5432/jax"
		log.Println("DATABASE_URL not set, using default")
	}

	if cfg.Port == "" {
		cfg.Port = "8100"
		log.Println("PORT not set, using default 8100")
	}

	if cfg.IBBridgeURL == "" {
		cfg.IBBridgeURL = "http://localhost:8092"
		log.Println("IB_BRIDGE_URL not set, using default")
	}

	return cfg
}

// parseFloatEnv parses a float from environment variable with a default value
func parseFloatEnv(key string, defaultValue float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		log.Printf("warning: invalid %s value '%s', using default %.4f", key, val, defaultValue)
		return defaultValue
	}
	return parsed
}

// parseIntEnv parses an int from environment variable with a default value
func parseIntEnv(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("warning: invalid %s value '%s', using default %d", key, val, defaultValue)
		return defaultValue
	}
	return parsed
}

// maskDSN masks sensitive parts of the database URL for logging
func maskDSN(dsn string) string {
	// Simple masking - just show it's postgresql without credentials
	return "postgresql://***:***@<host>/<database>"
}

// HTTP Handlers

func handleHealth(sigGen *signalgenerator.InProcessSignalGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check signal generator health (includes DB and registry checks)
		if err := sigGen.Health(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"service": "jax-trader",
				"status":  "unhealthy",
				"error":   err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "jax-trader",
			"version": version,
			"status":  "healthy",
			"uptime":  time.Since(startTime).String(),
		})
	}
}

func handleGenerateSignals(sigGen *signalgenerator.InProcessSignalGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		// Parse request body
		var req struct {
			Symbols []string `json:"symbols"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		if len(req.Symbols) == 0 {
			http.Error(w, "symbols array is required and cannot be empty", http.StatusBadRequest)
			return
		}

		// Generate signals
		startTime := time.Now()
		signals, err := sigGen.GenerateSignals(ctx, req.Symbols)
		if err != nil {
			log.Printf("failed to generate signals: %v", err)
			http.Error(w, fmt.Sprintf("signal generation failed: %v", err), http.StatusInternalServerError)
			return
		}

		duration := time.Since(startTime)
		log.Printf("generated %d signals for %d symbols in %v", len(signals), len(req.Symbols), duration)

		// Return response (compatible with jax-signal-generator format)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"signals":  signals,
			"count":    len(signals),
			"duration": duration.String(),
		})
	}
}

func handleGetSignals(sigGen *signalgenerator.InProcessSignalGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		// Parse query parameters
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol query parameter is required", http.StatusBadRequest)
			return
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 50 // default
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "invalid limit parameter", http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}

		// Get signal history
		signals, err := sigGen.GetSignalHistory(ctx, symbol, limit)
		if err != nil {
			log.Printf("failed to get signal history for %s: %v", symbol, err)
			http.Error(w, fmt.Sprintf("failed to get signals: %v", err), http.StatusInternalServerError)
			return
		}

		// Return response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"symbol":  symbol,
			"signals": signals,
			"count":   len(signals),
		})
	}
}

func handleMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service":        "jax-trader",
			"version":        version,
			"uptime_seconds": time.Since(startTime).Seconds(),
			"go_routines":    "N/A", // Can add runtime.NumGoroutine() if needed
		})
	}
}

func handleExecute(execService *execution.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		// Parse request body
		var req struct {
			SignalID   string `json:"signal_id"`
			ApprovedBy string `json:"approved_by"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// Validate required fields
		if req.SignalID == "" {
			http.Error(w, "signal_id is required", http.StatusBadRequest)
			return
		}
		if req.ApprovedBy == "" {
			http.Error(w, "approved_by is required", http.StatusBadRequest)
			return
		}

		// Parse signal ID
		signalID, err := uuid.Parse(req.SignalID)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid signal_id: %v", err), http.StatusBadRequest)
			return
		}

		// Execute trade
		startTime := time.Now()
		trade, err := execService.ExecuteTrade(ctx, signalID, req.ApprovedBy)
		if err != nil {
			log.Printf("trade execution failed for signal %s: %v", req.SignalID, err)
			http.Error(w, fmt.Sprintf("execution failed: %v", err), http.StatusInternalServerError)
			return
		}

		duration := time.Since(startTime)
		log.Printf("trade executed for signal %s: order_id=%s symbol=%s qty=%d status=%s duration=%v",
			req.SignalID, trade.OrderID, trade.Symbol, trade.Quantity, trade.Status, duration)

		// Return response (compatible with jax-trade-executor format)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  true,
			"trade_id": trade.TradeID.String(),
			"order_id": trade.OrderID,
			"trade":    trade,
			"duration": duration.String(),
		})
	}
}
