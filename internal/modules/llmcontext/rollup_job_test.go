package llmcontext

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresCostRollupJobAggregatesStrategySymbolAndTask(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()
	job := NewPostgresCostRollupJob(db)
	from := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	for _, rollupType := range []string{"strategy", "symbol", "task_type"} {
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_cost_rollups")).
			WithArgs(rollupType, from, to).
			WillReturnResult(sqlmock.NewResult(1, 2))
	}

	if err := job.RunWindow(context.Background(), from, to); err != nil {
		t.Fatalf("RunWindow returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestPostgresCostRollupJobRejectsInvalidWindow(t *testing.T) {
	job := NewPostgresCostRollupJob(nil)
	now := time.Now().UTC()
	if err := job.RunWindow(context.Background(), now, now); err == nil {
		t.Fatal("expected invalid window error")
	}
}
