package marketdata

import (
	"context"
	"fmt"
	"time"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
)

// PolygonProvider implements the Provider interface for Polygon.io
type PolygonProvider struct {
	client *polygon.Client
	config ProviderConfig
}

// NewPolygonProvider creates a new Polygon.io provider
func NewPolygonProvider(config ProviderConfig) (*PolygonProvider, error) {
	client := polygon.New(config.APIKey)

	return &PolygonProvider{
		client: client,
		config: config,
	}, nil
}

// Name returns the provider name
func (p *PolygonProvider) Name() string {
	return "polygon"
}

// GetQuote fetches a real-time or delayed quote
func (p *PolygonProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	params := &models.GetLastTradeParams{
		Ticker: symbol,
	}

	resp, err := p.client.GetLastTrade(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderError, err)
	}

	if resp.Results == nil {
		return nil, ErrNoData
	}

	// Also get snapshot for bid/ask
	snapshotParams := &models.GetSnapshotTickerParams{
		Ticker: symbol,
	}
	snapshot, err := p.client.GetSnapshotTicker(ctx, snapshotParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderError, err)
	}

	quote := &Quote{
		Symbol:    symbol,
		Price:     resp.Results.Price,
		Timestamp: time.UnixMilli(resp.Results.SipTimestamp),
		Exchange:  resp.Results.Exchange,
	}

	if snapshot.Ticker != nil && snapshot.Ticker.LastQuote != nil {
		quote.Bid = snapshot.Ticker.LastQuote.BidPrice
		quote.Ask = snapshot.Ticker.LastQuote.AskPrice
		quote.BidSize = int64(snapshot.Ticker.LastQuote.BidSize)
		quote.AskSize = int64(snapshot.Ticker.LastQuote.AskSize)
	}

	if snapshot.Ticker != nil && snapshot.Ticker.Day != nil {
		quote.Volume = int64(snapshot.Ticker.Day.Volume)
	}

	return quote, nil
}

// GetCandles fetches historical OHLCV data
func (p *PolygonProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Convert timeframe to Polygon format
	multiplier := 1
	timespan := "day"

	switch timeframe {
	case Timeframe1Min:
		multiplier, timespan = 1, "minute"
	case Timeframe5Min:
		multiplier, timespan = 5, "minute"
	case Timeframe15Min:
		multiplier, timespan = 15, "minute"
	case Timeframe1Hour:
		multiplier, timespan = 1, "hour"
	case Timeframe1Day:
		multiplier, timespan = 1, "day"
	case Timeframe1Week:
		multiplier, timespan = 1, "week"
	default:
		return nil, ErrInvalidTimeframe
	}

	// Calculate date range
	to := time.Now()
	from := to.AddDate(0, 0, -limit)

	params := &models.ListAggregatesParams{
		Ticker:     symbol,
		Multiplier: multiplier,
		Timespan:   timespan,
		From:       models.Millis(from),
		To:         models.Millis(to),
		Limit:      int64(limit),
		Sort:       "asc",
	}

	iter := p.client.ListAggregates(ctx, params)

	candles := make([]Candle, 0, limit)
	for iter.Next() {
		agg := iter.Item()
		candle := Candle{
			Symbol:    symbol,
			Timestamp: time.UnixMilli(int64(agg.Timestamp)),
			Open:      agg.Open,
			High:      agg.High,
			Low:       agg.Low,
			Close:     agg.Close,
			Volume:    int64(agg.Volume),
			VWAP:      agg.VWAP,
		}
		candles = append(candles, candle)
	}

	if iter.Err() != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderError, iter.Err())
	}

	if len(candles) == 0 {
		return nil, ErrNoData
	}

	return candles, nil
}

// GetTrades fetches recent trades (not implemented for Polygon free tier)
func (p *PolygonProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	return nil, fmt.Errorf("trades not available on %s tier", p.config.Tier)
}

// GetEarnings fetches earnings data (placeholder - requires financials API)
func (p *PolygonProvider) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	// Polygon earnings require separate API tier
	// For now, return stub implementation
	return nil, fmt.Errorf("earnings not available on %s tier", p.config.Tier)
}

// StreamQuotes streams real-time quotes (requires WebSocket - placeholder)
func (p *PolygonProvider) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	// WebSocket streaming requires additional implementation
	// For now, return error
	return nil, fmt.Errorf("streaming not yet implemented for polygon")
}

// HealthCheck verifies the provider is accessible
func (p *PolygonProvider) HealthCheck(ctx context.Context) error {
	// Try to fetch SPY quote as health check
	_, err := p.GetQuote(ctx, "SPY")
	return err
}
