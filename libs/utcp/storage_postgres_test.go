package utcp

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresStorage_SaveEvent_Upsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewPostgresStorage(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO events")).
		WithArgs("e1", "AAPL", "gap_open", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.SaveEvent(context.Background(), StoredEvent{
		ID:     "e1",
		Symbol: "AAPL",
		Type:   "gap_open",
		Time:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Payload: map[string]any{
			"gapPct": 5.0,
		},
	})
	if err != nil {
		t.Fatalf("save event: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostgresStorage_SaveTrade_Upsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewPostgresStorage(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO trades")).
		WithArgs("t1", "AAPL", "long", 100.0, 95.0, sqlmock.AnyArg(), sqlmock.AnyArg(), "earnings_gap_v1", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.SaveTrade(context.Background(), StoredTrade{
		ID:         "t1",
		Symbol:     "AAPL",
		Direction:  "long",
		Entry:      100,
		Stop:       95,
		Targets:    []float64{110, 115},
		StrategyID: "earnings_gap_v1",
	}, &StoredRisk{PositionSize: 60, TotalRisk: 300}, nil)
	if err != nil {
		t.Fatalf("save trade: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
