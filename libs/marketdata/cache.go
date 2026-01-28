package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache provides Redis-backed caching for market data
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewCache creates a new cache instance
func NewCache(config CacheConfig) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
		DB:   0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Cache{
		client: client,
		ttl:    config.TTL,
	}, nil
}

// GetQuote retrieves a cached quote
func (c *Cache) GetQuote(ctx context.Context, symbol string) (*Quote, error) {
	key := fmt.Sprintf("quote:%s", symbol)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNoData
		}
		return nil, fmt.Errorf("%w: %v", ErrCacheError, err)
	}

	var quote Quote
	if err := json.Unmarshal(data, &quote); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal quote: %v", ErrCacheError, err)
	}

	return &quote, nil
}

// SetQuote caches a quote
func (c *Cache) SetQuote(ctx context.Context, quote *Quote) error {
	key := fmt.Sprintf("quote:%s", quote.Symbol)
	data, err := json.Marshal(quote)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal quote: %v", ErrCacheError, err)
	}

	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrCacheError, err)
	}

	return nil
}

// GetCandles retrieves cached candles
func (c *Cache) GetCandles(ctx context.Context, symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	key := fmt.Sprintf("candles:%s:%s:%d", symbol, timeframe, limit)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrNoData
		}
		return nil, fmt.Errorf("%w: %v", ErrCacheError, err)
	}

	var candles []Candle
	if err := json.Unmarshal(data, &candles); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal candles: %v", ErrCacheError, err)
	}

	return candles, nil
}

// SetCandles caches candles
func (c *Cache) SetCandles(ctx context.Context, symbol string, timeframe Timeframe, candles []Candle) error {
	key := fmt.Sprintf("candles:%s:%s:%d", symbol, timeframe, len(candles))
	data, err := json.Marshal(candles)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal candles: %v", ErrCacheError, err)
	}

	// Candles can be cached longer than quotes
	ttl := c.ttl * 2
	if timeframe == Timeframe1Day {
		ttl = 24 * time.Hour
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrCacheError, err)
	}

	return nil
}

// Close closes the cache connection
func (c *Cache) Close() error {
	return c.client.Close()
}
