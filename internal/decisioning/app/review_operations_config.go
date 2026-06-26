package app

import (
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
)

const (
	OperationGetReviewQueueSummary     = "get_review_queue_summary"
	OperationGetTriageItemDetail       = "get_triage_item_detail"
	OperationGetFollowUpActionDetail   = "get_follow_up_action_detail"
	OperationBuildReviewBatch          = "build_review_batch"
	OperationExportReviewBatchJSON     = "export_review_batch_json"
	OperationExportReviewBatchMarkdown = "export_review_batch_markdown"
	OperationExportFollowUpActionsJSON = "export_follow_up_actions_json"
	OperationExportFollowUpActionsMD   = "export_follow_up_actions_markdown"
	OperationAcceptSuggestion          = "accept_suggestion"
	OperationRejectSuggestion          = "reject_suggestion"
	OperationDeferSuggestion           = "defer_suggestion"
	OperationRequestMoreEvidence       = "request_more_evidence"
	OperationCloseNoAction             = "close_no_action"
	defaultReviewOperationsServiceName = "review_operations"
	defaultReviewOperationsRequestID   = "review_operations_request"
	defaultReviewOperationsBatchID     = "review_batch"
	defaultReviewOperationsExportID    = "review_operations_export"
)

type ReviewOperationsConfig struct {
	ServiceName      string
	DefaultRequestID string
	DefaultBatchID   string
	DefaultExportID  string
	DefaultCreatedAt time.Time
	ForbiddenActions []string
}

type SafetyDefaults struct {
	RequiresHumanApproval bool
	AutoApplyAllowed      bool
	ReadOnly              bool
	ForbiddenActions      []string
}

func DefaultReviewOperationsConfig() ReviewOperationsConfig {
	return ReviewOperationsConfig{
		ServiceName:      defaultReviewOperationsServiceName,
		DefaultRequestID: defaultReviewOperationsRequestID,
		DefaultBatchID:   defaultReviewOperationsBatchID,
		DefaultExportID:  defaultReviewOperationsExportID,
		DefaultCreatedAt: time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC),
		ForbiddenActions: feedback.ForbiddenActions(),
	}
}

func (c ReviewOperationsConfig) normalize() ReviewOperationsConfig {
	if c.ServiceName == "" {
		c.ServiceName = defaultReviewOperationsServiceName
	}
	if c.DefaultRequestID == "" {
		c.DefaultRequestID = defaultReviewOperationsRequestID
	}
	if c.DefaultBatchID == "" {
		c.DefaultBatchID = defaultReviewOperationsBatchID
	}
	if c.DefaultExportID == "" {
		c.DefaultExportID = defaultReviewOperationsExportID
	}
	if c.DefaultCreatedAt.IsZero() {
		c.DefaultCreatedAt = time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	}
	c.ForbiddenActions = mergeForbiddenActions(c.ForbiddenActions)
	return c
}
