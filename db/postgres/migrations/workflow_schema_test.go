package migrations

import (
	"os"
	"strings"
	"testing"
)

func TestWorkflowPersistenceMigrationsDefineStateAndAppendOnlyAuditTables(t *testing.T) {
	instances, err := os.ReadFile("000061_workflow_instances.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	audit, err := os.ReadFile("000062_workflow_audit_events.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	instancesSQL, auditSQL := strings.ToLower(string(instances)), strings.ToLower(string(audit))
	for _, fragment := range []string{"create table if not exists workflow_instances", "workflow_id text primary key", "revision bigint", "payload jsonb not null"} {
		if !strings.Contains(instancesSQL, fragment) {
			t.Fatalf("workflow instance migration missing %q", fragment)
		}
	}
	for _, fragment := range []string{"create table if not exists workflow_audit_events", "workflow_id text not null references workflow_instances", "idempotency_key text not null unique", "uq_workflow_audit_sequence"} {
		if !strings.Contains(auditSQL, fragment) {
			t.Fatalf("workflow audit migration missing %q", fragment)
		}
	}
}
