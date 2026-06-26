package readmodel

import (
	"time"

	reviewexport "jax-trading-assistant/internal/decisioning/export"
)

type ExportSummary struct {
	ExportID         string                    `json:"export_id"`
	ExportType       reviewexport.ExportType   `json:"export_type"`
	GeneratedAt      time.Time                 `json:"generated_at"`
	SourceBatchID    string                    `json:"source_batch_id"`
	ItemCount        int                       `json:"item_count"`
	Format           reviewexport.ExportFormat `json:"format"`
	ContentBytes     int                       `json:"content_bytes"`
	Warnings         []string                  `json:"warnings"`
	ForbiddenActions []string                  `json:"forbidden_actions"`
	ReadOnly         bool                      `json:"read_only"`
	AutoApplyAllowed bool                      `json:"auto_apply_allowed"`
}

func BuildExportSummary(result reviewexport.ExportResult) ExportSummary {
	return ExportSummary{
		ExportID:         result.ExportID,
		ExportType:       result.ExportType,
		GeneratedAt:      result.GeneratedAt,
		SourceBatchID:    result.SourceBatchID,
		ItemCount:        result.ItemCount,
		Format:           result.Format,
		ContentBytes:     len(result.Content),
		Warnings:         append([]string{}, result.Warnings...),
		ForbiddenActions: append([]string{}, result.ForbiddenActions...),
		ReadOnly:         true,
		AutoApplyAllowed: false,
	}
}
