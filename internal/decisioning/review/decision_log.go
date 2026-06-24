package review

import (
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/paper"
)

type RiskAssessmentSummary struct {
	RiskDecision       string   `json:"risk_decision"`
	FinalDecision      string   `json:"final_decision"`
	VetoReasons        []string `json:"veto_reasons,omitempty"`
	DowngradeReasons   []string `json:"downgrade_reasons,omitempty"`
	RequiredActions    []string `json:"required_actions,omitempty"`
	LiveTradingBlocked bool     `json:"live_trading_blocked"`
}

type ResearchEvidenceSummary struct {
	PromotionDecision string   `json:"promotion_decision"`
	Summary           string   `json:"summary"`
	Warnings          []string `json:"warnings,omitempty"`
}

type PaperTicketSummary struct {
	PaperTicketID       string               `json:"paper_ticket_id"`
	HumanApprovalStatus paper.ApprovalStatus `json:"human_approval_status"`
	PaperOnly           bool                 `json:"paper_only"`
	LiveTradingBlocked  bool                 `json:"live_trading_blocked"`
}

type DecisionLogInput struct {
	Decision                core.Decision           `json:"decision"`
	FinalDecision           string                  `json:"final_decision"`
	Asset                   string                  `json:"asset"`
	SetupFamily             string                  `json:"setup_family"`
	RiskAssessmentSummary   RiskAssessmentSummary   `json:"risk_assessment_summary"`
	ResearchEvidenceSummary ResearchEvidenceSummary `json:"research_evidence_summary"`
	PaperTicketSummary      PaperTicketSummary      `json:"paper_ticket_summary"`
	CreatedAt               time.Time               `json:"created_at"`
	MemoryTags              []string                `json:"memory_tags"`
}

type DecisionLog struct {
	DecisionLogID           string                  `json:"decision_log_id"`
	DecisionID              string                  `json:"decision_id"`
	EventID                 string                  `json:"event_id"`
	SourceBrain             string                  `json:"source_brain"`
	OriginalDecision        core.DecisionValue      `json:"original_decision"`
	FinalDecision           string                  `json:"final_decision"`
	Asset                   string                  `json:"asset"`
	SetupFamily             string                  `json:"setup_family"`
	PrimaryReason           string                  `json:"primary_reason"`
	SupportingReasons       []string                `json:"supporting_reasons"`
	Scores                  core.Scores             `json:"scores"`
	AllowedActions          []string                `json:"allowed_actions"`
	ForbiddenActions        []string                `json:"forbidden_actions"`
	RequiredConfirmations   []string                `json:"required_confirmations"`
	InvalidationConditions  []string                `json:"invalidation_conditions"`
	RiskAssessmentSummary   RiskAssessmentSummary   `json:"risk_assessment_summary"`
	ResearchEvidenceSummary ResearchEvidenceSummary `json:"research_evidence_summary"`
	PaperTicketSummary      PaperTicketSummary      `json:"paper_ticket_summary"`
	CreatedAt               time.Time               `json:"created_at"`
	ReviewSchedule          ReviewSchedule          `json:"review_schedule"`
	MemoryTags              []string                `json:"memory_tags"`
}

func NewDecisionLog(input DecisionLogInput) (DecisionLog, ValidationResult) {
	forbidden := mandatoryForbiddenActions(input.Decision.ForbiddenActions)
	log := DecisionLog{
		DecisionLogID:     "dlog_" + input.Decision.DecisionID,
		DecisionID:        input.Decision.DecisionID,
		EventID:           input.Decision.EventID,
		SourceBrain:       input.Decision.Brain,
		OriginalDecision:  input.Decision.Decision,
		FinalDecision:     firstNonEmpty(input.FinalDecision, string(input.Decision.Decision)),
		Asset:             input.Asset,
		SetupFamily:       input.SetupFamily,
		PrimaryReason:     input.Decision.PrimaryReason,
		SupportingReasons: input.Decision.SupportingReasons,
		Scores: core.Scores{
			ClarityScore:  input.Decision.ClarityScore,
			EdgeScore:     input.Decision.EdgeScore,
			ConflictScore: input.Decision.ConflictScore,
			RiskScore:     input.Decision.RiskScore,
		},
		AllowedActions:          input.Decision.AllowedActions,
		ForbiddenActions:        forbidden,
		RequiredConfirmations:   input.Decision.RequiredConfirmations,
		InvalidationConditions:  input.Decision.InvalidationConditions,
		RiskAssessmentSummary:   input.RiskAssessmentSummary,
		ResearchEvidenceSummary: input.ResearchEvidenceSummary,
		PaperTicketSummary:      input.PaperTicketSummary,
		CreatedAt:               input.CreatedAt,
		ReviewSchedule:          NewReviewSchedule(input.Decision.DecisionID, input.CreatedAt),
		MemoryTags:              input.MemoryTags,
	}
	return log, ValidateDecisionLog(log)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
