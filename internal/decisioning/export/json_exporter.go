package export

import (
	"encoding/json"
	"strings"

	"jax-trading-assistant/internal/decisioning/operations"
	"jax-trading-assistant/internal/decisioning/workflow"
)

func ExportReviewBatch(batch workflow.ReviewBatch, options ExportOptions) (ExportResult, error) {
	if err := validateOptions(options); err != nil {
		return ExportResult{}, err
	}
	payloadBatch := copyBatch(batch)
	payloadBatch.Warnings = scrubWarnings(payloadBatch.Warnings)
	result := baseResult(options, batch.BatchID, batch.TotalItems)
	result.Warnings = append([]string{}, payloadBatch.Warnings...)
	result.Warnings = append(result.Warnings, livePromotionWarnings(batch)...)

	switch options.Format {
	case FormatJSON:
		content, err := marshalJSON(reviewBatchPayload{
			ExportID:         result.ExportID,
			ExportType:       result.ExportType,
			GeneratedAt:      result.GeneratedAt,
			SourceBatchID:    result.SourceBatchID,
			ItemCount:        result.ItemCount,
			Format:           result.Format,
			ReviewBatch:      payloadBatch,
			Warnings:         result.Warnings,
			ForbiddenActions: result.ForbiddenActions,
			ReadOnly:         true,
			AutoApplyAllowed: false,
		})
		if err != nil {
			return ExportResult{}, err
		}
		result.Content = content
	case FormatMarkdown:
		result.Content = renderReviewBatchMarkdown(payloadBatch, result)
	}
	return result, nil
}

func ExportFollowUpActions(actions []operations.FollowUpAction, sourceBatchID string, options ExportOptions) (ExportResult, error) {
	if err := validateOptions(options); err != nil {
		return ExportResult{}, err
	}
	copied := make([]operations.FollowUpAction, 0, len(actions))
	for _, action := range actions {
		copied = append(copied, copyAction(action))
	}
	result := baseResult(options, sourceBatchID, len(copied))
	result.Warnings = []string{"Follow-up actions are exported as a read-only manual action list; no automatic application is allowed."}
	switch options.Format {
	case FormatJSON:
		content, err := marshalJSON(followUpActionsPayload{
			ExportID:         result.ExportID,
			ExportType:       result.ExportType,
			GeneratedAt:      result.GeneratedAt,
			SourceBatchID:    result.SourceBatchID,
			ItemCount:        result.ItemCount,
			Format:           result.Format,
			FollowUpActions:  copied,
			Warnings:         result.Warnings,
			ForbiddenActions: result.ForbiddenActions,
			ReadOnly:         true,
			AutoApplyAllowed: false,
		})
		if err != nil {
			return ExportResult{}, err
		}
		result.Content = content
	case FormatMarkdown:
		result.Content = renderFollowUpActionsMarkdown(copied, result)
	}
	return result, nil
}

func ExportOperationsReport(report operations.ReviewOperationsReport, sourceBatchID string, options ExportOptions) (ExportResult, error) {
	if err := validateOptions(options); err != nil {
		return ExportResult{}, err
	}
	result := baseResult(options, sourceBatchID, report.TotalTriageItems)
	result.Warnings = scrubWarnings(append([]string{}, report.Warnings...))
	switch options.Format {
	case FormatJSON:
		content, err := marshalJSON(operationsReportPayload{
			ExportID:         result.ExportID,
			ExportType:       result.ExportType,
			GeneratedAt:      result.GeneratedAt,
			SourceBatchID:    result.SourceBatchID,
			ItemCount:        result.ItemCount,
			Format:           result.Format,
			Report:           report,
			Warnings:         result.Warnings,
			ForbiddenActions: result.ForbiddenActions,
			ReadOnly:         true,
			AutoApplyAllowed: false,
		})
		if err != nil {
			return ExportResult{}, err
		}
		result.Content = content
	case FormatMarkdown:
		result.Content = renderOperationsReportMarkdown(report, result)
	}
	return result, nil
}

type reviewBatchPayload struct {
	ExportID         string               `json:"export_id"`
	ExportType       ExportType           `json:"export_type"`
	GeneratedAt      any                  `json:"generated_at"`
	SourceBatchID    string               `json:"source_batch_id"`
	ItemCount        int                  `json:"item_count"`
	Format           ExportFormat         `json:"format"`
	ReviewBatch      workflow.ReviewBatch `json:"review_batch"`
	Warnings         []string             `json:"warnings"`
	ForbiddenActions []string             `json:"forbidden_actions"`
	ReadOnly         bool                 `json:"read_only"`
	AutoApplyAllowed bool                 `json:"auto_apply_allowed"`
}

type followUpActionsPayload struct {
	ExportID         string                      `json:"export_id"`
	ExportType       ExportType                  `json:"export_type"`
	GeneratedAt      any                         `json:"generated_at"`
	SourceBatchID    string                      `json:"source_batch_id"`
	ItemCount        int                         `json:"item_count"`
	Format           ExportFormat                `json:"format"`
	FollowUpActions  []operations.FollowUpAction `json:"follow_up_actions"`
	Warnings         []string                    `json:"warnings"`
	ForbiddenActions []string                    `json:"forbidden_actions"`
	ReadOnly         bool                        `json:"read_only"`
	AutoApplyAllowed bool                        `json:"auto_apply_allowed"`
}

type operationsReportPayload struct {
	ExportID         string                            `json:"export_id"`
	ExportType       ExportType                        `json:"export_type"`
	GeneratedAt      any                               `json:"generated_at"`
	SourceBatchID    string                            `json:"source_batch_id"`
	ItemCount        int                               `json:"item_count"`
	Format           ExportFormat                      `json:"format"`
	Report           operations.ReviewOperationsReport `json:"operations_report"`
	Warnings         []string                          `json:"warnings"`
	ForbiddenActions []string                          `json:"forbidden_actions"`
	ReadOnly         bool                              `json:"read_only"`
	AutoApplyAllowed bool                              `json:"auto_apply_allowed"`
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func copyAction(action operations.FollowUpAction) operations.FollowUpAction {
	copied := action
	copied.ForbiddenActions = append([]string{}, action.ForbiddenActions...)
	copied.RequiresHumanApproval = true
	copied.AutoApplyAllowed = false
	return copied
}

func copyBatch(batch workflow.ReviewBatch) workflow.ReviewBatch {
	copied := batch
	copied.TriageItems = append([]workflow.ReviewPacket{}, batch.TriageItems...)
	copied.FollowUpActions = make([]operations.FollowUpAction, 0, len(batch.FollowUpActions))
	for _, action := range batch.FollowUpActions {
		copied.FollowUpActions = append(copied.FollowUpActions, copyAction(action))
	}
	copied.ForbiddenActions = append([]string{}, batch.ForbiddenActions...)
	copied.Warnings = append([]string{}, batch.Warnings...)
	for i := range copied.TriageItems {
		copied.TriageItems[i].EvidenceRefs = append([]string{}, batch.TriageItems[i].EvidenceRefs...)
		copied.TriageItems[i].AllowedFollowUpActions = append([]string{}, batch.TriageItems[i].AllowedFollowUpActions...)
		copied.TriageItems[i].ForbiddenActions = append([]string{}, batch.TriageItems[i].ForbiddenActions...)
	}
	return copied
}

func scrubWarnings(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		normalized := strings.ToLower(warning)
		if strings.Contains(normalized, "chain-of-thought") ||
			strings.Contains(normalized, "hidden reasoning") ||
			strings.Contains(normalized, "reasoning dump") {
			out = append(out, "Reasoning internals are excluded from review operations exports.")
			continue
		}
		out = append(out, warning)
	}
	return out
}

func livePromotionWarnings(batch workflow.ReviewBatch) []string {
	var warnings []string
	for _, packet := range batch.TriageItems {
		if containsLivePromotion(packet.SuggestedAction) || containsLivePromotion(packet.Reason) {
			warnings = append(warnings, "Live-readiness or live trading request blocked for triage item "+packet.TriageItemID+".")
		}
	}
	return warnings
}

func containsLivePromotion(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "live_ready") ||
		strings.Contains(normalized, "live ready") ||
		strings.Contains(normalized, "live trading") ||
		strings.Contains(normalized, "live order")
}
