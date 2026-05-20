package etfnews

import (
	"testing"
)

func TestClassifyMarketPanic(t *testing.T) {
	c, ok := Classify("Market selloff accelerates on recession fears", "US equities in risk-off crash mode")
	if !ok {
		t.Fatal("expected classification for market panic text")
	}
	if c.Class != "market_panic" {
		t.Fatalf("class = %q, want market_panic", c.Class)
	}
	if c.Sentiment != "bearish" {
		t.Fatalf("sentiment = %q, want bearish", c.Sentiment)
	}
	assertContains(t, c.Symbols, "SPY", "QQQ", "DIA", "IWM")
}

func TestClassifySemiconductorPositive(t *testing.T) {
	c, ok := Classify("NVIDIA smashes earnings — semiconductor sector rallies", "Chip stocks surge on AI accelerator demand")
	if !ok {
		t.Fatal("expected classification for semiconductor text")
	}
	if c.Class != "sector_news" {
		t.Fatalf("class = %q, want sector_news", c.Class)
	}
	if c.Sector != "semiconductors" {
		t.Fatalf("sector = %q, want semiconductors", c.Sector)
	}
	if c.Sentiment != "positive" {
		t.Fatalf("sentiment = %q, want positive", c.Sentiment)
	}
	assertContains(t, c.Symbols, "SMH", "SOXX")
}

func TestClassifyRatesInflation(t *testing.T) {
	c, ok := Classify("Fed signals rate cut at next FOMC meeting", "Treasury yields drop on dovish Federal Reserve outlook")
	if !ok {
		t.Fatal("expected classification for rates/inflation text")
	}
	if c.Class != "rates_inflation" {
		t.Fatalf("class = %q, want rates_inflation", c.Class)
	}
	assertContains(t, c.Symbols, "TLT", "GLD", "SPY", "QQQ", "XLF")
}

func TestClassifyRatesInflationHawkish(t *testing.T) {
	c, ok := Classify("CPI hotter than expected — rate hike back on the table", "Inflation hot, Fed forced to tighten")
	if !ok {
		t.Fatal("expected classification for hawkish rates text")
	}
	if c.Class != "rates_inflation" {
		t.Fatalf("class = %q, want rates_inflation", c.Class)
	}
	if c.Sentiment != "negative" {
		t.Fatalf("sentiment = %q, want negative", c.Sentiment)
	}
}

func TestClassifyEnergyNews(t *testing.T) {
	c, ok := Classify("OPEC cuts production", "Oil supply tightens after OPEC decision")
	if !ok {
		t.Fatal("expected classification for energy text")
	}
	if c.Class != "sector_news" {
		t.Fatalf("class = %q, want sector_news", c.Class)
	}
	if c.Sector != "energy" {
		t.Fatalf("sector = %q, want energy", c.Sector)
	}
	assertContains(t, c.Symbols, "XLE")
}

func TestClassifyBankingStress(t *testing.T) {
	c, ok := Classify("Regional lender fails — banking sector under pressure", "Credit stress spreads across banks")
	if !ok {
		t.Fatal("expected classification for banking stress text")
	}
	if c.Class != "sector_news" {
		t.Fatalf("class = %q, want sector_news", c.Class)
	}
	if c.Sector != "financials" {
		t.Fatalf("sector = %q, want financials", c.Sector)
	}
	assertContains(t, c.Symbols, "XLF")
}

func TestClassifyUnrelatedSingleStock(t *testing.T) {
	_, ok := Classify("Apple launches new iPhone 17", "Cupertino announces incremental hardware refresh")
	if ok {
		t.Fatal("expected no classification for unrelated single-stock news")
	}
}

func TestClassifyEmptyText(t *testing.T) {
	_, ok := Classify("", "")
	if ok {
		t.Fatal("expected no classification for empty text")
	}
}

// assertContains fails if any of the expected symbols is absent from symbols.
func assertContains(t *testing.T, symbols []string, expected ...string) {
	t.Helper()
	set := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		set[s] = true
	}
	for _, e := range expected {
		if !set[e] {
			t.Fatalf("expected symbol %s in %v", e, symbols)
		}
	}
}
