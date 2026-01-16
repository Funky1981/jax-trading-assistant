package testing

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func TestInMemoryMemoryStore_RetainAndRecall(t *testing.T) {
	store := NewInMemoryMemoryStore()

	_, err := store.Retain(context.Background(), "trade_decisions", contracts.MemoryItem{
		TS:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:    "decision",
		Symbol:  "AAPL",
		Tags:    []string{"earnings", "gap"},
		Summary: "AAPL gap-up after earnings, candidate long.",
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}

	got, err := store.Recall(context.Background(), "trade_decisions", contracts.MemoryQuery{
		Symbol: "AAPL",
		Tags:   []string{"gap"},
		Q:      "earnings",
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Type != "decision" {
		t.Fatalf("unexpected: %#v", got[0])
	}
}

func TestInMemoryMemoryStore_Retain_Validation(t *testing.T) {
	store := NewInMemoryMemoryStore()

	_, err := store.Retain(context.Background(), "", contracts.MemoryItem{Type: "decision", Summary: "x"})
	if err == nil {
		t.Fatalf("expected error for empty bank")
	}

	_, err = store.Retain(context.Background(), "bank", contracts.MemoryItem{Summary: "x"})
	if err == nil {
		t.Fatalf("expected error for empty type")
	}

	_, err = store.Retain(context.Background(), "bank", contracts.MemoryItem{Type: "decision"})
	if err == nil {
		t.Fatalf("expected error for empty summary")
	}
}

func TestInMemoryMemoryStore_Reflect_RequiresQuery(t *testing.T) {
	store := NewInMemoryMemoryStore()

	_, err := store.Reflect(context.Background(), "trade_decisions", contracts.ReflectionParams{})
	if err == nil {
		t.Fatalf("expected error for missing query")
	}
}
