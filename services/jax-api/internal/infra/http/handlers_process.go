package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"jax-trading-assistant/services/jax-api/internal/app"
)

type ProcessHandler struct {
	orchestrator     *app.Orchestrator
	defaultAccount   float64
	defaultRiskPct   float64
	defaultGapThresh float64
}

func (s *Server) RegisterProcess(orchestrator *app.Orchestrator, defaultAccountSize float64, defaultRiskPercent float64, defaultGapThresholdPct float64) {
	h := &ProcessHandler{
		orchestrator:     orchestrator,
		defaultAccount:   defaultAccountSize,
		defaultRiskPct:   defaultRiskPercent,
		defaultGapThresh: defaultGapThresholdPct,
	}
	s.mux.HandleFunc("/symbols/", h.handle)
}

type processRequest struct {
	AccountSize     float64 `json:"accountSize,omitempty"`
	RiskPercent     float64 `json:"riskPercent,omitempty"`
	GapThresholdPct float64 `json:"gapThresholdPct,omitempty"`
}

func (h *ProcessHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/symbols/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "process" || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	symbol := parts[0]

	var req processRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	account := req.AccountSize
	if account == 0 {
		account = h.defaultAccount
	}
	riskPct := req.RiskPercent
	if riskPct == 0 {
		riskPct = h.defaultRiskPct
	}
	gapThresh := req.GapThresholdPct
	if gapThresh == 0 {
		gapThresh = h.defaultGapThresh
	}

	out, err := h.orchestrator.ProcessSymbol(r.Context(), symbol, account, riskPct, gapThresh)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
