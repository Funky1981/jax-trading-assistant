package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTraceSink struct {
	pool *pgxpool.Pool
}

func NewPostgresTraceSink(pool *pgxpool.Pool) *PostgresTraceSink {
	if pool == nil {
		return nil
	}
	return &PostgresTraceSink{pool: pool}
}

func (s *PostgresTraceSink) WriteTrace(t Trace) error {
	if s == nil || s.pool == nil {
		return nil
	}
	t = redactTrace(t)

	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal trace payload: %w", err)
	}
	toolNames, err := json.Marshal(t.ToolNames)
	if err != nil {
		return fmt.Errorf("marshal trace tool names: %w", err)
	}
	validatorNotes, err := json.Marshal(t.ValidatorNotes)
	if err != nil {
		return fmt.Errorf("marshal trace validator notes: %w", err)
	}

	_, err = s.pool.Exec(context.Background(), `
		INSERT INTO harness_traces (
			trace_id, session_id, question, tool_names, validator_failures, payload, created_at
		) VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7)
		ON CONFLICT (trace_id) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			question = EXCLUDED.question,
			tool_names = EXCLUDED.tool_names,
			validator_failures = EXCLUDED.validator_failures,
			payload = EXCLUDED.payload,
			created_at = EXCLUDED.created_at
	`, t.TraceID, t.SessionID, t.Question, string(toolNames), string(validatorNotes), string(payload), t.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert harness trace: %w", err)
	}
	return nil
}

func (s *PostgresTraceSink) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("trace sink unavailable")
	}
	var payload []byte
	if err := s.pool.QueryRow(ctx, `SELECT payload FROM harness_traces WHERE trace_id = $1`, traceID).Scan(&payload); err != nil {
		return nil, err
	}
	var trace Trace
	if err := json.Unmarshal(payload, &trace); err != nil {
		return nil, fmt.Errorf("decode harness trace: %w", err)
	}
	return &trace, nil
}

func redactTrace(t Trace) Trace {
	t.Question = redactString(t.Question)
	t.FinalAnswer = redactString(t.FinalAnswer)
	t.ValidatorNotes = redactStringSlice(t.ValidatorNotes)
	for i := range t.ValidationAttempts {
		t.ValidationAttempts[i].Answer = redactString(t.ValidationAttempts[i].Answer)
		t.ValidationAttempts[i].Reasons = redactStringSlice(t.ValidationAttempts[i].Reasons)
	}
	for i := range t.ToolRuns {
		t.ToolRuns[i].Call.Args = redactJSON(t.ToolRuns[i].Call.Args)
		t.ToolRuns[i].Result = redactJSON(t.ToolRuns[i].Result)
		t.ToolRuns[i].Error = redactString(t.ToolRuns[i].Error)
	}
	return t
}

func redactStringSlice(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = redactString(value)
	}
	return out
}

func redactString(value string) string {
	lower := strings.ToLower(value)
	for _, token := range []string{"authorization", "bearer ", "api_key", "apikey", "token=", "password=", "secret="} {
		if strings.Contains(lower, token) {
			return "[redacted]"
		}
	}
	return value
}

func redactJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	redacted, changed := redactValue(value)
	if !changed {
		return raw
	}
	b, err := json.Marshal(redacted)
	if err != nil {
		return raw
	}
	return json.RawMessage(b)
}

func redactValue(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if shouldRedactKey(key) {
				out[key] = "[redacted]"
				changed = true
				continue
			}
			next, nextChanged := redactValue(item)
			out[key] = next
			changed = changed || nextChanged
		}
		return out, changed
	case []any:
		changed := false
		out := make([]any, len(typed))
		for i, item := range typed {
			next, nextChanged := redactValue(item)
			out[i] = next
			changed = changed || nextChanged
		}
		return out, changed
	case string:
		redacted := redactString(typed)
		return redacted, redacted != typed
	default:
		return value, false
	}
}

func shouldRedactKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	for _, candidate := range []string{
		"authorization",
		"api_key",
		"apikey",
		"token",
		"access_token",
		"refresh_token",
		"password",
		"secret",
		"database_url",
		"dsn",
		"connection_string",
	} {
		if lower == candidate {
			return true
		}
	}
	return false
}
