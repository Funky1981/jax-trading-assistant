package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventStore struct {
	pool *pgxpool.Pool
}

type persistEventInput struct {
	SourceID      string
	SourceName    string
	ProviderType  string
	SourceEventID string
	EventKind     string
	EventTime     time.Time
	PrimarySymbol string
	Title         string
	Summary       string
	Severity      string
	Confidence    float64
	Payload       map[string]any
	Attributes    map[string]any
	Symbols       []string
}

func newEventStore(pool *pgxpool.Pool) *eventStore {
	return &eventStore{pool: pool}
}

func (s *eventStore) SaveEarnings(ctx context.Context, symbol, sourceID string, events []utcp.EarningsEntry) error {
	if s == nil || s.pool == nil || len(events) == 0 {
		return nil
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	var errs []error
	for _, item := range events {
		eventTime := parseEventDate(item.Date)
		payload := map[string]any{
			"symbol":       symbol,
			"date":         item.Date,
			"eps_actual":   item.EPSActual,
			"eps_estimate": item.EPSEstimate,
			"surprise_pct": item.SurprisePct,
		}
		sourceEventID := deterministicEventID(sourceID, "earnings", symbol, eventTime.Format(time.RFC3339), item.Date)
		attrs := map[string]any{
			"epsActual":   item.EPSActual,
			"epsEstimate": item.EPSEstimate,
			"surprisePct": item.SurprisePct,
		}
		input := persistEventInput{
			SourceID:      sourceID,
			SourceName:    strings.ToUpper(sourceID),
			ProviderType:  "external",
			SourceEventID: sourceEventID,
			EventKind:     "earnings",
			EventTime:     eventTime,
			PrimarySymbol: symbol,
			Title:         fmt.Sprintf("%s earnings", symbol),
			Summary:       earningsSummary(item),
			Severity:      earningsSeverity(item.SurprisePct),
			Confidence:    1.0,
			Payload:       payload,
			Attributes:    attrs,
			Symbols:       []string{symbol},
		}
		if err := s.persistEvent(ctx, input); err != nil {
			errs = append(errs, err)
		}
	}
	return errorsJoin(errs...)
}

func (s *eventStore) SaveNews(ctx context.Context, symbol, sourceID string, events []utcp.NewsEntry) error {
	if s == nil || s.pool == nil || len(events) == 0 {
		return nil
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	var errs []error
	for _, item := range events {
		ts := item.Timestamp.UTC()
		if ts.IsZero() {
			ts = time.Now().UTC()
		}
		payload := map[string]any{
			"symbol":    symbol,
			"headline":  item.Headline,
			"summary":   item.Summary,
			"url":       item.URL,
			"source":    item.Source,
			"category":  item.Category,
			"timestamp": ts.Format(time.RFC3339),
		}
		sourceEventID := deterministicEventID(sourceID, "news", symbol, item.URL, item.Headline, ts.Format(time.RFC3339))
		attrs := map[string]any{
			"url":      item.URL,
			"source":   item.Source,
			"category": item.Category,
		}
		input := persistEventInput{
			SourceID:      sourceID,
			SourceName:    strings.ToUpper(sourceID),
			ProviderType:  "external",
			SourceEventID: sourceEventID,
			EventKind:     "news",
			EventTime:     ts,
			PrimarySymbol: symbol,
			Title:         strings.TrimSpace(item.Headline),
			Summary:       strings.TrimSpace(item.Summary),
			Severity:      newsSeverity(item.Category),
			Confidence:    1.0,
			Payload:       payload,
			Attributes:    attrs,
			Symbols:       []string{symbol},
		}
		if err := s.persistEvent(ctx, input); err != nil {
			errs = append(errs, err)
		}
	}
	return errorsJoin(errs...)
}

func (s *eventStore) SaveMacroNews(ctx context.Context, sourceID string, events []utcp.NewsEntry) error {
	if s == nil || s.pool == nil || len(events) == 0 {
		return nil
	}
	var errs []error
	for _, item := range events {
		ts := item.Timestamp.UTC()
		if ts.IsZero() {
			ts = time.Now().UTC()
		}
		payload := map[string]any{
			"headline":  item.Headline,
			"summary":   item.Summary,
			"source":    item.Source,
			"category":  item.Category,
			"timestamp": ts.Format(time.RFC3339),
		}
		sourceEventID := deterministicEventID(sourceID, "macro", item.Headline, ts.Format(time.RFC3339))
		input := persistEventInput{
			SourceID:      sourceID,
			SourceName:    "Economic Calendar",
			ProviderType:  "calendar",
			SourceEventID: sourceEventID,
			EventKind:     "macro",
			EventTime:     ts,
			PrimarySymbol: "",
			Title:         strings.TrimSpace(item.Headline),
			Summary:       strings.TrimSpace(item.Summary),
			Severity:      macroSeverity(item.Summary),
			Confidence:    0.9,
			Payload:       payload,
			Attributes: map[string]any{
				"source":   item.Source,
				"category": item.Category,
			},
		}
		if err := s.persistEvent(ctx, input); err != nil {
			errs = append(errs, err)
		}
	}
	return errorsJoin(errs...)
}

func (s *eventStore) persistEvent(ctx context.Context, in persistEventInput) error {
	if in.EventTime.IsZero() {
		in.EventTime = time.Now().UTC()
	}
	if strings.TrimSpace(in.SourceID) == "" {
		return fmt.Errorf("event source is required")
	}
	if strings.TrimSpace(in.SourceEventID) == "" {
		in.SourceEventID = deterministicEventID(in.SourceID, in.EventKind, in.Title, in.EventTime.Format(time.RFC3339))
	}
	if strings.TrimSpace(in.Title) == "" {
		in.Title = fmt.Sprintf("%s event", strings.TrimSpace(in.EventKind))
	}
	if strings.TrimSpace(in.Severity) == "" {
		in.Severity = "unknown"
	}
	if in.Confidence <= 0 {
		in.Confidence = 1.0
	}
	if in.Payload == nil {
		in.Payload = map[string]any{}
	}
	if in.Attributes == nil {
		in.Attributes = map[string]any{}
	}

	if err := s.ensureSource(ctx, in.SourceID, in.SourceName, in.ProviderType); err != nil {
		return err
	}

	payloadJSON, err := json.Marshal(in.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	attrJSON, err := json.Marshal(in.Attributes)
	if err != nil {
		return fmt.Errorf("marshal event attributes: %w", err)
	}
	contentHash := hashBytes(payloadJSON)
	canonicalKey := deterministicEventID(in.EventKind, strings.ToUpper(strings.TrimSpace(in.PrimarySymbol)), in.Title, in.EventTime.UTC().Format(time.RFC3339))
	flowID := observability.FlowIDFromContext(ctx)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin event tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var rawID string
	err = tx.QueryRow(ctx, `
		INSERT INTO event_raw (
			source_id, source_event_id, event_kind, event_time, symbol, payload, content_hash,
			flow_id, data_source_type, source_provider, is_synthetic, synthetic_reason, provenance_verified_at
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5,''), $6::jsonb, $7,
			NULLIF($8,''), 'real', $9, FALSE, '', NOW()
		)
		ON CONFLICT (source_id, source_event_id)
		DO UPDATE SET
			event_kind = EXCLUDED.event_kind,
			event_time = EXCLUDED.event_time,
			symbol = EXCLUDED.symbol,
			payload = EXCLUDED.payload,
			content_hash = EXCLUDED.content_hash,
			flow_id = EXCLUDED.flow_id,
			data_source_type = EXCLUDED.data_source_type,
			source_provider = EXCLUDED.source_provider,
			is_synthetic = EXCLUDED.is_synthetic,
			synthetic_reason = EXCLUDED.synthetic_reason,
			provenance_verified_at = EXCLUDED.provenance_verified_at,
			received_at = NOW()
		RETURNING id::text
	`, in.SourceID, in.SourceEventID, in.EventKind, in.EventTime.UTC(), strings.TrimSpace(in.PrimarySymbol), string(payloadJSON), contentHash,
		flowID, in.SourceID).Scan(&rawID)
	if err != nil {
		return fmt.Errorf("upsert event_raw: %w", err)
	}

	var normalizedID string
	err = tx.QueryRow(ctx, `
		INSERT INTO event_normalized (
			raw_event_id, canonical_key, event_kind, title, summary, severity, event_time, source_id,
			primary_symbol, confidence, attributes, data_source_type, source_provider,
			is_synthetic, synthetic_reason, provenance_verified_at
		)
		VALUES (
			$1::uuid, $2, $3, $4, NULLIF($5,''), $6, $7, $8,
			NULLIF($9,''), $10, $11::jsonb, 'real', $12,
			FALSE, '', NOW()
		)
		ON CONFLICT (canonical_key)
		DO UPDATE SET
			raw_event_id = EXCLUDED.raw_event_id,
			event_kind = EXCLUDED.event_kind,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			severity = EXCLUDED.severity,
			event_time = EXCLUDED.event_time,
			source_id = EXCLUDED.source_id,
			primary_symbol = EXCLUDED.primary_symbol,
			confidence = EXCLUDED.confidence,
			attributes = EXCLUDED.attributes,
			data_source_type = EXCLUDED.data_source_type,
			source_provider = EXCLUDED.source_provider,
			is_synthetic = EXCLUDED.is_synthetic,
			synthetic_reason = EXCLUDED.synthetic_reason,
			provenance_verified_at = EXCLUDED.provenance_verified_at
		RETURNING id::text
	`, rawID, canonicalKey, in.EventKind, in.Title, strings.TrimSpace(in.Summary), in.Severity, in.EventTime.UTC(), in.SourceID,
		strings.TrimSpace(in.PrimarySymbol), in.Confidence, string(attrJSON), in.SourceID).Scan(&normalizedID)
	if err != nil {
		return fmt.Errorf("upsert event_normalized: %w", err)
	}

	symbols := normalizeSymbols(in.PrimarySymbol, in.Symbols)
	if len(symbols) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM event_symbol_map WHERE normalized_event_id = $1::uuid`, normalizedID); err != nil {
			return fmt.Errorf("clear event symbol map: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			DELETE FROM event_symbol_map
			WHERE normalized_event_id = $1::uuid
			  AND NOT (symbol = ANY($2))
		`, normalizedID, symbols); err != nil {
			return fmt.Errorf("prune event symbol map: %w", err)
		}
		for _, symbol := range symbols {
			isPrimary := strings.EqualFold(symbol, strings.TrimSpace(in.PrimarySymbol))
			if strings.TrimSpace(in.PrimarySymbol) == "" {
				isPrimary = symbol == symbols[0]
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO event_symbol_map (normalized_event_id, symbol, relevance, mapping_method, is_primary)
				VALUES ($1::uuid, $2, 1.0, 'provider_symbol', $3)
				ON CONFLICT (normalized_event_id, symbol)
				DO UPDATE SET
					relevance = EXCLUDED.relevance,
					mapping_method = EXCLUDED.mapping_method,
					is_primary = EXCLUDED.is_primary
			`, normalizedID, symbol, isPrimary); err != nil {
				return fmt.Errorf("upsert event symbol map: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit event tx: %w", err)
	}
	s.logAudit(ctx, flowID, normalizedID, in)
	return nil
}

func (s *eventStore) ensureSource(ctx context.Context, sourceID, sourceName, providerType string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	if strings.TrimSpace(sourceName) == "" {
		sourceName = strings.ToUpper(strings.TrimSpace(sourceID))
	}
	if strings.TrimSpace(providerType) == "" {
		providerType = "external"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO event_sources (id, display_name, provider_type, enabled, priority, metadata)
		VALUES ($1, $2, $3, TRUE, 100, '{}'::jsonb)
		ON CONFLICT (id)
		DO UPDATE SET
			display_name = EXCLUDED.display_name,
			provider_type = EXCLUDED.provider_type,
			updated_at = NOW()
	`, sourceID, sourceName, providerType)
	if err != nil {
		return fmt.Errorf("upsert event source %q: %w", sourceID, err)
	}
	return nil
}

func (s *eventStore) logAudit(ctx context.Context, flowID, eventID string, in persistEventInput) {
	if s == nil || s.pool == nil {
		return
	}
	correlationID := strings.TrimSpace(flowID)
	if correlationID == "" {
		correlationID = eventID
	}
	meta, _ := json.Marshal(map[string]any{
		"eventId":       eventID,
		"eventKind":     in.EventKind,
		"sourceId":      in.SourceID,
		"sourceEventId": in.SourceEventID,
		"primarySymbol": in.PrimarySymbol,
		"severity":      in.Severity,
		"symbols":       normalizeSymbols(in.PrimarySymbol, in.Symbols),
	})
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO audit_events (
			id, correlation_id, category, action, outcome, message, metadata, timestamp
		)
		VALUES ($1, $2, 'trading_data', 'event.persist', 'success', $3, $4::jsonb, NOW())
	`, uuid.NewString(), correlationID, fmt.Sprintf("persisted %s event", in.EventKind), string(meta)); err != nil {
		observability.LogEvent(ctx, "warn", "events.audit_log_failed", map[string]any{
			"event_id": eventID,
			"source":   in.SourceID,
			"error":    err.Error(),
		})
	}
}

func normalizeSymbols(primary string, symbols []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(symbols)+1)
	add := func(raw string) {
		v := strings.ToUpper(strings.TrimSpace(raw))
		if v == "" {
			return
		}
		if _, exists := seen[v]; exists {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	add(primary)
	for _, symbol := range symbols {
		add(symbol)
	}
	sort.Strings(out)
	if p := strings.ToUpper(strings.TrimSpace(primary)); p != "" {
		for i, symbol := range out {
			if symbol == p {
				out[0], out[i] = out[i], out[0]
				break
			}
		}
	}
	return out
}

func deterministicEventID(parts ...string) string {
	joined := strings.ToLower(strings.TrimSpace(strings.Join(parts, "|")))
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:16])
}

func hashBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func parseEventDate(raw string) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Now().UTC()
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, trimmed); err == nil {
			return ts.UTC()
		}
	}
	return time.Now().UTC()
}

func earningsSummary(item utcp.EarningsEntry) string {
	if item.EPSEstimate == 0 {
		return fmt.Sprintf("EPS %.2f", item.EPSActual)
	}
	return fmt.Sprintf("EPS %.2f vs %.2f (%.2f%% surprise)", item.EPSActual, item.EPSEstimate, item.SurprisePct)
}

func earningsSeverity(surprisePct float64) string {
	abs := surprisePct
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 20:
		return "high"
	case abs >= 8:
		return "medium"
	default:
		return "low"
	}
}

func newsSeverity(category string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	switch c {
	case "top news", "breaking", "major", "company_news":
		return "high"
	case "general", "business":
		return "medium"
	default:
		return "low"
	}
}

func macroSeverity(summary string) string {
	s := strings.ToLower(summary)
	if strings.Contains(s, "high") {
		return "high"
	}
	if strings.Contains(s, "medium") {
		return "medium"
	}
	return "low"
}

func errorsJoin(errs ...error) error {
	var filtered []error
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	msg := make([]string, 0, len(filtered))
	for _, err := range filtered {
		msg = append(msg, err.Error())
	}
	return errors.New(strings.Join(msg, "; "))
}
