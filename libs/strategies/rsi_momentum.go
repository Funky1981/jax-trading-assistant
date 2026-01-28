package strategies

import (
	"context"
	"fmt"
)

// RSIMomentumStrategy implements a momentum strategy based on RSI
type RSIMomentumStrategy struct {
	id              string
	name            string
	oversoldLevel   float64
	overboughtLevel float64
	minConfidence   float64
}

// NewRSIMomentumStrategy creates a new RSI momentum strategy
func NewRSIMomentumStrategy() *RSIMomentumStrategy {
	return &RSIMomentumStrategy{
		id:              "rsi_momentum_v1",
		name:            "RSI Momentum V1",
		oversoldLevel:   30.0,
		overboughtLevel: 70.0,
		minConfidence:   0.6,
	}
}

func (s *RSIMomentumStrategy) ID() string {
	return s.id
}

func (s *RSIMomentumStrategy) Name() string {
	return s.name
}

func (s *RSIMomentumStrategy) Analyze(ctx context.Context, input AnalysisInput) (Signal, error) {
	signal := Signal{
		Symbol:     input.Symbol,
		Timestamp:  input.Timestamp,
		Type:       SignalHold,
		Confidence: 0.0,
		Indicators: make(map[string]interface{}),
	}

	// Store indicator values for transparency
	signal.Indicators["rsi"] = input.RSI
	signal.Indicators["price"] = input.Price
	signal.Indicators["atr"] = input.ATR

	// Oversold condition - potential buy
	if input.RSI < s.oversoldLevel {
		signal.Type = SignalBuy
		signal.Confidence = s.calculateConfidence(input, true)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.Price - (2.0 * input.ATR)
		signal.TakeProfit = []float64{
			input.Price + (2.0 * input.ATR), // 2R target
			input.Price + (3.0 * input.ATR), // 3R target
		}
		signal.Reason = fmt.Sprintf("RSI oversold at %.2f, bullish reversal expected", input.RSI)
		return signal, nil
	}

	// Overbought condition - potential sell
	if input.RSI > s.overboughtLevel {
		signal.Type = SignalSell
		signal.Confidence = s.calculateConfidence(input, false)
		signal.EntryPrice = input.Price
		signal.StopLoss = input.Price + (2.0 * input.ATR)
		signal.TakeProfit = []float64{
			input.Price - (2.0 * input.ATR), // 2R target
			input.Price - (3.0 * input.ATR), // 3R target
		}
		signal.Reason = fmt.Sprintf("RSI overbought at %.2f, bearish reversal expected", input.RSI)
		return signal, nil
	}

	// Neutral zone
	signal.Reason = fmt.Sprintf("RSI neutral at %.2f, no clear signal", input.RSI)
	return signal, nil
}

func (s *RSIMomentumStrategy) calculateConfidence(input AnalysisInput, isBullish bool) float64 {
	confidence := s.minConfidence

	// Boost confidence if market trend aligns
	if isBullish && input.MarketTrend == "bullish" {
		confidence += 0.15
	} else if !isBullish && input.MarketTrend == "bearish" {
		confidence += 0.15
	}

	// Boost confidence if volume is strong
	if input.Volume > input.AvgVolume20 {
		confidence += 0.10
	}

	// Boost confidence based on RSI extreme
	if isBullish {
		if input.RSI < 20 {
			confidence += 0.15
		}
	} else {
		if input.RSI > 80 {
			confidence += 0.15
		}
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetMetadata returns strategy metadata
func (s *RSIMomentumStrategy) GetMetadata() StrategyMetadata {
	return StrategyMetadata{
		ID:          s.id,
		Name:        s.name,
		Description: "Mean reversion strategy based on RSI oversold/overbought levels",
		EventTypes:  []string{"rsi_oversold", "rsi_overbought"},
		MinRR:       2.0,
		MaxRisk:     2.0,
		Timeframes:  []string{"5m", "15m", "1h", "4h"},
	}
}
