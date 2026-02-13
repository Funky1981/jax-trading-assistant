//go:build golden
// +build golden

package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGolden_Signals(t *testing.T) {
	baselineFile := filepath.Join("signals", "baseline-2026-02-13.json")

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Skipf("Baseline file not found: %s. Run 'go run ./tests/golden/cmd/capture.go' first", baselineFile)
	}

	expected, err := LoadSnapshot(baselineFile)
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	// Capture current behavior
	// NOTE: This requires services to be running
	// For CI, we'll use pre-captured "current" snapshots
	currentFile := filepath.Join("signals", "current.json")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		t.Skipf("Current snapshot not found. Run capture first.")
	}

	actual, err := LoadSnapshot(currentFile)
	if err != nil {
		t.Fatalf("Failed to load current snapshot: %v", err)
	}

	// Compare
	result := CompareSnapshots(expected, actual)
	if !result.Match {
		t.Errorf("Golden test failed for signals. Differences:\n")
		for _, diff := range result.Differences {
			t.Errorf("  - %s\n", diff)
		}
		t.Errorf("\nIf this change is intentional, update the baseline:\n")
		t.Errorf("  go run ./tests/golden/cmd/capture.go\n")
	}
}

func TestGolden_Executions(t *testing.T) {
	baselineFile := filepath.Join("executions", "baseline-2026-02-13.json")

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Skipf("Baseline file not found: %s", baselineFile)
	}

	expected, err := LoadSnapshot(baselineFile)
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	currentFile := filepath.Join("executions", "current.json")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		t.Skipf("Current snapshot not found")
	}

	actual, err := LoadSnapshot(currentFile)
	if err != nil {
		t.Fatalf("Failed to load current snapshot: %v", err)
	}

	result := CompareSnapshots(expected, actual)
	if !result.Match {
		t.Errorf("Golden test failed for executions. Differences:\n")
		for _, diff := range result.Differences {
			t.Errorf("  - %s\n", diff)
		}
		t.Errorf("\nIf this change is intentional, update the baseline:\n")
		t.Errorf("  go run ./tests/golden/cmd/capture.go\n")
	}
}

func TestGolden_Orchestration(t *testing.T) {
	baselineFile := filepath.Join("orchestration", "baseline-2026-02-13.json")

	if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
		t.Skipf("Baseline file not found: %s", baselineFile)
	}

	expected, err := LoadSnapshot(baselineFile)
	if err != nil {
		t.Fatalf("Failed to load baseline: %v", err)
	}

	currentFile := filepath.Join("orchestration", "current.json")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		t.Skipf("Current snapshot not found")
	}

	actual, err := LoadSnapshot(currentFile)
	if err != nil {
		t.Fatalf("Failed to load current snapshot: %v", err)
	}

	result := CompareSnapshots(expected, actual)
	if !result.Match {
		t.Errorf("Golden test failed for orchestration. Differences:\n")
		for _, diff := range result.Differences {
			t.Errorf("  - %s\n", diff)
		}
		t.Errorf("\nIf this change is intentional, update the baseline:\n")
		t.Errorf("  go run ./tests/golden/cmd/capture.go\n")
	}
}
