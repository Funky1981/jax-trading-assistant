package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type JaxCoreConfig struct {
	HTTPPort       int      `json:"httpPort"`
	DefaultSymbols []string `json:"defaultSymbols"`
	AccountSize    float64  `json:"accountSize"`
	RiskPercent    float64  `json:"riskPercent"`
	UseDexter      bool     `json:"useDexter"`
	PostgresDSN    string   `json:"postgresDsn"`
}

func LoadJaxCoreConfig(path string) (JaxCoreConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return JaxCoreConfig{}, fmt.Errorf("read jax-core config: %w", err)
	}

	var cfg JaxCoreConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return JaxCoreConfig{}, fmt.Errorf("parse jax-core config: %w", err)
	}

	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = 8081
	}
	if cfg.RiskPercent == 0 {
		cfg.RiskPercent = 3
	}

	return cfg, nil
}
