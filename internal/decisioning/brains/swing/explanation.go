package swing

import (
	"fmt"
	"strings"

	"jax-trading-assistant/internal/decisioning/core"
)

func supportingReasons(input EvaluationInput, outcome ruleOutcome, family SetupFamily) []string {
	reasons := cloneStrings(input.SupportingReasons)
	if input.Catalyst != "" {
		reasons = append(reasons, "Catalyst: "+input.Catalyst)
	}
	if family != UnknownSwingSetup {
		reasons = append(reasons, "Setup family: "+string(family))
	}
	if input.MarketSectorAlignmentNote != "" {
		reasons = append(reasons, "Market/sector alignment: "+input.MarketSectorAlignmentNote)
	}
	if input.RiskRewardRatio > 0 {
		reasons = append(reasons, fmt.Sprintf("Risk/reward: %.2f:1", input.RiskRewardRatio))
	}
	if len(input.UnresolvedEventRisk) > 0 {
		reasons = append(reasons, "Unresolved event risk: "+strings.Join(input.UnresolvedEventRisk, ", "))
	}
	if len(input.Event.UncertaintyNotes) > 0 {
		reasons = append(reasons, "Event uncertainty: "+strings.Join(input.Event.UncertaintyNotes, ", "))
	}
	if len(reasons) == 0 {
		reasons = append(reasons, outcome.primaryReason)
	}
	return reasons
}

func requiredConfirmations(input EvaluationInput, outcome ruleOutcome) []string {
	confirmations := cloneStrings(input.RequiredConfirmations)
	if outcome.decision == core.DecisionNoTrade && len(confirmations) == 0 {
		confirmations = append(confirmations, "new evidence that creates a clean swing edge")
	}
	return confirmations
}

func allowedActions(decision core.DecisionValue) []string {
	switch decision {
	case core.DecisionTradeCandidate:
		return []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionPrepareResearch, core.ActionPreparePaper}
	case core.DecisionWatch, core.DecisionSetupForming:
		return []string{core.ActionStoreEvent, core.ActionMonitor, core.ActionReviewLater, core.ActionPrepareResearch}
	default:
		return []string{core.ActionStoreEvent, core.ActionReviewLater}
	}
}

func forbiddenActions() []string {
	return []string{core.ActionExecuteTrade, core.ActionCreateLiveOrder, core.ActionAutoApprove}
}

func reviewAfter() []string {
	return []string{core.ReviewAfter1Day, core.ReviewAfter1Week, core.ReviewAfter1Month}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}
