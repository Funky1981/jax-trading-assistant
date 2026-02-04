package httpapi

import (
	"encoding/json"
	"net/http"

	"jax-trading-assistant/services/jax-api/internal/app"
)

type TradingGuardHandler struct {
	guard *app.TradingGuard
}

type tradingOutcomeRequest struct {
	PnL float64 `json:"pnl"`
}

func (s *Server) RegisterTradingGuard(guard *app.TradingGuard) {
	h := &TradingGuardHandler{guard: guard}
	// Protected endpoints - require authentication
	s.mux.HandleFunc("/trading/guard", s.protect(h.handleStatus))
	s.mux.HandleFunc("/trading/guard/outcome", s.protect(h.handleOutcome))
}

func (h *TradingGuardHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.guard == nil {
		http.Error(w, "trading guard not configured", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.guard.Status()); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *TradingGuardHandler) handleOutcome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.guard == nil {
		http.Error(w, "trading guard not configured", http.StatusServiceUnavailable)
		return
	}
	var req tradingOutcomeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	h.guard.RecordOutcome(r.Context(), req.PnL)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.guard.Status()); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
