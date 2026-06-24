package workflow

import (
	"sort"
	"time"

	"jax-trading-assistant/internal/decisioning/triage"
)

type SelectionReason string

const (
	SelectionActiveReview SelectionReason = "ACTIVE_REVIEW"
	SelectionDueReview    SelectionReason = "DUE_REVIEW"
	SelectionOverdue      SelectionReason = "OVERDUE_REVIEW"
	SelectionHighPriority SelectionReason = "HIGH_PRIORITY_REVIEW"
)

type SelectionOptions struct {
	AsOf                  time.Time
	IncludeClosedRejected bool
}

func SelectReviewItems(items []triage.Item, options SelectionOptions) []triage.Item {
	asOf := options.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	selected := make([]triage.Item, 0, len(items))
	for _, item := range items {
		if !options.IncludeClosedRejected && !activeStatus(item.Status) {
			continue
		}
		selected = append(selected, item)
	}
	sort.SliceStable(selected, func(i, j int) bool {
		left := selected[i]
		right := selected[j]
		if priorityRank(left.Priority) != priorityRank(right.Priority) {
			return priorityRank(left.Priority) > priorityRank(right.Priority)
		}
		leftDueRank := dueRank(left.DueAt, asOf)
		rightDueRank := dueRank(right.DueAt, asOf)
		if leftDueRank != rightDueRank {
			return leftDueRank > rightDueRank
		}
		if !left.DueAt.Equal(right.DueAt) {
			return left.DueAt.Before(right.DueAt)
		}
		return left.TriageItemID < right.TriageItemID
	})
	return selected
}

func activeStatus(status triage.Status) bool {
	switch status {
	case triage.StatusOpen, triage.StatusDeferred, triage.StatusNeedsMoreEvidence:
		return true
	default:
		return false
	}
}

func priorityRank(priority triage.Priority) int {
	switch priority {
	case triage.PriorityCritical:
		return 4
	case triage.PriorityHigh:
		return 3
	case triage.PriorityMedium:
		return 2
	case triage.PriorityLow:
		return 1
	default:
		return 0
	}
}

func dueRank(dueAt, asOf time.Time) int {
	if dueAt.IsZero() {
		return 0
	}
	if dueAt.Before(asOf) {
		return 3
	}
	if sameUTCDate(dueAt, asOf) {
		return 2
	}
	return 1
}

func sameUTCDate(left, right time.Time) bool {
	l := left.UTC()
	r := right.UTC()
	return l.Year() == r.Year() && l.YearDay() == r.YearDay()
}
