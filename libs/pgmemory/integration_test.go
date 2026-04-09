//go:build integration
// +build integration

// Integration tests for pgmemory.Store.
// Requires a running Postgres instance with pgvector installed and a
// migration-applied schema (run 000020_memory_items.up.sql first).
//
// Run with:
//
//	TEST_DATABASE_URL="postgresql://jax:jax@localhost:5433/jax?sslmode=disable" \
//	  go test -tags integration ./libs/pgmemory/... -v
//
// The tests create and clean up their own rows via unique IDs so they can run
// safely against a shared dev database.
package pgmemory

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"jax-trading-assistant/libs/contracts"
)

// ── fixtures ──────────────────────────────────────────────────────────────────

// noopEmbedder returns a fixed 1536-element zero vector.
// It lets integration tests run without an OpenAI key.
type noopEmbedder struct{}

func (n *noopEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	v := make([]float32, 1536)
	for i := range v {
		v[i] = float32(i) * 0.0001
	}
	return v, nil
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("postgres not available (%v) — set TEST_DATABASE_URL to run", err)
	}
	return db
}

func newTestStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	db := openTestDB(t)
	store := New(db, &noopEmbedder{})
	return store, db
}

// cleanupByPrefix deletes any memory_items whose id starts with prefix.
func cleanupByPrefix(t *testing.T, db *sql.DB, prefix string) {
	t.Helper()
	_, err := db.ExecContext(context.Background(),
		"DELETE FROM memory_items WHERE id LIKE $1", prefix+"%")
	if err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func testItem(suffix string) contracts.MemoryItem {
	return contracts.MemoryItem{
		ID:      fmt.Sprintf("pgmem-integ-%s", suffix),
		TS:      time.Now().UTC(),
		Type:    "signal",
		Symbol:  "AAPL",
		Summary: fmt.Sprintf("MACD crossover on AAPL daily chart (%s)", suffix),
		Tags:    []string{"macd", "aapl"},
		Data:    map[string]any{"confidence": 0.9},
		Source:  &contracts.MemorySource{System: "integration-test"},
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestIntegration_Ping(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()

	if err := store.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestIntegration_Retain(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-retain-")

	item := testItem("retain-001")
	id, err := store.Retain(context.Background(), "research", item)
	if err != nil {
		t.Fatalf("Retain: %v", err)
	}
	if string(id) == "" {
		t.Error("Retain returned empty ID")
	}
	t.Logf("retained id=%s", id)
}

func TestIntegration_Retain_MissingType(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()

	bad := testItem("bad-type")
	bad.Type = ""
	_, err := store.Retain(context.Background(), "research", bad)
	if err == nil {
		t.Error("expected error for missing type, got nil")
	}
}

func TestIntegration_Retain_UnknownBank(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()

	_, err := store.Retain(context.Background(), "invalid-bank", testItem("x"))
	if err == nil || !strings.Contains(err.Error(), "unknown bank") {
		t.Errorf("expected unknown bank error, got: %v", err)
	}
}

func TestIntegration_StructuredRecall(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-recall-")

	for i := 0; i < 3; i++ {
		item := testItem(fmt.Sprintf("recall-%03d", i))
		if _, err := store.Retain(context.Background(), "research", item); err != nil {
			t.Fatalf("setup Retain %d: %v", i, err)
		}
	}

	items, err := store.Recall(context.Background(), "research", contracts.MemoryQuery{
		Symbol: "AAPL",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(items) < 3 {
		t.Errorf("expected >= 3 items, got %d", len(items))
	}
	t.Logf("structured recall returned %d items", len(items))
}

func TestIntegration_VectorRecall(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-vec-")

	for i := 0; i < 3; i++ {
		item := testItem(fmt.Sprintf("vec-%03d", i))
		if _, err := store.Retain(context.Background(), "signals", item); err != nil {
			t.Fatalf("setup Retain %d: %v", i, err)
		}
	}

	items, err := store.Recall(context.Background(), "signals", contracts.MemoryQuery{
		Q:     "MACD crossover AAPL",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("vector Recall: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least 1 result from vector recall")
	}
	t.Logf("vector recall returned %d items", len(items))
}

func TestIntegration_TagFilter(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-tag-")

	tagged := testItem("tag-001")
	tagged.Tags = []string{"rare-tag-xyz", "aapl"}
	if _, err := store.Retain(context.Background(), "research", tagged); err != nil {
		t.Fatalf("Retain tagged: %v", err)
	}

	items, err := store.Recall(context.Background(), "research", contracts.MemoryQuery{
		Tags:  []string{"rare-tag-xyz"},
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Recall by tag: %v", err)
	}
	if len(items) == 0 {
		t.Error("expected at least 1 result for tag filter")
	}
	for _, it := range items {
		found := false
		for _, tg := range it.Tags {
			if tg == "rare-tag-xyz" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("item %s missing expected tag", it.ID)
		}
	}
}

func TestIntegration_LimitClamping(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()

	items, err := store.Recall(context.Background(), "research", contracts.MemoryQuery{
		Limit: 9999,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(items) > maxLimit {
		t.Errorf("expected <= %d items (max limit), got %d", maxLimit, len(items))
	}
}

func TestIntegration_GetByID(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-get-")

	item := testItem("get-001")
	id, err := store.Retain(context.Background(), "trades", item)
	if err != nil {
		t.Fatalf("Retain: %v", err)
	}

	got, err := store.GetByID(context.Background(), "trades", string(id))
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != string(id) {
		t.Errorf("GetByID id = %q, want %q", got.ID, id)
	}
	if got.Summary == "" {
		t.Error("GetByID returned item with empty Summary")
	}
}

func TestIntegration_GetByID_NotFound(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()

	_, err := store.GetByID(context.Background(), "research", "nonexistent-id-xyz")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestIntegration_Reflect(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-refl-")

	for i := 0; i < 2; i++ {
		item := testItem(fmt.Sprintf("refl-%03d", i))
		if _, err := store.Retain(context.Background(), "reflections", item); err != nil {
			t.Fatalf("setup Retain %d: %v", i, err)
		}
	}

	out, err := store.Reflect(context.Background(), "reflections", contracts.ReflectionParams{
		Query: "AAPL patterns",
	})
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("Reflect returned %d items, want 1", len(out))
	}
	if out[0].Type != "reflection" {
		t.Errorf("Reflect item type = %q, want reflection", out[0].Type)
	}
	t.Logf("reflect summary: %s", out[0].Summary)
}

func TestIntegration_DateFilter(t *testing.T) {
	store, db := newTestStore(t)
	defer db.Close()
	defer cleanupByPrefix(t, db, "pgmem-integ-date-")

	item := testItem("date-001")
	item.TS = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := store.Retain(context.Background(), "research", item); err != nil {
		t.Fatalf("Retain: %v", err)
	}

	// Query with From after the item's ts — should return nothing.
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	items, err := store.Recall(context.Background(), "research", contracts.MemoryQuery{
		Symbol: "AAPL",
		From:   &from,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("Recall with from filter: %v", err)
	}
	for _, it := range items {
		if it.ID == item.ID {
			t.Errorf("item from 2025 appeared in from=2026 query")
		}
	}
}
