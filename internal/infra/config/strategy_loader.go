package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"jax-trading-assistant/internal/domain"
)

func LoadStrategyConfigs(dir string) (map[string]domain.StrategyConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read strategies dir: %w", err)
	}

	out := make(map[string]domain.StrategyConfig)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read strategy file %s: %w", path, err)
		}

		var cfg domain.StrategyConfig
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("parse strategy file %s: %w", path, err)
		}
		if cfg.ID == "" {
			return nil, fmt.Errorf("strategy file %s: id is required", path)
		}
		if _, exists := out[cfg.ID]; exists {
			return nil, fmt.Errorf("duplicate strategy id %q", cfg.ID)
		}

		out[cfg.ID] = cfg
	}

	return out, nil
}
