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
	s.mux.HandleFunc("/strategies", h.handleList)
}

func (h *StrategiesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"strategies": h.registry.List()})
}
