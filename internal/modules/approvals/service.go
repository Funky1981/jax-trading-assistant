package approvals

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service applies approval business rules.
type Service struct {
	store *Store
	pool  *pgxpool.Pool
}

// NewService creates an approval Service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{store: NewStore(pool), pool: pool}
}

// ApprovalRequest carries input for any approval action.
type ApprovalRequest struct {
	CandidateID uuid.UUID
	Decision    string
	ApprovedBy  string
	Notes       *string
	ExpiryAt    *time.Time
	SnoozeHours int
}

// Decide records an approval decision and, if approved, creates an execution instruction.
// Returns ErrAlreadyDecided if a final decision was already made.
// Returns ErrCandidateExpired if the candidate has passed its expiry.
func (s *Service) Decide(ctx context.Context, req ApprovalRequest) (*Approval, error) {
	// Verify candidate is awaiting_approval and not expired.
	var status string
	var expiresAt *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT status, expires_at FROM candidate_trades WHERE id = $1`, req.CandidateID,
	).Scan(&status, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("approvals.Service.Decide: candidate lookup: %w", err)
	}
	if expiresAt != nil && time.Now().UTC().After(*expiresAt) {
		return nil, ErrCandidateExpired
	}
	if status != "awaiting_approval" {
		return nil, fmt.Errorf("%w: status=%s", ErrNotAwaitingApproval, status)
	}

	a := &Approval{
		CandidateID:         req.CandidateID,
		Decision:            req.Decision,
		ApprovedBy:          req.ApprovedBy,
		Notes:               req.Notes,
		ExpiryAt:            req.ExpiryAt,
		ReanalysisRequested: req.Decision == DecisionReanalysisRequested,
	}
	if req.Decision == DecisionSnoozed && req.SnoozeHours > 0 {
		t := time.Now().UTC().Add(time.Duration(req.SnoozeHours) * time.Hour)
		a.SnoozeUntil = &t
	}

	approval, err := s.store.RecordDecision(ctx, a)
	if err != nil {
		return nil, err
	}

	// Update candidate status to match.
	newCandidateStatus := map[string]string{
		DecisionApproved:            "approved",
		DecisionRejected:            "rejected",
		DecisionSnoozed:             "awaiting_approval", // stays in queue
		DecisionReanalysisRequested: "awaiting_approval", // stays in queue
	}[req.Decision]

	_, err = s.pool.Exec(ctx,
		`UPDATE candidate_trades SET status = $2, updated_at = NOW() WHERE id = $1`,
		req.CandidateID, newCandidateStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Service.Decide: update candidate status: %w", err)
	}

	// If approved, build an execution instruction.
	if req.Decision == DecisionApproved {
		if err := s.buildInstruction(ctx, approval); err != nil {
			return nil, fmt.Errorf("approvals.Service.Decide: build instruction: %w", err)
		}
	}

	return approval, nil
}

// buildInstruction creates an execution_instruction row from an approved candidate.
func (s *Service) buildInstruction(ctx context.Context, approval *Approval) error {
	var (
		symbol, signalType        string
		entryPrice, stopLoss, tpx *float64
	)
	err := s.pool.QueryRow(ctx,
		`SELECT symbol, signal_type, entry_price, stop_loss, take_profit
		   FROM candidate_trades WHERE id = $1`, approval.CandidateID,
	).Scan(&symbol, &signalType, &entryPrice, &stopLoss, &tpx)
	if err != nil {
		return fmt.Errorf("buildInstruction lookup candidate: %w", err)
	}
	inst := &ExecutionInstruction{
		ApprovalID:  approval.ID,
		CandidateID: approval.CandidateID,
		Symbol:      symbol,
		SignalType:  signalType,
		EntryPrice:  entryPrice,
		StopLoss:    stopLoss,
		TakeProfit:  tpx,
	}
	_, err = s.store.CreateExecutionInstruction(ctx, inst)
	return err
}

// GetQueue returns candidates awaiting_approval.
func (s *Service) GetQueue(ctx context.Context, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListQueue(ctx, limit)
}

// GetByCandidate returns the latest approval for a candidate.
func (s *Service) GetByCandidate(ctx context.Context, candidateID uuid.UUID) (*Approval, error) {
	return s.store.GetByCandidateID(ctx, candidateID)
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrCandidateExpired    = fmt.Errorf("candidate has expired and cannot be approved")
	ErrNotAwaitingApproval = fmt.Errorf("candidate is not in awaiting_approval state")
)
