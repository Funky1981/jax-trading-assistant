package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func eventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		query, args := buildEventsFilter(r)
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		if limit > 500 {
			limit = 500
		}
		offset := parseIntParam(r.URL.Query().Get("offset"), 0)

		var total int
		if err := pool.QueryRow(r.Context(), `
			SELECT COUNT(*)
			FROM event_normalized n
			WHERE `+query, args...).Scan(&total); err != nil {
			http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
			return
		}

		args = append(args, limit, offset)
		rows, err := pool.Query(r.Context(), `
			SELECT
				n.id::text, n.event_kind, n.title, COALESCE(n.summary,''), COALESCE(n.severity,''), n.event_time,
				n.source_id, COALESCE(n.primary_symbol,''), n.confidence, n.attributes::text, n.created_at,
				COALESCE(array_agg(DISTINCT sm.symbol) FILTER (WHERE sm.symbol IS NOT NULL), '{}')
			FROM event_normalized n
			LEFT JOIN event_symbol_map sm ON sm.normalized_event_id = n.id
			WHERE `+query+`
			GROUP BY
				n.id, n.event_kind, n.title, n.summary, n.severity, n.event_time,
				n.source_id, n.primary_symbol, n.confidence, n.attributes, n.created_at
			ORDER BY n.event_time DESC, n.created_at DESC
			LIMIT $`+fmt.Sprintf("%d", len(args)-1)+` OFFSET $`+fmt.Sprintf("%d", len(args)), args...)
		if err != nil {
			http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		events := make([]map[string]any, 0, limit)
		for rows.Next() {
			var (
				id, kind, title, summary, severity, sourceID, primarySymbol, attrs string
				eventTime, createdAt                                               time.Time
				confidence                                                         float64
				symbols                                                            []string
			)
			if err := rows.Scan(&id, &kind, &title, &summary, &severity, &eventTime, &sourceID, &primarySymbol, &confidence, &attrs, &createdAt, &symbols); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if strings.TrimSpace(attrs) == "" {
				attrs = "{}"
			}
			events = append(events, map[string]any{
				"id":            id,
				"kind":          kind,
				"title":         title,
				"summary":       summary,
				"severity":      severity,
				"eventTime":     eventTime,
				"sourceId":      sourceID,
				"primarySymbol": primarySymbol,
				"symbols":       symbols,
				"confidence":    confidence,
				"attributes":    json.RawMessage(attrs),
				"createdAt":     createdAt,
			})
		}
		jsonOK(w, map[string]any{
			"events": events,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

func eventsDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/events/"), "/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(path, "/")
		eventID := strings.TrimSpace(parts[0])
		if _, err := uuid.Parse(eventID); err != nil {
			http.Error(w, "invalid event ID", http.StatusBadRequest)
			return
		}
		if len(parts) > 1 && parts[1] == "timeline" {
			eventTimelineHandler(w, r, pool, eventID)
			return
		}
		eventDetailGet(w, r, pool, eventID)
	}
}

func eventDetailGet(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, eventID string) {
	var (
		id, kind, title, summary, severity, sourceID, primarySymbol, attrs string
		eventTime, createdAt                                               time.Time
		confidence                                                         float64
		symbols                                                            []string
	)
	err := pool.QueryRow(r.Context(), `
		SELECT
			n.id::text, n.event_kind, n.title, COALESCE(n.summary,''), COALESCE(n.severity,''), n.event_time,
			n.source_id, COALESCE(n.primary_symbol,''), n.confidence, n.attributes::text, n.created_at,
			COALESCE(array_agg(DISTINCT sm.symbol) FILTER (WHERE sm.symbol IS NOT NULL), '{}')
		FROM event_normalized n
		LEFT JOIN event_symbol_map sm ON sm.normalized_event_id = n.id
		WHERE n.id = $1::uuid
		GROUP BY
			n.id, n.event_kind, n.title, n.summary, n.severity, n.event_time,
			n.source_id, n.primary_symbol, n.confidence, n.attributes, n.created_at
	`, eventID).Scan(&id, &kind, &title, &summary, &severity, &eventTime, &sourceID, &primarySymbol, &confidence, &attrs, &createdAt, &symbols)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(attrs) == "" {
		attrs = "{}"
	}

	rawRows, err := pool.Query(r.Context(), `
		SELECT
			r.id::text, r.source_id, r.source_event_id, r.event_kind, r.event_time, r.received_at,
			COALESCE(r.symbol,''), r.payload::text, r.content_hash, COALESCE(r.flow_id,''),
			r.data_source_type, COALESCE(r.source_provider,''), r.is_synthetic, COALESCE(r.synthetic_reason,''),
			r.provenance_verified_at, r.created_at
		FROM event_raw r
		JOIN event_normalized n ON n.raw_event_id = r.id
		WHERE n.id = $1::uuid
		ORDER BY r.received_at DESC
	`, eventID)
	if err != nil {
		http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
		return
	}
	defer rawRows.Close()

	raw := make([]map[string]any, 0, 4)
	for rawRows.Next() {
		var (
			rawID, rawSourceID, rawSourceEventID, rawKind, rawSymbol, payload, contentHash, flowID, dataSourceType, sourceProvider, syntheticReason string
			eventTS, receivedAt, rawCreatedAt                                                                                                       time.Time
			provenanceVerifiedAt                                                                                                                    *time.Time
			isSynthetic                                                                                                                             bool
		)
		if err := rawRows.Scan(&rawID, &rawSourceID, &rawSourceEventID, &rawKind, &eventTS, &receivedAt,
			&rawSymbol, &payload, &contentHash, &flowID, &dataSourceType, &sourceProvider, &isSynthetic, &syntheticReason, &provenanceVerifiedAt, &rawCreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		raw = append(raw, map[string]any{
			"id":                   rawID,
			"sourceId":             rawSourceID,
			"sourceEventId":        rawSourceEventID,
			"kind":                 rawKind,
			"eventTime":            eventTS,
			"receivedAt":           receivedAt,
			"symbol":               rawSymbol,
			"payload":              json.RawMessage(payload),
			"contentHash":          contentHash,
			"flowId":               flowID,
			"dataSourceType":       dataSourceType,
			"sourceProvider":       sourceProvider,
			"isSynthetic":          isSynthetic,
			"syntheticReason":      syntheticReason,
			"provenanceVerifiedAt": provenanceVerifiedAt,
			"createdAt":            rawCreatedAt,
		})
	}

	jsonOK(w, map[string]any{
		"id":            id,
		"kind":          kind,
		"title":         title,
		"summary":       summary,
		"severity":      severity,
		"eventTime":     eventTime,
		"sourceId":      sourceID,
		"primarySymbol": primarySymbol,
		"symbols":       symbols,
		"confidence":    confidence,
		"attributes":    json.RawMessage(attrs),
		"createdAt":     createdAt,
		"raw":           raw,
	})
}

func eventTimelineHandler(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool, eventID string) {
	var normCreatedAt time.Time
	if err := pool.QueryRow(r.Context(), `SELECT created_at FROM event_normalized WHERE id = $1::uuid`, eventID).Scan(&normCreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
		return
	}

	timeline := make([]map[string]any, 0, 8)

	rawRows, err := pool.Query(r.Context(), `
		SELECT r.id::text, r.event_time, r.received_at, COALESCE(r.flow_id,''), r.payload::text
		FROM event_raw r
		JOIN event_normalized n ON n.raw_event_id = r.id
		WHERE n.id = $1::uuid
		ORDER BY r.received_at ASC
	`, eventID)
	if err != nil {
		http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
		return
	}
	defer rawRows.Close()
	for rawRows.Next() {
		var rawID, flowID, payload string
		var eventAt, receivedAt time.Time
		if err := rawRows.Scan(&rawID, &eventAt, &receivedAt, &flowID, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.TrimSpace(payload) == "" {
			payload = "{}"
		}
		timeline = append(timeline, map[string]any{
			"type":    "raw.event_time",
			"ts":      eventAt,
			"message": "source event timestamp",
			"rawId":   rawID,
			"flowId":  flowID,
			"payload": json.RawMessage(payload),
		})
		timeline = append(timeline, map[string]any{
			"type":    "raw.received",
			"ts":      receivedAt,
			"message": "raw event persisted",
			"rawId":   rawID,
		})
	}

	timeline = append(timeline, map[string]any{
		"type":    "normalized.created",
		"ts":      normCreatedAt,
		"message": "normalized event upserted",
		"eventId": eventID,
	})

	mapRows, err := pool.Query(r.Context(), `
		SELECT symbol, relevance, mapping_method, is_primary, created_at
		FROM event_symbol_map
		WHERE normalized_event_id = $1::uuid
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		http.Error(w, schemaAwareError(err), http.StatusInternalServerError)
		return
	}
	defer mapRows.Close()
	for mapRows.Next() {
		var symbol, method string
		var relevance float64
		var isPrimary bool
		var createdAt time.Time
		if err := mapRows.Scan(&symbol, &relevance, &method, &isPrimary, &createdAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		timeline = append(timeline, map[string]any{
			"type":          "symbol.mapped",
			"ts":            createdAt,
			"symbol":        symbol,
			"relevance":     relevance,
			"mappingMethod": method,
			"isPrimary":     isPrimary,
		})
	}

	slices.SortFunc(timeline, func(a, b map[string]any) int {
		aTS, _ := a["ts"].(time.Time)
		bTS, _ := b["ts"].(time.Time)
		switch {
		case aTS.Before(bTS):
			return -1
		case aTS.After(bTS):
			return 1
		default:
			return 0
		}
	})

	jsonOK(w, map[string]any{
		"eventId":   eventID,
		"timeline":  timeline,
		"totalRows": len(timeline),
	})
}

func buildEventsFilter(r *http.Request) (string, []any) {
	q := r.URL.Query()
	var (
		filters = []string{"1=1"}
		args    []any
	)
	addArg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if kind := strings.TrimSpace(q.Get("kind")); kind != "" {
		filters = append(filters, fmt.Sprintf("n.event_kind = %s", addArg(kind)))
	}
	if sourceID := strings.TrimSpace(q.Get("sourceId")); sourceID != "" {
		filters = append(filters, fmt.Sprintf("n.source_id = %s", addArg(sourceID)))
	}
	if symbol := strings.ToUpper(strings.TrimSpace(q.Get("symbol"))); symbol != "" {
		arg := addArg(symbol)
		filters = append(filters, fmt.Sprintf("EXISTS (SELECT 1 FROM event_symbol_map smf WHERE smf.normalized_event_id = n.id AND smf.symbol = %s)", arg))
	}
	if search := strings.TrimSpace(q.Get("search")); search != "" {
		arg := addArg("%" + search + "%")
		filters = append(filters, fmt.Sprintf("(n.title ILIKE %s OR COALESCE(n.summary,'') ILIKE %s)", arg, arg))
	}

	from, to := parseOptionalRange(q.Get("from"), q.Get("to"))
	if !from.IsZero() {
		filters = append(filters, fmt.Sprintf("n.event_time >= %s", addArg(from.UTC())))
	}
	if !to.IsZero() {
		filters = append(filters, fmt.Sprintf("n.event_time <= %s", addArg(to.UTC())))
	}

	return strings.Join(filters, " AND "), args
}

func schemaAwareError(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "event_normalized") ||
		strings.Contains(msg, "event_raw") ||
		strings.Contains(msg, "event_symbol_map") {
		return "event data schema not available yet; apply migrations through 000010"
	}
	return msg
}
