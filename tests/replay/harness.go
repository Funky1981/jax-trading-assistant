package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"jax-trading-assistant/libs/strategies"
)

// Fixture represents a captured market scenario for replay testing
type Fixture struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Timestamp   time.Time                 `json:"timestamp"`
	MarketData  map[string]MarketSnapshot `json:"market_data"`
	Portfolio   PortfolioSnapshot         `json:"portfolio"`
}

// MarketSnapshot contains market data for a single symbol at a point in time
type MarketSnapshot struct {
	Symbol     string                 `json:"symbol,omitempty"`
	Price      float64                `json:"price"`
	Volume     int64                  `json:"volume"`
	Bid        float64                `json:"bid"`
	Ask        float64                `json:"ask"`
	Indicators IndicatorSnapshot      `json:"indicators"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// IndicatorSnapshot contains technical indicator values
type IndicatorSnapshot struct {
	RSI           float64 `json:"rsi"`
	SMA20         float64 `json:"sma20"`
	SMA50         float64 `json:"sma50"`
	SMA200        float64 `json:"sma200"`
	ATR           float64 `json:"atr"`
	MACDValue     float64 `json:"macd_value"`
	MACDSignal    float64 `json:"macd_signal"`
	MACDHistogram float64 `json:"macd_histogram"`
	BBUpper       float64 `json:"bb_upper"`
	BBMiddle      float64 `json:"bb_middle"`
	BBLower       float64 `json:"bb_lower"`
	AvgVolume20   int64   `json:"avg_volume_20"`
}

// PortfolioSnapshot contains portfolio state at a point in time
type PortfolioSnapshot struct {
	Cash      float64    `json:"cash"`
	Positions []Position `json:"positions"`
}

// Position represents a holding in the portfolio
type Position struct {
	Symbol   string  `json:"symbol"`
	Quantity int     `json:"quantity"`
	AvgPrice float64 `json:"avg_price"`
}

// Result contains the outcome of a replay execution
type Result struct {
	Signal   strategies.Signal
	Duration time.Duration
	Error    error
	Metadata map[string]interface{}
}

// StrategyFunc is a function that analyzes market data and returns a signal
type StrategyFunc func(ctx context.Context, input strategies.AnalysisInput) (strategies.Signal, error)

// LoadFixture loads a fixture from the fixtures directory by name
func LoadFixture(name string) (*Fixture, error) {
	// Support both with and without .json extension
	filename := name
	if filepath.Ext(name) != ".json" {
		filename = name + ".json"
	}

	// Try relative to current directory first, then absolute
	paths := []string{
		filepath.Join("tests", "replay", "fixtures", filename),
		filepath.Join("fixtures", filename),
		filename,
	}

	var data []byte
	var err error
	var foundPath string

	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read fixture %s (tried: %v): %w", name, paths, err)
	}

	var fixture Fixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to parse fixture %s: %w", foundPath, err)
	}

	// Populate symbol field in market snapshots if missing
	for symbol, snapshot := range fixture.MarketData {
		if snapshot.Symbol == "" {
			snapshot.Symbol = symbol
			fixture.MarketData[symbol] = snapshot
		}
	}

	return &fixture, nil
}

// ReplayStrategy executes a strategy function with fixture data and returns the result
func ReplayStrategy(ctx context.Context, strategyFunc StrategyFunc, fixture *Fixture) (Result, error) {
	startTime := time.Now()
	result := Result{
		Metadata: make(map[string]interface{}),
	}

	// Validate fixture has at least one symbol
	if len(fixture.MarketData) == 0 {
		return result, fmt.Errorf("fixture has no market data")
	}

	// For simplicity, analyze the first symbol in the fixture
	// In production, you might analyze multiple symbols
	var symbol string
	var snapshot MarketSnapshot
	for s, snap := range fixture.MarketData {
		symbol = s
		snapshot = snap
		break
	}

	// Convert fixture data to AnalysisInput
	input := strategies.AnalysisInput{
		Symbol:    symbol,
		Price:     snapshot.Price,
		Timestamp: fixture.Timestamp,
		RSI:       snapshot.Indicators.RSI,
		MACD: strategies.MACD{
			Value:     snapshot.Indicators.MACDValue,
			Signal:    snapshot.Indicators.MACDSignal,
			Histogram: snapshot.Indicators.MACDHistogram,
		},
		SMA20:  snapshot.Indicators.SMA20,
		SMA50:  snapshot.Indicators.SMA50,
		SMA200: snapshot.Indicators.SMA200,
		ATR:    snapshot.Indicators.ATR,
		BollingerBands: strategies.BollingerBands{
			Upper:  snapshot.Indicators.BBUpper,
			Middle: snapshot.Indicators.BBMiddle,
			Lower:  snapshot.Indicators.BBLower,
		},
		Volume:      snapshot.Volume,
		AvgVolume20: snapshot.Indicators.AvgVolume20,
		MarketTrend: inferMarketTrend(snapshot),
		SectorTrend: "neutral", // Could be added to fixture
	}

	// Execute strategy
	signal, err := strategyFunc(ctx, input)
	result.Signal = signal
	result.Error = err
	result.Duration = time.Since(startTime)

	// Add metadata
	result.Metadata["fixture_name"] = fixture.Name
	result.Metadata["symbol"] = symbol
	result.Metadata["timestamp"] = fixture.Timestamp

	return result, err
}

// VerifyDeterminism runs a strategy function multiple times with the same fixture
// and verifies that the results are identical across all runs
func VerifyDeterminism(ctx context.Context, strategyFunc StrategyFunc, fixture *Fixture, runs int) error {
	if runs < 2 {
		return fmt.Errorf("runs must be at least 2, got %d", runs)
	}

	var results []Result

	// Run strategy multiple times
	for i := 0; i < runs; i++ {
		result, err := ReplayStrategy(ctx, strategyFunc, fixture)
		if err != nil {
			return fmt.Errorf("run %d failed: %w", i+1, err)
		}
		results = append(results, result)
	}

	// Compare all results against the first one
	baseline := results[0]
	for i := 1; i < len(results); i++ {
		if err := compareResults(baseline, results[i], i+1); err != nil {
			return fmt.Errorf("determinism violation: %w", err)
		}
	}

	return nil
}

// compareResults compares two replay results for equality
func compareResults(baseline, current Result, runNumber int) error {
	// Compare signal types
	if baseline.Signal.Type != current.Signal.Type {
		return fmt.Errorf("run %d: signal type mismatch: baseline=%s, current=%s",
			runNumber, baseline.Signal.Type, current.Signal.Type)
	}

	// Compare symbols
	if baseline.Signal.Symbol != current.Signal.Symbol {
		return fmt.Errorf("run %d: symbol mismatch: baseline=%s, current=%s",
			runNumber, baseline.Signal.Symbol, current.Signal.Symbol)
	}

	// Compare confidence (with small tolerance for floating point)
	const tolerance = 0.0001
	if abs(baseline.Signal.Confidence-current.Signal.Confidence) > tolerance {
		return fmt.Errorf("run %d: confidence mismatch: baseline=%.4f, current=%.4f",
			runNumber, baseline.Signal.Confidence, current.Signal.Confidence)
	}

	// Compare entry price
	if abs(baseline.Signal.EntryPrice-current.Signal.EntryPrice) > tolerance {
		return fmt.Errorf("run %d: entry price mismatch: baseline=%.2f, current=%.2f",
			runNumber, baseline.Signal.EntryPrice, current.Signal.EntryPrice)
	}

	// Compare stop loss
	if abs(baseline.Signal.StopLoss-current.Signal.StopLoss) > tolerance {
		return fmt.Errorf("run %d: stop loss mismatch: baseline=%.2f, current=%.2f",
			runNumber, baseline.Signal.StopLoss, current.Signal.StopLoss)
	}

	// Compare take profit levels
	if !reflect.DeepEqual(baseline.Signal.TakeProfit, current.Signal.TakeProfit) {
		return fmt.Errorf("run %d: take profit mismatch: baseline=%v, current=%v",
			runNumber, baseline.Signal.TakeProfit, current.Signal.TakeProfit)
	}

	// Compare timestamps
	if !baseline.Signal.Timestamp.Equal(current.Signal.Timestamp) {
		return fmt.Errorf("run %d: timestamp mismatch: baseline=%v, current=%v",
			runNumber, baseline.Signal.Timestamp, current.Signal.Timestamp)
	}

	return nil
}

// inferMarketTrend infers market trend from moving averages
func inferMarketTrend(snapshot MarketSnapshot) string {
	sma20 := snapshot.Indicators.SMA20
	sma50 := snapshot.Indicators.SMA50
	sma200 := snapshot.Indicators.SMA200

	if sma20 > sma50 && sma50 > sma200 {
		return "bullish"
	} else if sma20 < sma50 && sma50 < sma200 {
		return "bearish"
	}
	return "neutral"
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
