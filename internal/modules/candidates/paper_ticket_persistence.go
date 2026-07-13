package candidates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrPaperTicketNotPersistable = errors.New("paper ticket boundary result is not persistable")

type PaperTicket struct {
	ID                          uuid.UUID  `json:"id"`
	PaperTicketID               string     `json:"paperTicketId"`
	CandidateID                 uuid.UUID  `json:"candidateId"`
	CreatedAt                   time.Time  `json:"createdAt"`
	UpdatedAt                   time.Time  `json:"updatedAt"`
	Status                      string     `json:"status"`
	SourceApprovalID            *uuid.UUID `json:"sourceApprovalId,omitempty"`
	ApprovalDecisionRef         string     `json:"approvalDecisionRef,omitempty"`
	Symbol                      string     `json:"symbol"`
	Direction                   string     `json:"direction"`
	SetupType                   string     `json:"setupType"`
	CatalystSummary             string     `json:"catalystSummary"`
	EntryPrice                  float64    `json:"entryPrice"`
	StopLossPrice               float64    `json:"stopLossPrice"`
	TargetPrice                 float64    `json:"targetPrice"`
	PositionSize                float64    `json:"positionSize"`
	MaxNormalLoss               float64    `json:"maxNormalLoss"`
	MaxSlippageAdjustedLoss     float64    `json:"maxSlippageAdjustedLoss"`
	RewardRiskRatio             float64    `json:"rewardRiskRatio"`
	EvidenceStatus              string     `json:"evidenceStatus"`
	GateStatus                  string     `json:"gateStatus"`
	RiskStatus                  string     `json:"riskStatus"`
	ApprovalStatus              string     `json:"approvalStatus"`
	PaperOnly                   bool       `json:"paperOnly"`
	BrokerExecutionAllowed      bool       `json:"brokerExecutionAllowed"`
	ExecutionInstructionCreated bool       `json:"executionInstructionCreated"`
	LiveTradingAllowed          bool       `json:"liveTradingAllowed"`
	LeverageAllowed             bool       `json:"leverageAllowed"`
	RejectReasons               []string   `json:"rejectReasons,omitempty"`
	WarningReasons              []string   `json:"warningReasons,omitempty"`
}

type PaperTicketReview struct {
	PaperTicketID           string    `json:"paperTicketId"`
	CandidateID             uuid.UUID `json:"candidateId"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
	Status                  string    `json:"status"`
	Symbol                  string    `json:"symbol"`
	Direction               string    `json:"direction"`
	SetupType               string    `json:"setupType"`
	CatalystSummary         string    `json:"catalystSummary"`
	EntryPrice              float64   `json:"entryPrice"`
	StopLossPrice           float64   `json:"stopLossPrice"`
	TargetPrice             float64   `json:"targetPrice"`
	PositionSize            float64   `json:"positionSize"`
	MaxNormalLoss           float64   `json:"maxNormalLoss"`
	MaxSlippageAdjustedLoss float64   `json:"maxSlippageAdjustedLoss"`
	RewardRiskRatio         float64   `json:"rewardRiskRatio"`
	EvidenceStatus          string    `json:"evidenceStatus"`
	GateStatus              string    `json:"gateStatus"`
	RiskStatus              string    `json:"riskStatus"`
	ApprovalStatus          string    `json:"approvalStatus"`
	PaperOnly               bool      `json:"paperOnly"`
	RejectReasons           []string  `json:"rejectReasons,omitempty"`
	WarningReasons          []string  `json:"warningReasons,omitempty"`
	ReviewNotes             string    `json:"reviewNotes,omitempty"`
}

func NewPersistedPaperTicket(candidate Candidate, evidence EvidenceScoreSummary, eligibility ApprovalEligibilityResult, result PaperTicketResult) (PaperTicket, error) {
	if !result.CanCreateTicket {
		return PaperTicket{}, fmt.Errorf("%w: status=%s reject_reasons=%v", ErrPaperTicketNotPersistable, result.Status, result.RejectReasons)
	}
	if result.PaperTicketID == "" || result.CandidateID == uuid.Nil {
		return PaperTicket{}, fmt.Errorf("%w: missing identity", ErrPaperTicketNotPersistable)
	}
	now := result.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return PaperTicket{
		ID:                          uuid.New(),
		PaperTicketID:               result.PaperTicketID,
		CandidateID:                 result.CandidateID,
		CreatedAt:                   now,
		UpdatedAt:                   now,
		Status:                      PaperTicketStatusPaperTicketCreated,
		SourceApprovalID:            result.SourceApprovalID,
		ApprovalDecisionRef:         result.ApprovalDecisionRef,
		Symbol:                      candidate.Symbol,
		Direction:                   candidate.Direction,
		SetupType:                   candidate.SetupType,
		CatalystSummary:             candidate.CatalystSummary,
		EntryPrice:                  result.EntryPrice,
		StopLossPrice:               result.StopLossPrice,
		TargetPrice:                 result.TargetPrice,
		PositionSize:                result.PositionSize,
		MaxNormalLoss:               result.MaxNormalLoss,
		MaxSlippageAdjustedLoss:     result.MaxSlippageAdjustedLoss,
		RewardRiskRatio:             result.RewardRiskRatio,
		EvidenceStatus:              string(evidence.EvidenceStatus),
		GateStatus:                  candidate.GateStatus,
		RiskStatus:                  candidate.RiskStatus,
		ApprovalStatus:              eligibility.ApprovalStatus,
		PaperOnly:                   true,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		LiveTradingAllowed:          false,
		LeverageAllowed:             false,
		RejectReasons:               append([]string{}, result.RejectReasons...),
		WarningReasons:              append([]string{}, eligibility.WarningReasons...),
	}, nil
}

func (t PaperTicket) ReviewModel() PaperTicketReview {
	return PaperTicketReview{
		PaperTicketID:           t.PaperTicketID,
		CandidateID:             t.CandidateID,
		CreatedAt:               t.CreatedAt,
		UpdatedAt:               t.UpdatedAt,
		Status:                  t.Status,
		Symbol:                  t.Symbol,
		Direction:               t.Direction,
		SetupType:               t.SetupType,
		CatalystSummary:         t.CatalystSummary,
		EntryPrice:              t.EntryPrice,
		StopLossPrice:           t.StopLossPrice,
		TargetPrice:             t.TargetPrice,
		PositionSize:            t.PositionSize,
		MaxNormalLoss:           t.MaxNormalLoss,
		MaxSlippageAdjustedLoss: t.MaxSlippageAdjustedLoss,
		RewardRiskRatio:         t.RewardRiskRatio,
		EvidenceStatus:          t.EvidenceStatus,
		GateStatus:              t.GateStatus,
		RiskStatus:              t.RiskStatus,
		ApprovalStatus:          t.ApprovalStatus,
		PaperOnly:               t.PaperOnly,
		RejectReasons:           append([]string{}, t.RejectReasons...),
		WarningReasons:          append([]string{}, t.WarningReasons...),
	}
}

func (s *Store) CreatePaperTicket(ctx context.Context, ticket PaperTicket) (*PaperTicket, error) {
	if ticket.ID == uuid.Nil {
		ticket.ID = uuid.New()
	}
	now := time.Now().UTC()
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = now
	}
	ticket.UpdatedAt = now
	if ticket.Status == "" {
		ticket.Status = PaperTicketStatusPaperTicketCreated
	}
	ticket.PaperOnly = true
	ticket.BrokerExecutionAllowed = false
	ticket.ExecutionInstructionCreated = false
	ticket.LiveTradingAllowed = false
	ticket.LeverageAllowed = false

	_, err := s.pool.Exec(ctx, `
		INSERT INTO candidate_paper_tickets (
			id, paper_ticket_id, candidate_id, created_at, updated_at, status,
			source_approval_id, approval_decision_ref, symbol, direction, setup_type, catalyst_summary,
			entry_price, stop_loss_price, target_price, position_size, max_normal_loss,
			max_slippage_adjusted_loss, reward_risk_ratio, evidence_status, gate_status,
			risk_status, approval_status, paper_only, broker_execution_allowed,
			execution_instruction_created, live_trading_allowed, leverage_allowed,
			reject_reasons, warning_reasons
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,$12,
			$13,$14,$15,$16,$17,
			$18,$19,$20,$21,
			$22,$23,$24,$25,
			$26,$27,$28,
			$29,$30
		)
		ON CONFLICT (candidate_id) DO UPDATE
		SET updated_at = EXCLUDED.updated_at,
		    status = EXCLUDED.status,
		    source_approval_id = EXCLUDED.source_approval_id,
		    approval_decision_ref = EXCLUDED.approval_decision_ref,
		    reject_reasons = EXCLUDED.reject_reasons,
		    warning_reasons = EXCLUDED.warning_reasons
	`, ticket.ID, ticket.PaperTicketID, ticket.CandidateID, ticket.CreatedAt, ticket.UpdatedAt, ticket.Status,
		ticket.SourceApprovalID, ticket.ApprovalDecisionRef, ticket.Symbol, ticket.Direction, ticket.SetupType, ticket.CatalystSummary,
		ticket.EntryPrice, ticket.StopLossPrice, ticket.TargetPrice, ticket.PositionSize, ticket.MaxNormalLoss,
		ticket.MaxSlippageAdjustedLoss, ticket.RewardRiskRatio, ticket.EvidenceStatus, ticket.GateStatus,
		ticket.RiskStatus, ticket.ApprovalStatus, ticket.PaperOnly, ticket.BrokerExecutionAllowed,
		ticket.ExecutionInstructionCreated, ticket.LiveTradingAllowed, ticket.LeverageAllowed,
		ticket.RejectReasons, ticket.WarningReasons)
	if err != nil {
		return nil, fmt.Errorf("candidates.Store.CreatePaperTicket: %w", err)
	}
	return &ticket, nil
}

func (s *Store) GetPaperTicketReviewByCandidateID(ctx context.Context, candidateID uuid.UUID) (*PaperTicketReview, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT paper_ticket_id, candidate_id, created_at, updated_at, status,
		       symbol, direction, setup_type, catalyst_summary, entry_price::float8,
		       stop_loss_price::float8, target_price::float8, position_size::float8,
		       max_normal_loss::float8, max_slippage_adjusted_loss::float8,
		       reward_risk_ratio::float8, evidence_status, gate_status, risk_status,
		       approval_status, paper_only, reject_reasons, warning_reasons, COALESCE(review_notes, '')
		FROM candidate_paper_tickets
		WHERE candidate_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, candidateID)

	var review PaperTicketReview
	err := row.Scan(
		&review.PaperTicketID,
		&review.CandidateID,
		&review.CreatedAt,
		&review.UpdatedAt,
		&review.Status,
		&review.Symbol,
		&review.Direction,
		&review.SetupType,
		&review.CatalystSummary,
		&review.EntryPrice,
		&review.StopLossPrice,
		&review.TargetPrice,
		&review.PositionSize,
		&review.MaxNormalLoss,
		&review.MaxSlippageAdjustedLoss,
		&review.RewardRiskRatio,
		&review.EvidenceStatus,
		&review.GateStatus,
		&review.RiskStatus,
		&review.ApprovalStatus,
		&review.PaperOnly,
		&review.RejectReasons,
		&review.WarningReasons,
	)
	if err != nil {
		return nil, fmt.Errorf("candidates.Store.GetPaperTicketReviewByCandidateID: %w", err)
	}
	return &review, nil
}

func (s *Store) ListPaperTicketReviews(ctx context.Context, limit int) ([]PaperTicketReview, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT paper_ticket_id, candidate_id, created_at, updated_at, status,
		       symbol, direction, setup_type, catalyst_summary, entry_price::float8,
		       stop_loss_price::float8, target_price::float8, position_size::float8,
		       max_normal_loss::float8, max_slippage_adjusted_loss::float8,
		       reward_risk_ratio::float8, evidence_status, gate_status, risk_status,
		       approval_status, paper_only, reject_reasons, warning_reasons, COALESCE(review_notes, '')
		FROM candidate_paper_tickets
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("candidates.Store.ListPaperTicketReviews: %w", err)
	}
	defer rows.Close()

	var reviews []PaperTicketReview
	for rows.Next() {
		review, err := scanPaperTicketReview(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("candidates.Store.ListPaperTicketReviews: %w", err)
		}
		reviews = append(reviews, review)
	}
	return reviews, rows.Err()
}

func (s *Store) MarkPaperTicketReviewed(ctx context.Context, paperTicketID, note string) (*PaperTicketReview, error) {
	return s.updatePaperTicketReview(ctx, paperTicketID, note, PaperTicketStatusPaperTicketReviewed, []string{
		PaperTicketStatusPaperTicketCreated,
		PaperTicketStatusPaperTicketReady,
	})
}

func (s *Store) CancelPaperTicketReview(ctx context.Context, paperTicketID, note string) (*PaperTicketReview, error) {
	return s.updatePaperTicketReview(ctx, paperTicketID, note, PaperTicketStatusPaperTicketCancelled, []string{
		PaperTicketStatusPaperTicketCreated,
		PaperTicketStatusPaperTicketReady,
	})
}

func (s *Store) AddPaperTicketReviewNote(ctx context.Context, paperTicketID, note string) (*PaperTicketReview, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE candidate_paper_tickets
		SET updated_at = NOW(),
		    review_notes = append_review_note(review_notes, $2)
		WHERE paper_ticket_id = $1
		  AND status <> 'paper_ticket_cancelled'
		RETURNING paper_ticket_id, candidate_id, created_at, updated_at, status,
		       symbol, direction, setup_type, catalyst_summary, entry_price::float8,
		       stop_loss_price::float8, target_price::float8, position_size::float8,
		       max_normal_loss::float8, max_slippage_adjusted_loss::float8,
		       reward_risk_ratio::float8, evidence_status, gate_status, risk_status,
		       approval_status, paper_only, reject_reasons, warning_reasons, COALESCE(review_notes, '')
	`, paperTicketID, note)
	review, err := scanPaperTicketReview(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("candidates.Store.AddPaperTicketReviewNote: %w", err)
	}
	return &review, nil
}

func (s *Store) updatePaperTicketReview(ctx context.Context, paperTicketID, note, nextStatus string, allowedStatuses []string) (*PaperTicketReview, error) {
	if len(allowedStatuses) != 2 {
		return nil, fmt.Errorf("candidates.Store.updatePaperTicketReview: exactly two allowed statuses required")
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE candidate_paper_tickets
		SET status = $2,
		    updated_at = NOW(),
		    review_notes = append_review_note(review_notes, $5)
		WHERE paper_ticket_id = $1
		  AND status IN ($3, $4)
		RETURNING paper_ticket_id, candidate_id, created_at, updated_at, status,
		       symbol, direction, setup_type, catalyst_summary, entry_price::float8,
		       stop_loss_price::float8, target_price::float8, position_size::float8,
		       max_normal_loss::float8, max_slippage_adjusted_loss::float8,
		       reward_risk_ratio::float8, evidence_status, gate_status, risk_status,
		       approval_status, paper_only, reject_reasons, warning_reasons, COALESCE(review_notes, '')
	`, paperTicketID, nextStatus, allowedStatuses[0], allowedStatuses[1], note)
	review, err := scanPaperTicketReview(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("candidates.Store.updatePaperTicketReview: %w", err)
	}
	return &review, nil
}

type paperTicketReviewScanner func(dest ...any) error

func scanPaperTicketReview(scan paperTicketReviewScanner) (PaperTicketReview, error) {
	var review PaperTicketReview
	err := scan(
		&review.PaperTicketID,
		&review.CandidateID,
		&review.CreatedAt,
		&review.UpdatedAt,
		&review.Status,
		&review.Symbol,
		&review.Direction,
		&review.SetupType,
		&review.CatalystSummary,
		&review.EntryPrice,
		&review.StopLossPrice,
		&review.TargetPrice,
		&review.PositionSize,
		&review.MaxNormalLoss,
		&review.MaxSlippageAdjustedLoss,
		&review.RewardRiskRatio,
		&review.EvidenceStatus,
		&review.GateStatus,
		&review.RiskStatus,
		&review.ApprovalStatus,
		&review.PaperOnly,
		&review.RejectReasons,
		&review.WarningReasons,
		&review.ReviewNotes,
	)
	return review, err
}
