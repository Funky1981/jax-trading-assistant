package strategytypes

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

func ptr(v float64) *float64 { return &v }

func getInt(params map[string]any, key string, def int) (int, error) {
	if params == nil {
		return def, nil
	}
	raw, ok := params[key]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("%s must be numeric", key)
	}
}

func getFloat(params map[string]any, key string, def float64) (float64, error) {
	if params == nil {
		return def, nil
	}
	raw, ok := params[key]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("%s must be numeric", key)
	}
}

func requireRangeInt(name string, v, min, max int) error {
	if v < min || v > max {
		return fmt.Errorf("%s out of range [%d,%d]: %d", name, min, max, v)
	}
	return nil
}

func requireRangeFloat(name string, v, min, max float64) error {
	if v < min || v > max {
		return fmt.Errorf("%s out of range [%.2f,%.2f]: %.4f", name, min, max, v)
	}
	return nil
}

func normalizeCandles(input StrategyInput, accepted ...string) ([]Candle, string, error) {
	if len(input.Candles) == 0 {
		return nil, "", errors.New("missing required inputs: intraday candles")
	}
	for _, tf := range accepted {
		if candles := input.Candles[tf]; len(candles) > 0 {
			sorted := slices.Clone(candles)
			slices.SortFunc(sorted, func(a, b Candle) int {
				return a.Timestamp.Compare(b.Timestamp)
			})
			return sorted, tf, nil
		}
	}
	return nil, "", fmt.Errorf("missing required inputs: candles timeframe %v", accepted)
}

func sessionOpen(candles []Candle) time.Time {
	return candles[0].Timestamp
}

func barAtOrAfter(candles []Candle, ts time.Time) (Candle, bool) {
	for _, c := range candles {
		if !c.Timestamp.Before(ts) {
			return c, true
		}
	}
	return Candle{}, false
}

func avgVolume(candles []Candle, n int) float64 {
	if len(candles) == 0 {
		return 0
	}
	if n <= 0 || n > len(candles) {
		n = len(candles)
	}
	total := 0.0
	for i := 0; i < n; i++ {
		total += candles[i].Volume
	}
	return total / float64(n)
}

func intradayVWAP(candles []Candle, upto int) float64 {
	if len(candles) == 0 {
		return 0
	}
	if upto <= 0 || upto > len(candles) {
		upto = len(candles)
	}
	sumPV := 0.0
	sumV := 0.0
	for i := 0; i < upto; i++ {
		p := (candles[i].High + candles[i].Low + candles[i].Close) / 3.0
		sumPV += p * candles[i].Volume
		sumV += candles[i].Volume
	}
	if sumV == 0 {
		return candles[upto-1].Close
	}
	return sumPV / sumV
}

func buildSignal(strategyID, symbol, direction, reason string, bar Candle) Signal {
	stop := bar.Close * 0.995
	target := bar.Close * 1.01
	if strings.EqualFold(direction, "SELL") {
		stop = bar.Close * 1.005
		target = bar.Close * 0.99
	}
	return Signal{
		Symbol:      symbol,
		StrategyID:  strategyID,
		Direction:   strings.ToUpper(direction),
		EntryPrice:  bar.Close,
		StopLoss:    stop,
		TakeProfit:  target,
		GeneratedAt: bar.Timestamp.UTC(),
		Reason:      reason,
	}
}

func pctChange(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return ((to - from) / math.Abs(from)) * 100.0
}
