package strategies

import (
	"context"
	"time"
)

// Strategy is the core interface all trading strategies must implement
type Strategy interface {
	// ID returns the unique identifier for this strategy
	ID() string

	// Name returns the human-readable name
	Name() string

	// Analyze examines market data and returns a signal
	Analyze(ctx context.Context, input AnalysisInput) (Signal, error)
}

// AnalysisInput contains all data a strategy needs to make decisions
type AnalysisInput struct {
	Symbol    string
	Price     float64
	Timestamp time.Time

	// Technical indicators
	RSI            float64
	MACD           MACD
	SMA20          float64
	SMA50          float64
	SMA200         float64
	ATR            float64
	BollingerBands BollingerBands

	// Volume data
	Volume      int64
	AvgVolume20 int64

	// Market context
	MarketTrend string // "bullish", "bearish", "neutral"
	SectorTrend string
}

// MACD holds MACD indicator values
type MACD struct {
	Value     float64
	Signal    float64
	Histogram float64
}

// BollingerBands holds Bollinger Band values
type BollingerBands struct {
	Upper  float64
	Middle float64
	Lower  float64
}

// SignalType represents the type of trading signal
type SignalType string

const (
	SignalBuy  SignalType = "buy"
	SignalSell SignalType = "sell"
	SignalHold SignalType = "hold"
)

// Signal is the output from a strategy analysis
type Signal struct {
	Type       SignalType
	Symbol     string
	Timestamp  time.Time
	Confidence float64 // 0.0 to 1.0

	// Trade parameters
	EntryPrice float64
	StopLoss   float64
	TakeProfit []float64 // Multiple targets

	// Rationale
	Reason     string
	Indicators map[string]interface{} // Supporting data
}

// StrategyMetadata provides information about a strategy
type StrategyMetadata struct {
	ID          string
	Name        string
	Description string
	EventTypes  []string
	MinRR       float64
	MaxRisk     float64
	Timeframes  []string               // "1m", "5m", "1h", "1d"
	Extra       map[string]interface{} // ADR-0012 Phase 4: Artifact tracking (artifact_id, artifact_hash)
}
