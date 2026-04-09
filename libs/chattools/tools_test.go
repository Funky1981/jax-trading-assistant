package chattools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultTools_HaveExpectedShape(t *testing.T) {
	tools := DefaultTools()
	if len(tools) != 18 {
		t.Fatalf("expected 18 tools, got %d", len(tools))
	}

	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		if tool.Name == "" {
			t.Fatal("tool name must not be empty")
		}
		if tool.Description == "" {
			t.Fatalf("tool %q missing description", tool.Name)
		}
		if !tool.ReadOnly {
			t.Fatalf("tool %q must be read-only", tool.Name)
		}
		if tool.Handler == nil {
			t.Fatalf("tool %q missing handler", tool.Name)
		}
		if _, ok := seen[tool.Name]; ok {
			t.Fatalf("duplicate tool %q", tool.Name)
		}
		seen[tool.Name] = struct{}{}
	}
}

func TestLookup_FindsKnownTool(t *testing.T) {
	tool, ok := Lookup("query_knowledge")
	if !ok {
		t.Fatal("expected query_knowledge to be registered")
	}
	if tool.ArgKey != "query" {
		t.Fatalf("unexpected arg key: %s", tool.ArgKey)
	}
}

func TestLookup_FindsNewResearchTool(t *testing.T) {
	tool, ok := Lookup("compare_runs")
	if !ok {
		t.Fatal("expected compare_runs to be registered")
	}
	if tool.ArgKey != "runIds" {
		t.Fatalf("unexpected arg key: %s", tool.ArgKey)
	}
}

func TestGetStrategy_ReturnsPlaceholderPayload(t *testing.T) {
	raw, err := GetStrategy(context.Background(), nil, json.RawMessage(`{"strategyId":"abc"}`))
	if err != nil {
		t.Fatalf("GetStrategy returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got["strategyId"] != "abc" {
		t.Fatalf("expected strategyId to round-trip, got %#v", got["strategyId"])
	}
}

func TestQueryKnowledge_RequiresQuery(t *testing.T) {
	if _, err := QueryKnowledge(context.Background(), nil, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected missing query to fail")
	}
}

func TestQueryKnowledge_ReadsConfiguredRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("Readiness checklist includes paper readiness evidence."), 0o600); err != nil {
		t.Fatalf("write note: %v", err)
	}

	t.Setenv("JAX_KNOWLEDGE_ROOT", dir)
	raw, err := QueryKnowledge(context.Background(), nil, json.RawMessage(`{"query":"paper readiness"}`))
	if err != nil {
		t.Fatalf("QueryKnowledge returned error: %v", err)
	}

	var matches []map[string]any
	if err := json.Unmarshal(raw, &matches); err != nil {
		t.Fatalf("unmarshal matches: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0]["path"] != path {
		t.Fatalf("unexpected match path: %#v", matches[0]["path"])
	}
}

func TestCompareRuns_RequiresMultipleRunIDs(t *testing.T) {
	if _, err := CompareRuns(context.Background(), nil, json.RawMessage(`{"runIds":"one"}`)); err == nil {
		t.Fatal("expected CompareRuns to require at least two run IDs")
	}
}

func TestStrategyInstanceSummary_RequiresInstanceID(t *testing.T) {
	if _, err := StrategyInstanceSummary(context.Background(), nil, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected StrategyInstanceSummary to require instanceId")
	}
}
