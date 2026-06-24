package export

import (
	"fmt"
	"strings"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/workflow"
)

func renderReviewBatchMarkdown(batch workflow.ReviewBatch, result ExportResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Review Batch %s\n\n", batch.BatchID)
	fmt.Fprintf(&b, "- Export ID: %s\n", result.ExportID)
	fmt.Fprintf(&b, "- Generated at: %s\n", result.GeneratedAt.Format(timeFormat))
	fmt.Fprintf(&b, "- Selection reason: %s\n", batch.SelectionReason)
	fmt.Fprintf(&b, "- Total items: %d\n", batch.TotalItems)
	fmt.Fprintf(&b, "- Critical count: %d\n", batch.CriticalCount)
	fmt.Fprintf(&b, "- High count: %d\n", batch.HighCount)
	fmt.Fprintf(&b, "- Overdue count: %d\n", batch.OverdueCount)
	fmt.Fprintf(&b, "- Due count: %d\n", batch.DueCount)
	fmt.Fprintf(&b, "- Read only: %t\n", result.ReadOnly)
	fmt.Fprintf(&b, "- Human review required: %t\n", batch.HumanReviewRequired)
	fmt.Fprintf(&b, "- Auto apply allowed: %t\n", result.AutoApplyAllowed)
	fmt.Fprintf(&b, "- Forbidden actions: %s\n\n", strings.Join(result.ForbiddenActions, ", "))

	b.WriteString("## Review Packets\n\n")
	for _, packet := range batch.TriageItems {
		fmt.Fprintf(&b, "### %s\n\n", packet.PacketID)
		fmt.Fprintf(&b, "- Triage item: %s\n", packet.TriageItemID)
		fmt.Fprintf(&b, "- Priority: %s\n", packet.Priority)
		fmt.Fprintf(&b, "- Status: %s\n", packet.Status)
		fmt.Fprintf(&b, "- Event: %s\n", packet.EventID)
		fmt.Fprintf(&b, "- Asset: %s\n", packet.Asset)
		fmt.Fprintf(&b, "- Setup family: %s\n", packet.SetupFamily)
		fmt.Fprintf(&b, "- Reason: %s\n", packet.Reason)
		fmt.Fprintf(&b, "- Suggested action: %s\n", packet.SuggestedAction)
		fmt.Fprintf(&b, "- Due at: %s\n", packet.DueAt.Format(timeFormat))
		fmt.Fprintf(&b, "- Requires human approval: %t\n", packet.RequiresHumanApproval)
		fmt.Fprintf(&b, "- Auto apply allowed: %t\n", packet.AutoApplyAllowed)
		fmt.Fprintf(&b, "- Forbidden actions: %s\n\n", strings.Join(packet.ForbiddenActions, ", "))
	}
	writeWarnings(&b, result.Warnings)
	return b.String()
}

func renderFollowUpActionsMarkdown(actions []operations.FollowUpAction, result ExportResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Follow-Up Actions %s\n\n", result.SourceBatchID)
	fmt.Fprintf(&b, "- Export ID: %s\n", result.ExportID)
	fmt.Fprintf(&b, "- Generated at: %s\n", result.GeneratedAt.Format(timeFormat))
	fmt.Fprintf(&b, "- Item count: %d\n", result.ItemCount)
	fmt.Fprintf(&b, "- Read only: %t\n", result.ReadOnly)
	fmt.Fprintf(&b, "- Auto apply allowed: %t\n", result.AutoApplyAllowed)
	fmt.Fprintf(&b, "- Forbidden actions: %s\n\n", strings.Join(result.ForbiddenActions, ", "))
	for _, action := range actions {
		fmt.Fprintf(&b, "## %s\n\n", action.ActionID)
		fmt.Fprintf(&b, "- Triage item: %s\n", action.TriageItemID)
		fmt.Fprintf(&b, "- Action type: %s\n", action.ActionType)
		fmt.Fprintf(&b, "- Description: %s\n", action.Description)
		fmt.Fprintf(&b, "- Requires human approval: %t\n", action.RequiresHumanApproval)
		fmt.Fprintf(&b, "- Auto apply allowed: %t\n", action.AutoApplyAllowed)
		fmt.Fprintf(&b, "- Status: %s\n\n", action.Status)
	}
	writeWarnings(&b, result.Warnings)
	return b.String()
}

func renderOperationsReportMarkdown(report operations.ReviewOperationsReport, result ExportResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Review Operations Report %s\n\n", report.ReportID)
	fmt.Fprintf(&b, "- Export ID: %s\n", result.ExportID)
	fmt.Fprintf(&b, "- Generated at: %s\n", result.GeneratedAt.Format(timeFormat))
	fmt.Fprintf(&b, "- Total triage items: %d\n", report.TotalTriageItems)
	fmt.Fprintf(&b, "- Open count: %d\n", report.OpenCount)
	fmt.Fprintf(&b, "- Critical count: %d\n", report.CriticalCount)
	fmt.Fprintf(&b, "- High count: %d\n", report.HighCount)
	fmt.Fprintf(&b, "- Follow-up action count: %d\n", report.FollowUpActionCount)
	fmt.Fprintf(&b, "- Read only: %t\n", result.ReadOnly)
	fmt.Fprintf(&b, "- Auto apply allowed: %t\n", result.AutoApplyAllowed)
	fmt.Fprintf(&b, "- Forbidden actions: %s\n\n", strings.Join(result.ForbiddenActions, ", "))
	fmt.Fprintf(&b, "Summary: %s\n\n", report.Summary)
	writeWarnings(&b, result.Warnings)
	return b.String()
}

func writeWarnings(b *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	b.WriteString("## Warnings\n\n")
	for _, warning := range warnings {
		fmt.Fprintf(b, "- %s\n", warning)
	}
}

const timeFormat = "2006-01-02T15:04:05Z"
