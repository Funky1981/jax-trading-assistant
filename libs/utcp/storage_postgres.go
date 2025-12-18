package utcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) (*PostgresStorage, error) {
	if db == nil {
		return nil, errors.New("postgres storage: db is nil")
	}
	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) SaveEvent(ctx context.Context, e StoredEvent) error {
	if strings.TrimSpace(e.ID) == "" {
		return errors.New("storage.save_event: event.id is required")
	}
	if strings.TrimSpace(e.Symbol) == "" {
		return errors.New("storage.save_event: event.symbol is required")
	}
	if strings.TrimSpace(e.Type) == "" {
		return errors.New("storage.save_event: event.type is required")
	}
	if e.Time.IsZero() {
		return errors.New("storage.save_event: event.time is required")
	}

	payloadJSON, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("storage.save_event: marshal payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO events (id, symbol, type, time, payload)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
  symbol = EXCLUDED.symbol,
  type = EXCLUDED.type,
  time = EXCLUDED.time,
  payload = EXCLUDED.payload
`, e.ID, e.Symbol, e.Type, e.Time.UTC(), payloadJSON)
	if err != nil {
		return fmt.Errorf("storage.save_event: %w", err)
	}
	return nil
}

func (s *PostgresStorage) SaveTrade(ctx context.Context, trade StoredTrade, risk *StoredRisk, event *StoredEvent) error {
	if strings.TrimSpace(trade.ID) == "" {
		return errors.New("storage.save_trade: trade.id is required")
	}
	if strings.TrimSpace(trade.Symbol) == "" {
		return errors.New("storage.save_trade: trade.symbol is required")
	}
	if strings.TrimSpace(trade.Direction) == "" {
		return errors.New("storage.save_trade: trade.direction is required")
	}
	if strings.TrimSpace(trade.StrategyID) == "" {
		return errors.New("storage.save_trade: trade.strategyId is required")
	}

	targetsJSON, err := json.Marshal(trade.Targets)
	if err != nil {
		return fmt.Errorf("storage.save_trade: marshal targets: %w", err)
	}

	var riskJSON []byte
	if risk != nil {
		riskJSON, err = json.Marshal(risk)
		if err != nil {
			return fmt.Errorf("storage.save_trade: marshal risk: %w", err)
		}
	}

	if event != nil {
		if err := s.SaveEvent(ctx, *event); err != nil {
			return err
		}
		if trade.EventID == "" {
			trade.EventID = event.ID
		}
	}

	createdAt := trade.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO trades (id, symbol, direction, entry, stop, targets, event_id, strategy_id, notes, risk, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (id) DO UPDATE SET
  symbol = EXCLUDED.symbol,
  direction = EXCLUDED.direction,
  entry = EXCLUDED.entry,
  stop = EXCLUDED.stop,
  targets = EXCLUDED.targets,
  event_id = EXCLUDED.event_id,
  strategy_id = EXCLUDED.strategy_id,
  notes = EXCLUDED.notes,
  risk = EXCLUDED.risk
`, trade.ID, trade.Symbol, trade.Direction, trade.Entry, trade.Stop, targetsJSON, nullIfEmpty(trade.EventID), trade.StrategyID, nullIfEmpty(trade.Notes), nullBytesIfEmpty(riskJSON), createdAt)
	if err != nil {
		return fmt.Errorf("storage.save_trade: %w", err)
	}
	return nil
}

func (s *PostgresStorage) GetTrade(ctx context.Context, id string) (GetTradeOutput, error) {
	if strings.TrimSpace(id) == "" {
		return GetTradeOutput{}, errors.New("storage.get_trade: id is required")
	}

	var trade StoredTrade
	var targetsJSON []byte
	var riskJSON []byte
	var eventID sql.NullString
	var notes sql.NullString
	var createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
SELECT id, symbol, direction, entry, stop, targets, event_id, strategy_id, notes, risk, created_at
FROM trades
WHERE id = $1
`, id).Scan(&trade.ID, &trade.Symbol, &trade.Direction, &trade.Entry, &trade.Stop, &targetsJSON, &eventID, &trade.StrategyID, &notes, &riskJSON, &createdAt)
	if err != nil {
		return GetTradeOutput{}, fmt.Errorf("storage.get_trade: %w", err)
	}

	if err := json.Unmarshal(targetsJSON, &trade.Targets); err != nil {
		return GetTradeOutput{}, fmt.Errorf("storage.get_trade: decode targets: %w", err)
	}
	trade.CreatedAt = createdAt
	if eventID.Valid {
		trade.EventID = eventID.String
	}
	if notes.Valid {
		trade.Notes = notes.String
	}

	var risk *StoredRisk
	if len(riskJSON) > 0 {
		var r StoredRisk
		if err := json.Unmarshal(riskJSON, &r); err != nil {
			return GetTradeOutput{}, fmt.Errorf("storage.get_trade: decode risk: %w", err)
		}
		risk = &r
	}

	var event *StoredEvent
	if trade.EventID != "" {
		e, err := s.getEvent(ctx, trade.EventID)
		if err != nil {
			return GetTradeOutput{}, err
		}
		event = &e
	}

	return GetTradeOutput{Trade: trade, Risk: risk, Event: event}, nil
}

func (s *PostgresStorage) ListTrades(ctx context.Context, in ListTradesInput) (ListTradesOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = DefaultListTradesLimit
	}

	where := make([]string, 0, 2)
	args := make([]any, 0, 3)
	argN := 1

	if strings.TrimSpace(in.Symbol) != "" {
		where = append(where, fmt.Sprintf("symbol = $%d", argN))
		args = append(args, in.Symbol)
		argN++
	}
	if strings.TrimSpace(in.StrategyID) != "" {
		where = append(where, fmt.Sprintf("strategy_id = $%d", argN))
		args = append(args, in.StrategyID)
		argN++
	}

	query := `
SELECT id, symbol, direction, entry, stop, targets, event_id, strategy_id, notes, risk, created_at
FROM trades
`
	if len(where) > 0 {
		query += "WHERE " + strings.Join(where, " AND ") + "\n"
	}
	query += fmt.Sprintf("ORDER BY created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListTradesOutput{}, fmt.Errorf("storage.list_trades: %w", err)
	}
	defer rows.Close()

	var out ListTradesOutput
	for rows.Next() {
		var trade StoredTrade
		var targetsJSON []byte
		var riskJSON []byte
		var eventID sql.NullString
		var notes sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&trade.ID, &trade.Symbol, &trade.Direction, &trade.Entry, &trade.Stop, &targetsJSON, &eventID, &trade.StrategyID, &notes, &riskJSON, &createdAt); err != nil {
			return ListTradesOutput{}, fmt.Errorf("storage.list_trades: %w", err)
		}

		if err := json.Unmarshal(targetsJSON, &trade.Targets); err != nil {
			return ListTradesOutput{}, fmt.Errorf("storage.list_trades: decode targets: %w", err)
		}
		trade.CreatedAt = createdAt
		if eventID.Valid {
			trade.EventID = eventID.String
		}
		if notes.Valid {
			trade.Notes = notes.String
		}

		var risk *StoredRisk
		if len(riskJSON) > 0 {
			var r StoredRisk
			if err := json.Unmarshal(riskJSON, &r); err != nil {
				return ListTradesOutput{}, fmt.Errorf("storage.list_trades: decode risk: %w", err)
			}
			risk = &r
		}

		out.Trades = append(out.Trades, GetTradeOutput{Trade: trade, Risk: risk})
	}
	if err := rows.Err(); err != nil {
		return ListTradesOutput{}, fmt.Errorf("storage.list_trades: %w", err)
	}
	return out, nil
}

func (s *PostgresStorage) getEvent(ctx context.Context, id string) (StoredEvent, error) {
	var e StoredEvent
	var payloadJSON []byte
	err := s.db.QueryRowContext(ctx, `
SELECT id, symbol, type, time, payload
FROM events
WHERE id = $1
`, id).Scan(&e.ID, &e.Symbol, &e.Type, &e.Time, &payloadJSON)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("storage.get_event: %w", err)
	}
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &e.Payload); err != nil {
			return StoredEvent{}, fmt.Errorf("storage.get_event: decode payload: %w", err)
		}
	}
	return e, nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullBytesIfEmpty(b []byte) any {
	if len(b) == 0 {
		return []byte(nil)
	}
	return b
}
