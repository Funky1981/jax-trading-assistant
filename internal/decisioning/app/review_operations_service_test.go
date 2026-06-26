package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestReviewOperationsServiceConstructsWithSafetyDefaults(t *testing.T) {
	service, err := NewReviewOperationsService(DefaultReviewOperationsConfig(), operations.NewMemoryRepository())
	if err != nil {
		t.Fatalf("NewReviewOperationsService returned error: %v", err)
	}

	got := service.SafetyDefaults()
	if got.AutoApplyAllowed {
		t.Fatalf("auto apply allowed = true")
	}
	if !got.RequiresHumanApproval || !got.ReadOnly {
		t.Fatalf("safety defaults = %#v", got)
	}
	assertAppContainsAll(t, got.ForbiddenActions, feedback.ForbiddenActions())
}

func TestReviewOperationsFactoryConstructsWithInMemoryDependencies(t *testing.T) {
	service, err := NewInMemoryReviewOperationsService(DefaultReviewOperationsConfig())
	if err != nil {
		t.Fatalf("NewInMemoryReviewOperationsService returned error: %v", err)
	}

	got, err := service.GetReviewQueueSummary(ReviewQueueRequest{
		RequestID:   "request_factory_queue",
		GeneratedAt: fixedAppNow(),
		AsOf:        fixedAppNow(),
	})
	if err != nil {
		t.Fatalf("GetReviewQueueSummary returned error: %v", err)
	}
	if !got.Result.Succeeded || !got.Result.ReadOnly || got.Result.AutoApplyAllowed {
		t.Fatalf("factory queue result safety = %#v", got.Result)
	}
}

func TestReviewOperationsServiceRejectsNilRepository(t *testing.T) {
	_, err := NewReviewOperationsService(DefaultReviewOperationsConfig(), nil)
	if err == nil {
		t.Fatalf("NewReviewOperationsService returned nil error for nil repository")
	}
}

func TestReviewOperationsReadMethodsAreReadOnly(t *testing.T) {
	repo := operations.NewMemoryRepository()
	now := fixedAppNow()
	item := appItem("triage_read", triage.SourceResearchGap, triage.PriorityHigh, triage.StatusOpen, now)
	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	original := mustAppItem(t, repo, item.TriageItemID)
	service := mustAppService(t, repo)

	queue, err := service.GetReviewQueueSummary(ReviewQueueRequest{
		RequestID:   "request_queue",
		GeneratedAt: now,
		AsOf:        now,
	})
	if err != nil {
		t.Fatalf("GetReviewQueueSummary returned error: %v", err)
	}
	if queue.Summary.TotalOpen != 1 || !queue.Result.ReadOnly || !queue.Result.Succeeded {
		t.Fatalf("queue summary result = %#v", queue)
	}
	assertAppContainsAll(t, queue.Result.ForbiddenActions, feedback.ForbiddenActions())

	detail, err := service.GetTriageItemDetail(TriageDetailRequest{
		RequestID:    "request_detail",
		TriageItemID: item.TriageItemID,
		CreatedAt:    now,
	})
	if err != nil {
		t.Fatalf("GetTriageItemDetail returned error: %v", err)
	}
	if detail.Detail.TriageItemID != item.TriageItemID || !detail.Result.ReadOnly || !detail.Result.Succeeded {
		t.Fatalf("triage detail result = %#v", detail)
	}

	if got := mustAppItem(t, repo, item.TriageItemID); !reflect.DeepEqual(got, original) {
		t.Fatalf("read methods mutated source item:\ngot %#v\nwant %#v", got, original)
	}
}

func TestReviewOperationsBuildBatchAndExportsAreReadOnly(t *testing.T) {
	repo := operations.NewMemoryRepository()
	now := fixedAppNow()
	item := appItem("triage_batch", triage.SourceResearchGap, triage.PriorityCritical, triage.StatusOpen, now)
	if err := repo.SaveTriageItem(item); err != nil {
		t.Fatalf("SaveTriageItem returned error: %v", err)
	}
	if err := repo.SaveFollowUpAction(appAction("action_batch", item.TriageItemID, now)); err != nil {
		t.Fatalf("SaveFollowUpAction returned error: %v", err)
	}
	original := mustAppItem(t, repo, item.TriageItemID)
	service := mustAppService(t, repo)

	batchResult, err := service.BuildReviewBatch(BuildReviewBatchRequest{
		RequestID:       "request_batch",
		BatchID:         "review_batch_app",
		GeneratedAt:     now,
		AsOf:            now,
		SelectionReason: "ACTIVE_REVIEW",
	})
	if err != nil {
		t.Fatalf("BuildReviewBatch returned error: %v", err)
	}
	if batchResult.Batch.TotalItems != 1 || !batchResult.Result.ReadOnly || !batchResult.Result.Succeeded {
		t.Fatalf("batch result = %#v", batchResult)
	}

	jsonResult, err := service.ExportReviewBatchJSON(ExportReviewBatchRequest{
		RequestID:   "request_export_json",
		ExportID:    "export_batch_json",
		GeneratedAt: now,
		Batch:       batchResult.Batch,
	})
	if err != nil {
		t.Fatalf("ExportReviewBatchJSON returned error: %v", err)
	}
	if !jsonResult.Result.ReadOnly || !jsonResult.Result.Succeeded || jsonResult.Export.AutoApplyAllowed {
		t.Fatalf("json export result = %#v", jsonResult)
	}
	if !strings.Contains(jsonResult.Export.Content, `"read_only": true`) {
		t.Fatalf("json export missing read_only true:\n%s", jsonResult.Export.Content)
	}

	markdownResult, err := service.ExportReviewBatchMarkdown(ExportReviewBatchRequest{
		RequestID:   "request_export_markdown",
		ExportID:    "export_batch_markdown",
		GeneratedAt: now,
		Batch:       batchResult.Batch,
	})
	if err != nil {
		t.Fatalf("ExportReviewBatchMarkdown returned error: %v", err)
	}
	if !strings.Contains(markdownResult.Export.Content, "Auto apply allowed: false") {
		t.Fatalf("markdown export missing safety boundary:\n%s", markdownResult.Export.Content)
	}

	followUpJSON, err := service.ExportFollowUpActionsJSON(ExportFollowUpActionsRequest{
		RequestID:     "request_actions_json",
		ExportID:      "export_actions_json",
		GeneratedAt:   now,
		SourceBatchID: batchResult.Batch.BatchID,
		Actions:       batchResult.Batch.FollowUpActions,
	})
	if err != nil {
		t.Fatalf("ExportFollowUpActionsJSON returned error: %v", err)
	}
	if followUpJSON.Export.ItemCount != 1 || !followUpJSON.Result.ReadOnly || followUpJSON.Result.AutoApplyAllowed {
		t.Fatalf("follow-up json result = %#v", followUpJSON)
	}

	followUpMarkdown, err := service.ExportFollowUpActionsMarkdown(ExportFollowUpActionsRequest{
		RequestID:     "request_actions_markdown",
		ExportID:      "export_actions_markdown",
		GeneratedAt:   now,
		SourceBatchID: batchResult.Batch.BatchID,
		Actions:       batchResult.Batch.FollowUpActions,
	})
	if err != nil {
		t.Fatalf("ExportFollowUpActionsMarkdown returned error: %v", err)
	}
	if !strings.Contains(followUpMarkdown.Export.Content, "read-only manual action list") {
		t.Fatalf("follow-up markdown export missing manual action boundary:\n%s", followUpMarkdown.Export.Content)
	}

	if got := mustAppItem(t, repo, item.TriageItemID); !reflect.DeepEqual(got, original) {
		t.Fatalf("batch/export methods mutated source item:\ngot %#v\nwant %#v", got, original)
	}
}

func fixedAppNow() time.Time {
	return time.Date(2026, 6, 26, 9, 0, 0, 0, time.UTC)
}
