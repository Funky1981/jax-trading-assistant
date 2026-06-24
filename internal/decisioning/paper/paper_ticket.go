package paper

import (
	"fmt"
	"time"

	"jax-trading-assistant/internal/decisioning/core"
	"jax-trading-assistant/internal/decisioning/research"
	"jax-trading-assistant/internal/decisioning/risk"
)

type EntryZone struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

func (z EntryZone) IsDefined() bool {
	return z.Low > 0 && z.High > 0 && z.Low <= z.High
}

type ResearchEvidenceSummary struct {
	HypothesisID      string                  `json:"hypothesis_id"`
	SetupFamily       string                  `json:"setup_family"`
	PromotionDecision research.PromotionState `json:"promotion_decision"`
	Summary           string                  `json:"summary"`
	Warnings          []string                `json:"warnings,omitempty"`
}

func (s ResearchEvidenceSummary) IsDefined() bool {
	return s.HypothesisID != "" ||
		s.SetupFamily != "" ||
		s.PromotionDecision != "" ||
		s.Summary != ""
}

type TicketRequest struct {
	PaperTicketID                 string                  `json:"paper_ticket_id"`
	SourceDecision                core.Decision           `json:"source_decision"`
	RiskAssessment                risk.RiskAssessment     `json:"risk_assessment"`
	ResearchEvidenceSummary       ResearchEvidenceSummary `json:"research_evidence_summary"`
	Asset                         string                  `json:"asset"`
	SetupFamily                   string                  `json:"setup_family"`
	ProposedEntryZone             EntryZone               `json:"proposed_entry_zone"`
	EntryUnavailableReason        string                  `json:"entry_unavailable_reason,omitempty"`
	ProposedStop                  float64                 `json:"proposed_stop"`
	ProposedTarget                float64                 `json:"proposed_target"`
	RiskReward                    float64                 `json:"risk_reward"`
	MaxPaperPositionSize          float64                 `json:"max_paper_position_size"`
	CreatedAt                     time.Time               `json:"created_at"`
	ExpiresAt                     time.Time               `json:"expires_at"`
	RequiredConfirmations         []string                `json:"required_confirmations,omitempty"`
	InvalidationConditions        []string                `json:"invalidation_conditions,omitempty"`
	ExplicitHumanApprovalRequired bool                    `json:"explicit_human_approval_required"`
}

type PaperTicket struct {
	PaperTicketID           string                  `json:"paper_ticket_id"`
	DecisionID              string                  `json:"decision_id"`
	EventID                 string                  `json:"event_id"`
	Asset                   string                  `json:"asset"`
	SetupFamily             string                  `json:"setup_family"`
	SourceDecision          core.Decision           `json:"source_decision"`
	RiskAssessment          risk.RiskAssessment     `json:"risk_assessment"`
	ResearchEvidenceSummary ResearchEvidenceSummary `json:"research_evidence_summary"`
	RequiredConfirmations   []string                `json:"required_confirmations"`
	InvalidationConditions  []string                `json:"invalidation_conditions"`
	ProposedEntryZone       EntryZone               `json:"proposed_entry_zone"`
	EntryUnavailableReason  string                  `json:"entry_unavailable_reason,omitempty"`
	ProposedStop            float64                 `json:"proposed_stop"`
	ProposedTarget          float64                 `json:"proposed_target"`
	RiskReward              float64                 `json:"risk_reward"`
	MaxPaperPositionSize    float64                 `json:"max_paper_position_size"`
	HumanApprovalStatus     ApprovalStatus          `json:"human_approval_status"`
	LifecycleState          LifecycleState          `json:"lifecycle_state"`
	CreatedAt               time.Time               `json:"created_at"`
	ExpiresAt               time.Time               `json:"expires_at"`
	PaperOnly               bool                    `json:"paper_only"`
	LiveTradingBlocked      bool                    `json:"live_trading_blocked"`
	ForbiddenActions        []string                `json:"forbidden_actions"`
	ReviewAfter             []string                `json:"review_after"`
}

func NewTicket(request TicketRequest) (PaperTicket, ValidationResult) {
	result := ValidateTicketRequest(request)
	if !result.CanCreateTicket {
		return PaperTicket{}, result
	}

	ticketID := request.PaperTicketID
	if ticketID == "" {
		ticketID = fmt.Sprintf("pt_%s", request.SourceDecision.DecisionID)
	}

	maxPaperPositionSize := request.MaxPaperPositionSize
	if maxPaperPositionSize == 0 {
		maxPaperPositionSize = request.RiskAssessment.MaxPositionSize
	}

	ticket := PaperTicket{
		PaperTicketID:           ticketID,
		DecisionID:              request.SourceDecision.DecisionID,
		EventID:                 request.SourceDecision.EventID,
		Asset:                   request.Asset,
		SetupFamily:             request.SetupFamily,
		SourceDecision:          request.SourceDecision,
		RiskAssessment:          request.RiskAssessment,
		ResearchEvidenceSummary: request.ResearchEvidenceSummary,
		RequiredConfirmations:   request.RequiredConfirmations,
		InvalidationConditions:  request.InvalidationConditions,
		ProposedEntryZone:       request.ProposedEntryZone,
		EntryUnavailableReason:  request.EntryUnavailableReason,
		ProposedStop:            request.ProposedStop,
		ProposedTarget:          request.ProposedTarget,
		RiskReward:              request.RiskReward,
		MaxPaperPositionSize:    maxPaperPositionSize,
		HumanApprovalStatus:     ApprovalPendingReview,
		LifecycleState:          LifecyclePendingReview,
		CreatedAt:               request.CreatedAt,
		ExpiresAt:               request.ExpiresAt,
		PaperOnly:               true,
		LiveTradingBlocked:      true,
		ForbiddenActions:        mandatoryForbiddenActions(request.RiskAssessment.ForbiddenActions, request.SourceDecision.ForbiddenActions),
		ReviewAfter:             request.RiskAssessment.ReviewAfter,
	}
	return ticket, result
}
