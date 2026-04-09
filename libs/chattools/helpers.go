package chattools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func rowQueryJSON(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (json.RawMessage, error) {
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	var raw []byte
	if err := pool.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, fmt.Errorf("not found: %v", err)
	}
	return json.RawMessage(raw), nil
}

func marshalJSON(data any) (json.RawMessage, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
