package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) RegisterHealth() {
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
}
