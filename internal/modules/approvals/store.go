// Package approvals manages the human approval workflow for candidate trades.
package approvals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Decision values for a candidate approval.
const (
	DecisionApproved            = "approved"
	DecisionRejected            = "rejected"
	DecisionSnoozed             = "snoozed"
	DecisionReanalysisRequested = "reanalysis_requested"
)

// Approval is a human decision record for a candidate trade.
type Approval struct {
	ID                  uuid.UUID  `json:"id"`
	CandidateID         uuid.UUID  `json:"candidateId"`
	Decision            string     `json:"decision"`
	ApprovedBy          string     `json:"approvedBy"`
	Notes               *string    `json:"notes,omitempty"`
	ExpiryAt            *time.Time `json:"expiryAt,omitempty"`
	SnoozeUntil         *time.Time `json:"snoozeUntil,omitempty"`
	ReanalysisRequested bool       `json:"reanalysisRequested"`
	DecidedAt           time.Time  `json:"decidedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
}

// ExecutionInstruction is the DB representation of an approved trade ready for execution.
type ExecutionInstruction struct {
	ID            uuid.UUID  `json:"id"`
	ApprovalID    uuid.UUID  `json:"approvalId"`
	CandidateID   uuid.UUID  `json:"candidateId"`
	Symbol        string     `json:"symbol"`
	SignalType    string     `json:"signalType"`
	EntryPrice    *float64   `json:"entryPrice,omitempty"`
	StopLoss      *float64   `json:"stopLoss,omitempty"`
	TakeProfit    *float64   `json:"takeProfit,omitempty"`
	Status        string     `json:"status"`
	BrokerOrderID *string    `json:"brokerOrderId,omitempty"`
	FillPrice     *float64   `json:"fillPrice,omitempty"`
	FillQty       *int       `json:"fillQty,omitempty"`
	ErrorMessage  *string    `json:"errorMessage,omitempty"`
	SubmittedAt   *time.Time `json:"submittedAt,omitempty"`
	FilledAt      *time.Time `json:"filledAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// Store handles DB persistence for approvals and execution instructions.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates an approval Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// RecordDecision inserts an approval record for a candidate.
func (s *Store) RecordDecision(ctx context.Context, a *Approval) (*Approval, error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	now := time.Now().UTC()
	a.DecidedAt = now
	a.CreatedAt = now

	_, err := s.pool.Exec(ctx, `
		INSERT INTO candidate_approvals
			(id, candidate_id, decision, approved_by, notes, expiry_at, snooze_until, reanalysis_requested, decided_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		a.ID, a.CandidateID, a.Decision, a.ApprovedBy, a.Notes,
		a.ExpiryAt, a.SnoozeUntil, a.ReanalysisRequested, a.DecidedAt, a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.RecordDecision: %w", err)
	}
	return a, nil
}

// GetByCandidateID returns the latest approval for a candidate.
func (s *Store) GetByCandidateID(ctx context.Context, candidateID uuid.UUID) (*Approval, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, candidate_id, decision, approved_by, notes, expiry_at,
		       snooze_until, reanalysis_requested, decided_at, created_at
		FROM candidate_approvals
		WHERE candidate_id = $1
		ORDER BY decided_at DESC LIMIT 1`, candidateID)
	return scanApproval(row)
}

// ListQueue returns candidates that are awaiting_approval.
func (s *Store) ListQueue(ctx context.Context, limit int) ([]map[string]any, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ct.id::text, ct.symbol, ct.signal_type, ct.confidence, ct.entry_price,
		       ct.stop_loss, ct.take_profit, ct.reasoning, ct.block_reason,
		       ct.detected_at, ct.expires_at, si.name AS instance_name
		FROM candidate_trades ct
		LEFT JOIN strategy_instances si ON si.id = ct.strategy_instance_id
		WHERE ct.status = 'awaiting_approval'
		ORDER BY ct.detected_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.ListQueue: %w", err)
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var (
			id, symbol, signalType, instanceName         string
			confidence, entryPrice, stopLoss, takeProfit *float64
			reasoning, blockReason                       *string
			detectedAt                                   time.Time
			expiresAt                                    *time.Time
		)
		if err := rows.Scan(
			&id, &symbol, &signalType, &confidence, &entryPrice,
			&stopLoss, &takeProfit, &reasoning, &blockReason,
			&detectedAt, &expiresAt, &instanceName,
		); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id":           id,
			"symbol":       symbol,
			"signalType":   signalType,
			"confidence":   confidence,
			"entryPrice":   entryPrice,
			"stopLoss":     stopLoss,
			"takeProfit":   takeProfit,
			"reasoning":    reasoning,
			"blockReason":  blockReason,
			"detectedAt":   detectedAt,
			"expiresAt":    expiresAt,
			"instanceName": instanceName,
		})
	}
	return out, rows.Err()
}

// CreateExecutionInstruction persists an execution instruction row.
func (s *Store) CreateExecutionInstruction(ctx context.Context, inst *ExecutionInstruction) (*ExecutionInstruction, error) {
	if inst.ID == uuid.Nil {
		inst.ID = uuid.New()
	}
	inst.CreatedAt = time.Now().UTC()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO execution_instructions
			(id, approval_id, candidate_id, symbol, signal_type,
			 entry_price, stop_loss, take_profit, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending',$9,$9)`,
		inst.ID, inst.ApprovalID, inst.CandidateID, inst.Symbol, inst.SignalType,
		inst.EntryPrice, inst.StopLoss, inst.TakeProfit, inst.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.CreateExecutionInstruction: %w", err)
	}
	return inst, nil
}

func scanApproval(row interface{ Scan(...any) error }) (*Approval, error) {
	var a Approval
	err := row.Scan(
		&a.ID, &a.CandidateID, &a.Decision, &a.ApprovedBy, &a.Notes,
		&a.ExpiryAt, &a.SnoozeUntil, &a.ReanalysisRequested, &a.DecidedAt, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanApproval: %w", err)
	}
	return &a, nil
}
