package strategies

import (
	"context"
)

// MACrossoverStrategy implements a trend strategy based on moving average crossovers
type MACrossoverStrategy struct {
	id            string
	name          string
	minConfidence float64
}

// NewMACrossoverStrategy creates a new MA crossover strategy
func NewMACrossoverStrategy() *MACrossoverStrategy {
	return &MACrossoverStrategy{
		id:            "ma_crossover_v1",
		name:          "MA Crossover V1",
		minConfidence: 0.65,
	}
}

func (s *MACrossoverStrategy) ID() string {
	return s.id
}

func (s *MACrossoverStrategy) Name() string {
	return s.name
}

func (s *MACrossoverStrategy) Analyze(ctx context.Context, input AnalysisInput) (Signal, error) {
	signal := Signal{
		Symbol:     input.Symbol,
		Timestamp:  input.Timestamp,
		Type:       SignalHold,
		Confidence: 0.0,
		Indicators: make(map[string]interface{}),
	}

	// Store indicator values
	signal.Indicators["sma20"] = input.SMA20
	signal.Indicators["sma50"] = input.SMA50
	signal.Indicators["sma200"] = input.SMA200
	signal.Indicators["price"] = input.Price
	signal.Indicators["atr"] = input.ATR

	// Golden cross: SMA20 > SMA50 > SMA200
	if input.SMA20 > input.SMA50 && input.SMA50 > input.SMA200 && input.Price > input.SMA20 {
		signal.Type = SignalBuy
		signal.Confidence = s.calculateConfidence(input, true)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.SMA50 - input.ATR
		signal.TakeProfit = []float64{
			input.Price + (3.0 * input.ATR), // 3R target
			input.Price + (5.0 * input.ATR), // 5R target
		}
		signal.Reason = "Golden cross: SMA20 > SMA50 > SMA200, strong bullish trend"
		return signal, nil
	}

	// Death cross: SMA20 < SMA50 < SMA200
	if input.SMA20 < input.SMA50 && input.SMA50 < input.SMA200 && input.Price < input.SMA20 {
		signal.Type = SignalSell
		signal.Confidence = s.calculateConfidence(input, false)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.SMA50 + input.ATR
		signal.TakeProfit = []float64{
			input.Price - (3.0 * input.ATR), // 3R target
			input.Price - (5.0 * input.ATR), // 5R target
		}
		signal.Reason = "Death cross: SMA20 < SMA50 < SMA200, strong bearish trend"
		return signal, nil
	}

	// Bullish pullback: price near SMA20 in uptrend
	if input.SMA20 > input.SMA50 && input.SMA50 > input.SMA200 {
		priceDiffPct := (input.Price - input.SMA20) / input.SMA20
		if priceDiffPct >= -0.02 && priceDiffPct <= 0.01 {
			signal.Type = SignalBuy
			signal.Confidence = s.minConfidence - 0.10 // Lower confidence for pullback
			signal.EntryPrice = input.Price
			signal.StopLoss = input.Price - (1.5 * input.ATR)
			signal.TakeProfit = []float64{
				input.Price + (2.0 * input.ATR),
				input.Price + (3.5 * input.ATR),
			}
			signal.Reason = "Bullish pullback to SMA20 in uptrend"
			return signal, nil
		}
	}

	// No clear signal
	signal.Reason = "No clear MA alignment for entry"
	return signal, nil
}

func (s *MACrossoverStrategy) calculateConfidence(input AnalysisInput, isBullish bool) float64 {
	confidence := s.minConfidence

	// Boost confidence if market trend aligns
	if isBullish && input.MarketTrend == "bullish" {
		confidence += 0.12
	} else if !isBullish && input.MarketTrend == "bearish" {
		confidence += 0.12
	}

	// Boost confidence if volume confirms
	if input.Volume > input.AvgVolume20 {
		confidence += 0.08
	}

	// Boost confidence based on MA separation (stronger trend)
	if isBullish {
		separation := (input.SMA20 - input.SMA200) / input.SMA200
		if separation > 0.05 {
			confidence += 0.10
		}
	} else {
		separation := (input.SMA200 - input.SMA20) / input.SMA200
		if separation > 0.05 {
			confidence += 0.10
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetMetadata returns strategy metadata
func (s *MACrossoverStrategy) GetMetadata() StrategyMetadata {
	return StrategyMetadata{
		ID:          s.id,
		Name:        s.name,
		Description: "Trend-following strategy based on moving average alignments (golden/death cross)",
		EventTypes:  []string{"golden_cross", "death_cross", "ma_pullback"},
		MinRR:       2.0,
		MaxRisk:     1.5,
		Timeframes:  []string{"1h", "4h", "1d"},
	}
}
