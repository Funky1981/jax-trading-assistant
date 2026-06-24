package swing

import (
	"strings"

	"jax-trading-assistant/internal/decisioning/core"
)

type ruleOutcome struct {
	decision      core.DecisionValue
	primaryReason string
}

func decide(input EvaluationInput, scores core.Scores) ruleOutcome {
	switch {
	case scores.ConflictScore >= 0.70 && scores.EdgeScore < 0.60:
		return noTrade("Conflicting drivers dominate and edge is weak.")
	case scores.ClarityScore < 0.50:
		return noTrade("Event clarity is below the swing threshold.")
	case scores.RiskScore > 0.70:
		return noTrade("Risk score is above the swing threshold.")
	case strings.TrimSpace(input.Catalyst) == "":
		return noTrade("No clear catalyst exists.")
	case len(input.InvalidationConditions) == 0:
		return noTrade("Missing invalidation condition.")
	case input.RiskRewardRatio <= 0:
		return noTrade("Risk/reward is not supplied.")
	case input.RiskRewardRatio < 2:
		return noTrade("Risk/reward is below the 2:1 minimum.")
	case len(input.MissingConfirmations) > 0:
		return ruleOutcome{decision: core.DecisionWatch, primaryReason: "Required confirmations are missing."}
	case len(input.UnresolvedEventRisk) > 0:
		return ruleOutcome{decision: core.DecisionWatch, primaryReason: "Major unresolved event risk is pending."}
	case readyForCandidate(input, scores):
		return ruleOutcome{decision: core.DecisionTradeCandidate, primaryReason: "Swing trade candidate meets catalyst, confirmation, invalidation, and risk/reward requirements."}
	default:
		return ruleOutcome{decision: core.DecisionSetupForming, primaryReason: "Swing setup is forming but is not ready."}
	}
}

func readyForCandidate(input EvaluationInput, scores core.Scores) bool {
	return scores.ClarityScore >= 0.50 &&
		scores.EdgeScore >= 0.75 &&
		scores.ConflictScore < 0.70 &&
		scores.RiskScore <= 0.70 &&
		strings.TrimSpace(input.Catalyst) != "" &&
		len(input.InvalidationConditions) > 0 &&
		input.RiskRewardRatio >= 2 &&
		len(input.RequiredConfirmations) > 0 &&
		len(input.MissingConfirmations) == 0 &&
		len(input.UnresolvedEventRisk) == 0 &&
		len(input.PresentConfirmations) >= len(input.RequiredConfirmations)
}

func noTrade(reason string) ruleOutcome {
	return ruleOutcome{decision: core.DecisionNoTrade, primaryReason: reason}
}
