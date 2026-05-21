package candidates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"jax-trading-assistant/internal/modules/instruments"
	"jax-trading-assistant/internal/modules/tradingmodule"
)

// Service applies business rules on top of the Store.
type Service struct {
	store          *Store
	instrumentGate *instruments.Catalog
	runtimeMode    string
}

// NewService creates a candidate Service.
func NewService(store *Store) *Service {
	svc := &Service{store: store, runtimeMode: "paper"}
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

// Propose creates a detected candidate after running hard pre-qualification checks.
// Returns the created candidate, or an error if a hard check fails.
func (s *Service) Propose(ctx context.Context, req ProposalRequest) (*Candidate, error) {
	module := tradingmodule.ResolveFromSymbol(s.instrumentGate, req.Symbol)
	if result, gated := s.evaluateETF(req.Symbol); gated && !result.Allowed {
		return nil, instrumentPolicyError{result: result}
	}

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
	if parsed := parseOptionalUUID(req.SignalID); parsed != nil {
		c.SignalID = parsed
	}
	if req.StrategyID != "" {
		c.StrategyID = &req.StrategyID
	}
	if parsed := parseOptionalUUID(req.ArtifactID); parsed != nil {
		c.ArtifactID = parsed
	}
	if req.TTL > 0 {
		exp := time.Now().UTC().Add(req.TTL)
		c.ExpiresAt = &exp
	}
	c.Metadata = metadataWithModule(c.Metadata, module)
	if result, gated := s.evaluateETF(req.Symbol); gated {
		c.Metadata = metadataWithETFResult(c.Metadata, result)
	}
	return s.store.Create(ctx, c)
}

// Qualify transitions a detected candidate to qualified and then to awaiting_approval.
func (s *Service) Qualify(ctx context.Context, id uuid.UUID) error {
	candidate, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	module := tradingmodule.ResolveFromSymbol(s.instrumentGate, candidate.Symbol)
	if result, gated := s.evaluateETF(candidate.Symbol); gated {
		if !result.Allowed {
			return s.store.UpdateStatus(ctx, id, StatusBlocked, map[string]any{
				"blockReason":       result.Reason,
				"blockedReasonCode": result.ReasonCode,
				"etfPolicy":         result.Metadata,
				"tradingModule":     module,
			})
		}
		if candidate.StopLoss == nil || *candidate.StopLoss <= 0 {
			result.Allowed = false
			result.ReasonCode = instruments.ReasonStopLossRequired
			result.Reason = "ETF candidates require a stop loss before approval."
			return s.store.UpdateStatus(ctx, id, StatusBlocked, map[string]any{
				"blockReason":       result.Reason,
				"blockedReasonCode": result.ReasonCode,
				"etfPolicy":         result.Metadata,
				"tradingModule":     module,
			})
		}
	}
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
	SignalID           string
	StrategyID         string
	ArtifactID         string
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

type BlockRequest struct {
	StrategyInstanceID uuid.UUID
	SignalID           string
	StrategyID         string
	ArtifactID         string
	Symbol             string
	SignalType         string
	EntryPrice         *float64
	StopLoss           *float64
	TakeProfit         *float64
	Confidence         *float64
	Reasoning          *string
	DataProvenance     string
	ReasonCode         string
	Reason             string
	TTL                time.Duration
}

func (s *Service) CreateBlocked(ctx context.Context, req BlockRequest) (*Candidate, error) {
	module := tradingmodule.ResolveFromSymbol(s.instrumentGate, req.Symbol)
	candidate := &Candidate{
		StrategyInstanceID: req.StrategyInstanceID,
		Symbol:             req.Symbol,
		SignalType:         req.SignalType,
		EntryPrice:         req.EntryPrice,
		StopLoss:           req.StopLoss,
		TakeProfit:         req.TakeProfit,
		Confidence:         req.Confidence,
		Reasoning:          req.Reasoning,
		SessionDate:        time.Now().UTC().Format("2006-01-02"),
		DataProvenance:     req.DataProvenance,
	}
	if req.SignalID != "" {
		candidate.SignalID = parseOptionalUUID(req.SignalID)
	}
	if req.StrategyID != "" {
		candidate.StrategyID = &req.StrategyID
	}
	if req.ArtifactID != "" {
		candidate.ArtifactID = parseOptionalUUID(req.ArtifactID)
	}
	if req.Reason != "" {
		candidate.BlockReason = &req.Reason
	}
	if req.ReasonCode != "" {
		candidate.BlockedReasonCode = &req.ReasonCode
	}
	if req.TTL > 0 {
		exp := time.Now().UTC().Add(req.TTL)
		candidate.ExpiresAt = &exp
	}
	candidate.Metadata = metadataWithModule(candidate.Metadata, module)
	if result, gated := s.evaluateETF(req.Symbol); gated {
		candidate.Metadata = metadataWithETFResult(candidate.Metadata, result)
	}
	return s.store.CreateBlocked(ctx, candidate)
}

// ErrDuplicateCandidate is returned when an open candidate already exists.
var ErrDuplicateCandidate = fmt.Errorf("open candidate already exists for this instance/symbol/session")

var ErrInstrumentPolicy = errors.New("instrument policy rejected candidate")

type instrumentPolicyError struct {
	result instruments.Evaluation
}

func (e instrumentPolicyError) Error() string {
	return fmt.Sprintf("%s: %s", ErrInstrumentPolicy, e.result.Reason)
}

func (e instrumentPolicyError) Unwrap() error {
	return ErrInstrumentPolicy
}

func InstrumentPolicyResult(err error) (instruments.Evaluation, bool) {
	var policyErr instrumentPolicyError
	if errors.As(err, &policyErr) {
		return policyErr.result, true
	}
	return instruments.Evaluation{}, false
}

func (s *Service) evaluateETF(symbol string) (instruments.Evaluation, bool) {
	if s.instrumentGate == nil {
		return instruments.Evaluation{}, false
	}
	if !s.instrumentGate.IsKnownETF(symbol) {
		return instruments.Evaluation{}, false
	}
	return s.instrumentGate.Evaluate(symbol, s.runtimeMode), true
}

func metadataWithModule(raw *json.RawMessage, module string) *json.RawMessage {
	metadata := map[string]any{}
	if raw != nil && len(*raw) > 0 {
		_ = json.Unmarshal(*raw, &metadata)
	}
	metadata["tradingModule"] = module
	data, _ := json.Marshal(metadata)
	msg := json.RawMessage(data)
	return &msg
}

func metadataWithETFResult(raw *json.RawMessage, result instruments.Evaluation) *json.RawMessage {
	metadata := map[string]any{}
	if raw != nil && len(*raw) > 0 {
		_ = json.Unmarshal(*raw, &metadata)
	}
	metadata["etfPolicy"] = map[string]any{
		"symbol":         result.Symbol,
		"allowed":        result.Allowed,
		"reasonCode":     result.ReasonCode,
		"reason":         result.Reason,
		"catalogVersion": result.CatalogVersion,
		"catalogHash":    result.CatalogHash,
		"metadata":       result.Metadata,
	}
	data, _ := json.Marshal(metadata)
	msg := json.RawMessage(data)
	return &msg
}

func parseOptionalUUID(raw string) *uuid.UUID {
	if raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &parsed
}
