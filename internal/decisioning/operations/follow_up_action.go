package operations

import "time"

type ActionType string

const (
	ActionCreateResearchTask         ActionType = "CREATE_RESEARCH_TASK"
	ActionReviewScoringRule          ActionType = "REVIEW_SCORING_RULE"
	ActionReviewRiskThreshold        ActionType = "REVIEW_RISK_THRESHOLD"
	ActionReviewConfirmationRule     ActionType = "REVIEW_CONFIRMATION_RULE"
	ActionReviewNoTradeRule          ActionType = "REVIEW_NO_TRADE_RULE"
	ActionReviewWatchlist            ActionType = "REVIEW_WATCHLIST"
	ActionReviewDataQuality          ActionType = "REVIEW_DATA_QUALITY"
	ActionAddToManualObservationList ActionType = "ADD_TO_MANUAL_OBSERVATION_LIST"
	ActionCloseWithNoAction          ActionType = "CLOSE_WITH_NO_ACTION"
)

type ActionStatus string

const (
	ActionStatusOpen ActionStatus = "OPEN"
)

type FollowUpAction struct {
	ActionID              string       `json:"action_id"`
	TriageItemID          string       `json:"triage_item_id"`
	ActionType            ActionType   `json:"action_type"`
	Description           string       `json:"description"`
	TargetModule          string       `json:"target_module"`
	TargetSetupFamily     string       `json:"target_setup_family"`
	TargetEventType       string       `json:"target_event_type"`
	RequiresHumanApproval bool         `json:"requires_human_approval"`
	AutoApplyAllowed      bool         `json:"auto_apply_allowed"`
	ForbiddenActions      []string     `json:"forbidden_actions"`
	Status                ActionStatus `json:"status"`
	CreatedAt             time.Time    `json:"created_at"`
}
