package operations

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestAcceptedSuggestionCreatesFollowUpActionRecordOnly(t *testing.T) {
	now := fixedOperationsNow()

	tests := []struct {
		name       string
		item       triage.Item
		wantAction ActionType
	}{
		{
			name:       "research gap creates research follow-up action after acceptance",
			item:       operationsItem("triage_research", triage.SourceResearchGap, "research", "macro_breakout", "macro", triage.PriorityHigh),
			wantAction: ActionCreateResearchTask,
		},
		{
			name:       "scoring review accepted but not applied",
			item:       operationsItem("triage_scoring", triage.SourceScoringReview, "decision_core", "earnings_revision", "earnings", triage.PriorityMedium),
			wantAction: ActionReviewScoringRule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplyFeedbackDecision(tt.item, FeedbackDecisionInput{
				FeedbackDecisionID: "feedback_decision_" + tt.item.TriageItemID,
				Decision:           DecisionAcceptSuggestion,
				HumanReviewer:      "reviewer@example.com",
				Rationale:          "Accepted for manual follow-up only.",
				CreatedAt:          now,
			})
			if err != nil {
				t.Fatalf("ApplyFeedbackDecision returned error: %v", err)
			}

			if result.Item.Status != triage.StatusAccepted {
				t.Fatalf("status = %s, want %s", result.Item.Status, triage.StatusAccepted)
			}
			if len(result.FollowUpActions) != 1 {
				t.Fatalf("follow-up actions = %d, want 1", len(result.FollowUpActions))
			}
			action := result.FollowUpActions[0]
			if action.ActionType != tt.wantAction {
				t.Fatalf("action type = %s, want %s", action.ActionType, tt.wantAction)
			}
			if action.AutoApplyAllowed {
				t.Fatalf("action auto apply allowed = true")
			}
			if !action.RequiresHumanApproval {
				t.Fatalf("action requires human approval = false")
			}
			assertContainsAllOperations(t, action.ForbiddenActions, feedback.ForbiddenActions())
		})
	}
}

func TestRejectedSuggestionRecordsRationaleAndCreatesNoAction(t *testing.T) {
	result, err := ApplyFeedbackDecision(operationsItem("triage_reject", triage.SourceNoTradeRuleReview, "review", "macro_conflict", "macro", triage.PriorityMedium), FeedbackDecisionInput{
		FeedbackDecisionID: "feedback_reject",
		Decision:           DecisionRejectSuggestion,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Evidence did not justify a rule review.",
		CreatedAt:          fixedOperationsNow(),
	})
	if err != nil {
		t.Fatalf("ApplyFeedbackDecision returned error: %v", err)
	}

	if result.Item.Status != triage.StatusRejected {
		t.Fatalf("status = %s, want %s", result.Item.Status, triage.StatusRejected)
	}
	if result.Decision.Rationale == "" {
		t.Fatalf("rationale was not recorded")
	}
	if len(result.FollowUpActions) != 0 {
		t.Fatalf("follow-up actions = %d, want 0", len(result.FollowUpActions))
	}
}

func TestRequestMoreEvidenceCreatesEvidenceFollowUpAction(t *testing.T) {
	result, err := ApplyFeedbackDecision(operationsItem("triage_more_evidence", triage.SourceDataQualityReview, "data", "rates_fx", "macro", triage.PriorityHigh), FeedbackDecisionInput{
		FeedbackDecisionID: "feedback_more_evidence",
		Decision:           DecisionRequestMoreEvidence,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Need cleaner event data before deciding.",
		RequiredEvidence:   []string{"validated data source", "replay comparison"},
		CreatedAt:          fixedOperationsNow(),
	})
	if err != nil {
		t.Fatalf("ApplyFeedbackDecision returned error: %v", err)
	}

	if result.Item.Status != triage.StatusNeedsMoreEvidence {
		t.Fatalf("status = %s, want %s", result.Item.Status, triage.StatusNeedsMoreEvidence)
	}
	if len(result.Decision.RequiredEvidence) != 2 {
		t.Fatalf("required evidence = %v", result.Decision.RequiredEvidence)
	}
	if len(result.FollowUpActions) != 1 {
		t.Fatalf("follow-up actions = %d, want 1", len(result.FollowUpActions))
	}
	if result.FollowUpActions[0].ActionType != ActionReviewDataQuality {
		t.Fatalf("action type = %s, want %s", result.FollowUpActions[0].ActionType, ActionReviewDataQuality)
	}
}

func TestCloseNoActionClosesWithoutFollowUpAction(t *testing.T) {
	result, err := ApplyFeedbackDecision(operationsItem("triage_close", triage.SourceRiskVetoHelped, "risk", "commodity_equity", "commodity", triage.PriorityLow), FeedbackDecisionInput{
		FeedbackDecisionID: "feedback_close",
		Decision:           DecisionCloseNoAction,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Risk veto worked as intended.",
		CreatedAt:          fixedOperationsNow(),
	})
	if err != nil {
		t.Fatalf("ApplyFeedbackDecision returned error: %v", err)
	}
	if result.Item.Status != triage.StatusClosed {
		t.Fatalf("status = %s, want %s", result.Item.Status, triage.StatusClosed)
	}
	if len(result.FollowUpActions) != 0 {
		t.Fatalf("follow-up actions = %d, want 0", len(result.FollowUpActions))
	}
}

func TestCriticalPriorityDoesNotAutoApply(t *testing.T) {
	result, err := ApplyFeedbackDecision(operationsItem("triage_critical", triage.SourceRiskVetoTooStrict, "risk", "macro_breakout", "macro", triage.PriorityCritical), FeedbackDecisionInput{
		FeedbackDecisionID: "feedback_critical",
		Decision:           DecisionAcceptSuggestion,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Review critical issue manually.",
		CreatedAt:          fixedOperationsNow(),
	})
	if err != nil {
		t.Fatalf("ApplyFeedbackDecision returned error: %v", err)
	}

	if result.Item.AutoApplyAllowed {
		t.Fatalf("critical item auto apply allowed = true")
	}
	if result.FollowUpActions[0].AutoApplyAllowed {
		t.Fatalf("critical follow-up auto apply allowed = true")
	}
	assertContainsAllOperations(t, result.FollowUpActions[0].ForbiddenActions, feedback.ForbiddenActions())
}

func TestLivePromotionBlocked(t *testing.T) {
	_, err := ApplyFeedbackDecision(triage.Item{
		TriageItemID:           "triage_live",
		SourceType:             triage.SourceScoringReview,
		SourceID:               "suggestion_live",
		Priority:               triage.PriorityCritical,
		Status:                 triage.StatusOpen,
		Reason:                 "attempts LIVE_READY promotion",
		SuggestedAction:        "approve live trading and promote to LIVE_READY",
		AllowedFollowUpActions: []string{string(ActionReviewScoringRule)},
		ForbiddenActions:       feedback.ForbiddenActions(),
		RequiresHumanApproval:  true,
		CreatedAt:              fixedOperationsNow(),
		UpdatedAt:              fixedOperationsNow(),
		DueAt:                  fixedOperationsNow(),
	}, FeedbackDecisionInput{
		FeedbackDecisionID: "feedback_live",
		Decision:           DecisionAcceptSuggestion,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Should be blocked.",
		CreatedAt:          fixedOperationsNow(),
	})
	if err == nil {
		t.Fatalf("ApplyFeedbackDecision returned nil, want live promotion error")
	}
}

func operationsItem(id string, source triage.SourceType, module, setupFamily, eventType string, priority triage.Priority) triage.Item {
	now := fixedOperationsNow()
	return triage.Item{
		TriageItemID:           id,
		SourceType:             source,
		SourceID:               "source_" + id,
		SetupFamily:            setupFamily,
		EventType:              eventType,
		Priority:               priority,
		Status:                 triage.StatusOpen,
		Reason:                 "manual review required",
		SuggestedAction:        "manual follow-up only",
		AllowedFollowUpActions: []string{},
		ForbiddenActions:       feedback.ForbiddenActions(),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		CreatedAt:              now,
		UpdatedAt:              now,
		DueAt:                  now.Add(24 * time.Hour),
		Asset:                  "SHEL",
	}
}

func fixedOperationsNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertContainsAllOperations(t *testing.T, got []string, want []string) {
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
