package httpapi

import (
	"encoding/json"
	"net/http"

	"jax-trading-assistant/services/jax-api/internal/app"
)

type StrategiesHandler struct {
	registry *app.StrategyRegistry
}

func (s *Server) RegisterStrategies(registry *app.StrategyRegistry) {
	h := &StrategiesHandler{registry: registry}
	// Protected endpoint - requires authentication
	s.mux.HandleFunc("/strategies", s.protect(h.handleList))
}

func (h *StrategiesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"strategies": h.registry.List()}); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
