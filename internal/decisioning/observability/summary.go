package observability

import (
	"jax-trading-assistant/internal/decisioning/persistence"
	"jax-trading-assistant/internal/decisioning/pipeline"
)

type Summary struct {
	PipelineID            string   `json:"pipeline_id"`
	EventID               string   `json:"event_id"`
	FinalStatus           string   `json:"final_status"`
	FinalDecision         string   `json:"final_decision"`
	SourceModules         []string `json:"source_modules"`
	NoTradeReason         string   `json:"no_trade_reason,omitempty"`
	RejectionReason       string   `json:"rejection_reason,omitempty"`
	PaperReviewReady      bool     `json:"paper_review_ready"`
	HumanApprovalRequired bool     `json:"human_approval_required"`
	PaperOnly             bool     `json:"paper_only"`
	LiveTradingBlocked    bool     `json:"live_trading_blocked"`
	ReviewScheduled       bool     `json:"review_scheduled"`
	WarningCount          int      `json:"warning_count"`
	ErrorCount            int      `json:"error_count"`
}

func NewSummary(record persistence.PipelineResultRecord) Summary {
	return Summary{
		PipelineID:            record.PipelineID,
		EventID:               record.EventID,
		FinalStatus:           record.FinalStatus,
		FinalDecision:         record.FinalDecision,
		SourceModules:         append([]string(nil), record.SourceModules...),
		NoTradeReason:         record.NoTradeReason,
		RejectionReason:       record.RejectionReason,
		PaperReviewReady:      record.FinalStatus == string(pipeline.StatusTradeCandidateReadyForPaperReview) && record.PaperTicketSummary != nil,
		HumanApprovalRequired: record.HumanApprovalRequired,
		PaperOnly:             record.PaperOnly,
		LiveTradingBlocked:    record.LiveTradingBlocked,
		ReviewScheduled:       record.ReviewSchedule.ScheduleID != "",
		WarningCount:          len(record.ValidationWarnings),
		ErrorCount:            len(record.ValidationErrors),
	}
}
