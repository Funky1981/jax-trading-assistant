package strategies

import (
	"context"
	"testing"
	"time"
)

// mockDataSource implements HistoricalDataSource for testing
type mockDataSource struct {
	candles    map[string][]Candle
	indicators map[string]map[time.Time]AnalysisInput
}

func newMockDataSource() *mockDataSource {
	return &mockDataSource{
		candles:    make(map[string][]Candle),
		indicators: make(map[string]map[time.Time]AnalysisInput),
	}
}

func (m *mockDataSource) GetCandles(ctx context.Context, symbol string, start, end time.Time) ([]Candle, error) {
	return m.candles[symbol], nil
}

func (m *mockDataSource) GetIndicators(ctx context.Context, symbol string, timestamp time.Time) (AnalysisInput, error) {
	if indicators, ok := m.indicators[symbol]; ok {
		if input, ok := indicators[timestamp]; ok {
			return input, nil
		}
	}
	return AnalysisInput{}, nil
}

func TestBacktester_SimpleWinningTrade(t *testing.T) {
	registry := NewRegistry()
	strategy := NewRSIMomentumStrategy()
	registry.Register(strategy, strategy.GetMetadata())

	backtester := NewBacktester(registry).
		WithCapital(10000.0).
		WithRiskPerTrade(0.02)

	// Create mock data
	dataSource := newMockDataSource()
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Add candles for AAPL
	dataSource.candles["AAPL"] = []Candle{
		{Symbol: "AAPL", Timestamp: baseTime, Open: 150, High: 152, Low: 149, Close: 151, Volume: 1000000},
		{Symbol: "AAPL", Timestamp: baseTime.Add(24 * time.Hour), Open: 151, High: 156, Low: 150, Close: 155, Volume: 1200000},
		{Symbol: "AAPL", Timestamp: baseTime.Add(48 * time.Hour), Open: 155, High: 158, Low: 154, Close: 157, Volume: 1100000},
	}

	// Add indicators with oversold RSI
	dataSource.indicators["AAPL"] = map[time.Time]AnalysisInput{
		baseTime: {
			Symbol:      "AAPL",
			Price:       150.0,
			Timestamp:   baseTime,
			RSI:         25.0, // Oversold - should trigger buy
			ATR:         2.0,
			MarketTrend: "bullish",
			Volume:      1000000,
			AvgVolume20: 900000,
		},
	}

	config := BacktestConfig{
		StrategyID: strategy.ID(),
		Symbols:    []string{"AAPL"},
		StartDate:  baseTime,
		EndDate:    baseTime.Add(72 * time.Hour),
		DataSource: dataSource,
	}

	result, err := backtester.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	if result.TotalTrades == 0 {
		t.Error("expected at least one trade")
	}

	if result.FinalCapital < result.InitialCapital {
		t.Errorf("expected profit, got loss: initial=%.2f, final=%.2f", result.InitialCapital, result.FinalCapital)
	}
}

func TestBacktester_StopLossHit(t *testing.T) {
	registry := NewRegistry()
	strategy := NewRSIMomentumStrategy()
	registry.Register(strategy, strategy.GetMetadata())

	backtester := NewBacktester(registry).
		WithCapital(10000.0).
		WithRiskPerTrade(0.01)

	dataSource := newMockDataSource()
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create scenario where stop loss gets hit
	dataSource.candles["TSLA"] = []Candle{
		{Symbol: "TSLA", Timestamp: baseTime, Open: 200, High: 202, Low: 199, Close: 201, Volume: 2000000},
		{Symbol: "TSLA", Timestamp: baseTime.Add(24 * time.Hour), Open: 201, High: 201, Low: 190, Close: 192, Volume: 2500000}, // Stop loss hit
	}

	dataSource.indicators["TSLA"] = map[time.Time]AnalysisInput{
		baseTime: {
			Symbol:      "TSLA",
			Price:       200.0,
			Timestamp:   baseTime,
			RSI:         28.0, // Oversold - buy signal
			ATR:         5.0,
			MarketTrend: "bullish",
			Volume:      2000000,
			AvgVolume20: 1800000,
		},
	}

	config := BacktestConfig{
		StrategyID: strategy.ID(),
		Symbols:    []string{"TSLA"},
		StartDate:  baseTime,
		EndDate:    baseTime.Add(48 * time.Hour),
		DataSource: dataSource,
	}

	result, err := backtester.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	if result.TotalTrades == 0 {
		t.Error("expected at least one trade")
	}

	if result.TotalTrades > 0 {
		trade := result.Trades[0]
		if trade.ExitReason != "Stop loss hit" {
			t.Errorf("expected 'Stop loss hit', got '%s'", trade.ExitReason)
		}
		if trade.PnL >= 0 {
			t.Error("expected negative PnL when stop loss hit")
		}
	}
}

func TestBacktester_MultipleSymbols(t *testing.T) {
	registry := NewRegistry()
	strategy := NewMACDCrossoverStrategy()
	registry.Register(strategy, strategy.GetMetadata())

	backtester := NewBacktester(registry).
		WithCapital(50000.0).
		WithRiskPerTrade(0.015).
		WithMaxPositions(3)

	dataSource := newMockDataSource()
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	symbols := []string{"AAPL", "MSFT", "GOOGL"}
	for _, symbol := range symbols {
		dataSource.candles[symbol] = []Candle{
			{Symbol: symbol, Timestamp: baseTime, Open: 100, High: 102, Low: 99, Close: 101, Volume: 1000000},
			{Symbol: symbol, Timestamp: baseTime.Add(24 * time.Hour), Open: 101, High: 105, Low: 100, Close: 104, Volume: 1200000},
			{Symbol: symbol, Timestamp: baseTime.Add(48 * time.Hour), Open: 104, High: 108, Low: 103, Close: 107, Volume: 1100000},
		}

		dataSource.indicators[symbol] = map[time.Time]AnalysisInput{
			baseTime: {
				Symbol:    symbol,
				Price:     100.0,
				Timestamp: baseTime,
				MACD: MACD{
					Value:     1.5,
					Signal:    0.8,
					Histogram: 0.7,
				},
				ATR:         2.0,
				MarketTrend: "bullish",
				Volume:      1000000,
				AvgVolume20: 900000,
			},
		}
	}

	config := BacktestConfig{
		StrategyID: strategy.ID(),
		Symbols:    symbols,
		StartDate:  baseTime,
		EndDate:    baseTime.Add(72 * time.Hour),
		DataSource: dataSource,
	}

	result, err := backtester.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	if result.TotalTrades == 0 {
		t.Error("expected trades across multiple symbols")
	}

	if result.TotalTrades > 3 {
		t.Errorf("expected max 3 trades (max positions), got %d", result.TotalTrades)
	}
}

func TestBacktester_PerformanceMetrics(t *testing.T) {
	registry := NewRegistry()
	strategy := NewMACrossoverStrategy()
	registry.Register(strategy, strategy.GetMetadata())

	backtester := NewBacktester(registry).
		WithCapital(100000.0).
		WithRiskPerTrade(0.01)

	dataSource := newMockDataSource()
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create a winning trade scenario
	dataSource.candles["SPY"] = []Candle{
		{Symbol: "SPY", Timestamp: baseTime, Open: 450, High: 452, Low: 449, Close: 451, Volume: 50000000},
		{Symbol: "SPY", Timestamp: baseTime.Add(24 * time.Hour), Open: 451, High: 465, Low: 450, Close: 464, Volume: 55000000}, // TP hit
	}

	dataSource.indicators["SPY"] = map[time.Time]AnalysisInput{
		baseTime: {
			Symbol:      "SPY",
			Price:       450.0,
			Timestamp:   baseTime,
			SMA20:       448.0,
			SMA50:       445.0,
			SMA200:      440.0,
			ATR:         5.0,
			MarketTrend: "bullish",
			Volume:      50000000,
			AvgVolume20: 45000000,
		},
	}

	config := BacktestConfig{
		StrategyID: strategy.ID(),
		Symbols:    []string{"SPY"},
		StartDate:  baseTime,
		EndDate:    baseTime.Add(48 * time.Hour),
		DataSource: dataSource,
	}

	result, err := backtester.Run(context.Background(), config)
	if err != nil {
		t.Fatalf("backtest failed: %v", err)
	}

	// Check that metrics are calculated
	if result.WinRate < 0 || result.WinRate > 1 {
		t.Errorf("invalid win rate: %.2f", result.WinRate)
	}

	if result.TotalReturn == 0 && result.TotalTrades > 0 {
		t.Error("expected non-zero total return")
	}

	if result.TotalTrades > 0 && result.AvgR == 0 {
		t.Error("expected non-zero average R")
	}
}
