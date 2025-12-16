package adapters

import (
	"context"

	"jax-trading-assistant/internal/app"
	"jax-trading-assistant/internal/infra/utcp"
)

type UTCPMarketAdapter struct {
	market *utcp.MarketDataService
}

func NewUTCPMarketAdapter(market *utcp.MarketDataService) *UTCPMarketAdapter {
	return &UTCPMarketAdapter{market: market}
}

func (a *UTCPMarketAdapter) GetDailyCandles(ctx context.Context, symbol string, limit int) ([]app.Candle, error) {
	out, err := a.market.GetCandles(ctx, utcp.GetCandlesInput{
		Symbol:    symbol,
		Timeframe: utcp.DefaultCandleTimeframe,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]app.Candle, 0, len(out.Candles))
	for _, c := range out.Candles {
		candles = append(candles, app.Candle{
			TS:     c.TS,
			Open:   c.Open,
			High:   c.High,
			Low:    c.Low,
			Close:  c.Close,
			Volume: c.Volume,
		})
	}

	return candles, nil
}
