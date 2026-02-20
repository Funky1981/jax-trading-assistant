package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides database operations for artifacts
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new artifact store
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateArtifact stores a new artifact in the database
func (s *Store) CreateArtifact(ctx context.Context, artifact *Artifact) error {
	query := `
		INSERT INTO strategy_artifacts (
			id, artifact_id, schema_version, strategy_name, strategy_version, code_ref,
			params, data_window_from, data_window_to, symbols, validation,
			risk_profile, hash, signature, payload, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	// Marshal complex fields to JSONB
	params, err := json.Marshal(artifact.Strategy.Params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	riskProfile, err := json.Marshal(artifact.RiskProfile)
	if err != nil {
		return fmt.Errorf("failed to marshal risk_profile: %w", err)
	}

	payload, err := artifact.CanonicalPayload()
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	var dataWindowFrom, dataWindowTo sql.NullTime
	var symbols []string
	var validationJSON []byte

	if artifact.DataWindow != nil {
		dataWindowFrom = sql.NullTime{Time: artifact.DataWindow.From, Valid: true}
		dataWindowTo = sql.NullTime{Time: artifact.DataWindow.To, Valid: true}
		symbols = artifact.DataWindow.Symbols
	}

	if artifact.Validation != nil {
		validationJSON, err = json.Marshal(artifact.Validation)
		if err != nil {
			return fmt.Errorf("failed to marshal validation: %w", err)
		}
	}

	_, err = s.pool.Exec(ctx, query,
		artifact.ID,
		artifact.ArtifactID,
		artifact.SchemaVersion,
		artifact.Strategy.Name,
		artifact.Strategy.Version,
		artifact.Strategy.CodeRef,
		params,
		dataWindowFrom,
		dataWindowTo,
		symbols,
		validationJSON,
		riskProfile,
		artifact.Hash,
		artifact.Signature,
		payload,
		artifact.CreatedBy,
		artifact.CreatedAt,
	)

	return err
}

// GetArtifactByID retrieves an artifact by its UUID
func (s *Store) GetArtifactByID(ctx context.Context, id uuid.UUID) (*Artifact, error) {
	query := `
		SELECT id, artifact_id, schema_version, strategy_name, strategy_version, code_ref,
		       params, data_window_from, data_window_to, symbols, validation,
		       risk_profile, hash, signature, created_by, created_at
		FROM strategy_artifacts
		WHERE id = $1
	`

	var artifact Artifact
	var codeRef sql.NullString
	var dataWindowFrom, dataWindowTo sql.NullTime
	var symbols []string
	var paramsJSON, validationJSON, riskProfileJSON []byte
	var signature sql.NullString

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&artifact.ID,
		&artifact.ArtifactID,
		&artifact.SchemaVersion,
		&artifact.Strategy.Name,
		&artifact.Strategy.Version,
		&codeRef,
		&paramsJSON,
		&dataWindowFrom,
		&dataWindowTo,
		&symbols,
		&validationJSON,
		&riskProfileJSON,
		&artifact.Hash,
		&signature,
		&artifact.CreatedBy,
		&artifact.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found: %s", id)
		}
		return nil, err
	}

	artifact.Strategy.CodeRef = codeRef.String
	artifact.Signature = signature.String

	if err := json.Unmarshal(paramsJSON, &artifact.Strategy.Params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if err := json.Unmarshal(riskProfileJSON, &artifact.RiskProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal risk_profile: %w", err)
	}

	if dataWindowFrom.Valid && dataWindowTo.Valid {
		artifact.DataWindow = &DataWindow{
			From:    dataWindowFrom.Time,
			To:      dataWindowTo.Time,
			Symbols: symbols,
		}
	}

	if len(validationJSON) > 0 {
		var validation ValidationInfo
		if err := json.Unmarshal(validationJSON, &validation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal validation: %w", err)
		}
		artifact.Validation = &validation
	}

	return &artifact, nil
}

// GetArtifactByHash retrieves an artifact by its SHA-256 hash
func (s *Store) GetArtifactByHash(ctx context.Context, hash string) (*Artifact, error) {
	query := `
		SELECT id
		FROM strategy_artifacts
		WHERE hash = $1
	`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, query, hash).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found for hash: %s", hash)
		}
		return nil, err
	}

	return s.GetArtifactByID(ctx, id)
}

// ListApprovedArtifacts returns all artifacts in APPROVED or ACTIVE state
// Optimized to fetch all data in single query (no N+1)
func (s *Store) ListApprovedArtifacts(ctx context.Context) ([]*Artifact, error) {
	query := `
		SELECT a.id, a.artifact_id, a.schema_version, a.strategy_name, a.strategy_version, a.code_ref,
		       a.params, a.data_window_from, a.data_window_to, a.symbols, a.validation,
		       a.risk_profile, a.hash, a.signature, a.created_by, a.created_at
		FROM strategy_artifacts a
		JOIN artifact_approvals ap ON a.id = ap.artifact_id
		WHERE ap.state IN ('APPROVED', 'ACTIVE')
		  AND ap.state <> 'REVOKED'
		ORDER BY a.strategy_name, a.created_at DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*Artifact
	for rows.Next() {
		var artifact Artifact
		var codeRef sql.NullString
		var dataWindowFrom, dataWindowTo sql.NullTime
		var symbols []string
		var paramsJSON, validationJSON, riskProfileJSON []byte
		var signature sql.NullString

		err := rows.Scan(
			&artifact.ID,
			&artifact.ArtifactID,
			&artifact.SchemaVersion,
			&artifact.Strategy.Name,
			&artifact.Strategy.Version,
			&codeRef,
			&paramsJSON,
			&dataWindowFrom,
			&dataWindowTo,
			&symbols,
			&validationJSON,
			&riskProfileJSON,
			&artifact.Hash,
			&signature,
			&artifact.CreatedBy,
			&artifact.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		artifact.Strategy.CodeRef = codeRef.String
		artifact.Signature = signature.String

		if err := json.Unmarshal(paramsJSON, &artifact.Strategy.Params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}

		if err := json.Unmarshal(riskProfileJSON, &artifact.RiskProfile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal risk_profile: %w", err)
		}

		if dataWindowFrom.Valid && dataWindowTo.Valid {
			artifact.DataWindow = &DataWindow{
				From:    dataWindowFrom.Time,
				To:      dataWindowTo.Time,
				Symbols: symbols,
			}
		}

		if len(validationJSON) > 0 {
			var validation ValidationInfo
			if err := json.Unmarshal(validationJSON, &validation); err != nil {
				return nil, fmt.Errorf("failed to unmarshal validation: %w", err)
			}
			artifact.Validation = &validation
		}

		artifacts = append(artifacts, &artifact)
	}

	return artifacts, rows.Err()
}

// GetLatestApprovedArtifact returns the most recent approved artifact for a strategy
func (s *Store) GetLatestApprovedArtifact(ctx context.Context, strategyName string) (*Artifact, error) {
	query := `
		SELECT a.id
		FROM strategy_artifacts a
		JOIN artifact_approvals ap ON a.id = ap.artifact_id
		WHERE a.strategy_name = $1
		  AND ap.state IN ('APPROVED', 'ACTIVE')
		  AND ap.state <> 'REVOKED'
		ORDER BY a.created_at DESC
		LIMIT 1
	`

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, query, strategyName).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no approved artifact found for strategy: %s", strategyName)
		}
		return nil, err
	}

	return s.GetArtifactByID(ctx, id)
}

// CreateApproval creates a new approval workflow entry
func (s *Store) CreateApproval(ctx context.Context, approval *Approval) error {
	query := `
		INSERT INTO artifact_approvals (
			id, artifact_id, state, previous_state, approved_by, approved_at,
			validation_run_id, validation_passed, validation_report_uri,
			review_notes, reviewer, reviewed_at,
			state_changed_by, state_changed_at, state_change_reason,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := s.pool.Exec(ctx, query,
		approval.ID,
		approval.ArtifactID,
		approval.State,
		approval.PreviousState,
		approval.ApprovedBy,
		approval.ApprovedAt,
		approval.ValidationRunID,
		approval.ValidationPassed,
		approval.ValidationReportURI,
		approval.ReviewNotes,
		approval.Reviewer,
		approval.ReviewedAt,
		approval.StateChangedBy,
		approval.StateChangedAt,
		approval.StateChangeReason,
		approval.CreatedAt,
		approval.UpdatedAt,
	)

	return err
}

// GetApproval retrieves the approval for an artifact
func (s *Store) GetApproval(ctx context.Context, artifactID uuid.UUID) (*Approval, error) {
	query := `
		SELECT id, artifact_id, state, previous_state, approved_by, approved_at,
		       validation_run_id, validation_passed, validation_report_uri,
		       review_notes, reviewer, reviewed_at,
		       state_changed_by, state_changed_at, state_change_reason,
		       created_at, updated_at
		FROM artifact_approvals
		WHERE artifact_id = $1
	`

	var approval Approval
	var previousState, approvedBy, reportURI, reviewNotes, reviewer sql.NullString
	var stateChangedBy, stateChangeReason sql.NullString
	var approvedAt, reviewedAt sql.NullTime
	var validationRunID *uuid.UUID

	err := s.pool.QueryRow(ctx, query, artifactID).Scan(
		&approval.ID,
		&approval.ArtifactID,
		&approval.State,
		&previousState,
		&approvedBy,
		&approvedAt,
		&validationRunID,
		&approval.ValidationPassed,
		&reportURI,
		&reviewNotes,
		&reviewer,
		&reviewedAt,
		&stateChangedBy,
		&approval.StateChangedAt,
		&stateChangeReason,
		&approval.CreatedAt,
		&approval.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("approval not found for artifact: %s", artifactID)
		}
		return nil, err
	}

	if previousState.Valid {
		state := ApprovalState(previousState.String)
		approval.PreviousState = &state
	}
	if approvedBy.Valid {
		approval.ApprovedBy = approvedBy.String
	}
	if approvedAt.Valid {
		approval.ApprovedAt = &approvedAt.Time
	}
	if validationRunID != nil {
		approval.ValidationRunID = validationRunID
	}
	if reportURI.Valid {
		approval.ValidationReportURI = reportURI.String
	}
	if reviewNotes.Valid {
		approval.ReviewNotes = reviewNotes.String
	}
	if reviewer.Valid {
		approval.Reviewer = reviewer.String
	}
	if reviewedAt.Valid {
		approval.ReviewedAt = &reviewedAt.Time
	}
	if stateChangedBy.Valid {
		approval.StateChangedBy = stateChangedBy.String
	}
	if stateChangeReason.Valid {
		approval.StateChangeReason = stateChangeReason.String
	}

	return &approval, nil
}

// UpdateApprovalState transitions the approval state and records the promotion
func (s *Store) UpdateApprovalState(ctx context.Context, artifactID uuid.UUID, toState ApprovalState, promotedBy, reason string) error {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Get current approval
	var currentState ApprovalState
	err = tx.QueryRow(ctx, `SELECT state FROM artifact_approvals WHERE artifact_id = $1`, artifactID).Scan(&currentState)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Validate transition
	approval := &Approval{State: currentState}
	if !approval.CanTransitionTo(toState) {
		return fmt.Errorf("invalid state transition: %s -> %s", currentState, toState)
	}

	// Update approval state
	updateQuery := `
		UPDATE artifact_approvals
		SET state = $1,
		    previous_state = $2,
		    state_changed_by = $3,
		    state_changed_at = $4,
		    state_change_reason = $5,
		    updated_at = $4
		WHERE artifact_id = $6
	`

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, updateQuery, toState, currentState, promotedBy, now, reason, artifactID)
	if err != nil {
		return fmt.Errorf("failed to update approval state: %w", err)
	}

	// Record promotion in audit log
	promotionQuery := `
		INSERT INTO artifact_promotions (id, artifact_id, from_state, to_state, promoted_by, promoted_at, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = tx.Exec(ctx, promotionQuery, uuid.New(), artifactID, currentState, toState, promotedBy, now, reason)
	if err != nil {
		return fmt.Errorf("failed to record promotion: %w", err)
	}

	return tx.Commit(ctx)
}

// CreateValidationReport stores a validation test result
func (s *Store) CreateValidationReport(ctx context.Context, report *ValidationReport) error {
	query := `
		INSERT INTO artifact_validation_reports (
			id, artifact_id, run_id, test_type, passed,
			metrics, errors, warnings, determinism_seed, test_environment,
			report_uri, started_at, completed_at, duration_seconds, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	metricsJSON, err := json.Marshal(report.Metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	testEnvJSON, err := json.Marshal(report.TestEnvironment)
	if err != nil {
		return fmt.Errorf("failed to marshal test_environment: %w", err)
	}

	_, err = s.pool.Exec(ctx, query,
		report.ID,
		report.ArtifactID,
		report.RunID,
		report.TestType,
		report.Passed,
		metricsJSON,
		report.Errors,
		report.Warnings,
		report.DeterminismSeed,
		testEnvJSON,
		report.ReportURI,
		report.StartedAt,
		report.CompletedAt,
		report.DurationSeconds,
		report.CreatedAt,
	)

	return err
}

// GetByStringArtifactID retrieves an artifact by its human-readable artifact_id
// string (e.g. "strat_momentum_2024-01-15T10:00:00Z"). Used by the artifact-approver
// CLI which works with string IDs rather than internal UUIDs.
func (s *Store) GetByStringArtifactID(ctx context.Context, artifactIDStr string) (*Artifact, error) {
	query := `
		SELECT id, artifact_id, schema_version, strategy_name, strategy_version, code_ref,
		       params, data_window_from, data_window_to, symbols, validation,
		       risk_profile, hash, signature, created_by, created_at
		FROM strategy_artifacts
		WHERE artifact_id = $1
	`

	var artifact Artifact
	var codeRef sql.NullString
	var dataWindowFrom, dataWindowTo sql.NullTime
	var symbols []string
	var paramsJSON, validationJSON, riskProfileJSON []byte
	var signature sql.NullString

	err := s.pool.QueryRow(ctx, query, artifactIDStr).Scan(
		&artifact.ID,
		&artifact.ArtifactID,
		&artifact.SchemaVersion,
		&artifact.Strategy.Name,
		&artifact.Strategy.Version,
		&codeRef,
		&paramsJSON,
		&dataWindowFrom,
		&dataWindowTo,
		&symbols,
		&validationJSON,
		&riskProfileJSON,
		&artifact.Hash,
		&signature,
		&artifact.CreatedBy,
		&artifact.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found: %s", artifactIDStr)
		}
		return nil, err
	}

	artifact.Strategy.CodeRef = codeRef.String
	artifact.Signature = signature.String

	if err := json.Unmarshal(paramsJSON, &artifact.Strategy.Params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal params: %w", err)
	}

	if err := json.Unmarshal(riskProfileJSON, &artifact.RiskProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal risk_profile: %w", err)
	}

	if dataWindowFrom.Valid && dataWindowTo.Valid {
		artifact.DataWindow = &DataWindow{
			From:    dataWindowFrom.Time,
			To:      dataWindowTo.Time,
			Symbols: symbols,
		}
	}

	if len(validationJSON) > 0 {
		var validation ValidationInfo
		if err := json.Unmarshal(validationJSON, &validation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal validation: %w", err)
		}
		artifact.Validation = &validation
	}

	return &artifact, nil
}
