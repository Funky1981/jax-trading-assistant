package golden

import (
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/readmodel"
	"jax-trading-assistant/internal/decisioning/triage"
)

type readModelGoldenCase struct {
	Name                           string                      `json:"name"`
	GeneratedAt                    time.Time                   `json:"generated_at"`
	AsOf                           time.Time                   `json:"as_of"`
	Items                          []triage.Item               `json:"items"`
	FollowUpActions                []operations.FollowUpAction `json:"follow_up_actions"`
	ExpectedTotalOpen              int                         `json:"expected_total_open"`
	ExpectedTotalCritical          int                         `json:"expected_total_critical"`
	ExpectedTotalHigh              int                         `json:"expected_total_high"`
	ExpectedTotalDue               int                         `json:"expected_total_due"`
	ExpectedTotalOverdue           int                         `json:"expected_total_overdue"`
	ExpectedTotalNeedsMoreEvidence int                         `json:"expected_total_needs_more_evidence"`
	ExpectedTotalDeferred          int                         `json:"expected_total_deferred"`
	ExpectedTotalFollowUpActions   int                         `json:"expected_total_follow_up_actions"`
}

func TestReadModelGoldenCases(t *testing.T) {
	var tc readModelGoldenCase
	readJSON(t, filepath.Join("readmodel", "review_queue_summary.json"), &tc)
	repo := operations.NewMemoryRepository()
	for _, item := range tc.Items {
		if err := repo.SaveTriageItem(item); err != nil {
			t.Fatalf("%s SaveTriageItem(%s) returned error: %v", tc.Name, item.TriageItemID, err)
		}
	}
	for _, action := range tc.FollowUpActions {
		if err := repo.SaveFollowUpAction(action); err != nil {
			t.Fatalf("%s SaveFollowUpAction(%s) returned error: %v", tc.Name, action.ActionID, err)
		}
	}

	got := readmodel.BuildReviewQueueSummary(repo, readmodel.Options{GeneratedAt: tc.GeneratedAt, AsOf: tc.AsOf})
	if got.TotalOpen != tc.ExpectedTotalOpen ||
		got.TotalCritical != tc.ExpectedTotalCritical ||
		got.TotalHigh != tc.ExpectedTotalHigh ||
		got.TotalDue != tc.ExpectedTotalDue ||
		got.TotalOverdue != tc.ExpectedTotalOverdue ||
		got.TotalNeedsMoreEvidence != tc.ExpectedTotalNeedsMoreEvidence ||
		got.TotalDeferred != tc.ExpectedTotalDeferred ||
		got.TotalFollowUpActions != tc.ExpectedTotalFollowUpActions {
		t.Fatalf("%s summary = %#v", tc.Name, got)
	}
	if !got.ReadOnly {
		t.Fatalf("%s read_only = false", tc.Name)
	}
	assertContainsAll(t, got.ForbiddenActions, feedback.ForbiddenActions())
}
