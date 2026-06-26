package operator

import (
	"fmt"
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/readmodel"
	"jax-trading-assistant/internal/decisioning/workflow"
)

type Service struct {
	repo operations.Repository
}

func NewService(repo operations.Repository) Service {
	return Service{repo: repo}
}

func (s Service) GetReviewQueueSummary(options readmodel.Options) readmodel.ReviewQueueSummary {
	return readmodel.BuildReviewQueueSummary(s.repo, options)
}

func (s Service) GetTriageItemDetail(triageItemID string) (readmodel.TriageItemDetail, bool, error) {
	item, ok := s.repo.GetTriageItem(triageItemID)
	if !ok {
		return readmodel.TriageItemDetail{}, false, nil
	}
	detail, err := readmodel.BuildTriageItemDetail(item)
	return detail, true, err
}

func (s Service) GetFollowUpActionDetail(actionID string) (readmodel.FollowUpActionDetail, bool, error) {
	action, ok := s.repo.GetFollowUpAction(actionID)
	if !ok {
		return operations.FollowUpAction{}, false, nil
	}
	detail, err := readmodel.BuildFollowUpActionDetail(action)
	return detail, true, err
}

func (s Service) GetReviewBatchSummary(batch workflow.ReviewBatch) readmodel.ReviewBatchSummary {
	return readmodel.BuildReviewBatchSummary(batch)
}

func (s Service) GetExportSummary(result reviewexport.ExportResult) readmodel.ExportSummary {
	return readmodel.BuildExportSummary(result)
}

func (s Service) apply(request ActionRequest, decision operations.HumanDecision) (ActionResult, error) {
	createdAt := request.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	result := baseActionResult(request, decision, createdAt)
	item, ok := s.repo.GetTriageItem(request.TriageItemID)
	if !ok {
		result.ValidationErrors = append(result.ValidationErrors, "triage item not found")
		return result, nil
	}
	result.PreviousStatus = item.Status
	result.NewStatus = item.Status
	result.ForbiddenActions = append([]string{}, item.ForbiddenActions...)

	result.ValidationErrors = append(result.ValidationErrors, validateActionRequest(request, decision, item)...)
	if len(result.ValidationErrors) > 0 {
		return result, nil
	}

	applied, err := operations.ApplyFeedbackDecision(item, operations.FeedbackDecisionInput{
		FeedbackDecisionID: request.FeedbackDecisionID,
		Decision:           decision,
		HumanReviewer:      request.HumanReviewer,
		Rationale:          request.Rationale,
		RequiredEvidence:   append([]string{}, request.RequiredEvidence...),
		CreatedAt:          createdAt,
	})
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, err.Error())
		return result, nil
	}

	if err := s.repo.SaveTriageItem(applied.Item); err != nil {
		return result, fmt.Errorf("save triage item: %w", err)
	}
	if err := s.repo.SaveHumanFeedbackDecision(applied.Decision); err != nil {
		return result, fmt.Errorf("save feedback decision: %w", err)
	}
	for _, action := range applied.FollowUpActions {
		if err := s.repo.SaveFollowUpAction(action); err != nil {
			return result, fmt.Errorf("save follow-up action: %w", err)
		}
		result.FollowUpActionIDs = append(result.FollowUpActionIDs, action.ActionID)
	}

	result.NewStatus = applied.Item.Status
	result.FeedbackDecisionID = applied.Decision.FeedbackDecisionID
	result.ForbiddenActions = append([]string{}, applied.Item.ForbiddenActions...)
	result.RequiresHumanApproval = true
	result.AutoApplyAllowed = false
	if request.AutoApplyAllowed {
		result.ValidationWarnings = append(result.ValidationWarnings, "auto_apply_allowed was normalised to false")
	}
	return result, nil
}

func baseActionResult(request ActionRequest, decision operations.HumanDecision, createdAt time.Time) ActionResult {
	feedbackDecisionID := request.FeedbackDecisionID
	if feedbackDecisionID == "" {
		feedbackDecisionID = "feedback_" + request.TriageItemID + "_" + string(decision)
	}
	return ActionResult{
		ActionResultID:        "operator_result_" + feedbackDecisionID,
		TriageItemID:          request.TriageItemID,
		Action:                decision,
		FeedbackDecisionID:    feedbackDecisionID,
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		CreatedAt:             createdAt,
	}
}
