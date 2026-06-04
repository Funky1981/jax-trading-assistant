package llmcontext

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMemoryRetrieverLimitsAndFiltersArtifacts(t *testing.T) {
	store := NewMemoryArtifactStore()
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		err := store.Save(context.Background(), MemoryArtifact{
			ID:         string(rune('a' + i)),
			Summary:    "memory",
			SourceIDs:  []string{"src"},
			CreatedAt:  now.Add(time.Duration(-i) * time.Hour),
			Quality:    float64(i),
			TaskType:   TaskHistoricalSummary,
			Symbol:     "SPY",
			StrategyID: "strat",
		})
		if err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
	}
	got, err := store.Retrieve(context.Background(), MemoryQuery{
		TaskType:   TaskHistoricalSummary,
		Symbol:     "SPY",
		StrategyID: "strat",
		Limit:      3,
		Now:        now,
		MaxAge:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 memories, got %d", len(got))
	}
	if got[0].Quality < got[1].Quality {
		t.Fatalf("expected highest-quality memories first, got %#v", got)
	}
}

func TestMemoryArtifactStoreRequiresSourceIDs(t *testing.T) {
	store := NewMemoryArtifactStore()
	err := store.Save(context.Background(), MemoryArtifact{ID: "mem-1", Summary: "missing sources"})
	if err == nil {
		t.Fatal("expected missing source IDs to fail")
	}
}

func TestPostgresMemoryArtifactStorePersistsSources(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	store := NewPostgresMemoryArtifactStore(db)
	created := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_memory_artifacts")).
		WithArgs("mem-1", TaskHistoricalSummary, "SPY", "strat", "summary", sqlmock.AnyArg(), 0.9, created).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.Save(context.Background(), MemoryArtifact{
		ID:         "mem-1",
		TaskType:   TaskHistoricalSummary,
		Symbol:     "SPY",
		StrategyID: "strat",
		Summary:    "summary",
		SourceIDs:  []string{"src-1"},
		Quality:    0.9,
		CreatedAt:  created,
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLLMMemoryArtifactMigration(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "postgres", "migrations", "000028_llm_memory_artifacts.up.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS llm_memory_artifacts",
		"source_ids JSONB NOT NULL",
		"quality DOUBLE PRECISION NOT NULL",
		"idx_llm_memory_artifacts_lookup",
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).Match(data) {
			t.Fatalf("migration missing %q", want)
		}
	}
}
