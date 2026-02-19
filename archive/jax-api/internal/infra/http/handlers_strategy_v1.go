package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/services/jax-api/internal/app"
)

type StrategyV1Handler struct {
	registry *app.StrategyRegistry
	db       *sql.DB
}

func (s *Server) RegisterStrategyV1(registry *app.StrategyRegistry, db *sql.DB) {
	h := &StrategyV1Handler{registry: registry, db: db}
	s.mux.HandleFunc("/api/v1/strategies", s.protect(h.handleList))
	s.mux.HandleFunc("/api/v1/strategies/", s.protect(h.handleStrategy))
}

func (h *StrategyV1Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type strategyItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	items := make([]strategyItem, 0)
	if h.registry != nil {
		for _, s := range h.registry.List() {
			desc := strings.TrimSpace(strings.Join([]string{
				s.EntryRule,
				s.StopRule,
				s.TargetRule,
			}, " | "))
			items = append(items, strategyItem{
				ID:          s.ID,
				Name:        s.Name,
				Description: desc,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *StrategyV1Handler) handleStrategy(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "signal store not configured", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/strategies/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, "invalid strategy path", http.StatusBadRequest)
		return
	}

	strategyID := parts[0]
	action := parts[1]

	switch action {
	case "signals":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleSignals(w, r, strategyID)
	case "performance":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handlePerformance(w, r, strategyID)
	case "analyze":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleAnalyze(w, r, strategyID)
	default:
		http.Error(w, "unknown strategy action", http.StatusNotFound)
	}
}

type strategySignalResponse struct {
	Type       string   `json:"type"`
	Symbol     string   `json:"symbol"`
	EntryPrice float64  `json:"entryPrice"`
	StopLoss   *float64 `json:"stopLoss,omitempty"`
	TakeProfit *float64 `json:"takeProfit,omitempty"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason"`
	Timestamp  string   `json:"timestamp"`
}

func (h *StrategyV1Handler) handleSignals(w http.ResponseWriter, r *http.Request, strategyID string) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}

	query := `
		SELECT signal_type, symbol, entry_price, stop_loss, take_profit, confidence, reasoning, generated_at
		FROM strategy_signals
		WHERE strategy_id = $1
		ORDER BY generated_at DESC
		LIMIT $2
	`
	rows, err := h.db.QueryContext(r.Context(), query, strategyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var signals []strategySignalResponse
	for rows.Next() {
		var (
			signalType  string
			symbol      string
			entryPrice  sql.NullFloat64
			stopLoss    sql.NullFloat64
			takeProfit  sql.NullFloat64
			confidence  float64
			reasoning   sql.NullString
			generatedAt time.Time
		)
		if err := rows.Scan(&signalType, &symbol, &entryPrice, &stopLoss, &takeProfit, &confidence, &reasoning, &generatedAt); err != nil {
			continue
		}

		signal := strategySignalResponse{
			Type:       strings.ToLower(signalType),
			Symbol:     symbol,
			EntryPrice: entryPrice.Float64,
			Confidence: confidence,
			Reason:     reasoning.String,
			Timestamp:  generatedAt.UTC().Format(time.RFC3339),
		}
		if stopLoss.Valid {
			val := stopLoss.Float64
			signal.StopLoss = &val
		}
		if takeProfit.Valid {
			val := takeProfit.Float64
			signal.TakeProfit = &val
		}
		signals = append(signals, signal)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signals)
}

func (h *StrategyV1Handler) handlePerformance(w http.ResponseWriter, r *http.Request, strategyID string) {
	query := `
		SELECT COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE status = 'approved') AS approved,
		       MAX(created_at) AS last_updated
		FROM strategy_signals
		WHERE strategy_id = $1
	`
	var total, approved int
	var lastUpdated sql.NullTime
	if err := h.db.QueryRowContext(r.Context(), query, strategyID).Scan(&total, &approved, &lastUpdated); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	winRate := 0.0
	if total > 0 {
		winRate = float64(approved) / float64(total)
	}

	resp := map[string]interface{}{
		"strategyId":        strategyID,
		"winRate":           winRate,
		"avgReturn":         0.0,
		"totalSignals":      total,
		"successfulSignals": approved,
		"lastUpdated":       time.Now().UTC().Format(time.RFC3339),
	}
	if lastUpdated.Valid {
		resp["lastUpdated"] = lastUpdated.Time.UTC().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type analyzeRequest struct {
	Symbol      string                 `json:"symbol"`
	Constraints map[string]interface{} `json:"constraints"`
}

func (h *StrategyV1Handler) handleAnalyze(w http.ResponseWriter, r *http.Request, strategyID string) {
	var req analyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT signal_type, symbol, entry_price, stop_loss, take_profit, confidence, reasoning, generated_at
		FROM strategy_signals
		WHERE strategy_id = $1 AND symbol = $2
		ORDER BY generated_at DESC
		LIMIT 1
	`
	var (
		signalType  string
		symbol      string
		entryPrice  sql.NullFloat64
		stopLoss    sql.NullFloat64
		takeProfit  sql.NullFloat64
		confidence  float64
		reasoning   sql.NullString
		generatedAt time.Time
	)
	err := h.db.QueryRowContext(r.Context(), query, strategyID, req.Symbol).Scan(
		&signalType, &symbol, &entryPrice, &stopLoss, &takeProfit, &confidence, &reasoning, &generatedAt,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "no recent signal for strategy/symbol", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := strategySignalResponse{
		Type:       strings.ToLower(signalType),
		Symbol:     symbol,
		EntryPrice: entryPrice.Float64,
		Confidence: confidence,
		Reason:     reasoning.String,
		Timestamp:  generatedAt.UTC().Format(time.RFC3339),
	}
	if stopLoss.Valid {
		val := stopLoss.Float64
		resp.StopLoss = &val
	}
	if takeProfit.Valid {
		val := takeProfit.Float64
		resp.TakeProfit = &val
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
