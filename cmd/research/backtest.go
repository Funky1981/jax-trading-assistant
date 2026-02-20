package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"jax-trading-assistant/internal/modules/backtest"
	"jax-trading-assistant/libs/dataset"
	"jax-trading-assistant/libs/strategies"
)

// ─── Backtest HTTP handler (L04) ──────────────────────────────────────────────

// backtestDeps are created once at startup and closed over per request.
type backtestDeps struct {
	engine   *backtest.Engine
	datasets *dataset.Registry
}

// newBacktestDeps wires up the backtest engine and dataset registry.
// The dataset catalog directory is configurable via DATASET_DIR env var;
// it defaults to "data/datasets" relative to the working directory.
func newBacktestDeps(registry *strategies.Registry, datasetDir string) (*backtestDeps, error) {
	if datasetDir == "" {
		datasetDir = filepath.Join("data", "datasets")
	}

	ds, err := dataset.Open(datasetDir)
	if err != nil {
		return nil, fmt.Errorf("backtest: open dataset registry at %q: %w", datasetDir, err)
	}

	return &backtestDeps{
		engine:   backtest.New(registry),
		datasets: ds,
	}, nil
}

// ─── request / response types ─────────────────────────────────────────────────

// BacktestRequest is the POST /backtest JSON payload.
type BacktestRequest struct {
	// Strategy must match a registered strategy name.
	Strategy string `json:"strategy"`
	// Symbols is the list of tickers to back-test.
	Symbols []string `json:"symbols"`
	// StartDate / EndDate in YYYY-MM-DD format (inclusive).
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	// InitialCapital in USD.  Defaults to 100 000 when 0.
	InitialCapital float64 `json:"initial_capital"`
	// RiskPerTrade as a fraction (e.g. 0.01 = 1 %).  Defaults to 0.01 when 0.
	RiskPerTrade float64 `json:"risk_per_trade"`
	// DatasetID is the UUID of a registered dataset from libs/dataset.
	// Required — the research runtime does not have a live broker connection.
	DatasetID string `json:"dataset_id"`
	// Seed makes the run deterministic.  0 = auto-generate from wall clock.
	Seed int64 `json:"seed"`
}

// BacktestResponse is the JSON payload returned on success.
type BacktestResponse struct {
	RunID      string   `json:"run_id"`
	Strategy   string   `json:"strategy"`
	Symbols    []string `json:"symbols"`
	Seed       int64    `json:"seed"`
	DurationMs int64    `json:"duration_ms"`
	// Core metrics forwarded from strategies.BacktestResult
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalReturn   float64 `json:"total_return"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	FinalCapital  float64 `json:"final_capital"`
	// DatasetInfo shows which dataset was used (for reproducibility).
	DatasetID   string `json:"dataset_id"`
	DatasetHash string `json:"dataset_hash,omitempty"`
}

// ─── handler ──────────────────────────────────────────────────────────────────

const dateFmt = "2006-01-02"

// handleBacktest processes POST /backtest.
func handleBacktest(deps *backtestDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req BacktestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		if err := validateBacktestRequest(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		startDate, _ := time.Parse(dateFmt, req.StartDate)
		endDate, _ := time.Parse(dateFmt, req.EndDate)

		// Load dataset from registry.
		ds, err := deps.datasets.Get(req.DatasetID)
		if err != nil {
			http.Error(w, fmt.Sprintf("dataset not found: %v", err), http.StatusBadRequest)
			return
		}

		// Guard against mutated files breaking reproducibility.
		if err := deps.datasets.VerifyHash(req.DatasetID); err != nil {
			log.Printf("[backtest] WARNING: dataset integrity failure: %v", err)
			http.Error(w, fmt.Sprintf("dataset integrity check failed: %v", err), http.StatusConflict)
			return
		}

		csvSrc, err := deps.datasets.LoadDataSource(r.Context(), req.DatasetID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load dataset: %v", err), http.StatusInternalServerError)
			return
		}

		cfg := backtest.Config{
			StrategyName:   req.Strategy,
			Symbols:        req.Symbols,
			StartDate:      startDate,
			EndDate:        endDate,
			DataSource:     csvSrc,
			Seed:           req.Seed,
			InitialCapital: req.InitialCapital,
			RiskPerTrade:   req.RiskPerTrade,
		}

		log.Printf("[backtest] running strategy=%q symbols=%v dataset=%s seed=%d",
			req.Strategy, req.Symbols, req.DatasetID, req.Seed)

		result, err := deps.engine.Run(r.Context(), cfg)
		if err != nil {
			log.Printf("[backtest] error: %v", err)
			http.Error(w, fmt.Sprintf("backtest failed: %v", err), http.StatusInternalServerError)
			return
		}

		resp := BacktestResponse{
			RunID:         result.RunID,
			Strategy:      req.Strategy,
			Symbols:       result.Symbols,
			Seed:          result.Seed,
			DurationMs:    result.DurationMs,
			TotalTrades:   result.TotalTrades,
			WinningTrades: result.WinningTrades,
			LosingTrades:  result.LosingTrades,
			WinRate:       result.WinRate,
			TotalReturn:   result.TotalReturn,
			SharpeRatio:   result.SharpeRatio,
			MaxDrawdown:   result.MaxDrawdown,
			FinalCapital:  result.FinalCapital,
			DatasetID:     ds.ID,
			DatasetHash:   ds.Hash[:12],
		}

		log.Printf("[backtest] complete run=%s trades=%d winRate=%.1f%% totalReturn=%.2f%%",
			result.RunID, result.TotalTrades, result.WinRate*100, result.TotalReturn*100)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

// validateBacktestRequest returns an error for any missing required field.
func validateBacktestRequest(req BacktestRequest) error {
	if req.Strategy == "" {
		return fmt.Errorf("strategy is required")
	}
	if len(req.Symbols) == 0 {
		return fmt.Errorf("at least one symbol is required")
	}
	if req.StartDate == "" {
		return fmt.Errorf("start_date is required (YYYY-MM-DD)")
	}
	if req.EndDate == "" {
		return fmt.Errorf("end_date is required (YYYY-MM-DD)")
	}
	if _, err := time.Parse(dateFmt, req.StartDate); err != nil {
		return fmt.Errorf("start_date must be YYYY-MM-DD, got %q", req.StartDate)
	}
	if _, err := time.Parse(dateFmt, req.EndDate); err != nil {
		return fmt.Errorf("end_date must be YYYY-MM-DD, got %q", req.EndDate)
	}
	if req.DatasetID == "" {
		return fmt.Errorf("dataset_id is required (research runtime has no live broker connection)")
	}
	return nil
}
