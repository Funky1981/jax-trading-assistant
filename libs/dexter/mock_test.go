package dexter

import (
	"context"
	"testing"
)

func TestMockClient_DefaultResearchCompany(t *testing.T) {
	client := &MockClient{}
	got, err := client.ResearchCompany(context.Background(), ResearchCompanyInput{Ticker: "msft"})
	if err != nil {
		t.Fatalf("ResearchCompany returned error: %v", err)
	}
	if got.Ticker != "MSFT" {
		t.Fatalf("expected normalized ticker MSFT, got %q", got.Ticker)
	}
	if got.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestMockClient_DefaultCompareCompanies(t *testing.T) {
	client := &MockClient{}
	got, err := client.CompareCompanies(context.Background(), CompareCompaniesInput{
		Tickers: []string{"AAPL", "NVDA"},
		Focus:   "valuation",
	})
	if err != nil {
		t.Fatalf("CompareCompanies returned error: %v", err)
	}
	if got.ComparisonAxis != "valuation" {
		t.Fatalf("expected comparison axis valuation, got %q", got.ComparisonAxis)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}
}
