package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"math"
)

type PairsEventRelative struct{}

func NewPairsEventRelative() *PairsEventRelative { return &PairsEventRelative{} }

func (s *PairsEventRelative) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "pairs_event_relative_v1",
		Name:        "Pairs Event Relative v1",
		Description: "Trades event-driven relative strength against a peer symbol (research-first).",
		RequiredInputs: RequiredInputs{
			Candles:   []string{"1m", "5m"},
			NeedsNews: true,
		},
		Parameters: []ParameterDef{
			{Key: "relativeStrengthPct", Type: "float", Default: 0, Minimum: ptr(-10), Maximum: ptr(10)},
			{Key: "minRelativeStrengthPct", Type: "float", Default: 1.0, Minimum: ptr(0.5), Maximum: ptr(10)},
		},
	}
}

func (s *PairsEventRelative) Validate(params map[string]any) error {
	relativeStrengthPct, err := getFloat(params, "relativeStrengthPct", 0)
	if err != nil {
		return err
	}
	minRelativeStrengthPct, err := getFloat(params, "minRelativeStrengthPct", 1.0)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("relativeStrengthPct", relativeStrengthPct, -10, 10); err != nil {
		return err
	}
	return requireRangeFloat("minRelativeStrengthPct", minRelativeStrengthPct, 0.5, 10)
}

func (s *PairsEventRelative) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if len(input.News) == 0 && len(input.Earnings) == 0 {
		return nil, errors.New("missing required inputs: event context")
	}
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	relativeStrengthPct, _ := getFloat(input.Parameters, "relativeStrengthPct", 0)
	minRelativeStrengthPct, _ := getFloat(input.Parameters, "minRelativeStrengthPct", 1.0)
	if math.Abs(relativeStrengthPct) < minRelativeStrengthPct {
		return nil, nil
	}
	direction := "BUY"
	if relativeStrengthPct < 0 {
		direction = "SELL"
	}
	bar := candles[len(candles)-1]
	reason := fmt.Sprintf("relative strength %.2f%%", relativeStrengthPct)
	return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, direction, reason, bar)}, nil
}
