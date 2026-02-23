package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SameDayEarningsDrift struct{}

func NewSameDayEarningsDrift() *SameDayEarningsDrift { return &SameDayEarningsDrift{} }

func (s *SameDayEarningsDrift) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "same_day_earnings_drift_v1",
		Name:        "Same-day Earnings Drift v1",
		Description: "Follows post-earnings drift after open once gap and volume filters confirm direction.",
		RequiredInputs: RequiredInputs{
			Candles:       []string{"1m", "5m"},
			NeedsEarnings: true,
		},
		Parameters: []ParameterDef{
			{Key: "entryDelayMins", Type: "int", Default: 30, Minimum: ptr(5), Maximum: ptr(120)},
			{Key: "minGapPct", Type: "float", Default: 2.0, Minimum: ptr(0), Maximum: ptr(20)},
			{Key: "minVolumeMultiple", Type: "float", Default: 1.5, Minimum: ptr(0), Maximum: ptr(10)},
		},
	}
}

func (s *SameDayEarningsDrift) Validate(params map[string]any) error {
	entryDelayMins, err := getInt(params, "entryDelayMins", 30)
	if err != nil {
		return err
	}
	minGapPct, err := getFloat(params, "minGapPct", 2.0)
	if err != nil {
		return err
	}
	minVolumeMultiple, err := getFloat(params, "minVolumeMultiple", 1.5)
	if err != nil {
		return err
	}
	if err := requireRangeInt("entryDelayMins", entryDelayMins, 5, 120); err != nil {
		return err
	}
	if err := requireRangeFloat("minGapPct", minGapPct, 0, 20); err != nil {
		return err
	}
	return requireRangeFloat("minVolumeMultiple", minVolumeMultiple, 0, 10)
}

func (s *SameDayEarningsDrift) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if len(input.Earnings) == 0 {
		return nil, errors.New("missing required inputs: earnings events")
	}
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, tf, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	entryDelayMins, _ := getInt(input.Parameters, "entryDelayMins", 30)
	minGapPct, _ := getFloat(input.Parameters, "minGapPct", 2.0)
	minVolumeMultiple, _ := getFloat(input.Parameters, "minVolumeMultiple", 1.5)

	openTS := sessionOpen(candles)
	entryTS := openTS.Add(time.Duration(entryDelayMins) * time.Minute)
	entryBar, ok := barAtOrAfter(candles, entryTS)
	if !ok {
		return nil, fmt.Errorf("missing required inputs: %s candles beyond entry window", tf)
	}

	event := input.Earnings[0]
	gapBase := event.PreviousClose
	if gapBase == 0 {
		gapBase = candles[0].Open
	}
	gapPct := pctChange(gapBase, entryBar.Close)
	volBaseline := avgVolume(candles, 20)
	if volBaseline == 0 {
		return nil, errors.New("missing required inputs: non-zero candle volume")
	}
	volMult := entryBar.Volume / volBaseline

	guidance := strings.ToLower(event.Guidance)
	if gapPct >= minGapPct && volMult >= minVolumeMultiple && event.SurprisePct > 0 && strings.Contains(guidance, "positive") {
		return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "BUY",
			fmt.Sprintf("earnings drift long gap=%.2f%% vol=%.2fx", gapPct, volMult), entryBar)}, nil
	}
	if gapPct <= -minGapPct && volMult >= minVolumeMultiple && event.SurprisePct < 0 && strings.Contains(guidance, "negative") {
		return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "SELL",
			fmt.Sprintf("earnings drift short gap=%.2f%% vol=%.2fx", gapPct, volMult), entryBar)}, nil
	}
	return nil, nil
}
