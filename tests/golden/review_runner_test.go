package golden

import (
	"path/filepath"
	"testing"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/review"
)

type reviewGoldenCase struct {
	Name                        string                    `json:"name"`
	Mode                        string                    `json:"mode"`
	LogInput                    review.DecisionLogInput   `json:"log_input"`
	OutcomeInput                review.OutcomeReviewInput `json:"outcome_input"`
	ExpectedValid               bool                      `json:"expected_valid"`
	ExpectedCanScheduleReview   bool                      `json:"expected_can_schedule_review"`
	ExpectedCanCompleteReview   bool                      `json:"expected_can_complete_review"`
	ExpectedReviewWindows       []string                  `json:"expected_review_windows"`
	ExpectedLessonType          review.LessonType         `json:"expected_lesson_type"`
	ExpectedDecisionCorrect     bool                      `json:"expected_was_decision_correct"`
	ExpectedMissedOpportunity   bool                      `json:"expected_missed_opportunity"`
	ExpectedAvoidedLoss         bool                      `json:"expected_avoided_loss"`
	ExpectedRequiresHumanReview bool                      `json:"expected_requires_human_review"`
	ExpectedLessonHumanApproval bool                      `json:"expected_lesson_requires_human_approval"`
	ExpectedPromotionBlocked    bool                      `json:"expected_promotion_blocked"`
	ExpectedForbiddenActions    []string                  `json:"expected_forbidden_actions"`
	ExpectedErrorsContain       []string                  `json:"expected_errors_contain"`
}

func TestReviewGoldenCases(t *testing.T) {
	files := []string{
		"no_trade_schedules_reviews.json",
		"correct_no_trade_review.json",
		"missed_opportunity_review.json",
		"watch_schedules_reviews.json",
		"risk_veto_avoided_loss.json",
		"risk_veto_too_strict.json",
		"approved_paper_setup_worked.json",
		"approved_paper_setup_failed.json",
		"lesson_cannot_auto_change_strategy.json",
		"live_promotion_blocked.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc reviewGoldenCase
			readJSON(t, filepath.Join("review", file), &tc)

			switch tc.Mode {
			case "schedule":
				log, got := review.NewDecisionLog(tc.LogInput)
				assertReviewValidation(t, tc, got)
				assertContainsAll(t, log.ReviewSchedule.ReviewWindows, tc.ExpectedReviewWindows)
				assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			case "outcome":
				outcome, got := review.NewOutcomeReview(tc.OutcomeInput)
				assertReviewValidation(t, tc, got)
				if outcome.Lesson.LessonType != tc.ExpectedLessonType {
					t.Fatalf("%s lesson type = %s, want %s", tc.Name, outcome.Lesson.LessonType, tc.ExpectedLessonType)
				}
				if outcome.WasDecisionCorrect != tc.ExpectedDecisionCorrect {
					t.Fatalf("%s was_decision_correct = %v, want %v", tc.Name, outcome.WasDecisionCorrect, tc.ExpectedDecisionCorrect)
				}
				if outcome.MissedOpportunity != tc.ExpectedMissedOpportunity {
					t.Fatalf("%s missed_opportunity = %v, want %v", tc.Name, outcome.MissedOpportunity, tc.ExpectedMissedOpportunity)
				}
				if outcome.AvoidedLoss != tc.ExpectedAvoidedLoss {
					t.Fatalf("%s avoided_loss = %v, want %v", tc.Name, outcome.AvoidedLoss, tc.ExpectedAvoidedLoss)
				}
				if outcome.RequiresHumanReview != tc.ExpectedRequiresHumanReview {
					t.Fatalf("%s requires_human_review = %v, want %v", tc.Name, outcome.RequiresHumanReview, tc.ExpectedRequiresHumanReview)
				}
				if outcome.Lesson.RequiresHumanApproval != tc.ExpectedLessonHumanApproval {
					t.Fatalf("%s lesson requires human approval = %v, want %v", tc.Name, outcome.Lesson.RequiresHumanApproval, tc.ExpectedLessonHumanApproval)
				}
				assertContainsAll(t, got.ForbiddenActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove})
			default:
				t.Fatalf("unsupported review golden mode %q", tc.Mode)
			}
		})
	}
}

func assertReviewValidation(t *testing.T, tc reviewGoldenCase, got review.ValidationResult) {
	t.Helper()
	if got.IsValid != tc.ExpectedValid {
		t.Fatalf("%s is_valid = %v, want %v; errors=%v", tc.Name, got.IsValid, tc.ExpectedValid, got.ValidationErrors)
	}
	if got.CanScheduleReview != tc.ExpectedCanScheduleReview {
		t.Fatalf("%s can_schedule_review = %v, want %v", tc.Name, got.CanScheduleReview, tc.ExpectedCanScheduleReview)
	}
	if got.CanCompleteReview != tc.ExpectedCanCompleteReview {
		t.Fatalf("%s can_complete_review = %v, want %v", tc.Name, got.CanCompleteReview, tc.ExpectedCanCompleteReview)
	}
	if got.PromotionBlocked != tc.ExpectedPromotionBlocked {
		t.Fatalf("%s promotion_blocked = %v, want %v", tc.Name, got.PromotionBlocked, tc.ExpectedPromotionBlocked)
	}
	assertTextContainsAll(t, got.ValidationErrors, tc.ExpectedErrorsContain)
}
