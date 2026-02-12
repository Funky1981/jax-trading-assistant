package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"jax-trading-assistant/services/jax-api/internal/app"

	"github.com/google/uuid"
)

// RecommendationsHandler handles HTTP requests for AI recommendations
type RecommendationsHandler struct {
	store app.SignalStore
}

// RegisterRecommendations registers recommendation endpoints
func (s *Server) RegisterRecommendations(store app.SignalStore) {
	h := &RecommendationsHandler{store: store}
	s.mux.HandleFunc("/api/v1/recommendations", s.protect(h.handleList))
	s.mux.HandleFunc("/api/v1/recommendations/", s.protect(h.handleGet))
}

// handleList handles GET /api/v1/recommendations
func (h *RecommendationsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.store == nil {
		http.Error(w, "signal store not configured", http.StatusServiceUnavailable)
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if n > 0 {
			limit = n
		}
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = n
	}

	response, err := h.store.GetRecommendations(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleGet handles GET /api/v1/recommendations/{id}
func (h *RecommendationsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.store == nil {
		http.Error(w, "signal store not configured", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/recommendations/")
	idStr := strings.Trim(path, "/")
	if idStr == "" {
		http.NotFound(w, r)
		return
	}

	signalID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid recommendation id", http.StatusBadRequest)
		return
	}

	rec, err := h.store.GetRecommendation(r.Context(), signalID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rec); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
