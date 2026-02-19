package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
			var in contracts.MemoryRetainRequest
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(in.Bank) == "" {
				http.Error(w, "bank is required", http.StatusBadRequest)
				return
			}
			in.Item.Tags = contracts.NormalizeMemoryTags(in.Item.Tags)
			if err := contracts.ValidateMemoryItem(in.Item); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Printf("memory.retain bank=%s type=%s", in.Bank, in.Item.Type)
			id, err := s.store.Retain(r.Context(), in.Bank, in.Item)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = contracts.MemoryRetainResponse{ID: id}
		case "memory.recall":
			var in contracts.MemoryRecallRequest
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(in.Bank) == "" {
				http.Error(w, "bank is required", http.StatusBadRequest)
				return
			}
			in.Query.Tags = contracts.NormalizeMemoryTags(in.Query.Tags)
			log.Printf("memory.recall bank=%s q=%s", in.Bank, in.Query.Q)
			items, err := s.store.Recall(r.Context(), in.Bank, in.Query)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = contracts.MemoryRecallResponse{Items: items}
		case "memory.reflect":
			var in contracts.MemoryReflectRequest
			if err := json.Unmarshal(req.Input, &in); err != nil {
				http.Error(w, "invalid input", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(in.Bank) == "" {
				http.Error(w, "bank is required", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(in.Params.Query) == "" {
				http.Error(w, "params.query is required", http.StatusBadRequest)
				return
			}
			log.Printf("memory.reflect bank=%s window_days=%d", in.Bank, in.Params.WindowDays)
			items, err := s.store.Reflect(r.Context(), in.Bank, in.Params)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			out = contracts.MemoryReflectResponse{Items: items}
		default:
			http.Error(w, "unknown tool", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(toolResponse{Output: out}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})
}
