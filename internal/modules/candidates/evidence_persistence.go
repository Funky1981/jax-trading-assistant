package candidates

import (
	"context"
	"fmt"
)

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
