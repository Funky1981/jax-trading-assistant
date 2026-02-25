package strategytypes

import (
	"context"
	"testing"
	"time"
)

func sampleCandles(start time.Time) map[string][]Candle {
	out := make([]Candle, 0, 120)
	price := 100.0
	for i := 0; i < 120; i++ {
		ts := start.Add(time.Duration(i) * time.Minute)
		open := price
		price = open + 0.1
		out = append(out, Candle{
			Timestamp: ts,
			Open:      open,
			High:      price + 0.05,
			Low:       open - 0.05,
			Close:     price,
			Volume:    1000 + float64(i*5),
		})
	}
	return map[string][]Candle{"1m": out}
}

func sampleCandlesWithStart(start time.Time, startPrice float64) map[string][]Candle {
	out := make([]Candle, 0, 120)
	price := startPrice
	for i := 0; i < 120; i++ {
		ts := start.Add(time.Duration(i) * time.Minute)
		open := price
		price = open + 0.1
		out = append(out, Candle{
			Timestamp: ts,
			Open:      open,
			High:      price + 0.05,
			Low:       open - 0.05,
			Close:     price,
			Volume:    1000 + float64(i*5),
		})
	}
	return map[string][]Candle{"1m": out}
}

func TestDefaultRegistry_HasEightTypes(t *testing.T) {
	r := DefaultRegistry()
	all := r.ListMetadata()
	if len(all) != 8 {
		t.Fatalf("expected 8 strategy types, got %d", len(all))
	}
	last := ""
	for _, m := range all {
		if m.StrategyID <= last {
			t.Fatalf("strategy metadata not in deterministic sorted order: %s then %s", last, m.StrategyID)
		}
		last = m.StrategyID
	}
}

func TestStrategyTypes_ValidateAndGenerateDeterministic(t *testing.T) {
	r := DefaultRegistry()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	input := StrategyInput{
		Symbol:      "SPY",
		SessionDate: sessionStart,
		Timezone:    "America/New_York",
		Candles:     sampleCandles(sessionStart),
		Earnings: []EarningsEvent{
			{Timestamp: sessionStart.Add(-2 * time.Hour), SurprisePct: 3.2, Guidance: "positive", PreviousClose: 98},
		},
		News: []NewsEvent{
			{Timestamp: sessionStart.Add(10 * time.Minute), Category: "earnings", Materiality: "high", Sentiment: "positive"},
		},
		Parameters: map[string]any{},
	}

	for _, meta := range r.ListMetadata() {
		st, ok := r.Get(meta.StrategyID)
		if !ok {
			t.Fatalf("missing strategy type: %s", meta.StrategyID)
		}
		if err := st.Validate(map[string]any{}); err != nil {
			t.Fatalf("%s validate failed: %v", meta.StrategyID, err)
		}
		first, err := st.Generate(context.Background(), input)
		if err != nil && meta.RequiredInputs.NeedsEarnings {
			t.Fatalf("%s generate failed unexpectedly: %v", meta.StrategyID, err)
		}
		second, err := st.Generate(context.Background(), input)
		if err != nil && meta.RequiredInputs.NeedsEarnings {
			t.Fatalf("%s second generate failed unexpectedly: %v", meta.StrategyID, err)
		}
		if len(first) != len(second) {
			t.Fatalf("%s non-deterministic signal count: %d vs %d", meta.StrategyID, len(first), len(second))
		}
	}
}

func TestNewsStrategy_MissingNewsError(t *testing.T) {
	s := NewSameDayNewsRepricing()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	_, err := s.Generate(context.Background(), StrategyInput{
		Symbol:  "AAPL",
		Candles: sampleCandles(sessionStart),
	})
	if err == nil {
		t.Fatal("expected missing required inputs error")
	}
}

func TestNewsShockMomentum_MissingNewsError(t *testing.T) {
	s := NewNewsShockMomentum()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	_, err := s.Generate(context.Background(), StrategyInput{
		Symbol:  "MSFT",
		Candles: sampleCandles(sessionStart),
	})
	if err == nil {
		t.Fatal("expected missing required inputs error")
	}
}

func TestEventGapContinuation_MissingEarningsError(t *testing.T) {
	s := NewEventGapContinuation()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	_, err := s.Generate(context.Background(), StrategyInput{
		Symbol:  "MSFT",
		Candles: sampleCandles(sessionStart),
	})
	if err == nil {
		t.Fatal("expected missing required inputs error")
	}
}

func TestNewsShockMomentum_GeneratesSignal(t *testing.T) {
	s := NewNewsShockMomentum()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	input := StrategyInput{
		Symbol:      "AAPL",
		SessionDate: sessionStart,
		Timezone:    "America/New_York",
		Candles:     sampleCandles(sessionStart),
		News: []NewsEvent{
			{Timestamp: sessionStart.Add(2 * time.Minute), Category: "earnings", Materiality: "high", Sentiment: "positive"},
		},
		Parameters: map[string]any{
			"minVolumeMultiple": 1.0,
		},
	}
	signals, err := s.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Direction != "BUY" {
		t.Fatalf("expected BUY direction, got %s", signals[0].Direction)
	}
}

func TestEventGapContinuation_GeneratesSignal(t *testing.T) {
	s := NewEventGapContinuation()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	input := StrategyInput{
		Symbol:      "TSLA",
		SessionDate: sessionStart,
		Timezone:    "America/New_York",
		Candles:     sampleCandlesWithStart(sessionStart, 103),
		Earnings: []EarningsEvent{
			{Timestamp: sessionStart.Add(-2 * time.Hour), SurprisePct: 5, Guidance: "positive", PreviousClose: 100},
		},
		Parameters: map[string]any{
			"minVolumeMultiple": 1.0,
		},
	}
	signals, err := s.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Direction != "BUY" {
		t.Fatalf("expected BUY direction, got %s", signals[0].Direction)
	}
}

func TestPairsEventRelative_GeneratesSignal(t *testing.T) {
	s := NewPairsEventRelative()
	sessionStart := time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	input := StrategyInput{
		Symbol:      "IBM",
		SessionDate: sessionStart,
		Timezone:    "America/New_York",
		Candles:     sampleCandles(sessionStart),
		News: []NewsEvent{
			{Timestamp: sessionStart.Add(5 * time.Minute), Category: "macro", Materiality: "medium", Sentiment: "positive"},
		},
		Parameters: map[string]any{
			"relativeStrengthPct": 2.5,
		},
	}
	signals, err := s.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].Direction != "BUY" {
		t.Fatalf("expected BUY direction, got %s", signals[0].Direction)
	}
}
