// Package testsupport contains opt-in, disposable-database test guardrails.
package testsupport

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDSN never falls back to the developer's Jax database. CI must supply
// a disposable database explicitly; missing required integration configuration
// is a failure, not a successful skipped test.
func PostgresDSN(t testing.TB, variable string) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv(variable))
	if dsn == "" {
		if os.Getenv("PAPER02R_REQUIRED_INTEGRATION") == "true" {
			t.Fatalf("required integration variable %s is missing", variable)
		}
		t.Skipf("%s is not configured for a disposable test database", variable)
	}
	if err := ValidatePostgresDSN(dsn); err != nil {
		t.Fatal(err)
	}
	return dsn
}

// ValidatePostgresDSN deliberately reports no connection string or secret.
func ValidatePostgresDSN(dsn string) error {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("invalid disposable PostgreSQL test configuration")
	}
	if !strings.HasPrefix(config.ConnConfig.Database, "jax_paper02r_test") {
		return fmt.Errorf("integration tests require a database named jax_paper02r_test or jax_paper02r_test_<suffix>; production databases are forbidden")
	}
	return nil
}
