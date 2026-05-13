package instruments

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCatalogEvaluatesApprovedETF(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "etf-instruments.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	result := catalog.Evaluate("spy", "paper")
	if !result.Allowed {
		t.Fatalf("expected SPY to be allowed in paper mode, got %#v", result)
	}
	if result.Symbol != "SPY" {
		t.Fatalf("expected normalized symbol SPY, got %q", result.Symbol)
	}
	if result.CatalogVersion == "" || result.CatalogHash == "" {
		t.Fatalf("expected catalog version and hash in result, got %#v", result)
	}
}

func TestEvaluateRejectsExcludedAndLiveETFs(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "etf-instruments.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	excluded := catalog.Evaluate("TQQQ", "paper")
	if excluded.Allowed {
		t.Fatalf("expected TQQQ to be rejected, got %#v", excluded)
	}
	if excluded.ReasonCode != ReasonExcludedClass {
		t.Fatalf("expected excluded class reason, got %q", excluded.ReasonCode)
	}

	live := catalog.Evaluate("SPY", "live")
	if live.Allowed {
		t.Fatalf("expected SPY to be rejected in live mode, got %#v", live)
	}
	if live.ReasonCode != ReasonModeNotAllowed {
		t.Fatalf("expected mode rejection, got %q", live.ReasonCode)
	}
}

func TestEvaluateRejectsQuoteAndSessionFailures(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "etf-instruments.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	checkedAt := time.Date(2026, 5, 13, 15, 0, 0, 0, time.UTC)

	stale := catalog.EvaluateSubmission("SPY", "paper", SubmissionContext{
		Now:          checkedAt,
		QuoteTime:    checkedAt.Add(-61 * time.Second),
		Bid:          100,
		Ask:          100.05,
		BidSize:      10,
		AskSize:      10,
		HasStopLoss:  true,
		FlattenByEOD: true,
	})
	if stale.Allowed {
		t.Fatalf("expected stale quote rejection, got %#v", stale)
	}
	if stale.ReasonCode != ReasonQuoteStale {
		t.Fatalf("expected stale quote reason, got %q", stale.ReasonCode)
	}

	wide := catalog.EvaluateSubmission("SPY", "paper", SubmissionContext{
		Now:          checkedAt,
		QuoteTime:    checkedAt,
		Bid:          100,
		Ask:          100.20,
		BidSize:      10,
		AskSize:      10,
		HasStopLoss:  true,
		FlattenByEOD: true,
	})
	if wide.Allowed {
		t.Fatalf("expected wide spread rejection, got %#v", wide)
	}
	if wide.ReasonCode != ReasonSpreadTooWide {
		t.Fatalf("expected spread reason, got %q", wide.ReasonCode)
	}

	afterHours := catalog.EvaluateSubmission("SPY", "paper", SubmissionContext{
		Now:          time.Date(2026, 5, 13, 21, 0, 0, 0, time.UTC),
		QuoteTime:    time.Date(2026, 5, 13, 21, 0, 0, 0, time.UTC),
		Bid:          100,
		Ask:          100.05,
		BidSize:      10,
		AskSize:      10,
		HasStopLoss:  true,
		FlattenByEOD: true,
	})
	if afterHours.Allowed {
		t.Fatalf("expected RTH rejection, got %#v", afterHours)
	}
	if afterHours.ReasonCode != ReasonOutsideSession {
		t.Fatalf("expected session reason, got %q", afterHours.ReasonCode)
	}
}
