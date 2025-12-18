package utcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func LoadProvidersConfig(path string) (ProvidersConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ProvidersConfig{}, fmt.Errorf("read providers config: %w", err)
	}

	var cfg ProvidersConfig
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return ProvidersConfig{}, fmt.Errorf("parse providers config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return ProvidersConfig{}, err
	}

	return cfg, nil
}

func (c ProvidersConfig) Validate() error {
	if len(c.Providers) == 0 {
		return errors.New("providers config: no providers defined")
	}

	seen := make(map[string]struct{}, len(c.Providers))
	for i, p := range c.Providers {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			return fmt.Errorf("providers config: providers[%d].id is required", i)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("providers config: duplicate provider id %q", id)
		}
		seen[id] = struct{}{}

		transport := strings.ToLower(strings.TrimSpace(p.Transport))
		if transport == "" {
			return fmt.Errorf("providers config: providers[%d].transport is required", i)
		}

		switch transport {
		case "local":
			if strings.TrimSpace(p.Endpoint) != "" {
				return fmt.Errorf("providers config: providers[%d] transport local must not set endpoint", i)
			}
		case "http":
			if strings.TrimSpace(p.Endpoint) == "" {
				return fmt.Errorf("providers config: providers[%d] transport http requires endpoint", i)
			}
		default:
			return fmt.Errorf("providers config: providers[%d] has unsupported transport %q", i, p.Transport)
		}
	}

	return nil
}
