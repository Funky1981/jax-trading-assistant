package httpapi

import (
	"encoding/json"
	"net/http"

	"jax-trading-assistant/libs/contracts"
)

type toolRequest struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

type toolResponse struct {
	Output any `json:"output"`
}

func (s *Server) RegisterTools() {
	s.mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.store == nil {
			http.Error(w, "memory store not configured", http.StatusServiceUnavailable)
			return
		}

		var req toolRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		var out any
		switch req.Tool {
		case "memory.retain":
			var in struct {
				Bank string               `json:"bank"`
				Item contracts.MemoryItem `json:"item"`
			}
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			id, err := s.store.Retain(r.Context(), in.Bank, in.Item)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = map[string]any{"id": id}
		case "memory.recall":
			var in struct {
				Bank  string                `json:"bank"`
				Query contracts.MemoryQuery `json:"query"`
			}
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			items, err := s.store.Recall(r.Context(), in.Bank, in.Query)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = map[string]any{"items": items}
		case "memory.reflect":
			var in struct {
				Bank   string                     `json:"bank"`
				Params contracts.ReflectionParams `json:"params"`
			}
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			items, err := s.store.Reflect(r.Context(), in.Bank, in.Params)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = map[string]any{"items": items}
		default:
			http.Error(w, "unknown tool", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toolResponse{Output: out})
	})
}
