package operations

import (
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestMemoryRepositorySavesAndRetrievesTriageItem(t *testing.T) {
	repo := NewMemoryRepository()
	item := repositoryItem("triage_open", triage.SourceMissedOpportunity, triage.PriorityHigh, triage.StatusOpen, fixedRepositoryNow().Add(-time.Hour))

	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	got, ok := repo.GetTriageItem(item.TriageItemID)
	if !ok {
		t.Fatalf("GetTriageItem ok = false")
	}
	if got.TriageItemID != item.TriageItemID {
		t.Fatalf("triage item id = %s, want %s", got.TriageItemID, item.TriageItemID)
	}
	if !got.RequiresHumanApproval {
		t.Fatalf("requires human approval = false")
	}
	if got.AutoApplyAllowed {
		t.Fatalf("auto apply allowed = true")
	}
	assertContainsAllOperations(t, got.ForbiddenActions, feedback.ForbiddenActions())
}

func TestMemoryRepositoryListsOpenHighPriorityAndDueItems(t *testing.T) {
	repo := NewMemoryRepository()
	now := fixedRepositoryNow()
	items := []triage.Item{
		repositoryItem("open_low_due", triage.SourceWatchlistReview, triage.PriorityLow, triage.StatusOpen, now.Add(-time.Hour)),
		repositoryItem("closed_high_due", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusClosed, now.Add(-time.Hour)),
		repositoryItem("rejected_critical_due", triage.SourceScoringReview, triage.PriorityCritical, triage.StatusRejected, now.Add(-time.Hour)),
		repositoryItem("open_high_due", triage.SourceRiskVetoTooStrict, triage.PriorityHigh, triage.StatusOpen, now),
		repositoryItem("open_critical_overdue", triage.SourcePaperSetupFailed, triage.PriorityCritical, triage.StatusOpen, now.Add(-2*time.Hour)),
		repositoryItem("open_medium_future", triage.SourceResearchGap, triage.PriorityMedium, triage.StatusOpen, now.Add(2*time.Hour)),
	}
	for _, item := range items {
		if err := repo.SaveTriageItem(item); err != nil {
			t.Fatalf("SaveTriageItem(%s) returned error: %v", item.TriageItemID, err)
		}
	}

	assertIDs(t, repo.ListOpenTriageItems(), []string{"open_critical_overdue", "open_high_due", "open_medium_future", "open_low_due"})
	assertIDs(t, repo.ListHighPriorityTriageItems(), []string{"open_critical_overdue", "open_high_due"})
	assertIDs(t, repo.ListDueTriageItems(now), []string{"open_critical_overdue", "open_high_due", "open_low_due"})
}

func TestMemoryRepositoryPersistsFeedbackDecisionFollowUpActionAndAudit(t *testing.T) {
	repo := NewMemoryRepository()
	item := repositoryItem("triage_feedback", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen, fixedRepositoryNow())
	decision := FeedbackDecision{
		FeedbackDecisionID: "feedback_accept",
		TriageItemID:       item.TriageItemID,
		Decision:           DecisionAcceptSuggestion,
		HumanReviewer:      "reviewer@example.com",
		Rationale:          "Create manual research task.",
		CreatedAt:          fixedRepositoryNow(),
	}
	action := FollowUpAction{
		ActionID:              "action_feedback",
		TriageItemID:          item.TriageItemID,
		ActionType:            ActionCreateResearchTask,
		Description:           "Create a manual research task; do not change trading rules automatically.",
		TargetModule:          "research",
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		Status:                ActionStatusOpen,
		CreatedAt:             fixedRepositoryNow(),
	}

	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	if err := repo.SaveHumanFeedbackDecision(decision); err != nil {
		t.Fatalf("SaveHumanFeedbackDecision returned error: %v", err)
	}
	if err := repo.SaveFollowUpAction(action); err != nil {
		t.Fatalf("SaveFollowUpAction returned error: %v", err)
	}

	gotDecision, ok := repo.GetHumanFeedbackDecision(decision.FeedbackDecisionID)
	if !ok {
		t.Fatalf("GetHumanFeedbackDecision ok = false")
	}
	if !reflect.DeepEqual(gotDecision, decision) {
		t.Fatalf("feedback decision = %#v, want %#v", gotDecision, decision)
	}
	gotAction, ok := repo.GetFollowUpAction(action.ActionID)
	if !ok {
		t.Fatalf("GetFollowUpAction ok = false")
	}
	if gotAction.AutoApplyAllowed {
		t.Fatalf("follow-up action auto apply allowed = true")
	}
	assertContainsAllOperations(t, gotAction.ForbiddenActions, feedback.ForbiddenActions())
	if len(repo.ListFeedbackDecisionsForTriageItem(item.TriageItemID)) != 1 {
		t.Fatalf("feedback decisions for triage item not persisted")
	}
	if len(repo.ListFollowUpActionsForTriageItem(item.TriageItemID)) != 1 {
		t.Fatalf("follow-up actions for triage item not persisted")
	}
	audits := repo.ListOperationAuditRecords()
	if len(audits) < 3 {
		t.Fatalf("audit records = %d, want at least 3", len(audits))
	}
	assertContainsAllOperations(t, audits[len(audits)-1].ForbiddenActions, feedback.ForbiddenActions())
}

func TestMemoryRepositoryNormalizesAutoApplyAndBlocksLivePromotion(t *testing.T) {
	repo := NewMemoryRepository()
	item := repositoryItem("triage_auto_apply", triage.SourceScoringReview, triage.PriorityCritical, triage.StatusOpen, fixedRepositoryNow())
	item.AutoApplyAllowed = true

	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	got, ok := repo.GetTriageItem(item.TriageItemID)
	if !ok {
		t.Fatalf("GetTriageItem ok = false")
	}
	if got.AutoApplyAllowed {
		t.Fatalf("normalized auto apply allowed = true")
	}
	audits := repo.ListOperationAuditRecords()
	assertAuditAction(t, audits, AuditActionAutoApplyBlocked)

	live := repositoryItem("triage_live_ready", triage.SourceScoringReview, triage.PriorityCritical, triage.StatusOpen, fixedRepositoryNow())
	live.SuggestedAction = "promote setup to LIVE_READY"
	if err := repo.SaveTriageItem(live); err == nil {
		t.Fatalf("SaveTriageItem live promotion returned nil error")
	}
}

func repositoryItem(id string, source triage.SourceType, priority triage.Priority, status triage.Status, dueAt time.Time) triage.Item {
	now := fixedRepositoryNow()
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
		EvidenceRefs:           []string{"evidence_" + id},
		SuggestedAction:        "manual follow-up only",
		AllowedFollowUpActions: []string{string(ActionCreateResearchTask)},
		ForbiddenActions:       feedback.ForbiddenActions(),
		RequiresHumanApproval:  true,
		AutoApplyAllowed:       false,
		CreatedAt:              now,
		UpdatedAt:              now,
		DueAt:                  dueAt,
	}
}

func fixedRepositoryNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertIDs(t *testing.T, items []triage.Item, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("items = %d, want %d: %#v", len(items), len(want), items)
	}
	for i, item := range items {
		if item.TriageItemID != want[i] {
			t.Fatalf("item[%d] = %s, want %s", i, item.TriageItemID, want[i])
		}
	}
}

func assertAuditAction(t *testing.T, records []OperationAuditRecord, want AuditAction) {
	t.Helper()
	for _, record := range records {
		if record.Action == want {
			return
		}
	}
	t.Fatalf("audit action %s not found in %#v", want, records)
}
