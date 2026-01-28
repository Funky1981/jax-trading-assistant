package strategies

import (
	"context"
	"fmt"
)

// MACDCrossoverStrategy implements a trend-following strategy based on MACD
type MACDCrossoverStrategy struct {
	id            string
	name          string
	minHistogram  float64
	minConfidence float64
}

// NewMACDCrossoverStrategy creates a new MACD crossover strategy
func NewMACDCrossoverStrategy() *MACDCrossoverStrategy {
	return &MACDCrossoverStrategy{
		id:            "macd_crossover_v1",
		name:          "MACD Crossover V1",
		minHistogram:  0.0,
		minConfidence: 0.6,
	}
}

func (s *MACDCrossoverStrategy) ID() string {
	return s.id
}

func (s *MACDCrossoverStrategy) Name() string {
	return s.name
}

func (s *MACDCrossoverStrategy) Analyze(ctx context.Context, input AnalysisInput) (Signal, error) {
	signal := Signal{
		Symbol:     input.Symbol,
		Timestamp:  input.Timestamp,
		Type:       SignalHold,
		Confidence: 0.0,
		Indicators: make(map[string]interface{}),
	}

	// Store indicator values
	signal.Indicators["macd_value"] = input.MACD.Value
	signal.Indicators["macd_signal"] = input.MACD.Signal
	signal.Indicators["macd_histogram"] = input.MACD.Histogram
	signal.Indicators["price"] = input.Price
	signal.Indicators["atr"] = input.ATR

	// Bullish crossover: MACD above signal and histogram positive
	if input.MACD.Histogram > s.minHistogram && input.MACD.Value > input.MACD.Signal {
		signal.Type = SignalBuy
		signal.Confidence = s.calculateConfidence(input, true)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.Price - (2.0 * input.ATR)
		signal.TakeProfit = []float64{
			input.Price + (2.5 * input.ATR), // 2.5R target
			input.Price + (4.0 * input.ATR), // 4R target
		}
		signal.Reason = fmt.Sprintf("MACD bullish crossover, histogram=%.4f", input.MACD.Histogram)
		return signal, nil
	}

	// Bearish crossover: MACD below signal and histogram negative
	if input.MACD.Histogram < -s.minHistogram && input.MACD.Value < input.MACD.Signal {
		signal.Type = SignalSell
		signal.Confidence = s.calculateConfidence(input, false)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.Price + (2.0 * input.ATR)
		signal.TakeProfit = []float64{
			input.Price - (2.5 * input.ATR), // 2.5R target
			input.Price - (4.0 * input.ATR), // 4R target
		}
		signal.Reason = fmt.Sprintf("MACD bearish crossover, histogram=%.4f", input.MACD.Histogram)
		return signal, nil
	}

	// No clear signal
	signal.Reason = fmt.Sprintf("MACD neutral, histogram=%.4f", input.MACD.Histogram)
	return signal, nil
}

func (s *MACDCrossoverStrategy) calculateConfidence(input AnalysisInput, isBullish bool) float64 {
	confidence := s.minConfidence

	// Boost confidence if market trend aligns
	if isBullish && input.MarketTrend == "bullish" {
		confidence += 0.15
	} else if !isBullish && input.MarketTrend == "bearish" {
		confidence += 0.15
	}

	// Boost confidence if sector trend aligns
	if isBullish && input.SectorTrend == "bullish" {
		confidence += 0.10
	} else if !isBullish && input.SectorTrend == "bearish" {
		confidence += 0.10
	}

	// Boost confidence based on histogram strength
	histogramAbs := input.MACD.Histogram
	if histogramAbs < 0 {
		histogramAbs = -histogramAbs
	}
	if histogramAbs > 0.5 {
		confidence += 0.10
	}

	// Boost confidence if volume is strong
	if input.Volume > input.AvgVolume20 {
		confidence += 0.05
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetMetadata returns strategy metadata
func (s *MACDCrossoverStrategy) GetMetadata() StrategyMetadata {
	return StrategyMetadata{
		ID:          s.id,
		Name:        s.name,
		Description: "Trend-following strategy based on MACD crossover signals",
		EventTypes:  []string{"macd_bullish_crossover", "macd_bearish_crossover"},
		MinRR:       2.5,
		MaxRisk:     2.0,
		Timeframes:  []string{"15m", "1h", "4h", "1d"},
	}
}
