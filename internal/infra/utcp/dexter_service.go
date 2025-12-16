package utcp

import (
	"context"
)

type DexterService struct {
	client Client
}

func NewDexterService(c Client) *DexterService {
	return &DexterService{client: c}
}

func (s *DexterService) ResearchCompany(ctx context.Context, ticker string, questions []string) (*ResearchBundle, error) {
	input := map[string]any{
		"ticker":    ticker,
		"questions": questions,
	}
	var out ResearchBundle
	if err := s.client.CallTool(ctx, DexterProviderID, ToolDexterResearchCompany, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *DexterService) CompareCompanies(ctx context.Context, tickers []string, focus string) (*ComparisonResult, error) {
	input := map[string]any{
		"tickers": tickers,
		"focus":   focus,
	}
	var out ComparisonResult
	if err := s.client.CallTool(ctx, DexterProviderID, ToolDexterCompareCompanies, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
