package triage

import (
	"strings"

	"jax-trading-assistant/internal/decisioning/feedback"
)

func ValidateItem(item Item) error {
	if item.TriageItemID == "" {
		return validationError("triage item id is required")
	}
	if !sourceAllowed(item.SourceType) {
		return validationError("source type %q is not allowed", item.SourceType)
	}
	if !priorityAllowed(item.Priority) {
		return validationError("priority %q is not allowed", item.Priority)
	}
	if !statusAllowed(item.Status) {
		return validationError("status %q is not allowed", item.Status)
	}
	if !item.RequiresHumanApproval {
		return validationError("human approval is required")
	}
	if item.AutoApplyAllowed {
		return validationError("auto apply is never allowed")
	}
	if !containsAll(item.ForbiddenActions, feedback.ForbiddenActions()) {
		return validationError("forbidden actions must include execute_trade, create_live_order, and auto_approve")
	}
	if containsLivePromotion(item.SuggestedAction) || containsLivePromotion(item.Reason) {
		return validationError("live-ready promotion and live trading approval are blocked")
	}
	return nil
}

func sourceAllowed(source SourceType) bool {
	switch source {
	case SourceMissedOpportunity, SourceRiskVetoTooStrict, SourceRiskVetoHelped, SourcePaperSetupWorked, SourcePaperSetupFailed, SourceResearchGap, SourceScoringReview, SourceConfirmationRuleReview, SourceNoTradeRuleReview, SourceDataQualityReview, SourceWatchlistReview:
		return true
	default:
		return false
	}
}

func priorityAllowed(priority Priority) bool {
	switch priority {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

func statusAllowed(status Status) bool {
	switch status {
	case StatusOpen, StatusAccepted, StatusRejected, StatusDeferred, StatusNeedsMoreEvidence, StatusClosed:
		return true
	default:
		return false
	}
}

func containsAll(got []string, want []string) bool {
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			return false
		}
	}
	return true
}

func containsLivePromotion(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "live_ready") ||
		strings.Contains(normalized, "live ready") ||
		strings.Contains(normalized, "live trading") ||
		strings.Contains(normalized, "live order")
}
