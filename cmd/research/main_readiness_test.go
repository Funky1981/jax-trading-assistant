package main

import (
	"context"
	"testing"

	"jax-trading-assistant/libs/pgmemory"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadMemoryReadiness_ReadyWhenSchemaIsCurrentAndComplete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(pgmemory.RequiredSchemaVersion, false))
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"memory_items",
			"unique_index",
			"hnsw_index",
			"bank_constraint",
			"summary_constraint",
			"tags_constraint",
			"data_constraint",
		}).AddRow(true, true, true, true, true, true, true))

	readiness := loadMemoryReadiness(context.Background(), db)
	if !readiness.Ready {
		t.Fatalf("expected memory readiness true, got %+v", readiness)
	}
	if readiness.Status != "ready" {
		t.Fatalf("expected ready status, got %q", readiness.Status)
	}
	if readiness.SchemaVersion != pgmemory.RequiredSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", pgmemory.RequiredSchemaVersion, readiness.SchemaVersion)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestLoadMemoryReadiness_NotReadyWhenSchemaVersionIsStale(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(pgmemory.RequiredSchemaVersion-1, false))
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"memory_items",
			"unique_index",
			"hnsw_index",
			"bank_constraint",
			"summary_constraint",
			"tags_constraint",
			"data_constraint",
		}).AddRow(true, true, true, true, true, true, true))

	readiness := loadMemoryReadiness(context.Background(), db)
	if readiness.Ready {
		t.Fatalf("expected memory readiness false, got %+v", readiness)
	}
	if readiness.Status != "not_ready" {
		t.Fatalf("expected not_ready status, got %q", readiness.Status)
	}
	if readiness.Error == "" {
		t.Fatal("expected readiness error for stale schema version")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestLoadMemoryReadiness_NotReadyWhenSchemaObjectsAreIncomplete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT version, dirty FROM schema_migrations ORDER BY version DESC LIMIT 1`).
		WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(pgmemory.RequiredSchemaVersion, false))
	mock.ExpectQuery(`SELECT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"memory_items",
			"unique_index",
			"hnsw_index",
			"bank_constraint",
			"summary_constraint",
			"tags_constraint",
			"data_constraint",
		}).AddRow(true, false, true, true, true, true, true))

	readiness := loadMemoryReadiness(context.Background(), db)
	if readiness.Ready {
		t.Fatalf("expected memory readiness false, got %+v", readiness)
	}
	if readiness.SchemaOK {
		t.Fatalf("expected schema check to fail, got %+v", readiness)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
