package marketdata

import (
	"errors"
	"time"
)

// ProviderType represents supported market data providers
type ProviderType string

const (
	ProviderPolygon ProviderType = "polygon"
	ProviderAlpaca  ProviderType = "alpaca"
	ProviderIB      ProviderType = "ib" // Interactive Brokers Gateway
)

// Config holds market data client configuration
type Config struct {
	Providers []ProviderConfig
	Cache     CacheConfig
	Symbols   []string
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	Name      ProviderType
	APIKey    string
	APISecret string // Only used for Alpaca
	Tier      string // "free", "starter", "developer", "unlimited"
	Priority  int    // Lower number = higher priority (1 is highest)
	Enabled   bool

	// IB-specific fields (format: "host:port:clientID" in APIKey, e.g. "127.0.0.1:7497:1")
	IBHost     string // IB Gateway host (default: 127.0.0.1)
	IBPort     int    // IB Gateway port (7497 for paper, 7496 for live)
	IBClientID int    // IB client ID (any integer)
}

// CacheConfig holds caching configuration
type CacheConfig struct {
	Enabled  bool
	RedisURL string
	TTL      time.Duration
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Providers: []ProviderConfig{},
		Cache: CacheConfig{
			Enabled:  true,
			RedisURL: "localhost:6379",
			TTL:      5 * time.Minute,
		},
		Symbols: []string{},
	}
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return errors.New("at least one provider must be configured")
	}

	for i, p := range c.Providers {
		if p.Name == "" {
			return errors.New("provider name cannot be empty")
		}
		// IB doesn't need API key (uses socket connection with Gateway login)
		if p.Name != ProviderIB && p.APIKey == "" {
			return errors.New("provider API key cannot be empty")
		}
		if p.Name == ProviderAlpaca && p.APISecret == "" {
			return errors.New("alpaca provider requires API secret")
		}
		if p.Priority == 0 {
			c.Providers[i].Priority = i + 1
		}
	}

	if c.Cache.TTL == 0 {
		c.Cache.TTL = 5 * time.Minute
	}

	return nil
}
