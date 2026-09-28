package candidates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PersistCandidateEconomicInput appends the explicit economic request. Exact
// replays are idempotent; a conflicting request for the same candidate fails.
func (s *Store) PersistCandidateEconomicInput(ctx context.Context, input CandidateEconomicInput) error {
	validated, err := BuildCandidateEconomicInput(input, input.CreatedAt)
	if err != nil {
		return fmt.Errorf("validate candidate economic input: %w", err)
	}
	payload, err := json.Marshal(validated)
	if err != nil {
		return fmt.Errorf("marshal candidate economic input: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO candidate_economic_inputs (
			candidate_id, contract_version, instrument_id, issuer_id, identity_source,
			identity_policy_version, risk_allocation, requested_leverage, sizing_policy_id,
			sizing_policy_version, content_identity, created_at, payload
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (candidate_id) DO NOTHING
	`, validated.CandidateID, validated.ContractVersion, validated.InstrumentID, validated.IssuerID,
		validated.IdentitySource, validated.IdentityPolicyVersion, validated.RiskAllocation, validated.RequestedLeverage,
		validated.SizingPolicyID, validated.SizingPolicyVersion, validated.ContentIdentity, validated.CreatedAt, payload)
	if err != nil {
		return fmt.Errorf("insert candidate economic input: %w", err)
	}
	var existingIdentity string
	if err := s.pool.QueryRow(ctx, `SELECT content_identity FROM candidate_economic_inputs WHERE candidate_id=$1`, validated.CandidateID).Scan(&existingIdentity); err != nil {
		return fmt.Errorf("verify candidate economic input replay: %w", err)
	}
	if existingIdentity != validated.ContentIdentity {
		return errors.New("conflicting candidate economic input replay")
	}
	return nil
}

func (s *Store) GetCandidateEconomicInput(ctx context.Context, candidateID uuid.UUID) (CandidateEconomicInput, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM candidate_economic_inputs WHERE candidate_id=$1`, candidateID).Scan(&payload)
	if err != nil {
		return CandidateEconomicInput{}, fmt.Errorf("load candidate economic input: %w", err)
	}
	var input CandidateEconomicInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return CandidateEconomicInput{}, fmt.Errorf("decode candidate economic input: %w", err)
	}
	validated, err := BuildCandidateEconomicInput(input, input.CreatedAt)
	if err != nil {
		return CandidateEconomicInput{}, fmt.Errorf("validate stored candidate economic input: %w", err)
	}
	return validated, nil
}

// MarkEconomicInputUnavailable keeps research candidates non-approvable until
// the canonical identity and sizing-request contract is present.
func (s *Store) MarkEconomicInputUnavailable(ctx context.Context, candidateID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		reason = "canonical_economic_inputs_unavailable"
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE candidate_trades
		SET risk_status=$2, approval_status=$3, gate_status=$4,
			reject_reasons=CASE WHEN $5 = ANY(COALESCE(reject_reasons, ARRAY[]::text[])) THEN COALESCE(reject_reasons, ARRAY[]::text[])
				ELSE array_append(COALESCE(reject_reasons, ARRAY[]::text[]), $5) END,
			updated_at=NOW()
		WHERE id=$1
	`, candidateID, RiskStatusPending, ApprovalStatusRiskNotReady, GateStatusRiskPending, reason)
	if err != nil {
		return fmt.Errorf("mark candidate economic input unavailable: %w", err)
	}
	return nil
}

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
