package strategytypes

import (
	"context"
	"fmt"
	"math"
	"strings"
)

type IndexFlow struct{}

func NewIndexFlow() *IndexFlow { return &IndexFlow{} }

func (s *IndexFlow) Metadata() StrategyMetadata {
	return StrategyMetadata{
		StrategyID:  "index_flow_v1",
		Name:        "Index Flow v1",
		Description: "Trades SPY/QQQ trend-day continuation on deterministic pullback thresholds.",
		RequiredInputs: RequiredInputs{
			Candles: []string{"1m", "5m"},
		},
		Parameters: []ParameterDef{
			{Key: "trendStrengthPct", Type: "float", Default: 1.0, Minimum: ptr(0.2), Maximum: ptr(8)},
			{Key: "pullbackPct", Type: "float", Default: 0.4, Minimum: ptr(0.1), Maximum: ptr(3)},
		},
	}
}

func (s *IndexFlow) Validate(params map[string]any) error {
	trendStrengthPct, err := getFloat(params, "trendStrengthPct", 1.0)
	if err != nil {
		return err
	}
	pullbackPct, err := getFloat(params, "pullbackPct", 0.4)
	if err != nil {
		return err
	}
	if err := requireRangeFloat("trendStrengthPct", trendStrengthPct, 0.2, 8); err != nil {
		return err
	}
	return requireRangeFloat("pullbackPct", pullbackPct, 0.1, 3)
}

func (s *IndexFlow) Generate(_ context.Context, input StrategyInput) ([]Signal, error) {
	if err := s.Validate(input.Parameters); err != nil {
		return nil, err
	}
	symbol := strings.ToUpper(input.Symbol)
	if symbol != "SPY" && symbol != "QQQ" {
		return nil, fmt.Errorf("index_flow_v1 only supports SPY/QQQ, got %s", input.Symbol)
	}
	candles, _, err := normalizeCandles(input, "1m", "5m")
	if err != nil {
		return nil, err
	}
	trendStrengthPct, _ := getFloat(input.Parameters, "trendStrengthPct", 1.0)
	pullbackPct, _ := getFloat(input.Parameters, "pullbackPct", 0.4)

	open := candles[0].Open
	last := candles[len(candles)-1].Close
	dayTrendPct := pctChange(open, last)
	dayHigh := candles[0].High
	dayLow := candles[0].Low
	for _, c := range candles {
		if c.High > dayHigh {
			dayHigh = c.High
		}
		if c.Low < dayLow {
			dayLow = c.Low
		}
	}
	rangePct := math.Abs(pctChange(dayLow, dayHigh))
	if math.Abs(dayTrendPct) < trendStrengthPct || rangePct < trendStrengthPct {
		return nil, nil
	}
	if dayTrendPct > 0 {
		peak := open
		for i := range candles {
			if candles[i].High > peak {
				peak = candles[i].High
				continue
			}
			retrace := pctChange(peak, candles[i].Low) * -1
			if retrace >= pullbackPct && i+1 < len(candles) && candles[i+1].Close > candles[i].Close {
				return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "BUY",
					fmt.Sprintf("trend continuation after %.2f%% pullback", retrace), candles[i+1])}, nil
			}
		}
		return nil, nil
	}
	trough := open
	for i := range candles {
		if candles[i].Low < trough {
			trough = candles[i].Low
			continue
		}
		bounce := pctChange(trough, candles[i].High)
		if bounce >= pullbackPct && i+1 < len(candles) && candles[i+1].Close < candles[i].Close {
			return []Signal{buildSignal(s.Metadata().StrategyID, input.Symbol, "SELL",
				fmt.Sprintf("downtrend continuation after %.2f%% bounce", bounce), candles[i+1])}, nil
		}
	}
	return nil, nil
}
