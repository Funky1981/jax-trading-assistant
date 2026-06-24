package golden

import (
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/triage"
)

type operationsReportingGoldenCase struct {
	Name                     string                      `json:"name"`
	ReportID                 string                      `json:"report_id"`
	GeneratedAt              time.Time                   `json:"generated_at"`
	AsOf                     time.Time                   `json:"as_of"`
	Items                    []triage.Item               `json:"items"`
	Actions                  []operations.FollowUpAction `json:"actions"`
	Expected                 expectedOperationsReport    `json:"expected"`
	ExpectedForbiddenActions []string                    `json:"expected_forbidden_actions"`
}

type expectedOperationsReport struct {
	TotalTriageItems              int `json:"total_triage_items"`
	OpenCount                     int `json:"open_count"`
	AcceptedCount                 int `json:"accepted_count"`
	RejectedCount                 int `json:"rejected_count"`
	DeferredCount                 int `json:"deferred_count"`
	NeedsMoreEvidenceCount        int `json:"needs_more_evidence_count"`
	ClosedCount                   int `json:"closed_count"`
	CriticalCount                 int `json:"critical_count"`
	HighCount                     int `json:"high_count"`
	MediumCount                   int `json:"medium_count"`
	LowCount                      int `json:"low_count"`
	OverdueCount                  int `json:"overdue_count"`
	DueCount                      int `json:"due_count"`
	ResearchGapCount              int `json:"research_gap_count"`
	MissedOpportunityCount        int `json:"missed_opportunity_count"`
	RiskVetoTooStrictCount        int `json:"risk_veto_too_strict_count"`
	PaperSetupFailedCount         int `json:"paper_setup_failed_count"`
	FollowUpActionCount           int `json:"follow_up_action_count"`
	ActionsRequiringHumanApproval int `json:"actions_requiring_human_approval"`
	AutoApplyBlockedCount         int `json:"auto_apply_blocked_count"`
}

func TestOperationsReportingGoldenCases(t *testing.T) {
	files := []string{"review_operations_report.json"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			var tc operationsReportingGoldenCase
			readJSON(t, filepath.Join("operations_reporting", file), &tc)

			repo := operations.NewMemoryRepository()
			for _, item := range tc.Items {
				if err := repo.SaveTriageItem(item); err != nil {
					t.Fatalf("%s SaveTriageItem(%s): %v", tc.Name, item.TriageItemID, err)
				}
			}
			for _, action := range tc.Actions {
				if err := repo.SaveFollowUpAction(action); err != nil {
					t.Fatalf("%s SaveFollowUpAction(%s): %v", tc.Name, action.ActionID, err)
				}
			}

			got := operations.GenerateReviewOperationsReport(repo, operations.ReportOptions{
				ReportID:    tc.ReportID,
				GeneratedAt: tc.GeneratedAt,
				AsOf:        tc.AsOf,
			})
			assertOperationsReport(t, got, tc.Expected)
			assertContainsAll(t, got.ForbiddenActions, tc.ExpectedForbiddenActions)
			if got.Summary == "" {
				t.Fatalf("%s summary missing", tc.Name)
			}
			if len(got.Warnings) == 0 {
				t.Fatalf("%s warnings missing", tc.Name)
			}
		})
	}
}

func assertOperationsReport(t *testing.T, got operations.ReviewOperationsReport, want expectedOperationsReport) {
	t.Helper()
	if got.TotalTriageItems != want.TotalTriageItems ||
		got.OpenCount != want.OpenCount ||
		got.AcceptedCount != want.AcceptedCount ||
		got.RejectedCount != want.RejectedCount ||
		got.DeferredCount != want.DeferredCount ||
		got.NeedsMoreEvidenceCount != want.NeedsMoreEvidenceCount ||
		got.ClosedCount != want.ClosedCount ||
		got.CriticalCount != want.CriticalCount ||
		got.HighCount != want.HighCount ||
		got.MediumCount != want.MediumCount ||
		got.LowCount != want.LowCount ||
		got.OverdueCount != want.OverdueCount ||
		got.DueCount != want.DueCount ||
		got.ResearchGapCount != want.ResearchGapCount ||
		got.MissedOpportunityCount != want.MissedOpportunityCount ||
		got.RiskVetoTooStrictCount != want.RiskVetoTooStrictCount ||
		got.PaperSetupFailedCount != want.PaperSetupFailedCount ||
		got.FollowUpActionCount != want.FollowUpActionCount ||
		got.ActionsRequiringHumanApproval != want.ActionsRequiringHumanApproval ||
		got.AutoApplyBlockedCount != want.AutoApplyBlockedCount {
		t.Fatalf("report counts = %#v, want %#v", got, want)
	}
}
