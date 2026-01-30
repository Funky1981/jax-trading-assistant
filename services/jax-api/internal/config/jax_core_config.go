package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

type JaxCoreConfig struct {
	HTTPPort             int      `json:"httpPort"`
	DefaultSymbols       []string `json:"defaultSymbols"`
	AccountSize          float64  `json:"accountSize"`
	RiskPercent          float64  `json:"riskPercent"`
	MaxConsecutiveLosses int      `json:"maxConsecutiveLosses"`
	UseDexter            bool     `json:"useDexter"`
	PostgresDSN          string   `json:"postgresDsn"`
	KnowledgeDSN         string   `json:"knowledgeDsn"` // Knowledge base DSN
}

// DefaultKnowledgeDSN is the default connection string for the knowledge database.
const DefaultKnowledgeDSN = "postgres://postgres:postgres@localhost:5432/jax_knowledge?sslmode=disable"

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
	if cfg.MaxConsecutiveLosses == 0 {
		cfg.MaxConsecutiveLosses = 3
	}

	// Allow overriding Postgres DSN with environment variable DATABASE_URL.
	// This is useful when running in Docker Compose or locally with .env files.
	if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		cfg.PostgresDSN = envDSN
	}

	// Allow overriding Knowledge DSN with environment variable JAX_KNOWLEDGE_DSN.
	// Falls back to default if not set in config or env.
	if envKnowledgeDSN := os.Getenv("JAX_KNOWLEDGE_DSN"); envKnowledgeDSN != "" {
		cfg.KnowledgeDSN = envKnowledgeDSN
	} else if cfg.KnowledgeDSN == "" {
		cfg.KnowledgeDSN = DefaultKnowledgeDSN
	}

	return cfg, nil
}
