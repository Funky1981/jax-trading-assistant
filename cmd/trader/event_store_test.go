package main

import (
	"net/http/httptest"
	"testing"
)

func TestDeterministicEventIDStable(t *testing.T) {
	a := deterministicEventID("polygon", "news", "AAPL", "launches new product")
	b := deterministicEventID("polygon", "news", "AAPL", "launches new product")
	if a != b {
		t.Fatalf("deterministicEventID should be stable: %q vs %q", a, b)
	}
}

func TestNormalizeSymbols_PrimaryFirstDeduped(t *testing.T) {
	got := normalizeSymbols("aapl", []string{"MSFT", "aapl", "msft", "GOOG"})
	if len(got) != 3 {
		t.Fatalf("expected 3 symbols, got %d (%v)", len(got), got)
	}
	if got[0] != "AAPL" {
		t.Fatalf("expected primary symbol first, got %v", got)
	}
}

func TestBuildEventsFilter(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/events?kind=news&symbol=aapl&sourceId=polygon&search=iphone&from=2026-01-01&to=2026-01-31", nil)
	where, args := buildEventsFilter(req)
	if where == "" {
		t.Fatal("expected WHERE clause to be non-empty")
	}
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d", len(args))
	}
}
