package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/workflow"
	"jax-trading-assistant/libs/runtimepolicy"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerExploratoryPaperRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	store := exploratorypaper.NewPostgresStore(pool)
	mux.HandleFunc("/api/v1/exploratory-paper/positions", protect(exploratoryPaperPositionsHandler(store)))
	mux.HandleFunc("/api/v1/exploratory-paper/positions/", protect(exploratoryPaperPositionHandler(store)))
	mux.HandleFunc("/api/v1/exploratory-paper/entry-queue", protect(exploratoryPaperEntryQueueHandler(store)))
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
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		positionID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/exploratory-paper/positions/"), "/")
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
