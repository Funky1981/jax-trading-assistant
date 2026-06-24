package operations

import (
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
)

type AuditAction string

const (
	AuditActionTriageItemSaved       AuditAction = "TRIAGE_ITEM_SAVED"
	AuditActionFeedbackDecisionSaved AuditAction = "FEEDBACK_DECISION_SAVED"
	AuditActionFollowUpActionSaved   AuditAction = "FOLLOW_UP_ACTION_SAVED"
	AuditActionAutoApplyBlocked      AuditAction = "AUTO_APPLY_BLOCKED"
)

type OperationAuditRecord struct {
	AuditID            string      `json:"audit_id"`
	TriageItemID       string      `json:"triage_item_id"`
	FeedbackDecisionID string      `json:"feedback_decision_id"`
	FollowUpActionID   string      `json:"follow_up_action_id"`
	SourceType         string      `json:"source_type"`
	Action             AuditAction `json:"action"`
	Actor              string      `json:"actor"`
	BeforeStatus       string      `json:"before_status"`
	AfterStatus        string      `json:"after_status"`
	Reason             string      `json:"reason"`
	ForbiddenActions   []string    `json:"forbidden_actions"`
	CreatedAt          time.Time   `json:"created_at"`
}

func newAuditRecord(action AuditAction, triageItemID, feedbackDecisionID, followUpActionID, sourceType, actor, beforeStatus, afterStatus, reason string, createdAt time.Time) OperationAuditRecord {
	return OperationAuditRecord{
		AuditID:            auditID(action, triageItemID, feedbackDecisionID, followUpActionID),
		TriageItemID:       triageItemID,
		FeedbackDecisionID: feedbackDecisionID,
		FollowUpActionID:   followUpActionID,
		SourceType:         sourceType,
		Action:             action,
		Actor:              actor,
		BeforeStatus:       beforeStatus,
		AfterStatus:        afterStatus,
		Reason:             reason,
		ForbiddenActions:   feedback.ForbiddenActions(),
		CreatedAt:          createdAt,
	}
}

func auditID(action AuditAction, triageItemID, feedbackDecisionID, followUpActionID string) string {
	switch {
	case feedbackDecisionID != "":
		return "audit_" + string(action) + "_" + feedbackDecisionID
	case followUpActionID != "":
		return "audit_" + string(action) + "_" + followUpActionID
	case triageItemID != "":
		return "audit_" + string(action) + "_" + triageItemID
	default:
		return "audit_" + string(action)
	}
}
