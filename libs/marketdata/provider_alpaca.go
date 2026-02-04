package marketdata

import (
	"context"
	"fmt"
	"time"

	"jax-trading-assistant/libs/resilience"

	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
)

// AlpacaProvider implements the Provider interface for Alpaca Market Data
type AlpacaProvider struct {
	client         *marketdata.Client
	config         ProviderConfig
	circuitBreaker *resilience.CircuitBreaker
}

// NewAlpacaProvider creates a new Alpaca provider
func NewAlpacaProvider(config ProviderConfig) (*AlpacaProvider, error) {
	// Determine base URL based on tier
	baseURL := "https://data.alpaca.markets"
	if config.Tier == "free" {
		baseURL = "https://data.alpaca.markets" // Free tier uses same endpoint
	}

	client := marketdata.NewClient(marketdata.ClientOpts{
		APIKey:    config.APIKey,
		APISecret: config.APISecret,
		BaseURL:   baseURL,
	})

	// Create circuit breaker for this provider
	cbConfig := resilience.DefaultConfig("alpaca-api")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &AlpacaProvider{
		client:         client,
		config:         config,
		circuitBreaker: cb,
	}, nil
}

// Name returns the provider name
func (p *AlpacaProvider) Name() string {
	return "alpaca"
}

// GetQuote fetches a real-time or delayed quote
func (p *AlpacaProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	result, err := p.circuitBreaker.ExecuteWithContext(ctx, func() (any, error) {
		snapshot, err := p.client.GetSnapshot(symbol, marketdata.GetSnapshotRequest{})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrProviderError, err)
		}

		if snapshot == nil || snapshot.LatestTrade == nil {
			return nil, ErrNoData
		}

		quote := &Quote{
			Symbol:    symbol,
			Price:     snapshot.LatestTrade.Price,
			Timestamp: snapshot.LatestTrade.Timestamp,
			Exchange:  snapshot.LatestTrade.Exchange,
		}

		if snapshot.LatestQuote != nil {
			quote.Bid = snapshot.LatestQuote.BidPrice
			quote.Ask = snapshot.LatestQuote.AskPrice
			quote.BidSize = int64(snapshot.LatestQuote.BidSize)
			quote.AskSize = int64(snapshot.LatestQuote.AskSize)
		}

		if snapshot.DailyBar != nil {
			quote.Volume = int64(snapshot.DailyBar.Volume)
		}

		return quote, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*Quote), nil
}

// GetCandles fetches historical OHLCV data
func (p *AlpacaProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Convert timeframe to Alpaca format
	var tf marketdata.TimeFrame

	switch timeframe {
	case Timeframe1Min:
		tf = marketdata.NewTimeFrame(1, marketdata.Min)
	case Timeframe5Min:
		tf = marketdata.NewTimeFrame(5, marketdata.Min)
	case Timeframe15Min:
		tf = marketdata.NewTimeFrame(15, marketdata.Min)
	case Timeframe1Hour:
		tf = marketdata.NewTimeFrame(1, marketdata.Hour)
	case Timeframe1Day:
		tf = marketdata.NewTimeFrame(1, marketdata.Day)
	case Timeframe1Week:
		tf = marketdata.NewTimeFrame(1, marketdata.Week)
	default:
		return nil, ErrInvalidTimeframe
	}

	// Calculate date range
	end := time.Now()
	start := end.AddDate(0, 0, -limit*2) // Request more than needed

	bars, err := p.client.GetBars(symbol, marketdata.GetBarsRequest{
		TimeFrame: tf,
		Start:     start,
		End:       end,
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderError, err)
	}

	if len(bars) == 0 {
		return nil, ErrNoData
	}

	candles := make([]Candle, 0, len(bars))
	for _, bar := range bars {
		candle := Candle{
			Symbol:    symbol,
			Timestamp: bar.Timestamp,
			Open:      bar.Open,
			High:      bar.High,
			Low:       bar.Low,
			Close:     bar.Close,
			Volume:    int64(bar.Volume),
			VWAP:      bar.VWAP,
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// GetTrades fetches recent trades
func (p *AlpacaProvider) GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error) {
	end := time.Now()
	start := end.Add(-1 * time.Hour) // Last hour of trades

	trades, err := p.client.GetTrades(symbol, marketdata.GetTradesRequest{
		Start:      start,
		End:        end,
		TotalLimit: limit,
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderError, err)
	}

	result := make([]Trade, 0, len(trades))
	for _, t := range trades {
		trade := Trade{
			Symbol:     symbol,
			Price:      t.Price,
			Size:       int64(t.Size),
			Timestamp:  t.Timestamp,
			Exchange:   t.Exchange,
			Conditions: t.Conditions,
		}
		result = append(result, trade)
	}

	return result, nil
}

// GetEarnings fetches earnings data (not available via Alpaca market data API)
func (p *AlpacaProvider) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	return nil, fmt.Errorf("earnings not available via alpaca market data API")
}

// StreamQuotes streams real-time quotes (placeholder for WebSocket implementation)
func (p *AlpacaProvider) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	// WebSocket streaming requires additional implementation
	return nil, fmt.Errorf("streaming not yet implemented for alpaca")
}

// HealthCheck verifies the provider is accessible
func (p *AlpacaProvider) HealthCheck(ctx context.Context) error {
	// Try to fetch SPY quote as health check
	_, err := p.GetQuote(ctx, "SPY")
	return err
}
