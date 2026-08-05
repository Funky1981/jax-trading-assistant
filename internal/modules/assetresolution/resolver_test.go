package assetresolution

import (
	"reflect"
	"testing"
	"time"
)

func testResolver(t *testing.T) Resolver {
	t.Helper()
	rules, err := LoadRuleset("../../../config/event-asset-resolution-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return Resolver{Rules: rules}
}

func input(headline string) Input {
	return Input{EventID: "e1", Headline: headline, Summary: "", SourceName: "CNBC Top News", SourceURL: "https://www.cnbc.com/story", EventType: "unknown", PublicationAt: time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC), ReceiptAt: time.Date(2026, 7, 29, 12, 1, 0, 0, time.UTC)}
}

func TestResolverExactTickerCompanyAliasAndDeterministicRepeat(t *testing.T) {
	r := testResolver(t)
	exact := input("anything")
	exact.ExplicitSymbols = []string{"AAPL"}
	if got := r.Resolve(exact); got.Symbol != "AAPL" || got.ConfidenceClass != "exact" {
		t.Fatalf("exact=%+v", got)
	}
	company := input("Apple reports quarterly earnings")
	a, b := r.Resolve(company), r.Resolve(company)
	if a.Symbol != "AAPL" || a.Relationship != "direct" || !reflect.DeepEqual(a, b) {
		t.Fatalf("company repeat mismatch: %+v %+v", a, b)
	}
}

func TestResolverRejectsAmbiguousAndBroadTopicsWithoutFallback(t *testing.T) {
	r := testResolver(t)
	if got := r.Resolve(input("Amazon, Meta and Microsoft face investors")); got.Status != StatusAmbiguous || got.Symbol != "" {
		t.Fatalf("ambiguous=%+v", got)
	}
	if got := r.Resolve(input("Wildfires spread during a heatwave")); got.Status != StatusUnresolved || got.Symbol != "" {
		t.Fatalf("broad=%+v", got)
	}
	if got := r.Resolve(input("FIFA is embroiled in a controversy")); got.Symbol != "" {
		t.Fatalf("oil substring false positive=%+v", got)
	}
	caseStory := input("Singer will face murder trial")
	caseStory.Summary = "A Tesla was mentioned incidentally in the article."
	if got := r.Resolve(caseStory); got.Symbol != "" {
		t.Fatalf("summary-only issuer false positive=%+v", got)
	}
}

func TestResolverSectorMacroProxyAndDateSensitiveSymbol(t *testing.T) {
	r := testResolver(t)
	fed := input("Federal Reserve Board issues enforcement action")
	fed.SourceName = "Federal Reserve"
	fed.SourceURL = "https://www.federalreserve.gov/newsevents.htm"
	if got := r.Resolve(fed); got.Symbol != "TLT" || got.Relationship != "proxy" {
		t.Fatalf("fed=%+v", got)
	}
	chips := input("Chip stocks shed more than $1 trillion")
	if got := r.Resolve(chips); got.Symbol != "SOXX" || got.Relationship != "proxy" {
		t.Fatalf("chips=%+v", got)
	}
	old := input("Meta reports earnings")
	old.PublicationAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := r.Resolve(old); got.Symbol == "META" {
		t.Fatalf("date-invalid META accepted: %+v", got)
	}
}

func TestProxyExposuresAndResolutionComeFromRuleset(t *testing.T) {
	r := testResolver(t)
	exposures, err := r.ProxyExposures()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"FEDERAL_RESERVE_OFFICIAL", "GOLD_NAMED_MARKET", "OIL_CATEGORY", "SEMICONDUCTOR_GROUP", "SP500_NAMED_INDEX", "US_RATES_CATEGORY"}
	if !reflect.DeepEqual(exposures, want) {
		t.Fatalf("exposures=%v, want %v", exposures, want)
	}
	got, ok := r.ResolveProxyExposure("OIL_CATEGORY")
	if !ok || got.Symbol != "XLE" || got.CanonicalEntity != "oil_category" || got.RulesetVersion != "event-asset-resolution-v1" {
		t.Fatalf("resolved exposure=%+v, ok=%v", got, ok)
	}
	if _, ok := r.ResolveProxyExposure("UNKNOWN_CATEGORY"); ok {
		t.Fatal("unknown exposure should remain unresolved")
	}
}

func TestProxyExposuresRejectInvalidPolicyEntries(t *testing.T) {
	for _, rule := range []ProxyRule{
		{Key: "NONE", Symbol: "SPY"},
		{Key: "bad exposure", Symbol: "SPY"},
		{Key: "VALID_EXPOSURE", Symbol: "bad ticker"},
	} {
		resolver := Resolver{Rules: Ruleset{Version: "test", Proxies: []ProxyRule{rule}}}
		if _, err := resolver.ProxyExposures(); err == nil {
			t.Fatalf("invalid proxy rule was accepted: %+v", rule)
		}
	}
}
