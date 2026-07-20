// Package approvals manages the human approval workflow for candidate trades.
package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
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
	TradeID       *string    `json:"tradeId,omitempty"`
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
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type ApprovalDetail struct {
	CandidateID    uuid.UUID                        `json:"candidateId"`
	State          string                           `json:"state"`
	LatestApproval *Approval                        `json:"latestApproval,omitempty"`
	PaperTicket    *candidatesmod.PaperTicketReview `json:"paperTicket,omitempty"`
	Execution      *ExecutionInstruction            `json:"execution,omitempty"`
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
		       ct.detected_at, ct.expires_at, si.name AS instance_name, ct.metadata
		FROM candidate_trades ct
		LEFT JOIN strategy_instances si ON si.id = ct.strategy_instance_id
		JOIN LATERAL (
			SELECT evidence_status, evidence_ready, evidence_gate_ready,
			       broker_execution_allowed, execution_instruction_created, approval_granted
			FROM candidate_evidence_scores
			WHERE candidate_id = ct.id
			ORDER BY scored_at DESC
			LIMIT 1
		) es ON TRUE
		WHERE ct.status = 'awaiting_approval'
		  AND ct.gate_status = 'ready_for_risk_review'
		  AND ct.risk_status = 'ready_for_approval_review'
		  AND ct.approval_status IN ('not_ready', 'approval_review_ready')
		  AND ct.human_approval_required = TRUE
		  AND es.evidence_status = 'sufficient'
		  AND es.evidence_ready = TRUE
		  AND es.evidence_gate_ready = TRUE
		  AND es.broker_execution_allowed = FALSE
		  AND es.execution_instruction_created = FALSE
		  AND es.approval_granted = FALSE
		  AND NOT EXISTS (
			SELECT 1 FROM execution_instructions ei WHERE ei.candidate_id = ct.id
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM candidate_approvals ca
			WHERE ca.candidate_id = ct.id AND ca.decision = 'approved'
		  )
		ORDER BY ct.detected_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.ListQueue: %w", err)
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var (
			id, symbol, signalType                       string
			instanceName                                 *string
			confidence, entryPrice, stopLoss, takeProfit *float64
			reasoning, blockReason                       *string
			detectedAt                                   time.Time
			expiresAt                                    *time.Time
			metadataRaw                                  []byte
			metadata                                     map[string]any
		)
		if err := rows.Scan(
			&id, &symbol, &signalType, &confidence, &entryPrice,
			&stopLoss, &takeProfit, &reasoning, &blockReason,
			&detectedAt, &expiresAt, &instanceName, &metadataRaw,
		); err != nil {
			return nil, err
		}
		if len(metadataRaw) > 0 {
			_ = json.Unmarshal(metadataRaw, &metadata)
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
			"metadata":     metadata,
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
	inst.UpdatedAt = inst.CreatedAt
	inst.Status = "pending"
	_, err := s.pool.Exec(ctx, `
		INSERT INTO execution_instructions
			(id, approval_id, candidate_id, symbol, signal_type,
			 entry_price, stop_loss, take_profit, trade_id, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'pending',$10,$10)`,
		inst.ID, inst.ApprovalID, inst.CandidateID, inst.Symbol, inst.SignalType,
		inst.EntryPrice, inst.StopLoss, inst.TakeProfit, inst.TradeID, inst.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.CreateExecutionInstruction: %w", err)
	}
	return inst, nil
}

func (s *Store) GetLatestExecutionByCandidateID(ctx context.Context, candidateID uuid.UUID) (*ExecutionInstruction, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, approval_id, candidate_id, trade_id, symbol, signal_type,
		       entry_price, stop_loss, take_profit, status, broker_order_id,
		       fill_price, fill_qty, error_message, submitted_at, filled_at, created_at, updated_at
		FROM execution_instructions
		WHERE candidate_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, candidateID)
	return scanExecutionInstruction(row)
}

func (s *Store) GetDetailByCandidateID(ctx context.Context, candidateID uuid.UUID) (*ApprovalDetail, error) {
	detail := &ApprovalDetail{CandidateID: candidateID}
	var candidateExists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM candidate_trades WHERE id = $1)`, candidateID).Scan(&candidateExists); err != nil {
		return nil, fmt.Errorf("approvals.Store.GetDetailByCandidateID: candidate lookup: %w", err)
	}
	if !candidateExists {
		return nil, fmt.Errorf("approval detail not found")
	}
	approval, err := s.GetByCandidateID(ctx, candidateID)
	if err == nil {
		detail.LatestApproval = approval
	} else if !isNoRows(err) {
		return nil, err
	}
	execution, err := s.GetLatestExecutionByCandidateID(ctx, candidateID)
	if err == nil {
		detail.Execution = execution
	} else if !isNoRows(err) {
		return nil, err
	}
	paperTicket, err := candidatesmod.NewStore(s.pool).GetPaperTicketReviewByCandidateID(ctx, candidateID)
	if err == nil {
		detail.PaperTicket = paperTicket
	} else if !isNoRows(err) {
		return nil, err
	}
	switch {
	case detail.LatestApproval == nil && detail.PaperTicket == nil && detail.Execution == nil:
		detail.State = "no_decision"
	case detail.LatestApproval != nil && detail.LatestApproval.Decision == DecisionRejected && detail.PaperTicket == nil && detail.Execution == nil:
		detail.State = "rejected"
	case detail.LatestApproval != nil && detail.LatestApproval.Decision == DecisionApproved && detail.PaperTicket == nil && detail.Execution == nil:
		detail.State = "approval_persisted_ticket_missing"
	case detail.LatestApproval != nil && detail.LatestApproval.Decision == DecisionApproved && detail.PaperTicket != nil && detail.Execution == nil:
		detail.State = "approval_and_ticket_persisted"
	default:
		detail.State = "partial_or_inconsistent_state"
	}
	return detail, nil
}

func (s *Store) CreateNotificationOutboxItem(ctx context.Context, item *NotificationOutboxItem) (*NotificationOutboxItem, error) {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.Status == "" {
		item.Status = "pending"
	}
	if item.SendAfter.IsZero() {
		item.SendAfter = time.Now().UTC()
	}
	payload := item.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.CreateNotificationOutboxItem: marshal payload: %w", err)
	}
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	_, err = s.pool.Exec(ctx, `
		INSERT INTO notification_outbox
			(id, channel, recipient, candidate_id, message, payload, status, send_after, sent_at, error, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`,
		item.ID, item.Channel, item.Recipient, item.CandidateID, item.Message,
		payloadBytes, item.Status, item.SendAfter, item.SentAt, item.Error, item.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.CreateNotificationOutboxItem: %w", err)
	}
	item.Payload = payload
	return item, nil
}

func (s *Store) ClaimPendingNotificationOutboxItems(ctx context.Context, channel string, limit int) ([]*NotificationOutboxItem, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.pool.Query(ctx, `
		UPDATE notification_outbox
		SET status = 'sending',
		    updated_at = NOW()
		WHERE id IN (
			SELECT id
			FROM notification_outbox
			WHERE status = 'pending'
			  AND channel = $1
			  AND send_after <= NOW()
			ORDER BY send_after ASC, created_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, channel, recipient, candidate_id, message, payload, status, send_after, sent_at, error, created_at, updated_at
	`, channel, limit)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.ClaimPendingNotificationOutboxItems: %w", err)
	}
	defer rows.Close()

	out := []*NotificationOutboxItem{}
	for rows.Next() {
		var item NotificationOutboxItem
		var payloadRaw []byte
		if err := rows.Scan(
			&item.ID,
			&item.Channel,
			&item.Recipient,
			&item.CandidateID,
			&item.Message,
			&payloadRaw,
			&item.Status,
			&item.SendAfter,
			&item.SentAt,
			&item.Error,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("approvals.Store.ClaimPendingNotificationOutboxItems: scan: %w", err)
		}
		if len(payloadRaw) > 0 {
			_ = json.Unmarshal(payloadRaw, &item.Payload)
		}
		if item.Payload == nil {
			item.Payload = map[string]any{}
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func (s *Store) MarkNotificationOutboxSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	if sentAt.IsZero() {
		sentAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE notification_outbox
		SET status = 'sent',
		    sent_at = $2,
		    error = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, id, sentAt)
	if err != nil {
		return fmt.Errorf("approvals.Store.MarkNotificationOutboxSent: %w", err)
	}
	return nil
}

func (s *Store) MarkNotificationOutboxFailed(ctx context.Context, id uuid.UUID, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "notification dispatch failed"
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE notification_outbox
		SET status = 'failed',
		    error = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, id, message)
	if err != nil {
		return fmt.Errorf("approvals.Store.MarkNotificationOutboxFailed: %w", err)
	}
	return nil
}

func (s *Store) CreateMobileApprovalToken(ctx context.Context, token *MobileApprovalTokenRecord) (*MobileApprovalTokenRecord, error) {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO mobile_approval_tokens
			(id, notification_id, candidate_id, channel, token_hash, guardrail_hash, expires_at, used_at, decision, used_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		token.ID, token.NotificationID, token.CandidateID, token.Channel, token.TokenHash,
		token.GuardrailHash, token.ExpiresAt, token.UsedAt, token.Decision, token.UsedBy, token.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("approvals.Store.CreateMobileApprovalToken: %w", err)
	}
	return token, nil
}

func (s *Store) GetMobileApprovalTokenByHash(ctx context.Context, tokenHash string) (*MobileApprovalTokenRecord, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, notification_id, candidate_id, channel, token_hash, guardrail_hash,
		       expires_at, used_at, COALESCE(decision, ''), COALESCE(used_by, ''), created_at
		FROM mobile_approval_tokens
		WHERE token_hash = $1`, tokenHash)
	var token MobileApprovalTokenRecord
	if err := row.Scan(
		&token.ID, &token.NotificationID, &token.CandidateID, &token.Channel,
		&token.TokenHash, &token.GuardrailHash, &token.ExpiresAt, &token.UsedAt,
		&token.Decision, &token.UsedBy, &token.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("approvals.Store.GetMobileApprovalTokenByHash: %w", err)
	}
	return &token, nil
}

func (s *Store) MarkMobileApprovalTokenUsed(ctx context.Context, tokenID uuid.UUID, decision, usedBy string, usedAt time.Time) error {
	if usedAt.IsZero() {
		usedAt = time.Now().UTC()
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE mobile_approval_tokens
		SET used_at = $2, decision = $3, used_by = $4
		WHERE id = $1 AND used_at IS NULL`,
		tokenID, usedAt, decision, usedBy,
	)
	if err != nil {
		return fmt.Errorf("approvals.Store.MarkMobileApprovalTokenUsed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMobileApprovalUsed
	}
	return nil
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

func scanExecutionInstruction(row interface{ Scan(...any) error }) (*ExecutionInstruction, error) {
	var inst ExecutionInstruction
	err := row.Scan(
		&inst.ID, &inst.ApprovalID, &inst.CandidateID, &inst.TradeID, &inst.Symbol, &inst.SignalType,
		&inst.EntryPrice, &inst.StopLoss, &inst.TakeProfit, &inst.Status, &inst.BrokerOrderID,
		&inst.FillPrice, &inst.FillQty, &inst.ErrorMessage, &inst.SubmittedAt, &inst.FilledAt, &inst.CreatedAt, &inst.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanExecutionInstruction: %w", err)
	}
	return &inst, nil
}

func isNoRows(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no rows")
}
