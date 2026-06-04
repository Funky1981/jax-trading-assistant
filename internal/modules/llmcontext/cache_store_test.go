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

func TestExactCacheKeyIsStableForSamePromptPackage(t *testing.T) {
	pkg := PromptPackage{
		TaskType:        TaskHistoricalSummary,
		Model:           "local-small",
		CacheablePrefix: "static",
		RetrievedMemory: "memory",
		DynamicContext:  "dynamic",
		ResponseSchema:  "schema",
	}
	first := ExactCacheKey(pkg)
	second := ExactCacheKey(pkg)
	if first == "" || first != second {
		t.Fatalf("expected stable cache key, got %q and %q", first, second)
	}
	pkg.DynamicContext = "different"
	if ExactCacheKey(pkg) == first {
		t.Fatal("expected dynamic context to affect exact cache key")
	}
}

func TestCachePolicyStoreRejectsUnsafeTasks(t *testing.T) {
	store := NewMemoryPromptCache(DefaultCachePolicy())
	err := store.Put(context.Background(), PromptPackage{
		TaskType:        TaskApprovalSummary,
		Model:           "local-small",
		CacheablePrefix: "static",
		DynamicContext:  "approval",
	}, LLMResult{Text: "do not cache"}, time.Hour)
	if err == nil {
		t.Fatal("expected approval summary cache write to fail")
	}
}

func TestMemoryPromptCacheRoundTrip(t *testing.T) {
	store := NewMemoryPromptCache(DefaultCachePolicy())
	pkg := PromptPackage{
		TaskType:        TaskHistoricalSummary,
		Model:           "local-small",
		CacheablePrefix: "static",
		DynamicContext:  "historical",
	}
	if err := store.Put(context.Background(), pkg, LLMResult{Text: "cached", InputTokens: 10}, time.Hour); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	got, ok, err := store.Get(context.Background(), pkg)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !ok || got.Text != "cached" || got.CachedTokens != 10 {
		t.Fatalf("unexpected cache result: ok=%v result=%#v", ok, got)
	}
}

func TestPostgresPromptCacheWritesAuditableEntry(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	store := NewPostgresPromptCache(db, DefaultCachePolicy())
	pkg := PromptPackage{
		TaskType:        TaskHistoricalSummary,
		Model:           "local-small",
		Provider:        "litellm",
		CacheablePrefix: "static",
		DynamicContext:  "historical",
		CorrelationID:   "corr-cache",
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_prompt_cache")).
		WithArgs(sqlmock.AnyArg(), TaskHistoricalSummary, "litellm", "local-small", "corr-cache", sqlmock.AnyArg(), "cached", 10, 2, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.Put(context.Background(), pkg, LLMResult{Text: "cached", InputTokens: 10, OutputTokens: 2}, time.Hour); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestLLMPromptCacheMigration(t *testing.T) {
	assertMigrationContains(t, "000027_llm_prompt_cache.up.sql", []string{
		"CREATE TABLE IF NOT EXISTS llm_prompt_cache",
		"cache_key TEXT PRIMARY KEY",
		"task_type TEXT NOT NULL",
		"source_hash TEXT NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"idx_llm_prompt_cache_expires_at",
	})
}

func assertMigrationContains(t *testing.T, path string, required []string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "db", "postgres", "migrations", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, want := range required {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).Match(data) {
			t.Fatalf("migration %s missing %q", path, want)
		}
	}
}
