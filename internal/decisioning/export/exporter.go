package export

import (
	"fmt"
	"time"

	"jax-trading-assistant/internal/decisioning/feedback"
)

type ExportFormat string

const (
	FormatJSON     ExportFormat = "JSON"
	FormatMarkdown ExportFormat = "MARKDOWN"
)

type ExportType string

const (
	ExportTypeReviewBatch      ExportType = "REVIEW_BATCH"
	ExportTypeTriageQueue      ExportType = "TRIAGE_QUEUE"
	ExportTypeFollowUpActions  ExportType = "FOLLOW_UP_ACTIONS"
	ExportTypeOperationsReport ExportType = "OPERATIONS_REPORT"
)

type ExportOptions struct {
	ExportID    string
	ExportType  ExportType
	Format      ExportFormat
	GeneratedAt time.Time
}

type ExportResult struct {
	ExportID         string       `json:"export_id"`
	ExportType       ExportType   `json:"export_type"`
	GeneratedAt      time.Time    `json:"generated_at"`
	SourceBatchID    string       `json:"source_batch_id"`
	ItemCount        int          `json:"item_count"`
	Format           ExportFormat `json:"format"`
	Content          string       `json:"content"`
	Warnings         []string     `json:"warnings"`
	ForbiddenActions []string     `json:"forbidden_actions"`
	ReadOnly         bool         `json:"read_only"`
	AutoApplyAllowed bool         `json:"auto_apply_allowed"`
}

func baseResult(options ExportOptions, sourceBatchID string, itemCount int) ExportResult {
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	exportID := options.ExportID
	if exportID == "" {
		exportID = "review_operations_export"
	}
	return ExportResult{
		ExportID:         exportID,
		ExportType:       options.ExportType,
		GeneratedAt:      generatedAt,
		SourceBatchID:    sourceBatchID,
		ItemCount:        itemCount,
		Format:           options.Format,
		ForbiddenActions: feedback.ForbiddenActions(),
		ReadOnly:         true,
		AutoApplyAllowed: false,
	}
}

func validateOptions(options ExportOptions) error {
	switch options.Format {
	case FormatJSON, FormatMarkdown:
	default:
		return fmt.Errorf("export format %q is not supported", options.Format)
	}
	switch options.ExportType {
	case ExportTypeReviewBatch, ExportTypeTriageQueue, ExportTypeFollowUpActions, ExportTypeOperationsReport:
	default:
		return fmt.Errorf("export type %q is not supported", options.ExportType)
	}
	return nil
}
