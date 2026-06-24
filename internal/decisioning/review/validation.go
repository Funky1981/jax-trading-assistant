package review

import "jax-trading-assistant/internal/decisioning/core"

type ValidationResult struct {
	IsValid             bool     `json:"is_valid"`
	ValidationErrors    []string `json:"validation_errors"`
	ValidationWarnings  []string `json:"validation_warnings"`
	CanScheduleReview   bool     `json:"can_schedule_review"`
	CanCompleteReview   bool     `json:"can_complete_review"`
	RequiresHumanReview bool     `json:"requires_human_review"`
	PromotionBlocked    bool     `json:"promotion_blocked"`
	ForbiddenActions    []string `json:"forbidden_actions"`
	RequiredRemediation []string `json:"required_remediation"`
}

func ValidateDecisionLog(log DecisionLog) ValidationResult {
	result := baseValidationResult(log.ForbiddenActions)
	fail := func(message string) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, message)
		result.RequiredRemediation = append(result.RequiredRemediation, message)
	}
	if log.DecisionID == "" {
		fail("decision_id is required")
	}
	if log.EventID == "" {
		fail("event_id is required")
	}
	if log.OriginalDecision == "" {
		fail("original_decision is required")
	}
	if log.FinalDecision == "" {
		fail("final_decision is required")
	}
	if len(log.ReviewSchedule.ReviewWindows) == 0 {
		fail("review_schedule is required")
	}
	for _, window := range DefaultReviewWindows() {
		if !contains(log.ReviewSchedule.ReviewWindows, window) {
			fail("default review window " + window + " is required")
		}
	}
	if containsAny(log.AllowedActions, []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove}) {
		fail("allowed actions must not include live order, execution, or auto approval")
	}
	result.CanScheduleReview = result.IsValid
	return result
}

func ValidateOutcomeReview(review OutcomeReview) ValidationResult {
	result := baseValidationResult(review.ForbiddenActions)
	result.RequiresHumanReview = review.RequiresHumanReview || review.Lesson.RequiresHumanApproval
	result.PromotionBlocked = review.PromotionBlocked
	fail := func(message string) {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, message)
		result.RequiredRemediation = append(result.RequiredRemediation, message)
	}
	if review.ReviewID == "" {
		fail("review_id is required")
	}
	if review.DecisionID == "" {
		fail("decision_id is required")
	}
	if review.EventID == "" {
		fail("event_id is required")
	}
	if !ReviewWindowAllowed(review.ReviewWindow) {
		fail("review_window must be one of the default review windows")
	}
	if review.OriginalDecision == "" {
		fail("original_decision is required")
	}
	if review.FinalDecision == "" {
		fail("final_decision is required")
	}
	if !LessonTypeAllowed(review.Lesson.LessonType) {
		fail("lesson_type is not allowed")
	}
	if review.Lesson.LessonSummary == "" {
		fail("lesson_summary is required")
	}
	if (review.ScoringAdjustmentSuggestion != "" || review.StrategyAdjustmentSuggestion != "" || review.Lesson.SuggestedAction != "") && !review.Lesson.RequiresHumanApproval {
		fail("suggested scoring, strategy, or risk changes require human approval")
	}
	if review.FinalDecision == "LIVE_READY" || review.Lesson.SuggestedAction == "LIVE_READY" {
		result.PromotionBlocked = true
		fail("LIVE_READY promotion is not allowed")
	}
	if review.AttemptedPromotion == "LIVE_READY" {
		result.PromotionBlocked = true
		fail("LIVE_READY promotion attempt is blocked")
	}
	if review.AttemptedLiveTradingApproval {
		result.PromotionBlocked = true
		fail("live trading approval is not allowed")
	}
	if review.AttemptedPaperExecution {
		result.PromotionBlocked = true
		fail("review cannot execute paper trading")
	}
	result.CanCompleteReview = result.IsValid && !result.PromotionBlocked
	if result.PromotionBlocked {
		result.CanCompleteReview = false
	}
	result.ForbiddenActions = mandatoryForbiddenActions(review.ForbiddenActions)
	return result
}

func baseValidationResult(forbidden []string) ValidationResult {
	return ValidationResult{
		IsValid:          true,
		ForbiddenActions: mandatoryForbiddenActions(forbidden),
	}
}

func mandatoryForbiddenActions(forbidden []string) []string {
	return appendUnique(forbidden, []string{
		core.ActionExecuteTrade,
		core.ActionCreateLiveOrder,
		core.ActionAutoApprove,
	})
}

func appendUnique(base []string, extra []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		if item == "" || seen[item] {
			continue
		}
		out = append(out, item)
		seen[item] = true
	}
	for _, item := range extra {
		if item == "" || seen[item] {
			continue
		}
		out = append(out, item)
		seen[item] = true
	}
	return out
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func containsAny(values []string, needles []string) bool {
	for _, value := range values {
		for _, needle := range needles {
			if value == needle {
				return true
			}
		}
	}
	return false
}
