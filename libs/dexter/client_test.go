package dexter

import (
	"context"
	"testing"
)

func TestMockClient_ResearchCompany(t *testing.T) {
	mock := &MockClient{
		ResearchCompanyFunc: func(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error) {
			if input.Ticker != "AAPL" {
				t.Errorf("expected ticker AAPL, got %s", input.Ticker)
			}
			return ResearchCompanyOutput{
				Ticker:    input.Ticker,
				Summary:   "Mock research summary",
				KeyPoints: []string{"Point 1", "Point 2"},
				Metrics:   map[string]interface{}{"pe_ratio": 28.5},
			}, nil
		},
	}

	output, err := mock.ResearchCompany(context.Background(), ResearchCompanyInput{
		Ticker:    "AAPL",
		Questions: []string{"What is the revenue growth?"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Ticker != "AAPL" {
		t.Errorf("expected ticker AAPL, got %s", output.Ticker)
	}

	if len(output.KeyPoints) != 2 {
		t.Errorf("expected 2 key points, got %d", len(output.KeyPoints))
	}
}

func TestMockClient_CompareCompanies(t *testing.T) {
	mock := &MockClient{
		CompareCompaniesFunc: func(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error) {
			return CompareCompaniesOutput{
				ComparisonAxis: input.Focus,
				Items: []ComparisonItem{
					{Ticker: "AAPL", Thesis: "Strong ecosystem", Notes: []string{"Note 1"}},
					{Ticker: "MSFT", Thesis: "Cloud leader", Notes: []string{"Note 2"}},
				},
			}, nil
		},
	}

	output, err := mock.CompareCompanies(context.Background(), CompareCompaniesInput{
		Tickers: []string{"AAPL", "MSFT"},
		Focus:   "Cloud revenue",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ComparisonAxis != "Cloud revenue" {
		t.Errorf("expected focus 'Cloud revenue', got %s", output.ComparisonAxis)
	}

	if len(output.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(output.Items))
	}
}
