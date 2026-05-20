package approvals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/instruments"
)

// Service applies approval business rules.
type Service struct {
	store          *Store
	pool           *pgxpool.Pool
	candidateStore *candidatesmod.Store
	instrumentGate *instruments.Catalog
	runtimeMode    string
}

// NewService creates an approval Service.
func NewService(pool *pgxpool.Pool) *Service {
	svc := &Service{
		store:          NewStore(pool),
		pool:           pool,
		candidateStore: candidatesmod.NewStore(pool),
		runtimeMode:    "paper",
	}
	if catalog, err := instruments.LoadDefaultCatalog(); err == nil {
		svc.instrumentGate = catalog
	}
	return svc
}

func (s *Service) WithInstrumentPolicy(catalog *instruments.Catalog, runtimeMode string) *Service {
	s.instrumentGate = catalog
	if runtimeMode != "" {
		s.runtimeMode = runtimeMode
	}
	return s
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
	var signalID *uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT status, expires_at, signal_id FROM candidate_trades WHERE id = $1`, req.CandidateID,
	).Scan(&status, &expiresAt, &signalID)
	if err != nil {
		return nil, fmt.Errorf("approvals.Service.Decide: candidate lookup: %w", err)
	}
	if expiresAt != nil && time.Now().UTC().After(*expiresAt) {
		return nil, ErrCandidateExpired
	}
	if status != "awaiting_approval" {
		return nil, fmt.Errorf("%w: status=%s", ErrNotAwaitingApproval, status)
	}
	if req.Decision == DecisionApproved && signalID == nil {
		return nil, ErrCandidateMissingSignal
	}
	if req.Decision == DecisionApproved {
		if err := s.checkETFApprovalGate(ctx, req.CandidateID); err != nil {
			return nil, err
		}
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

	if req.Decision == DecisionSnoozed || req.Decision == DecisionReanalysisRequested {
		_, err = s.pool.Exec(ctx,
			`UPDATE candidate_trades SET updated_at = NOW() WHERE id = $1`,
			req.CandidateID,
		)
	} else {
		err = s.candidateStore.UpdateStatus(ctx, req.CandidateID, newCandidateStatus, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("approvals.Service.Decide: update candidate status: %w", err)
	}

	if signalID != nil {
		if err := s.syncSignalDecision(ctx, *signalID, req); err != nil {
			return nil, fmt.Errorf("approvals.Service.Decide: sync signal decision: %w", err)
		}
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
		symbol, signalType string
		signalID           *uuid.UUID
		entryPrice         *float64
		stopLoss           *float64
		tpx                *float64
	)
	err := s.pool.QueryRow(ctx,
		`SELECT signal_id, symbol, signal_type, entry_price, stop_loss, take_profit
		   FROM candidate_trades WHERE id = $1`, approval.CandidateID,
	).Scan(&signalID, &symbol, &signalType, &entryPrice, &stopLoss, &tpx)
	if err != nil {
		return fmt.Errorf("buildInstruction lookup candidate: %w", err)
	}
	if signalID == nil {
		return ErrCandidateMissingSignal
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

// GetByCandidate returns approval detail for a candidate including execution status.
func (s *Service) GetByCandidate(ctx context.Context, candidateID uuid.UUID) (*ApprovalDetail, error) {
	return s.store.GetDetailByCandidateID(ctx, candidateID)
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrCandidateExpired       = fmt.Errorf("candidate has expired and cannot be approved")
	ErrNotAwaitingApproval    = fmt.Errorf("candidate is not in awaiting_approval state")
	ErrCandidateMissingSignal = fmt.Errorf("candidate is missing signal linkage required for paper execution")
	ErrInstrumentPolicy       = errors.New("instrument policy rejected approval")
)

func (s *Service) checkETFApprovalGate(ctx context.Context, candidateID uuid.UUID) error {
	if s.instrumentGate == nil {
		return nil
	}
	var symbol string
	var stopLoss *float64
	err := s.pool.QueryRow(ctx, `SELECT symbol, stop_loss FROM candidate_trades WHERE id = $1`, candidateID).Scan(&symbol, &stopLoss)
	if err != nil {
		return err
	}
	result := s.instrumentGate.Evaluate(symbol, s.runtimeMode)
	if !result.Allowed {
		return fmt.Errorf("%w: %s: %s", ErrInstrumentPolicy, result.ReasonCode, result.Reason)
	}
	if stopLoss == nil || *stopLoss <= 0 {
		return fmt.Errorf("%w: %s: ETF candidates require a stop loss before approval", ErrInstrumentPolicy, instruments.ReasonStopLossRequired)
	}
	return nil
}

func (s *Service) syncSignalDecision(ctx context.Context, signalID uuid.UUID, req ApprovalRequest) error {
	switch req.Decision {
	case DecisionApproved:
		if _, err := s.pool.Exec(ctx, `
			UPDATE strategy_signals
			SET status = 'approved'
			WHERE id = $1
			  AND status IN ('pending','approved')
		`, signalID); err != nil {
			return err
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO trade_approvals (signal_id, orchestration_run_id, approved, approved_at, approved_by, modification_notes)
			SELECT id, orchestration_run_id, TRUE, NOW(), $2, $3
			FROM strategy_signals
			WHERE id = $1
		`, signalID, req.ApprovedBy, req.Notes)
		return err
	case DecisionRejected:
		if _, err := s.pool.Exec(ctx, `
			UPDATE strategy_signals
			SET status = 'rejected'
			WHERE id = $1
			  AND status IN ('pending','approved','rejected')
		`, signalID); err != nil {
			return err
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO trade_approvals (signal_id, orchestration_run_id, approved, approved_at, approved_by, modification_notes)
			SELECT id, orchestration_run_id, FALSE, NOW(), $2, $3
			FROM strategy_signals
			WHERE id = $1
		`, signalID, req.ApprovedBy, req.Notes)
		return err
	default:
		return nil
	}
}
