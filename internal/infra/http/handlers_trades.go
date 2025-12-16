package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"jax-trading-assistant/internal/app"
)

type TradesHandler struct {
	store app.TradeStore
}

func (s *Server) RegisterTrades(store app.TradeStore) {
	h := &TradesHandler{store: store}
	s.mux.HandleFunc("/trades", h.handleList)
	s.mux.HandleFunc("/trades/", h.handleGet)
}

func (h *TradesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.store == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	q := r.URL.Query()
	symbol := q.Get("symbol")
	strategyID := q.Get("strategyId")
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = n
	}

	trades, err := h.store.ListTrades(r.Context(), symbol, strategyID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"trades": trades})
}

func (h *TradesHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.store == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/trades/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	trade, err := h.store.GetTrade(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(trade)
}
