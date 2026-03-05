package dexter

import (
	"context"
	"strings"
)

// MockClient implements a mock Dexter client for testing
type MockClient struct {
	ResearchCompanyFunc  func(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error)
	CompareCompaniesFunc func(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error)
	HealthFunc           func(ctx context.Context) error
}

func (m *MockClient) ResearchCompany(ctx context.Context, input ResearchCompanyInput) (ResearchCompanyOutput, error) {
	if m.ResearchCompanyFunc != nil {
		return m.ResearchCompanyFunc(ctx, input)
	}
	ticker := strings.ToUpper(strings.TrimSpace(input.Ticker))
	if ticker == "" {
		ticker = "UNKNOWN"
	}
	return ResearchCompanyOutput{
		Ticker:    ticker,
		Summary:   "Mock research summary for " + ticker,
		KeyPoints: []string{"Deterministic mock output", "No external service call made"},
		Metrics: map[string]interface{}{
			"confidence": 0.5,
			"source":     "mock",
		},
		RawMarkdown: "# Mock Research\n\nNo external Dexter service configured.",
	}, nil
}

func (m *MockClient) CompareCompanies(ctx context.Context, input CompareCompaniesInput) (CompareCompaniesOutput, error) {
	if m.CompareCompaniesFunc != nil {
		return m.CompareCompaniesFunc(ctx, input)
	}
	items := make([]ComparisonItem, 0, len(input.Tickers))
	for _, ticker := range input.Tickers {
		normalized := strings.ToUpper(strings.TrimSpace(ticker))
		if normalized == "" {
			continue
		}
		items = append(items, ComparisonItem{
			Ticker: normalized,
			Thesis: "Mock thesis for " + normalized,
			Notes:  []string{"deterministic mock comparison"},
		})
	}
	focus := strings.TrimSpace(input.Focus)
	if focus == "" {
		focus = "general"
	}
	return CompareCompaniesOutput{
		ComparisonAxis: focus,
		Items:          items,
	}, nil
}

func (m *MockClient) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}
