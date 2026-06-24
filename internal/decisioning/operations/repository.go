package operations

import (
	"time"

	"jax-trading-assistant/internal/decisioning/triage"
)

type Repository interface {
	SaveTriageItem(item triage.Item) error
	GetTriageItem(id string) (triage.Item, bool)
	ListTriageItems() []triage.Item
	ListOpenTriageItems() []triage.Item
	ListHighPriorityTriageItems() []triage.Item
	ListDueTriageItems(asOf time.Time) []triage.Item
	SaveHumanFeedbackDecision(decision FeedbackDecision) error
	GetHumanFeedbackDecision(id string) (FeedbackDecision, bool)
	ListFeedbackDecisionsForTriageItem(triageItemID string) []FeedbackDecision
	SaveFollowUpAction(action FollowUpAction) error
	GetFollowUpAction(id string) (FollowUpAction, bool)
	ListFollowUpActionsForTriageItem(triageItemID string) []FollowUpAction
	SaveOperationAuditRecord(record OperationAuditRecord)
	ListOperationAuditRecords() []OperationAuditRecord
}
