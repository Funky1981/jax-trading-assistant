package workflow

import (
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

func TestBuildReviewBatchFromOpenItems(t *testing.T) {
	now := fixedWorkflowNow()
	items := []triage.Item{
		workflowItem("triage_open", triage.PriorityMedium, triage.StatusOpen, now),
		workflowItem("triage_closed", triage.PriorityCritical, triage.StatusClosed, now),
		workflowItem("triage_rejected", triage.PriorityHigh, triage.StatusRejected, now),
		workflowItem("triage_deferred", triage.PriorityLow, triage.StatusDeferred, now),
		workflowItem("triage_more_evidence", triage.PriorityHigh, triage.StatusNeedsMoreEvidence, now),
	}

	got, err := BuildReviewBatch(items, nil, BatchOptions{
		BatchID:         "review_batch_active",
		GeneratedAt:     now,
		AsOf:            now,
		SelectionReason: SelectionActiveReview,
	})
	if err != nil {
		t.Fatalf("BuildReviewBatch returned error: %v", err)
	}

	wantIDs := []string{"triage_more_evidence", "triage_open", "triage_deferred"}
	if packetIDs(got.TriageItems) != wantIDs[0]+","+wantIDs[1]+","+wantIDs[2] {
		t.Fatalf("packet order = %v, want %v", packetIDs(got.TriageItems), wantIDs)
	}
	if got.TotalItems != 3 {
		t.Fatalf("total items = %d, want 3", got.TotalItems)
	}
	if !got.ReadOnly || !got.HumanReviewRequired {
		t.Fatalf("batch safety flags = read_only %v human_review_required %v", got.ReadOnly, got.HumanReviewRequired)
	}
	assertWorkflowContainsAll(t, got.ForbiddenActions, feedback.ForbiddenActions())
}

func TestPriorityAndDueOrdering(t *testing.T) {
	now := fixedWorkflowNow()
	items := []triage.Item{
		workflowItemWithDue("triage_low_future", triage.PriorityLow, now.Add(48*time.Hour), now),
		workflowItemWithDue("triage_high_future", triage.PriorityHigh, now.Add(24*time.Hour), now),
		workflowItemWithDue("triage_critical_future", triage.PriorityCritical, now.Add(24*time.Hour), now),
		workflowItemWithDue("triage_medium_due", triage.PriorityMedium, now, now),
		workflowItemWithDue("triage_high_overdue", triage.PriorityHigh, now.Add(-2*time.Hour), now),
		workflowItemWithDue("triage_high_due", triage.PriorityHigh, now, now),
	}

	got := SelectReviewItems(items, SelectionOptions{AsOf: now})

	want := []string{
		"triage_critical_future",
		"triage_high_overdue",
		"triage_high_due",
		"triage_high_future",
		"triage_medium_due",
		"triage_low_future",
	}
	if ids(got) != want[0]+","+want[1]+","+want[2]+","+want[3]+","+want[4]+","+want[5] {
		t.Fatalf("selected order = %s, want %v", ids(got), want)
	}
}

func TestReviewPacketPreservesSafetyFields(t *testing.T) {
	now := fixedWorkflowNow()
	item := workflowItem("triage_packet", triage.PriorityCritical, triage.StatusOpen, now)
	item.ForbiddenActions = append([]string{"manual_only"}, feedback.ForbiddenActions()...)

	got, err := BuildReviewPacket(item)
	if err != nil {
		t.Fatalf("BuildReviewPacket returned error: %v", err)
	}

	if got.PacketID != "packet_triage_packet" {
		t.Fatalf("packet id = %s", got.PacketID)
	}
	if !got.RequiresHumanApproval {
		t.Fatalf("requires human approval = false")
	}
	if got.AutoApplyAllowed {
		t.Fatalf("auto apply allowed = true")
	}
	if !reflect.DeepEqual(got.ForbiddenActions, item.ForbiddenActions) {
		t.Fatalf("forbidden actions = %#v, want %#v", got.ForbiddenActions, item.ForbiddenActions)
	}
}

func TestBuildReviewBatchIncludesFollowUpActionsReadOnly(t *testing.T) {
	now := fixedWorkflowNow()
	item := workflowItem("triage_follow_up", triage.PriorityHigh, triage.StatusAccepted, now)
	action := operations.FollowUpAction{
		ActionID:              "action_triage_follow_up",
		TriageItemID:          item.TriageItemID,
		ActionType:            operations.ActionCreateResearchTask,
		Description:           "Create a manual research task; do not change trading rules automatically.",
		RequiresHumanApproval: true,
		AutoApplyAllowed:      false,
		ForbiddenActions:      feedback.ForbiddenActions(),
		Status:                operations.ActionStatusOpen,
		CreatedAt:             now,
	}

	got, err := BuildReviewBatch([]triage.Item{item}, []operations.FollowUpAction{action}, BatchOptions{
		BatchID:               "review_batch_actions",
		GeneratedAt:           now,
		AsOf:                  now,
		SelectionReason:       SelectionActiveReview,
		IncludeClosedRejected: true,
	})
	if err != nil {
		t.Fatalf("BuildReviewBatch returned error: %v", err)
	}

	if len(got.FollowUpActions) != 1 {
		t.Fatalf("follow-up actions = %d, want 1", len(got.FollowUpActions))
	}
	if got.FollowUpActions[0].AutoApplyAllowed {
		t.Fatalf("follow-up action auto apply allowed = true")
	}
	if !got.ReadOnly {
		t.Fatalf("batch read only = false")
	}
}

func workflowItem(id string, priority triage.Priority, status triage.Status, now time.Time) triage.Item {
	item := workflowItemWithDue(id, priority, now, now)
	item.Status = status
	return item
}

func workflowItemWithDue(id string, priority triage.Priority, dueAt, now time.Time) triage.Item {
	return triage.Item{
		TriageItemID:           id,
		SourceType:             triage.SourceResearchGap,
		SourceID:               "source_" + id,
		SourceDecisionID:       "decision_" + id,
		EventID:                "event_" + id,
		Asset:                  "SHEL",
		SetupFamily:            "macro_breakout",
		EventType:              "macro",
		Priority:               priority,
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
		DueAt:                  dueAt,
	}
}

func fixedWorkflowNow() time.Time {
	return time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
}

func assertWorkflowContainsAll(t *testing.T, got []string, want []string) {
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

func ids(items []triage.Item) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ","
		}
		out += item.TriageItemID
	}
	return out
}

func packetIDs(packets []ReviewPacket) string {
	out := ""
	for i, packet := range packets {
		if i > 0 {
			out += ","
		}
		out += packet.TriageItemID
	}
	return out
}
