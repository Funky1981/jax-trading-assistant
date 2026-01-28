package strategies

import (
	"context"
	"testing"
	"time"
)

func TestRSIMomentumStrategy_Oversold(t *testing.T) {
	strategy := NewRSIMomentumStrategy()

	input := AnalysisInput{
		Symbol:      "AAPL",
		Price:       150.0,
		Timestamp:   time.Now(),
		RSI:         25.0, // Oversold
		ATR:         2.5,
		MarketTrend: "bullish",
		Volume:      1000000,
		AvgVolume20: 800000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalBuy {
		t.Errorf("expected SignalBuy, got %v", signal.Type)
	}

	if signal.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6, got %.2f", signal.Confidence)
	}

	if signal.StopLoss >= input.Price {
		t.Errorf("stop loss should be below entry price")
	}

	if len(signal.TakeProfit) != 2 {
		t.Errorf("expected 2 take profit targets, got %d", len(signal.TakeProfit))
	}
}

func TestRSIMomentumStrategy_Overbought(t *testing.T) {
	strategy := NewRSIMomentumStrategy()

	input := AnalysisInput{
		Symbol:      "TSLA",
		Price:       200.0,
		Timestamp:   time.Now(),
		RSI:         75.0, // Overbought
		ATR:         5.0,
		MarketTrend: "bearish",
		Volume:      2000000,
		AvgVolume20: 1500000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalSell {
		t.Errorf("expected SignalSell, got %v", signal.Type)
	}

	if signal.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6, got %.2f", signal.Confidence)
	}

	if signal.StopLoss <= input.Price {
		t.Errorf("stop loss should be above entry price for sell")
	}
}

func TestRSIMomentumStrategy_Neutral(t *testing.T) {
	strategy := NewRSIMomentumStrategy()

	input := AnalysisInput{
		Symbol:    "MSFT",
		Price:     300.0,
		Timestamp: time.Now(),
		RSI:       50.0, // Neutral
		ATR:       3.0,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalHold {
		t.Errorf("expected SignalHold, got %v", signal.Type)
	}

	if signal.Confidence != 0.0 {
		t.Errorf("expected zero confidence for hold, got %.2f", signal.Confidence)
	}
}

func TestMACDCrossoverStrategy_Bullish(t *testing.T) {
	strategy := NewMACDCrossoverStrategy()

	input := AnalysisInput{
		Symbol:    "NVDA",
		Price:     500.0,
		Timestamp: time.Now(),
		MACD: MACD{
			Value:     2.5,
			Signal:    1.0,
			Histogram: 1.5,
		},
		ATR:         10.0,
		MarketTrend: "bullish",
		SectorTrend: "bullish",
		Volume:      3000000,
		AvgVolume20: 2500000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalBuy {
		t.Errorf("expected SignalBuy, got %v", signal.Type)
	}

	if signal.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6, got %.2f", signal.Confidence)
	}

	if len(signal.TakeProfit) != 2 {
		t.Errorf("expected 2 take profit targets, got %d", len(signal.TakeProfit))
	}
}

func TestMACDCrossoverStrategy_Bearish(t *testing.T) {
	strategy := NewMACDCrossoverStrategy()

	input := AnalysisInput{
		Symbol:    "AMD",
		Price:     100.0,
		Timestamp: time.Now(),
		MACD: MACD{
			Value:     -2.0,
			Signal:    -0.5,
			Histogram: -1.5,
		},
		ATR:         2.0,
		MarketTrend: "bearish",
		SectorTrend: "bearish",
		Volume:      1500000,
		AvgVolume20: 1000000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalSell {
		t.Errorf("expected SignalSell, got %v", signal.Type)
	}

	if signal.Confidence < 0.6 {
		t.Errorf("expected confidence >= 0.6, got %.2f", signal.Confidence)
	}
}

func TestMACrossoverStrategy_GoldenCross(t *testing.T) {
	strategy := NewMACrossoverStrategy()

	input := AnalysisInput{
		Symbol:      "SPY",
		Price:       450.0,
		Timestamp:   time.Now(),
		SMA20:       445.0,
		SMA50:       440.0,
		SMA200:      430.0,
		ATR:         5.0,
		MarketTrend: "bullish",
		Volume:      50000000,
		AvgVolume20: 40000000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalBuy {
		t.Errorf("expected SignalBuy for golden cross, got %v", signal.Type)
	}

	if signal.Confidence < 0.65 {
		t.Errorf("expected confidence >= 0.65, got %.2f", signal.Confidence)
	}

	if signal.Reason != "Golden cross: SMA20 > SMA50 > SMA200, strong bullish trend" {
		t.Errorf("unexpected reason: %s", signal.Reason)
	}
}

func TestMACrossoverStrategy_DeathCross(t *testing.T) {
	strategy := NewMACrossoverStrategy()

	input := AnalysisInput{
		Symbol:      "QQQ",
		Price:       350.0,
		Timestamp:   time.Now(),
		SMA20:       352.0,
		SMA50:       360.0,
		SMA200:      370.0,
		ATR:         4.0,
		MarketTrend: "bearish",
		Volume:      30000000,
		AvgVolume20: 25000000,
	}

	signal, err := strategy.Analyze(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if signal.Type != SignalSell {
		t.Errorf("expected SignalSell for death cross, got %v", signal.Type)
	}

	if signal.Confidence < 0.65 {
		t.Errorf("expected confidence >= 0.65, got %.2f", signal.Confidence)
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry()

	strategy := NewRSIMomentumStrategy()
	metadata := strategy.GetMetadata()

	err := registry.Register(strategy, metadata)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	retrieved, err := registry.Get(strategy.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID() != strategy.ID() {
		t.Errorf("expected strategy ID %s, got %s", strategy.ID(), retrieved.ID())
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	registry := NewRegistry()

	strategy := NewRSIMomentumStrategy()
	metadata := strategy.GetMetadata()

	err := registry.Register(strategy, metadata)
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	err = registry.Register(strategy, metadata)
	if err == nil {
		t.Error("expected error on duplicate registration")
	}
}

func TestRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent strategy")
	}
}

func TestRegistry_ListAll(t *testing.T) {
	registry := NewRegistry()

	rsi := NewRSIMomentumStrategy()
	macd := NewMACDCrossoverStrategy()
	ma := NewMACrossoverStrategy()

	registry.Register(rsi, rsi.GetMetadata())
	registry.Register(macd, macd.GetMetadata())
	registry.Register(ma, ma.GetMetadata())

	all := registry.ListAll()
	if len(all) != 3 {
		t.Errorf("expected 3 strategies, got %d", len(all))
	}

	ids := registry.List()
	if len(ids) != 3 {
		t.Errorf("expected 3 strategy IDs, got %d", len(ids))
	}
}
