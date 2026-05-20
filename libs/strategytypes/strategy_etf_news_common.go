package strategytypes

import (
	"math"
	"strings"
)

// etfLatestCandle returns the last candle in the slice or (Candle{}, false).
func etfLatestCandle(candles []Candle) (Candle, bool) {
	if len(candles) == 0 {
		return Candle{}, false
	}
	return candles[len(candles)-1], true
}

// etfConfirmedNews returns the news items that match all supplied tags.
// etfConfirmedNews returns news items matching the given category and sentiment keywords.
// Pass an empty category to match any. Pass empty sentiments to match any.
func etfConfirmedNews(items []NewsEvent, category string, sentiments ...string) []NewsEvent {
	var out []NewsEvent
	for _, n := range items {
		if category != "" && !strings.EqualFold(n.Category, category) {
			continue
		}
		if len(sentiments) > 0 {
			matched := false
			for _, s := range sentiments {
				if strings.EqualFold(n.Sentiment, s) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

// etfVolumeMultiple returns how many times the latest volume exceeds the average.
func etfVolumeMultiple(candles []Candle) float64 {
	if len(candles) < 2 {
		return 1.0
	}
	avg := avgVolume(candles[:len(candles)-1], len(candles)-1)
	if avg == 0 {
		return 1.0
	}
	return candles[len(candles)-1].Volume / avg
}

// etfATR computes the Average True Range over the last n bars.
func etfATR(candles []Candle, n int) float64 {
	if len(candles) < 2 {
		return 0
	}
	if n <= 0 || n >= len(candles) {
		n = len(candles) - 1
	}
	sum := 0.0
	for i := len(candles) - n; i < len(candles); i++ {
		prev := candles[i-1].Close
		tr := math.Max(candles[i].High-candles[i].Low,
			math.Max(math.Abs(candles[i].High-prev), math.Abs(candles[i].Low-prev)))
		sum += tr
	}
	return sum / float64(n)
}

// etfSignal constructs a Signal with ATR-based stop and R:R target.
func etfSignal(strategyID, symbol, direction, reason string, bar Candle, atrStop, rrMultiple float64) Signal {
	var stop, target float64
	if direction == "BUY" {
		stop = bar.Close - atrStop
		riskPerShare := bar.Close - stop
		target = bar.Close + riskPerShare*rrMultiple
	} else {
		stop = bar.Close + atrStop
		riskPerShare := stop - bar.Close
		target = bar.Close - riskPerShare*rrMultiple
	}
	return Signal{
		Symbol:      symbol,
		StrategyID:  strategyID,
		Direction:   direction,
		EntryPrice:  bar.Close,
		StopLoss:    stop,
		TakeProfit:  target,
		GeneratedAt: bar.Timestamp.UTC(),
		Reason:      reason,
	}
}

// etfDropPct returns the percentage drop from the highest candle in the window to the latest close.
// A positive value means the price dropped.
func etfDropPct(candles []Candle) float64 {
	if len(candles) < 2 {
		return 0
	}
	high := candles[0].High
	for _, c := range candles {
		if c.High > high {
			high = c.High
		}
	}
	latest := candles[len(candles)-1].Close
	if high == 0 {
		return 0
	}
	return ((high - latest) / high) * 100.0
}
