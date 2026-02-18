package artifacts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ApprovalState represents the artifact approval workflow state
type ApprovalState string

const (
	StateDraft      ApprovalState = "DRAFT"
	StateValidated  ApprovalState = "VALIDATED"
	StateReviewed   ApprovalState = "REVIEWED"
	StateApproved   ApprovalState = "APPROVED"
	StateActive     ApprovalState = "ACTIVE"
	StateDeprecated ApprovalState = "DEPRECATED"
	StateRevoked    ApprovalState = "REVOKED"
)

// ValidTransitions defines allowed state transitions
var ValidTransitions = map[ApprovalState][]ApprovalState{
	StateDraft:      {StateValidated, StateRevoked},
	StateValidated:  {StateReviewed, StateDraft, StateRevoked},
	StateReviewed:   {StateApproved, StateValidated, StateRevoked},
	StateApproved:   {StateActive, StateRevoked},
	StateActive:     {StateDeprecated, StateRevoked},
	StateDeprecated: {StateRevoked},
}

// Artifact represents an immutable strategy definition
type Artifact struct {
	ID            uuid.UUID       `json:"id"`
	ArtifactID    string          `json:"artifact_id"`
	SchemaVersion string          `json:"schema_version"`
	Strategy      StrategyInfo    `json:"strategy"`
	DataWindow    *DataWindow     `json:"data_window,omitempty"`
	Validation    *ValidationInfo `json:"validation,omitempty"`
	RiskProfile   RiskProfile     `json:"risk_profile"`
	Hash          string          `json:"hash"`
	Signature     string          `json:"signature,omitempty"`
	CreatedBy     string          `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

// StrategyInfo contains strategy identification
type StrategyInfo struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	CodeRef string         `json:"code_ref,omitempty"`
	Params  map[string]any `json:"params"`
}

// DataWindow defines the validation data range
type DataWindow struct {
	From    time.Time `json:"from"`
	To      time.Time `json:"to"`
	Symbols []string  `json:"symbols"`
}

// ValidationInfo contains backtest and validation metrics
type ValidationInfo struct {
	BacktestRunID   uuid.UUID      `json:"backtest_run_id"`
	Metrics         map[string]any `json:"metrics"`
	DeterminismSeed int            `json:"determinism_seed"`
	ReportURI       string         `json:"report_uri,omitempty"`
}

// RiskProfile defines risk management constraints
type RiskProfile struct {
	MaxPositionPct    float64  `json:"max_position_pct"`
	MaxDailyLoss      float64  `json:"max_daily_loss"`
	AllowedOrderTypes []string `json:"allowed_order_types"`
}

// Approval represents the approval workflow state
type Approval struct {
	ID                  uuid.UUID      `json:"id"`
	ArtifactID          uuid.UUID      `json:"artifact_id"`
	State               ApprovalState  `json:"state"`
	PreviousState       *ApprovalState `json:"previous_state,omitempty"`
	ApprovedBy          string         `json:"approved_by,omitempty"`
	ApprovedAt          *time.Time     `json:"approved_at,omitempty"`
	ValidationRunID     *uuid.UUID     `json:"validation_run_id,omitempty"`
	ValidationPassed    bool           `json:"validation_passed"`
	ValidationReportURI string         `json:"validation_report_uri,omitempty"`
	ReviewNotes         string         `json:"review_notes,omitempty"`
	Reviewer            string         `json:"reviewer,omitempty"`
	ReviewedAt          *time.Time     `json:"reviewed_at,omitempty"`
	StateChangedBy      string         `json:"state_changed_by"`
	StateChangedAt      time.Time      `json:"state_changed_at"`
	StateChangeReason   string         `json:"state_change_reason,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// Promotion represents a state transition in the audit log
type Promotion struct {
	ID             uuid.UUID      `json:"id"`
	ArtifactID     uuid.UUID      `json:"artifact_id"`
	FromState      ApprovalState  `json:"from_state"`
	ToState        ApprovalState  `json:"to_state"`
	PromotedBy     string         `json:"promoted_by"`
	PromotedAt     time.Time      `json:"promoted_at"`
	Reason         string         `json:"reason,omitempty"`
	ValidationData map[string]any `json:"validation_data,omitempty"`
}

// ValidationReport represents detailed test results
type ValidationReport struct {
	ID              uuid.UUID      `json:"id"`
	ArtifactID      uuid.UUID      `json:"artifact_id"`
	RunID           uuid.UUID      `json:"run_id"`
	TestType        string         `json:"test_type"`
	Passed          bool           `json:"passed"`
	Metrics         map[string]any `json:"metrics,omitempty"`
	Errors          []string       `json:"errors,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
	DeterminismSeed *int           `json:"determinism_seed,omitempty"`
	TestEnvironment map[string]any `json:"test_environment,omitempty"`
	ReportURI       string         `json:"report_uri,omitempty"`
	StartedAt       time.Time      `json:"started_at"`
	CompletedAt     time.Time      `json:"completed_at"`
	DurationSeconds float64        `json:"duration_seconds"`
	CreatedAt       time.Time      `json:"created_at"`
}

// CanonicalPayload creates a deterministic JSON representation for hashing
func (a *Artifact) CanonicalPayload() ([]byte, error) {
	// Create a canonical representation with sorted keys
	canonical := map[string]interface{}{
		"artifact_id":    a.ArtifactID,
		"schema_version": a.SchemaVersion,
		"strategy": map[string]interface{}{
			"name":     a.Strategy.Name,
			"version":  a.Strategy.Version,
			"code_ref": a.Strategy.CodeRef,
			"params":   sortedMap(a.Strategy.Params),
		},
		"risk_profile": map[string]interface{}{
			"max_position_pct":    a.RiskProfile.MaxPositionPct,
			"max_daily_loss":      a.RiskProfile.MaxDailyLoss,
			"allowed_order_types": a.RiskProfile.AllowedOrderTypes,
		},
		"created_by": a.CreatedBy,
		"created_at": a.CreatedAt.UTC().Format(time.RFC3339),
	}

	// Add optional fields if present
	if a.DataWindow != nil {
		canonical["data_window"] = map[string]interface{}{
			"from":    a.DataWindow.From.UTC().Format(time.RFC3339),
			"to":      a.DataWindow.To.UTC().Format(time.RFC3339),
			"symbols": a.DataWindow.Symbols,
		}
	}

	if a.Validation != nil {
		canonical["validation"] = map[string]interface{}{
			"backtest_run_id":  a.Validation.BacktestRunID.String(),
			"metrics":          sortedMap(a.Validation.Metrics),
			"determinism_seed": a.Validation.DeterminismSeed,
			"report_uri":       a.Validation.ReportURI,
		}
	}

	// Marshal to canonical JSON (sorted keys)
	return json.Marshal(canonical)
}

// ComputeHash computes SHA-256 hash of the canonical payload
func (a *Artifact) ComputeHash() (string, error) {
	payload, err := a.CanonicalPayload()
	if err != nil {
		return "", fmt.Errorf("failed to create canonical payload: %w", err)
	}

	hash := sha256.Sum256(payload)
	return fmt.Sprintf("%x", hash), nil
}

// VerifyHash checks if the artifact's hash matches its payload
func (a *Artifact) VerifyHash() error {
	computed, err := a.ComputeHash()
	if err != nil {
		return err
	}

	if computed != a.Hash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", a.Hash, computed)
	}

	return nil
}

// IsApproved checks if the artifact is in an approved state
func (a *Approval) IsApproved() bool {
	return a.State == StateApproved || a.State == StateActive
}

// IsUsable checks if the artifact can be used by the trader runtime
func (a *Approval) IsUsable() bool {
	return (a.State == StateApproved || a.State == StateActive) && a.State != StateRevoked
}

// CanTransitionTo checks if a state transition is valid
func (a *Approval) CanTransitionTo(toState ApprovalState) bool {
	allowed, ok := ValidTransitions[a.State]
	if !ok {
		return false
	}

	for _, state := range allowed {
		if state == toState {
			return true
		}
	}

	return false
}

// sortedMap returns a map with sorted keys for deterministic JSON
func sortedMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create new map with sorted iteration
	sorted := make(map[string]any, len(m))
	for _, k := range keys {
		sorted[k] = m[k]
	}

	return sorted
}

// NewArtifact creates a new artifact with computed hash
func NewArtifact(
	strategyName string,
	strategyVersion string,
	params map[string]any,
	riskProfile RiskProfile,
	createdBy string,
) (*Artifact, error) {
	now := time.Now().UTC()

	artifact := &Artifact{
		ID:            uuid.New(),
		ArtifactID:    fmt.Sprintf("strat_%s_%s", strategyName, now.Format("2006-01-02T15:04:05Z")),
		SchemaVersion: "1.0.0",
		Strategy: StrategyInfo{
			Name:    strategyName,
			Version: strategyVersion,
			Params:  params,
		},
		RiskProfile: riskProfile,
		CreatedBy:   createdBy,
		CreatedAt:   now,
	}

	// Compute hash
	hash, err := artifact.ComputeHash()
	if err != nil {
		return nil, fmt.Errorf("failed to compute artifact hash: %w", err)
	}
	artifact.Hash = hash

	return artifact, nil
}

// NewApproval creates a new approval in DRAFT state
func NewApproval(artifactID uuid.UUID, createdBy string) *Approval {
	now := time.Now().UTC()
	return &Approval{
		ID:             uuid.New(),
		ArtifactID:     artifactID,
		State:          StateDraft,
		StateChangedBy: createdBy,
		StateChangedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
