package marketdata

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/resilience"

	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
)

// PolygonProvider implements the Provider interface for Polygon.io
type PolygonProvider struct {
	client         *polygon.Client
	config         ProviderConfig
	circuitBreaker *resilience.CircuitBreaker
}

// NewPolygonProvider creates a new Polygon.io provider
func NewPolygonProvider(config ProviderConfig) (*PolygonProvider, error) {
	client := polygon.New(config.APIKey)
	baseURL := strings.TrimSpace(os.Getenv("POLYGON_BASE_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("MASSIVE_BASE_URL"))
	}
	if baseURL != "" {
		client.Client.HTTP.SetBaseURL(strings.TrimRight(baseURL, "/"))
	}

	// Create circuit breaker for this provider
	cbConfig := resilience.DefaultConfig("polygon-api")
	cb := resilience.NewCircuitBreaker(cbConfig)

	return &PolygonProvider{
		client:         client,
		config:         config,
		circuitBreaker: cb,
	}, nil
}

// Name returns the provider name
func (p *PolygonProvider) Name() string {
	return "polygon"
}

// GetQuote fetches a real-time or delayed quote
func (p *PolygonProvider) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	result, err := p.circuitBreaker.ExecuteWithContext(ctx, func() (any, error) {
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
	})

	if err != nil {
		return nil, err
	}
	return result.(*Quote), nil
}

// GetCandles fetches historical OHLCV data
func (p *PolygonProvider) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Convert timeframe to Polygon format
	var multiplier int
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
	if limit <= 0 {
		limit = 50
	}
	if limit > 50000 {
		limit = 50000
	}
	result, err := p.circuitBreaker.ExecuteWithContext(ctx, func() (any, error) {
		params := models.ListTradesParams{
			Ticker: symbol,
		}.WithSort(models.Timestamp).WithOrder(models.Desc).WithLimit(limit)
		iter := p.client.ListTrades(ctx, params)

		trades := make([]Trade, 0, limit)
		for iter.Next() {
			raw := iter.Item()
			trades = append(trades, Trade{
				Symbol:     symbol,
				Price:      raw.Price,
				Size:       int64(math.Round(raw.Size)),
				Timestamp:  time.Time(raw.SipTimestamp),
				Exchange:   strconv.Itoa(raw.Exchange),
				Conditions: intSliceToStringSlice(raw.Conditions),
			})
			if len(trades) >= limit {
				break
			}
		}
		if iter.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrProviderError, iter.Err())
		}
		if len(trades) == 0 {
			return nil, ErrNoData
		}
		return trades, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]Trade), nil
}

// GetEarnings fetches earnings data (placeholder - requires financials API)
func (p *PolygonProvider) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	if limit <= 0 {
		limit = 4
	}
	if limit > 100 {
		limit = 100
	}
	result, err := p.circuitBreaker.ExecuteWithContext(ctx, func() (any, error) {
		params := models.ListStockFinancialsParams{}.
			WithTicker(symbol).
			WithOrder(models.Desc).
			WithLimit(limit)
		iter := p.client.VX.ListStockFinancials(ctx, params)

		earnings := make([]Earnings, 0, limit)
		for iter.Next() {
			item := iter.Item()
			reportDate, err := time.Parse("2006-01-02", item.FilingDate)
			if err != nil {
				reportDate = time.Time{}
			}
			eps := extractFinancialValue(item.Financials, "income_statement", "basic_earnings_per_share")
			epsEstimate := extractFinancialValue(item.Financials, "income_statement", "diluted_earnings_per_share")
			revenue := extractFinancialValue(item.Financials, "income_statement", "revenues")
			revenueEstimate := extractFinancialValue(item.Financials, "income_statement", "revenue")
			year, _ := strconv.Atoi(item.FiscalYear)
			earnings = append(earnings, Earnings{
				Symbol:          symbol,
				FiscalQuarter:   item.FiscalPeriod,
				FiscalYear:      year,
				ReportDate:      reportDate,
				EPS:             eps,
				EPSEstimate:     epsEstimate,
				Revenue:         revenue,
				RevenueEstimate: revenueEstimate,
			})
			if len(earnings) >= limit {
				break
			}
		}
		if iter.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrProviderError, iter.Err())
		}
		if len(earnings) == 0 {
			return nil, ErrNoData
		}
		return earnings, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]Earnings), nil
}

// StreamQuotes streams real-time quotes (requires WebSocket - placeholder)
func (p *PolygonProvider) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("%w: at least one symbol required", ErrInvalidSymbol)
	}
	updates := make(chan StreamUpdate, 128)
	last := make(map[string]float64, len(symbols))
	interval := streamPollIntervalByTier(p.config.Tier)

	go func() {
		defer close(updates)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		poll := func() {
			for _, symbol := range symbols {
				symbol = strings.ToUpper(strings.TrimSpace(symbol))
				if symbol == "" {
					continue
				}
				quote, err := p.GetQuote(ctx, symbol)
				if err != nil {
					continue
				}
				if quote.Price <= 0 || quote.Price == last[symbol] {
					continue
				}
				last[symbol] = quote.Price
				select {
				case updates <- StreamUpdate{
					Type:      "quote",
					Quote:     quote,
					Timestamp: quote.Timestamp,
				}:
				case <-ctx.Done():
					return
				}
			}
		}

		poll()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	}()

	return updates, nil
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

func streamPollIntervalByTier(tier string) time.Duration {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "free":
		return 20 * time.Second
	case "starter":
		return 5 * time.Second
	default:
		return 2 * time.Second
	}
}

func intSliceToStringSlice(values []int32) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, strconv.Itoa(int(v)))
	}
	return out
}

func extractFinancialValue(financials map[string]models.Financial, section, metric string) float64 {
	sec, ok := financials[section]
	if !ok {
		return 0
	}
	if value, ok := sec[metric]; ok {
		return value.Value
	}
	return 0
}
