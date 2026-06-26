package operator

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/readmodel"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestOperatorServiceReadModels(t *testing.T) {
	repo := operations.NewMemoryRepository()
	service := NewService(repo)
	now := fixedOperatorNow()
	item := operatorItem("triage_read", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen)
	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	if err := repo.SaveFollowUpAction(operatorAction("action_read", item.TriageItemID)); err != nil {
		t.Fatalf("SaveFollowUpAction returned error: %v", err)
	}

	queue := service.GetReviewQueueSummary(readmodelOptions(now))
	if queue.TotalOpen != 1 || queue.TotalFollowUpActions != 1 || !queue.ReadOnly {
		t.Fatalf("queue summary = %#v", queue)
	}
	detail, ok, err := service.GetTriageItemDetail(item.TriageItemID)
	if err != nil || !ok {
		t.Fatalf("GetTriageItemDetail ok=%v err=%v", ok, err)
	}
	if detail.AutoApplyAllowed || !detail.RequiresHumanApproval {
		t.Fatalf("detail safety fields = %#v", detail)
	}
	actionDetail, ok, err := service.GetFollowUpActionDetail("action_read")
	if err != nil || !ok {
		t.Fatalf("GetFollowUpActionDetail ok=%v err=%v", ok, err)
	}
	if actionDetail.AutoApplyAllowed || !actionDetail.RequiresHumanApproval {
		t.Fatalf("action detail safety fields = %#v", actionDetail)
	}
}

func TestOperatorActionsPersistRecordsOnly(t *testing.T) {
	tests := []struct {
		name              string
		action            func(Service, ActionRequest) (ActionResult, error)
		item              triage.Item
		request           ActionRequest
		wantStatus        triage.Status
		wantFollowUpCount int
	}{
		{
			name:              "accept suggestion creates manual follow-up",
			action:            Service.AcceptSuggestion,
			item:              operatorItem("triage_accept", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen),
			request:           operatorRequest("triage_accept", "feedback_accept", "Accepted for manual research follow-up.", nil),
			wantStatus:        triage.StatusAccepted,
			wantFollowUpCount: 1,
		},
		{
			name:              "reject suggestion records rationale with no action",
			action:            Service.RejectSuggestion,
			item:              operatorItem("triage_reject", triage.SourceScoringReview, triage.PriorityMedium, triage.StatusOpen),
			request:           operatorRequest("triage_reject", "feedback_reject", "Rejected because evidence was weak.", nil),
			wantStatus:        triage.StatusRejected,
			wantFollowUpCount: 0,
		},
		{
			name:              "defer suggestion records reason",
			action:            Service.DeferSuggestion,
			item:              operatorItem("triage_defer", triage.SourceNoTradeRuleReview, triage.PriorityMedium, triage.StatusOpen),
			request:           operatorRequest("triage_defer", "feedback_defer", "Defer until weekly review.", nil),
			wantStatus:        triage.StatusDeferred,
			wantFollowUpCount: 0,
		},
		{
			name:              "request more evidence creates research action",
			action:            Service.RequestMoreEvidence,
			item:              operatorItem("triage_evidence", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen),
			request:           operatorRequest("triage_evidence", "feedback_evidence", "Need more evidence.", []string{"replay comparison"}),
			wantStatus:        triage.StatusNeedsMoreEvidence,
			wantFollowUpCount: 1,
		},
		{
			name:              "close no action creates no follow-up",
			action:            Service.CloseNoAction,
			item:              operatorItem("triage_close", triage.SourceRiskVetoHelped, triage.PriorityLow, triage.StatusOpen),
			request:           operatorRequest("triage_close", "feedback_close", "No action required.", nil),
			wantStatus:        triage.StatusClosed,
			wantFollowUpCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := operations.NewMemoryRepository()
			if err := repo.SaveTriageItem(tt.item); err != nil {
				t.Fatalf("SaveTriageItem returned error: %v", err)
			}
			service := NewService(repo)

			got, err := tt.action(service, tt.request)
			if err != nil {
				t.Fatalf("operator action returned error: %v", err)
			}
			if len(got.ValidationErrors) != 0 {
				t.Fatalf("validation errors = %v", got.ValidationErrors)
			}
			if got.PreviousStatus != triage.StatusOpen || got.NewStatus != tt.wantStatus {
				t.Fatalf("status transition = %s -> %s, want OPEN -> %s", got.PreviousStatus, got.NewStatus, tt.wantStatus)
			}
			if len(got.FollowUpActionIDs) != tt.wantFollowUpCount {
				t.Fatalf("follow-up action ids = %v, want count %d", got.FollowUpActionIDs, tt.wantFollowUpCount)
			}
			item, ok := repo.GetTriageItem(tt.item.TriageItemID)
			if !ok || item.Status != tt.wantStatus {
				t.Fatalf("persisted item ok=%v item=%#v", ok, item)
			}
			if _, ok := repo.GetHumanFeedbackDecision(tt.request.FeedbackDecisionID); !ok {
				t.Fatalf("feedback decision was not persisted")
			}
			if item.AutoApplyAllowed || !item.RequiresHumanApproval {
				t.Fatalf("persisted item safety fields = %#v", item)
			}
			assertContainsAllOperator(t, item.ForbiddenActions, feedback.ForbiddenActions())
		})
	}
}

func TestOperatorValidationBlocksAutoApplyAndLivePromotionWithoutMutation(t *testing.T) {
	tests := []struct {
		name          string
		request       ActionRequest
		wantErrorText string
	}{
		{
			name: "auto apply rejected",
			request: func() ActionRequest {
				req := operatorRequest("triage_blocked", "feedback_auto", "Attempt auto apply.", nil)
				req.AutoApplyAllowed = true
				return req
			}(),
			wantErrorText: "auto_apply_allowed",
		},
		{
			name: "live order action blocked",
			request: func() ActionRequest {
				req := operatorRequest("triage_blocked", "feedback_live", "Attempt create_live_order.", nil)
				req.AttemptedActions = []string{"create_live_order", "LIVE_READY"}
				return req
			}(),
			wantErrorText: "blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := operations.NewMemoryRepository()
			item := operatorItem("triage_blocked", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen)
			if err := repo.SaveTriageItem(item); err != nil {
				t.Fatalf("SaveTriageItem returned error: %v", err)
			}
			got, err := NewService(repo).AcceptSuggestion(tt.request)
			if err != nil {
				t.Fatalf("AcceptSuggestion returned error: %v", err)
			}
			if len(got.ValidationErrors) == 0 {
				t.Fatalf("validation errors empty, want %q", tt.wantErrorText)
			}
			if !strings.Contains(strings.Join(got.ValidationErrors, " "), tt.wantErrorText) {
				t.Fatalf("validation errors = %v, want containing %q", got.ValidationErrors, tt.wantErrorText)
			}
			persisted, ok := repo.GetTriageItem(item.TriageItemID)
			if !ok || persisted.Status != triage.StatusOpen {
				t.Fatalf("item mutated after blocked action: ok=%v item=%#v", ok, persisted)
			}
			if decisions := repo.ListFeedbackDecisionsForTriageItem(item.TriageItemID); len(decisions) != 0 {
				t.Fatalf("feedback decisions persisted for blocked action: %#v", decisions)
			}
		})
	}
}

func operatorItem(id string, source triage.SourceType, priority triage.Priority, status triage.Status) triage.Item {
	now := fixedOperatorNow()
	return triage.Item{
		TriageItemID:           id,
		SourceType:             source,
		SourceID:               "source_" + id,
		EventID:                "event_" + id,
		Asset:                  "SHEL",
		SetupFamily:            "macro_breakout",
		EventType:              "macro",
		Priority:               priority,
		Status:                 status,
		Reason:                 "manual review required",
		EvidenceRefs:           []string{"evidence_" + id},
		SuggestedAction:        "manual follow-up only",
		AllowedFollowUpActions: []string{string(operations.ActionCreateResearchTask)},
		ForbiddenActions:       feedback.ForbiddenActions(),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		CreatedAt:              now,
		UpdatedAt:              now,
		DueAt:                  now,
	}
}

func operatorAction(id, triageItemID string) operations.FollowUpAction {
	now := fixedOperatorNow()
	return operations.FollowUpAction{
		ActionID:              id,
		TriageItemID:          triageItemID,
		ActionType:            operations.ActionCreateResearchTask,
		Description:           "Create a manual research task; do not change trading rules automatically.",
		TargetModule:          "research",
		TargetSetupFamily:     "macro_breakout",
		TargetEventType:       "macro",
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		Status:                operations.ActionStatusOpen,
		CreatedAt:             now,
	}
}

func operatorRequest(itemID, decisionID, rationale string, evidence []string) ActionRequest {
	return ActionRequest{
		TriageItemID:       itemID,
		FeedbackDecisionID: decisionID,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          rationale,
		RequiredEvidence:   evidence,
		CreatedAt:          fixedOperatorNow(),
	}
}

func readmodelOptions(now time.Time) readmodel.Options {
	return readmodel.Options{GeneratedAt: now, AsOf: now}
}

func fixedOperatorNow() time.Time {
	return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
}

func assertContainsAllOperator(t *testing.T, got []string, want []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, item := range got {
		seen[item] = true
	}
	for _, item := range want {
		if !seen[item] {
			t.Fatalf("missing %q in %v", item, got)
		}
	}
}
