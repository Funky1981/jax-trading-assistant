package instruments

import (
	"encoding/json"
	"os"
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

func TestPhaseOneDefaultsUseOnlyApprovedETFUniverse(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "etf-instruments.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	configs := []struct {
		name string
		path string
		key  string
	}{
		{name: "core defaults", path: filepath.Join("..", "..", "..", "config", "jax-core.json"), key: "defaultSymbols"},
		{name: "market ingest", path: filepath.Join("..", "..", "..", "config", "jax-market.json"), key: "symbols"},
		{name: "research ingest", path: filepath.Join("..", "..", "..", "config", "jax-ingest.json"), key: "symbols"},
		{name: "ib ingest", path: filepath.Join("..", "..", "..", "config", "jax-ingest-ib.json"), key: "symbols"},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			symbols := symbolsFromConfig(t, cfg.path, cfg.key)
			if len(symbols) == 0 {
				t.Fatalf("%s has no %s defaults", cfg.path, cfg.key)
			}
			for _, symbol := range symbols {
				result := catalog.Evaluate(symbol, "paper")
				if !result.Allowed {
					t.Fatalf("%s contains non-approved phase-one ETF symbol %q: %s", cfg.path, symbol, result.Reason)
				}
			}
		})
	}
}

func TestPhaseOneStrategyInstanceDefaultsUseOnlyApprovedETFUniverse(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "etf-instruments.json"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join("..", "..", "..", "config", "strategy-instances"))
	if err != nil {
		t.Fatalf("read strategy instances: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join("..", "..", "..", "config", "strategy-instances", entry.Name())
		t.Run(entry.Name(), func(t *testing.T) {
			symbols := symbolsFromNestedConfig(t, path, "config", "symbols")
			if len(symbols) == 0 {
				t.Fatalf("%s has no config.symbols defaults", path)
			}
			for _, symbol := range symbols {
				result := catalog.Evaluate(symbol, "paper")
				if !result.Allowed {
					t.Fatalf("%s contains non-approved phase-one ETF symbol %q: %s", path, symbol, result.Reason)
				}
			}
		})
	}
}

func symbolsFromConfig(t *testing.T, path string, key string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	values, ok := raw[key].([]any)
	if !ok {
		t.Fatalf("%s missing array key %q", path, key)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		symbol, ok := value.(string)
		if !ok {
			t.Fatalf("%s key %q contains non-string value %#v", path, key, value)
		}
		out = append(out, symbol)
	}
	return out
}

func symbolsFromNestedConfig(t *testing.T, path string, objectKey string, symbolsKey string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	nested, ok := raw[objectKey].(map[string]any)
	if !ok {
		t.Fatalf("%s missing object key %q", path, objectKey)
	}
	values, ok := nested[symbolsKey].([]any)
	if !ok {
		t.Fatalf("%s missing array key %q.%q", path, objectKey, symbolsKey)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		symbol, ok := value.(string)
		if !ok {
			t.Fatalf("%s key %q.%q contains non-string value %#v", path, objectKey, symbolsKey, value)
		}
		out = append(out, symbol)
	}
	return out
}
