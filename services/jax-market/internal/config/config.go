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
	IB             IBConfig      `json:"ib"`
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

// IBConfig holds Interactive Brokers Gateway configuration
type IBConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`      // Default: "127.0.0.1"
	Port     int    `json:"port"`      // 7497 for paper, 7496 for live
	ClientID int    `json:"client_id"` // Any integer to identify connection
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
