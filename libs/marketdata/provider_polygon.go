package marketdata

import (
	"context"
	"fmt"
	"strconv"
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

	// Check if we got valid data (Results is a value, not pointer - check price)
	if resp.Results.Price == 0 {
		return nil, ErrNoData
	}

	// Also get snapshot for bid/ask
	snapshotParams := &models.GetTickerSnapshotParams{
		Ticker:     symbol,
		Locale:     models.US,
		MarketType: models.Stocks,
	}
	snapshot, err := p.client.GetTickerSnapshot(ctx, snapshotParams)
	if err != nil {
		// Snapshot failed but we can still return price from last trade
		return &Quote{
			Symbol:    symbol,
			Price:     resp.Results.Price,
			Timestamp: time.Time(resp.Results.Timestamp),
			Exchange:  strconv.FormatInt(int64(resp.Results.Exchange), 10),
		}, nil
	}

	quote := &Quote{
		Symbol:    symbol,
		Price:     resp.Results.Price,
		Timestamp: time.Time(resp.Results.Timestamp),
		Exchange:  strconv.FormatInt(int64(resp.Results.Exchange), 10),
	}

	// Access Snapshot field, not Ticker
	quote.Bid = snapshot.Snapshot.LastQuote.BidPrice
	quote.Ask = snapshot.Snapshot.LastQuote.AskPrice
	quote.BidSize = int64(snapshot.Snapshot.LastQuote.BidSize)
	quote.AskSize = int64(snapshot.Snapshot.LastQuote.AskSize)
	quote.Volume = int64(snapshot.Snapshot.Day.Volume)

	return quote, nil
}

// GetCandles fetches historical OHLCV data
func (p *PolygonProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Convert timeframe to Polygon format
	multiplier := 1
	var timespan models.Timespan

	switch timeframe {
	case Timeframe1Min:
		multiplier, timespan = 1, models.Minute
	case Timeframe5Min:
		multiplier, timespan = 5, models.Minute
	case Timeframe15Min:
		multiplier, timespan = 15, models.Minute
	case Timeframe1Hour:
		multiplier, timespan = 1, models.Hour
	case Timeframe1Day:
		multiplier, timespan = 1, models.Day
	case Timeframe1Week:
		multiplier, timespan = 1, models.Week
	default:
		return nil, ErrInvalidTimeframe
	}

	// Calculate date range
	to := time.Now()
	from := to.AddDate(0, 0, -limit)

	params := models.ListAggsParams{
		Ticker:     symbol,
		Multiplier: multiplier,
		Timespan:   timespan,
		From:       models.Millis(from),
		To:         models.Millis(to),
	}.WithLimit(limit)

	iter := p.client.ListAggs(ctx, params)

	candles := make([]Candle, 0, limit)
	for iter.Next() {
		agg := iter.Item()
		candle := Candle{
			Symbol:    symbol,
			Timestamp: time.Time(agg.Timestamp),
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

// Close cleans up provider resources
func (p *PolygonProvider) Close() error {
	// Polygon REST client doesn't need explicit cleanup
	return nil
}
