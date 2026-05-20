package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMobileApprovalFlowSchema(t *testing.T) {
	path := filepath.Join("000024_mobile_approval_flow.up.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sql := strings.Join(strings.Fields(string(data)), " ")
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS notification_outbox",
		"CREATE TABLE IF NOT EXISTS mobile_approval_tokens",
		"candidate_id UUID NOT NULL REFERENCES candidate_trades(id)",
		"token_hash TEXT NOT NULL UNIQUE",
		"guardrail_hash TEXT NOT NULL",
		"used_at TIMESTAMPTZ",
		"idx_notification_outbox_pending",
		"idx_mobile_approval_tokens_token_hash",
		"idx_mobile_approval_tokens_expires",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("mobile approval schema missing %q", required)
		}
	}
}
