package persistence

import (
	"time"

	"jax-trading-assistant/internal/decisioning/paper"
	"jax-trading-assistant/internal/decisioning/pipeline"
	"jax-trading-assistant/internal/decisioning/review"
)

type PipelineResultRecord struct {
	TraceID                 string                  `json:"trace_id"`
	PipelineID              string                  `json:"pipeline_id"`
	EventID                 string                  `json:"event_id"`
	DecisionID              string                  `json:"decision_id"`
	FinalDecision           string                  `json:"final_decision"`
	FinalStatus             string                  `json:"final_status"`
	SourceModules           []string                `json:"source_modules"`
	NoTradeReason           string                  `json:"no_trade_reason,omitempty"`
	RejectionReason         string                  `json:"rejection_reason,omitempty"`
	AllowedActions          []string                `json:"allowed_actions"`
	ForbiddenActions        []string                `json:"forbidden_actions"`
	RiskAssessmentSummary   RiskAssessmentSummary   `json:"risk_assessment_summary"`
	ResearchEvidenceSummary ResearchEvidenceSummary `json:"research_evidence_summary"`
	PaperTicketSummary      *PaperTicketSummary     `json:"paper_ticket_summary,omitempty"`
	ReviewSchedule          review.ReviewSchedule   `json:"review_schedule"`
	HumanApprovalRequired   bool                    `json:"human_approval_required"`
	PaperOnly               bool                    `json:"paper_only"`
	LiveTradingBlocked      bool                    `json:"live_trading_blocked"`
	ValidationWarnings      []string                `json:"validation_warnings"`
	ValidationErrors        []string                `json:"validation_errors"`
	CreatedAt               time.Time               `json:"created_at"`
}

type RiskAssessmentSummary struct {
	RiskDecision       string   `json:"risk_decision"`
	FinalDecision      string   `json:"final_decision"`
	VetoReasons        []string `json:"veto_reasons"`
	DowngradeReasons   []string `json:"downgrade_reasons"`
	RequiredActions    []string `json:"required_actions"`
	LiveTradingBlocked bool     `json:"live_trading_blocked"`
}

type ResearchEvidenceSummary struct {
	PromotionDecision string   `json:"promotion_decision"`
	Summary           string   `json:"summary"`
	Warnings          []string `json:"warnings"`
}

type PaperTicketSummary struct {
	PaperTicketID       string `json:"paper_ticket_id"`
	HumanApprovalStatus string `json:"human_approval_status"`
	PaperOnly           bool   `json:"paper_only"`
	LiveTradingBlocked  bool   `json:"live_trading_blocked"`
}

func FromPipelineResult(result pipeline.Result, traceID string) PipelineResultRecord {
	decision := result.DecisionCoreResult.FinalDecision
	record := PipelineResultRecord{
		TraceID:                 traceID,
		PipelineID:              result.PipelineID,
		EventID:                 result.EventID,
		DecisionID:              decision.DecisionID,
		FinalDecision:           string(result.FinalDecision),
		FinalStatus:             string(result.FinalStatus),
		SourceModules:           defaultPipelineModules(result),
		NoTradeReason:           noTradeReason(result),
		RejectionReason:         rejectionReason(result),
		AllowedActions:          append([]string(nil), result.AllowedActions...),
		ForbiddenActions:        append([]string(nil), result.ForbiddenActions...),
		RiskAssessmentSummary:   riskSummary(result),
		ResearchEvidenceSummary: researchSummary(result),
		PaperTicketSummary:      paperSummary(result.PaperTicketResult),
		ReviewSchedule:          result.ReviewScheduleResult,
		HumanApprovalRequired:   result.HumanApprovalRequired,
		PaperOnly:               result.PaperOnly,
		LiveTradingBlocked:      result.LiveTradingBlocked,
		ValidationWarnings:      append([]string(nil), result.ValidationWarnings...),
		ValidationErrors:        append([]string(nil), result.ValidationErrors...),
		CreatedAt:               result.CreatedAt,
	}
	return record
}

func defaultPipelineModules(result pipeline.Result) []string {
	modules := []string{"event_intelligence", "decision_core", "swing_brain", "risk_veto", "review_schedule"}
	if result.ResearchEvidenceResult != nil {
		modules = append(modules, "research_evidence")
	}
	if result.PaperTicketResult != nil {
		modules = append(modules, "paper_review")
	}
	return modules
}

func noTradeReason(result pipeline.Result) string {
	if result.FinalDecision == "NO_TRADE" {
		return result.DecisionCoreResult.FinalDecision.PrimaryReason
	}
	return ""
}

func rejectionReason(result pipeline.Result) string {
	if len(result.RiskAssessment.VetoReasons) > 0 {
		return result.RiskAssessment.VetoReasons[0]
	}
	return ""
}

func riskSummary(result pipeline.Result) RiskAssessmentSummary {
	return RiskAssessmentSummary{
		RiskDecision:       string(result.RiskAssessment.RiskDecision),
		FinalDecision:      string(result.RiskAssessment.FinalDecision),
		VetoReasons:        append([]string(nil), result.RiskAssessment.VetoReasons...),
		DowngradeReasons:   append([]string(nil), result.RiskAssessment.DowngradeReasons...),
		RequiredActions:    append([]string(nil), result.RiskAssessment.RequiredActions...),
		LiveTradingBlocked: result.RiskAssessment.LiveTradingBlocked,
	}
}

func researchSummary(result pipeline.Result) ResearchEvidenceSummary {
	if result.ResearchEvidenceResult == nil {
		return ResearchEvidenceSummary{}
	}
	return ResearchEvidenceSummary{
		PromotionDecision: string(result.ResearchEvidenceResult.PromotionDecision),
		Summary:           string(result.ResearchEvidenceResult.MaxAllowedPromotionState),
		Warnings:          append([]string(nil), result.ResearchEvidenceResult.ValidationWarnings...),
	}
}

func paperSummary(ticket *paper.PaperTicket) *PaperTicketSummary {
	if ticket == nil {
		return nil
	}
	return &PaperTicketSummary{
		PaperTicketID:       ticket.PaperTicketID,
		HumanApprovalStatus: string(ticket.HumanApprovalStatus),
		PaperOnly:           ticket.PaperOnly,
		LiveTradingBlocked:  ticket.LiveTradingBlocked,
	}
}
