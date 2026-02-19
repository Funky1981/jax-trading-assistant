package httpapi

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) RegisterHealth() {
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if s.store == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"ok":        false,
				"healthy":   false,
				"status":    "unhealthy",
				"error":     "memory store not configured",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}); err != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
			return
		}
		if err := s.store.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if encErr := json.NewEncoder(w).Encode(map[string]any{
				"ok":        false,
				"healthy":   false,
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}); encErr != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"healthy":   true,
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})
}
