package macroevents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ScenarioKey string
type ScenarioResult string
type CandidateBias string

const (
	ScenarioHawkishRates    ScenarioKey = "hawkish_rates"
	ScenarioDovishRates     ScenarioKey = "dovish_rates"
	ScenarioSemiconductorAI ScenarioKey = "semiconductor_ai"
	ScenarioBankStress      ScenarioKey = "bank_stress"
	ScenarioOilShock        ScenarioKey = "oil_shock"
	ScenarioUnknown         ScenarioKey = "unknown"
)

const (
	ScenarioResultEligibleForReactionCheck    ScenarioResult = "eligible_for_reaction_check"
	ScenarioResultBlockedUnknownEvent         ScenarioResult = "blocked_unknown_event"
	ScenarioResultBlockedNoETFMapping         ScenarioResult = "blocked_no_etf_mapping"
	ScenarioResultBlockedConflicting          ScenarioResult = "blocked_conflicting_scenario"
	ScenarioResultBlockedDisallowedInstrument ScenarioResult = "blocked_disallowed_instrument"
)

const (
	CandidateBiasShortOrAvoidLong CandidateBias = "short_or_avoid_long"
	CandidateBiasLong             CandidateBias = "long"
	CandidateBiasRiskOff          CandidateBias = "risk_off"
	CandidateBiasWatchOnly        CandidateBias = "watch_only"
)

type ScenarioEvaluation struct {
	ID                        string
	MacroEventID              string
	PlaybookKey               string
	PlaybookName              string
	ScenarioKey               ScenarioKey
	CandidateBias             CandidateBias
	PrimarySymbols            []string
	SecondarySymbols          []string
	RequiredConfirmations     []string
	RequiredTechnicalChecks   []string
	RequiredFundamentalChecks []string
	ConfoundersToCheck        []string
	WalkawayRules             []string
	ExpectedReactions         map[string]ReactionDirection
	Result                    ScenarioResult
	Reason                    string
}

func EvaluateScenario(input EventInput) ScenarioEvaluation {
	if !supportedEventType(input.EventType) {
		return blockedScenario(ScenarioResultBlockedUnknownEvent, "macro event type is not supported")
	}
	playbook, ok := LookupEventPlaybook(input)
	if !ok {
		return blockedScenario(ScenarioResultBlockedUnknownEvent, "no deterministic macro playbook matched")
	}
	mappings, err := ValidateAndNormalizeETFs(input.AffectedETFs)
	if err != nil {
		if len(nonEmptyStrings(input.AffectedETFs)) == 0 {
			return blockedScenario(ScenarioResultBlockedNoETFMapping, "macro event has no ETF mapping")
		}
		return blockedScenario(ScenarioResultBlockedDisallowedInstrument, err.Error())
	}

	return scenarioFromPlaybook(input, mappings, ScenarioEvaluation{
		PlaybookKey:               playbook.Key,
		PlaybookName:              playbook.Name,
		ScenarioKey:               playbook.ScenarioKey,
		CandidateBias:             playbook.CandidateBias,
		PrimarySymbols:            append([]string(nil), playbook.PrimarySymbols...),
		SecondarySymbols:          append([]string(nil), playbook.SecondarySymbols...),
		RequiredConfirmations:     append([]string(nil), playbook.RequiredConfirmations...),
		RequiredTechnicalChecks:   append([]string(nil), playbook.RequiredTechnicalChecks...),
		RequiredFundamentalChecks: append([]string(nil), playbook.RequiredFundamentalChecks...),
		ConfoundersToCheck:        append([]string(nil), playbook.ConfoundersToCheck...),
		WalkawayRules:             append([]string(nil), playbook.WalkawayRules...),
		ExpectedReactions:         playbook.ExpectedReactions,
		Result:                    ScenarioResultEligibleForReactionCheck,
		Reason:                    playbook.Name + " playbook selected",
	})
}

func hawkishConfirmations(eventType EventType) []string {
	switch EventType(strings.ToUpper(strings.TrimSpace(string(eventType)))) {
	case EventTypeFedChairPressConference, EventTypeFOMCStatement:
		return []string{string(TimeframePostEvent15M), string(TimeframePostEvent30M)}
	default:
		return []string{string(TimeframePostEvent5M), string(TimeframePostEvent15M)}
	}
}

func scenarioFromPlaybook(input EventInput, mappings []ETFMapping, base ScenarioEvaluation) ScenarioEvaluation {
	allowed := map[string]bool{}
	for _, mapping := range mappings {
		allowed[mapping.Symbol] = true
	}
	base.PrimarySymbols = filterAllowedSymbols(base.PrimarySymbols, allowed)
	base.SecondarySymbols = filterAllowedSymbols(base.SecondarySymbols, allowed)
	if len(base.PrimarySymbols) == 0 && len(base.SecondarySymbols) == 0 {
		return blockedScenario(ScenarioResultBlockedNoETFMapping, "scenario has no matching ETF mapping")
	}
	base.MacroEventID = strings.TrimSpace(input.MacroEventID)
	if base.MacroEventID == "" {
		base.MacroEventID = strings.TrimSpace(input.SourceEventID)
	}
	return base
}

func filterAllowedSymbols(symbols []string, allowed map[string]bool) []string {
	out := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		normalized := strings.ToUpper(strings.TrimSpace(symbol))
		if allowed[normalized] {
			out = append(out, normalized)
		}
	}
	return out
}

func blockedScenario(result ScenarioResult, reason string) ScenarioEvaluation {
	return ScenarioEvaluation{
		ScenarioKey:               ScenarioUnknown,
		PlaybookKey:               "",
		PlaybookName:              "",
		CandidateBias:             CandidateBiasWatchOnly,
		RequiredConfirmations:     []string{},
		RequiredTechnicalChecks:   []string{},
		RequiredFundamentalChecks: []string{},
		ConfoundersToCheck:        []string{},
		WalkawayRules:             []string{},
		ExpectedReactions:         map[string]ReactionDirection{},
		Result:                    result,
		Reason:                    reason,
	}
}

type scenarioStore interface {
	SaveScenarioResult(ctx context.Context, result ScenarioEvaluation) (ScenarioEvaluation, error)
}

func MarshalExpectedReactions(reactions map[string]ReactionDirection) ([]byte, error) {
	raw := map[string]string{}
	for symbol, direction := range reactions {
		raw[symbol] = string(direction)
	}
	return json.Marshal(raw)
}

func UnmarshalExpectedReactions(raw []byte) (map[string]ReactionDirection, error) {
	values := map[string]string{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("unmarshal expected reactions: %w", err)
	}
	out := map[string]ReactionDirection{}
	for symbol, direction := range values {
		out[symbol] = ReactionDirection(direction)
	}
	return out, nil
}
