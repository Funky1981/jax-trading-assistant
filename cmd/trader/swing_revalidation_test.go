package main

import (
	"testing"
	"time"
)

func TestSwingRevalidationMarksOpenCandidateDueDaily(t *testing.T) {
	now := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	decision := evaluateSwingRevalidation(swingRevalidationCandidate{
		ID:              "candidate-1",
		Status:          "awaiting_approval",
		Horizon:         "swing",
		LastReviewedAt:  now.Add(-25 * time.Hour),
		Invalidators:    []string{"daily_close_breaks_risk_level"},
		PaperRuntime:    true,
		DailyReviewMode: true,
	}, now, nil)

	if !decision.Due {
		t.Fatalf("expected daily revalidation to be due: %#v", decision)
	}
	if decision.Action != swingRevalidationActionReviewRequired {
		t.Fatalf("action = %q, want review_required", decision.Action)
	}
}

func TestSwingRevalidationBlocksFailedInvalidator(t *testing.T) {
	now := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	decision := evaluateSwingRevalidation(swingRevalidationCandidate{
		ID:              "candidate-1",
		Status:          "awaiting_approval",
		Horizon:         "swing",
		LastReviewedAt:  now.Add(-25 * time.Hour),
		Invalidators:    []string{"event_thesis_invalidated"},
		PaperRuntime:    true,
		DailyReviewMode: true,
	}, now, map[string]bool{"event_thesis_invalidated": true})

	if decision.Action != swingRevalidationActionBlocked {
		t.Fatalf("action = %q, want blocked", decision.Action)
	}
	if decision.ReasonCode != "event_thesis_invalidated" {
		t.Fatalf("reason code = %q, want event_thesis_invalidated", decision.ReasonCode)
	}
}

func TestSwingRevalidationNeverCreatesBrokerInstruction(t *testing.T) {
	now := time.Date(2026, 6, 17, 16, 0, 0, 0, time.UTC)
	decision := evaluateSwingRevalidation(swingRevalidationCandidate{
		ID:              "candidate-1",
		Status:          "approved",
		Horizon:         "swing",
		LastReviewedAt:  now.Add(-25 * time.Hour),
		PaperRuntime:    true,
		DailyReviewMode: true,
	}, now, nil)

	if decision.CreateBrokerInstruction {
		t.Fatalf("revalidation must not create broker instructions: %#v", decision)
	}
}
