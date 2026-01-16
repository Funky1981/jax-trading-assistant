package reflection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
)

func TestGenerateBeliefs(t *testing.T) {
	now := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	window := Window{
		From: now.AddDate(0, 0, -7),
		To:   now,
	}

	decisions := []contracts.MemoryItem{
		{
			ID:      "dec_1",
			TS:      now.Add(-72 * time.Hour),
			Type:    "decision",
			Tags:    []string{"earnings", "breakout"},
			Summary: "Decision 1",
			Data:    map[string]any{"ok": true},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}

	outcomes := []contracts.MemoryItem{
		{
			ID:      "out_1",
			TS:      now.Add(-48 * time.Hour),
			Type:    "outcome",
			Tags:    []string{"earnings", "win"},
			Summary: "Closed for profit",
			Data:    map[string]any{"result": "win"},
			Source:  &contracts.MemorySource{System: "test"},
		},
		{
			ID:      "out_2",
			TS:      now.Add(-24 * time.Hour),
			Type:    "outcome",
			Tags:    []string{"earnings"},
			Summary: "Stopped out",
			Data:    map[string]any{"pnl": -0.8},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}

	beliefs := GenerateBeliefs(now, window, decisions, outcomes)
	if len(beliefs) == 0 {
		t.Fatalf("expected beliefs")
	}

	first := beliefs[0]
	if first.Type != "belief" {
		t.Fatalf("expected belief type, got %q", first.Type)
	}
	if first.Source == nil || first.Source.System != SourceSystem {
		t.Fatalf("expected source system %q", SourceSystem)
	}
	if first.Data == nil {
		t.Fatalf("expected data payload")
	}
	if _, ok := first.Data["confidence"]; !ok {
		t.Fatalf("expected confidence in data")
	}
	if _, ok := first.Data["time_window"]; !ok {
		t.Fatalf("expected time_window in data")
	}
}

func TestGenerateBeliefs_Golden(t *testing.T) {
	now := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)
	window := Window{
		From: now.AddDate(0, 0, -7),
		To:   now,
	}

	decisions := []contracts.MemoryItem{
		{
			ID:      "dec_1",
			TS:      now.Add(-120 * time.Hour),
			Type:    "decision",
			Tags:    []string{"earnings", "breakout"},
			Summary: "Decision 1",
			Data:    map[string]any{"ok": true},
			Source:  &contracts.MemorySource{System: "test"},
		},
		{
			ID:      "dec_2",
			TS:      now.Add(-96 * time.Hour),
			Type:    "decision",
			Tags:    []string{"earnings"},
			Summary: "Decision 2",
			Data:    map[string]any{"ok": true},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}

	outcomes := []contracts.MemoryItem{
		{
			ID:      "out_1",
			TS:      now.Add(-72 * time.Hour),
			Type:    "outcome",
			Tags:    []string{"earnings", "win"},
			Summary: "Closed for profit",
			Data:    map[string]any{"pnl": 1.2},
			Source:  &contracts.MemorySource{System: "test"},
		},
		{
			ID:      "out_2",
			TS:      now.Add(-48 * time.Hour),
			Type:    "outcome",
			Tags:    []string{"loss"},
			Summary: "Stopped out",
			Data:    map[string]any{"pnl": -0.5},
			Source:  &contracts.MemorySource{System: "test"},
		},
		{
			ID:      "out_3",
			TS:      now.Add(-24 * time.Hour),
			Type:    "outcome",
			Tags:    []string{"earnings"},
			Summary: "Held gain",
			Data:    map[string]any{"result": "win"},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}

	beliefs := GenerateBeliefs(now, window, decisions, outcomes)
	got, err := json.MarshalIndent(beliefs, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	goldenPath := filepath.Join("testdata", "beliefs_golden.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if normalizeGolden(string(got)) != normalizeGolden(string(expected)) {
		t.Fatalf("golden mismatch\n--- expected\n%s\n--- got\n%s", string(expected), string(got))
	}
}

func normalizeGolden(raw string) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.TrimSpace(normalized)
}
