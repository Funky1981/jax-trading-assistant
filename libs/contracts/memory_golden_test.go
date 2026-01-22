package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMemoryItemJSON_Golden(t *testing.T) {
	item := MemoryItem{
		TS:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Type:    "decision",
		Symbol:  "AAPL",
		Tags:    []string{"earnings"},
		Summary: "Entered on earnings gap.",
		Data:    map[string]any{"confidence": 0.72},
		Source:  &MemorySource{System: "dexter"},
	}

	raw, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	goldenPath := filepath.Join("testdata", "memory_item_golden.json")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if normalizeJSON(string(raw)) != normalizeJSON(string(expected)) {
		t.Fatalf("golden mismatch\nexpected:\n%s\nactual:\n%s", string(expected), string(raw))
	}
}

func normalizeJSON(raw string) string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.TrimSpace(normalized)
}
