package config

import (
	"encoding/json"
	"os"
)

// Config holds signal generator configuration
type Config struct {
	DatabaseDSN      string   `json:"database_dsn"`
	GenerateInterval int      `json:"generate_interval"` // seconds
	Symbols          []string `json:"symbols"`
	MinConfidence    float64  `json:"min_confidence"`
	SignalExpiry     int      `json:"signal_expiry"` // hours
}

// Load reads configuration from a JSON file
func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.GenerateInterval == 0 {
		cfg.GenerateInterval = 300 // 5 minutes
	}
	if cfg.MinConfidence == 0 {
		cfg.MinConfidence = 0.6
	}
	if cfg.SignalExpiry == 0 {
		cfg.SignalExpiry = 24 // 24 hours
	}

	return &cfg, nil
}
