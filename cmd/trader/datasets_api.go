package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func datasetsListHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		if limit > 500 {
			limit = 500
		}
		offset := parseIntParam(r.URL.Query().Get("offset"), 0)
		symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))

		rows, err := pool.Query(r.Context(), `
			SELECT
				s.dataset_id, s.dataset_hash, COALESCE(s.dataset_name,''), COALESCE(s.symbol,''), COALESCE(s.source,''),
				COALESCE(s.schema_ver,''), COALESCE(s.record_count,0), s.start_date, s.end_date, COALESCE(s.file_path,''),
				s.metadata::text, s.created_at, s.updated_at, s.last_seen_at,
				COUNT(l.id) AS link_count
			FROM dataset_snapshots s
			LEFT JOIN dataset_snapshot_links l ON l.dataset_id = s.dataset_id
			WHERE ($1 = '' OR s.symbol = $1)
			GROUP BY
				s.dataset_id, s.dataset_hash, s.dataset_name, s.symbol, s.source, s.schema_ver, s.record_count,
				s.start_date, s.end_date, s.file_path, s.metadata, s.created_at, s.updated_at, s.last_seen_at
			ORDER BY s.last_seen_at DESC
			LIMIT $2 OFFSET $3
		`, symbol, limit, offset)
		if err != nil {
			http.Error(w, datasetSchemaAwareError(err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		out := make([]map[string]any, 0, limit)
		for rows.Next() {
			var (
				datasetID, hash, name, dsSymbol, source, schemaVer, filePath, metadata string
				recordCount                                                            int
				startDate, endDate                                                     *time.Time
				createdAt, updatedAt, lastSeenAt                                       time.Time
				linkCount                                                              int
			)
			if err := rows.Scan(&datasetID, &hash, &name, &dsSymbol, &source, &schemaVer, &recordCount, &startDate, &endDate, &filePath, &metadata, &createdAt, &updatedAt, &lastSeenAt, &linkCount); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if strings.TrimSpace(metadata) == "" {
				metadata = "{}"
			}
			out = append(out, map[string]any{
				"datasetId":   datasetID,
				"datasetHash": hash,
				"name":        name,
				"symbol":      dsSymbol,
				"source":      source,
				"schemaVer":   schemaVer,
				"recordCount": recordCount,
				"startDate":   startDate,
				"endDate":     endDate,
				"filePath":    filePath,
				"metadata":    json.RawMessage(metadata),
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
				"lastSeenAt":  lastSeenAt,
				"linkCount":   linkCount,
			})
		}
		jsonOK(w, map[string]any{
			"datasets": out,
			"limit":    limit,
			"offset":   offset,
		})
	}
}

func datasetsDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		datasetID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/datasets/"), "/")
		if datasetID == "" {
			http.NotFound(w, r)
			return
		}

		var (
			id, hash, name, dsSymbol, source, schemaVer, filePath, metadata string
			recordCount                                                     int
			startDate, endDate                                              *time.Time
			createdAt, updatedAt, lastSeenAt                                time.Time
		)
		err := pool.QueryRow(r.Context(), `
			SELECT
				dataset_id, dataset_hash, COALESCE(dataset_name,''), COALESCE(symbol,''), COALESCE(source,''),
				COALESCE(schema_ver,''), COALESCE(record_count,0), start_date, end_date, COALESCE(file_path,''),
				metadata::text, created_at, updated_at, last_seen_at
			FROM dataset_snapshots
			WHERE dataset_id = $1
		`, datasetID).Scan(&id, &hash, &name, &dsSymbol, &source, &schemaVer, &recordCount, &startDate, &endDate, &filePath, &metadata, &createdAt, &updatedAt, &lastSeenAt)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			http.Error(w, datasetSchemaAwareError(err), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(metadata) == "" {
			metadata = "{}"
		}

		rows, err := pool.Query(r.Context(), `
			SELECT run_type, run_ref_id, observed_hash, linked_at, metadata::text
			FROM dataset_snapshot_links
			WHERE dataset_id = $1
			ORDER BY linked_at DESC
			LIMIT 200
		`, datasetID)
		if err != nil {
			http.Error(w, datasetSchemaAwareError(err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		links := make([]map[string]any, 0, 32)
		for rows.Next() {
			var runType, runRefID, observedHash, linkMetadata string
			var linkedAt time.Time
			if err := rows.Scan(&runType, &runRefID, &observedHash, &linkedAt, &linkMetadata); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if strings.TrimSpace(linkMetadata) == "" {
				linkMetadata = "{}"
			}
			links = append(links, map[string]any{
				"runType":      runType,
				"runRefId":     runRefID,
				"observedHash": observedHash,
				"linkedAt":     linkedAt,
				"metadata":     json.RawMessage(linkMetadata),
			})
		}

		jsonOK(w, map[string]any{
			"datasetId":   id,
			"datasetHash": hash,
			"name":        name,
			"symbol":      dsSymbol,
			"source":      source,
			"schemaVer":   schemaVer,
			"recordCount": recordCount,
			"startDate":   startDate,
			"endDate":     endDate,
			"filePath":    filePath,
			"metadata":    json.RawMessage(metadata),
			"createdAt":   createdAt,
			"updatedAt":   updatedAt,
			"lastSeenAt":  lastSeenAt,
			"links":       links,
		})
	}
}

func datasetSchemaAwareError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "dataset_snapshots") || strings.Contains(msg, "dataset_snapshot_links") {
		return "dataset snapshot schema not available yet; apply migrations through 000011"
	}
	return msg
}
