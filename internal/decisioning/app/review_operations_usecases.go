package app

import (
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/operator"
	"jax-trading-assistant/internal/decisioning/readmodel"
	"jax-trading-assistant/internal/decisioning/triage"
	"jax-trading-assistant/internal/decisioning/workflow"
)

type ReviewActionRequest struct {
	RequestID          string
	TriageItemID       string
	FeedbackDecisionID string
	HumanReviewer      string
	Rationale          string
	RequiredEvidence   []string
	AttemptedActions   []string
	AutoApplyAllowed   bool
	CreatedAt          time.Time
}

type ReviewActionResult struct {
	Result OperationResult       `json:"result"`
	Action operator.ActionResult `json:"action"`
}

func (s ReviewOperationsService) GetReviewQueueSummary(request ReviewQueueRequest) (ReviewQueueSummaryResult, error) {
	result := s.baseResult(OperationGetReviewQueueSummary, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.GeneratedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if !result.Succeeded {
		return ReviewQueueSummaryResult{Result: result}, nil
	}
	summary := s.operator.GetReviewQueueSummary(readmodel.Options{
		GeneratedAt: request.GeneratedAt,
		AsOf:        request.AsOf,
	})
	result.ForbiddenActions = mergeForbiddenActions(summary.ForbiddenActions)
	return ReviewQueueSummaryResult{Result: result, Summary: summary}, nil
}

func (s ReviewOperationsService) GetTriageItemDetail(request TriageDetailRequest) (TriageDetailResult, error) {
	result := s.baseResult(OperationGetTriageItemDetail, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.CreatedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if request.TriageItemID == "" {
		result = result.withValidation([]string{"triage item id is required"}, nil)
	}
	if !result.Succeeded {
		return TriageDetailResult{Result: result}, nil
	}
	detail, found, err := s.operator.GetTriageItemDetail(request.TriageItemID)
	if err != nil {
		return TriageDetailResult{Result: result}, err
	}
	result.Succeeded = found
	if found {
		result.ForbiddenActions = mergeForbiddenActions(detail.ForbiddenActions)
	} else {
		result = result.withValidation([]string{"triage item not found"}, nil)
	}
	return TriageDetailResult{Result: result, Detail: detail, Found: found}, nil
}

func (s ReviewOperationsService) GetFollowUpActionDetail(request FollowUpActionDetailRequest) (FollowUpActionDetailResult, error) {
	result := s.baseResult(OperationGetFollowUpActionDetail, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.CreatedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if request.ActionID == "" {
		result = result.withValidation([]string{"follow-up action id is required"}, nil)
	}
	if !result.Succeeded {
		return FollowUpActionDetailResult{Result: result}, nil
	}
	detail, found, err := s.operator.GetFollowUpActionDetail(request.ActionID)
	if err != nil {
		return FollowUpActionDetailResult{Result: result}, err
	}
	result.Succeeded = found
	if found {
		result.ForbiddenActions = mergeForbiddenActions(detail.ForbiddenActions)
	} else {
		result = result.withValidation([]string{"follow-up action not found"}, nil)
	}
	return FollowUpActionDetailResult{Result: result, Detail: detail, Found: found}, nil
}

func (s ReviewOperationsService) BuildReviewBatch(request BuildReviewBatchRequest) (BuildReviewBatchResult, error) {
	result := s.baseResult(OperationBuildReviewBatch, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.GeneratedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if !result.Succeeded {
		return BuildReviewBatchResult{Result: result}, nil
	}
	items := s.repo.ListTriageItems()
	actions := s.followUpActionsForItems(items)
	batchID := request.BatchID
	if batchID == "" {
		batchID = s.config.DefaultBatchID
	}
	batch, err := workflow.BuildReviewBatch(items, actions, workflow.BatchOptions{
		BatchID:               batchID,
		GeneratedAt:           request.GeneratedAt,
		AsOf:                  request.AsOf,
		SelectionReason:       request.SelectionReason,
		IncludeClosedRejected: request.IncludeClosedRejected,
	})
	if err != nil {
		return BuildReviewBatchResult{Result: result}, err
	}
	result.ForbiddenActions = mergeForbiddenActions(batch.ForbiddenActions)
	result.ValidationWarnings = append(result.ValidationWarnings, batch.Warnings...)
	return BuildReviewBatchResult{Result: result, Batch: batch}, nil
}

func (s ReviewOperationsService) ExportReviewBatchJSON(request ExportReviewBatchRequest) (ExportResult, error) {
	return s.exportReviewBatch(request, reviewexport.FormatJSON, OperationExportReviewBatchJSON)
}

func (s ReviewOperationsService) ExportReviewBatchMarkdown(request ExportReviewBatchRequest) (ExportResult, error) {
	return s.exportReviewBatch(request, reviewexport.FormatMarkdown, OperationExportReviewBatchMarkdown)
}

func (s ReviewOperationsService) ExportFollowUpActionsJSON(request ExportFollowUpActionsRequest) (ExportResult, error) {
	return s.exportFollowUpActions(request, reviewexport.FormatJSON, OperationExportFollowUpActionsJSON)
}

func (s ReviewOperationsService) ExportFollowUpActionsMarkdown(request ExportFollowUpActionsRequest) (ExportResult, error) {
	return s.exportFollowUpActions(request, reviewexport.FormatMarkdown, OperationExportFollowUpActionsMD)
}

func (s ReviewOperationsService) exportReviewBatch(request ExportReviewBatchRequest, format reviewexport.ExportFormat, operation string) (ExportResult, error) {
	result := s.baseResult(operation, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.GeneratedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if !result.Succeeded {
		return ExportResult{Result: result}, nil
	}
	exportID := request.ExportID
	if exportID == "" {
		exportID = s.config.DefaultExportID
	}
	exported, err := reviewexport.ExportReviewBatch(request.Batch, reviewexport.ExportOptions{
		ExportID:    exportID,
		ExportType:  reviewexport.ExportTypeReviewBatch,
		Format:      format,
		GeneratedAt: request.GeneratedAt,
	})
	if err != nil {
		return ExportResult{Result: result}, err
	}
	result.ForbiddenActions = mergeForbiddenActions(exported.ForbiddenActions)
	result.ValidationWarnings = append(result.ValidationWarnings, exported.Warnings...)
	return ExportResult{Result: result, Export: exported}, nil
}

func (s ReviewOperationsService) exportFollowUpActions(request ExportFollowUpActionsRequest, format reviewexport.ExportFormat, operation string) (ExportResult, error) {
	result := s.baseResult(operation, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.GeneratedAt,
		attemptedActions: request.AttemptedActions,
	}, true)
	if !result.Succeeded {
		return ExportResult{Result: result}, nil
	}
	exportID := request.ExportID
	if exportID == "" {
		exportID = s.config.DefaultExportID
	}
	exported, err := reviewexport.ExportFollowUpActions(request.Actions, request.SourceBatchID, reviewexport.ExportOptions{
		ExportID:    exportID,
		ExportType:  reviewexport.ExportTypeFollowUpActions,
		Format:      format,
		GeneratedAt: request.GeneratedAt,
	})
	if err != nil {
		return ExportResult{Result: result}, err
	}
	result.ForbiddenActions = mergeForbiddenActions(exported.ForbiddenActions)
	result.ValidationWarnings = append(result.ValidationWarnings, exported.Warnings...)
	return ExportResult{Result: result, Export: exported}, nil
}

func (s ReviewOperationsService) AcceptSuggestion(request ReviewActionRequest) (ReviewActionResult, error) {
	return s.applyAction(OperationAcceptSuggestion, request, operations.DecisionAcceptSuggestion)
}

func (s ReviewOperationsService) RejectSuggestion(request ReviewActionRequest) (ReviewActionResult, error) {
	return s.applyAction(OperationRejectSuggestion, request, operations.DecisionRejectSuggestion)
}

func (s ReviewOperationsService) DeferSuggestion(request ReviewActionRequest) (ReviewActionResult, error) {
	return s.applyAction(OperationDeferSuggestion, request, operations.DecisionDeferDecision)
}

func (s ReviewOperationsService) RequestMoreEvidence(request ReviewActionRequest) (ReviewActionResult, error) {
	return s.applyAction(OperationRequestMoreEvidence, request, operations.DecisionRequestMoreEvidence)
}

func (s ReviewOperationsService) CloseNoAction(request ReviewActionRequest) (ReviewActionResult, error) {
	return s.applyAction(OperationCloseNoAction, request, operations.DecisionCloseNoAction)
}

func (s ReviewOperationsService) applyAction(operation string, request ReviewActionRequest, decision operations.HumanDecision) (ReviewActionResult, error) {
	result := s.baseResult(operation, requestMetadata{
		requestID:        request.RequestID,
		createdAt:        request.CreatedAt,
		attemptedActions: append(append([]string{}, request.AttemptedActions...), request.Rationale),
	}, false)
	if !result.Succeeded {
		return ReviewActionResult{Result: result}, nil
	}
	action, err := s.runOperatorAction(decision, operator.ActionRequest{
		TriageItemID:       request.TriageItemID,
		FeedbackDecisionID: request.FeedbackDecisionID,
		HumanReviewer:      request.HumanReviewer,
		Rationale:          request.Rationale,
		RequiredEvidence:   append([]string{}, request.RequiredEvidence...),
		AttemptedActions:   append([]string{}, request.AttemptedActions...),
		AutoApplyAllowed:   request.AutoApplyAllowed,
		CreatedAt:          request.CreatedAt,
	})
	if err != nil {
		return ReviewActionResult{Result: result}, err
	}
	result = result.withValidation(action.ValidationErrors, action.ValidationWarnings)
	result.ForbiddenActions = mergeForbiddenActions(action.ForbiddenActions)
	result.Succeeded = len(result.ValidationErrors) == 0
	return ReviewActionResult{Result: result, Action: action}, nil
}

func (s ReviewOperationsService) runOperatorAction(decision operations.HumanDecision, request operator.ActionRequest) (operator.ActionResult, error) {
	switch decision {
	case operations.DecisionAcceptSuggestion:
		return s.operator.AcceptSuggestion(request)
	case operations.DecisionRejectSuggestion:
		return s.operator.RejectSuggestion(request)
	case operations.DecisionDeferDecision:
		return s.operator.DeferSuggestion(request)
	case operations.DecisionRequestMoreEvidence:
		return s.operator.RequestMoreEvidence(request)
	case operations.DecisionCloseNoAction:
		return s.operator.CloseNoAction(request)
	default:
		return operator.ActionResult{}, nil
	}
}

func (s ReviewOperationsService) followUpActionsForItems(items []triage.Item) []operations.FollowUpAction {
	var actions []operations.FollowUpAction
	for _, item := range items {
		actions = append(actions, s.repo.ListFollowUpActionsForTriageItem(item.TriageItemID)...)
	}
	return actions
}
