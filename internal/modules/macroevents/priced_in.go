package macroevents

import (
	"math"
	"strings"
)

type PricedInVerdict string

const (
	PricedInVerdictNotPricedIn       PricedInVerdict = "not_priced_in"
	PricedInVerdictPartiallyPricedIn PricedInVerdict = "partially_priced_in"
	PricedInVerdictPricedIn          PricedInVerdict = "priced_in"
	PricedInVerdictOverreaction      PricedInVerdict = "overreaction"
	PricedInVerdictUnclear           PricedInVerdict = "unclear"
)

type PricedInInput struct {
	MacroEventID          string
	Symbol                string
	Event                 EventInput
	PreEventMovePercent   float64
	NewsSaturationScore   float64
	VolatilityElevated    bool
	AnalystConsensusTight bool
	Reaction              ReactionSnapshot
}

type PricedInScore struct {
	ID              string
	MacroEventID    string
	Symbol          string
	Verdict         PricedInVerdict
	Score           float64
	Reasons         []string
	BlocksCandidate bool
}

type ConfounderInput struct {
	Type     string
	Headline string
	Source   string
	Severity string
	Reason   string
}

type Confounder struct {
	ID              string
	MacroEventID    string
	Type            string
	Headline        string
	Source          string
	Severity        string
	Reason          string
	BlocksCandidate bool
}

func ScorePricedIn(input PricedInInput) PricedInScore {
	score := PricedInScore{
		MacroEventID: input.MacroEventID,
		Symbol:       strings.ToUpper(strings.TrimSpace(input.Symbol)),
		Score:        0.5,
		Verdict:      PricedInVerdictUnclear,
		Reasons:      []string{},
	}
	if input.Reaction.Status != ReactionStatusAvailable || input.Event.ActualValue == nil || input.Event.ExpectedValue == nil {
		score.Reasons = append(score.Reasons, "priced-in verdict unclear because required event or reaction data is missing")
		score.BlocksCandidate = true
		return score
	}
	if input.Reaction.TooExtended {
		score.Verdict = PricedInVerdictOverreaction
		score.Score = 0.8
		score.Reasons = append(score.Reasons, "post-event reaction is too extended for an immediate candidate")
		score.BlocksCandidate = true
		return score
	}

	surprise := surpriseRatio(input.Event.ActualValue, input.Event.ExpectedValue)
	preMove := math.Abs(input.PreEventMovePercent)
	switch {
	case surprise >= 0.5 && preMove < 0.005 && input.Reaction.ConfirmsEvent:
		score.Verdict = PricedInVerdictNotPricedIn
		score.Score = 0.15
		score.Reasons = append(score.Reasons, "large macro surprise with small pre-event drift and confirming reaction")
	case surprise < 0.05 && preMove >= 0.015:
		score.Verdict = PricedInVerdictPricedIn
		score.Score = 0.9
		score.Reasons = append(score.Reasons, "small surprise and large pre-event move indicate market was positioned")
	case input.NewsSaturationScore >= 0.7 || input.VolatilityElevated || input.AnalystConsensusTight:
		score.Verdict = PricedInVerdictPartiallyPricedIn
		score.Score = 0.6
		score.Reasons = append(score.Reasons, "pre-release saturation or consensus clustering suggests partial pricing")
	default:
		score.Verdict = PricedInVerdictUnclear
		score.Score = 0.5
		score.Reasons = append(score.Reasons, "priced-in verdict unclear because surprise and reaction evidence are mixed")
	}
	score.BlocksCandidate = score.Verdict == PricedInVerdictPricedIn || score.Verdict == PricedInVerdictUnclear || score.Verdict == PricedInVerdictOverreaction
	return score
}

func surpriseRatio(actual, expected *float64) float64 {
	if actual == nil || expected == nil || *expected == 0 {
		return 0
	}
	return math.Abs((*actual - *expected) / *expected)
}

func DetectConfounders(inputs []ConfounderInput) []Confounder {
	out := make([]Confounder, 0, len(inputs))
	for _, input := range inputs {
		severity := strings.ToLower(strings.TrimSpace(input.Severity))
		if severity == "" {
			severity = "medium"
		}
		confounder := Confounder{
			Type:            strings.TrimSpace(input.Type),
			Headline:        strings.TrimSpace(input.Headline),
			Source:          strings.TrimSpace(input.Source),
			Severity:        severity,
			Reason:          strings.TrimSpace(input.Reason),
			BlocksCandidate: severity == "high" || severity == "critical",
		}
		if confounder.Type == "" || confounder.Headline == "" {
			continue
		}
		if confounder.Reason == "" {
			confounder.Reason = "confounder may explain macro reaction"
		}
		out = append(out, confounder)
	}
	return out
}
