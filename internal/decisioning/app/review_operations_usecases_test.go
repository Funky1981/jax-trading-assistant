package app

import (
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestReviewOperationsActionUseCasesPersistReviewStateOnly(t *testing.T) {
	now := fixedAppNow()
	tests := []struct {
		name              string
		run               func(ReviewOperationsService, ReviewActionRequest) (ReviewActionResult, error)
		item              triage.Item
		request           ReviewActionRequest
		wantStatus        triage.Status
		wantFollowUpCount int
	}{
		{
			name:              "accept suggestion creates manual follow-up",
			run:               ReviewOperationsService.AcceptSuggestion,
			item:              appItem("triage_accept", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen, now),
			request:           appActionRequest("request_accept", "triage_accept", "feedback_accept", "Accepted for manual research follow-up.", nil, now),
			wantStatus:        triage.StatusAccepted,
			wantFollowUpCount: 1,
		},
		{
			name:              "reject suggestion records rationale only",
			run:               ReviewOperationsService.RejectSuggestion,
			item:              appItem("triage_reject", triage.SourceScoringReview, triage.PriorityMedium, triage.StatusOpen, now),
			request:           appActionRequest("request_reject", "triage_reject", "feedback_reject", "Rejected because the evidence is weak.", nil, now),
			wantStatus:        triage.StatusRejected,
			wantFollowUpCount: 0,
		},
		{
			name:              "defer suggestion records manual deferral",
			run:               ReviewOperationsService.DeferSuggestion,
			item:              appItem("triage_defer", triage.SourceNoTradeRuleReview, triage.PriorityMedium, triage.StatusOpen, now),
			request:           appActionRequest("request_defer", "triage_defer", "feedback_defer", "Defer until weekly review.", nil, now),
			wantStatus:        triage.StatusDeferred,
			wantFollowUpCount: 0,
		},
		{
			name:              "request more evidence creates manual research follow-up",
			run:               ReviewOperationsService.RequestMoreEvidence,
			item:              appItem("triage_evidence", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen, now),
			request:           appActionRequest("request_evidence", "triage_evidence", "feedback_evidence", "Need replay and data quality evidence.", []string{"replay comparison", "data quality check"}, now),
			wantStatus:        triage.StatusNeedsMoreEvidence,
			wantFollowUpCount: 1,
		},
		{
			name:              "close no action records closure only",
			run:               ReviewOperationsService.CloseNoAction,
			item:              appItem("triage_close", triage.SourceRiskVetoHelped, triage.PriorityLow, triage.StatusOpen, now),
			request:           appActionRequest("request_close", "triage_close", "feedback_close", "No action required.", nil, now),
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
			service := mustAppService(t, repo)

			got, err := tt.run(service, tt.request)
			if err != nil {
				t.Fatalf("action returned error: %v", err)
			}
			if !got.Result.Succeeded || len(got.Result.ValidationErrors) != 0 {
				t.Fatalf("action result = %#v", got.Result)
			}
			if got.Action.NewStatus != tt.wantStatus {
				t.Fatalf("new status = %s, want %s", got.Action.NewStatus, tt.wantStatus)
			}
			if len(got.Action.FollowUpActionIDs) != tt.wantFollowUpCount {
				t.Fatalf("follow-up action ids = %v, want count %d", got.Action.FollowUpActionIDs, tt.wantFollowUpCount)
			}
			if got.Result.AutoApplyAllowed || !got.Result.RequiresHumanApproval || got.Result.ReadOnly {
				t.Fatalf("action result safety flags = %#v", got.Result)
			}
			assertAppContainsAll(t, got.Result.ForbiddenActions, feedback.ForbiddenActions())

			persisted := mustAppItem(t, repo, tt.item.TriageItemID)
			if persisted.Status != tt.wantStatus || persisted.AutoApplyAllowed || !persisted.RequiresHumanApproval {
				t.Fatalf("persisted item = %#v", persisted)
			}
			if _, ok := repo.GetHumanFeedbackDecision(tt.request.FeedbackDecisionID); !ok {
				t.Fatalf("feedback decision was not persisted")
			}
		})
	}
}

func TestReviewOperationsBlocksLiveExecutionAndAutoApprovalWithoutMutation(t *testing.T) {
	repo := operations.NewMemoryRepository()
	now := fixedAppNow()
	item := appItem("triage_blocked", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen, now)
	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	service := mustAppService(t, repo)

	request := appActionRequest("request_blocked", item.TriageItemID, "feedback_blocked", "Attempt LIVE_READY create_live_order execute_trade broker execution auto approval.", nil, now)
	request.AttemptedActions = []string{"LIVE_READY", "create_live_order", "execute_trade", "broker_execution", "auto_approve"}
	got, err := service.AcceptSuggestion(request)
	if err != nil {
		t.Fatalf("AcceptSuggestion returned error: %v", err)
	}

	if got.Result.Succeeded {
		t.Fatalf("blocked action succeeded: %#v", got.Result)
	}
	joined := strings.ToLower(strings.Join(append(got.Result.ValidationErrors, got.Result.ValidationWarnings...), " "))
	for _, want := range []string{"live", "create_live_order", "execute_trade", "auto_approve"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("validation output %q missing %q", joined, want)
		}
	}
	assertAppContainsAll(t, got.Result.ForbiddenActions, feedback.ForbiddenActions())

	persisted := mustAppItem(t, repo, item.TriageItemID)
	if persisted.Status != triage.StatusOpen {
		t.Fatalf("blocked action mutated item: %#v", persisted)
	}
	if decisions := repo.ListFeedbackDecisionsForTriageItem(item.TriageItemID); len(decisions) != 0 {
		t.Fatalf("blocked action persisted feedback decisions: %#v", decisions)
	}
}

func mustAppService(t *testing.T, repo operations.Repository) ReviewOperationsService {
	t.Helper()
	service, err := NewReviewOperationsService(DefaultReviewOperationsConfig(), repo)
	if err != nil {
		t.Fatalf("NewReviewOperationsService returned error: %v", err)
	}
	return service
}

func appItem(id string, source triage.SourceType, priority triage.Priority, status triage.Status, now time.Time) triage.Item {
	return triage.Item{
		TriageItemID:           id,
		SourceType:             source,
		SourceID:               "source_" + id,
		SourceDecisionID:       "decision_" + id,
		EventID:                "event_" + id,
		Asset:                  "SHEL",
		SetupFamily:            "macro_breakout",
		EventType:              "macro",
		Priority:               priority,
		Status:                 status,
		Reason:                 "manual review required",
		EvidenceRefs:           []string{"review:1", "replay:1"},
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

func appAction(id, triageItemID string, now time.Time) operations.FollowUpAction {
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

func appActionRequest(requestID, itemID, decisionID, rationale string, evidence []string, now time.Time) ReviewActionRequest {
	return ReviewActionRequest{
		RequestID:          requestID,
		TriageItemID:       itemID,
		FeedbackDecisionID: decisionID,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          rationale,
		RequiredEvidence:   evidence,
		CreatedAt:          now,
	}
}

func mustAppItem(t *testing.T, repo operations.Repository, id string) triage.Item {
	t.Helper()
	item, ok := repo.GetTriageItem(id)
	if !ok {
		t.Fatalf("triage item %q not found", id)
	}
	return item
}

func assertAppContainsAll(t *testing.T, got []string, want []string) {
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
