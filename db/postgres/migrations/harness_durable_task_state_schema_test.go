package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestHarnessDurableTaskStateMigrationIsAdditiveAndPaper02Independent(t *testing.T) {
	data, err := os.ReadFile("000075_harness_durable_task_state.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(data))
	for _, fragment := range []string{
		"create table if not exists harness_tasks",
		"create table if not exists harness_task_state_versions",
		"create table if not exists harness_checkpoints",
		"create table if not exists harness_compactions",
		"unique (task_id, idempotency_key)",
		"primary key (task_id, state_version)",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("migration missing %q", fragment)
		}
	}
	if strings.Contains(sql, "paper02") || strings.Contains(sql, "exploratory_paper") || strings.Contains(sql, "alter table") {
		t.Fatal("HARNESS-03 migration must not modify PAPER-02 or existing tables")
	}
}
