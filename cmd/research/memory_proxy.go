// memory_proxy.go — Part of cmd/research (package main).
// Registers the memory HTTP endpoints on the research runtime's mux
// (ADR-0012 Phase 6). Backing store is Postgres + pgvector with no
// in-memory fallback.
//
// Endpoints exposed:
//
//	POST /tools                                  UTCP dispatcher (retain/recall/reflect)
//	GET  /v1/memory/banks                        list banks → string[]
//	GET  /v1/memory/search                       search → { items }
//	GET  /v1/memory/banks/{bank}/items           list items in bank
//	POST /v1/memory/banks/{bank}/items           create item in bank
//	GET  /v1/memory/banks/{bank}/items/{id}      get item by id
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/pgmemory"
)

// buildMemoryStore constructs a Postgres/pgvector-backed MemoryStore.
// The DB connection is required; the service fails to start if it is nil.
// Embedding configuration is read from the environment:
//
//	OPENAI_API_KEY   – required for embedding generation
//	OPENAI_BASE_URL  – optional custom API root (Azure OpenAI, local proxy)
//	EMBEDDING_MODEL  – optional embedding model (default: text-embedding-3-small)
func buildMemoryStore(db *sql.DB) (*pgmemory.Store, error) {
	cfg := pgmemory.OpenAIEmbedderConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Model:   os.Getenv("EMBEDDING_MODEL"),
	}
	if err := pgmemory.ValidateOpenAIEmbedderConfig(cfg); err != nil {
		return nil, err
	}
	embedder := pgmemory.NewOpenAIEmbedder(cfg)
	store := pgmemory.New(db, embedder)
	log.Println("memory store → postgres (pgvector)")
	return store, nil
}

// registerMemoryRoutes adds memory endpoints to mux.
func registerMemoryRoutes(mux *http.ServeMux, store contracts.MemoryStore) {
	mux.HandleFunc("/tools", memoryToolHandler(store))
	mux.HandleFunc("/v1/memory/banks", memoryBanksHandler())
	mux.HandleFunc("/v1/memory/search", memorySearchHandler(store))
	// Item CRUD — prefix-matched, method dispatch inside the router.
	mux.HandleFunc("/v1/memory/banks/", memoryBankRouter(store))
	log.Println("memory proxy routes registered: /tools /v1/memory/banks /v1/memory/search /v1/memory/banks/{bank}/items")
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
		if err := json.NewEncoder(w).Encode(pgmemory.Banks()); err != nil {
			http.Error(w, "failed to encode banks", http.StatusInternalServerError)
		}
	}
}

// ── /v1/memory/search ────────────────────────────────────────────────────────

func memorySearchHandler(store contracts.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil {
			http.Error(w, "memory store not configured", http.StatusServiceUnavailable)
			return
		}
		bank := strings.TrimSpace(r.URL.Query().Get("bank"))
		if bank == "" {
			http.Error(w, "bank is required", http.StatusBadRequest)
			return
		}
		query, err := memoryQueryFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		items, err := store.Recall(r.Context(), bank, query)
		if err != nil {
			writeMemoryStoreError(w, err)
			return
		}
		if items == nil {
			items = []contracts.MemoryItem{}
		}
		w.Header().Set("Content-Type", "application/json")
		if items == nil {
			items = []contracts.MemoryItem{}
		}

		if err := json.NewEncoder(w).Encode(contracts.MemoryRecallResponse{Items: items}); err != nil {
			http.Error(w, "failed to encode search results", http.StatusInternalServerError)
		}
	}
}

// ── /v1/memory/banks/{bank}/items[/{id}] ────────────────────────────────────

// memoryBankRouter dispatches item CRUD requests for a named bank.
// Path pattern: /v1/memory/banks/{bank}/items[/{id}]
func memoryBankRouter(store contracts.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip the /v1/memory/banks/ prefix.
		rest := strings.TrimPrefix(r.URL.Path, "/v1/memory/banks/")
		// rest = "{bank}/items" or "{bank}/items/{id}"
		parts := strings.SplitN(rest, "/", 3)
		if len(parts) < 2 || parts[1] != "items" {
			http.NotFound(w, r)
			return
		}
		bank := parts[0]
		if bank == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 2 {
			// /v1/memory/banks/{bank}/items
			switch r.Method {
			case http.MethodGet:
				memoryItemListHandler(store, bank)(w, r)
			case http.MethodPost:
				memoryItemCreateHandler(store, bank)(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		// /v1/memory/banks/{bank}/items/{id}
		id := parts[2]
		if id == "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		memoryItemGetHandler(store, bank, id)(w, r)
	}
}

func memoryItemListHandler(store contracts.MemoryStore, bank string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := memoryQueryFromRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		items, err := store.Recall(r.Context(), bank, query)
		if err != nil {
			writeMemoryStoreError(w, err)
			return
		}
		if items == nil {
			items = []contracts.MemoryItem{}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(contracts.MemoryRecallResponse{Items: items}); err != nil {
			log.Printf("memoryItemListHandler encode: %v", err)
		}
	}
}

func memoryItemCreateHandler(store contracts.MemoryStore, bank string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var item contracts.MemoryItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		item.Tags = contracts.NormalizeMemoryTags(item.Tags)
		if item.TS.IsZero() {
			item.TS = time.Now().UTC()
		}
		if err := contracts.ValidateMemoryItem(item); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, err := store.Retain(r.Context(), bank, item)
		if err != nil {
			writeMemoryStoreError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(contracts.MemoryRetainResponse{ID: id}); err != nil {
			log.Printf("memoryItemCreateHandler encode: %v", err)
		}
	}
}

// getByIDer is a subset of *pgmemory.Store used by the get-by-id handler.
type getByIDer interface {
	GetByID(ctx context.Context, bank, id string) (contracts.MemoryItem, error)
}

func memoryItemGetHandler(store contracts.MemoryStore, bank, id string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gs, ok := store.(getByIDer)
		if !ok {
			http.Error(w, "not implemented", http.StatusNotImplemented)
			return
		}
		item, err := gs.GetByID(r.Context(), bank, id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(item); err != nil {
			log.Printf("memoryItemGetHandler encode: %v", err)
		}
	}
}

// buildMemoryStore helper — context with timeout
func newTimeoutCtx(secs int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
}

func memoryQueryFromRequest(r *http.Request) (contracts.MemoryQuery, error) {
	query := contracts.MemoryQuery{
		Q:      strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("q"), r.URL.Query().Get("query"))),
		Symbol: strings.TrimSpace(r.URL.Query().Get("symbol")),
		Types:  splitQueryValues(r, "types", "type"),
		Tags:   contracts.NormalizeMemoryTags(splitQueryValues(r, "tags", "tag")),
		Limit:  20,
	}

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			return contracts.MemoryQuery{}, fmt.Errorf("limit must be an integer")
		}
		query.Limit = limit
	}

	if from, err := parseOptionalTime(firstNonEmpty(r.URL.Query().Get("from"), r.URL.Query().Get("since"))); err != nil {
		return contracts.MemoryQuery{}, fmt.Errorf("invalid from/since timestamp")
	} else if from != nil {
		query.From = from
	}
	if to, err := parseOptionalTime(firstNonEmpty(r.URL.Query().Get("to"), r.URL.Query().Get("until"))); err != nil {
		return contracts.MemoryQuery{}, fmt.Errorf("invalid to/until timestamp")
	} else if to != nil {
		query.To = to
	}

	return query, nil
}

func splitQueryValues(r *http.Request, keys ...string) []string {
	var values []string
	for _, key := range keys {
		for _, raw := range r.URL.Query()[key] {
			for _, part := range strings.Split(raw, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					values = append(values, part)
				}
			}
		}
	}
	return values
}

func parseOptionalTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writeMemoryStoreError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	msg := err.Error()
	switch {
	case strings.Contains(msg, "bank is required"),
		strings.Contains(msg, "unknown bank"),
		strings.Contains(msg, "params.query is required"),
		strings.Contains(msg, "memory item"):
		status = http.StatusBadRequest
	case strings.Contains(msg, "duplicate source reference"):
		status = http.StatusConflict
	}
	http.Error(w, msg, status)
}
