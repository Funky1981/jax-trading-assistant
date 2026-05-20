//go:build integration
// +build integration

package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	jaxdb "jax-trading-assistant/libs/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestIntegrationEventStudySchemaMigration(t *testing.T) {
	baseDSN := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	schemaName := fmt.Sprintf("event_study_schema_test_%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB, err := sql.Open("pgx", baseDSN)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, `CREATE SCHEMA `+schemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.ExecContext(cleanupCtx, `DROP SCHEMA IF EXISTS `+schemaName+` CASCADE`)
	})

	scopedDSN, err := dsnWithSearchPath(baseDSN, schemaName)
	if err != nil {
		t.Skipf("TEST_DATABASE_URL must be a postgres URL for schema-scoped migration test: %v", err)
	}

	db, err := sql.Open("pgx", scopedDSN)
	if err != nil {
		t.Fatalf("open scoped db: %v", err)
	}
	defer db.Close()

	if err := jaxdb.RunMigrations(db, "file://"+migrationsDir(t)); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	for _, table := range []string{
		"event_windows",
		"event_confounders",
		"event_priced_in_scores",
		"etf_context_snapshots",
		"research_summaries",
	} {
		requireRegclass(t, ctx, db, table)
	}

	for _, constraint := range []string{
		"uq_event_windows_event_symbol_window",
		"uq_event_confounders_event_confounding_symbol",
		"uq_event_priced_in_scores_event_symbol",
		"chk_event_confounders_relationship_type",
		"chk_event_priced_in_scores_verdict",
	} {
		requireConstraint(t, ctx, db, constraint)
	}

	for _, index := range []string{
		"idx_event_windows_symbol_window",
		"idx_event_windows_event_symbol",
		"idx_event_confounders_event_symbol",
		"idx_event_priced_in_scores_event_symbol",
		"idx_etf_context_snapshots_symbol_validity",
		"idx_research_summaries_event_symbol",
	} {
		requireIndex(t, ctx, db, index)
	}
}

func dsnWithSearchPath(rawDSN, schemaName string) (string, error) {
	parsed, err := url.Parse(rawDSN)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	query := parsed.Query()
	query.Set("search_path", schemaName)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func requireRegclass(t *testing.T, ctx context.Context, db *sql.DB, name string) {
	t.Helper()
	var objectName sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT to_regclass($1)`, name).Scan(&objectName); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	if !objectName.Valid {
		t.Fatalf("table %s not found", name)
	}
}

func requireConstraint(t *testing.T, ctx context.Context, db *sql.DB, name string) {
	t.Helper()
	var found bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_constraint
			WHERE conname = $1
		 )`, name).Scan(&found); err != nil {
		t.Fatalf("check constraint %s: %v", name, err)
	}
	if !found {
		t.Fatalf("constraint %s not found", name)
	}
}

func requireIndex(t *testing.T, ctx context.Context, db *sql.DB, name string) {
	t.Helper()
	var found bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_indexes
			WHERE indexname = $1
		)`, name).Scan(&found); err != nil {
		t.Fatalf("check index %s: %v", name, err)
	}
	if !found {
		t.Fatalf("index %s not found", name)
	}
}
