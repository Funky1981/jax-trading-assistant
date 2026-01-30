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
		_ = json.NewEncoder(w).Encode([]string{"default", "strategies", "trades", "reflections"})
	})

	// Memory search endpoint
	s.mux.HandleFunc("/v1/memory/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return empty results for now
		_ = json.NewEncoder(w).Encode([]any{})
	})
}
