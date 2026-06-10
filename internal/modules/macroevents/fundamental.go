package macroevents

import (
	"math"
	"strings"
	"time"
)

type FundamentalVerdict string

type FundamentalCheck struct {
	Symbol    string `json:"symbol"`
	Expected  string `json:"expected"`
	Observed  string `json:"observed"`
	Confirmed bool   `json:"confirmed"`
	Reason    string `json:"reason"`
}

const (
	FundamentalVerdictStrongBullish    FundamentalVerdict = "strong_bullish"
	FundamentalVerdictModerateBullish  FundamentalVerdict = "moderate_bullish"
	FundamentalVerdictNeutral          FundamentalVerdict = "neutral"
	FundamentalVerdictModerateBearish  FundamentalVerdict = "moderate_bearish"
	FundamentalVerdictStrongBearish    FundamentalVerdict = "strong_bearish"
	FundamentalVerdictConflicted       FundamentalVerdict = "conflicted"
	FundamentalVerdictInsufficientData FundamentalVerdict = "insufficient_data"
)

type FundamentalInput struct {
	MacroEventID      string
	Symbol            string
	AnalysisTimeUTC   time.Time
	Event             EventInput
	Scenario          ScenarioEvaluation
	EventSummary      string
	ExpectedImpact    string
	AffectedThemes    []string
	CrossMarketChecks []FundamentalCheck
	Confounders       []Confounder
	MissingEvidence   []string
}

type FundamentalSnapshot struct {
	ID                   string
	MacroEventID         string
	Symbol               string
	AnalysisTimeUTC      time.Time
	EventSummary         string
	ExpectedMarketImpact string
	AffectedThemes       []string
	CrossMarketChecks    []FundamentalCheck
	Confounders          []Confounder
	FundamentalScore     float64
	Verdict              FundamentalVerdict
	Reasons              []string
	MissingEvidence      []string
}

func EvaluateFundamentalSnapshot(input FundamentalInput) FundamentalSnapshot {
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	if symbol == "" && len(input.Event.AffectedETFs) > 0 {
		symbol = strings.ToUpper(strings.TrimSpace(input.Event.AffectedETFs[0]))
	}
	snapshot := FundamentalSnapshot{
		MacroEventID:         strings.TrimSpace(input.MacroEventID),
		Symbol:               symbol,
		AnalysisTimeUTC:      analysisTime(input.AnalysisTimeUTC),
		EventSummary:         strings.TrimSpace(input.EventSummary),
		ExpectedMarketImpact: strings.TrimSpace(input.ExpectedImpact),
		AffectedThemes:       normalizeStrings(input.AffectedThemes),
		CrossMarketChecks:    cloneFundamentalChecks(input.CrossMarketChecks),
		Confounders:          cloneConfounders(input.Confounders),
		MissingEvidence:      normalizeStrings(input.MissingEvidence),
		Reasons:              []string{},
	}

	if snapshot.EventSummary == "" {
		snapshot.EventSummary = strings.TrimSpace(input.Event.Headline)
	}
	if snapshot.ExpectedMarketImpact == "" {
		snapshot.ExpectedMarketImpact = expectedImpactFromEvent(input.Event, input.Scenario, symbol)
	}
	if len(snapshot.AffectedThemes) == 0 {
		snapshot.AffectedThemes = []string{themeForSymbol(symbol)}
	}

	if strings.TrimSpace(string(input.Event.EventType)) == "" || !supportedEventType(input.Event.EventType) {
		snapshot.Verdict = FundamentalVerdictInsufficientData
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "supported event type")
		snapshot.Reasons = append(snapshot.Reasons, "event type is unsupported")
		return snapshot
	}
	if numericReleaseRequiresValues(input.Event.EventType) && (input.Event.ActualValue == nil || input.Event.ExpectedValue == nil) {
		snapshot.Verdict = FundamentalVerdictInsufficientData
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "actual vs expected")
		snapshot.Reasons = append(snapshot.Reasons, "missing actual versus expected release data")
		return snapshot
	}
	if snapshot.ExpectedMarketImpact == "" {
		snapshot.Verdict = FundamentalVerdictInsufficientData
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "expected market impact")
		snapshot.Reasons = append(snapshot.Reasons, "expected market impact is unclear")
		return snapshot
	}
	if len(snapshot.AffectedThemes) == 0 {
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "affected themes")
	}

	surprise := fundamentalSurprise(input.Event.ActualValue, input.Event.ExpectedValue)
	confidence := clampConfidence(input.Event.Confidence)
	score := 0.0

	if surprise >= 0.5 {
		score += 25
		snapshot.Reasons = append(snapshot.Reasons, "release surprise is large enough to matter")
	} else if surprise >= 0.2 {
		score += 15
		snapshot.Reasons = append(snapshot.Reasons, "release surprise is meaningful")
	} else {
		score += 6
		snapshot.Reasons = append(snapshot.Reasons, "release surprise is modest")
	}

	if strings.Contains(strings.ToLower(snapshot.ExpectedMarketImpact), "hawkish") || strings.Contains(strings.ToLower(snapshot.ExpectedMarketImpact), "dovish") {
		score += 20
	} else if strings.Contains(strings.ToLower(snapshot.ExpectedMarketImpact), "bearish") || strings.Contains(strings.ToLower(snapshot.ExpectedMarketImpact), "bullish") {
		score += 15
	} else {
		score += 8
	}

	score += 15 * themeRelevanceScore(snapshot.AffectedThemes, symbol)
	score += 15 * crossMarketConfirmationScore(snapshot.CrossMarketChecks)
	score += 10 * confounderCleanlinessScore(snapshot.Confounders)
	score += 5 * confidence

	if scenarioBlocksFundamental(input.Scenario) {
		snapshot.Verdict = FundamentalVerdictConflicted
		snapshot.Reasons = append(snapshot.Reasons, input.Scenario.Reason)
		snapshot.FundamentalScore = math.Round(score*100) / 100
		return snapshot
	}
	if unresolvedConfounder(snapshot.Confounders) {
		snapshot.Verdict = FundamentalVerdictConflicted
		snapshot.Reasons = append(snapshot.Reasons, "major unresolved confounder blocks thesis")
		snapshot.FundamentalScore = math.Round(score*100) / 100
		return snapshot
	}
	if len(snapshot.CrossMarketChecks) > 0 && allCrossMarketChecksOppose(snapshot.CrossMarketChecks) {
		snapshot.Verdict = FundamentalVerdictConflicted
		snapshot.Reasons = append(snapshot.Reasons, "cross-market checks contradict thesis")
		snapshot.FundamentalScore = math.Round(score*100) / 100
		return snapshot
	}

	if confidence < 0.5 {
		snapshot.Verdict = FundamentalVerdictInsufficientData
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "source quality")
		snapshot.Reasons = append(snapshot.Reasons, "source quality too low")
		snapshot.FundamentalScore = math.Round(score*100) / 100
		return snapshot
	}

	if score >= 70 {
		snapshot.Verdict = verdictFromFundamentalScore(score, input.Event, symbol)
	} else if score >= 60 {
		snapshot.Verdict = FundamentalVerdictNeutral
	} else {
		snapshot.Verdict = FundamentalVerdictModerateBearish
	}

	if shouldFlagSymbolMismatch(symbol, input.Event.AffectedETFs) {
		snapshot.MissingEvidence = appendUnique(snapshot.MissingEvidence, "ETF relevance")
	}

	snapshot.FundamentalScore = math.Round(score*100) / 100
	if snapshot.ExpectedMarketImpact == "" {
		snapshot.ExpectedMarketImpact = "headline is one input, not the whole thesis"
	}
	return snapshot
}

func fundamentalSurprise(actual, expected *float64) float64 {
	if actual == nil || expected == nil || *expected == 0 {
		return 0
	}
	return math.Abs((*actual - *expected) / *expected)
}

func scenarioBlocksFundamental(s ScenarioEvaluation) bool {
	switch s.Result {
	case ScenarioResultBlockedUnknownEvent, ScenarioResultBlockedConflicting, ScenarioResultBlockedNoETFMapping, ScenarioResultBlockedDisallowedInstrument:
		return true
	default:
		return false
	}
}

func unresolvedConfounder(confounders []Confounder) bool {
	for _, confounder := range confounders {
		if confounder.BlocksCandidate {
			return true
		}
	}
	return false
}

func allCrossMarketChecksOppose(checks []FundamentalCheck) bool {
	if len(checks) == 0 {
		return false
	}
	for _, check := range checks {
		if check.Confirmed {
			return false
		}
	}
	return true
}

func themeRelevanceScore(themes []string, symbol string) float64 {
	if len(themes) == 0 {
		return 0.3
	}
	for _, theme := range themes {
		if strings.TrimSpace(theme) != "" {
			return 1
		}
	}
	return 0.3
}

func crossMarketConfirmationScore(checks []FundamentalCheck) float64 {
	if len(checks) == 0 {
		return 0.4
	}
	confirmed := 0
	for _, check := range checks {
		if check.Confirmed {
			confirmed++
		}
	}
	return float64(confirmed) / float64(len(checks))
}

func confounderCleanlinessScore(confounders []Confounder) float64 {
	if len(confounders) == 0 {
		return 1
	}
	for _, confounder := range confounders {
		if confounder.BlocksCandidate {
			return 0
		}
	}
	return 0.6
}

func expectedImpactFromEvent(event EventInput, scenario ScenarioEvaluation, symbol string) string {
	direction := strings.ToLower(strings.TrimSpace(string(event.Direction)))
	switch direction {
	case string(DirectionInflationHot), string(DirectionHawkishRates), string(DirectionGrowthStrong):
		return "hawkish rates pressure growth-sensitive ETFs"
	case string(DirectionInflationCool), string(DirectionDovishRates), string(DirectionGrowthWeak):
		return "dovish rates support duration and growth"
	default:
		if scenario.ScenarioKey != ScenarioUnknown {
			return string(scenario.ScenarioKey) + " scenario affects mapped ETF basket"
		}
		return "headline may affect the mapped ETF basket"
	}
}

func verdictFromFundamentalScore(score float64, event EventInput, symbol string) FundamentalVerdict {
	direction := strings.ToLower(strings.TrimSpace(string(event.Direction)))
	switch direction {
	case string(DirectionInflationHot), string(DirectionHawkishRates), string(DirectionGrowthStrong):
		return FundamentalVerdictStrongBearish
	case string(DirectionInflationCool), string(DirectionDovishRates), string(DirectionGrowthWeak):
		return FundamentalVerdictStrongBullish
	default:
		if score >= 90 {
			return FundamentalVerdictStrongBullish
		}
		return FundamentalVerdictNeutral
	}
}

func shouldFlagSymbolMismatch(symbol string, affected []string) bool {
	if symbol == "" || len(affected) == 0 {
		return true
	}
	upper := strings.ToUpper(strings.TrimSpace(symbol))
	for _, item := range affected {
		if strings.ToUpper(strings.TrimSpace(item)) == upper {
			return false
		}
	}
	return true
}

func cloneFundamentalChecks(checks []FundamentalCheck) []FundamentalCheck {
	if len(checks) == 0 {
		return nil
	}
	out := make([]FundamentalCheck, len(checks))
	copy(out, checks)
	return out
}

func cloneConfounders(confounders []Confounder) []Confounder {
	if len(confounders) == 0 {
		return nil
	}
	out := make([]Confounder, len(confounders))
	copy(out, confounders)
	return out
}

func appendUnique(values []string, additions ...string) []string {
	index := make(map[string]struct{}, len(values))
	for _, value := range values {
		index[value] = struct{}{}
	}
	for _, addition := range additions {
		if addition == "" {
			continue
		}
		if _, exists := index[addition]; exists {
			continue
		}
		values = append(values, addition)
		index[addition] = struct{}{}
	}
	return values
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
