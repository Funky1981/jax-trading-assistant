package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"jax-trading-assistant/services/jax-api/internal/app"

	"github.com/google/uuid"
)

// SignalsHandler handles HTTP requests for signal management
type SignalsHandler struct {
	store app.SignalStore
}

// RegisterSignals registers signal endpoints
func (s *Server) RegisterSignals(store app.SignalStore) {
	h := &SignalsHandler{store: store}
	// Protected endpoints - require authentication
	s.mux.HandleFunc("/api/v1/signals", s.protect(h.handleList))
	s.mux.HandleFunc("/api/v1/signals/", s.protect(h.handleSignal))
}

// handleList handles GET /api/v1/signals - list signals with filtering
func (h *SignalsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.store == nil {
		http.Error(w, "signal store not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	q := r.URL.Query()
	status := q.Get("status")
	symbol := q.Get("symbol")
	strategy := q.Get("strategy")

	limit := 100 // Default limit
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
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
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = n
	}

	// Call store to get signals
	response, err := h.store.ListSignals(r.Context(), status, symbol, strategy, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleSignal handles signal-specific operations based on URL path
func (h *SignalsHandler) handleSignal(w http.ResponseWriter, r *http.Request) {
	if h.store == nil {
		http.Error(w, "signal store not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse the path: /api/v1/signals/{id} or /api/v1/signals/{id}/approve or /api/v1/signals/{id}/reject
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/signals/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	// Parse UUID
	id, err := uuid.Parse(parts[0])
	if err != nil {
		http.Error(w, "invalid signal ID", http.StatusBadRequest)
		return
	}

	// Route based on action
	if len(parts) == 1 {
		// GET /api/v1/signals/{id}
		h.handleGet(w, r, id)
	} else if len(parts) == 2 {
		action := parts[1]
		switch action {
		case "approve":
			// POST /api/v1/signals/{id}/approve
			h.handleApprove(w, r, id)
		case "reject":
			// POST /api/v1/signals/{id}/reject
			h.handleReject(w, r, id)
		default:
			http.NotFound(w, r)
		}
	} else {
		http.NotFound(w, r)
	}
}

// handleGet handles GET /api/v1/signals/{id} - get single signal
func (h *SignalsHandler) handleGet(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	signal, err := h.store.GetSignal(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(signal); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ApprovalRequest represents the request body for approve/reject endpoints
type ApprovalRequest struct {
	ApprovedBy        string `json:"approved_by"`
	ModificationNotes string `json:"modification_notes,omitempty"`
	RejectionReason   string `json:"rejection_reason,omitempty"`
}

// handleApprove handles POST /api/v1/signals/{id}/approve
func (h *SignalsHandler) handleApprove(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ApprovedBy == "" {
		http.Error(w, "approved_by is required", http.StatusBadRequest)
		return
	}

	// Approve the signal
	signal, err := h.store.ApproveSignal(r.Context(), id, req.ApprovedBy, req.ModificationNotes)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(signal); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// handleReject handles POST /api/v1/signals/{id}/reject
func (h *SignalsHandler) handleReject(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ApprovedBy == "" {
		http.Error(w, "approved_by is required", http.StatusBadRequest)
		return
	}

	// Reject the signal
	signal, err := h.store.RejectSignal(r.Context(), id, req.ApprovedBy, req.RejectionReason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(signal); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
