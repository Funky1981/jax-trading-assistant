package workflow

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type BatchOptions struct {
	BatchID               string
	GeneratedAt           time.Time
	AsOf                  time.Time
	SelectionReason       SelectionReason
	IncludeClosedRejected bool
}

type ReviewBatch struct {
	BatchID             string                      `json:"batch_id"`
	GeneratedAt         time.Time                   `json:"generated_at"`
	SelectionReason     SelectionReason             `json:"selection_reason"`
	TotalItems          int                         `json:"total_items"`
	CriticalCount       int                         `json:"critical_count"`
	HighCount           int                         `json:"high_count"`
	OverdueCount        int                         `json:"overdue_count"`
	DueCount            int                         `json:"due_count"`
	TriageItems         []ReviewPacket              `json:"triage_items"`
	FollowUpActions     []operations.FollowUpAction `json:"follow_up_actions"`
	ForbiddenActions    []string                    `json:"forbidden_actions"`
	HumanReviewRequired bool                        `json:"human_review_required"`
	ReadOnly            bool                        `json:"read_only"`
	Warnings            []string                    `json:"warnings"`
}

func BuildReviewBatch(items []triage.Item, actions []operations.FollowUpAction, options BatchOptions) (ReviewBatch, error) {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	asOf := options.AsOf
	if asOf.IsZero() {
		asOf = generatedAt
	}
	batchID := options.BatchID
	if batchID == "" {
		batchID = "review_batch"
	}
	selectionReason := options.SelectionReason
	if selectionReason == "" {
		selectionReason = SelectionActiveReview
	}

	selected := SelectReviewItems(items, SelectionOptions{
		AsOf:                  asOf,
		IncludeClosedRejected: options.IncludeClosedRejected,
	})
	packets := make([]ReviewPacket, 0, len(selected))
	for _, item := range selected {
		packet, err := BuildReviewPacket(item)
		if err != nil {
			return ReviewBatch{}, err
		}
		packets = append(packets, packet)
	}
	copiedActions := make([]operations.FollowUpAction, 0, len(actions))
	for _, action := range actions {
		if err := operations.ValidateFollowUpAction(action); err != nil {
			return ReviewBatch{}, err
		}
		copiedActions = append(copiedActions, copyFollowUpAction(action))
	}

	batch := ReviewBatch{
		BatchID:             batchID,
		GeneratedAt:         generatedAt,
		SelectionReason:     selectionReason,
		TotalItems:          len(packets),
		TriageItems:         packets,
		FollowUpActions:     copiedActions,
		ForbiddenActions:    feedback.ForbiddenActions(),
		HumanReviewRequired: true,
		ReadOnly:            true,
		Warnings: []string{
			"Review workflow batches are read-only and require human review.",
			"Auto-apply, live orders, live trading approval, broker execution, and paper execution remain blocked.",
		},
	}
	for _, packet := range packets {
		switch packet.Priority {
		case triage.PriorityCritical:
			batch.CriticalCount++
		case triage.PriorityHigh:
			batch.HighCount++
		}
		if !packet.DueAt.IsZero() && packet.DueAt.Before(asOf) {
			batch.OverdueCount++
		}
		if !packet.DueAt.IsZero() && !packet.DueAt.After(asOf) {
			batch.DueCount++
		}
		if containsLivePromotion(packet.SuggestedAction) || containsLivePromotion(packet.Reason) {
			batch.Warnings = append(batch.Warnings, fmt.Sprintf("Live-readiness request blocked for triage item %s.", packet.TriageItemID))
		}
	}
	return batch, nil
}

func copyFollowUpAction(action operations.FollowUpAction) operations.FollowUpAction {
	copied := action
	copied.ForbiddenActions = append([]string{}, action.ForbiddenActions...)
	copied.AutoApplyAllowed = false
	copied.RequiresHumanApproval = true
	return copied
}

func containsLivePromotion(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "live_ready") ||
		strings.Contains(normalized, "live ready") ||
		strings.Contains(normalized, "live trading") ||
		strings.Contains(normalized, "live order")
}
