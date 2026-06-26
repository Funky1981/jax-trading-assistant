package readmodel

import (
	"testing"
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
	"jax-trading-assistant/internal/decisioning/workflow"
)

func TestReviewQueueSummaryCountsAndSafetyFields(t *testing.T) {
	repo := operations.NewMemoryRepository()
	now := fixedReadModelNow()
	items := []triage.Item{
		readModelItem("open_critical_overdue", triage.PriorityCritical, triage.StatusOpen, now.Add(-time.Hour)),
		readModelItem("open_high_due", triage.PriorityHigh, triage.StatusOpen, now),
		readModelItem("deferred_high_future", triage.PriorityHigh, triage.StatusDeferred, now.Add(time.Hour)),
		readModelItem("needs_more_evidence", triage.PriorityMedium, triage.StatusNeedsMoreEvidence, now.Add(2*time.Hour)),
	}
	for _, item := range items {
		if err := repo.SaveTriageItem(item); err != nil {
			t.Fatalf("SaveTriageItem returned error: %v", err)
		}
	}
	if err := repo.SaveFollowUpAction(readModelAction("action_one", "open_critical_overdue", now)); err != nil {
		t.Fatalf("SaveFollowUpAction returned error: %v", err)
	}

	got := BuildReviewQueueSummary(repo, Options{GeneratedAt: now, AsOf: now})
	if got.TotalOpen != 2 || got.TotalCritical != 1 || got.TotalHigh != 2 || got.TotalDue != 2 || got.TotalOverdue != 1 {
		t.Fatalf("summary counts = %#v", got)
	}
	if got.TotalNeedsMoreEvidence != 1 || got.TotalDeferred != 1 || got.TotalFollowUpActions != 1 {
		t.Fatalf("summary state/action counts = %#v", got)
	}
	if !got.ReadOnly {
		t.Fatalf("read_only = false")
	}
	assertContainsAllReadModel(t, got.ForbiddenActions, feedback.ForbiddenActions())
}

func TestDetailReadModelsCopySafetyFieldsWithoutMutatingSource(t *testing.T) {
	item := readModelItem("triage_detail", triage.PriorityHigh, triage.StatusOpen, fixedReadModelNow())
	detail, err := BuildTriageItemDetail(item)
	if err != nil {
		t.Fatalf("BuildTriageItemDetail returned error: %v", err)
	}
	detail.EvidenceRefs[0] = "mutated"
	if item.EvidenceRefs[0] == "mutated" {
		t.Fatalf("detail mutated source item evidence refs")
	}
	if !detail.RequiresHumanApproval || detail.AutoApplyAllowed {
		t.Fatalf("detail safety fields = human %v auto %v", detail.RequiresHumanApproval, detail.AutoApplyAllowed)
	}

	action := readModelAction("action_detail", item.TriageItemID, fixedReadModelNow())
	action.AutoApplyAllowed = true
	actionDetail, err := BuildFollowUpActionDetail(action)
	if err != nil {
		t.Fatalf("BuildFollowUpActionDetail returned error: %v", err)
	}
	if !actionDetail.RequiresHumanApproval || actionDetail.AutoApplyAllowed {
		t.Fatalf("action detail safety fields = human %v auto %v", actionDetail.RequiresHumanApproval, actionDetail.AutoApplyAllowed)
	}
}

func TestBatchAndExportSummariesAreReadOnly(t *testing.T) {
	now := fixedReadModelNow()
	item := readModelItem("triage_batch", triage.PriorityCritical, triage.StatusOpen, now)
	batch, err := workflow.BuildReviewBatch([]triage.Item{item}, []operations.FollowUpAction{readModelAction("action_batch", item.TriageItemID, now)}, workflow.BatchOptions{
		BatchID:     "batch_001",
		GeneratedAt: now,
		AsOf:        now,
	})
	if err != nil {
		t.Fatalf("BuildReviewBatch returned error: %v", err)
	}
	batchSummary := BuildReviewBatchSummary(batch)
	if !batchSummary.ReadOnly || !batchSummary.HumanReviewRequired || batchSummary.FollowUpActionCount != 1 {
		t.Fatalf("batch summary = %#v", batchSummary)
	}

	exportResult, err := reviewexport.ExportReviewBatch(batch, reviewexport.ExportOptions{
		ExportID:    "export_001",
		ExportType:  reviewexport.ExportTypeReviewBatch,
		Format:      reviewexport.FormatJSON,
		GeneratedAt: now,
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}
	exportSummary := BuildExportSummary(exportResult)
	if !exportSummary.ReadOnly || exportSummary.AutoApplyAllowed || exportSummary.ContentBytes == 0 {
		t.Fatalf("export summary = %#v", exportSummary)
	}
}

func readModelItem(id string, priority triage.Priority, status triage.Status, dueAt time.Time) triage.Item {
	now := fixedReadModelNow()
	return triage.Item{
		TriageItemID:           id,
		SourceType:             triage.SourceResearchGap,
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
		DueAt:                  dueAt,
	}
}

func readModelAction(id, triageItemID string, now time.Time) operations.FollowUpAction {
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

func fixedReadModelNow() time.Time {
	return time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
}

func assertContainsAllReadModel(t *testing.T, got []string, want []string) {
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
