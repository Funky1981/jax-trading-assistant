package utcp

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestStorageTools_SaveAndGetTrade_RoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewPostgresStorage(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	registry := NewLocalRegistry()
	if err := RegisterStorageTools(registry, store); err != nil {
		t.Fatalf("register: %v", err)
	}

	client, err := NewUTCPClient(ProvidersConfig{
		Providers: []ProviderConfig{{ID: StorageProviderID, Transport: "local"}},
	}, WithLocalRegistry(registry))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	svc := NewStorageService(client)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO trades")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := svc.SaveTrade(context.Background(), SaveTradeInput{
		Trade: StoredTrade{
			ID:         "t1",
			Symbol:     "AAPL",
			Direction:  "long",
			Entry:      100,
			Stop:       95,
			Targets:    []float64{110, 115},
			StrategyID: "earnings_gap_v1",
		},
	}); err != nil {
		t.Fatalf("save trade: %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, symbol, direction, entry, stop, targets, event_id, strategy_id, notes, risk, created_at")).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "symbol", "direction", "entry", "stop", "targets", "event_id", "strategy_id", "notes", "risk", "created_at",
		}).AddRow(
			"t1", "AAPL", "long", 100.0, 95.0, `[110,115]`, nil, "earnings_gap_v1", nil, `{"positionSize":60,"totalRisk":300}`, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		))

	out, err := svc.GetTrade(context.Background(), "t1")
	if err != nil {
		t.Fatalf("get trade: %v", err)
	}
	if out.Trade.ID != "t1" || out.Trade.Symbol != "AAPL" {
		t.Fatalf("unexpected trade: %#v", out.Trade)
	}
	if out.Risk == nil || out.Risk.PositionSize != 60 {
		t.Fatalf("unexpected risk: %#v", out.Risk)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
