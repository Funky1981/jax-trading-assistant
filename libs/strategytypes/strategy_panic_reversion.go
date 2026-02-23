package strategytypes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

type PanicReversion struct{}

func NewPanicReversion() *PanicReversion { return &PanicReversion{} }

func (s *PanicReversion) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "panic_reversion_v1",
		Name:        "Panic Reversion v1",
		Description: "Looks for intraday panic drop and mean-reversion when price stabilizes near VWAP.",
		RequiredInputs: RequiredInputs{
			Candles: []string{"1m", "5m"},
		},
		Parameters: []ParameterDef{
			{Key: "dropPctThreshold", Type: "float", Default: 2.5, Minimum: ptr(0.5), Maximum: ptr(20)},
			{Key: "vwapDistancePct", Type: "float", Default: 0.25, Minimum: ptr(0.05), Maximum: ptr(2)},
			{Key: "minTimeAfterOpenMins", Type: "int", Default: 60, Minimum: ptr(10), Maximum: ptr(180)},
		},
	}
}

func (s *PanicReversion) Validate(params map[string]any) error {
	dropPctThreshold, err := getFloat(params, "dropPctThreshold", 2.5)
	if err != nil {
		return err
	}
	vwapDistancePct, err := getFloat(params, "vwapDistancePct", 0.25)
	if err != nil {
		return err
	}
	minTimeAfterOpenMins, err := getInt(params, "minTimeAfterOpenMins", 60)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("dropPctThreshold", dropPctThreshold, 0.5, 20); err != nil {
		return err
	}
	if err := requireRangeFloat("vwapDistancePct", vwapDistancePct, 0.05, 2); err != nil {
		return err
	}
	return requireRangeInt("minTimeAfterOpenMins", minTimeAfterOpenMins, 10, 180)
}

func (s *PanicReversion) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	dropPctThreshold, _ := getFloat(input.Parameters, "dropPctThreshold", 2.5)
	vwapDistancePct, _ := getFloat(input.Parameters, "vwapDistancePct", 0.25)
	minTimeAfterOpenMins, _ := getInt(input.Parameters, "minTimeAfterOpenMins", 60)

	open := candles[0].Open
	openTS := candles[0].Timestamp
	startTS := openTS.Add(time.Duration(minTimeAfterOpenMins) * time.Minute)
	panicIdx := -1
	for i := range candles {
		if candles[i].Timestamp.Before(startTS) {
			continue
		}
		dropPct := pctChange(open, candles[i].Low) * -1
		if dropPct >= dropPctThreshold {
			panicIdx = i
			break
		}
	}
	if panicIdx == -1 {
		return nil, nil
	}
	for i := panicIdx + 1; i < len(candles); i++ {
		vwap := intradayVWAP(candles, i+1)
		if vwap == 0 {
			return nil, errors.New("missing required inputs: non-zero vwap")
		}
		dist := math.Abs(pctChange(vwap, candles[i].Close))
		if dist <= vwapDistancePct && candles[i].Close >= vwap {
			return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "BUY",
				fmt.Sprintf("panic reversion: reclaim near VWAP after %.2f%% drop", dropPctThreshold), candles[i])}, nil
		}
	}
	return nil, nil
}
