// memory_proxy.go — Part of cmd/research (package main).
// Registers the jax-memory-compatible HTTP endpoints on the research runtime's
// mux so that agent0-service can point MEMORY_SERVICE_URL at jax-research:8091
// and jax-memory can be removed from docker-compose (ADR-0012 Phase 6).
//
// Endpoints exposed:
//
//	POST /tools                 – UTCP tool dispatcher (memory.retain/recall/reflect)
//	GET  /v1/memory/banks       – list available banks
//	GET  /v1/memory/search      – search memories (delegated to recall)
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/hindsight"
	"jax-trading-assistant/libs/testing"
)

// buildMemoryStore creates a contracts.MemoryStore backed by Hindsight if the
// HINDSIGHT_URL env var is set, otherwise falls back to an in-memory store.
func buildMemoryStore() contracts.MemoryStore {
	url := os.Getenv("HINDSIGHT_URL")
	if url == "" {
		log.Println("memory store: HINDSIGHT_URL not set, using in-memory store")
		return testing.NewInMemoryMemoryStore()
	}
	client, err := hindsight.New(url)
	if err != nil {
		log.Printf("memory store: invalid HINDSIGHT_URL (%v), using in-memory fallback", err)
		return testing.NewInMemoryMemoryStore()
	}
	ctx, cancel := newTimeoutCtx(5)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		log.Printf("memory store: hindsight ping failed (%v), using in-memory fallback", err)
		return testing.NewInMemoryMemoryStore()
	}
	log.Printf("memory store → hindsight %s", url)
	return client
}

// registerMemoryRoutes adds jax-memory-compatible endpoints to mux.
func registerMemoryRoutes(mux *http.ServeMux, store contracts.MemoryStore) {
	mux.HandleFunc("/tools", memoryToolHandler(store))
	mux.HandleFunc("/v1/memory/banks", memoryBanksHandler())
	mux.HandleFunc("/v1/memory/search", memorySearchHandler(store))
	log.Println("memory proxy routes registered: /tools, /v1/memory/banks, /v1/memory/search")
}

// ── /tools ────────────────────────────────────────────────────────────────────

type toolRequest struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

type toolResponse struct {
	Output any `json:"output"`
}

func memoryToolHandler(store contracts.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if store == nil {
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
			id, err := store.Retain(r.Context(), in.Bank, in.Item)
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
			items, err := store.Recall(r.Context(), in.Bank, in.Query)
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
			items, err := store.Reflect(r.Context(), in.Bank, in.Params)
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
	}
}

// ── /v1/memory/banks ─────────────────────────────────────────────────────────

func memoryBanksHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]string{"default", "strategies", "trades", "reflections"})
	}
}

// ── /v1/memory/search ────────────────────────────────────────────────────────

func memorySearchHandler(store contracts.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if store == nil {
			json.NewEncoder(w).Encode([]any{})
			return
		}
		q := r.URL.Query().Get("q")
		bank := r.URL.Query().Get("bank")
		if bank == "" {
			bank = "default"
		}
		if q == "" {
			json.NewEncoder(w).Encode([]any{})
			return
		}
		items, err := store.Recall(r.Context(), bank, contracts.MemoryQuery{Q: q, Limit: 20})
		if err != nil {
			log.Printf("memory search error: %v", err)
			json.NewEncoder(w).Encode([]any{})
			return
		}
		json.NewEncoder(w).Encode(items)
	}
}

// buildMemoryStore helper — context with timeout
func newTimeoutCtx(secs int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
}
