package readmodel

import (
	"time"

	"jax-trading-assistant/internal/decisioning/workflow"
)

type ReviewBatchSummary struct {
	BatchID             string                   `json:"batch_id"`
	GeneratedAt         time.Time                `json:"generated_at"`
	SelectionReason     workflow.SelectionReason `json:"selection_reason"`
	TotalItems          int                      `json:"total_items"`
	CriticalCount       int                      `json:"critical_count"`
	HighCount           int                      `json:"high_count"`
	OverdueCount        int                      `json:"overdue_count"`
	DueCount            int                      `json:"due_count"`
	FollowUpActionCount int                      `json:"follow_up_action_count"`
	ForbiddenActions    []string                 `json:"forbidden_actions"`
	HumanReviewRequired bool                     `json:"human_review_required"`
	ReadOnly            bool                     `json:"read_only"`
	Warnings            []string                 `json:"warnings"`
}

func BuildReviewBatchSummary(batch workflow.ReviewBatch) ReviewBatchSummary {
	return ReviewBatchSummary{
		BatchID:             batch.BatchID,
		GeneratedAt:         batch.GeneratedAt,
		SelectionReason:     batch.SelectionReason,
		TotalItems:          batch.TotalItems,
		CriticalCount:       batch.CriticalCount,
		HighCount:           batch.HighCount,
		OverdueCount:        batch.OverdueCount,
		DueCount:            batch.DueCount,
		FollowUpActionCount: len(batch.FollowUpActions),
		ForbiddenActions:    append([]string{}, batch.ForbiddenActions...),
		HumanReviewRequired: true,
		ReadOnly:            true,
		Warnings:            append([]string{}, batch.Warnings...),
	}
}
