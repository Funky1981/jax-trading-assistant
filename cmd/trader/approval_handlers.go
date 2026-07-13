package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
	candidatesmod "jax-trading-assistant/internal/modules/candidates"
)

// registerApprovalRoutes registers all human-approval-flow endpoints on mux.
func registerApprovalRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	svc := approvalsmod.NewService(pool)

	// POST /api/v1/mobile/telegram/webhook
	mux.HandleFunc("/api/v1/mobile/telegram/webhook", mobileTelegramWebhookHandler(svc))

	// GET  /api/v1/approvals/queue
	mux.HandleFunc("/api/v1/approvals/queue", protect(approvalQueueHandler(svc)))

	// GET  /api/v1/paper-tickets
	// POST /api/v1/paper-tickets/{paperTicketId}/mark-reviewed
	// POST /api/v1/paper-tickets/{paperTicketId}/cancel
	// POST /api/v1/paper-tickets/{paperTicketId}/notes
	mux.HandleFunc("/api/v1/paper-tickets", protect(paperTicketQueueHandler(svc)))
	mux.HandleFunc("/api/v1/paper-tickets/", protect(paperTicketReviewRouter(svc)))

	// GET  /api/v1/approvals/{candidateId}
	// POST /api/v1/approvals/{candidateId}/approve
	// POST /api/v1/approvals/{candidateId}/reject
	// POST /api/v1/approvals/{candidateId}/snooze
	// POST /api/v1/approvals/{candidateId}/reanalyze
	mux.HandleFunc("/api/v1/approvals/", protect(approvalDetailRouter(svc)))
}

type mobileTelegramWebhookBody struct {
	Token         string `json:"token"`
	Action        string `json:"action"`
	Actor         string `json:"actor"`
	Reason        string `json:"reason"`
	GuardrailHash string `json:"guardrailHash"`
	RuntimeMode   string `json:"runtimeMode"`
	CallbackQuery *struct {
		Data string `json:"data"`
		From *struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"callback_query"`
}

func (b mobileTelegramWebhookBody) mobileDecisionRequest(now time.Time) approvalsmod.MobileApprovalDecisionRequest {
	token := strings.TrimSpace(b.Token)
	action := strings.TrimSpace(b.Action)
	actor := strings.TrimSpace(b.Actor)
	if b.CallbackQuery != nil {
		parts := strings.SplitN(strings.TrimSpace(b.CallbackQuery.Data), ":", 2)
		if len(parts) == 2 {
			action = parts[0]
			token = parts[1]
		}
		if actor == "" && b.CallbackQuery.From != nil {
			actor = strings.TrimSpace(b.CallbackQuery.From.Username)
			if actor == "" {
				actor = strconv.FormatInt(b.CallbackQuery.From.ID, 10)
			}
		}
	}
	if b.RuntimeMode == "" {
		b.RuntimeMode = "paper"
	}
	if actor == "" {
		actor = "telegram"
	}
	return approvalsmod.MobileApprovalDecisionRequest{
		Token:         token,
		Decision:      action,
		Actor:         actor,
		Channel:       "telegram",
		GuardrailHash: b.GuardrailHash,
		RejectReason:  b.Reason,
		RuntimeMode:   b.RuntimeMode,
		Now:           now,
	}
}

func mobileTelegramWebhookHandler(svc *approvalsmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body mobileTelegramWebhookBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid telegram approval payload", http.StatusBadRequest)
			return
		}
		approval, err := svc.SubmitMobileDecision(r.Context(), body.mobileDecisionRequest(time.Now().UTC()))
		if err != nil {
			writeMobileApprovalError(w, err)
			return
		}
		publishEvent("approval.mobile."+approval.Decision, map[string]any{
			"candidateId": approval.CandidateID,
			"approvalId":  approval.ID,
			"decision":    approval.Decision,
			"approvedBy":  approval.ApprovedBy,
			"decidedAt":   approval.DecidedAt,
			"channel":     "telegram",
		})
		jsonOK(w, map[string]any{
			"approvalId":  approval.ID,
			"candidateId": approval.CandidateID,
			"decision":    approval.Decision,
			"runtimeMode": "paper",
		})
	}
}

func writeMobileApprovalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, approvalsmod.ErrMobileApprovalExpired), errors.Is(err, approvalsmod.ErrCandidateExpired):
		http.Error(w, err.Error(), http.StatusGone)
	case errors.Is(err, approvalsmod.ErrMobileApprovalUsed), errors.Is(err, approvalsmod.ErrNotAwaitingApproval):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, approvalsmod.ErrMobileApprovalInvalid),
		errors.Is(err, approvalsmod.ErrMobileApprovalGuardrailChanged),
		errors.Is(err, approvalsmod.ErrMobileApprovalLiveMode),
		errors.Is(err, approvalsmod.ErrMobileApprovalDecisionInvalid),
		errors.Is(err, approvalsmod.ErrInstrumentPolicy):
		http.Error(w, err.Error(), http.StatusForbidden)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func paperTicketQueueHandler(svc *approvalsmod.Service) http.HandlerFunc {
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
		queue, err := svc.GetPaperTicketQueue(r.Context(), limit)
		if err != nil {
			http.Error(w, fmt.Sprintf("paper ticket queue: %v", err), http.StatusInternalServerError)
			return
		}
		if queue == nil {
			queue = []candidatesmod.PaperTicketReview{}
		}
		jsonOK(w, queue)
	}
}

func paperTicketReviewRouter(svc *approvalsmod.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tail := strings.TrimPrefix(r.URL.Path, "/api/v1/paper-tickets/")
		parts := strings.SplitN(tail, "/", 2)
		if len(parts) != 2 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		paperTicketID := parts[0]
		action := parts[1]
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body paperTicketReviewActionBody
		if r.ContentLength > 0 {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		var (
			review *candidatesmod.PaperTicketReview
			err    error
		)
		switch action {
		case "mark-reviewed":
			review, err = svc.MarkPaperTicketReviewed(r.Context(), paperTicketID, body.Note)
		case "cancel":
			review, err = svc.CancelPaperTicketReview(r.Context(), paperTicketID, body.Note)
		case "notes":
			review, err = svc.AddPaperTicketReviewNote(r.Context(), paperTicketID, body.Note)
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		jsonOK(w, review)
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
		if strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "no rows") {
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
			if errors.Is(err, approvalsmod.ErrInstrumentPolicy) {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
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
	detail, err := svc.GetByCandidate(r.Context(), candidateID)
	if err != nil {
		jsonOK(w, approval)
		return
	}
	jsonOK(w, detail)
}

type snoozeBody struct {
	Notes       *string `json:"notes"`
	SnoozeHours int     `json:"snoozeHours"`
}

type paperTicketReviewActionBody struct {
	Note string `json:"note"`
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
	detail, err := svc.GetByCandidate(r.Context(), candidateID)
	if err != nil {
		jsonOK(w, approval)
		return
	}
	jsonOK(w, detail)
}

// actorFromRequest extracts the actor identity from JWT claims or falls back to
// the X-User-ID header, then to "anonymous".
func actorFromRequest(r *http.Request) string {
	if id := r.Header.Get("X-User-ID"); id != "" {
		return id
	}
	return "anonymous"
}
