package review

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
)

func TestDecisionLogSchedulesDefaultReviews(t *testing.T) {
	tests := []struct {
		name     string
		decision core.DecisionValue
		final    string
	}{
		{name: "no trade schedules reviews", decision: core.DecisionNoTrade},
		{name: "watch schedules reviews", decision: core.DecisionWatch},
		{name: "setup forming schedules reviews", decision: core.DecisionSetupForming},
		{name: "trade candidate schedules reviews", decision: core.DecisionTradeCandidate},
		{name: "risk rejected schedules reviews", decision: core.DecisionRejectedByRisk},
		{name: "approved paper schedules reviews", decision: core.DecisionTradeCandidate, final: "APPROVED_FOR_PAPER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, got := NewDecisionLog(DecisionLogInput{
				Decision:      baseDecision(tt.decision),
				FinalDecision: tt.final,
				CreatedAt:     fixedNow(),
				MemoryTags:    []string{"phase_7"},
			})
			if !got.CanScheduleReview {
				t.Fatalf("can schedule = false; errors=%v", got.ValidationErrors)
			}
			assertReviewContainsAll(t, log.ReviewSchedule.ReviewWindows, DefaultReviewWindows())
			assertReviewContainsAll(t, log.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
		})
	}
}

func TestOutcomeReviewLessons(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*OutcomeReviewInput)
		wantLesson  LessonType
		wantHuman   bool
		wantBlocked bool
	}{
		{
			name:       "correct no trade",
			wantLesson: LessonCorrectNoTrade,
		},
		{
			name: "missed opportunity requires human review",
			mutate: func(input *OutcomeReviewInput) {
				input.WasDecisionCorrect = false
				input.MissedOpportunity = true
				input.MarketOutcome.CleanSetupFormed = true
				input.LessonSummary = "A clean setup formed after the initial rejection."
			},
			wantLesson: LessonMissedOpportunity,
			wantHuman:  true,
		},
		{
			name: "risk veto helped avoid loss",
			mutate: func(input *OutcomeReviewInput) {
				input.OriginalDecision = core.DecisionRejectedByRisk
				input.FinalDecision = string(core.DecisionRejectedByRisk)
				input.WasDecisionCorrect = true
				input.AvoidedLoss = true
				input.LessonSummary = "Risk veto avoided a poor trade."
			},
			wantLesson: LessonRiskVetoHelped,
		},
		{
			name: "risk veto too strict requires human review",
			mutate: func(input *OutcomeReviewInput) {
				input.OriginalDecision = core.DecisionRejectedByRisk
				input.FinalDecision = string(core.DecisionRejectedByRisk)
				input.WasDecisionCorrect = false
				input.MissedOpportunity = true
				input.LessonSummary = "Risk veto may have been too strict."
			},
			wantLesson: LessonRiskVetoTooStrict,
			wantHuman:  true,
		},
		{
			name: "paper setup worked",
			mutate: func(input *OutcomeReviewInput) {
				input.OriginalDecision = core.DecisionTradeCandidate
				input.FinalDecision = "APPROVED_FOR_PAPER"
				input.AssetOutcome.HitTarget = true
				input.LessonSummary = "Paper setup reached target without invalidation."
			},
			wantLesson: LessonPaperSetupWorked,
		},
		{
			name: "paper setup failed",
			mutate: func(input *OutcomeReviewInput) {
				input.OriginalDecision = core.DecisionTradeCandidate
				input.FinalDecision = "APPROVED_FOR_PAPER"
				input.AssetOutcome.HitInvalidation = true
				input.LessonSummary = "Paper setup hit invalidation."
			},
			wantLesson: LessonPaperSetupFailed,
		},
		{
			name: "lesson cannot auto change strategy",
			mutate: func(input *OutcomeReviewInput) {
				input.MissedOpportunity = true
				input.StrategyAdjustmentSuggestion = "review confirmation threshold"
				input.LessonSummary = "Review threshold with human approval."
			},
			wantLesson: LessonMissedOpportunity,
			wantHuman:  true,
		},
		{
			name: "live promotion blocked",
			mutate: func(input *OutcomeReviewInput) {
				input.FinalDecision = "LIVE_READY"
				input.AttemptedPromotion = "LIVE_READY"
				input.LessonSummary = "Attempted live promotion must be blocked."
			},
			wantLesson:  LessonCorrectNoTrade,
			wantBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := baseOutcomeReviewInput()
			if tt.mutate != nil {
				tt.mutate(&input)
			}
			review, got := NewOutcomeReview(input)
			if tt.wantBlocked {
				if !got.PromotionBlocked {
					t.Fatalf("promotion blocked = false, want true")
				}
				return
			}
			if !got.CanCompleteReview {
				t.Fatalf("can complete = false; errors=%v", got.ValidationErrors)
			}
			if review.Lesson.LessonType != tt.wantLesson {
				t.Fatalf("lesson type = %s, want %s", review.Lesson.LessonType, tt.wantLesson)
			}
			if review.RequiresHumanReview != tt.wantHuman {
				t.Fatalf("requires human review = %v, want %v", review.RequiresHumanReview, tt.wantHuman)
			}
			assertReviewContainsAll(t, got.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
		})
	}
}

func baseDecision(decision core.DecisionValue) core.Decision {
	return core.Decision{
		DecisionID:             "dec_review_001",
		EventID:                "evt_review_001",
		Brain:                  core.BrainDecisionCore,
		Decision:               decision,
		ConfidenceScore:        0.72,
		ClarityScore:           0.60,
		EdgeScore:              0.45,
		ConflictScore:          0.70,
		RiskScore:              0.50,
		PrimaryReason:          "Structured decision requires later review.",
		SupportingReasons:      []string{"review every decision"},
		RequiredConfirmations:  []string{"clean setup forms"},
		InvalidationConditions: []string{"thesis invalidated"},
		AllowedActions:         []string{core.ActionStoreEvent, core.ActionReviewLater},
		ForbiddenActions:       []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove},
		ReviewAfter:            []string{ReviewWindow1Day, ReviewWindow1Week, ReviewWindow1Month},
	}
}

func baseOutcomeReviewInput() OutcomeReviewInput {
	return OutcomeReviewInput{
		ReviewID:           "rev_dec_review_001_1_week",
		DecisionID:         "dec_review_001",
		EventID:            "evt_review_001",
		ReviewWindow:       ReviewWindow1Week,
		OriginalDecision:   core.DecisionNoTrade,
		FinalDecision:      string(core.DecisionNoTrade),
		AssetOutcome:       OutcomeSummary{Summary: "No clean tradable setup formed."},
		MarketOutcome:      OutcomeSummary{Summary: "Conflicting drivers persisted."},
		WasDecisionCorrect: true,
		LessonSummary:      "No-trade was correct because no clean setup formed.",
		MemoryTags:         []string{"no_trade", "review"},
		CreatedAt:          fixedNow(),
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertReviewContainsAll(t *testing.T, got []string, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q in %v", item, got)
		}
	}
}
