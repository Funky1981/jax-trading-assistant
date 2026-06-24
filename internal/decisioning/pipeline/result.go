package pipeline

import (
	"time"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/classify"
	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/review"
	"jax-trading-assistant/internal/decisioning/risk"
)

type FinalStatus string

const (
	StatusNoTradeRecorded                   FinalStatus = "NO_TRADE_RECORDED"
	StatusWatchRecorded                     FinalStatus = "WATCH_RECORDED"
	StatusSetupFormingRecorded              FinalStatus = "SETUP_FORMING_RECORDED"
	StatusTradeCandidateRejectedByRisk      FinalStatus = "TRADE_CANDIDATE_REJECTED_BY_RISK"
	StatusTradeCandidateNeedsResearch       FinalStatus = "TRADE_CANDIDATE_NEEDS_RESEARCH"
	StatusTradeCandidateReadyForPaperReview FinalStatus = "TRADE_CANDIDATE_READY_FOR_PAPER_REVIEW"
	StatusPaperReviewBlocked                FinalStatus = "PAPER_REVIEW_BLOCKED"
	StatusPipelineInvalid                   FinalStatus = "PIPELINE_INVALID"
)

type Result struct {
	PipelineID              string                     `json:"pipeline_id"`
	EventID                 string                     `json:"event_id"`
	EventIntelligenceResult classify.EventIntelligence `json:"event_intelligence_result"`
	DecisionCoreResult      core.EvidenceBundle        `json:"decision_core_result"`
	SwingBrainResult        swing.Decision             `json:"swing_brain_result"`
	RiskAssessment          risk.RiskAssessment        `json:"risk_assessment"`
	ResearchEvidenceResult  *research.ValidationResult `json:"research_evidence_result,omitempty"`
	PaperTicketResult       *paper.PaperTicket         `json:"paper_ticket_result,omitempty"`
	ReviewScheduleResult    review.ReviewSchedule      `json:"review_schedule_result"`
	FinalDecision           core.DecisionValue         `json:"final_decision"`
	FinalStatus             FinalStatus                `json:"final_status"`
	AllowedActions          []string                   `json:"allowed_actions"`
	ForbiddenActions        []string                   `json:"forbidden_actions"`
	HumanApprovalRequired   bool                       `json:"human_approval_required"`
	PaperOnly               bool                       `json:"paper_only"`
	LiveTradingBlocked      bool                       `json:"live_trading_blocked"`
	ValidationErrors        []string                   `json:"validation_errors"`
	ValidationWarnings      []string                   `json:"validation_warnings"`
	CreatedAt               time.Time                  `json:"created_at"`
}
