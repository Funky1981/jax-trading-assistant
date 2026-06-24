package feedback

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/replay"
	"jax-trading-assistant/internal/decisioning/review"
)

func TestBuildReportSummarizesReplayLessonsAndResearchGaps(t *testing.T) {
	got := BuildReport(ReportInput{
		ReportID: "feedback_phase_10",
		ReplayResult: replay.ReplayResult{
			ReplayID:               "replay_phase_10",
			RecordsProcessed:       6,
			NoTradeCount:           1,
			CorrectNoTradeCount:    1,
			MissedOpportunityCount: 1,
			AvoidedLossCount:       1,
			RiskVetoHelpedCount:    1,
			RiskVetoTooStrictCount: 1,
			PaperSetupWorkedCount:  1,
			PaperSetupFailedCount:  1,
			RejectedByRiskCount:    2,
			PaperApprovedCount:     2,
			LessonsGenerated:       7,
			CreatedAt:              fixedFeedbackNow(),
		},
		Lessons: []review.Lesson{
			{LessonID: "lesson_correct_no_trade", LessonType: review.LessonCorrectNoTrade, AppliesToSetupFamily: "macro_conflict", AppliesToEventType: "macro"},
			{LessonID: "lesson_missed", LessonType: review.LessonMissedOpportunity, AppliesToSetupFamily: "commodity_equity", AppliesToEventType: "commodity"},
			{LessonID: "lesson_risk_helped", LessonType: review.LessonRiskVetoHelped, AppliesToSetupFamily: "commodity_equity"},
			{LessonID: "lesson_risk_strict", LessonType: review.LessonRiskVetoTooStrict, AppliesToSetupFamily: "earnings_revision"},
			{LessonID: "lesson_paper_worked", LessonType: review.LessonPaperSetupWorked, AppliesToSetupFamily: "earnings_revision"},
			{LessonID: "lesson_paper_failed", LessonType: review.LessonPaperSetupFailed, AppliesToSetupFamily: "macro_breakout"},
			{LessonID: "lesson_research_weak", LessonType: review.LessonResearchEvidenceInsufficient, AppliesToSetupFamily: "macro_breakout"},
		},
		ResearchEvidence: []ResearchEvidence{
			{EvidenceID: "evidence_weak", SetupFamily: "macro_breakout", Status: "BACKTESTED_WEAK", Summary: "Only weak backtest evidence exists."},
			{EvidenceID: "evidence_missing", SetupFamily: "rates_fx", Status: "", Summary: "No evidence bundle supplied."},
		},
		CreatedAt: fixedFeedbackNow(),
	})

	assertContainsText(t, got.NoTradeFindings, "helped")
	assertContainsText(t, got.NoTradeFindings, "missed opportunity")
	assertContainsText(t, got.RiskVetoFindings, "avoided bad setup")
	assertContainsText(t, got.RiskVetoFindings, "too strict")
	assertContainsText(t, got.PaperOutcomeFindings, "promising")
	assertContainsText(t, got.PaperOutcomeFindings, "weakness")
	assertContainsText(t, got.ResearchGaps, "weak")
	assertContainsText(t, got.ResearchGaps, "missing")

	if !got.RequiresHumanApproval {
		t.Fatalf("requires human approval = false, want true")
	}
	assertContainsAll(t, got.ForbiddenActions, ForbiddenActions())
	if len(got.SuggestedRuleChanges) == 0 {
		t.Fatalf("expected rule suggestions")
	}
	for _, suggestion := range got.SuggestedRuleChanges {
		if !suggestion.RequiresHumanApproval {
			t.Fatalf("suggestion %s requires human approval = false", suggestion.SuggestionID)
		}
		if suggestion.AutoApplyAllowed {
			t.Fatalf("suggestion %s auto apply allowed = true", suggestion.SuggestionID)
		}
		assertContainsAll(t, suggestion.ForbiddenActions, ForbiddenActions())
	}
	assertSuggestionType(t, got.SuggestedRuleChanges, SuggestionNoTradeRuleReview)
	assertSuggestionType(t, got.SuggestedRuleChanges, SuggestionRiskThresholdReview)
	assertSuggestionType(t, got.SuggestedRuleChanges, SuggestionSetupFamilyResearch)
}

func TestBuildReportBlocksLivePromotionAndPreservesForbiddenActions(t *testing.T) {
	got := BuildReport(ReportInput{
		ReportID:                  "feedback_live_blocked",
		ReplayResult:              replay.ReplayResult{ReplayID: "replay_live_blocked", RecordsProcessed: 1},
		AttemptedPromotion:        "LIVE_READY",
		AttemptedForbiddenActions: []string{"execute_trade", "create_live_order", "auto_approve"},
		CreatedAt:                 fixedFeedbackNow(),
	})

	if len(got.Errors) == 0 {
		t.Fatalf("expected live promotion error")
	}
	assertContainsText(t, got.Errors, "LIVE_READY")
	assertContainsAll(t, got.ForbiddenActions, ForbiddenActions())
}

func fixedFeedbackNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertContainsText(t *testing.T, got []string, want string) {
	t.Helper()
	for _, item := range got {
		if contains(item, want) {
			return
		}
	}
	t.Fatalf("missing text %q in %v", want, got)
}

func assertContainsAll(t *testing.T, got []string, want []string) {
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

func assertSuggestionType(t *testing.T, suggestions []RuleSuggestion, want SuggestionType) {
	t.Helper()
	for _, suggestion := range suggestions {
		if suggestion.SuggestionType == want {
			return
		}
	}
	t.Fatalf("missing suggestion type %s in %#v", want, suggestions)
}
