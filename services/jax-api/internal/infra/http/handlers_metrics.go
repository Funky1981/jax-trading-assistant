package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) RegisterMetrics() {
	// Stub metrics endpoint - returns empty array until real metrics are implemented
	s.mux.HandleFunc("/api/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
	})

	s.mux.HandleFunc("/api/v1/metrics/runs/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{})
	})
}
