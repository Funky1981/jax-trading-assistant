package operations

import (
	"reflect"
	"strings"
	"testing"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestGenerateReviewOperationsReportCountsDeterministically(t *testing.T) {
	repo := NewMemoryRepository()
	now := fixedRepositoryNow()
	items := []triage.Item{
		repositoryItem("triage_open", triage.SourceMissedOpportunity, triage.PriorityHigh, triage.StatusOpen, now),
		repositoryItem("triage_rejected", triage.SourceMissedOpportunity, triage.PriorityMedium, triage.StatusRejected, now.Add(24*60*60*1000000000)),
		repositoryItem("triage_more_evidence", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusNeedsMoreEvidence, now.Add(-2*60*60*1000000000)),
		repositoryItem("triage_closed", triage.SourceRiskVetoTooStrict, triage.PriorityLow, triage.StatusClosed, now.Add(2*60*60*1000000000)),
		repositoryItem("triage_accepted", triage.SourcePaperSetupFailed, triage.PriorityCritical, triage.StatusAccepted, now.Add(-1*60*60*1000000000)),
		repositoryItem("triage_deferred", triage.SourceResearchGap, triage.PriorityLow, triage.StatusDeferred, now.Add(3*60*60*1000000000)),
	}
	for _, item := range items {
		if err := repo.SaveTriageItem(item); err != nil {
			t.Fatalf("SaveTriageItem(%s) returned error: %v", item.TriageItemID, err)
		}
	}
	if err := repo.SaveHumanFeedbackDecision(FeedbackDecision{
		FeedbackDecisionID: "feedback_rejected",
		TriageItemID:       "triage_rejected",
		Decision:           DecisionRejectSuggestion,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Insufficient evidence.",
		CreatedAt:          now,
	}); err != nil {
		t.Fatalf("SaveHumanFeedbackDecision returned error: %v", err)
	}
	if err := repo.SaveFollowUpAction(FollowUpAction{
		ActionID:              "action_more_evidence",
		TriageItemID:          "triage_more_evidence",
		ActionType:            ActionReviewDataQuality,
		Description:           "Review source data quality manually.",
		TargetModule:          "data_quality",
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		Status:                ActionStatusOpen,
		CreatedAt:             now,
	}); err != nil {
		t.Fatalf("SaveFollowUpAction returned error: %v", err)
	}
	autoApply := repositoryItem("triage_auto_apply_report", triage.SourceScoringReview, triage.PriorityCritical, triage.StatusOpen, now)
	autoApply.AutoApplyAllowed = true
	if err := repo.SaveTriageItem(autoApply); err != nil {
		t.Fatalf("SaveTriageItem auto apply returned error: %v", err)
	}

	got := GenerateReviewOperationsReport(repo, ReportOptions{
		ReportID:    "review_ops_report_2026_06_24",
		GeneratedAt: now,
		AsOf:        now,
	})

	if got.ReportID != "review_ops_report_2026_06_24" {
		t.Fatalf("report id = %s", got.ReportID)
	}
	if got.TotalTriageItems != 7 ||
		got.OpenCount != 2 ||
		got.AcceptedCount != 1 ||
		got.RejectedCount != 1 ||
		got.DeferredCount != 1 ||
		got.NeedsMoreEvidenceCount != 1 ||
		got.ClosedCount != 1 {
		t.Fatalf("unexpected status counts: %#v", got)
	}
	if got.CriticalCount != 2 || got.HighCount != 2 || got.MediumCount != 1 || got.LowCount != 2 {
		t.Fatalf("unexpected priority counts: %#v", got)
	}
	if got.DueCount != 4 || got.OverdueCount != 2 {
		t.Fatalf("due counts = due %d overdue %d, want 4 and 2", got.DueCount, got.OverdueCount)
	}
	if got.ResearchGapCount != 2 || got.MissedOpportunityCount != 2 || got.RiskVetoTooStrictCount != 1 || got.PaperSetupFailedCount != 1 {
		t.Fatalf("unexpected source counts: %#v", got)
	}
	if got.FollowUpActionCount != 1 || got.ActionsRequiringHumanApproval != 1 {
		t.Fatalf("action counts = %#v", got)
	}
	if got.AutoApplyBlockedCount != 1 {
		t.Fatalf("auto apply blocked count = %d, want 1", got.AutoApplyBlockedCount)
	}
	if !reflect.DeepEqual(got.ForbiddenActions, feedback.ForbiddenActions()) {
		t.Fatalf("forbidden actions = %#v, want %#v", got.ForbiddenActions, feedback.ForbiddenActions())
	}
	if got.Summary == "" || len(got.Warnings) == 0 {
		t.Fatalf("summary or warnings missing: %#v", got)
	}
	for _, value := range append([]string{got.Summary}, got.Warnings...) {
		normalized := strings.ToLower(value)
		if strings.Contains(normalized, "chain-of-thought") || strings.Contains(normalized, "hidden reasoning") {
			t.Fatalf("report leaked hidden reasoning marker: %q", value)
		}
	}
}

func TestGenerateReviewOperationsReportBlocksLiveReadinessAndPreservesForbiddenActions(t *testing.T) {
	repo := NewMemoryRepository()
	now := fixedRepositoryNow()
	item := repositoryItem("triage_live_report", triage.SourceScoringReview, triage.PriorityCritical, triage.StatusOpen, now)
	item.SuggestedAction = "approve live trading"
	if err := repo.SaveTriageItem(item); err == nil {
		t.Fatalf("SaveTriageItem returned nil error for live approval")
	}

	got := GenerateReviewOperationsReport(repo, ReportOptions{
		ReportID:    "review_ops_empty",
		GeneratedAt: now,
		AsOf:        now,
	})

	if got.TotalTriageItems != 0 {
		t.Fatalf("total triage items = %d, want 0", got.TotalTriageItems)
	}
	assertContainsAllOperations(t, got.ForbiddenActions, feedback.ForbiddenActions())
	if got.ActionsRequiringHumanApproval != 0 {
		t.Fatalf("actions requiring human approval = %d, want 0", got.ActionsRequiringHumanApproval)
	}
}
