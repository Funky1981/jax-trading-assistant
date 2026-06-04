package llmcontext

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresUsageLoggerRecordPlannedInsertsUsageLog(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	logger := NewPostgresUsageLogger(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_usage_logs")).
		WithArgs(
			TaskEvidenceBundleSummary,
			"local-small",
			"ollama/local-small",
			120,
			40,
			0,
			0.012,
			0.0,
			false,
			true,
			"jax-paper-trading",
			"evt-1",
			"cand-1",
			"strat-1",
			"SPY",
			"corr-1",
			false,
			BlockReasonNone,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := logger.RecordPlanned(context.Background(), UsageRecord{
		TaskType:              TaskEvidenceBundleSummary,
		ModelAlias:            "local-small",
		ProviderModel:         "ollama/local-small",
		EstimatedInputTokens:  120,
		EstimatedOutputTokens: 40,
		CachedInputTokens:     0,
		EstimatedCostUSD:      0.012,
		ActualCostUSD:         0,
		CacheHit:              false,
		CacheEligible:         true,
		VirtualKey:            "jax-paper-trading",
		EventID:               "evt-1",
		CandidateID:           "cand-1",
		StrategyID:            "strat-1",
		Symbol:                "SPY",
		CorrelationID:         "corr-1",
		Blocked:               false,
		BlockReason:           BlockReasonNone,
	})
	if err != nil {
		t.Fatalf("RecordPlanned returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPostgresUsageLoggerRecordActualUpdatesUsageLog(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	logger := NewPostgresUsageLogger(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE llm_usage_logs")).
		WithArgs(111, 22, 133, 12, 0.0045, true, "corr-actual").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := logger.RecordActual(context.Background(), LLMResult{
		CorrelationID: "corr-actual",
		InputTokens:   111,
		OutputTokens:  22,
		CachedTokens:  12,
		ActualCostUSD: 0.0045,
	})
	if err != nil {
		t.Fatalf("RecordActual returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPostgresUsageLoggerUpsertRollup(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()
	logger := NewPostgresUsageLogger(db)
	from := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO llm_cost_rollups")).
		WithArgs(
			"strategy",
			"ETF_NEWS_002",
			7,
			3,
			1,
			500,
			80,
			0.031,
			4,
			2,
			120,
			from,
			to,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := logger.UpsertRollup(context.Background(), CostRollup{
		RollupType:          "strategy",
		RollupKey:           "ETF_NEWS_002",
		EventCount:          7,
		CandidateCount:      3,
		ApprovedCount:       1,
		TotalInputTokens:    500,
		TotalOutputTokens:   80,
		TotalCostUSD:        0.031,
		PaidCallsAvoided:    4,
		CacheHitCount:       2,
		HeadroomTokensSaved: 120,
		From:                from,
		To:                  to,
	})
	if err != nil {
		t.Fatalf("UpsertRollup returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return db, mock
}
