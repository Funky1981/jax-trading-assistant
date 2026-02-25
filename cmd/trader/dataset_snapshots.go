package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func persistDatasetSnapshotAndLinks(ctx context.Context, pool *pgxpool.Pool, datasetID, datasetHash string, upstream map[string]any, runRowID, externalRunID string) error {
	if pool == nil {
		return nil
	}
	datasetID = strings.TrimSpace(datasetID)
	if datasetID == "" {
		return nil
	}
	datasetHash = strings.TrimSpace(datasetHash)
	if datasetHash == "" {
		datasetHash = strings.TrimSpace(toString(upstream["dataset_hash"]))
	}
	if datasetHash == "" {
		datasetHash = "unknown"
	}

	meta := map[string]any{
		"upstreamRunId": externalRunID,
		"capturedAt":    time.Now().UTC().Format(time.RFC3339),
	}
	metaJSON, _ := json.Marshal(meta)

	start := parseOptionalRFC3339(toString(upstream["dataset_start_date"]))
	end := parseOptionalRFC3339(toString(upstream["dataset_end_date"]))
	_, err := pool.Exec(ctx, `
		INSERT INTO dataset_snapshots (
			dataset_id, dataset_hash, dataset_name, symbol, source, schema_ver, record_count,
			start_date, end_date, file_path, metadata, created_at, updated_at, last_seen_at
		)
		VALUES (
			$1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), $7,
			$8, $9, NULLIF($10,''), $11::jsonb, NOW(), NOW(), NOW()
		)
		ON CONFLICT (dataset_id)
		DO UPDATE SET
			dataset_hash = EXCLUDED.dataset_hash,
			dataset_name = EXCLUDED.dataset_name,
			symbol = EXCLUDED.symbol,
			source = EXCLUDED.source,
			schema_ver = EXCLUDED.schema_ver,
			record_count = EXCLUDED.record_count,
			start_date = EXCLUDED.start_date,
			end_date = EXCLUDED.end_date,
			file_path = EXCLUDED.file_path,
			metadata = EXCLUDED.metadata,
			last_seen_at = NOW(),
			updated_at = NOW()
	`, datasetID, datasetHash, toString(upstream["dataset_name"]), toString(upstream["dataset_symbol"]), toString(upstream["dataset_source"]),
		toString(upstream["dataset_schema_ver"]), intFromAny(upstream["dataset_record_count"]), nullableTime(start), nullableTime(end),
		toString(upstream["dataset_file_path"]), string(metaJSON))
	if err != nil {
		return fmt.Errorf("upsert dataset snapshot: %w", err)
	}

	if strings.TrimSpace(runRowID) != "" {
		if err := upsertDatasetSnapshotLink(ctx, pool, datasetID, datasetHash, "run", runRowID, map[string]any{
			"source": "codex_api",
		}); err != nil {
			return err
		}
	}

	if strings.TrimSpace(externalRunID) != "" {
		var backtestRowID string
		err := pool.QueryRow(ctx, `
			SELECT id::text
			FROM backtest_runs
			WHERE external_run_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		`, externalRunID).Scan(&backtestRowID)
		if err == nil && strings.TrimSpace(backtestRowID) != "" {
			if err := upsertDatasetSnapshotLink(ctx, pool, datasetID, datasetHash, "backtest_run", backtestRowID, map[string]any{
				"externalRunId": externalRunID,
			}); err != nil {
				return err
			}
		} else if err != nil && err != pgx.ErrNoRows {
			return fmt.Errorf("lookup backtest run for dataset link: %w", err)
		}
	}

	return nil
}

func upsertDatasetSnapshotLink(ctx context.Context, pool *pgxpool.Pool, datasetID, observedHash, runType, runRefID string, metadata map[string]any) error {
	metaJSON, _ := json.Marshal(metadata)
	_, err := pool.Exec(ctx, `
		INSERT INTO dataset_snapshot_links (
			dataset_id, run_type, run_ref_id, observed_hash, linked_at, metadata
		) VALUES (
			$1, $2, $3, $4, NOW(), $5::jsonb
		)
		ON CONFLICT (dataset_id, run_type, run_ref_id)
		DO UPDATE SET
			observed_hash = EXCLUDED.observed_hash,
			linked_at = NOW(),
			metadata = EXCLUDED.metadata
	`, datasetID, runType, runRefID, observedHash, string(metaJSON))
	if err != nil {
		return fmt.Errorf("upsert dataset snapshot link (%s %s): %w", runType, runRefID, err)
	}
	return nil
}

func parseOptionalRFC3339(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	utc := ts.UTC()
	return &utc
}

func nullableTime(ts *time.Time) any {
	if ts == nil {
		return nil
	}
	return *ts
}
