package utcp

import "context"

type MarketDataService struct {
	client Client
}

func NewMarketDataService(c Client) *MarketDataService {
	return &MarketDataService{client: c}
}

func (s *MarketDataService) GetQuote(ctx context.Context, symbol string) (GetQuoteOutput, error) {
	var out GetQuoteOutput
	if err := s.client.CallTool(ctx, MarketDataProviderID, ToolMarketGetQuote, GetQuoteInput{Symbol: symbol}, &out); err != nil {
		return GetQuoteOutput{}, err
	}
	return out, nil
}

func (s *MarketDataService) GetCandles(ctx context.Context, in GetCandlesInput) (GetCandlesOutput, error) {
	if in.Timeframe == "" {
		in.Timeframe = DefaultCandleTimeframe
	}
	if in.Limit == 0 {
		in.Limit = 200
	}
	var out GetCandlesOutput
	if err := s.client.CallTool(ctx, MarketDataProviderID, ToolMarketGetCandles, in, &out); err != nil {
		return GetCandlesOutput{}, err
	}
	return out, nil
}

func (s *MarketDataService) GetEarnings(ctx context.Context, in GetEarningsInput) (GetEarningsOutput, error) {
	if in.Limit == 0 {
		in.Limit = 8
	}
	var out GetEarningsOutput
	if err := s.client.CallTool(ctx, MarketDataProviderID, ToolMarketGetEarnings, in, &out); err != nil {
		return GetEarningsOutput{}, err
	}
	return out, nil
}
