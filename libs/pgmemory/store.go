// Package pgmemory provides a PostgreSQL/pgvector-backed MemoryStore.
// It is part of the jax-trading-assistant root module (no separate go.mod).
//
// Usage:
//
//	store := pgmemory.New(db, pgmemory.NewOpenAIEmbedder(cfg))
//	id, err := store.Retain(ctx, "research", item)
package pgmemory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// validBanks is the fixed set of memory banks for phase 1.
var validBanks = map[string]bool{
	"research":    true,
	"trades":      true,
	"signals":     true,
	"reflections": true,
}

// Banks returns the ordered list of valid memory bank names.
func Banks() []string {
	return []string{"research", "trades", "signals", "reflections"}
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Store is a PostgreSQL/pgvector-backed implementation of contracts.MemoryStore.
type Store struct {
	db       *sql.DB
	embedder EmbeddingProvider
}

// New creates a Store that uses db for persistence and embedder for vector generation.
func New(db *sql.DB, embedder EmbeddingProvider) *Store {
	return &Store{db: db, embedder: embedder}
}

var _ contracts.MemoryStore = (*Store)(nil)

// Ping checks database connectivity.
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Retain inserts a MemoryItem into the named bank.
// Embedding is generated synchronously from item.Summary.
// If embedding generation fails the item is NOT written and an error is returned.
func (s *Store) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	bank = strings.TrimSpace(bank)
	if !validBanks[bank] {
		return "", fmt.Errorf("retain: unknown bank %q", bank)
	}
	item.Tags = contracts.NormalizeMemoryTags(item.Tags)
	if item.TS.IsZero() {
		item.TS = time.Now().UTC()
	}
	item.Summary = strings.TrimSpace(item.Summary)
	if err := contracts.ValidateMemoryItem(item); err != nil {
		return "", fmt.Errorf("retain: %w", err)
	}

	// Embedding is generated synchronously; any failure aborts the write.
	vec, err := s.embedder.Embed(ctx, item.Summary)
	if err != nil {
		return "", fmt.Errorf("retain: embedding failed: %w", err)
	}

	id := strings.TrimSpace(item.ID)
	if id == "" {
		id = uuid.New().String()
	}

	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return "", fmt.Errorf("retain: marshal tags: %w", err)
	}

	var dataJSON []byte
	if item.Data != nil {
		dataJSON, err = json.Marshal(item.Data)
		if err != nil {
			return "", fmt.Errorf("retain: marshal data: %w", err)
		}
	}

	var sourceSystem, sourceRef sql.NullString
	if item.Source != nil {
		sourceSystem = sql.NullString{String: item.Source.System, Valid: item.Source.System != ""}
		sourceRef = sql.NullString{String: item.Source.Ref, Valid: item.Source.Ref != ""}
	}

	const insertSQL = `
		INSERT INTO memory_items
			(id, bank, ts, type, symbol, summary, tags, data, source_system, source_ref, embedding)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::vector)
	`
	_, err = s.db.ExecContext(ctx, insertSQL,
		id,
		bank,
		item.TS.UTC(),
		item.Type,
		sql.NullString{String: item.Symbol, Valid: item.Symbol != ""},
		item.Summary,
		tagsJSON,
		dataJSON,
		sourceSystem,
		sourceRef,
		formatVector(vec),
	)
	if err != nil {
		if isDuplicateMemorySource(err) {
			return "", fmt.Errorf("retain: duplicate source reference for bank %q", bank)
		}
		return "", fmt.Errorf("retain: insert: %w", err)
	}

	return contracts.MemoryID(id), nil
}

// Recall retrieves memory items from a bank applying structured filters.
// If query.Q is non-empty a vector similarity search is performed.
// If query.Q is empty a structured filter with ORDER BY ts DESC is used.
// Bank is always required.
func (s *Store) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	bank = strings.TrimSpace(bank)
	if bank == "" {
		return nil, fmt.Errorf("recall: bank is required")
	}
	if !validBanks[bank] {
		return nil, fmt.Errorf("recall: unknown bank %q", bank)
	}

	limit := clampLimit(query.Limit)
	q := strings.TrimSpace(query.Q)
	if q != "" {
		return s.vectorRecall(ctx, bank, query, q, limit)
	}
	return s.structuredRecall(ctx, bank, query, limit)
}

func (s *Store) structuredRecall(ctx context.Context, bank string, query contracts.MemoryQuery, limit int) ([]contracts.MemoryItem, error) {
	conditions := []string{"bank = $1"}
	args := []any{bank}
	idx := 2

	idx = applyFilters(&conditions, &args, idx, query)

	args = append(args, limit)
	querySQL := fmt.Sprintf(`
		SELECT id, ts, type, symbol, summary, tags, data, source_system, source_ref
		FROM memory_items
		WHERE %s
		ORDER BY ts DESC
		LIMIT $%d
	`, strings.Join(conditions, " AND "), idx)

	return s.scanItems(ctx, querySQL, args...)
}

func (s *Store) vectorRecall(ctx context.Context, bank string, query contracts.MemoryQuery, q string, limit int) ([]contracts.MemoryItem, error) {
	vec, err := s.embedder.Embed(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("recall: embed query: %w", err)
	}

	conditions := []string{"bank = $1", "embedding IS NOT NULL"}
	args := []any{bank}
	idx := 2

	idx = applyFilters(&conditions, &args, idx, query)

	vecIdx := idx
	args = append(args, formatVector(vec))
	idx++
	args = append(args, limit)

	querySQL := fmt.Sprintf(`
		SELECT id, ts, type, symbol, summary, tags, data, source_system, source_ref
		FROM memory_items
		WHERE %s
		ORDER BY embedding <=> $%d::vector
		LIMIT $%d
	`, strings.Join(conditions, " AND "), vecIdx, idx)

	return s.scanItems(ctx, querySQL, args...)
}

// Reflect recalls the top N most relevant items and returns a single ephemeral
// synthesized summary item. The result is never persisted.
func (s *Store) Reflect(ctx context.Context, bank string, params contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	bank = strings.TrimSpace(bank)
	if bank == "" {
		return nil, fmt.Errorf("reflect: bank is required")
	}
	if !validBanks[bank] {
		return nil, fmt.Errorf("reflect: unknown bank %q", bank)
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("reflect: params.query is required")
	}

	query := contracts.MemoryQuery{
		Q:     params.Query,
		Limit: 10,
	}
	if params.WindowDays > 0 {
		from := time.Now().UTC().AddDate(0, 0, -params.WindowDays)
		query.From = &from
	}

	recalled, err := s.Recall(ctx, bank, query)
	if err != nil {
		return nil, fmt.Errorf("reflect: recall: %w", err)
	}

	typeCounts := countBy(recalled, func(item contracts.MemoryItem) string { return item.Type })
	symbolCounts := countBy(recalled, func(item contracts.MemoryItem) string { return item.Symbol })
	tagCounts := countTags(recalled)

	summaryParts := []string{
		fmt.Sprintf("Reflection over %d %s memories for %q.", len(recalled), bank, strings.TrimSpace(params.Query)),
	}
	if params.WindowDays > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Window: last %d days.", params.WindowDays))
	}
	if repeated := formatCounts("Repeated types", typeCounts); repeated != "" {
		summaryParts = append(summaryParts, repeated)
	}
	if repeated := formatCounts("Repeated symbols", symbolCounts); repeated != "" {
		summaryParts = append(summaryParts, repeated)
	}
	if repeated := formatCounts("Repeated tags", tagCounts); repeated != "" {
		summaryParts = append(summaryParts, repeated)
	}
	if hint := strings.TrimSpace(params.PromptHint); hint != "" {
		summaryParts = append(summaryParts, fmt.Sprintf("Prompt hint: %s.", hint))
	}
	if len(recalled) == 0 {
		summaryParts = append(summaryParts, "No prior memories matched.")
	}

	return []contracts.MemoryItem{
		{
			TS:      time.Now().UTC(),
			Type:    "reflection",
			Summary: strings.Join(summaryParts, " "),
			Data: map[string]any{
				"query":            strings.TrimSpace(params.Query),
				"matched_count":    len(recalled),
				"source_bank":      bank,
				"window_days":      params.WindowDays,
				"repeated_types":   typeCounts,
				"repeated_symbols": symbolCounts,
				"repeated_tags":    tagCounts,
			},
			Source: &contracts.MemorySource{System: "pgmemory"},
		},
	}, nil
}

// GetByID retrieves a single memory item by its ID within a bank.
// This method is not part of contracts.MemoryStore and is used by the REST handler.
func (s *Store) GetByID(ctx context.Context, bank, id string) (contracts.MemoryItem, error) {
	bank = strings.TrimSpace(bank)
	id = strings.TrimSpace(id)
	if bank == "" || id == "" {
		return contracts.MemoryItem{}, fmt.Errorf("GetByID: bank and id are required")
	}
	if !validBanks[bank] {
		return contracts.MemoryItem{}, fmt.Errorf("GetByID: unknown bank %q", bank)
	}

	const q = `
		SELECT id, ts, type, symbol, summary, tags, data, source_system, source_ref
		FROM memory_items
		WHERE bank = $1 AND id = $2
		LIMIT 1
	`
	items, err := s.scanItems(ctx, q, bank, id)
	if err != nil {
		return contracts.MemoryItem{}, err
	}
	if len(items) == 0 {
		return contracts.MemoryItem{}, fmt.Errorf("GetByID: not found: %s/%s", bank, id)
	}
	return items[0], nil
}

// ── internals ─────────────────────────────────────────────────────────────────

// applyFilters appends WHERE clauses and args for structured fields.
// Returns the next placeholder index.
func applyFilters(conditions *[]string, args *[]any, idx int, query contracts.MemoryQuery) int {
	if query.Symbol != "" {
		*conditions = append(*conditions, fmt.Sprintf("symbol = $%d", idx))
		*args = append(*args, strings.TrimSpace(query.Symbol))
		idx++
	}
	if len(query.Types) > 0 {
		*conditions = append(*conditions, fmt.Sprintf("type = ANY($%d)", idx))
		*args = append(*args, query.Types)
		idx++
	}
	if query.From != nil {
		*conditions = append(*conditions, fmt.Sprintf("ts >= $%d", idx))
		*args = append(*args, query.From.UTC())
		idx++
	}
	if query.To != nil {
		*conditions = append(*conditions, fmt.Sprintf("ts <= $%d", idx))
		*args = append(*args, query.To.UTC())
		idx++
	}
	if len(query.Tags) > 0 {
		tagsJSON, _ := json.Marshal(query.Tags)
		*conditions = append(*conditions, fmt.Sprintf("tags @> $%d", idx))
		*args = append(*args, tagsJSON)
		idx++
	}
	return idx
}

func (s *Store) scanItems(ctx context.Context, query string, args ...any) ([]contracts.MemoryItem, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgmemory: query: %w", err)
	}
	defer rows.Close()

	var items []contracts.MemoryItem
	for rows.Next() {
		var (
			item         contracts.MemoryItem
			symbol       sql.NullString
			tagsJSON     []byte
			dataJSON     []byte
			sourceSystem sql.NullString
			sourceRef    sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.TS,
			&item.Type,
			&symbol,
			&item.Summary,
			&tagsJSON,
			&dataJSON,
			&sourceSystem,
			&sourceRef,
		); err != nil {
			return nil, fmt.Errorf("pgmemory: scan: %w", err)
		}
		item.TS = item.TS.UTC()
		item.Symbol = symbol.String

		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &item.Tags); err != nil {
				return nil, fmt.Errorf("pgmemory: unmarshal tags: %w", err)
			}
		}
		if dataJSON != nil {
			if err := json.Unmarshal(dataJSON, &item.Data); err != nil {
				return nil, fmt.Errorf("pgmemory: unmarshal data: %w", err)
			}
		}
		if sourceSystem.Valid || sourceRef.Valid {
			item.Source = &contracts.MemorySource{
				System: sourceSystem.String,
				Ref:    sourceRef.String,
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgmemory: rows: %w", err)
	}
	return items, nil
}

// formatVector formats a float32 slice as a pgvector literal: "[f1,f2,...,fN]".
func formatVector(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', 8, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func clampLimit(n int) int {
	if n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func uniqueTypes(items []contracts.MemoryItem) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range items {
		if !seen[item.Type] {
			seen[item.Type] = true
			out = append(out, item.Type)
		}
	}
	return out
}

func isDuplicateMemorySource(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505"
}

func countBy(items []contracts.MemoryItem, keyFn func(contracts.MemoryItem) string) map[string]int {
	out := make(map[string]int)
	for _, item := range items {
		key := strings.TrimSpace(keyFn(item))
		if key == "" {
			continue
		}
		out[key]++
	}
	return out
}

func countTags(items []contracts.MemoryItem) map[string]int {
	out := make(map[string]int)
	for _, item := range items {
		for _, tag := range item.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			out[tag]++
		}
	}
	return out
}

func formatCounts(label string, counts map[string]int) string {
	var repeated []string
	for key, count := range counts {
		if count < 2 {
			continue
		}
		repeated = append(repeated, fmt.Sprintf("%s (%d)", key, count))
	}
	if len(repeated) == 0 {
		return ""
	}
	return fmt.Sprintf("%s: %s.", label, strings.Join(repeated, ", "))
}
