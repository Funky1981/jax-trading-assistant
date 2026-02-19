package app

import (
	"context"
	"testing"
	"time"
)

type fakeMarket struct {
	candles []Candle
	err     error
}

func (m *fakeMarket) GetDailyCandles(_ context.Context, _ string, _ int) ([]Candle, error) {
	return m.candles, m.err
}

func TestEventDetector_DetectGaps_AboveThreshold(t *testing.T) {
	market := &fakeMarket{
		candles: []Candle{
			{TS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
			{TS: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 105},
		},
	}
	detector := NewEventDetector(market, nil)

	events, err := detector.DetectGaps(context.Background(), "AAPL", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Symbol != "AAPL" {
		t.Fatalf("unexpected symbol: %s", events[0].Symbol)
	}
}

func TestEventDetector_DetectGaps_BelowThreshold(t *testing.T) {
	market := &fakeMarket{
		candles: []Candle{
			{TS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
			{TS: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 101},
		},
	}
	detector := NewEventDetector(market, nil)

	events, err := detector.DetectGaps(context.Background(), "AAPL", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}
