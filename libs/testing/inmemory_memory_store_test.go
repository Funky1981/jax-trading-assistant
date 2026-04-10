package testing

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func TestInMemoryMemoryStore_RetainAndRecall(t *testing.T) {
	store := NewInMemoryMemoryStore()

	_, err := store.Retain(context.Background(), "trades", contracts.MemoryItem{
		TS:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:    "decision",
		Symbol:  "AAPL",
		Tags:    []string{"earnings", "gap"},
		Summary: "AAPL gap-up after earnings, candidate long.",
		Data:    map[string]any{"confidence": 0.8},
		Source:  &contracts.MemorySource{System: "test"},
	})
	if err != nil {
		t.Fatalf("retain: %v", err)
	}

	got, err := store.Recall(context.Background(), "trades", contracts.MemoryQuery{
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

	_, err := store.Retain(context.Background(), "", contracts.MemoryItem{
		Type:    "decision",
		Summary: "x",
		Data:    map[string]any{"ok": true},
		Source:  &contracts.MemorySource{System: "test"},
	})
	if err == nil {
		t.Fatalf("expected error for empty bank")
	}

	_, err = store.Retain(context.Background(), "trades", contracts.MemoryItem{
		Summary: "x",
		Data:    map[string]any{"ok": true},
		Source:  &contracts.MemorySource{System: "test"},
	})
	if err == nil {
		t.Fatalf("expected error for empty type")
	}

	_, err = store.Retain(context.Background(), "trades", contracts.MemoryItem{
		Type:   "decision",
		Data:   map[string]any{"ok": true},
		Source: &contracts.MemorySource{System: "test"},
	})
	if err == nil {
		t.Fatalf("expected error for empty summary")
	}
}

func TestInMemoryMemoryStore_Reflect_RequiresQuery(t *testing.T) {
	store := NewInMemoryMemoryStore()

	_, err := store.Reflect(context.Background(), "trades", contracts.ReflectionParams{})
	if err == nil {
		t.Fatalf("expected error for missing query")
	}
}
