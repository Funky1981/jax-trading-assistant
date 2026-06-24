package triage

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/review"
)

func TestBuildItemFromFeedbackSuggestion(t *testing.T) {
	now := fixedTriageNow()

	tests := []struct {
		name         string
		suggestion   feedback.RuleSuggestion
		wantSource   SourceType
		wantPriority Priority
		wantAction   string
	}{
		{
			name: "missed opportunity becomes human triage item",
			suggestion: feedback.RuleSuggestion{
				SuggestionID:      "suggestion_missed",
				SourceLessonIDs:   []string{"lesson_missed"},
				SuggestionType:    feedback.SuggestionNoTradeRuleReview,
				TargetModule:      "review",
				TargetSetupFamily: "commodity_equity",
				TargetEventType:   "commodity",
				Summary:           "missed opportunity after no-trade",
				Rationale:         "Clean setup formed after initial rejection.",
				EvidenceRefs:      []string{"review:rev_missed"},
				ForbiddenActions:  feedback.ForbiddenActions(),
			},
			wantSource:   SourceMissedOpportunity,
			wantPriority: PriorityHigh,
			wantAction:   "Review no-trade rule before any strategy change.",
		},
		{
			name: "risk veto too strict becomes high priority triage item",
			suggestion: feedback.RuleSuggestion{
				SuggestionID:      "suggestion_risk_strict",
				SourceLessonIDs:   []string{"lesson_risk_strict"},
				SuggestionType:    feedback.SuggestionRiskThresholdReview,
				TargetModule:      "risk",
				TargetSetupFamily: "earnings_revision",
				Summary:           "risk veto too strict",
				Rationale:         "Veto blocked a setup that later met evidence.",
				EvidenceRefs:      []string{"review:rev_risk"},
			},
			wantSource:   SourceRiskVetoTooStrict,
			wantPriority: PriorityHigh,
			wantAction:   "Review risk threshold; do not change veto rules automatically.",
		},
		{
			name: "paper setup failed becomes high priority triage item",
			suggestion: feedback.RuleSuggestion{
				SuggestionID:      "suggestion_paper_failed",
				SourceLessonIDs:   []string{"lesson_paper_failed"},
				SuggestionType:    feedback.SuggestionSetupFamilyResearch,
				TargetModule:      "research",
				TargetSetupFamily: "macro_breakout",
				Summary:           "paper setup failed",
				Rationale:         "Paper setup hit invalidation quickly.",
				EvidenceRefs:      []string{"paper:pt_failed"},
			},
			wantSource:   SourcePaperSetupFailed,
			wantPriority: PriorityHigh,
			wantAction:   "Create research task before promotion or rule changes.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item, err := NewItemFromSuggestion(tt.suggestion, now)
			if err != nil {
				t.Fatalf("NewItemFromSuggestion returned error: %v", err)
			}

			if item.Status != StatusOpen {
				t.Fatalf("status = %s, want %s", item.Status, StatusOpen)
			}
			if item.SourceType != tt.wantSource {
				t.Fatalf("source type = %s, want %s", item.SourceType, tt.wantSource)
			}
			if item.Priority != tt.wantPriority {
				t.Fatalf("priority = %s, want %s", item.Priority, tt.wantPriority)
			}
			if !item.RequiresHumanApproval {
				t.Fatalf("requires human approval = false")
			}
			if item.AutoApplyAllowed {
				t.Fatalf("auto apply allowed = true")
			}
			assertContainsAllTriage(t, item.ForbiddenActions, feedback.ForbiddenActions())
			if item.SuggestedAction != tt.wantAction {
				t.Fatalf("suggested action = %q, want %q", item.SuggestedAction, tt.wantAction)
			}
		})
	}
}

func TestBuildItemFromReviewLesson(t *testing.T) {
	item, err := NewItemFromLesson(review.Lesson{
		LessonID:             "lesson_risk_helped",
		DecisionID:           "decision_001",
		EventID:              "event_001",
		LessonType:           review.LessonRiskVetoHelped,
		LessonSummary:        "risk veto avoided loss",
		EvidenceRefs:         []string{"review:rev_001"},
		AppliesToSetupFamily: "macro_conflict",
		AppliesToEventType:   "macro",
		CreatedAt:            fixedTriageNow(),
	}, fixedTriageNow())
	if err != nil {
		t.Fatalf("NewItemFromLesson returned error: %v", err)
	}

	if item.SourceType != SourceRiskVetoHelped {
		t.Fatalf("source type = %s, want %s", item.SourceType, SourceRiskVetoHelped)
	}
	if item.Priority != PriorityLow {
		t.Fatalf("priority = %s, want %s", item.Priority, PriorityLow)
	}
	if item.SourceDecisionID != "decision_001" || item.EventID != "event_001" {
		t.Fatalf("lesson identifiers not preserved: %+v", item)
	}
	assertContainsAllTriage(t, item.ForbiddenActions, feedback.ForbiddenActions())
}

func TestQueueOrdersOpenItemsByPriorityAndDueDate(t *testing.T) {
	now := fixedTriageNow()
	queue := NewQueue([]Item{
		{TriageItemID: "low_due", SourceType: SourceWatchlistReview, Priority: PriorityLow, Status: StatusOpen, RequiresHumanApproval: true, CreatedAt: now, UpdatedAt: now, DueAt: now.Add(time.Hour), ForbiddenActions: feedback.ForbiddenActions()},
		{TriageItemID: "high_later", SourceType: SourceDataQualityReview, Priority: PriorityHigh, Status: StatusOpen, RequiresHumanApproval: true, CreatedAt: now, UpdatedAt: now, DueAt: now.Add(2 * time.Hour), ForbiddenActions: feedback.ForbiddenActions()},
		{TriageItemID: "critical_later", SourceType: SourceScoringReview, Priority: PriorityCritical, Status: StatusOpen, RequiresHumanApproval: true, CreatedAt: now, UpdatedAt: now, DueAt: now.Add(3 * time.Hour), ForbiddenActions: feedback.ForbiddenActions()},
		{TriageItemID: "accepted", SourceType: SourceResearchGap, Priority: PriorityCritical, Status: StatusAccepted, RequiresHumanApproval: true, CreatedAt: now, UpdatedAt: now, DueAt: now, ForbiddenActions: feedback.ForbiddenActions()},
	})

	got, err := queue.OpenItems(now.Add(24 * time.Hour))
	if err != nil {
		t.Fatalf("OpenItems returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("open items = %d, want 3", len(got))
	}
	if got[0].TriageItemID != "critical_later" || got[1].TriageItemID != "high_later" || got[2].TriageItemID != "low_due" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestValidationBlocksLiveReadyPromotionAndAutoApply(t *testing.T) {
	item := Item{
		TriageItemID:           "triage_live_ready",
		SourceType:             SourceScoringReview,
		SourceID:               "suggestion_live_ready",
		Priority:               PriorityCritical,
		Status:                 StatusOpen,
		Reason:                 "attempts LIVE_READY promotion",
		SuggestedAction:        "promote to LIVE_READY",
		AllowedFollowUpActions: []string{"CREATE_RESEARCH_TASK"},
		ForbiddenActions:       []string{"execute_trade", "create_live_order", "auto_approve"},
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       true,
		CreatedAt:              fixedTriageNow(),
		UpdatedAt:              fixedTriageNow(),
		DueAt:                  fixedTriageNow(),
	}

	err := ValidateItem(item)
	if err == nil {
		t.Fatalf("ValidateItem returned nil, want error")
	}
	assertContainsAllTriage(t, item.ForbiddenActions, feedback.ForbiddenActions())
}

func fixedTriageNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertContainsAllTriage(t *testing.T, got []string, want []string) {
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
