package marketdata

import (
	"context"
	"fmt"
	"log"
	"sort"
)

// Provider defines the interface for market data providers
type Provider interface {
	Name() string
	GetQuote(ctx context.Context, symbol string) (*Quote, error)
	GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]Trade, error)
	GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error)
	StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error)
	HealthCheck(ctx context.Context) error
}

// Client aggregates multiple providers with fallback and caching
type Client struct {
	providers []Provider
	cache     *Cache
	config    *Config
}

// NewClient creates a new market data client
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	client := &Client{
		providers: make([]Provider, 0, len(config.Providers)),
		config:    config,
	}

	// Initialize cache if enabled
	if config.Cache.Enabled {
		cache, err := NewCache(config.Cache)
		if err != nil {
			log.Printf("failed to initialize cache: %v", err)
		} else {
			client.cache = cache
		}
	}

	// Initialize providers in priority order
	for _, pc := range config.Providers {
		if !pc.Enabled {
			continue
		}

		var provider Provider
		var err error

		switch pc.Name {
		case ProviderPolygon:
			provider, err = NewPolygonProvider(pc)
		case ProviderAlpaca:
			provider, err = NewAlpacaProvider(pc)
		case ProviderIB:
			provider, err = NewIBProvider(pc)
		case ProviderIBBridge:
			provider, err = NewIBBridgeProvider(pc)
		default:
			log.Printf("unknown provider: %s", pc.Name)
			continue
		}

		if err != nil {
			log.Printf("failed to initialize %s provider: %v", pc.Name, err)
			continue
		}

		client.providers = append(client.providers, provider)
	}

	// Sort providers by priority
	sort.Slice(client.providers, func(i, j int) bool {
		return getPriority(client.providers[i].Name(), config) < getPriority(client.providers[j].Name(), config)
	})

	if len(client.providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	log.Printf("initialized market data client with %d provider(s)", len(client.providers))
	return client, nil
}

// GetQuote fetches a quote with provider fallback and caching
func (c *Client) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	// Try cache first
	if c.cache != nil {
		if quote, err := c.cache.GetQuote(ctx, symbol); err == nil && quote != nil {
			return quote, nil
		}
	}

	// Try providers in priority order
	var lastErr error
	for _, provider := range c.providers {
		quote, err := provider.GetQuote(ctx, symbol)
		if err == nil {
			// Cache successful result
			if c.cache != nil {
				_ = c.cache.SetQuote(ctx, quote)
			}
			return quote, nil
		}
		lastErr = err
		log.Printf("%s provider failed for %s: %v", provider.Name(), symbol, err)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoProviderAvailable, lastErr)
	}
	return nil, ErrNoProviderAvailable
}

// GetCandles fetches historical candles with provider fallback
func (c *Client) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	// Try cache first
	if c.cache != nil {
		if candles, err := c.cache.GetCandles(ctx, symbol, timeframe, limit); err == nil && len(candles) > 0 {
			return candles, nil
		}
	}

	// Try providers in priority order
	var lastErr error
	for _, provider := range c.providers {
		candles, err := provider.GetCandles(ctx, symbol, timeframe, limit)
		if err == nil && len(candles) > 0 {
			// Cache successful result
			if c.cache != nil {
				_ = c.cache.SetCandles(ctx, symbol, timeframe, candles)
			}
			return candles, nil
		}
		lastErr = err
		log.Printf("%s provider failed for %s candles: %v", provider.Name(), symbol, err)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoProviderAvailable, lastErr)
	}
	return nil, ErrNoProviderAvailable
}

// GetEarnings fetches earnings data with provider fallback
func (c *Client) GetEarnings(ctx context.Context, symbol string, limit int) ([]Earnings, error) {
	var lastErr error
	for _, provider := range c.providers {
		earnings, err := provider.GetEarnings(ctx, symbol, limit)
		if err == nil {
			return earnings, nil
		}
		lastErr = err
		log.Printf("%s provider failed for %s earnings: %v", provider.Name(), symbol, err)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoProviderAvailable, lastErr)
	}
	return nil, ErrNoProviderAvailable
}

// StreamQuotes streams real-time quotes from the highest priority provider
func (c *Client) StreamQuotes(ctx context.Context, symbols []string) (<-chan StreamUpdate, error) {
	if len(c.providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	// Use highest priority provider for streaming
	return c.providers[0].StreamQuotes(ctx, symbols)
}

// HealthCheck checks the health of all configured providers
func (c *Client) HealthCheck(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for _, provider := range c.providers {
		err := provider.HealthCheck(ctx)
		results[provider.Name()] = err
	}
	return results
}

// Close closes the client and cleanup resources
func (c *Client) Close() error {
	if c.cache != nil {
		return c.cache.Close()
	}
	return nil
}

func getPriority(name string, config *Config) int {
	for _, pc := range config.Providers {
		if string(pc.Name) == name {
			return pc.Priority
		}
	}
	return 999
}
