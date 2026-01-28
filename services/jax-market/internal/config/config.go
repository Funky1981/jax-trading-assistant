package config

import (
	"encoding/json"
	"os"
)

// Config holds jax-market configuration
type Config struct {
	DatabaseDSN    string        `json:"database_dsn"`
	IngestInterval int           `json:"ingest_interval"` // seconds
	Symbols        []string      `json:"symbols"`
	Polygon        PolygonConfig `json:"polygon"`
	Alpaca         AlpacaConfig  `json:"alpaca"`
	Cache          CacheConfig   `json:"cache"`
}

// PolygonConfig holds Polygon.io configuration
type PolygonConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key"`
	Tier    string `json:"tier"`
}

// AlpacaConfig holds Alpaca configuration
type AlpacaConfig struct {
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Tier      string `json:"tier"`
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled  bool   `json:"enabled"`
	RedisURL string `json:"redis_url"`
	TTL      int    `json:"ttl"` // seconds
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply environment variable overrides
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		config.DatabaseDSN = dsn
	}
	if key := os.Getenv("POLYGON_API_KEY"); key != "" {
		config.Polygon.APIKey = key
	}
	if key := os.Getenv("ALPACA_API_KEY"); key != "" {
		config.Alpaca.APIKey = key
	}
	if secret := os.Getenv("ALPACA_API_SECRET"); secret != "" {
		config.Alpaca.APISecret = secret
	}
	if redis := os.Getenv("REDIS_URL"); redis != "" {
		config.Cache.RedisURL = redis
	}

	return &config, nil
}
