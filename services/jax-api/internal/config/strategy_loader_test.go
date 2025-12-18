package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStrategyConfigs_LoadsJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "s1.json"), []byte(`{
  "id": "s1",
  "name": "S1",
  "eventTypes": ["gap_open"],
  "minRR": 2.0,
  "maxRiskPercent": 3.0,
  "entryRule": "x",
  "stopRule": "y",
  "targetRule": "2R"
}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := LoadStrategyConfigs(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got["s1"].Name != "S1" {
		t.Fatalf("unexpected: %#v", got["s1"])
	}
}
