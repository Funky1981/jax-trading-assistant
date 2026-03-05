package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"jax-trading-assistant/internal/domain/artifacts"

	"github.com/google/uuid"
)

type artifactStore interface {
	ListApprovedArtifacts(ctx context.Context) ([]*artifacts.Artifact, error)
	ListArtifacts(ctx context.Context, stateFilter string) ([]*artifacts.Artifact, error)
	GetApproval(ctx context.Context, artifactID uuid.UUID) (*artifacts.Approval, error)
	GetApprovals(ctx context.Context, artifactIDs []uuid.UUID) (map[uuid.UUID]*artifacts.Approval, error)
	GetArtifactByID(ctx context.Context, id uuid.UUID) (*artifacts.Artifact, error)
	CreateArtifact(ctx context.Context, artifact *artifacts.Artifact) error
	CreateApproval(ctx context.Context, approval *artifacts.Approval) error
	UpdateApprovalState(ctx context.Context, artifactID uuid.UUID, toState artifacts.ApprovalState, promotedBy, reason string) error
	CreateValidationReport(ctx context.Context, report *artifacts.ValidationReport) error
	RecordValidationOutcome(ctx context.Context, artifactID, runID uuid.UUID, passed bool, reportURI string) error
}

// ArtifactHandlers provides HTTP handlers for artifact management
type ArtifactHandlers struct {
	store             artifactStore
	now               func() time.Time
	runReplayGate     func(ctx context.Context) map[string]any
	runPromotionGate  func(ctx context.Context) map[string]any
	defaultReplayGate string
	defaultReplayType string
	defaultGate       string
	defaultType       string
}

// NewArtifactHandlers creates artifact API handlers
func NewArtifactHandlers(store artifactStore) *ArtifactHandlers {
	return &ArtifactHandlers{
		store:             store,
		now:               time.Now,
		defaultReplayGate: "Gate2",
		defaultReplayType: "deterministic_replay",
		defaultGate:       "Gate3",
		defaultType:       "artifact_promotion",
	}
}

// RegisterRoutes adds artifact endpoints to the HTTP server
func (h *ArtifactHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/artifacts", h.handleArtifacts)
	mux.HandleFunc("/api/v1/artifacts/", h.handleArtifactByID)
}

// handleArtifacts routes to list or create artifacts
func (h *ArtifactHandlers) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.handleListArtifacts(w, r)
	case "POST":
		h.handleCreateArtifact(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleArtifactByID routes to get, promote, or validate artifact
func (h *ArtifactHandlers) handleArtifactByID(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path: /api/v1/artifacts/{id}[/action]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/artifacts/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "artifact ID required", http.StatusBadRequest)
		return
	}

	artifactID := parts[0]

	// Check for action (promote or validate)
	if len(parts) > 1 {
		action := parts[1]
		switch action {
		case "promote":
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.handlePromoteArtifact(w, r, artifactID)
		case "validate":
			if r.Method != "POST" {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.handleValidateArtifact(w, r, artifactID)
		default:
			http.Error(w, "unknown action", http.StatusNotFound)
		}
		return
	}

	// No action - GET artifact details
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.handleGetArtifact(w, r, artifactID)
}

// CreateArtifactRequest represents artifact creation payload
type CreateArtifactRequest struct {
	StrategyName    string                 `json:"strategy_name"`
	StrategyVersion string                 `json:"strategy_version"`
	Params          map[string]interface{} `json:"params"`
	RiskProfile     RiskProfileRequest     `json:"risk_profile"`
	CreatedBy       string                 `json:"created_by"`
}

type RiskProfileRequest struct {
	MaxPositionPct    float64  `json:"max_position_pct"`
	MaxDailyLoss      float64  `json:"max_daily_loss"`
	AllowedOrderTypes []string `json:"allowed_order_types"`
}

// PromoteArtifactRequest represents state transition payload
type PromoteArtifactRequest struct {
	ToState    string `json:"to_state"`
	PromotedBy string `json:"promoted_by"`
	Reason     string `json:"reason"`
}

// ArtifactResponse represents artifact details in API responses
type ArtifactResponse struct {
	ID           string                 `json:"id"`
	ArtifactID   string                 `json:"artifact_id"`
	StrategyName string                 `json:"strategy_name"`
	Version      string                 `json:"version"`
	Params       map[string]interface{} `json:"params"`
	Hash         string                 `json:"hash"`
	State        string                 `json:"state"`
	CreatedBy    string                 `json:"created_by"`
	CreatedAt    string                 `json:"created_at"`
	ApprovedBy   *string                `json:"approved_by,omitempty"`
	ApprovedAt   *string                `json:"approved_at,omitempty"`
}

// handleListArtifacts returns all artifacts (optionally filtered by state)
func (h *ArtifactHandlers) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check for state filter
	stateFilter := r.URL.Query().Get("state")

	var artifactList []*artifacts.Artifact
	var err error

	if stateFilter == "approved" {
		artifactList, err = h.store.ListApprovedArtifacts(ctx)
	} else {
		artifactList, err = h.store.ListArtifacts(ctx, stateFilter)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list artifacts: %v", err), http.StatusInternalServerError)
		return
	}
	artifactIDs := make([]uuid.UUID, 0, len(artifactList))
	for _, artifact := range artifactList {
		artifactIDs = append(artifactIDs, artifact.ID)
	}
	approvalsByArtifact, err := h.store.GetApprovals(ctx, artifactIDs)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load artifact approvals: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to API response format
	response := make([]ArtifactResponse, 0, len(artifactList))
	for _, artifact := range artifactList {
		approval := approvalsByArtifact[artifact.ID]

		resp := ArtifactResponse{
			ID:           artifact.ID.String(),
			ArtifactID:   artifact.ArtifactID,
			StrategyName: artifact.Strategy.Name,
			Version:      artifact.Strategy.Version,
			Params:       artifact.Strategy.Params,
			Hash:         artifact.Hash,
			CreatedBy:    artifact.CreatedBy,
			CreatedAt:    artifact.CreatedAt.Format(time.RFC3339),
		}

		if approval != nil {
			stateStr := string(approval.State)
			resp.State = stateStr
			if approval.ApprovedBy != "" {
				resp.ApprovedBy = &approval.ApprovedBy
			}
			if approval.ApprovedAt != nil && !approval.ApprovedAt.IsZero() {
				approvedAt := approval.ApprovedAt.Format(time.RFC3339)
				resp.ApprovedAt = &approvedAt
			}
		}

		response = append(response, resp)
	}

	jsonOK(w, response)
}

// handleGetArtifact returns a single artifact by ID
func (h *ArtifactHandlers) handleGetArtifact(w http.ResponseWriter, r *http.Request, artifactIDStr string) {
	ctx := r.Context()

	artifactID, err := uuid.Parse(artifactIDStr)
	if err != nil {
		http.Error(w, "invalid artifact ID", http.StatusBadRequest)
		return
	}

	artifact, err := h.store.GetArtifactByID(ctx, artifactID)
	if err != nil {
		http.Error(w, fmt.Sprintf("artifact not found: %v", err), http.StatusNotFound)
		return
	}

	// Get approval info
	approval, _ := h.store.GetApproval(ctx, artifact.ID)

	resp := ArtifactResponse{
		ID:           artifact.ID.String(),
		ArtifactID:   artifact.ArtifactID,
		StrategyName: artifact.Strategy.Name,
		Version:      artifact.Strategy.Version,
		Params:       artifact.Strategy.Params,
		Hash:         artifact.Hash,
		CreatedBy:    artifact.CreatedBy,
		CreatedAt:    artifact.CreatedAt.Format(time.RFC3339),
	}

	if approval != nil {
		resp.State = string(approval.State)
		if approval.ApprovedBy != "" {
			resp.ApprovedBy = &approval.ApprovedBy
		}
		if approval.ApprovedAt != nil && !approval.ApprovedAt.IsZero() {
			approvedAt := approval.ApprovedAt.Format(time.RFC3339)
			resp.ApprovedAt = &approvedAt
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("handleGetArtifact encode: %v", err)
	}
}

// handleCreateArtifact creates a new artifact in DRAFT state
func (h *ArtifactHandlers) handleCreateArtifact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.StrategyName == "" {
		http.Error(w, "strategy_name is required", http.StatusBadRequest)
		return
	}
	if req.StrategyVersion == "" {
		http.Error(w, "strategy_version is required", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		http.Error(w, "created_by is required", http.StatusBadRequest)
		return
	}

	// Convert request to domain model
	riskProfile := artifacts.RiskProfile{
		MaxPositionPct:    req.RiskProfile.MaxPositionPct,
		MaxDailyLoss:      req.RiskProfile.MaxDailyLoss,
		AllowedOrderTypes: req.RiskProfile.AllowedOrderTypes,
	}

	// Create artifact
	artifact, err := artifacts.NewArtifact(
		req.StrategyName,
		req.StrategyVersion,
		req.Params,
		riskProfile,
		req.CreatedBy,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create artifact: %v", err), http.StatusInternalServerError)
		return
	}

	// Store artifact in database
	if err := h.store.CreateArtifact(ctx, artifact); err != nil {
		http.Error(w, fmt.Sprintf("failed to store artifact: %v", err), http.StatusInternalServerError)
		return
	}

	// Create approval record in DRAFT state
	approval := artifacts.NewApproval(artifact.ID, req.CreatedBy)
	if err := h.store.CreateApproval(ctx, approval); err != nil {
		log.Printf("Warning: failed to create approval record: %v", err)
	}

	log.Printf("Created artifact %s (hash: %s...)", artifact.ArtifactID, artifact.Hash[:8])

	// Return created artifact
	resp := ArtifactResponse{
		ID:           artifact.ID.String(),
		ArtifactID:   artifact.ArtifactID,
		StrategyName: artifact.Strategy.Name,
		Version:      artifact.Strategy.Version,
		Params:       artifact.Strategy.Params,
		Hash:         artifact.Hash,
		State:        string(artifacts.StateDraft),
		CreatedBy:    artifact.CreatedBy,
		CreatedAt:    artifact.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("handleCreateArtifact encode: %v", err)
	}
}

// handlePromoteArtifact transitions an artifact to a new state
func (h *ArtifactHandlers) handlePromoteArtifact(w http.ResponseWriter, r *http.Request, artifactIDStr string) {
	ctx := r.Context()

	artifactID, err := uuid.Parse(artifactIDStr)
	if err != nil {
		http.Error(w, "invalid artifact ID", http.StatusBadRequest)
		return
	}

	var req PromoteArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ToState == "" {
		http.Error(w, "to_state is required", http.StatusBadRequest)
		return
	}
	if req.PromotedBy == "" {
		http.Error(w, "promoted_by is required", http.StatusBadRequest)
		return
	}

	// Convert string to ApprovalState
	toState := artifacts.ApprovalState(req.ToState)

	// Validate state is valid
	validStates := []artifacts.ApprovalState{
		artifacts.StateDraft,
		artifacts.StateValidated,
		artifacts.StateReviewed,
		artifacts.StateApproved,
		artifacts.StateActive,
		artifacts.StateDeprecated,
		artifacts.StateRevoked,
	}
	valid := false
	for _, s := range validStates {
		if toState == s {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, fmt.Sprintf("invalid state: %s", req.ToState), http.StatusBadRequest)
		return
	}

	// Perform state transition (with validation)
	if err := h.store.UpdateApprovalState(ctx, artifactID, toState, req.PromotedBy, req.Reason); err != nil {
		http.Error(w, fmt.Sprintf("failed to promote artifact: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Promoted artifact %s to %s by %s", artifactID, toState, req.PromotedBy)

	// Return updated artifact
	artifact, err := h.store.GetArtifactByID(ctx, artifactID)
	if err != nil {
		http.Error(w, fmt.Sprintf("artifact not found after promotion: %v", err), http.StatusInternalServerError)
		return
	}

	approval, _ := h.store.GetApproval(ctx, artifactID)

	resp := ArtifactResponse{
		ID:           artifact.ID.String(),
		ArtifactID:   artifact.ArtifactID,
		StrategyName: artifact.Strategy.Name,
		Version:      artifact.Strategy.Version,
		Params:       artifact.Strategy.Params,
		Hash:         artifact.Hash,
		State:        string(toState),
		CreatedBy:    artifact.CreatedBy,
		CreatedAt:    artifact.CreatedAt.Format(time.RFC3339),
	}

	if approval != nil && approval.ApprovedBy != "" {
		resp.ApprovedBy = &approval.ApprovedBy
	}
	if approval != nil && approval.ApprovedAt != nil && !approval.ApprovedAt.IsZero() {
		approvedAt := approval.ApprovedAt.Format(time.RFC3339)
		resp.ApprovedAt = &approvedAt
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("artifact handler encode: %v", err)
	}
}

// handleValidateArtifact runs validation tests on an artifact
func (h *ArtifactHandlers) handleValidateArtifact(w http.ResponseWriter, r *http.Request, artifactIDStr string) {
	ctx := r.Context()

	artifactID, err := uuid.Parse(artifactIDStr)
	if err != nil {
		http.Error(w, "invalid artifact ID", http.StatusBadRequest)
		return
	}

	// Get artifact
	artifact, err := h.store.GetArtifactByID(ctx, artifactID)
	if err != nil {
		http.Error(w, fmt.Sprintf("artifact not found: %v", err), http.StatusNotFound)
		return
	}

	approval, err := h.store.GetApproval(ctx, artifactID)
	if err != nil {
		http.Error(w, fmt.Sprintf("approval not found: %v", err), http.StatusNotFound)
		return
	}

	report := buildArtifactValidationReport(artifact, h.now().UTC())
	var replayRun map[string]any
	if h.runReplayGate != nil {
		replayRun = h.runReplayGate(ctx)
		report.TestEnvironment["replayGate"] = gateEvidence(h.defaultReplayGate, h.defaultReplayType, replayRun)
		if !isGateResultPassed(replayRun) {
			report.Passed = false
			report.Errors = append(report.Errors, "Gate2 deterministic replay trust gate failed")
		}
	} else {
		report.Passed = false
		report.Errors = append(report.Errors, "Gate2 deterministic replay gate is unavailable")
	}

	var gateRun map[string]any
	if h.runPromotionGate != nil {
		gateRun = h.runPromotionGate(ctx)
		report.TestEnvironment["gate"] = gateEvidence(h.defaultGate, h.defaultType, gateRun)
		if !isGateResultPassed(gateRun) {
			report.Passed = false
			report.Errors = append(report.Errors, "Gate3 artifact promotion trust gate failed")
		}
	} else {
		report.Passed = false
		report.Errors = append(report.Errors, "Gate3 artifact promotion gate is unavailable")
	}

	if runID := bestGateRunID(gateRun, replayRun); runID != uuid.Nil {
		report.RunID = runID
	}
	reportURI := bestGateArtifactURI(gateRun, replayRun, report.ReportURI)
	if reportURI != "" {
		report.ReportURI = reportURI
	}
	if err := h.store.CreateValidationReport(ctx, report); err != nil {
		http.Error(w, fmt.Sprintf("failed to store validation report: %v", err), http.StatusInternalServerError)
		return
	}
	if err := h.store.RecordValidationOutcome(ctx, artifactID, report.RunID, report.Passed, reportURI); err != nil {
		http.Error(w, fmt.Sprintf("failed to record validation outcome: %v", err), http.StatusInternalServerError)
		return
	}

	newState := string(approval.State)
	if report.Passed && approval.State == artifacts.StateDraft {
		if err := h.store.UpdateApprovalState(ctx, artifactID, artifacts.StateValidated, "system", "artifact validation passed with Gate2/Gate3 evidence"); err != nil {
			http.Error(w, fmt.Sprintf("failed to promote artifact: %v", err), http.StatusInternalServerError)
			return
		}
		newState = string(artifacts.StateValidated)
	}

	log.Printf("Validated artifact %s: passed=%v", artifact.ArtifactID, report.Passed)

	// Return validation report
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"artifact_id":       artifact.ArtifactID,
		"validation_run_id": report.RunID.String(),
		"passed":            report.Passed,
		"metrics":           report.Metrics,
		"errors":            report.Errors,
		"warnings":          report.Warnings,
		"tested_at":         report.CompletedAt.Format(time.RFC3339),
		"new_state":         newState,
		"replayRun":         replayRun,
		"gateRun":           gateRun,
	}); err != nil {
		log.Printf("handleValidateArtifact encode: %v", err)
	}
}

func gateEvidence(gate, testType string, run map[string]any) map[string]any {
	evidence := map[string]any{
		"name":     gate,
		"testType": testType,
	}
	if run == nil {
		evidence["status"] = "missing"
		return evidence
	}
	summary, _ := run["summary"].(map[string]any)
	evidence["testRunId"] = toString(run["testRunId"])
	evidence["artifactUri"] = toString(run["artifactUri"])
	evidence["status"] = toString(run["status"])
	evidence["summaryState"] = toString(summary["status"])
	return evidence
}

func isGateResultPassed(run map[string]any) bool {
	if run == nil {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(toString(run["status"])))
	if status != "completed" {
		return false
	}
	summary, ok := run["summary"].(map[string]any)
	if !ok {
		return false
	}
	return strings.EqualFold(toString(summary["status"]), "passed")
}

func bestGateRunID(primary, secondary map[string]any) uuid.UUID {
	candidates := []map[string]any{primary, secondary}
	for _, run := range candidates {
		if run == nil {
			continue
		}
		raw := strings.TrimSpace(toString(run["testRunId"]))
		if raw == "" {
			continue
		}
		if parsed, err := uuid.Parse(raw); err == nil {
			return parsed
		}
	}
	return uuid.Nil
}

func bestGateArtifactURI(primary, secondary map[string]any, fallback string) string {
	for _, run := range []map[string]any{primary, secondary} {
		if run == nil {
			continue
		}
		if uri := strings.TrimSpace(toString(run["artifactUri"])); uri != "" {
			return uri
		}
	}
	return strings.TrimSpace(fallback)
}

func buildArtifactValidationReport(artifact *artifacts.Artifact, now time.Time) *artifacts.ValidationReport {
	startedAt := now.UTC()
	errors := make([]string, 0, 8)
	warnings := make([]string, 0, 4)
	metrics := make(map[string]any)
	var determinismSeed *int
	reportURI := ""

	if err := artifact.VerifyHash(); err != nil {
		errors = append(errors, fmt.Sprintf("artifact hash verification failed: %v", err))
	}
	if strings.TrimSpace(artifact.Strategy.Name) == "" || strings.TrimSpace(artifact.Strategy.Version) == "" {
		errors = append(errors, "artifact strategy metadata is incomplete")
	}
	if strings.TrimSpace(artifact.CreatedBy) == "" {
		errors = append(errors, "artifact created_by is required")
	}
	if artifact.DataWindow == nil {
		errors = append(errors, "artifact validation requires a data window")
	} else {
		if artifact.DataWindow.To.Before(artifact.DataWindow.From) {
			errors = append(errors, "artifact data window is invalid")
		}
		if len(artifact.DataWindow.Symbols) == 0 {
			errors = append(errors, "artifact validation requires at least one symbol")
		}
	}
	if len(artifact.RiskProfile.AllowedOrderTypes) == 0 {
		errors = append(errors, "artifact risk profile must declare allowed order types")
	}
	if artifact.RiskProfile.MaxPositionPct <= 0 {
		errors = append(errors, "artifact risk profile must declare max_position_pct > 0")
	}
	if artifact.Validation == nil {
		errors = append(errors, "artifact validation payload is missing")
	} else {
		if artifact.Validation.BacktestRunID == uuid.Nil {
			errors = append(errors, "artifact validation is missing backtest_run_id")
		}
		if len(artifact.Validation.Metrics) == 0 {
			errors = append(errors, "artifact validation metrics are missing")
		}
		for key, value := range artifact.Validation.Metrics {
			metrics[key] = value
		}
		determinismSeed = &artifact.Validation.DeterminismSeed
		reportURI = artifact.Validation.ReportURI
	}

	requiredMetrics := []string{
		"total_trades",
		"win_rate",
		"total_return_pct",
		"max_drawdown",
		"sharpe_ratio",
		"profit_factor",
	}
	for _, key := range requiredMetrics {
		if _, ok := metrics[key]; !ok {
			warnings = append(warnings, fmt.Sprintf("validation metric %q is missing", key))
		}
	}
	if artifact.Validation != nil && artifact.Validation.DeterminismSeed == 0 {
		warnings = append(warnings, "determinism seed is zero; confirm this run was intentionally seeded")
	}

	completedAt := startedAt
	return &artifacts.ValidationReport{
		ID:              uuid.New(),
		ArtifactID:      artifact.ID,
		RunID:           uuid.New(),
		TestType:        "artifact_integrity",
		Passed:          len(errors) == 0,
		Metrics:         metrics,
		Errors:          errors,
		Warnings:        warnings,
		DeterminismSeed: determinismSeed,
		TestEnvironment: map[string]any{
			"validator":        "cmd/trader",
			"go_version":       runtime.Version(),
			"schema_version":   artifact.SchemaVersion,
			"strategy_name":    artifact.Strategy.Name,
			"strategy_version": artifact.Strategy.Version,
			"artifact_hash":    artifact.Hash,
		},
		ReportURI:       reportURI,
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		DurationSeconds: completedAt.Sub(startedAt).Seconds(),
		CreatedAt:       completedAt,
	}
}
