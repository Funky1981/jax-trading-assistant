package operations

import (
	"fmt"
	"time"

	"jax-trading-assistant/internal/decisioning/triage"
)

func GenerateReviewOperationsReport(repo Repository, options ReportOptions) ReviewOperationsReport {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	asOf := options.AsOf
	if asOf.IsZero() {
		asOf = generatedAt
	}
	reportID := options.ReportID
	if reportID == "" {
		reportID = "review_operations_report"
	}

	items := repo.ListTriageItems()
	report := ReviewOperationsReport{
		ReportID:         reportID,
		GeneratedAt:      generatedAt,
		TotalTriageItems: len(items),
		ForbiddenActions: forbiddenActions(),
		Warnings: []string{
			"Review operations are manual-only; human approval remains required.",
			"Auto-apply, live orders, live trading approval, and broker execution remain blocked.",
		},
	}

	actionIDs := map[string]bool{}
	for _, item := range items {
		countItem(&report, item, asOf)
		for _, action := range repo.ListFollowUpActionsForTriageItem(item.TriageItemID) {
			if actionIDs[action.ActionID] {
				continue
			}
			actionIDs[action.ActionID] = true
			report.FollowUpActionCount++
			if action.RequiresHumanApproval {
				report.ActionsRequiringHumanApproval++
			}
		}
	}
	for _, record := range repo.ListOperationAuditRecords() {
		if record.Action == AuditActionAutoApplyBlocked {
			report.AutoApplyBlockedCount++
		}
	}
	report.Summary = fmt.Sprintf(
		"%d review triage items persisted: %d open, %d accepted, %d rejected, %d need more evidence; %d manual follow-up actions.",
		report.TotalTriageItems,
		report.OpenCount,
		report.AcceptedCount,
		report.RejectedCount,
		report.NeedsMoreEvidenceCount,
		report.FollowUpActionCount,
	)
	return report
}

func countItem(report *ReviewOperationsReport, item triage.Item, asOf time.Time) {
	switch item.Status {
	case triage.StatusOpen:
		report.OpenCount++
	case triage.StatusAccepted:
		report.AcceptedCount++
	case triage.StatusRejected:
		report.RejectedCount++
	case triage.StatusDeferred:
		report.DeferredCount++
	case triage.StatusNeedsMoreEvidence:
		report.NeedsMoreEvidenceCount++
	case triage.StatusClosed:
		report.ClosedCount++
	}

	switch item.Priority {
	case triage.PriorityCritical:
		report.CriticalCount++
	case triage.PriorityHigh:
		report.HighCount++
	case triage.PriorityMedium:
		report.MediumCount++
	case triage.PriorityLow:
		report.LowCount++
	}

	if !item.DueAt.IsZero() && !item.DueAt.After(asOf) {
		report.DueCount++
		if item.DueAt.Before(asOf) {
			report.OverdueCount++
		}
	}

	switch item.SourceType {
	case triage.SourceResearchGap:
		report.ResearchGapCount++
	case triage.SourceMissedOpportunity:
		report.MissedOpportunityCount++
	case triage.SourceRiskVetoTooStrict:
		report.RiskVetoTooStrictCount++
	case triage.SourcePaperSetupFailed:
		report.PaperSetupFailedCount++
	}
}
