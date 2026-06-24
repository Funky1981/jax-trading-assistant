package golden

import (
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
	"jax-trading-assistant/internal/decisioning/workflow"
)

type workflowGoldenCase struct {
	Name                     string                      `json:"name"`
	BatchID                  string                      `json:"batch_id"`
	GeneratedAt              time.Time                   `json:"generated_at"`
	AsOf                     time.Time                   `json:"as_of"`
	SelectionReason          workflow.SelectionReason    `json:"selection_reason"`
	Items                    []triage.Item               `json:"items"`
	Actions                  []operations.FollowUpAction `json:"actions"`
	ExpectedPacketIDs        []string                    `json:"expected_packet_ids"`
	ExpectedTotalItems       int                         `json:"expected_total_items"`
	ExpectedCriticalCount    int                         `json:"expected_critical_count"`
	ExpectedHighCount        int                         `json:"expected_high_count"`
	ExpectedOverdueCount     int                         `json:"expected_overdue_count"`
	ExpectedDueCount         int                         `json:"expected_due_count"`
	ExpectedForbiddenActions []string                    `json:"expected_forbidden_actions"`
}

func TestWorkflowGoldenCases(t *testing.T) {
	files := []string{"review_batch_active.json"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc workflowGoldenCase
			readJSON(t, filepath.Join("workflow", file), &tc)

			got, err := workflow.BuildReviewBatch(tc.Items, tc.Actions, workflow.BatchOptions{
				BatchID:         tc.BatchID,
				GeneratedAt:     tc.GeneratedAt,
				AsOf:            tc.AsOf,
				SelectionReason: tc.SelectionReason,
			})
			if err != nil {
				t.Fatalf("%s BuildReviewBatch returned error: %v", tc.Name, err)
			}

			if got.TotalItems != tc.ExpectedTotalItems ||
				got.CriticalCount != tc.ExpectedCriticalCount ||
				got.HighCount != tc.ExpectedHighCount ||
				got.OverdueCount != tc.ExpectedOverdueCount ||
				got.DueCount != tc.ExpectedDueCount {
				t.Fatalf("%s counts = %#v", tc.Name, got)
			}
			if got.ReadOnly != true || got.HumanReviewRequired != true {
				t.Fatalf("%s safety flags = read_only %v human_review_required %v", tc.Name, got.ReadOnly, got.HumanReviewRequired)
			}
			if gotPacketIDs(got.TriageItems) != joinGoldenIDs(tc.ExpectedPacketIDs) {
				t.Fatalf("%s packet ids = %s, want %s", tc.Name, gotPacketIDs(got.TriageItems), joinGoldenIDs(tc.ExpectedPacketIDs))
			}
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
		})
	}
}

func gotPacketIDs(packets []workflow.ReviewPacket) string {
	out := make([]string, 0, len(packets))
	for _, packet := range packets {
		out = append(out, packet.TriageItemID)
	}
	return joinGoldenIDs(out)
}

func joinGoldenIDs(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}
