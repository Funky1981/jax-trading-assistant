package operations

import (
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

type HumanDecision string

const (
	DecisionAcceptSuggestion    HumanDecision = "ACCEPT_SUGGESTION"
	DecisionRejectSuggestion    HumanDecision = "REJECT_SUGGESTION"
	DecisionDeferDecision       HumanDecision = "DEFER_DECISION"
	DecisionRequestMoreEvidence HumanDecision = "REQUEST_MORE_EVIDENCE"
	DecisionCloseNoAction       HumanDecision = "CLOSE_NO_ACTION"
)

type FeedbackDecisionInput struct {
	FeedbackDecisionID      string
	Decision                HumanDecision
	HumanReviewer           string
	Rationale               string
	AcceptedFollowUpActions []string
	RejectedFollowUpActions []string
	RequiredEvidence        []string
	CreatedAt               time.Time
}

type FeedbackDecision struct {
	FeedbackDecisionID      string        `json:"feedback_decision_id"`
	TriageItemID            string        `json:"triage_item_id"`
	Decision                HumanDecision `json:"decision"`
	HumanReviewer           string        `json:"human_reviewer"`
	Rationale               string        `json:"rationale"`
	AcceptedFollowUpActions []string      `json:"accepted_follow_up_actions"`
	RejectedFollowUpActions []string      `json:"rejected_follow_up_actions"`
	RequiredEvidence        []string      `json:"required_evidence"`
	CreatedAt               time.Time     `json:"created_at"`
}

type ReviewOperationResult struct {
	Item            triage.Item      `json:"triage_item"`
	Decision        FeedbackDecision `json:"feedback_decision"`
	FollowUpActions []FollowUpAction `json:"follow_up_actions"`
}

func ApplyFeedbackDecision(item triage.Item, input FeedbackDecisionInput) (ReviewOperationResult, error) {
	if err := triage.ValidateItem(item); err != nil {
		return ReviewOperationResult{}, err
	}
	if err := validateDecisionInput(input); err != nil {
		return ReviewOperationResult{}, err
	}

	decision := FeedbackDecision{
		FeedbackDecisionID:      input.FeedbackDecisionID,
		TriageItemID:            item.TriageItemID,
		Decision:                input.Decision,
		HumanReviewer:           input.HumanReviewer,
		Rationale:               input.Rationale,
		AcceptedFollowUpActions: append([]string{}, input.AcceptedFollowUpActions...),
		RejectedFollowUpActions: append([]string{}, input.RejectedFollowUpActions...),
		RequiredEvidence:        append([]string{}, input.RequiredEvidence...),
		CreatedAt:               input.CreatedAt,
	}

	updated := item
	updated.AutoApplyAllowed = false
	updated.RequiresHumanApproval = true
	updated.ForbiddenActions = mergeForbiddenActions(updated.ForbiddenActions)
	updated.UpdatedAt = input.CreatedAt

	var actions []FollowUpAction
	switch input.Decision {
	case DecisionAcceptSuggestion:
		updated.Status = triage.StatusAccepted
		action := actionForItem(updated, input.CreatedAt)
		decision.AcceptedFollowUpActions = []string{string(action.ActionType)}
		actions = append(actions, action)
	case DecisionRejectSuggestion:
		updated.Status = triage.StatusRejected
	case DecisionDeferDecision:
		updated.Status = triage.StatusDeferred
	case DecisionRequestMoreEvidence:
		updated.Status = triage.StatusNeedsMoreEvidence
		action := evidenceActionForItem(updated, input.CreatedAt)
		decision.AcceptedFollowUpActions = []string{string(action.ActionType)}
		actions = append(actions, action)
	case DecisionCloseNoAction:
		updated.Status = triage.StatusClosed
	}

	if err := triage.ValidateItem(updated); err != nil {
		return ReviewOperationResult{}, err
	}
	for _, action := range actions {
		if err := ValidateFollowUpAction(action); err != nil {
			return ReviewOperationResult{}, err
		}
	}

	return ReviewOperationResult{
		Item:            updated,
		Decision:        decision,
		FollowUpActions: actions,
	}, nil
}

func validateDecisionInput(input FeedbackDecisionInput) error {
	if input.FeedbackDecisionID == "" {
		return fmt.Errorf("feedback decision id is required")
	}
	if input.HumanReviewer == "" {
		return fmt.Errorf("human reviewer is required")
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return fmt.Errorf("rationale is required")
	}
	switch input.Decision {
	case DecisionAcceptSuggestion, DecisionRejectSuggestion, DecisionDeferDecision, DecisionRequestMoreEvidence, DecisionCloseNoAction:
	default:
		return fmt.Errorf("decision %q is not allowed", input.Decision)
	}
	if input.Decision == DecisionRequestMoreEvidence && len(input.RequiredEvidence) == 0 {
		return fmt.Errorf("required evidence is required when requesting more evidence")
	}
	return nil
}

func actionForItem(item triage.Item, now time.Time) FollowUpAction {
	return buildAction(item, actionTypeForSource(item.SourceType), now)
}

func evidenceActionForItem(item triage.Item, now time.Time) FollowUpAction {
	if item.SourceType == triage.SourceDataQualityReview {
		return buildAction(item, ActionReviewDataQuality, now)
	}
	return buildAction(item, ActionCreateResearchTask, now)
}

func buildAction(item triage.Item, actionType ActionType, now time.Time) FollowUpAction {
	return FollowUpAction{
		ActionID:              "action_" + item.TriageItemID,
		TriageItemID:          item.TriageItemID,
		ActionType:            actionType,
		Description:           descriptionForAction(actionType),
		TargetModule:          targetModuleForAction(actionType),
		TargetSetupFamily:     item.SetupFamily,
		TargetEventType:       item.EventType,
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      mergeForbiddenActions(item.ForbiddenActions),
		Status:                ActionStatusOpen,
		CreatedAt:             now,
	}
}

func actionTypeForSource(source triage.SourceType) ActionType {
	switch source {
	case triage.SourceResearchGap, triage.SourcePaperSetupFailed:
		return ActionCreateResearchTask
	case triage.SourceScoringReview:
		return ActionReviewScoringRule
	case triage.SourceRiskVetoTooStrict, triage.SourceRiskVetoHelped:
		return ActionReviewRiskThreshold
	case triage.SourceConfirmationRuleReview:
		return ActionReviewConfirmationRule
	case triage.SourceNoTradeRuleReview, triage.SourceMissedOpportunity:
		return ActionReviewNoTradeRule
	case triage.SourceWatchlistReview:
		return ActionReviewWatchlist
	case triage.SourceDataQualityReview:
		return ActionReviewDataQuality
	default:
		return ActionCloseWithNoAction
	}
}

func descriptionForAction(actionType ActionType) string {
	switch actionType {
	case ActionCreateResearchTask:
		return "Create a manual research task; do not change trading rules automatically."
	case ActionReviewScoringRule:
		return "Review scoring rule manually; no automatic scoring change is allowed."
	case ActionReviewRiskThreshold:
		return "Review risk threshold manually; no automatic risk-rule change is allowed."
	case ActionReviewConfirmationRule:
		return "Review confirmation rule manually."
	case ActionReviewNoTradeRule:
		return "Review no-trade rule manually."
	case ActionReviewWatchlist:
		return "Review watchlist manually."
	case ActionReviewDataQuality:
		return "Review source data quality manually."
	case ActionAddToManualObservationList:
		return "Add setup to manual observation list."
	default:
		return "Close with no action."
	}
}

func targetModuleForAction(actionType ActionType) string {
	switch actionType {
	case ActionCreateResearchTask:
		return "research"
	case ActionReviewScoringRule:
		return "scoring"
	case ActionReviewRiskThreshold:
		return "risk"
	case ActionReviewConfirmationRule:
		return "confirmation"
	case ActionReviewNoTradeRule:
		return "review"
	case ActionReviewWatchlist:
		return "watchlist"
	case ActionReviewDataQuality:
		return "data_quality"
	default:
		return "review"
	}
}

func ValidateFollowUpAction(action FollowUpAction) error {
	if action.ActionID == "" {
		return fmt.Errorf("action id is required")
	}
	if action.TriageItemID == "" {
		return fmt.Errorf("triage item id is required")
	}
	if !action.RequiresHumanApproval {
		return fmt.Errorf("follow-up action requires human approval")
	}
	if action.AutoApplyAllowed {
		return fmt.Errorf("follow-up action auto apply is never allowed")
	}
	if !containsAll(action.ForbiddenActions, feedback.ForbiddenActions()) {
		return fmt.Errorf("follow-up action missing forbidden actions")
	}
	if containsLivePromotion(action.Description) {
		return fmt.Errorf("follow-up action cannot promote live readiness")
	}
	return nil
}

func mergeForbiddenActions(actions []string) []string {
	seen := map[string]bool{}
	var merged []string
	for _, action := range append(actions, feedback.ForbiddenActions()...) {
		if action == "" || seen[action] {
			continue
		}
		seen[action] = true
		merged = append(merged, action)
	}
	return merged
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
