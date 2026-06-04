package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLLMCostUsageSchema(t *testing.T) {
	path := filepath.Join("000026_llm_cost_context.up.sql")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sql := string(data)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS llm_usage_logs",
		"task_type TEXT NOT NULL",
		"model_alias TEXT NOT NULL",
		"provider_model TEXT NOT NULL",
		"estimated_cost_usd NUMERIC NOT NULL DEFAULT 0",
		"actual_cost_usd NUMERIC NOT NULL DEFAULT 0",
		"blocked BOOLEAN NOT NULL DEFAULT FALSE",
		"block_reason TEXT",
		"CREATE TABLE IF NOT EXISTS llm_cost_rollups",
		"rollup_type TEXT NOT NULL",
		"paid_calls_avoided INT NOT NULL DEFAULT 0",
		"headroom_tokens_saved INT NOT NULL DEFAULT 0",
		"idx_llm_usage_logs_task_created",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("llm cost schema missing %q", required)
		}
	}
}
