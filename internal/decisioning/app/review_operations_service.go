package app

import (
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/readmodel"
	"jax-trading-assistant/internal/decisioning/workflow"
)

type OperationResult struct {
	RequestID             string    `json:"request_id"`
	Operation             string    `json:"operation"`
	Succeeded             bool      `json:"succeeded"`
	ValidationErrors      []string  `json:"validation_errors"`
	ValidationWarnings    []string  `json:"validation_warnings"`
	ForbiddenActions      []string  `json:"forbidden_actions"`
	RequiresHumanApproval bool      `json:"requires_human_approval"`
	AutoApplyAllowed      bool      `json:"auto_apply_allowed"`
	ReadOnly              bool      `json:"read_only"`
	CreatedAt             time.Time `json:"created_at"`
}

type ReviewQueueRequest struct {
	RequestID        string
	GeneratedAt      time.Time
	AsOf             time.Time
	AttemptedActions []string
}

type ReviewQueueSummaryResult struct {
	Result  OperationResult              `json:"result"`
	Summary readmodel.ReviewQueueSummary `json:"summary"`
}

type TriageDetailRequest struct {
	RequestID        string
	TriageItemID     string
	CreatedAt        time.Time
	AttemptedActions []string
}

type TriageDetailResult struct {
	Result OperationResult            `json:"result"`
	Detail readmodel.TriageItemDetail `json:"detail"`
	Found  bool                       `json:"found"`
}

type FollowUpActionDetailRequest struct {
	RequestID        string
	ActionID         string
	CreatedAt        time.Time
	AttemptedActions []string
}

type FollowUpActionDetailResult struct {
	Result OperationResult                `json:"result"`
	Detail readmodel.FollowUpActionDetail `json:"detail"`
	Found  bool                           `json:"found"`
}

type BuildReviewBatchRequest struct {
	RequestID             string
	BatchID               string
	GeneratedAt           time.Time
	AsOf                  time.Time
	SelectionReason       workflow.SelectionReason
	IncludeClosedRejected bool
	AttemptedActions      []string
}

type BuildReviewBatchResult struct {
	Result OperationResult      `json:"result"`
	Batch  workflow.ReviewBatch `json:"batch"`
}

type ExportReviewBatchRequest struct {
	RequestID        string
	ExportID         string
	GeneratedAt      time.Time
	Batch            workflow.ReviewBatch
	AttemptedActions []string
}

type ExportResult struct {
	Result OperationResult           `json:"result"`
	Export reviewexport.ExportResult `json:"export"`
}

type ExportFollowUpActionsRequest struct {
	RequestID        string
	ExportID         string
	GeneratedAt      time.Time
	SourceBatchID    string
	Actions          []operations.FollowUpAction
	AttemptedActions []string
}
