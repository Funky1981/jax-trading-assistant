package app

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/internal/domain"
)

func TestTradeGenerator_GeneratesTargetsFromRule(t *testing.T) {
	market := &fakeMarket{
		candles: []Candle{
			{TS: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100, Low: 95, High: 105},
			{TS: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), Open: 100, Close: 101, Low: 95, High: 105},
		},
	}

	strategies := map[string]domain.StrategyConfig{
		"earnings_gap_v1": {
			ID:         "earnings_gap_v1",
			Name:       "Earnings Gap V1",
			EventTypes: []string{"gap_open"},
			TargetRule: "2R,3R",
		},
	}

	g := NewTradeGenerator(market, strategies)
	e := domain.Event{
		ID:     "ev1",
		Symbol: "AAPL",
		Type:   domain.EventGapOpen,
		Time:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			"gapPct": 5.0,
		},
	}

	setups, err := g.GenerateFromEvent(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(setups) != 1 {
		t.Fatalf("expected 1 setup, got %d", len(setups))
	}
	if len(setups[0].Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(setups[0].Targets))
	}
	// entry=100 stop=95 => risk=5 => 2R=110, 3R=115
	if setups[0].Targets[0] != 110 || setups[0].Targets[1] != 115 {
		t.Fatalf("unexpected targets: %#v", setups[0].Targets)
	}
}
