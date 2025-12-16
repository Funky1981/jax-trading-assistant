package adapters

import (
	"context"

	"jax-trading-assistant/internal/domain"
	"jax-trading-assistant/internal/infra/utcp"
)

type UTCPDexterAdapter struct {
	dexter *utcp.DexterService
}

func NewUTCPDexterAdapter(dexter *utcp.DexterService) *UTCPDexterAdapter {
	return &UTCPDexterAdapter{dexter: dexter}
}

func (a *UTCPDexterAdapter) ResearchCompany(ctx context.Context, ticker string, questions []string) (*domain.ResearchBundle, error) {
	out, err := a.dexter.ResearchCompany(ctx, ticker, questions)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return &domain.ResearchBundle{
		Ticker:      out.Ticker,
		Summary:     out.Summary,
		KeyPoints:   out.KeyPoints,
		Metrics:     out.Metrics,
		RawMarkdown: out.RawMarkdown,
	}, nil
}
