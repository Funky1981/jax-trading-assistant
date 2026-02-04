package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) RegisterMetrics() {
	// Stub metrics endpoint - returns empty array until real metrics are implemented
	// Protected endpoints - require authentication
	s.mux.HandleFunc("/api/v1/metrics", s.protect(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]any{}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}))

	s.mux.HandleFunc("/api/v1/metrics/runs/", s.protect(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]any{}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}))
}
