package candidates

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RiskReviewPersistence struct {
	Result              RiskReviewResult          `json:"result"`
	ApprovalEligibility ApprovalEligibilityResult `json:"approvalEligibility"`
	AccountEquitySource string                    `json:"accountEquitySource"`
	SlippageSource      string                    `json:"slippageSource"`
	RiskPolicySource    string                    `json:"riskPolicySource"`
	PositionNotional    float64                   `json:"positionNotional"`
}

func (s *Store) LatestEvidenceScore(ctx context.Context, candidateID uuid.UUID) (EvidenceScoreSummary, error) {
	score := EvidenceScoreSummary{CandidateID: candidateID}
	err := s.pool.QueryRow(ctx, `
		SELECT support_score::float8, contradiction_score::float8, quality_score::float8,
		       freshness_score::float8, overall_evidence_score::float8,
		       evidence_item_count, supporting_item_count, contradictory_item_count, stale_item_count,
		       evidence_status, evidence_ready, evidence_gate_ready,
		       approval_granted, broker_execution_allowed, execution_instruction_created
		FROM candidate_evidence_scores WHERE candidate_id=$1 ORDER BY scored_at DESC LIMIT 1
	`, candidateID).Scan(&score.SupportScore, &score.ContradictionScore, &score.QualityScore,
		&score.FreshnessScore, &score.OverallEvidenceScore, &score.EvidenceItemCount,
		&score.SupportingItemCount, &score.ContradictoryItemCount, &score.StaleItemCount,
		&score.EvidenceStatus, &score.EvidenceReady, &score.EvidenceGateReady,
		&score.ApprovalGranted, &score.BrokerExecutionAllowed, &score.ExecutionInstructionCreated)
	if err != nil {
		return score, fmt.Errorf("candidates.Store.LatestEvidenceScore: %w", err)
	}
	return score, nil
}

// PersistRiskReview stores the current deterministic risk result without
// changing lifecycle status or creating approvals, tickets, or instructions.
func (s *Store) PersistRiskReview(ctx context.Context, candidate Candidate, persistence RiskReviewPersistence) error {
	payload, err := json.Marshal(persistence)
	if err != nil {
		return fmt.Errorf("candidates.Store.PersistRiskReview marshal: %w", err)
	}
	rejectReasons := append([]string{}, persistence.Result.RejectReasons...)
	rejectReasons = appendUnique(rejectReasons, persistence.ApprovalEligibility.RejectReasons...)
	_, err = s.pool.Exec(ctx, `
		UPDATE candidate_trades SET
			expected_reward_risk_ratio=$2, max_normal_loss=$3,
			max_slippage_adjusted_loss=$4, position_size=$5, risk_status=$6,
			approval_status=$7, reject_reasons=$8,
			metadata=jsonb_set(COALESCE(metadata,'{}'::jsonb), '{riskReview}', $9::jsonb, true),
			updated_at=$10
		WHERE id=$1
	`, candidate.ID, persistence.Result.RewardRiskRatio, persistence.Result.MaxNormalLoss,
		persistence.Result.MaxSlippageAdjustedLoss, persistence.Result.PositionSize,
		persistence.Result.RiskStatus, persistence.ApprovalEligibility.ApprovalStatus,
		rejectReasons, payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("candidates.Store.PersistRiskReview: %w", err)
	}
	return nil
}

// PersistEvidenceEvaluation stores the genuine evidence inputs, their computed
// score, and the resulting trust-gate decision without changing candidate
// lifecycle, approval, or execution state.
func (s *Store) PersistEvidenceEvaluation(ctx context.Context, items []EvidenceItem, score EvidenceScoreSummary, gate GateResult) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("candidates.Store.PersistEvidenceEvaluation begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO candidate_evidence_items (
				evidence_id, candidate_id, source_type, source_ref, observed_at,
				summary, evidence_kind, supports_candidate, contradicts_candidate,
				confidence, impact_score, quality_score, freshness_status, notes
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT (evidence_id) DO NOTHING
		`, item.EvidenceID, item.CandidateID, item.SourceType, item.SourceRef, item.ObservedAt,
			item.Summary, item.EvidenceKind, item.SupportsCandidate, item.ContradictsCandidate,
			item.Confidence, item.ImpactScore, item.QualityScore, item.FreshnessStatus, item.Notes); err != nil {
			return fmt.Errorf("candidates.Store.PersistEvidenceEvaluation item: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO candidate_evidence_scores (
			candidate_id, support_score, contradiction_score, quality_score,
			freshness_score, overall_evidence_score, evidence_item_count,
			supporting_item_count, contradictory_item_count, stale_item_count,
			evidence_status, evidence_ready, evidence_gate_ready,
			broker_execution_allowed, execution_instruction_created, approval_granted
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, score.CandidateID, score.SupportScore, score.ContradictionScore, score.QualityScore,
		score.FreshnessScore, score.OverallEvidenceScore, score.EvidenceItemCount,
		score.SupportingItemCount, score.ContradictoryItemCount, score.StaleItemCount,
		score.EvidenceStatus, score.EvidenceReady, score.EvidenceGateReady,
		score.BrokerExecutionAllowed, score.ExecutionInstructionCreated, score.ApprovalGranted); err != nil {
		return fmt.Errorf("candidates.Store.PersistEvidenceEvaluation score: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE candidate_trades
		SET gate_status = $2,
			supporting_evidence_summary = $3,
			contradictory_evidence_summary = $4,
			evidence_source_count = $5,
			has_contradictory_evidence = $6,
			reject_reasons = $7,
			updated_at = NOW()
		WHERE id = $1
	`, score.CandidateID, gate.GateStatus, evidenceSummary(items, true), evidenceSummary(items, false),
		score.EvidenceItemCount, score.ContradictoryItemCount > 0, gate.RejectReasons); err != nil {
		return fmt.Errorf("candidates.Store.PersistEvidenceEvaluation candidate: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("candidates.Store.PersistEvidenceEvaluation commit: %w", err)
	}
	return nil
}

func evidenceSummary(items []EvidenceItem, supporting bool) *string {
	for _, item := range items {
		matches := item.SupportsCandidate
		if !supporting {
			matches = item.ContradictsCandidate
		}
		if matches && item.Summary != "" {
			value := item.Summary
			return &value
		}
	}
	return nil
}
