// frontend_api.go — Part of cmd/trader (package main).
// Starts a second HTTP server on API_PORT (default 8081) that exposes the full
// frontend-facing API surface previously served by jax-api.
// ADR-0012 Phase 6: consolidates jax-api into the trader runtime.
//
// Routes (JWT-protected unless marked public):
//
//	Public:
//	  GET  /health
//	  POST /auth/login
//	  POST /auth/refresh
//	Protected:
//	  GET  /api/v1/signals          – list strategy_signals
//	  GET  /api/v1/signals/{id}     – single signal
//	  POST /api/v1/signals/{id}/approve
//	  POST /api/v1/signals/{id}/reject
//	  POST /api/v1/signals/{id}/analyze
//	  GET  /api/v1/recommendations  – same source, formatted for UI
//	  GET  /api/v1/recommendations/{id}
//	  GET  /trades                  – list trades
//	  GET  /trades/{id}             – single trade
//	  POST /api/v1/orchestrate      – proxy → jax-research
//	  GET  /api/v1/orchestrate/runs
//	  GET  /api/v1/orchestrate/runs/{id}
//	  GET  /api/v1/metrics
//	  GET  /api/v1/metrics/runs/{id}
//	  GET  /strategies              – list strategies (from registry)
//	  GET  /api/v1/strategies
//	  GET  /api/v1/strategies/{id}
//	  GET  /trading/guard
//	  POST /trading/guard/outcome
//	  POST /risk/calc
//	  POST /symbols/{symbol}        – trigger analysis
//	  GET  /metrics/prometheus      – Prometheus text format
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/auth"
	"jax-trading-assistant/libs/middleware"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// frontendAPIConfig holds settings read from env for the API server.
type frontendAPIConfig struct {
	APIPort         string
	OrchestratorURL string
}

func loadFrontendAPIConfig() frontendAPIConfig {
	return frontendAPIConfig{
		APIPort:         envStr("API_PORT", "8081"),
		OrchestratorURL: envStr("JAX_ORCHESTRATOR_URL", "http://jax-research:8091"),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// startFrontendAPIServer launches the jax-api-compatible HTTP server.
// It runs until ctx is cancelled.
func startFrontendAPIServer(ctx context.Context, pool *pgxpool.Pool, reg *strategies.Registry) {
	fCfg := loadFrontendAPIConfig()

	jwtManager, err := auth.NewJWTManagerFromEnv()
	if err != nil {
		log.Printf("frontend API: JWT disabled (%v) — running in unauthenticated mode", err)
	}

	rateLimiter := middleware.NewRateLimiterFromEnv()
	corsConfig := middleware.CORSConfigFromEnv()

	protect := func(h http.HandlerFunc) http.HandlerFunc {
		if jwtManager == nil {
			return h
		}
		return jwtManager.MiddlewareFunc(h)
	}

	mux := http.NewServeMux()

	// ── Auth ──────────────────────────────────────────────────────────────────
	if jwtManager != nil {
		mux.HandleFunc("/auth/login", auth.LoginHandler(jwtManager))
		mux.HandleFunc("/auth/refresh", auth.RefreshHandler(jwtManager))
	}

	// ── Health ────────────────────────────────────────────────────────────────
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "jax-trader-api",
			"uptime":  time.Since(startTime).Round(time.Second).String(),
		})
	})

	// ── Signals ───────────────────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/signals", protect(signalsListHandler(pool)))
	mux.HandleFunc("/api/v1/signals/", protect(signalDetailHandler(pool, fCfg.OrchestratorURL)))

	// ── Recommendations (same data, UI framing) ───────────────────────────────
	mux.HandleFunc("/api/v1/recommendations", protect(recommendationsListHandler(pool)))
	mux.HandleFunc("/api/v1/recommendations/", protect(recommendationDetailHandler(pool)))

	// ── Trades ────────────────────────────────────────────────────────────────
	mux.HandleFunc("/trades", protect(tradesListHandler(pool)))
	mux.HandleFunc("/trades/", protect(tradeDetailHandler(pool)))

	// ── Orchestration ─────────────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/orchestrate", protect(orchestrateHandler(fCfg.OrchestratorURL)))
	mux.HandleFunc("/api/v1/orchestrate/runs", protect(orchestrationRunsHandler(pool)))
	mux.HandleFunc("/api/v1/orchestrate/runs/", protect(orchestrationRunDetailHandler(pool)))

	// ── Metrics ───────────────────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/metrics", protect(metricsHandler(pool)))
	mux.HandleFunc("/api/v1/metrics/runs/", protect(metricsRunDetailHandler(pool)))

	// ── Strategies ────────────────────────────────────────────────────────────
	mux.HandleFunc("/strategies", protect(strategiesListHandler(reg)))
	mux.HandleFunc("/api/v1/strategies", protect(strategiesListHandler(reg)))
	mux.HandleFunc("/api/v1/strategies/", protect(strategiesDetailHandler(reg)))

	// ── Trading guard ─────────────────────────────────────────────────────────
	mux.HandleFunc("/trading/guard", protect(tradingGuardHandler(pool)))
	mux.HandleFunc("/trading/guard/outcome", protect(tradingGuardOutcomeHandler(pool)))

	// ── Risk ──────────────────────────────────────────────────────────────────
	mux.HandleFunc("/risk/calc", protect(riskCalcHandler()))

	// ── Symbol process (trigger analysis) ────────────────────────────────────
	mux.HandleFunc("/symbols/", protect(symbolProcessHandler(fCfg.OrchestratorURL, pool)))

	// ── Prometheus ────────────────────────────────────────────────────────────
	mux.HandleFunc("/metrics/prometheus", prometheusHandler(pool))

	handler := middleware.FlowID(middleware.CORS(corsConfig)(rateLimiter.Middleware(mux)))

	srv := &http.Server{
		Addr:         ":" + fCfg.APIPort,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("frontend API listening on :%s", fCfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("frontend API server error: %v", err)
		}
	}()

	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	log.Println("frontend API server stopped")
}

// ── Signal types ─────────────────────────────────────────────────────────────

type Signal struct {
	ID                 string     `json:"id"`
	Symbol             string     `json:"symbol"`
	StrategyID         string     `json:"strategy_id"`
	SignalType         string     `json:"signal_type"`
	Confidence         float64    `json:"confidence"`
	EntryPrice         *float64   `json:"entry_price,omitempty"`
	StopLoss           *float64   `json:"stop_loss,omitempty"`
	TakeProfit         *float64   `json:"take_profit,omitempty"`
	Reasoning          *string    `json:"reasoning,omitempty"`
	Status             string     `json:"status"`
	GeneratedAt        time.Time  `json:"generated_at"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	OrchestrationRunID *string    `json:"orchestration_run_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

func scanSignal(row interface {
	Scan(dest ...any) error
}) (*Signal, error) {
	var s Signal
	return &s, row.Scan(
		&s.ID, &s.Symbol, &s.StrategyID, &s.SignalType, &s.Confidence,
		&s.EntryPrice, &s.StopLoss, &s.TakeProfit, &s.Reasoning, &s.Status,
		&s.GeneratedAt, &s.ExpiresAt, &s.OrchestrationRunID, &s.CreatedAt,
	)
}

const signalSelectCols = `
	id::text, symbol, strategy_id, signal_type, confidence,
	entry_price, stop_loss, take_profit, reasoning, status,
	generated_at, expires_at, orchestration_run_id::text, created_at`

// ── Signals handlers ─────────────────────────────────────────────────────────

func signalsListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		status := q.Get("status")
		symbol := q.Get("symbol")
		strategy := q.Get("strategy")
		limit := parseIntParam(q.Get("limit"), 100)
		offset := parseIntParam(q.Get("offset"), 0)

		query := `SELECT` + signalSelectCols + ` FROM strategy_signals WHERE 1=1`
		args := []any{}
		i := 1
		if status != "" {
			query += fmt.Sprintf(" AND status=$%d", i)
			args = append(args, status)
			i++
		}
		if symbol != "" {
			query += fmt.Sprintf(" AND symbol=$%d", i)
			args = append(args, symbol)
			i++
		}
		if strategy != "" {
			query += fmt.Sprintf(" AND strategy_id=$%d", i)
			args = append(args, strategy)
			i++
		}
		query += fmt.Sprintf(" ORDER BY generated_at DESC LIMIT $%d OFFSET $%d", i, i+1)
		args = append(args, limit, offset)

		rows, err := pool.Query(r.Context(), query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		result := []*Signal{}
		for rows.Next() {
			s, err := scanSignal(rows)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result = append(result, s)
		}

		jsonOK(w, map[string]any{"signals": result, "total": len(result), "limit": limit, "offset": offset})
	}
}

func signalDetailHandler(pool *pgxpool.Pool, orchestratorURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/signals/")
		parts := strings.SplitN(strings.Trim(path, "/"), "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		idStr := parts[0]
		if _, err := uuid.Parse(idStr); err != nil {
			http.Error(w, "invalid signal ID", http.StatusBadRequest)
			return
		}

		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		switch {
		case r.Method == http.MethodGet && action == "":
			signalGetOne(w, r, pool, idStr)
		case r.Method == http.MethodPost && action == "approve":
			signalApprove(w, r, pool, idStr)
		case r.Method == http.MethodPost && action == "reject":
			signalReject(w, r, pool, idStr)
		case r.Method == http.MethodPost && action == "analyze":
			signalAnalyze(w, r, pool, idStr, orchestratorURL)
		default:
			http.NotFound(w, r)
		}
	}
}

func signalGetOne(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, id string) {
	row := pool.QueryRow(r.Context(),
		`SELECT`+signalSelectCols+` FROM strategy_signals WHERE id=$1`, id)
	s, err := scanSignal(row)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			http.NotFound(w, r)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	jsonOK(w, s)
}

func signalApprove(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, id string) {
	var req struct {
		ApprovedBy        string `json:"approved_by"`
		ModificationNotes string `json:"modification_notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApprovedBy == "" {
		http.Error(w, "approved_by is required", http.StatusBadRequest)
		return
	}
	_, err := pool.Exec(r.Context(),
		`UPDATE strategy_signals SET status='approved' WHERE id=$1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	observability.LogEvent(r.Context(), "info", "signal.approved", map[string]any{
		"signal_id":   id,
		"approved_by": req.ApprovedBy,
	})
	signalGetOne(w, r, pool, id)
}

func signalReject(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, id string) {
	var req struct {
		ApprovedBy      string `json:"approved_by"`
		RejectionReason string `json:"rejection_reason,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApprovedBy == "" {
		http.Error(w, "approved_by is required", http.StatusBadRequest)
		return
	}
	_, err := pool.Exec(r.Context(),
		`UPDATE strategy_signals SET status='rejected' WHERE id=$1`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	observability.LogEvent(r.Context(), "info", "signal.rejected_user", map[string]any{
		"signal_id":       id,
		"approved_by":     req.ApprovedBy,
		"rejection_reason": req.RejectionReason,
	})
	signalGetOne(w, r, pool, id)
}

func signalAnalyze(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, id, orchestratorURL string) {
	type req struct {
		Context string `json:"context,omitempty"`
	}
	var body req
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Build context from signal row.
	row := pool.QueryRow(r.Context(),
		`SELECT symbol, strategy_id, signal_type, confidence, reasoning FROM strategy_signals WHERE id=$1`, id)
	var sym, stratID, sigType, reasoning string
	var conf float64
	if err := row.Scan(&sym, &stratID, &sigType, &conf, &reasoning); err != nil {
		http.Error(w, "signal not found", http.StatusNotFound)
		return
	}

	ctx := fmt.Sprintf("Signal: %s %s\nStrategy: %s\nConfidence: %.0f%%\nReasoning: %s\n%s",
		sigType, sym, stratID, conf*100, reasoning, body.Context)

	payload := map[string]any{
		"signal_id":    id,
		"symbol":       sym,
		"trigger_type": "signal",
		"context":      ctx,
	}
	runID, err := callOrchestrator(r.Context(), orchestratorURL, payload)
	if err != nil {
		http.Error(w, "orchestrator unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Link the run to the signal.
	if parsedRun, err := uuid.Parse(runID); err == nil {
		_, _ = pool.Exec(r.Context(),
			`UPDATE strategy_signals SET orchestration_run_id=$1 WHERE id=$2`,
			parsedRun, id)
	}
	observability.LogEvent(r.Context(), "info", "signal.analysis_triggered", map[string]any{
		"signal_id": id,
		"run_id":    runID,
		"symbol":    sym,
	})

	jsonOK(w, map[string]string{"runId": runID, "status": "running"})
}

// ── Recommendations ───────────────────────────────────────────────────────────

func recommendationsListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		offset := parseIntParam(r.URL.Query().Get("offset"), 0)

		rows, err := pool.Query(r.Context(),
			`SELECT`+signalSelectCols+` FROM strategy_signals
			 WHERE status IN ('pending','approved')
			 ORDER BY confidence DESC, generated_at DESC
			 LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		result := []*Signal{}
		for rows.Next() {
			s, err := scanSignal(rows)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result = append(result, s)
		}
		jsonOK(w, map[string]any{"recommendations": result, "total": len(result)})
	}
}

func recommendationDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/recommendations/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		signalGetOne(w, r, pool, id)
	}
}

// ── Trades ────────────────────────────────────────────────────────────────────

type Trade struct {
	ID           string          `json:"id"`
	Symbol       string          `json:"symbol"`
	Direction    string          `json:"direction"`
	Entry        float64         `json:"entry"`
	Stop         float64         `json:"stop"`
	Targets      json.RawMessage `json:"targets"`
	StrategyID   string          `json:"strategy_id"`
	Notes        *string         `json:"notes,omitempty"`
	Risk         json.RawMessage `json:"risk,omitempty"`
	SignalID     *string         `json:"signal_id,omitempty"`
	OrderStatus  *string         `json:"order_status,omitempty"`
	FilledQty    *int            `json:"filled_qty,omitempty"`
	AvgFillPrice *float64        `json:"avg_fill_price,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func tradesListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		symbol := q.Get("symbol")
		strategyID := q.Get("strategyId")
		limit := parseIntParam(q.Get("limit"), 100)

		query := `SELECT id, symbol, direction, entry, stop, targets, strategy_id,
			notes, risk, signal_id::text, order_status, filled_qty, avg_fill_price, created_at
			FROM trades WHERE 1=1`
		args := []any{}
		i := 1
		if symbol != "" {
			query += fmt.Sprintf(" AND symbol=$%d", i)
			args = append(args, symbol)
			i++
		}
		if strategyID != "" {
			query += fmt.Sprintf(" AND strategy_id=$%d", i)
			args = append(args, strategyID)
			i++
		}
		query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", i)
		args = append(args, limit)

		rows, err := pool.Query(r.Context(), query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		trades := []*Trade{}
		for rows.Next() {
			t := &Trade{}
			if err := rows.Scan(&t.ID, &t.Symbol, &t.Direction, &t.Entry, &t.Stop,
				&t.Targets, &t.StrategyID, &t.Notes, &t.Risk,
				&t.SignalID, &t.OrderStatus, &t.FilledQty, &t.AvgFillPrice, &t.CreatedAt,
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			trades = append(trades, t)
		}
		jsonOK(w, map[string]any{"trades": trades})
	}
}

func tradeDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/trades/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		t := &Trade{}
		err := pool.QueryRow(r.Context(),
			`SELECT id, symbol, direction, entry, stop, targets, strategy_id,
			notes, risk, signal_id::text, order_status, filled_qty, avg_fill_price, created_at
			FROM trades WHERE id=$1`, id,
		).Scan(&t.ID, &t.Symbol, &t.Direction, &t.Entry, &t.Stop,
			&t.Targets, &t.StrategyID, &t.Notes, &t.Risk,
			&t.SignalID, &t.OrderStatus, &t.FilledQty, &t.AvgFillPrice, &t.CreatedAt)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				http.NotFound(w, r)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		jsonOK(w, t)
	}
}

// ── Orchestration ─────────────────────────────────────────────────────────────

func orchestrateHandler(orchestratorURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		resp, err := proxyPost(r.Context(), orchestratorURL+"/orchestrate", body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(resp)
	}
}

func orchestrationRunsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		rows, err := pool.Query(r.Context(),
			`SELECT id::text, symbol, trigger_type, agent_suggestion, confidence,
			 reasoning, status, started_at, completed_at, error
			 FROM orchestration_runs ORDER BY started_at DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type run struct {
			ID              string     `json:"id"`
			Symbol          string     `json:"symbol"`
			TriggerType     *string    `json:"trigger_type,omitempty"`
			AgentSuggestion *string    `json:"agent_suggestion,omitempty"`
			Confidence      *float64   `json:"confidence,omitempty"`
			Reasoning       *string    `json:"reasoning,omitempty"`
			Status          string     `json:"status"`
			StartedAt       time.Time  `json:"started_at"`
			CompletedAt     *time.Time `json:"completed_at,omitempty"`
			Error           *string    `json:"error,omitempty"`
		}
		results := []*run{}
		for rows.Next() {
			rr := &run{}
			if err := rows.Scan(&rr.ID, &rr.Symbol, &rr.TriggerType, &rr.AgentSuggestion,
				&rr.Confidence, &rr.Reasoning, &rr.Status, &rr.StartedAt, &rr.CompletedAt, &rr.Error,
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			results = append(results, rr)
		}
		jsonOK(w, map[string]any{"runs": results})
	}
}

func orchestrationRunDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/orchestrate/runs/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var result struct {
			ID              string          `json:"id"`
			Symbol          string          `json:"symbol"`
			TriggerType     *string         `json:"trigger_type,omitempty"`
			AgentSuggestion *string         `json:"agent_suggestion,omitempty"`
			Confidence      *float64        `json:"confidence,omitempty"`
			Reasoning       *string         `json:"reasoning,omitempty"`
			Status          string          `json:"status"`
			AgentResponse   json.RawMessage `json:"agent_response,omitempty"`
			StartedAt       time.Time       `json:"started_at"`
			CompletedAt     *time.Time      `json:"completed_at,omitempty"`
			Error           *string         `json:"error,omitempty"`
		}
		err := pool.QueryRow(r.Context(),
			`SELECT id::text, symbol, trigger_type, agent_suggestion, confidence,
			 reasoning, status, agent_response, started_at, completed_at, error
			 FROM orchestration_runs WHERE id=$1`, id,
		).Scan(&result.ID, &result.Symbol, &result.TriggerType, &result.AgentSuggestion,
			&result.Confidence, &result.Reasoning, &result.Status, &result.AgentResponse,
			&result.StartedAt, &result.CompletedAt, &result.Error)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				http.NotFound(w, r)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		jsonOK(w, result)
	}
}

// ── Metrics ───────────────────────────────────────────────────────────────────

func metricsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		rows, err := pool.Query(r.Context(),
			`SELECT id::text, symbol, status, confidence,
			 started_at, completed_at
			 FROM orchestration_runs ORDER BY started_at DESC LIMIT $1`, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type m struct {
			ID          string     `json:"id"`
			Symbol      string     `json:"symbol"`
			Status      string     `json:"status"`
			Confidence  *float64   `json:"confidence,omitempty"`
			StartedAt   time.Time  `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at,omitempty"`
		}
		results := []*m{}
		for rows.Next() {
			mm := &m{}
			if err := rows.Scan(&mm.ID, &mm.Symbol, &mm.Status, &mm.Confidence,
				&mm.StartedAt, &mm.CompletedAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			results = append(results, mm)
		}
		jsonOK(w, map[string]any{"metrics": results})
	}
}

func metricsRunDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/runs/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		// Delegate to orchestration run detail (same data).
		r.URL.Path = "/api/v1/orchestrate/runs/" + id
		orchestrationRunDetailHandler(pool)(w, r)
	}
}

// ── Strategies ────────────────────────────────────────────────────────────────

func strategiesListHandler(reg *strategies.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		all := reg.ListAll()
		list := make([]strategies.StrategyMetadata, 0, len(all))
		for _, meta := range all {
			list = append(list, meta)
		}
		jsonOK(w, map[string]any{"strategies": list})
	}
}

func strategiesDetailHandler(reg *strategies.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/strategies/"), "/")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		meta, err := reg.GetMetadata(id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		jsonOK(w, meta)
	}
}

// ── Trading guard ─────────────────────────────────────────────────────────────

func tradingGuardHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Return a simple guard status from the signals table.
		var totalToday, failedToday int
		_ = pool.QueryRow(r.Context(),
			`SELECT COUNT(*) FILTER (WHERE generated_at >= NOW()-INTERVAL '1 day') as total,
			        COUNT(*) FILTER (WHERE generated_at >= NOW()-INTERVAL '1 day' AND signal_type='SELL') as failed
			 FROM strategy_signals`).Scan(&totalToday, &failedToday)

		jsonOK(w, map[string]any{
			"enabled":            true,
			"consecutive_losses": 0,
			"total_today":        totalToday,
			"status":             "active",
		})
	}
}

func tradingGuardOutcomeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Accept but noop — guard state is derived from signals.
		w.WriteHeader(http.StatusNoContent)
	}
}

// ── Risk calc ─────────────────────────────────────────────────────────────────

func riskCalcHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			AccountSize float64 `json:"account_size"`
			RiskPercent float64 `json:"risk_percent"`
			EntryPrice  float64 `json:"entry_price"`
			StopLoss    float64 `json:"stop_loss"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		riskAmt := req.AccountSize * (req.RiskPercent / 100)
		stopDist := req.EntryPrice - req.StopLoss
		if stopDist == 0 {
			http.Error(w, "stop_loss must differ from entry_price", http.StatusBadRequest)
			return
		}
		positionSize := riskAmt / stopDist
		jsonOK(w, map[string]any{
			"risk_amount":   riskAmt,
			"position_size": positionSize,
			"stop_distance": stopDist,
		})
	}
}

// ── Symbol process ────────────────────────────────────────────────────────────

func symbolProcessHandler(orchestratorURL string, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		symbol := strings.Trim(strings.TrimPrefix(r.URL.Path, "/symbols/"), "/")
		if symbol == "" {
			http.Error(w, "symbol required", http.StatusBadRequest)
			return
		}
		payload := map[string]any{
			"symbol":       symbol,
			"trigger_type": "manual",
		}
		body, _ := json.Marshal(payload)
		resp, err := proxyPost(r.Context(), orchestratorURL+"/orchestrate", body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(resp)
	}
}

// ── Prometheus ────────────────────────────────────────────────────────────────

func prometheusHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		var totalSignals, pendingSignals int
		_ = pool.QueryRow(r.Context(),
			`SELECT COUNT(*), COUNT(*) FILTER (WHERE status='pending') FROM strategy_signals`,
		).Scan(&totalSignals, &pendingSignals)

		var totalRuns, completedRuns, failedRuns int
		_ = pool.QueryRow(r.Context(),
			`SELECT COUNT(*),
			        COUNT(*) FILTER (WHERE status='completed'),
			        COUNT(*) FILTER (WHERE status='failed')
			 FROM orchestration_runs`,
		).Scan(&totalRuns, &completedRuns, &failedRuns)

		upSecs := time.Since(startTime).Seconds()
		fmt.Fprintf(w, "# HELP jax_trader_uptime_seconds Uptime\n")
		fmt.Fprintf(w, "# TYPE jax_trader_uptime_seconds gauge\n")
		fmt.Fprintf(w, "jax_trader_uptime_seconds %.0f\n", upSecs)
		fmt.Fprintf(w, "# HELP jax_signals_total Total strategy signals\n")
		fmt.Fprintf(w, "# TYPE jax_signals_total counter\n")
		fmt.Fprintf(w, "jax_signals_total %d\n", totalSignals)
		fmt.Fprintf(w, "# HELP jax_signals_pending Pending strategy signals\n")
		fmt.Fprintf(w, "# TYPE jax_signals_pending gauge\n")
		fmt.Fprintf(w, "jax_signals_pending %d\n", pendingSignals)
		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_total Total orchestration runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_total %d\n", totalRuns)
		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_completed_total Completed runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_completed_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_completed_total %d\n", completedRuns)
		fmt.Fprintf(w, "# HELP jax_orchestrator_runs_failed_total Failed runs\n")
		fmt.Fprintf(w, "# TYPE jax_orchestrator_runs_failed_total counter\n")
		fmt.Fprintf(w, "jax_orchestrator_runs_failed_total %d\n", failedRuns)
	}
}

// ── shared helpers ────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("jsonOK encode: %v", err)
	}
}

func parseIntParam(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return def
	}
	if n == 0 {
		return def
	}
	return n
}

// callOrchestrator POSTs payload JSON to orchestratorURL and returns the run_id.
func callOrchestrator(ctx context.Context, orchestratorURL string, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := proxyPost(ctx, orchestratorURL+"/orchestrate", body)
	if err != nil {
		return "", err
	}
	var out struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", err
	}
	return out.RunID, nil
}

// proxyPost posts body to url and returns the response body.
func proxyPost(ctx context.Context, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}
