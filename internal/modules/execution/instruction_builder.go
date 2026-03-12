package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApprovedInstruction is the input contract for the execution engine coming
// from the approval flow.  Only rows created via candidate_approvals with
// decision='approved' should ever reach here.
type ApprovedInstruction struct {
	InstructionID uuid.UUID
	ApprovalID    uuid.UUID
	CandidateID   uuid.UUID
	Symbol        string
	SignalType    string // BUY | SELL
	EntryPrice    *float64
	StopLoss      *float64
	TakeProfit    *float64
	ApprovedBy    string
	ApprovedAt    time.Time
}

// InstructionBuilder reads execution_instructions rows that are 'pending'
// and converts them to ApprovedInstruction values ready for engine.Execute.
type InstructionBuilder struct {
	pool *pgxpool.Pool
}

// NewInstructionBuilder creates an InstructionBuilder.
func NewInstructionBuilder(pool *pgxpool.Pool) *InstructionBuilder {
	return &InstructionBuilder{pool: pool}
}

// NextPending returns the oldest pending execution instruction (if any).
func (b *InstructionBuilder) NextPending(ctx context.Context) (*ApprovedInstruction, error) {
	row := b.pool.QueryRow(ctx, `
		SELECT ei.id, ei.approval_id, ei.candidate_id,
		       ei.symbol, ei.signal_type, ei.entry_price, ei.stop_loss, ei.take_profit,
		       ca.approved_by, ca.decided_at
		FROM execution_instructions ei
		JOIN candidate_approvals ca ON ca.id = ei.approval_id
		WHERE ei.status = 'pending'
		ORDER BY ei.created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED`)

	var inst ApprovedInstruction
	err := row.Scan(
		&inst.InstructionID, &inst.ApprovalID, &inst.CandidateID,
		&inst.Symbol, &inst.SignalType, &inst.EntryPrice, &inst.StopLoss, &inst.TakeProfit,
		&inst.ApprovedBy, &inst.ApprovedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("InstructionBuilder.NextPending: %w", err)
	}
	return &inst, nil
}

// MarkSubmitted updates the instruction row to 'submitted' with a broker order ID.
func (b *InstructionBuilder) MarkSubmitted(ctx context.Context, instructionID uuid.UUID, brokerOrderID string) error {
	_, err := b.pool.Exec(ctx, `
		UPDATE execution_instructions
		   SET status = 'submitted', broker_order_id = $2, submitted_at = NOW(), updated_at = NOW()
		 WHERE id = $1`, instructionID, brokerOrderID)
	return err
}

// MarkFilled updates the instruction row to 'filled'.
func (b *InstructionBuilder) MarkFilled(ctx context.Context, instructionID uuid.UUID, fillPrice float64, fillQty int) error {
	_, err := b.pool.Exec(ctx, `
		UPDATE execution_instructions
		   SET status = 'filled', fill_price = $2, fill_qty = $3, filled_at = NOW(), updated_at = NOW()
		 WHERE id = $1`, instructionID, fillPrice, fillQty)
	return err
}

// MarkRejected records a rejection from the broker for an instruction.
func (b *InstructionBuilder) MarkRejected(ctx context.Context, instructionID uuid.UUID, reason string) error {
	_, err := b.pool.Exec(ctx, `
		UPDATE execution_instructions
		   SET status = 'rejected', error_message = $2, updated_at = NOW()
		 WHERE id = $1`, instructionID, reason)
	return err
}
