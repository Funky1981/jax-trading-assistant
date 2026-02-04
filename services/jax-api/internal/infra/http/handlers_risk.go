package httpapi

import (
	"encoding/json"
	"net/http"

	"jax-trading-assistant/services/jax-api/internal/app"
)

type RiskHandler struct {
	engine *app.RiskEngine
}

func (s *Server) RegisterRisk(engine *app.RiskEngine) {
	h := &RiskHandler{engine: engine}
	// Protected endpoint - requires authentication
	s.mux.HandleFunc("/risk/calc", s.protect(h.handleCalc))
}

type riskCalcRequest struct {
	AccountSize float64 `json:"accountSize"`
	RiskPercent float64 `json:"riskPercent"`
	Entry       float64 `json:"entry"`
	Stop        float64 `json:"stop"`
	Target      float64 `json:"target,omitempty"`
}

func (h *RiskHandler) handleCalc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req riskCalcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	var target *float64
	if req.Target != 0 {
		target = &req.Target
	}

	out, err := h.engine.Calculate(r.Context(), req.AccountSize, req.RiskPercent, req.Entry, req.Stop, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
