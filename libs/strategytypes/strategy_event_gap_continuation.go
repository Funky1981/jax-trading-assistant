package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

type EventGapContinuation struct{}

func NewEventGapContinuation() *EventGapContinuation { return &EventGapContinuation{} }

func (s *EventGapContinuation) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "event_gap_continuation_v1",
		Name:        "Event Gap Continuation v1",
		Description: "Trades post-event opening gaps that continue through early confirmation.",
		RequiredInputs: RequiredInputs{
			Candles:       []string{"1m", "5m"},
			NeedsEarnings: true,
		},
		Parameters: []ParameterDef{
			{Key: "minGapPct", Type: "float", Default: 1.5, Minimum: ptr(0.5), Maximum: ptr(10)},
			{Key: "confirmationMins", Type: "int", Default: 15, Minimum: ptr(5), Maximum: ptr(90)},
			{Key: "minContinuationPct", Type: "float", Default: 0.3, Minimum: ptr(0.1), Maximum: ptr(5)},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.1, Minimum: ptr(0.5), Maximum: ptr(10)},
		},
	}
}

func (s *EventGapContinuation) Validate(params map[string]any) error {
	minGapPct, err := getFloat(params, "minGapPct", 1.5)
	if err != nil {
		return err
	}
	confirmationMins, err := getInt(params, "confirmationMins", 15)
	if err != nil {
		return err
	}
	minContinuationPct, err := getFloat(params, "minContinuationPct", 0.3)
	if err != nil {
		return err
	}
	minVolumeMultiple, err := getFloat(params, "minVolumeMultiple", 1.1)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("minGapPct", minGapPct, 0.5, 10); err != nil {
		return err
	}
	if err := requireRangeInt("confirmationMins", confirmationMins, 5, 90); err != nil {
		return err
	}
	if err := requireRangeFloat("minContinuationPct", minContinuationPct, 0.1, 5); err != nil {
		return err
	}
	return requireRangeFloat("minVolumeMultiple", minVolumeMultiple, 0.5, 10)
}

func (s *EventGapContinuation) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if len(input.Earnings) == 0 {
		return nil, errors.New("missing required inputs: earnings events")
	}
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, errors.New("missing required inputs: candles")
	}
	minGapPct, _ := getFloat(input.Parameters, "minGapPct", 1.5)
	confirmationMins, _ := getInt(input.Parameters, "confirmationMins", 15)
	minContinuationPct, _ := getFloat(input.Parameters, "minContinuationPct", 0.3)
	minVolumeMultiple, _ := getFloat(input.Parameters, "minVolumeMultiple", 1.1)

	prevClose := input.Earnings[0].PreviousClose
	if prevClose <= 0 {
		return nil, errors.New("missing required inputs: earnings previous close")
	}
	open := candles[0].Open
	gapPct := pctChange(prevClose, open)
	if math.Abs(gapPct) < minGapPct {
		return nil, nil
	}
	direction := "BUY"
	if gapPct < 0 {
		direction = "SELL"
	}

	confirmTime := sessionOpen(candles).Add(time.Duration(confirmationMins) * time.Minute)
	confirmBar, confirmIdx := candleAtOrAfter(candles, confirmTime)
	if confirmIdx < 0 {
		return nil, nil
	}
	continuationPct := pctChange(open, confirmBar.Close)
	if direction == "BUY" && continuationPct < minContinuationPct {
		return nil, nil
	}
	if direction == "SELL" && continuationPct > -minContinuationPct {
		return nil, nil
	}

	volBaseline := avgVolume(candles[:confirmIdx+1], minInt(confirmIdx+1, 20))
	if volBaseline > 0 && (confirmBar.Volume/volBaseline) < minVolumeMultiple {
		return nil, nil
	}

	reason := fmt.Sprintf("gap %.2f%% continuation %.2f%%", math.Abs(gapPct), math.Abs(continuationPct))
	return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, direction, reason, confirmBar)}, nil
}
