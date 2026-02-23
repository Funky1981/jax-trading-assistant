package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type OpeningRangeToClose struct{}

func NewOpeningRangeToClose() *OpeningRangeToClose { return &OpeningRangeToClose{} }

func (s *OpeningRangeToClose) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "opening_range_to_close_v1",
		Name:        "Opening Range to Close v1",
		Description: "Enters after opening range breakout + hold confirmation with volume filter.",
		RequiredInputs: RequiredInputs{
			Candles: []string{"1m", "5m"},
		},
		Parameters: []ParameterDef{
			{Key: "openingRangeMins", Type: "int", Default: 30, Minimum: ptr(5), Maximum: ptr(90)},
			{Key: "holdBars", Type: "int", Default: 3, Minimum: ptr(1), Maximum: ptr(10)},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.2, Minimum: ptr(0.5), Maximum: ptr(10)},
		},
	}
}

func (s *OpeningRangeToClose) Validate(params map[string]any) error {
	openingRangeMins, err := getInt(params, "openingRangeMins", 30)
	if err != nil {
		return err
	}
	holdBars, err := getInt(params, "holdBars", 3)
	if err != nil {
		return err
	}
	minVolumeMultiple, err := getFloat(params, "minVolumeMultiple", 1.2)
	if err != nil {
		return err
	}
	if err := requireRangeInt("openingRangeMins", openingRangeMins, 5, 90); err != nil {
		return err
	}
	if err := requireRangeInt("holdBars", holdBars, 1, 10); err != nil {
		return err
	}
	return requireRangeFloat("minVolumeMultiple", minVolumeMultiple, 0.5, 10)
}

func (s *OpeningRangeToClose) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	openingRangeMins, _ := getInt(input.Parameters, "openingRangeMins", 30)
	holdBars, _ := getInt(input.Parameters, "holdBars", 3)
	minVolumeMultiple, _ := getFloat(input.Parameters, "minVolumeMultiple", 1.2)

	sessionStart := sessionOpen(candles)
	openEnd := sessionStart.Add(time.Duration(openingRangeMins) * time.Minute)
	openingRange := make([]Candle, 0, 64)
	for _, c := range candles {
		if c.Timestamp.Before(openEnd) {
			openingRange = append(openingRange, c)
		}
	}
	if len(openingRange) < 2 {
		return nil, errors.New("missing required inputs: opening range candles")
	}
	rangeHigh := openingRange[0].High
	rangeLow := openingRange[0].Low
	for _, c := range openingRange {
		if c.High > rangeHigh {
			rangeHigh = c.High
		}
		if c.Low < rangeLow {
			rangeLow = c.Low
		}
	}
	volBaseline := avgVolume(openingRange, len(openingRange))

	for i := len(openingRange); i+holdBars < len(candles); i++ {
		cur := candles[i]
		volMult := 0.0
		if volBaseline > 0 {
			volMult = cur.Volume / volBaseline
		}
		if volMult < minVolumeMultiple {
			continue
		}
		if cur.Close > rangeHigh {
			holds := true
			for j := 1; j <= holdBars; j++ {
				if candles[i+j].Close <= rangeHigh {
					holds = false
					break
				}
			}
			if holds {
				return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "BUY",
					fmt.Sprintf("opening range breakout hold=%d bars", holdBars), cur)}, nil
			}
		}
		if cur.Close < rangeLow {
			holds := true
			for j := 1; j <= holdBars; j++ {
				if candles[i+j].Close >= rangeLow {
					holds = false
					break
				}
			}
			if holds {
				return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "SELL",
					fmt.Sprintf("opening range breakdown hold=%d bars", holdBars), cur)}, nil
			}
		}
	}
	return nil, nil
}
