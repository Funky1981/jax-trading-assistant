package macroevents

import (
	"strings"
)

type EventPlaybook struct {
	Key                       string
	Name                      string
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
}

func LookupEventPlaybook(input EventInput) (EventPlaybook, bool) {
	headline := strings.ToLower(strings.TrimSpace(input.Headline + " " + input.Summary))
	switch EventType(strings.ToUpper(strings.TrimSpace(string(input.EventType)))) {
	case EventTypeUSNonfarmPayrolls, EventTypeUSUnemploymentRate, EventTypeUSAverageHourlyEarnings:
		if isBullishRatesDirection(input.Direction, headline) {
			return hotJobsPlaybook(input), true
		}
		if isDovishRatesDirection(input.Direction, headline) {
			return weakJobsPlaybook(input), true
		}
	case EventTypeUSCPIHeadline, EventTypeUSCPICore, EventTypeUSPPI:
		if isBullishRatesDirection(input.Direction, headline) {
			return hotCPIPlaybook(input), true
		}
		if isDovishRatesDirection(input.Direction, headline) {
			return coolCPIPlaybook(input), true
		}
	case EventTypeFOMCRateDecision, EventTypeFOMCStatement, EventTypeFOMCDotPlot, EventTypeFedChairPressConference:
		if isBullishRatesDirection(input.Direction, headline) {
			return hawkishFedPlaybook(input), true
		}
		if isDovishRatesDirection(input.Direction, headline) {
			return dovishFedPlaybook(input), true
		}
	}

	switch {
	case containsAny(headline, "nvidia", "semiconductor", "semis", "chip", "ai", "gpu"):
		return semiconductorPlaybook(input), true
	case containsAny(headline, "bank", "banks", "credit", "regional bank", "financial stress"):
		return bankStressPlaybook(input), true
	case containsAny(headline, "oil", "crude", "opec", "energy shock", "geopolitical"):
		return oilShockPlaybook(input), true
	default:
		return EventPlaybook{}, false
	}
}

func isBullishRatesDirection(direction Direction, headline string) bool {
	normalized := Direction(strings.ToLower(strings.TrimSpace(string(direction))))
	return normalized == DirectionHawkishRates || normalized == DirectionInflationHot || normalized == DirectionGrowthStrong || containsAny(headline, "beat", "hot", "strong", "hawkish", "higher for longer")
}

func isDovishRatesDirection(direction Direction, headline string) bool {
	normalized := Direction(strings.ToLower(strings.TrimSpace(string(direction))))
	return normalized == DirectionDovishRates || normalized == DirectionInflationCool || normalized == DirectionGrowthWeak || containsAny(headline, "miss", "cool", "weak", "dovish", "cuts")
}

func hotCPIPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "hot_cpi_rates_hawkish",
		Name:                      "Hot CPI",
		ScenarioKey:               ScenarioHawkishRates,
		CandidateBias:             CandidateBiasShortOrAvoidLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     hawkishConfirmations(input.EventType),
		RequiredTechnicalChecks:   []string{"break below pre-event low", "VWAP rejection", "volume expansion", "relative weakness vs SPY"},
		RequiredFundamentalChecks: []string{"actual CPI above expected", "cross-market rates confirmation", "no stronger same-time confounder"},
		ConfoundersToCheck:        []string{"Fed speaker", "Treasury auction", "same-time labor data", "oil shock"},
		WalkawayRules:             []string{"QQQ reclaims VWAP", "TLT rallies", "print is only marginally hot", "move is too extended"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionDown, "SPY": ReactionDirectionDown, "TLT": ReactionDirectionDown, "IWM": ReactionDirectionDown, "XLK": ReactionDirectionDown, "SMH": ReactionDirectionDown, "SOXX": ReactionDirectionDown},
	}
}

func coolCPIPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "cool_cpi_rates_dovish",
		Name:                      "Cool CPI",
		ScenarioKey:               ScenarioDovishRates,
		CandidateBias:             CandidateBiasLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     []string{string(TimeframePostEvent5M), string(TimeframePostEvent15M)},
		RequiredTechnicalChecks:   []string{"break above pre-event high", "VWAP hold", "volume expansion", "relative strength vs SPY"},
		RequiredFundamentalChecks: []string{"actual CPI below expected", "yields soften", "no dovish reversal confounder"},
		ConfoundersToCheck:        []string{"Fed speaker", "Treasury auction", "same-time labor data", "oil shock"},
		WalkawayRules:             []string{"TLT fails to rally", "QQQ fades below VWAP", "market already priced in the cool print"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionUp, "SPY": ReactionDirectionUp, "TLT": ReactionDirectionUp, "IWM": ReactionDirectionUp, "XLK": ReactionDirectionUp, "SMH": ReactionDirectionUp, "SOXX": ReactionDirectionUp},
	}
}

func hotJobsPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "strong_jobs_hawkish_rates",
		Name:                      "Strong Jobs / Hawkish Rates",
		ScenarioKey:               ScenarioHawkishRates,
		CandidateBias:             CandidateBiasShortOrAvoidLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     hawkishConfirmations(input.EventType),
		RequiredTechnicalChecks:   []string{"break below event range", "VWAP rejection", "volume expansion", "TLT confirmation"},
		RequiredFundamentalChecks: []string{"payrolls materially above expected", "unemployment not offsetting hawkish read", "wage growth not weak"},
		ConfoundersToCheck:        []string{"Fed speaker", "wage revisions", "same-time ISM data", "Treasury auction"},
		WalkawayRules:             []string{"QQQ reclaims event range", "TLT rallies", "wages weak", "unemployment unexpectedly jumps"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionDown, "SPY": ReactionDirectionDown, "TLT": ReactionDirectionDown, "IWM": ReactionDirectionDown, "XLK": ReactionDirectionDown, "SMH": ReactionDirectionDown, "SOXX": ReactionDirectionDown},
	}
}

func weakJobsPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "weak_jobs_dovish_rates",
		Name:                      "Weak Jobs / Dovish Rates",
		ScenarioKey:               ScenarioDovishRates,
		CandidateBias:             CandidateBiasLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     []string{string(TimeframePostEvent5M), string(TimeframePostEvent15M)},
		RequiredTechnicalChecks:   []string{"break above event range", "VWAP hold", "risk-on confirmation"},
		RequiredFundamentalChecks: []string{"payrolls miss expectations", "yields soften", "no recession scare override"},
		ConfoundersToCheck:        []string{"Fed speaker", "credit stress", "same-time data release"},
		WalkawayRules:             []string{"market treats miss as recession scare", "SPY/IWM collapse", "TLT does not confirm dovish move"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionUp, "SPY": ReactionDirectionUp, "TLT": ReactionDirectionUp, "IWM": ReactionDirectionUp, "XLK": ReactionDirectionUp, "SMH": ReactionDirectionUp, "SOXX": ReactionDirectionUp},
	}
}

func hawkishFedPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "hawkish_fed",
		Name:                      "Hawkish Fed",
		ScenarioKey:               ScenarioHawkishRates,
		CandidateBias:             CandidateBiasShortOrAvoidLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     []string{string(TimeframePostEvent15M), string(TimeframePostEvent30M)},
		RequiredTechnicalChecks:   []string{"wait for confirmation", "VWAP rejection", "post-statement hold/fail"},
		RequiredFundamentalChecks: []string{"statement/dot plot/press conference aligned", "no dovish reversal"},
		ConfoundersToCheck:        []string{"Powell reversal", "market already priced the hike", "same-time macro release"},
		WalkawayRules:             []string{"Powell reverses statement reaction", "dot plot and press conference conflict", "TLT/QQQ diverge"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionDown, "SPY": ReactionDirectionDown, "TLT": ReactionDirectionDown, "IWM": ReactionDirectionDown, "XLK": ReactionDirectionDown, "SMH": ReactionDirectionDown, "SOXX": ReactionDirectionDown},
	}
}

func dovishFedPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "dovish_fed",
		Name:                      "Dovish Fed",
		ScenarioKey:               ScenarioDovishRates,
		CandidateBias:             CandidateBiasLong,
		PrimarySymbols:            []string{"QQQ", "SPY", "TLT"},
		SecondarySymbols:          []string{"IWM", "XLK", "SMH", "SOXX"},
		RequiredConfirmations:     []string{string(TimeframePostEvent15M), string(TimeframePostEvent30M)},
		RequiredTechnicalChecks:   []string{"statement confirms lower rates path", "VWAP hold", "post-event trend stabilization"},
		RequiredFundamentalChecks: []string{"statement/dot plot/press conference aligned", "no hawkish reversal"},
		ConfoundersToCheck:        []string{"Powell reversal", "market already priced the cuts", "same-time macro release"},
		WalkawayRules:             []string{"first move reverses", "TLT fails to rally", "QQQ loses post-event range"},
		ExpectedReactions:         map[string]ReactionDirection{"QQQ": ReactionDirectionUp, "SPY": ReactionDirectionUp, "TLT": ReactionDirectionUp, "IWM": ReactionDirectionUp, "XLK": ReactionDirectionUp, "SMH": ReactionDirectionUp, "SOXX": ReactionDirectionUp},
	}
}

func semiconductorPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "mega_cap_ai_semiconductor",
		Name:                      "Mega-Cap AI / Semiconductor",
		ScenarioKey:               ScenarioSemiconductorAI,
		CandidateBias:             CandidateBiasWatchOnly,
		PrimarySymbols:            []string{"SMH", "SOXX", "QQQ"},
		SecondarySymbols:          []string{"SPY"},
		RequiredConfirmations:     []string{string(TimeframePostEvent5M), string(TimeframePostEvent15M)},
		RequiredTechnicalChecks:   []string{"sector confirmation", "relative strength vs SPY", "avoid chasing gap extension"},
		RequiredFundamentalChecks: []string{"new information is actually new", "sector-wide not company-only", "no direct export-control confounder"},
		ConfoundersToCheck:        []string{"rates against growth", "valuation stretched", "competing chip headline"},
		WalkawayRules:             []string{"information is not new", "semis are conflicting", "move is too extended"},
		ExpectedReactions:         map[string]ReactionDirection{"SMH": ReactionDirectionUp, "SOXX": ReactionDirectionUp, "QQQ": ReactionDirectionUp},
	}
}

func bankStressPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "bank_stress",
		Name:                      "Bank Stress",
		ScenarioKey:               ScenarioBankStress,
		CandidateBias:             CandidateBiasRiskOff,
		PrimarySymbols:            []string{"XLF", "SPY", "TLT", "GLD"},
		SecondarySymbols:          []string{"QQQ", "IWM"},
		RequiredConfirmations:     []string{string(TimeframePostEvent15M), string(TimeframePostEvent30M)},
		RequiredTechnicalChecks:   []string{"XLF under pressure", "risk-off follow-through", "no rescue reversal"},
		RequiredFundamentalChecks: []string{"credit stress is real", "not just isolated rumor", "official backstop not reversing the thesis"},
		ConfoundersToCheck:        []string{"regulatory action", "rescue package", "earnings surprise"},
		WalkawayRules:             []string{"official rescue/backstop news", "XLF reclaims VWAP", "stress is isolated to one bank"},
		ExpectedReactions:         map[string]ReactionDirection{"XLF": ReactionDirectionDown, "SPY": ReactionDirectionDown, "TLT": ReactionDirectionUp, "GLD": ReactionDirectionUp},
	}
}

func oilShockPlaybook(input EventInput) EventPlaybook {
	return EventPlaybook{
		Key:                       "oil_shock",
		Name:                      "Oil Shock",
		ScenarioKey:               ScenarioOilShock,
		CandidateBias:             CandidateBiasRiskOff,
		PrimarySymbols:            []string{"XLE", "SPY", "IWM"},
		SecondarySymbols:          []string{"TLT", "GLD"},
		RequiredConfirmations:     []string{string(TimeframePostEvent15M), string(TimeframePostEvent30M)},
		RequiredTechnicalChecks:   []string{"oil move confirmed", "sector follow-through", "avoid first-spike chase"},
		RequiredFundamentalChecks: []string{"supply shock or geopolitics confirmed", "not purely rumor-driven"},
		ConfoundersToCheck:        []string{"geopolitical escalation", "inventory report", "policy intervention"},
		WalkawayRules:             []string{"shock is quickly faded", "cause is not durable", "XLE fails to hold gains"},
		ExpectedReactions:         map[string]ReactionDirection{"XLE": ReactionDirectionUp, "SPY": ReactionDirectionDown, "IWM": ReactionDirectionDown},
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, strings.ToLower(strings.TrimSpace(needle))) {
			return true
		}
	}
	return false
}
