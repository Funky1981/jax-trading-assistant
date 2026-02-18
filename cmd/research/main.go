// cmd/research is the Research Runtime for jax-trading-assistant.
// It replaces the jax-orchestrator microservice, hosting the orchestration
// pipeline (Agent0, memory, Dexter) in-process.
//
// Exposes the same HTTP API surface as jax-orchestrator so that jax-api
// requires no changes — only the compose service name and URL env var change.
//
// ADR-0012 Phase 5: one of the two authoritative runtimes.
// import rule: cmd/research MAY import libs/agent0, libs/dexter, libs/utcp.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jax-trading-assistant/internal/modules/orchestration"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
	startTime = time.Now()
)

// Config holds all configuration for the research runtime.
type Config struct {
	DatabaseURL      string
	Port             string
	MemoryServiceURL string
	Agent0ServiceURL string
	DexterServiceURL string
	HindsightURL     string // used by the in-process memory proxy
}

func main() {
	cfg := loadConfig()

	log.Printf("starting jax-research v%s (built: %s)", version, buildTime)
	log.Printf("port: %s", cfg.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Postgres (database/sql so we can run metrics queries)
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connected")

	// Wire up orchestration clients (all optional with graceful degradation)
	memoryClient, err := orchestration.NewMemoryClient(cfg.MemoryServiceURL)
	if err != nil {
		log.Fatalf("failed to create memory client: %v", err)
	}
	log.Printf("memory client → %s", cfg.MemoryServiceURL)

	agentClient, err := orchestration.NewAgent0Client(cfg.Agent0ServiceURL)
	if err != nil {
		log.Fatalf("failed to create Agent0 client: %v", err)
	}
	log.Printf("agent0 client → %s", cfg.Agent0ServiceURL)

	var dexterAdapter *orchestration.DexterClientAdapter
	if cfg.DexterServiceURL != "" {
		dexterAdapter, err = orchestration.NewDexterClient(cfg.DexterServiceURL)
		if err != nil {
			log.Printf("warning: Dexter unavailable (%v) — continuing without research tools", err)
		} else {
			log.Printf("dexter client → %s", cfg.DexterServiceURL)
		}
	}

	toolRunner := orchestration.NewToolRunner(dexterAdapter)
	registry := strategies.NewRegistry()

	orchSvc := orchestration.NewService(memoryClient, agentClient, toolRunner, registry)
	if dexterAdapter != nil {
		orchSvc = orchSvc.WithDexter(dexterAdapter)
	}

	// Build HTTP server
	mux := http.NewServeMux()
	registerRoutes(mux, orchSvc, db)

	// ADR-0012 Phase 6: memory proxy (replaces jax-memory service).
	// agent0-service can now point MEMORY_SERVICE_URL at jax-research:8091.
	memStore := buildMemoryStore()
	registerMemoryRoutes(mux, memStore)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("jax-research listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down jax-research...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("jax-research stopped")
}

// registerRoutes wires up all HTTP routes.
// Routes match the jax-orchestrator API surface exactly for backwards compatibility.
func registerRoutes(mux *http.ServeMux, svc *orchestration.Service, db *sql.DB) {
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/orchestrate", handleOrchestrate(svc, db))
	mux.HandleFunc("/metrics/prometheus", handlePrometheus(db))
}

// handleHealth returns a simple liveness response.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "jax-research",
		"status":  "healthy",
		"uptime":  time.Since(startTime).Round(time.Second).String(),
		"version": version,
	})
}

// OrchestrateRequest is the inbound payload — matches jax-orchestrator's schema.
type OrchestrateRequest struct {
	SignalID    string `json:"signal_id"`
	Symbol      string `json:"symbol"`
	TriggerType string `json:"trigger_type"`
	Context     string `json:"context"`
}

// handleOrchestrate accepts a trigger and runs the orchestration pipeline.
func handleOrchestrate(svc *orchestration.Service, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req OrchestrateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if req.Symbol == "" {
			http.Error(w, "symbol is required", http.StatusBadRequest)
			return
		}
		if req.TriggerType == "" {
			req.TriggerType = "manual"
		}

		// Parse optional signal ID
		var signalID uuid.UUID
		if req.SignalID != "" {
			var parseErr error
			signalID, parseErr = uuid.Parse(req.SignalID)
			if parseErr != nil {
				http.Error(w, "invalid signal_id", http.StatusBadRequest)
				return
			}
		}

		// Create run record in DB
		runID := uuid.New()
		if err := createOrchestrationRun(r.Context(), db, runID, req.Symbol, req.TriggerType, signalID); err != nil {
			log.Printf("failed to create orchestration run: %v", err)
			http.Error(w, "failed to create orchestration run", http.StatusInternalServerError)
			return
		}

		// Enrich context from DB if signal ID available
		enrichedCtx := req.Context
		if signalID != uuid.Nil {
			if extra, err := fetchSignalContext(r.Context(), db, signalID); err == nil {
				enrichedCtx = enrichedCtx + "\n\n" + extra
			}
		}

		// Run asynchronously — return immediately with run ID
		go runOrchestration(svc, db, runID, req.Symbol, enrichedCtx)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"run_id": runID.String(),
			"status": "running",
		})
	}
}

// runOrchestration executes the pipeline and writes the result back to Postgres.
func runOrchestration(svc *orchestration.Service, db *sql.DB, runID uuid.UUID, symbol, contextStr string) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("orchestration run %s panicked: %v", runID, r)
			updateRunError(ctx, db, runID, fmt.Sprintf("panic: %v", r))
		}
	}()

	result, err := svc.Orchestrate(ctx, orchestration.OrchestrationRequest{
		Symbol:      symbol,
		Bank:        "trade_decisions",
		UserContext: contextStr,
	})
	if err != nil {
		log.Printf("orchestration run %s failed: %v", runID, err)
		updateRunError(ctx, db, runID, err.Error())
		return
	}

	if err := updateRunComplete(ctx, db, runID, result); err != nil {
		log.Printf("failed to persist run %s: %v", runID, err)
	}
	log.Printf("orchestration run %s complete: action=%s confidence=%.2f",
		runID, result.Plan.Action, result.Plan.Confidence)
}

// handlePrometheus emits Prometheus text-format metrics for this service.
func handlePrometheus(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		var total, completed, failed int
		query := `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'completed') AS completed,
				COUNT(*) FILTER (WHERE status = 'failed') AS failed
			FROM orchestration_runs`
		if err := db.QueryRowContext(r.Context(), query).Scan(&total, &completed, &failed); err != nil {
			fmt.Fprintln(w, "# HELP jax_research_metrics_error Metrics query error")
			fmt.Fprintln(w, "# TYPE jax_research_metrics_error gauge")
			fmt.Fprintln(w, "jax_research_metrics_error 1")
			return
		}

		upSecs := time.Since(startTime).Seconds()
		fmt.Fprintf(w, "# HELP jax_research_uptime_seconds Uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE jax_research_uptime_seconds gauge\n")
		fmt.Fprintf(w, "jax_research_uptime_seconds %.0f\n", upSecs)

		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_total Total orchestration runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_total %d\n", total)

		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_completed_total Completed orchestration runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_completed_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_completed_total %d\n", completed)

		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_failed_total Failed orchestration runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_failed_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_failed_total %d\n", failed)
	}
}

// ── DB helpers ────────────────────────────────────────────────────────────────

func createOrchestrationRun(ctx context.Context, db *sql.DB, runID uuid.UUID, symbol, triggerType string, signalID uuid.UUID) error {
	var triggerPtr *uuid.UUID
	if signalID != uuid.Nil {
		triggerPtr = &signalID
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO orchestration_runs (id, symbol, trigger_type, trigger_id, status, started_at)
		VALUES ($1, $2, $3, $4, 'running', NOW())`,
		runID, symbol, triggerType, triggerPtr,
	)
	return err
}

func fetchSignalContext(ctx context.Context, db *sql.DB, signalID uuid.UUID) (string, error) {
	var sym, stratID, sigType, reasoning string
	var conf, entry, stop, take float64
	err := db.QueryRowContext(ctx, `
		SELECT symbol, strategy_id, signal_type, confidence, entry_price, stop_loss, take_profit, reasoning
		FROM strategy_signals WHERE id = $1`, signalID,
	).Scan(&sym, &stratID, &sigType, &conf, &entry, &stop, &take, &reasoning)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"Symbol: %s\nStrategy: %s\nType: %s\nConfidence: %.2f%%\nEntry: $%.2f\nStop: $%.2f\nTake: $%.2f\nReasoning: %s",
		sym, stratID, sigType, conf*100, entry, stop, take, reasoning,
	), nil
}

func updateRunComplete(ctx context.Context, db *sql.DB, runID uuid.UUID, result orchestration.OrchestrationResult) error {
	conf := result.Plan.Confidence
	if conf > 1 {
		conf /= 100
	}
	rawJSON, _ := json.Marshal(result)
	_, err := db.ExecContext(ctx, `
		UPDATE orchestration_runs
		SET agent_suggestion = $1,
		    confidence = $2,
		    reasoning = $3,
		    agent_response = $4,
		    status = 'completed',
		    completed_at = NOW()
		WHERE id = $5`,
		result.Plan.Action, conf, result.Plan.ReasoningNotes, rawJSON, runID,
	)
	return err
}

func updateRunError(ctx context.Context, db *sql.DB, runID uuid.UUID, msg string) {
	_, _ = db.ExecContext(ctx, `
		UPDATE orchestration_runs
		SET error = $1, status = 'failed', completed_at = NOW()
		WHERE id = $2`, msg, runID,
	)
}

// ── Config ────────────────────────────────────────────────────────────────────

func loadConfig() Config {
	return Config{
		DatabaseURL:      envOrDefault("DATABASE_URL", "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"),
		Port:             envOrDefault("PORT", "8091"),
		MemoryServiceURL: envOrDefault("MEMORY_SERVICE_URL", "http://jax-memory:8090"),
		Agent0ServiceURL: envOrDefault("AGENT0_SERVICE_URL", "http://agent0-service:8093"),
		DexterServiceURL: envOrDefault("DEXTER_SERVICE_URL", ""),
		HindsightURL:     envOrDefault("HINDSIGHT_URL", ""),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
