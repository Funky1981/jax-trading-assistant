package macroevents

import (
	"testing"
)

func TestValidateAndNormalizeETFsAcceptsKnownSymbols(t *testing.T) {
	mappings, err := ValidateAndNormalizeETFs([]string{"qqq", "SPY", "TLT"})
	if err != nil {
		t.Fatalf("ValidateAndNormalizeETFs returned error: %v", err)
	}

	got := mappingSymbols(mappings)
	want := []string{"QQQ", "SPY", "TLT"}
	if !equalStrings(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
	if mappings[0].Theme != "growth/technology" {
		t.Fatalf("QQQ theme = %q, want growth/technology", mappings[0].Theme)
	}
}

func TestValidateAndNormalizeETFsRejectsUnknownSymbol(t *testing.T) {
	_, err := ValidateAndNormalizeETFs([]string{"QQQ", "NOTREAL"})
	if err == nil {
		t.Fatal("expected unknown symbol to be rejected")
	}
}

func TestValidateAndNormalizeETFsRejectsNonAllowedSymbols(t *testing.T) {
	_, err := ValidateAndNormalizeETFs([]string{"AAPL"})
	if err == nil {
		t.Fatal("expected single-stock symbol to be rejected")
	}
}

func TestValidateAndNormalizeETFsRejectsEmptyList(t *testing.T) {
	_, err := ValidateAndNormalizeETFs([]string{" ", ""})
	if err == nil {
		t.Fatal("expected empty ETF list to be rejected")
	}
}

func TestValidateAndNormalizeETFsDedupesAndUppercasesSymbols(t *testing.T) {
	mappings, err := ValidateAndNormalizeETFs([]string{"qqq", "QQQ", "spy"})
	if err != nil {
		t.Fatalf("ValidateAndNormalizeETFs returned error: %v", err)
	}

	got := mappingSymbols(mappings)
	want := []string{"QQQ", "SPY"}
	if !equalStrings(got, want) {
		t.Fatalf("symbols = %#v, want %#v", got, want)
	}
}

func mappingSymbols(mappings []ETFMapping) []string {
	out := make([]string, 0, len(mappings))
	for _, mapping := range mappings {
		out = append(out, mapping.Symbol)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
