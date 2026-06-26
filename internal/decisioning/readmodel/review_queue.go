package readmodel

import (
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type Options struct {
	GeneratedAt time.Time
	AsOf        time.Time
}

type ReviewQueueSummary struct {
	GeneratedAt            time.Time `json:"generated_at"`
	TotalOpen              int       `json:"total_open"`
	TotalCritical          int       `json:"total_critical"`
	TotalHigh              int       `json:"total_high"`
	TotalDue               int       `json:"total_due"`
	TotalOverdue           int       `json:"total_overdue"`
	TotalNeedsMoreEvidence int       `json:"total_needs_more_evidence"`
	TotalDeferred          int       `json:"total_deferred"`
	TotalFollowUpActions   int       `json:"total_follow_up_actions"`
	NextDueAt              time.Time `json:"next_due_at"`
	Warnings               []string  `json:"warnings"`
	ForbiddenActions       []string  `json:"forbidden_actions"`
	ReadOnly               bool      `json:"read_only"`
}

func BuildReviewQueueSummary(repo operations.Repository, options Options) ReviewQueueSummary {
	generatedAt, asOf := resolveTimes(options)
	summary := ReviewQueueSummary{
		GeneratedAt:      generatedAt,
		Warnings:         safetyWarnings(),
		ForbiddenActions: feedback.ForbiddenActions(),
		ReadOnly:         true,
	}
	actionIDs := map[string]bool{}
	for _, item := range repo.ListTriageItems() {
		countQueueItem(&summary, item, asOf)
		for _, action := range repo.ListFollowUpActionsForTriageItem(item.TriageItemID) {
			if actionIDs[action.ActionID] {
				continue
			}
			actionIDs[action.ActionID] = true
			summary.TotalFollowUpActions++
		}
	}
	return summary
}

func countQueueItem(summary *ReviewQueueSummary, item triage.Item, asOf time.Time) {
	switch item.Status {
	case triage.StatusOpen:
		summary.TotalOpen++
	case triage.StatusNeedsMoreEvidence:
		summary.TotalNeedsMoreEvidence++
	case triage.StatusDeferred:
		summary.TotalDeferred++
	}
	switch item.Priority {
	case triage.PriorityCritical:
		summary.TotalCritical++
	case triage.PriorityHigh:
		summary.TotalHigh++
	}
	if !item.DueAt.IsZero() {
		if !item.DueAt.After(asOf) {
			summary.TotalDue++
		}
		if item.DueAt.Before(asOf) {
			summary.TotalOverdue++
		}
		if summary.NextDueAt.IsZero() || item.DueAt.Before(summary.NextDueAt) {
			summary.NextDueAt = item.DueAt
		}
	}
}

func resolveTimes(options Options) (time.Time, time.Time) {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	asOf := options.AsOf
	if asOf.IsZero() {
		asOf = generatedAt
	}
	return generatedAt, asOf
}

func safetyWarnings() []string {
	return []string{
		"Review operation read models are read-only.",
		"Human approval remains required; auto-apply, live orders, live trading approval, and broker execution remain blocked.",
	}
}
