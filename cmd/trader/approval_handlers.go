package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
)

// registerApprovalRoutes registers all human-approval-flow endpoints on mux.
func registerApprovalRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	svc := approvalsmod.NewService(pool)

	// GET  /api/v1/approvals/queue
	mux.HandleFunc("/api/v1/approvals/queue", protect(approvalQueueHandler(svc)))

	// GET  /api/v1/approvals/{candidateId}
	// POST /api/v1/approvals/{candidateId}/approve
	// POST /api/v1/approvals/{candidateId}/reject
	// POST /api/v1/approvals/{candidateId}/snooze
	// POST /api/v1/approvals/{candidateId}/reanalyze
	mux.HandleFunc("/api/v1/approvals/", protect(approvalDetailRouter(svc)))
}

// GET /api/v1/approvals/queue
func approvalQueueHandler(svc *approvalsmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
				limit = v
			}
		}
		queue, err := svc.GetQueue(r.Context(), limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("approvals queue: %v", err), http.StatusInternalServerError)
			return
		}
		if queue == nil {
			queue = []map[string]any{}
		}
		jsonOK(w, queue)
	}
}

// Router for /api/v1/approvals/{candidateId}[/action]
func approvalDetailRouter(svc *approvalsmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip prefix and split: {candidateId}[/{action}]
		tail := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
		parts := strings.SplitN(tail, "/", 2)
		rawID := parts[0]
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		// Special sub-path: /api/v1/approvals/queue — handled above
		if rawID == "queue" {
			http.NotFound(w, r)
			return
		}

		candidateID, err := uuid.Parse(rawID)
		if err != nil {
			http.Error(w, "invalid candidate id", http.StatusBadRequest)
			return
		}

		switch {
		case r.Method == http.MethodGet && action == "":
			handleApprovalGet(w, r, svc, candidateID)
		case r.Method == http.MethodPost && action == "approve":
			handleApprovalDecision(w, r, svc, candidateID, approvalsmod.DecisionApproved)
		case r.Method == http.MethodPost && action == "reject":
			handleApprovalDecision(w, r, svc, candidateID, approvalsmod.DecisionRejected)
		case r.Method == http.MethodPost && action == "snooze":
			handleApprovalSnooze(w, r, svc, candidateID)
		case r.Method == http.MethodPost && action == "reanalyze":
			handleApprovalDecision(w, r, svc, candidateID, approvalsmod.DecisionReanalysisRequested)
		default:
			http.NotFound(w, r)
		}
	}
}

// GET /api/v1/approvals/{candidateId}
func handleApprovalGet(w http.ResponseWriter, r *http.Request, svc *approvalsmod.Service, candidateID uuid.UUID) {
	a, err := svc.GetByCandidate(r.Context(), candidateID)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			http.Error(w, "no approval found for candidate", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, a)
}

// approvalDecisionBody is the optional JSON body for an approval action.
type approvalDecisionBody struct {
	Notes    *string    `json:"notes"`
	ExpiryAt *time.Time `json:"expiryAt"`
}

func handleApprovalDecision(w http.ResponseWriter, r *http.Request, svc *approvalsmod.Service, candidateID uuid.UUID, decision string) {
	var body approvalDecisionBody
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	actor := actorFromRequest(r)
	req := approvalsmod.ApprovalRequest{
		CandidateID: candidateID,
		Decision:    decision,
		ApprovedBy:  actor,
		Notes:       body.Notes,
		ExpiryAt:    body.ExpiryAt,
	}
	approval, err := svc.Decide(r.Context(), req)
	if err != nil {
		switch err {
		case approvalsmod.ErrCandidateExpired:
			http.Error(w, err.Error(), http.StatusGone)
		default:
			if strings.Contains(err.Error(), "not in awaiting_approval") {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	publishEvent("approval."+decision, map[string]any{
		"candidateId": candidateID,
		"approvalId":  approval.ID,
		"decision":    decision,
		"approvedBy":  actor,
		"decidedAt":   approval.DecidedAt,
	})
	jsonOK(w, approval)
}

type snoozeBody struct {
	Notes       *string `json:"notes"`
	SnoozeHours int     `json:"snoozeHours"`
}

func handleApprovalSnooze(w http.ResponseWriter, r *http.Request, svc *approvalsmod.Service, candidateID uuid.UUID) {
	var body snoozeBody
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	if body.SnoozeHours <= 0 {
		body.SnoozeHours = 4
	}
	actor := actorFromRequest(r)
	req := approvalsmod.ApprovalRequest{
		CandidateID: candidateID,
		Decision:    approvalsmod.DecisionSnoozed,
		ApprovedBy:  actor,
		Notes:       body.Notes,
		SnoozeHours: body.SnoozeHours,
	}
	approval, err := svc.Decide(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	publishEvent("approval.snoozed", map[string]any{
		"candidateId": candidateID,
		"approvalId":  approval.ID,
		"snoozeUntil": approval.SnoozeUntil,
	})
	jsonOK(w, approval)
}

// actorFromRequest extracts the actor identity from JWT claims or falls back to
// the X-User-ID header, then to "anonymous".
func actorFromRequest(r *http.Request) string {
	if id := r.Header.Get("X-User-ID"); id != "" {
		return id
	}
	return "anonymous"
}
