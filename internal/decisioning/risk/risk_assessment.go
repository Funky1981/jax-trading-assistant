package risk

import "jax-trading-assistant/internal/decisioning/core"

type RiskDecision string

const (
	RiskDecisionPass                 RiskDecision = "PASS"
	RiskDecisionDowngradeToWatch     RiskDecision = "DOWNGRADE_TO_WATCH"
	RiskDecisionReject               RiskDecision = "REJECT"
	RiskDecisionRejectedByRisk       RiskDecision = "REJECTED_BY_RISK"
	RiskDecisionNeedsMoreInformation RiskDecision = "NEEDS_MORE_INFORMATION"
)

const (
	RequiredActionHumanApprovalRequired = "human_approval_required"
	RequiredActionProvidePortfolio      = "provide_portfolio_context"
	RequiredActionResolveEventRisk      = "resolve_event_risk"
	RequiredActionReviewExposure        = "review_exposure"
)

type RiskAssessment struct {
	RiskDecision          RiskDecision       `json:"risk_decision"`
	OriginalDecision      core.DecisionValue `json:"original_decision"`
	FinalDecision         core.DecisionValue `json:"final_decision"`
	VetoReasons           []string           `json:"veto_reasons"`
	DowngradeReasons      []string           `json:"downgrade_reasons"`
	RequiredActions       []string           `json:"required_actions"`
	ForbiddenActions      []string           `json:"forbidden_actions"`
	AllowedActions        []string           `json:"allowed_actions"`
	MaxPositionRisk       float64            `json:"max_position_risk"`
	MaxPositionSize       float64            `json:"max_position_size,omitempty"`
	RequiresHumanApproval bool               `json:"requires_human_approval"`
	PaperOnly             bool               `json:"paper_only"`
	LiveTradingBlocked    bool               `json:"live_trading_blocked"`
	ReviewAfter           []string           `json:"review_after"`
}
