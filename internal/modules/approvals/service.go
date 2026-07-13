package approvals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/instruments"
	"jax-trading-assistant/internal/modules/tradingmodule"
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

// Decide records an approval decision and, if approved, marks the candidate paper-ticket-ready without creating execution instructions.
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
		if _, err := s.ensureApprovalReviewEligible(ctx, req.CandidateID); err != nil {
			return nil, err
		}
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

	if req.Decision == DecisionApproved {
		if _, err := s.pool.Exec(ctx,
			`UPDATE candidate_trades
			    SET approval_status = $2,
			        updated_at = NOW()
			  WHERE id = $1`,
			req.CandidateID,
			candidatesmod.ApprovalStatusPaperTicketReady,
		); err != nil {
			return nil, fmt.Errorf("approvals.Service.Decide: mark paper ticket ready: %w", err)
		}
	}

	return approval, nil
}

// buildInstruction is the legacy execution-instruction bridge. The current
// roadmap approval path must not call it directly after human approval.
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

// QueueMobileApprovalNotification ensures a candidate awaiting approval has a
// one-time token and queued Telegram notification. Calls are idempotent.
func (s *Service) QueueMobileApprovalNotification(ctx context.Context, candidateID uuid.UUID) error {
	var (
		symbol     string
		strategyID string
		signalType string
		confidence float64
		reasoning  string
		entryPrice float64
		stopLoss   float64
		takeProfit float64
		expiresAt  time.Time
		status     string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT
			symbol,
			COALESCE(strategy_id, ''),
			signal_type,
			COALESCE(confidence, 0),
			COALESCE(reasoning, ''),
			COALESCE(entry_price, 0),
			COALESCE(stop_loss, 0),
			COALESCE(take_profit, 0),
			COALESCE(expires_at, NOW() + INTERVAL '10 minutes'),
			status
		FROM candidate_trades
		WHERE id = $1`, candidateID,
	).Scan(&symbol, &strategyID, &signalType, &confidence, &reasoning, &entryPrice, &stopLoss, &takeProfit, &expiresAt, &status)
	if err != nil {
		return fmt.Errorf("approvals.Service.QueueMobileApprovalNotification: candidate lookup: %w", err)
	}

	if status != candidatesmod.StatusAwaitingApproval {
		return nil
	}

	var activeTokens int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM mobile_approval_tokens
		WHERE candidate_id = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()`,
		candidateID,
	).Scan(&activeTokens)
	if err != nil {
		return fmt.Errorf("approvals.Service.QueueMobileApprovalNotification: token lookup: %w", err)
	}
	if activeTokens > 0 {
		return nil
	}

	now := time.Now().UTC()
	guardrailHash := mobileGuardrailHash(candidateID, symbol, signalType, stopLoss, s.runtimeMode)
	plainToken, tokenRecord, err := NewMobileApprovalToken(candidateID, "telegram", guardrailHash, now, 10*time.Minute)
	if err != nil {
		return fmt.Errorf("approvals.Service.QueueMobileApprovalNotification: new token: %w", err)
	}

	if strategyID == "" {
		strategyID = "unknown_strategy"
	}
	if reasoning == "" {
		reasoning = "Awaiting human review before paper execution."
	}

	notification := BuildMobileApprovalNotification(MobileApprovalSummary{
		CandidateID:   candidateID,
		Symbol:        symbol,
		Strategy:      strategyID,
		Action:        signalType,
		Confidence:    confidence,
		Why:           reasoning,
		PricedInCheck: "Review priced-in verdict in candidate evidence.",
		OtherNews:     "Check confounders in the research timeline.",
		Entry:         entryPrice,
		StopLoss:      stopLoss,
		Target:        takeProfit,
		Risk:          mobileRiskLabel(entryPrice, stopLoss),
		ExpiresAt:     expiresAt,
		PlainToken:    plainToken,
		GuardrailHash: guardrailHash,
		RuntimeMode:   s.runtimeMode,
	})

	outboxItem, err := s.store.CreateNotificationOutboxItem(ctx, &NotificationOutboxItem{
		Channel:     notification.Channel,
		CandidateID: candidateID,
		Message:     notification.Message,
		Payload: map[string]any{
			"buttons": notification.Buttons,
			"payload": notification.Payload,
		},
		Status:    "pending",
		SendAfter: now,
	})
	if err != nil {
		return fmt.Errorf("approvals.Service.QueueMobileApprovalNotification: outbox create: %w", err)
	}

	tokenRecord.NotificationID = &outboxItem.ID
	if _, err := s.store.CreateMobileApprovalToken(ctx, &tokenRecord); err != nil {
		return fmt.Errorf("approvals.Service.QueueMobileApprovalNotification: token create: %w", err)
	}

	return nil
}

func mobileGuardrailHash(candidateID uuid.UUID, symbol, signalType string, stopLoss float64, runtimeMode string) string {
	payload := fmt.Sprintf("%s|%s|%s|%.4f|%s", candidateID.String(), symbol, signalType, stopLoss, runtimeMode)
	sum := sha256.Sum256([]byte(payload))
	return "guardrails:v1:" + hex.EncodeToString(sum[:])
}

func mobileRiskLabel(entryPrice, stopLoss float64) string {
	if entryPrice <= 0 || stopLoss <= 0 {
		return "Risk requires a valid entry and stop-loss."
	}
	riskPct := math.Abs(entryPrice-stopLoss) / entryPrice * 100
	switch {
	case riskPct <= 1:
		return fmt.Sprintf("Low risk (%.2f%% to stop-loss)", riskPct)
	case riskPct <= 2.5:
		return fmt.Sprintf("Moderate risk (%.2f%% to stop-loss)", riskPct)
	default:
		return fmt.Sprintf("Elevated risk (%.2f%% to stop-loss)", riskPct)
	}
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrCandidateExpired             = fmt.Errorf("candidate has expired and cannot be approved")
	ErrNotAwaitingApproval          = fmt.Errorf("candidate is not in awaiting_approval state")
	ErrCandidateMissingSignal       = fmt.Errorf("candidate is missing signal linkage required for paper execution")
	ErrCandidateNotApprovalEligible = fmt.Errorf("candidate is not eligible for approval review")
	ErrInstrumentPolicy             = errors.New("instrument policy rejected approval")
)

func (s *Service) ensureApprovalReviewEligible(ctx context.Context, candidateID uuid.UUID) (candidatesmod.ApprovalEligibilityResult, error) {
	candidate, err := s.candidateStore.GetByID(ctx, candidateID)
	if err != nil {
		return candidatesmod.ApprovalEligibilityResult{}, fmt.Errorf("approvals.Service.ensureApprovalReviewEligible: candidate lookup: %w", err)
	}

	evidence, err := s.latestEvidenceScore(ctx, candidateID)
	if err != nil {
		return candidatesmod.ApprovalEligibilityResult{}, err
	}

	gate := candidatesmod.GateResult{
		CandidateID:                 candidateID,
		GateStatus:                  candidate.GateStatus,
		GateReady:                   candidate.GateStatus == candidatesmod.GateStatusReadyForRiskReview,
		NextRequiredPhase:           candidatesmod.NextPhaseRiskReview,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		ApprovalGranted:             false,
	}
	risk := candidatesmod.RiskReviewResult{
		CandidateID:                 candidateID.String(),
		RiskStatus:                  candidatesmod.RiskStatus(candidate.RiskStatus),
		RiskReady:                   candidatesmod.RiskStatus(candidate.RiskStatus) == candidatesmod.RiskStatusReadyForApprovalReview,
		MaxSlippageAdjustedLoss:     floatPtrValue(candidate.MaxSlippageAdjustedLoss),
		NextRequiredPhase:           candidatesmod.NextPhaseApprovalReview,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		ApprovalGranted:             false,
	}

	eligibility := candidatesmod.EvaluateApprovalEligibility(*candidate, evidence, gate, risk, time.Now().UTC())
	if !eligibility.ApprovalEligible {
		return eligibility, fmt.Errorf("%w: status=%s reject_reasons=%v warning_reasons=%v",
			ErrCandidateNotApprovalEligible,
			eligibility.ApprovalStatus,
			eligibility.RejectReasons,
			eligibility.WarningReasons,
		)
	}
	return eligibility, nil
}

func (s *Service) latestEvidenceScore(ctx context.Context, candidateID uuid.UUID) (candidatesmod.EvidenceScoreSummary, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT support_score::float8, contradiction_score::float8, quality_score::float8,
		       freshness_score::float8, overall_evidence_score::float8,
		       evidence_item_count, supporting_item_count, contradictory_item_count, stale_item_count,
		       evidence_status, evidence_ready, evidence_gate_ready,
		       approval_granted, broker_execution_allowed, execution_instruction_created
		FROM candidate_evidence_scores
		WHERE candidate_id = $1
		ORDER BY scored_at DESC
		LIMIT 1
	`, candidateID)

	score := candidatesmod.EvidenceScoreSummary{CandidateID: candidateID}
	if err := row.Scan(
		&score.SupportScore,
		&score.ContradictionScore,
		&score.QualityScore,
		&score.FreshnessScore,
		&score.OverallEvidenceScore,
		&score.EvidenceItemCount,
		&score.SupportingItemCount,
		&score.ContradictoryItemCount,
		&score.StaleItemCount,
		&score.EvidenceStatus,
		&score.EvidenceReady,
		&score.EvidenceGateReady,
		&score.ApprovalGranted,
		&score.BrokerExecutionAllowed,
		&score.ExecutionInstructionCreated,
	); err != nil {
		if isNoRows(err) {
			return score, fmt.Errorf("%w: evidence_score_missing", ErrCandidateNotApprovalEligible)
		}
		return score, fmt.Errorf("approvals.Service.latestEvidenceScore: %w", err)
	}
	return score, nil
}

func floatPtrValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

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
	if !tradingmodule.IsETF(tradingmodule.ResolveFromSymbol(s.instrumentGate, symbol)) {
		return nil
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
