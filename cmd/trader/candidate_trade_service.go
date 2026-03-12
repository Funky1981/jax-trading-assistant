package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
)

// candidateTradeService wraps the candidates module for HTTP handler use.
type candidateTradeService struct {
	svc *candidatesmod.Service
}

func newCandidateTradeService(pool *pgxpool.Pool) *candidateTradeService {
	return &candidateTradeService{svc: candidatesmod.NewService(candidatesmod.NewStore(pool))}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// GET /api/v1/candidates
func candidatesListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	cts := newCandidateTradeService(pool)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status := r.URL.Query().Get("status")
		symbol := r.URL.Query().Get("symbol")
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
				limit = v
			}
		}
		list, err := cts.svc.List(r.Context(), status, symbol, limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("list candidates: %v", err), http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []*candidatesmod.Candidate{}
		}
		jsonOK(w, list)
	}
}

// GET /api/v1/candidates/{id}
// POST /api/v1/candidates/{id}/refresh
func candidatesDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	cts := newCandidateTradeService(pool)
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/candidates/")
		parts := strings.SplitN(path, "/", 2)
		rawID := parts[0]
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		id, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "invalid candidate id", http.StatusBadRequest)
			return
		}

		switch {
		case r.Method == http.MethodGet && action == "":
			c, err := cts.svc.GetByID(r.Context(), id)
			if err != nil {
				if strings.Contains(err.Error(), "no rows") {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, c)

		case r.Method == http.MethodPost && action == "refresh":
			handleCandidateRefresh(w, r, cts, id)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
}

// POST /api/v1/candidates/{id}/refresh — re-qualify a detected/blocked candidate.
func handleCandidateRefresh(w http.ResponseWriter, r *http.Request, cts *candidateTradeService, id uuid.UUID) {
	c, err := cts.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "candidate not found", http.StatusNotFound)
		return
	}
	// Only detected or blocked candidates can be refreshed.
	if c.Status != candidatesmod.StatusDetected && c.Status != candidatesmod.StatusBlocked {
		http.Error(w, fmt.Sprintf("cannot refresh candidate in status %q", c.Status), http.StatusConflict)
		return
	}
	if err := cts.svc.Qualify(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("qualify: %v", err), http.StatusInternalServerError)
		return
	}
	updated, _ := cts.svc.GetByID(r.Context(), id)
	publishEvent("candidate.qualified", map[string]any{
		"candidateId": id,
		"symbol":      c.Symbol,
		"refreshedAt": time.Now().UTC(),
	})
	jsonOK(w, updated)
}

// ── Internal propose helper (called by watcher integration) ──────────────────

func proposeCandidate(ctx context.Context, pool *pgxpool.Pool, req candidatesmod.ProposalRequest) (*candidatesmod.Candidate, error) {
	cts := newCandidateTradeService(pool)
	c, err := cts.svc.Propose(ctx, req)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(c)
	publishEvent("candidate.detected", json.RawMessage(b))
	return c, nil
}
