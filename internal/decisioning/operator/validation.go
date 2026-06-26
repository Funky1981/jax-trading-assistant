package operator

import (
	"fmt"
	"strings"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

func validateActionRequest(request ActionRequest, decision operations.HumanDecision, item triage.Item) []string {
	var errors []string
	if request.FeedbackDecisionID == "" {
		errors = append(errors, "feedback decision id is required")
	}
	if strings.TrimSpace(request.HumanReviewer) == "" {
		errors = append(errors, "human reviewer is required")
	}
	if strings.TrimSpace(request.Rationale) == "" {
		errors = append(errors, "rationale is required")
	}
	if decision == operations.DecisionRequestMoreEvidence && len(request.RequiredEvidence) == 0 {
		errors = append(errors, "required evidence is required when requesting more evidence")
	}
	if request.AutoApplyAllowed {
		errors = append(errors, "auto_apply_allowed is never permitted for operator actions")
	}
	for _, action := range request.AttemptedActions {
		if isForbiddenAction(action) {
			errors = append(errors, fmt.Sprintf("forbidden action %q is blocked", action))
		}
		if containsLivePromotion(action) {
			errors = append(errors, fmt.Sprintf("live-ready or live-order action %q is blocked", action))
		}
	}
	for _, value := range []string{request.Rationale, item.Reason, item.SuggestedAction} {
		if containsLivePromotion(value) {
			errors = append(errors, "live-ready promotion and live trading approval are blocked")
			break
		}
	}
	return errors
}

func isForbiddenAction(action string) bool {
	normalized := strings.ToLower(strings.TrimSpace(action))
	for _, forbidden := range feedback.ForbiddenActions() {
		if normalized == strings.ToLower(forbidden) {
			return true
		}
	}
	return false
}

func containsLivePromotion(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "live_ready") ||
		strings.Contains(normalized, "live ready") ||
		strings.Contains(normalized, "live trading") ||
		strings.Contains(normalized, "live approval") ||
		strings.Contains(normalized, "live order")
}
