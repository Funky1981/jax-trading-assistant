package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jax-trading-assistant/services/jax-api/internal/app"

	"github.com/google/uuid"
)

// PostgresSignalStore implements app.SignalStore using PostgreSQL
type PostgresSignalStore struct {
	db *sql.DB
}

// NewPostgresSignalStore creates a new PostgreSQL signal store
func NewPostgresSignalStore(db *sql.DB) *PostgresSignalStore {
	return &PostgresSignalStore{db: db}
}

// ListSignals returns a list of signals with filtering and pagination
func (s *PostgresSignalStore) ListSignals(ctx context.Context, status, symbol, strategy string, limit, offset int) (*app.SignalListResponse, error) {
	// Build the query dynamically based on filters
	query := `
		SELECT id, symbol, strategy_id, signal_type, confidence, 
		       entry_price, stop_loss, take_profit, reasoning,
		       generated_at, expires_at, status, orchestration_run_id, created_at
		FROM strategy_signals
		WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM strategy_signals WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	// Add filters if provided
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argCount)
		countQuery += fmt.Sprintf(" AND symbol = $%d", argCount)
		args = append(args, symbol)
		argCount++
	}

	if strategy != "" {
		query += fmt.Sprintf(" AND strategy_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND strategy_id = $%d", argCount)
		args = append(args, strategy)
		argCount++
	}

	// Add ordering and pagination
	query += " ORDER BY generated_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
		argCount++
	}

	// Get total count
	var total int
	countArgs := args
	if limit > 0 {
		countArgs = args[:len(args)-1]
		if offset > 0 {
			countArgs = args[:len(args)-2]
		}
	}
	err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count signals: %w", err)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query signals: %w", err)
	}
	defer rows.Close()

	signals := []app.Signal{}
	for rows.Next() {
		var sig app.Signal
		err := rows.Scan(
			&sig.ID, &sig.Symbol, &sig.StrategyID, &sig.SignalType, &sig.Confidence,
			&sig.EntryPrice, &sig.StopLoss, &sig.TakeProfit, &sig.Reasoning,
			&sig.GeneratedAt, &sig.ExpiresAt, &sig.Status, &sig.OrchestrationRunID, &sig.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal: %w", err)
		}
		signals = append(signals, sig)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating signals: %w", err)
	}

	// Set defaults for limit/offset if not provided
	if limit == 0 {
		limit = total
	}

	return &app.SignalListResponse{
		Signals: signals,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

// GetSignal returns a single signal by ID
func (s *PostgresSignalStore) GetSignal(ctx context.Context, id uuid.UUID) (*app.Signal, error) {
	query := `
		SELECT id, symbol, strategy_id, signal_type, confidence, 
		       entry_price, stop_loss, take_profit, reasoning,
		       generated_at, expires_at, status, orchestration_run_id, created_at
		FROM strategy_signals
		WHERE id = $1`

	var sig app.Signal
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&sig.ID, &sig.Symbol, &sig.StrategyID, &sig.SignalType, &sig.Confidence,
		&sig.EntryPrice, &sig.StopLoss, &sig.TakeProfit, &sig.Reasoning,
		&sig.GeneratedAt, &sig.ExpiresAt, &sig.Status, &sig.OrchestrationRunID, &sig.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get signal: %w", err)
	}

	return &sig, nil
}

// ApproveSignal approves a signal and creates a trade_approvals record
func (s *PostgresSignalStore) ApproveSignal(ctx context.Context, id uuid.UUID, approvedBy, modificationNotes string) (*app.Signal, error) {
	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update signal status
	updateQuery := `
		UPDATE strategy_signals
		SET status = 'approved'
		WHERE id = $1
		RETURNING id, symbol, strategy_id, signal_type, confidence, 
		          entry_price, stop_loss, take_profit, reasoning,
		          generated_at, expires_at, status, orchestration_run_id, created_at`

	var sig app.Signal
	err = tx.QueryRowContext(ctx, updateQuery, id).Scan(
		&sig.ID, &sig.Symbol, &sig.StrategyID, &sig.SignalType, &sig.Confidence,
		&sig.EntryPrice, &sig.StopLoss, &sig.TakeProfit, &sig.Reasoning,
		&sig.GeneratedAt, &sig.ExpiresAt, &sig.Status, &sig.OrchestrationRunID, &sig.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update signal: %w", err)
	}

	// Create trade_approvals record
	insertQuery := `
		INSERT INTO trade_approvals (signal_id, orchestration_run_id, approved, approved_at, approved_by, modification_notes)
		VALUES ($1, $2, true, $3, $4, $5)`

	var notes *string
	if modificationNotes != "" {
		notes = &modificationNotes
	}

	_, err = tx.ExecContext(ctx, insertQuery, sig.ID, sig.OrchestrationRunID, time.Now(), approvedBy, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to create approval record: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &sig, nil
}

// RejectSignal rejects a signal and creates a trade_approvals record with approved=false
func (s *PostgresSignalStore) RejectSignal(ctx context.Context, id uuid.UUID, approvedBy, rejectionReason string) (*app.Signal, error) {
	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update signal status
	updateQuery := `
		UPDATE strategy_signals
		SET status = 'rejected'
		WHERE id = $1
		RETURNING id, symbol, strategy_id, signal_type, confidence, 
		          entry_price, stop_loss, take_profit, reasoning,
		          generated_at, expires_at, status, orchestration_run_id, created_at`

	var sig app.Signal
	err = tx.QueryRowContext(ctx, updateQuery, id).Scan(
		&sig.ID, &sig.Symbol, &sig.StrategyID, &sig.SignalType, &sig.Confidence,
		&sig.EntryPrice, &sig.StopLoss, &sig.TakeProfit, &sig.Reasoning,
		&sig.GeneratedAt, &sig.ExpiresAt, &sig.Status, &sig.OrchestrationRunID, &sig.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update signal: %w", err)
	}

	// Create trade_approvals record with approved=false
	// Store rejection reason in modification_notes field
	insertQuery := `
		INSERT INTO trade_approvals (signal_id, orchestration_run_id, approved, approved_at, approved_by, modification_notes)
		VALUES ($1, $2, false, $3, $4, $5)`

	var notes *string
	if rejectionReason != "" {
		notes = &rejectionReason
	}

	_, err = tx.ExecContext(ctx, insertQuery, sig.ID, sig.OrchestrationRunID, time.Now(), approvedBy, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to create rejection record: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &sig, nil
}
