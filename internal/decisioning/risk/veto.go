package risk

import (
	"strings"

	"jax-trading-assistant/internal/decisioning/brains/swing"
	"jax-trading-assistant/internal/decisioning/core"
)

const minimumRiskRewardRatio = 2.0

type AssessmentInput struct {
	SwingDecision            swing.Decision    `json:"swing_decision"`
	Asset                    string            `json:"asset"`
	SetupFamily              swing.SetupFamily `json:"setup_family"`
	ProposedEntry            float64           `json:"proposed_entry,omitempty"`
	ProposedStop             float64           `json:"proposed_stop,omitempty"`
	ProposedTarget           float64           `json:"proposed_target,omitempty"`
	RiskRewardRatio          float64           `json:"risk_reward_ratio,omitempty"`
	AccountMode              AccountMode       `json:"account_mode"`
	Portfolio                PortfolioContext  `json:"portfolio"`
	CurrentExposure          Exposure          `json:"current_exposure"`
	SectorExposure           Exposure          `json:"sector_exposure"`
	CorrelatedExposure       Exposure          `json:"correlated_exposure"`
	LiquiditySpreadNotes     []string          `json:"liquidity_spread_notes,omitempty"`
	UnresolvedEventRisk      []string          `json:"unresolved_event_risk,omitempty"`
	UpcomingMajorEventRisk   []string          `json:"upcoming_major_event_risk,omitempty"`
	BrokerExecutionRequested bool              `json:"broker_execution_requested,omitempty"`
	LiveOrderRequested       bool              `json:"live_order_requested,omitempty"`
}

func Assess(input AssessmentInput) RiskAssessment {
	original := input.SwingDecision.Decision.Decision
	assessment := baseAssessment(input, original)

	if original == core.DecisionNoTrade {
		return assessment
	}
	if original == core.DecisionWatch || original == core.DecisionSetupForming {
		applyNonCandidateDowngrades(input, &assessment)
		return assessment
	}
	if original != core.DecisionTradeCandidate {
		reject(&assessment, "unsupported decision cannot progress through risk veto")
		return assessment
	}

	applyCandidateVetoes(input, &assessment)
	if assessment.RiskDecision == RiskDecisionRejectedByRisk || assessment.RiskDecision == RiskDecisionReject {
		assessment.FinalDecision = core.DecisionRejectedByRisk
		assessment.AllowedActions = allowedActionsForFinal(assessment.FinalDecision, assessment.RiskDecision)
		return assessment
	}

	applyCandidateDowngrades(input, &assessment)
	if assessment.RiskDecision == RiskDecisionDowngradeToWatch || assessment.RiskDecision == RiskDecisionNeedsMoreInformation {
		assessment.FinalDecision = core.DecisionWatch
	}
	assessment.AllowedActions = allowedActionsForFinal(assessment.FinalDecision, assessment.RiskDecision)
	return assessment
}

func baseAssessment(input AssessmentInput, original core.DecisionValue) RiskAssessment {
	maxRisk := maxPositionRisk(input.Portfolio)
	return RiskAssessment{
		RiskDecision:          RiskDecisionPass,
		OriginalDecision:      original,
		FinalDecision:         original,
		RequiredActions:       []string{RequiredActionHumanApprovalRequired},
		ForbiddenActions:      mandatoryForbiddenActions(input.SwingDecision.ForbiddenActions),
		AllowedActions:        allowedActionsForFinal(original, RiskDecisionPass),
		MaxPositionRisk:       maxRisk,
		MaxPositionSize:       maxPositionSize(input.ProposedEntry, input.ProposedStop, maxRisk),
		RequiresHumanApproval: true,
		PaperOnly:             true,
		LiveTradingBlocked:    true,
		ReviewAfter:           reviewAfter(input.SwingDecision.ReviewAfter),
	}
}

func applyCandidateVetoes(input AssessmentInput, assessment *RiskAssessment) {
	if strings.TrimSpace(input.Asset) == "" {
		reject(assessment, "asset is required for risk assessment")
	}
	if input.SetupFamily == "" || input.SetupFamily == swing.UnknownSwingSetup {
		needsMoreInformation(assessment, "setup family is missing or unknown")
	}
	if input.RiskRewardRatio <= 0 {
		reject(assessment, "risk/reward is required")
	} else if input.RiskRewardRatio < minimumRiskRewardRatio {
		reject(assessment, "risk/reward is below 2:1 minimum")
	}
	if input.ProposedStop <= 0 && len(input.SwingDecision.InvalidationConditions) == 0 {
		reject(assessment, "stop or invalidation is required")
	}
	if input.AccountMode == AccountModeLive {
		reject(assessment, "live trading is out of scope")
	}
	if input.BrokerExecutionRequested || input.LiveOrderRequested {
		reject(assessment, "broker execution and live order actions are out of scope")
	}
	if hasPoorLiquidity(input.LiquiditySpreadNotes) {
		reject(assessment, "spread/liquidity is marked poor")
	}
}

func applyCandidateDowngrades(input AssessmentInput, assessment *RiskAssessment) {
	if !input.Portfolio.hasUsableEquity() {
		needsMoreInformation(assessment, "portfolio context is required")
		addRequiredAction(assessment, RequiredActionProvidePortfolio)
	}
	if len(input.UnresolvedEventRisk) > 0 || len(input.UpcomingMajorEventRisk) > 0 {
		downgrade(assessment, "unresolved major event risk requires review")
		addRequiredAction(assessment, RequiredActionResolveEventRisk)
	}
	if input.CurrentExposure.above(defaultMaxCurrentExposurePct) {
		downgrade(assessment, "current exposure is above risk threshold")
		addRequiredAction(assessment, RequiredActionReviewExposure)
	}
	if input.SectorExposure.above(defaultMaxSectorExposurePct) {
		downgrade(assessment, "sector exposure is above 25% threshold")
		addRequiredAction(assessment, RequiredActionReviewExposure)
	}
	if input.CorrelatedExposure.above(defaultMaxCorrelatedExposurePct) {
		downgrade(assessment, "correlated exposure is above 30% threshold")
		addRequiredAction(assessment, RequiredActionReviewExposure)
	}
}

func applyNonCandidateDowngrades(input AssessmentInput, assessment *RiskAssessment) {
	if len(input.UnresolvedEventRisk) > 0 || len(input.UpcomingMajorEventRisk) > 0 {
		downgrade(assessment, "unresolved major event risk requires review")
		addRequiredAction(assessment, RequiredActionResolveEventRisk)
		assessment.FinalDecision = input.SwingDecision.Decision.Decision
	}
	assessment.AllowedActions = allowedActionsForFinal(assessment.FinalDecision, assessment.RiskDecision)
}

func reject(assessment *RiskAssessment, reason string) {
	assessment.RiskDecision = RiskDecisionRejectedByRisk
	assessment.VetoReasons = appendUnique(assessment.VetoReasons, reason)
}

func downgrade(assessment *RiskAssessment, reason string) {
	if assessment.RiskDecision == RiskDecisionPass {
		assessment.RiskDecision = RiskDecisionDowngradeToWatch
	}
	if assessment.RiskDecision == RiskDecisionDowngradeToWatch {
		assessment.DowngradeReasons = appendUnique(assessment.DowngradeReasons, reason)
	}
}

func needsMoreInformation(assessment *RiskAssessment, reason string) {
	if assessment.RiskDecision == RiskDecisionPass || assessment.RiskDecision == RiskDecisionDowngradeToWatch {
		assessment.RiskDecision = RiskDecisionNeedsMoreInformation
	}
	assessment.DowngradeReasons = appendUnique(assessment.DowngradeReasons, reason)
}

func addRequiredAction(assessment *RiskAssessment, action string) {
	assessment.RequiredActions = appendUnique(assessment.RequiredActions, action)
}

func hasPoorLiquidity(notes []string) bool {
	for _, note := range notes {
		normalised := strings.ToLower(note)
		if strings.Contains(normalised, "poor") || strings.Contains(normalised, "illiquid") || strings.Contains(normalised, "wide spread") {
			return true
		}
	}
	return false
}

func allowedActionsForFinal(decision core.DecisionValue, riskDecision RiskDecision) []string {
	switch decision {
	case core.DecisionTradeCandidate:
		if riskDecision == RiskDecisionPass {
			return []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper}
		}
		return []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch}
	case core.DecisionWatch, core.DecisionSetupForming:
		return []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionReviewLater, core.ActionPrepareResearch}
	default:
		return []string{core.ActionStoreEvent, core.ActionReviewLater}
	}
}

func mandatoryForbiddenActions(existing []string) []string {
	actions := append([]string{}, existing...)
	actions = appendUnique(actions, core.ActionExecuteTrade)
	actions = appendUnique(actions, core.ActionCreateLiveOrder)
	actions = appendUnique(actions, core.ActionAutoApprove)
	return actions
}

func reviewAfter(existing []string) []string {
	if len(existing) > 0 {
		return append([]string{}, existing...)
	}
	return []string{core.ReviewAfter1Day, core.ReviewAfter1Week, core.ReviewAfter1Month}
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
