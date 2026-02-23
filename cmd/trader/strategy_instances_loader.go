package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

type strategyInstanceFile struct {
	ID                 string          `json:"id,omitempty"`
	Name               string          `json:"name"`
	StrategyTypeID     string          `json:"strategyTypeId"`
	StrategyID         string          `json:"strategyId,omitempty"`
	Enabled            bool            `json:"enabled"`
	SessionTimezone    string          `json:"sessionTimezone,omitempty"`
	FlattenByCloseTime string          `json:"flattenByCloseTime,omitempty"`
	ConfigJSON         json.RawMessage `json:"config"`
	ArtifactID         string          `json:"artifactId,omitempty"`
}

func loadStrategyInstancesFromDir(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read strategy instance dir %q: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var cfg strategyInstanceFile
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return fmt.Errorf("decode %s: %w", path, err)
		}
		if cfg.Name == "" || cfg.StrategyTypeID == "" {
			return fmt.Errorf("%s missing required fields name/strategyTypeId", path)
		}
		if len(cfg.ConfigJSON) == 0 {
			cfg.ConfigJSON = json.RawMessage(`{}`)
		}
		sum := sha256.Sum256(cfg.ConfigJSON)
		hash := hex.EncodeToString(sum[:])
		if cfg.SessionTimezone == "" {
			cfg.SessionTimezone = "America/New_York"
		}
		if cfg.FlattenByCloseTime == "" {
			cfg.FlattenByCloseTime = "15:55"
		}
		_, err = pool.Exec(ctx, `
			INSERT INTO strategy_instances (
				name, strategy_type_id, strategy_id, enabled, session_timezone, flatten_by_close_time, config, config_hash, artifact_id
			) VALUES (
				$1, $2, NULLIF($3,''), $4, $5, $6, $7::jsonb, $8, NULLIF($9,'')::uuid
			)
			ON CONFLICT (name)
			DO UPDATE SET
				strategy_type_id = EXCLUDED.strategy_type_id,
				strategy_id = EXCLUDED.strategy_id,
				enabled = EXCLUDED.enabled,
				session_timezone = EXCLUDED.session_timezone,
				flatten_by_close_time = EXCLUDED.flatten_by_close_time,
				config = EXCLUDED.config,
				config_hash = EXCLUDED.config_hash,
				artifact_id = EXCLUDED.artifact_id
		`, cfg.Name, cfg.StrategyTypeID, cfg.StrategyID, cfg.Enabled, cfg.SessionTimezone, cfg.FlattenByCloseTime, string(cfg.ConfigJSON), hash, cfg.ArtifactID)
		if err != nil {
			return fmt.Errorf("upsert %s: %w", cfg.Name, err)
		}
	}
	return nil
}
