package macroevents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) FindBySourceEventID(ctx context.Context, source, sourceEventID string) (StoredEvent, bool, error) {
	var event StoredEvent
	if s == nil || s.pool == nil {
		return StoredEvent{}, false, nil
	}
	var rejectionReason *string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, status, rejection_reason
		FROM macro_events
		WHERE source = $1 AND source_event_id = $2
	`, source, sourceEventID).Scan(&event.ID, &event.Status, &rejectionReason)
	if err != nil {
		if err == pgx.ErrNoRows {
			return StoredEvent{}, false, nil
		}
		return StoredEvent{}, false, fmt.Errorf("lookup macro event duplicate: %w", err)
	}
	if rejectionReason != nil {
		event.RejectionReason = *rejectionReason
	}
	mappings, err := s.loadMappings(ctx, event.ID)
	if err != nil {
		return StoredEvent{}, false, err
	}
	event.Mappings = mappings
	return event, true, nil
}

func (s *Store) Save(ctx context.Context, event StoredEvent) (StoredEvent, error) {
	if s == nil || s.pool == nil {
		if event.ID == "" {
			event.ID = uuid.NewString()
		}
		return event, nil
	}

	rawPayload := event.Input.RawPayload
	if rawPayload == nil {
		rawPayload = map[string]any{}
	}
	rawPayloadJSON, err := json.Marshal(rawPayload)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("marshal macro event raw payload: %w", err)
	}
	surpriseValue, surprisePercent := ComputeSurprise(event.Input.ActualValue, event.Input.ExpectedValue)
	id := uuid.NewString()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("begin macro event insert: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO macro_events (
			id, source, source_event_id, event_type, region, event_time_utc,
			headline, summary, actual_value, expected_value, previous_value, unit,
			surprise_value, surprise_percent, direction, confidence, raw_payload,
			status, rejection_reason
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6,
			$7, NULLIF($8, ''), $9, $10, $11, NULLIF($12, ''),
			$13, $14, $15, $16, $17::jsonb,
			$18, NULLIF($19, '')
		)
		ON CONFLICT (source, source_event_id) DO UPDATE SET
			updated_at = NOW()
		RETURNING id::text, status, COALESCE(rejection_reason, '')
	`, id, event.Input.Source, event.Input.SourceEventID, event.Input.EventType, event.Input.Region, event.Input.EventTimeUTC.UTC(),
		event.Input.Headline, event.Input.Summary, event.Input.ActualValue, event.Input.ExpectedValue, event.Input.PreviousValue, event.Input.Unit,
		surpriseValue, surprisePercent, event.Input.Direction, event.Input.Confidence, string(rawPayloadJSON), event.Status, event.RejectionReason,
	).Scan(&event.ID, &event.Status, &event.RejectionReason)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("insert macro event: %w", err)
	}

	for _, mapping := range event.Mappings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO macro_event_etf_map (
				macro_event_id, symbol, theme, mapping_reason, confidence
			)
			VALUES ($1::uuid, $2, $3, $4, $5)
			ON CONFLICT (macro_event_id, symbol) DO UPDATE SET
				theme = EXCLUDED.theme,
				mapping_reason = EXCLUDED.mapping_reason,
				confidence = EXCLUDED.confidence
		`, event.ID, mapping.Symbol, mapping.Theme, mapping.MappingReason, mapping.Confidence); err != nil {
			return StoredEvent{}, fmt.Errorf("insert macro ETF mapping: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return StoredEvent{}, fmt.Errorf("commit macro event insert: %w", err)
	}
	return event, nil
}

func (s *Store) loadMappings(ctx context.Context, macroEventID string) ([]ETFMapping, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT symbol, theme, mapping_reason, confidence::float8
		FROM macro_event_etf_map
		WHERE macro_event_id = $1::uuid
		ORDER BY symbol
	`, macroEventID)
	if err != nil {
		return nil, fmt.Errorf("load macro ETF mappings: %w", err)
	}
	defer rows.Close()

	mappings := []ETFMapping{}
	for rows.Next() {
		var mapping ETFMapping
		if err := rows.Scan(&mapping.Symbol, &mapping.Theme, &mapping.MappingReason, &mapping.Confidence); err != nil {
			return nil, fmt.Errorf("scan macro ETF mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate macro ETF mappings: %w", err)
	}
	return mappings, nil
}
