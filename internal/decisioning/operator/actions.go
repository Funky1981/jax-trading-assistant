package operator

import (
	"time"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type ActionRequest struct {
	TriageItemID       string
	FeedbackDecisionID string
	HumanReviewer      string
	Rationale          string
	RequiredEvidence   []string
	AttemptedActions   []string
	AutoApplyAllowed   bool
	CreatedAt          time.Time
}

type ActionResult struct {
	ActionResultID        string                   `json:"action_result_id"`
	TriageItemID          string                   `json:"triage_item_id"`
	Action                operations.HumanDecision `json:"action"`
	PreviousStatus        triage.Status            `json:"previous_status"`
	NewStatus             triage.Status            `json:"new_status"`
	FeedbackDecisionID    string                   `json:"feedback_decision_id"`
	FollowUpActionIDs     []string                 `json:"follow_up_action_ids"`
	ValidationErrors      []string                 `json:"validation_errors"`
	ValidationWarnings    []string                 `json:"validation_warnings"`
	RequiresHumanApproval bool                     `json:"requires_human_approval"`
	AutoApplyAllowed      bool                     `json:"auto_apply_allowed"`
	ForbiddenActions      []string                 `json:"forbidden_actions"`
	CreatedAt             time.Time                `json:"created_at"`
}

func (s Service) AcceptSuggestion(request ActionRequest) (ActionResult, error) {
	return s.apply(request, operations.DecisionAcceptSuggestion)
}

func (s Service) RejectSuggestion(request ActionRequest) (ActionResult, error) {
	return s.apply(request, operations.DecisionRejectSuggestion)
}

func (s Service) DeferSuggestion(request ActionRequest) (ActionResult, error) {
	return s.apply(request, operations.DecisionDeferDecision)
}

func (s Service) RequestMoreEvidence(request ActionRequest) (ActionResult, error) {
	return s.apply(request, operations.DecisionRequestMoreEvidence)
}

func (s Service) CloseNoAction(request ActionRequest) (ActionResult, error) {
	return s.apply(request, operations.DecisionCloseNoAction)
}
