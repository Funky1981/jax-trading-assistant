package export

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
	"jax-trading-assistant/internal/decisioning/workflow"
)

func TestJSONExportIsDeterministicAndReadOnly(t *testing.T) {
	batch := exportBatchFixture(t)

	first, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_review_batch_json",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatJSON,
		GeneratedAt: fixedExportNow(),
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}
	second, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_review_batch_json",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatJSON,
		GeneratedAt: fixedExportNow(),
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch second call returned error: %v", err)
	}

	if first.Content != second.Content {
		t.Fatalf("JSON export was not deterministic:\nfirst=%s\nsecond=%s", first.Content, second.Content)
	}
	if !first.ReadOnly || first.AutoApplyAllowed {
		t.Fatalf("export safety flags = read_only %v auto_apply_allowed %v", first.ReadOnly, first.AutoApplyAllowed)
	}
	if first.SourceBatchID != batch.BatchID || first.ItemCount != batch.TotalItems {
		t.Fatalf("export source/count = %s/%d, want %s/%d", first.SourceBatchID, first.ItemCount, batch.BatchID, batch.TotalItems)
	}
	assertExportContainsAll(t, first.ForbiddenActions, feedback.ForbiddenActions())

	var decoded map[string]any
	if err := json.Unmarshal([]byte(first.Content), &decoded); err != nil {
		t.Fatalf("export content is not valid JSON: %v", err)
	}
	if decoded["read_only"] != true || decoded["auto_apply_allowed"] != false {
		t.Fatalf("decoded safety fields = %#v", decoded)
	}
}

func TestMarkdownExportIsDeterministicAndIncludesSafetyBoundaries(t *testing.T) {
	batch := exportBatchFixture(t)

	got, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_review_batch_markdown",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatMarkdown,
		GeneratedAt: fixedExportNow(),
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}

	for _, want := range []string{
		"# Review Batch review_batch_export",
		"- Priority: CRITICAL",
		"- Reason: manual review required",
		"- Suggested action: manual follow-up only",
		"- Due at: 2026-06-24T12:00:00Z",
		"- Read only: true",
		"- Auto apply allowed: false",
		"- Forbidden actions: execute_trade, create_live_order, auto_approve",
	} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("markdown export missing %q:\n%s", want, got.Content)
		}
	}
	assertNoHiddenReasoning(t, got.Content)
}

func TestExportDoesNotMutateSourceRecords(t *testing.T) {
	batch := exportBatchFixture(t)
	originalBatch := batch
	originalActions := append([]operations.FollowUpAction{}, batch.FollowUpActions...)

	if _, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_no_mutation",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatJSON,
		GeneratedAt: fixedExportNow(),
	}); err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}

	if !reflect.DeepEqual(batch, originalBatch) {
		t.Fatalf("batch mutated after export:\ngot %#v\nwant %#v", batch, originalBatch)
	}
	if !reflect.DeepEqual(batch.FollowUpActions, originalActions) {
		t.Fatalf("follow-up actions mutated after export")
	}
}

func TestExportExcludesHiddenReasoningMarkers(t *testing.T) {
	batch := exportBatchFixture(t)
	batch.Warnings = append(batch.Warnings, "manual review only; no hidden reasoning is exported")

	got, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_no_hidden_reasoning",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatJSON,
		GeneratedAt: fixedExportNow(),
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}

	assertNoHiddenReasoning(t, got.Content)
}

func TestLivePromotionBlockedInExportWarnings(t *testing.T) {
	batch := exportBatchFixture(t)
	batch.TriageItems[0].SuggestedAction = "promote setup to LIVE_READY"

	got, err := ExportReviewBatch(batch, ExportOptions{
		ExportID:    "export_live_blocked",
		ExportType:  ExportTypeReviewBatch,
		Format:      FormatJSON,
		GeneratedAt: fixedExportNow(),
	})
	if err != nil {
		t.Fatalf("ExportReviewBatch returned error: %v", err)
	}

	if len(got.Warnings) == 0 {
		t.Fatalf("warnings missing for live promotion")
	}
	if !strings.Contains(strings.ToLower(strings.Join(got.Warnings, " ")), "live") {
		t.Fatalf("warnings do not mention live block: %v", got.Warnings)
	}
	if got.AutoApplyAllowed {
		t.Fatalf("auto apply allowed = true")
	}
	assertExportContainsAll(t, got.ForbiddenActions, feedback.ForbiddenActions())
}

func TestFollowUpActionsExportIsReadOnlyActionList(t *testing.T) {
	now := fixedExportNow()
	actions := []operations.FollowUpAction{
		{
			ActionID:              "action_research",
			TriageItemID:          "triage_research",
			ActionType:            operations.ActionCreateResearchTask,
			Description:           "Create a manual research task; do not change trading rules automatically.",
			RequiresHumanApproval: true,
			AutoApplyAllowed:      false,
			ForbiddenActions:      feedback.ForbiddenActions(),
			Status:                operations.ActionStatusOpen,
			CreatedAt:             now,
		},
	}

	got, err := ExportFollowUpActions(actions, "review_batch_export", ExportOptions{
		ExportID:    "export_follow_up_actions",
		ExportType:  ExportTypeFollowUpActions,
		Format:      FormatMarkdown,
		GeneratedAt: now,
	})
	if err != nil {
		t.Fatalf("ExportFollowUpActions returned error: %v", err)
	}

	if got.ItemCount != 1 || !got.ReadOnly || got.AutoApplyAllowed {
		t.Fatalf("follow-up export flags/count = %#v", got)
	}
	if !strings.Contains(got.Content, "Create a manual research task") {
		t.Fatalf("follow-up action description missing:\n%s", got.Content)
	}
}

func exportBatchFixture(t *testing.T) workflow.ReviewBatch {
	t.Helper()
	now := fixedExportNow()
	item := triage.Item{
		TriageItemID:           "triage_export",
		SourceType:             triage.SourceResearchGap,
		SourceID:               "source_triage_export",
		SourceDecisionID:       "decision_triage_export",
		EventID:                "event_triage_export",
		Asset:                  "SHEL",
		SetupFamily:            "macro_breakout",
		EventType:              "macro",
		Priority:               triage.PriorityCritical,
		Status:                 triage.StatusOpen,
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
	action := operations.FollowUpAction{
		ActionID:              "action_triage_export",
		TriageItemID:          item.TriageItemID,
		ActionType:            operations.ActionCreateResearchTask,
		Description:           "Create a manual research task; do not change trading rules automatically.",
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		Status:                operations.ActionStatusOpen,
		CreatedAt:             now,
	}
	batch, err := workflow.BuildReviewBatch([]triage.Item{item}, []operations.FollowUpAction{action}, workflow.BatchOptions{
		BatchID:         "review_batch_export",
		GeneratedAt:     now,
		AsOf:            now,
		SelectionReason: workflow.SelectionActiveReview,
	})
	if err != nil {
		t.Fatalf("BuildReviewBatch returned error: %v", err)
	}
	return batch
}

func fixedExportNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertNoHiddenReasoning(t *testing.T, content string) {
	t.Helper()
	normalized := strings.ToLower(content)
	for _, forbidden := range []string{"chain-of-thought", "hidden reasoning", "reasoning dump"} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("export leaked hidden reasoning marker %q in:\n%s", forbidden, content)
		}
	}
}

func assertExportContainsAll(t *testing.T, got []string, want []string) {
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
