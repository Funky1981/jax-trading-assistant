package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/workflow"
	"jax-trading-assistant/libs/auth"
	"jax-trading-assistant/libs/runtimepolicy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerExploratoryPaperRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	operationalStore, operationalErr := newExploratoryOperationalStore(pool)
	historicalReadOnlyStore := exploratorypaper.NewPostgresStore(pool)
	pilotStore := exploratorypaper.NewPilotPostgresStore(pool)
	if operationalErr != nil {
		unavailableHandler := unavailableExploratoryOperationalHandler(operationalErr)
		mux.HandleFunc("/api/v1/exploratory-paper/positions", protect(unavailableHandler))
		mux.HandleFunc("/api/v1/exploratory-paper/positions/", protect(unavailableHandler))
		mux.HandleFunc("/api/v1/exploratory-paper/entry-queue", protect(unavailableHandler))
	} else {
		mux.HandleFunc("/api/v1/exploratory-paper/positions", protect(exploratoryPaperPositionsHandler(operationalStore)))
		mux.HandleFunc("/api/v1/exploratory-paper/positions/", protect(exploratoryPaperPositionHandler(operationalStore)))
		mux.HandleFunc("/api/v1/exploratory-paper/entry-queue", protect(exploratoryPaperEntryQueueHandler(operationalStore)))
	}
	mux.HandleFunc("/api/v1/exploratory-paper/pilot", protect(exploratoryPilotHandler(pilotStore, historicalReadOnlyStore)))
	mux.HandleFunc("/api/v1/exploratory-paper/pilot/opportunities", protect(exploratoryPilotOpportunitiesHandler(pilotStore)))
	mux.HandleFunc("/api/v1/exploratory-paper/pilot-readiness", protect(exploratoryPilotReadinessHandler))
}

func newExploratoryOperationalStore(pool *pgxpool.Pool) (*exploratorypaper.PostgresStore, error) {
	accountID := strings.TrimSpace(os.Getenv("PAPER_ACCOUNT_ID"))
	if accountID == "" {
		return nil, errors.New("exploratory paper operational routes require PAPER_ACCOUNT_ID")
	}
	return exploratorypaper.NewPostgresStoreForAccount(pool, accountID), nil
}

func unavailableExploratoryOperationalHandler(err error) http.HandlerFunc {
	message := "exploratory paper operational routes unavailable"
	if err != nil {
		message += ": " + err.Error()
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, message, http.StatusServiceUnavailable)
	}
}

func exploratoryPilotReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonOK(w, map[string]any{
		"mode":              exploratorypaper.PilotMode,
		"formalEvidence":    false,
		"notFormalEvidence": true,
		"readiness":         loadPaper02RuntimeReadiness(),
	})
}

// exploratoryPaperEntryQueueHandler is a handoff only: it accepts an entry
// after the existing human-approved workflow has reached PAPER_INTENT_CREATED.
// It never approves, creates a broker order, or executes anything.
func exploratoryPaperEntryQueueHandler(store *exploratorypaper.PostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if runtimepolicy.CurrentMode() != runtimepolicy.ModePaper {
			http.Error(w, "exploratory paper entry queue requires PAPER runtime mode", http.StatusConflict)
			return
		}
		var entry exploratorypaper.EntryRequest
		if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
			http.Error(w, "invalid approved entry payload: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.QueueApprovedEntry(r.Context(), entry); err != nil {
			http.Error(w, "approved entry was rejected: "+err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, map[string]any{"queued": true, "candidateId": entry.CandidateID, "mode": exploratorypaper.ExploratoryPaperMode, "execution": "ISOLATED_SIMULATED_PAPER_ONLY"})
	}
}

// The PAPER-02 surface is read-only in this package. There is deliberately no
// activation endpoint; external review must perform that handoff separately.
func exploratoryPilotHandler(pilotStore *exploratorypaper.PilotPostgresStore, lifecycleStore *exploratorypaper.PostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pilotID := strings.TrimSpace(r.URL.Query().Get("pilotId"))
		if pilotID == "" {
			http.Error(w, "pilotId is required", http.StatusBadRequest)
			return
		}
		model, err := pilotStore.ReadModel(r.Context(), pilotID)
		if err != nil {
			http.Error(w, "PAPER-02 read model unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		active, err := lifecycleStore.ListActive(r.Context(), 200)
		if err != nil {
			http.Error(w, "exploratory lifecycle read model unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		model.ActivePositions = active
		jsonOK(w, model)
	}
}

func exploratoryPilotOpportunitiesHandler(store *exploratorypaper.PilotPostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		pilotID := strings.TrimSpace(r.URL.Query().Get("pilotId"))
		if pilotID == "" {
			http.Error(w, "pilotId is required", http.StatusBadRequest)
			return
		}
		opportunities, err := store.ListOpportunities(r.Context(), pilotID)
		if err != nil {
			http.Error(w, "PAPER-02 opportunities unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		jsonOK(w, map[string]any{"mode": exploratorypaper.PilotMode, "formalEvidence": false, "notFormalEvidence": true, "opportunities": opportunities})
	}
}

func exploratoryPaperPositionsHandler(store *exploratorypaper.PostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		records, err := store.ListActive(r.Context(), 50)
		if err != nil {
			http.Error(w, "exploratory paper read model unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		jsonOK(w, exploratoryPaperReadModels(records))
	}
}

func exploratoryPaperPositionHandler(store *exploratorypaper.PostgresStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tail := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/exploratory-paper/positions/"), "/")
		parts := strings.Split(tail, "/")
		if len(parts) == 2 && (parts[1] == "exit-approval" || parts[1] == "exit-rejection") {
			handleExploratoryExitDecision(w, r, store, parts[0], parts[1])
			return
		}
		if r.Method != http.MethodGet || len(parts) != 1 {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionID := parts[0]
		if positionID == "" {
			http.Error(w, "position ID is required", http.StatusBadRequest)
			return
		}
		record, err := store.Get(r.Context(), positionID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "exploratory paper record unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		jsonOK(w, exploratoryPaperReadModelFromRecord(record))
	}
}

type exploratoryExitDecisionStore interface {
	PersistExitDecision(context.Context, string, exploratorypaper.ApprovalSnapshot) error
	Get(context.Context, string) (exploratorypaper.LifecycleRecord, error)
}

// handleExploratoryExitDecision is the existing protected operator/API handoff
// for the durable workflow approval architecture. It accepts a complete
// workflow snapshot, not an approval boolean, and the store validates every
// identity against the requested lifecycle review before persisting it.
func handleExploratoryExitDecision(w http.ResponseWriter, r *http.Request, store exploratoryExitDecisionStore, positionID, action string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if runtimepolicy.CurrentMode() != runtimepolicy.ModePaper {
		http.Error(w, "exploratory paper exit approval requires PAPER runtime mode", http.StatusConflict)
		return
	}
	actor, authenticated := authenticatedExploratoryExitActor(r)
	if !authenticated {
		http.Error(w, "explicit human operator identity is required", http.StatusForbidden)
		return
	}
	var approval exploratorypaper.ApprovalSnapshot
	if err := json.NewDecoder(r.Body).Decode(&approval); err != nil {
		http.Error(w, "invalid exit workflow approval payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if approval.Workflow.Confirmation == nil || approval.Workflow.Confirmation.Actor != actor {
		http.Error(w, "exit decision actor does not match the authenticated operator", http.StatusForbidden)
		return
	}
	wantState := workflow.StatePaperIntentCreated
	if action == "exit-rejection" {
		wantState = workflow.StateHumanRejected
	}
	if approval.Workflow.State != wantState {
		http.Error(w, "exit decision state does not match the requested action", http.StatusBadRequest)
		return
	}
	if err := store.PersistExitDecision(r.Context(), positionID, approval); err != nil {
		http.Error(w, "exit decision was rejected: "+err.Error(), http.StatusConflict)
		return
	}
	record, err := store.Get(r.Context(), positionID)
	if err != nil {
		http.Error(w, "exit decision persisted but lifecycle could not be reloaded: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	jsonOK(w, exploratoryPaperReadModelFromRecord(record))
}

func authenticatedExploratoryExitActor(r *http.Request) (string, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		return "", false
	}
	if userID, ok := auth.UserIDFromContext(r.Context()); ok && strings.TrimSpace(userID) != "" {
		return strings.TrimSpace(userID), true
	}
	if username, ok := auth.UsernameFromContext(r.Context()); ok && strings.TrimSpace(username) != "" {
		return strings.TrimSpace(username), true
	}
	return "", false
}

type exploratoryPaperReadModel struct {
	LifecycleID       string                        `json:"lifecycleId"`
	CandidateID       string                        `json:"candidateId"`
	Mode              string                        `json:"mode"`
	FormalEvidence    bool                          `json:"formalEvidence"`
	NotFormalEvidence bool                          `json:"notFormalEvidence"`
	Thesis            exploratorypaper.FrozenThesis `json:"frozenThesis"`
	WorkflowID        string                        `json:"workflowId"`
	PaperIntentID     string                        `json:"paperIntentId"`
	EntryOrderID      string                        `json:"entryOrderId"`
	EntryFillID       string                        `json:"entryFillId"`
	ExitReview        *exploratorypaper.Review      `json:"exitReview,omitempty"`
	ExitApprovalState string                        `json:"exitApprovalState"`
	ExitOrderID       string                        `json:"exitOrderId,omitempty"`
	ExitFillID        string                        `json:"exitFillId,omitempty"`
	Position          exploratorypaper.Position     `json:"position"`
	Reviews           []exploratorypaper.Review     `json:"reviews"`
	Checkpoints       []exploratorypaper.Checkpoint `json:"checkpoints"`
	Outcome           *exploratorypaper.Outcome     `json:"outcome,omitempty"`
	WorkflowState     string                        `json:"workflowState"`
	ApprovalConfirmed bool                          `json:"approvalConfirmed"`
}

func exploratoryPaperReadModels(records []exploratorypaper.LifecycleRecord) []exploratoryPaperReadModel {
	result := make([]exploratoryPaperReadModel, 0, len(records))
	for _, record := range records {
		result = append(result, exploratoryPaperReadModelFromRecord(record))
	}
	return result
}

func exploratoryPaperReadModelFromRecord(record exploratorypaper.LifecycleRecord) exploratoryPaperReadModel {
	model := exploratoryPaperReadModel{LifecycleID: record.LifecycleID, CandidateID: record.CandidateID, Mode: exploratorypaper.ExploratoryPaperMode, FormalEvidence: false, NotFormalEvidence: true, Thesis: record.Thesis, WorkflowID: record.Binding.WorkflowID, PaperIntentID: record.Binding.PaperIntentID, EntryOrderID: record.EntryOrder.OrderID, EntryFillID: record.EntryFill.FillID, Position: record.Position, Reviews: record.Reviews, Checkpoints: record.Checkpoints, Outcome: record.Outcome, WorkflowState: string(record.Workflow.State), ApprovalConfirmed: record.Workflow.Confirmation != nil && record.Workflow.Confirmation.Decision == workflow.ConfirmationApprove, ExitApprovalState: "NOT_REQUESTED"}
	for index := range record.Reviews {
		review := &record.Reviews[index]
		if review.Status == "EXIT_RECOMMENDED" || review.ExitReason != "" || review.ExitApprovalWorkflowID != "" {
			model.ExitReview = review
			model.ExitApprovalState = "PENDING_HUMAN_APPROVAL"
			break
		}
	}
	if record.Outcome != nil {
		model.ExitOrderID = record.Outcome.ExitOrderID
		model.ExitFillID = record.Outcome.ExitFillID
		model.ExitApprovalState = "APPROVED_AND_CLOSED"
	}
	return model
}
