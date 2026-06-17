package main

import "time"

const (
	swingRevalidationActionNone           = "none"
	swingRevalidationActionReviewRequired = "review_required"
	swingRevalidationActionBlocked        = "blocked"
)

type swingRevalidationCandidate struct {
	ID              string
	Status          string
	Horizon         string
	LastReviewedAt  time.Time
	Invalidators    []string
	PaperRuntime    bool
	DailyReviewMode bool
}

type swingRevalidationDecision struct {
	CandidateID             string
	Due                     bool
	Action                  string
	ReasonCode              string
	Summary                 string
	CreateBrokerInstruction bool
}

func evaluateSwingRevalidation(candidate swingRevalidationCandidate, now time.Time, failedInvalidators map[string]bool) swingRevalidationDecision {
	decision := swingRevalidationDecision{
		CandidateID: candidate.ID,
		Action:      swingRevalidationActionNone,
		Summary:     "no swing revalidation required",
	}
	if candidate.Horizon != "swing" || !candidate.PaperRuntime || !candidate.DailyReviewMode {
		return decision
	}
	if candidate.Status != "awaiting_approval" && candidate.Status != "approved" && candidate.Status != "submitted" {
		return decision
	}
	if !candidate.LastReviewedAt.IsZero() && now.Sub(candidate.LastReviewedAt) < 24*time.Hour {
		return decision
	}

	decision.Due = true
	for _, invalidator := range candidate.Invalidators {
		if failedInvalidators[invalidator] {
			decision.Action = swingRevalidationActionBlocked
			decision.ReasonCode = invalidator
			decision.Summary = "swing thesis invalidator failed; candidate requires block/review"
			return decision
		}
	}
	decision.Action = swingRevalidationActionReviewRequired
	decision.Summary = "swing candidate is due for daily review"
	return decision
}
