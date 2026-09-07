package portfoliorisk

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

var knownReasonCodes = map[ReasonCode]bool{
	ReasonAcceptWithinPolicy: true, ReasonAmendPositionCap: true, ReasonAmendRiskBudget: true,
	ReasonRejectMissingPortfolio: true, ReasonRejectStalePortfolio: true, ReasonRejectUnknownExposure: true,
	ReasonRejectInvalidPolicy: true, ReasonRejectUnsupportedCurrency: true, ReasonRejectConcentrationLimit: true,
	ReasonRejectGrossLimit: true, ReasonRejectNetLimit: true, ReasonRejectInsufficientCapital: true,
	ReasonRejectLeverageLimit: true, ReasonRejectInvalidRecommendation: true, ReasonRejectExecutionAuthority: true,
	ReasonRejectNoCapacity: true,
}

func (decision RiskDecision) Validate() error {
	if strings.TrimSpace(decision.DecisionID) == "" || decision.Algorithm != RiskDecisionAlgorithmV1 {
		return fmt.Errorf("invalid risk decision identity")
	}
	if decision.Outcome != DecisionAccept && decision.Outcome != DecisionAmend && decision.Outcome != DecisionReject {
		return fmt.Errorf("invalid risk decision outcome")
	}
	if decision.ExecutionAuthority != "NONE" {
		return fmt.Errorf("risk decision cannot grant execution authority")
	}
	if (decision.Outcome == DecisionAccept || decision.Outcome == DecisionAmend) && (decision.RecommendationID == "" || decision.PortfolioSnapshotID == "" || decision.AnalyticsID == "" || decision.PolicyID == "") {
		return fmt.Errorf("non-reject risk decision requires complete input identities")
	}
	if len(decision.ReasonCodes) == 0 || len(decision.Explanations) == 0 {
		return fmt.Errorf("risk decision requires reason codes and explanations")
	}
	seen := make(map[ReasonCode]bool)
	for _, code := range decision.ReasonCodes {
		if !knownReasonCodes[code] || seen[code] {
			return fmt.Errorf("risk decision has unknown or duplicate reason code %q", code)
		}
		seen[code] = true
		if decision.Outcome == DecisionAccept && code != ReasonAcceptWithinPolicy {
			return fmt.Errorf("accept has non-accept reason code %q", code)
		}
		if decision.Outcome == DecisionAmend && !strings.HasPrefix(string(code), "AMEND_") {
			return fmt.Errorf("amend has non-amend reason code %q", code)
		}
		if decision.Outcome == DecisionReject && !strings.HasPrefix(string(code), "REJECT_") {
			return fmt.Errorf("reject has non-reject reason code %q", code)
		}
	}
	if !finite(decision.RequestedValue) || !finite(decision.ResultingValue) {
		return fmt.Errorf("risk decision values must be finite")
	}
	if decision.DecisionID != decisionIdentity(decision) {
		return fmt.Errorf("risk decision identity does not match content")
	}
	return nil
}

type RiskDecisionStore interface {
	Save(context.Context, RiskDecision) error
	Get(context.Context, string) (RiskDecision, error)
}

type MemoryRiskDecisionStore struct {
	mu    sync.RWMutex
	items map[string]RiskDecision
}

func NewMemoryRiskDecisionStore() *MemoryRiskDecisionStore {
	return &MemoryRiskDecisionStore{items: make(map[string]RiskDecision)}
}
func (s *MemoryRiskDecisionStore) Save(_ context.Context, decision RiskDecision) error {
	if err := decision.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.items[decision.DecisionID]; ok {
		if stringify(existing) != stringify(decision) {
			return ErrAuditConflict
		}
		return nil
	}
	s.items[decision.DecisionID] = decision
	return nil
}
func (s *MemoryRiskDecisionStore) Get(_ context.Context, id string) (RiskDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	decision, ok := s.items[strings.TrimSpace(id)]
	if !ok {
		return RiskDecision{}, sql.ErrNoRows
	}
	return decision, nil
}

var ErrAuditConflict = fmt.Errorf("risk decision audit identity already contains different content")

type PostgresRiskDecisionStore struct{ db *sql.DB }

func NewPostgresRiskDecisionStore(db *sql.DB) (*PostgresRiskDecisionStore, error) {
	if db == nil {
		return nil, fmt.Errorf("risk decision store: db is nil")
	}
	return &PostgresRiskDecisionStore{db: db}, nil
}
func (s *PostgresRiskDecisionStore) Save(ctx context.Context, decision RiskDecision) error {
	if err := decision.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("risk decision audit: encode: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO portfolio_risk_decisions (decision_id, recommendation_id, portfolio_snapshot_id, analytics_id, policy_id, algorithm, outcome, reason_codes, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now()) ON CONFLICT (decision_id) DO NOTHING`, decision.DecisionID, decision.RecommendationID, decision.PortfolioSnapshotID, decision.AnalyticsID, decision.PolicyID, decision.Algorithm, decision.Outcome, reasonStrings(decision.ReasonCodes), payload)
	if err != nil {
		return fmt.Errorf("risk decision audit: save: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		var existing []byte
		if err := s.db.QueryRowContext(ctx, `SELECT payload FROM portfolio_risk_decisions WHERE decision_id=$1`, decision.DecisionID).Scan(&existing); err != nil {
			return err
		}
		var stored RiskDecision
		if err := json.Unmarshal(existing, &stored); err != nil {
			return err
		}
		if stringify(stored) != stringify(decision) {
			return ErrAuditConflict
		}
	}
	return nil
}
func (s *PostgresRiskDecisionStore) Get(ctx context.Context, id string) (RiskDecision, error) {
	var payload []byte
	if err := s.db.QueryRowContext(ctx, `SELECT payload FROM portfolio_risk_decisions WHERE decision_id=$1`, id).Scan(&payload); err != nil {
		return RiskDecision{}, err
	}
	var decision RiskDecision
	if err := json.Unmarshal(payload, &decision); err != nil {
		return RiskDecision{}, err
	}
	return decision, decision.Validate()
}
func reasonStrings(codes []ReasonCode) []string {
	values := make([]string, len(codes))
	for i, code := range codes {
		values[i] = string(code)
	}
	return values
}
func stringify(value any) string { b, _ := json.Marshal(value); return string(b) }
