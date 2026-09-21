package testsupport

import "testing"

func TestProductionDatabaseRejected(t *testing.T) {
	for _, dsn := range []string{"postgres://jax:secret@localhost:5433/jax", "postgres://jax@localhost/postgres", "postgres://jax@localhost/jax_paper02r_testfoo", "malformed password=secret"} {
		if ValidatePostgresDSN(dsn) == nil {
			t.Fatal("unsafe test database accepted")
		}
	}
	if err := ValidatePostgresDSN("postgres://test@localhost/jax_paper02r_test_ci?sslmode=disable"); err != nil {
		t.Fatal(err)
	}
}
