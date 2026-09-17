package main

import (
	"errors"
	"net/http"
	"strings"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/workflow"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerExploratoryPaperRoutes(mux *http.ServeMux, protect func(http.HandlerFunc) http.HandlerFunc, pool *pgxpool.Pool) {
	store := exploratorypaper.NewPostgresStore(pool)
	mux.HandleFunc("/api/v1/exploratory-paper/positions", protect(exploratoryPaperPositionsHandler(store)))
	mux.HandleFunc("/api/v1/exploratory-paper/positions/", protect(exploratoryPaperPositionHandler(store)))
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
	return exploratoryPaperReadModel{LifecycleID: record.LifecycleID, CandidateID: record.CandidateID, Mode: exploratorypaper.ExploratoryPaperMode, FormalEvidence: false, NotFormalEvidence: true, Thesis: record.Thesis, WorkflowID: record.Binding.WorkflowID, PaperIntentID: record.Binding.PaperIntentID, EntryOrderID: record.EntryOrder.OrderID, EntryFillID: record.EntryFill.FillID, Position: record.Position, Reviews: record.Reviews, Checkpoints: record.Checkpoints, Outcome: record.Outcome, WorkflowState: string(record.Workflow.State), ApprovalConfirmed: record.Workflow.Confirmation != nil && record.Workflow.Confirmation.Decision == workflow.ConfirmationApprove}
}
