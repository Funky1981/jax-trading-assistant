package core

import (
	"strings"
	"time"
)

type EvaluationInput struct {
	Event                  Event
	Scores                 Scores
	MarketContext          map[string]any
	GeneratedAt            time.Time
	SupportingReasons      []string
	RequiredConfirmations  []string
	MissingConfirmations   []string
	InvalidationConditions []string
}

func Evaluate(input EvaluationInput) EvidenceBundle {
	scores := input.Scores.withConfidence()
	decisionValue, reason := evaluateDecision(scores, input)
	decisionValue, reason = enforcePhaseOneDowngrades(decisionValue, reason, input)

	decision := Decision{
		DecisionID:             deterministicID("dec", input.Event.EventID),
		EventID:                input.Event.EventID,
		Brain:                  BrainDecisionCore,
		Decision:               decisionValue,
		ConfidenceScore:        scores.ConfidenceScore,
		ClarityScore:           scores.ClarityScore,
		EdgeScore:              scores.EdgeScore,
		ConflictScore:          scores.ConflictScore,
		RiskScore:              scores.RiskScore,
		PrimaryReason:          reason,
		SupportingReasons:      supportingReasons(input, decisionValue),
		RequiredConfirmations:  requiredConfirmations(input, decisionValue),
		InvalidationConditions: invalidationConditions(input, decisionValue),
		AllowedActions:         allowedActions(decisionValue),
		ForbiddenActions:       defaultForbiddenActions(),
		ReviewAfter:            defaultReviewWindows(),
	}

	generatedAt := input.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = input.Event.ReceivedAt
	}

	return EvidenceBundle{
		EvidenceID:            deterministicID("evb", input.Event.EventID),
		InputEvent:            input.Event,
		MarketContext:         marketContext(input.MarketContext),
		ReasoningTraceSummary: decision.PrimaryReason,
		Scores:                scores,
		FinalDecision:         decision,
		GeneratedAt:           generatedAt,
		Version:               VersionDecisionCoreV1,
	}
}

func evaluateDecision(scores Scores, input EvaluationInput) (DecisionValue, string) {
	switch {
	case scores.ConflictScore >= 0.70 && scores.EdgeScore < 0.60:
		return DecisionNoTrade, "Conflicting macro drivers and no clean asset-specific edge."
	case scores.ClarityScore < 0.50:
		return DecisionNoTrade, "Decision clarity is too low to justify upgrading from NO_TRADE."
	case scores.RiskScore > 0.70:
		return DecisionNoTrade, "Risk score is too high to justify a trade candidate."
	case scores.EdgeScore >= 0.50 && scores.EdgeScore < 0.75 && len(input.MissingConfirmations) > 0:
		return DecisionWatch, "Potential edge exists, but required confirmations are missing."
	case scores.EdgeScore >= 0.75 && scores.RiskScore <= 0.60 && scores.ClarityScore >= 0.70 && scores.ConflictScore < 0.60:
		return DecisionTradeCandidate, "Structured evidence supports a trade candidate for human review."
	default:
		return DecisionNoTrade, "Evidence does not justify upgrading from NO_TRADE."
	}
}

func enforcePhaseOneDowngrades(decision DecisionValue, reason string, input EvaluationInput) (DecisionValue, string) {
	if decision == DecisionPaperApprovalRequired {
		return DecisionTradeCandidate, "Paper approval requires later-phase workflow; Phase 1 returns TRADE_CANDIDATE only."
	}
	if decision != DecisionTradeCandidate {
		return decision, reason
	}
	if len(input.RequiredConfirmations) == 0 || len(input.InvalidationConditions) == 0 {
		return DecisionWatch, "Candidate evidence is promising, but confirmations or invalidation conditions are incomplete."
	}
	return decision, reason
}

func supportingReasons(input EvaluationInput, decision DecisionValue) []string {
	if len(input.SupportingReasons) > 0 {
		return cloneStrings(input.SupportingReasons)
	}
	if decision == DecisionNoTrade && len(input.Event.ConflictingDrivers) > 0 {
		return []string{
			"Conflicting drivers: " + strings.Join(input.Event.ConflictingDrivers, ", "),
			"NO_TRADE is the default until evidence improves.",
		}
	}
	return []string{"Decision follows deterministic Decision Core v1 scoring rules."}
}

func requiredConfirmations(input EvaluationInput, decision DecisionValue) []string {
	if len(input.RequiredConfirmations) > 0 {
		return cloneStrings(input.RequiredConfirmations)
	}
	if len(input.MissingConfirmations) > 0 {
		return cloneStrings(input.MissingConfirmations)
	}
	if decision == DecisionNoTrade {
		return []string{"conflicting drivers resolve", "asset-specific edge becomes clear"}
	}
	return []string{"risk veto pass", "confirmation remains valid"}
}

func invalidationConditions(input EvaluationInput, decision DecisionValue) []string {
	if len(input.InvalidationConditions) > 0 {
		return cloneStrings(input.InvalidationConditions)
	}
	if decision == DecisionNoTrade {
		return []string{"conflicting macro drivers remain unresolved"}
	}
	return []string{"setup loses catalyst support"}
}

func allowedActions(decision DecisionValue) []string {
	base := []string{ActionStoreEvent, ActionMonitor, ActionReviewLater}
	if decision == DecisionTradeCandidate {
		return append(base, ActionPreparePaper)
	}
	return base
}

func marketContext(ctx map[string]any) map[string]any {
	if ctx == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(ctx))
	for key, value := range ctx {
		out[key] = value
	}
	return out
}

func deterministicID(prefix, eventID string) string {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		eventID = "unknown_event"
	}
	return prefix + "_" + eventID
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}
