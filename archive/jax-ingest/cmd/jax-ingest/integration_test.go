package main

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/services/jax-ingest/internal/ingest"
)

type retainCall struct {
	bank string
	item contracts.MemoryItem
}

type fakeMemoryStore struct {
	calls []retainCall
}

func (s *fakeMemoryStore) Ping(context.Context) error { return nil }

func (s *fakeMemoryStore) Retain(_ context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	s.calls = append(s.calls, retainCall{bank: bank, item: item})
	return "mem_1", nil
}

func (s *fakeMemoryStore) Recall(context.Context, string, contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	return nil, nil
}

func (s *fakeMemoryStore) Reflect(context.Context, string, contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	return nil, nil
}

func TestIntegration_RetainDexterOutput(t *testing.T) {
	store := &fakeMemoryStore{}
	ts := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	observations := []ingest.DexterObservation{
		{
			Type:           "earnings_detected",
			Symbol:         "AAPL",
			ImpactEstimate: 0.88,
			Confidence:     0.74,
			Tags:           []string{"EARNINGS"},
			TS:             ts,
		},
		{
			Type:           "price_gap",
			Symbol:         "TSLA",
			ImpactEstimate: 0.1,
			Confidence:     0.4,
			GapPercent:     -6.2,
			Tags:           []string{"gap"},
			Bookmarked:     true,
			TS:             ts.Add(2 * time.Minute),
		},
	}

	result, err := ingest.RetainDexterObservations(context.Background(), store, observations, ingest.RetentionConfig{
		SignificanceThreshold: 0.7,
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}
	if result.Retained != 2 {
		t.Fatalf("expected 2 retained, got %d", result.Retained)
	}
	if len(store.calls) != 2 {
		t.Fatalf("expected 2 retain calls, got %d", len(store.calls))
	}

	first := store.calls[0]
	if first.bank != "market_events" {
		t.Fatalf("expected market_events bank, got %q", first.bank)
	}
	if first.item.Type != "earnings_event" {
		t.Fatalf("expected earnings_event type, got %q", first.item.Type)
	}
	if first.item.Source == nil || first.item.Source.System != "dexter" {
		t.Fatalf("expected dexter source system")
	}
	if first.item.Data["event_type"] != "earnings" {
		t.Fatalf("expected event_type earnings, got %#v", first.item.Data["event_type"])
	}

	second := store.calls[1]
	if second.bank != "signals" {
		t.Fatalf("expected signals bank, got %q", second.bank)
	}
	if second.item.Type != "signal" {
		t.Fatalf("expected signal type, got %q", second.item.Type)
	}
	if second.item.Data["event_type"] != "price_gap" {
		t.Fatalf("expected event_type price_gap, got %#v", second.item.Data["event_type"])
	}
	if !hasTag(second.item.Tags, "bookmarked") {
		t.Fatalf("expected bookmarked tag, got %#v", second.item.Tags)
	}
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if tag == target {
			return true
		}
	}
	return false
}
