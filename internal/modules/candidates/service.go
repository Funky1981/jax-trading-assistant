package candidates

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service applies business rules on top of the Store.
type Service struct {
	store *Store
}

// NewService creates a candidate Service.
func NewService(store *Store) *Service {
	return &Service{store: store}
}

// Propose creates a detected candidate after running hard pre-qualification checks.
// Returns the created candidate, or an error if a hard check fails.
func (s *Service) Propose(ctx context.Context, req ProposalRequest) (*Candidate, error) {
	// Hard check: duplicate guard
	today := time.Now().UTC().Format("2006-01-02")
	dup, err := s.store.HasOpenForInstanceSymbol(ctx, req.StrategyInstanceID, req.Symbol, today)
	if err != nil {
		return nil, fmt.Errorf("candidates.Service.Propose dedup check: %w", err)
	}
	if dup {
		return nil, ErrDuplicateCandidate
	}

	c := &Candidate{
		StrategyInstanceID: req.StrategyInstanceID,
		Symbol:             req.Symbol,
		SignalType:         req.SignalType,
		EntryPrice:         req.EntryPrice,
		StopLoss:           req.StopLoss,
		TakeProfit:         req.TakeProfit,
		Confidence:         req.Confidence,
		Reasoning:          req.Reasoning,
		SessionDate:        today,
		DataProvenance:     req.DataProvenance,
	}
	if req.TTL > 0 {
		exp := time.Now().UTC().Add(req.TTL)
		c.ExpiresAt = &exp
	}
	return s.store.Create(ctx, c)
}

// Qualify transitions a detected candidate to qualified and then to awaiting_approval.
func (s *Service) Qualify(ctx context.Context, id uuid.UUID) error {
	if err := s.store.UpdateStatus(ctx, id, StatusQualified, nil); err != nil {
		return err
	}
	return s.store.UpdateStatus(ctx, id, StatusAwaitingApproval, nil)
}

// Block marks a candidate as blocked with a reason.
func (s *Service) Block(ctx context.Context, id uuid.UUID, reason string) error {
	return s.store.UpdateStatus(ctx, id, StatusBlocked, map[string]any{"blockReason": reason})
}

// Expire marks candidates that have passed their expiry time.
func (s *Service) ExpireStale(ctx context.Context) error {
	_, err := s.store.pool.Exec(ctx, `
		UPDATE candidate_trades
		   SET status = 'expired', updated_at = NOW()
		 WHERE status IN ('detected','qualified','awaiting_approval')
		   AND expires_at IS NOT NULL
		   AND expires_at < NOW()`)
	return err
}

// GetByID delegates to the store.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Candidate, error) {
	return s.store.GetByID(ctx, id)
}

// List delegates to the store.
func (s *Service) List(ctx context.Context, status, symbol string, limit int) ([]*Candidate, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.List(ctx, status, symbol, limit)
}

// ── Request / error types ─────────────────────────────────────────────────────

// ProposalRequest carries the input for creating a candidate.
type ProposalRequest struct {
	StrategyInstanceID uuid.UUID
	Symbol             string
	SignalType         string
	EntryPrice         *float64
	StopLoss           *float64
	TakeProfit         *float64
	Confidence         *float64
	Reasoning          *string
	DataProvenance     string
	TTL                time.Duration
}

// ErrDuplicateCandidate is returned when an open candidate already exists.
var ErrDuplicateCandidate = fmt.Errorf("open candidate already exists for this instance/symbol/session")
