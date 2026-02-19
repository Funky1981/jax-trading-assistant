package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) RegisterMemoryAPI() {
	// Memory banks endpoint
	s.mux.HandleFunc("/v1/memory/banks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return default memory banks
		if err := json.NewEncoder(w).Encode([]string{"default", "strategies", "trades", "reflections"}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})

	// Memory search endpoint
	s.mux.HandleFunc("/v1/memory/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return empty results for now
		if err := json.NewEncoder(w).Encode([]any{}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})
}
