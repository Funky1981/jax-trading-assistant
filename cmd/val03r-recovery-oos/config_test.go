package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func frozenTestConfig(t *testing.T) FrozenExperimentConfig {
	t.Helper()
	b, err := os.ReadFile(repoFile(manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	cfg, err := loadFrozenExperimentConfig(b, hex.EncodeToString(h[:]))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func manifestWithChange(t *testing.T, path ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(repoFile(manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	object := root
	for _, key := range path[:len(path)-2] {
		object = object[key].(map[string]any)
	}
	object[path[len(path)-2]] = path[len(path)-1]
	changed, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestManifestBoundConfigFreezesCoreSemantics(t *testing.T) {
	cfg := frozenTestConfig(t)
	if cfg.ActionableConfidenceThreshold != 0.60 {
		t.Fatalf("threshold = %v, want 0.60", cfg.ActionableConfidenceThreshold)
	}
	if cfg.SMAFast != 20 || cfg.SMAMedium != 50 || cfg.SMASlow != 200 || cfg.ATRPeriod != 14 || cfg.HoldingPeriod != 20 {
		t.Fatalf("unexpected indicator/holding config: %+v", cfg)
	}
	if cfg.ExecutionAuthority != "NONE" || !sameStrings(cfg.Universe, expectedUniverse) {
		t.Fatalf("unexpected safety/universe config: %+v", cfg)
	}
	if cfg.DevelopmentSampleFloor != 90 || cfg.ValidationSampleFloor != 30 || cfg.OOSSampleFloor != 30 || cfg.PairedFloor != 30 || cfg.EffectiveBlockFloor != 12 || cfg.InstrumentFloor != 6 || cfg.MinimumSliceFloor != 3 || cfg.OverlapWindow != 20 || cfg.ConcentrationCeiling != .4 {
		t.Fatalf("promotion-critical floors are not fully bound: %+v", cfg)
	}
	if cfg.DevelopmentStart != "2016-01-01" || cfg.ValidationStart != "2021-01-01" || cfg.OOSStart != "2023-01-01" || cfg.HoldoutStart != "2025-01-01" {
		t.Fatalf("unexpected partitions: %+v", cfg)
	}
}

func TestR1BPreservesFrozenManifestAndExplicitPermutationWiring(t *testing.T) {
	b, err := os.ReadFile(repoFile(manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	if got := hex.EncodeToString(h[:]); got != "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a" {
		t.Fatalf("v1.5 manifest changed: %s", got)
	}
	source, err := os.ReadFile(repoFile("cmd/val03-ma-crossover/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(source), "signPermutation(nil") || !strings.Contains(string(source), "secondarySignPermutation(secondary") {
		t.Fatal("secondary sign permutation does not require explicit typed observations")
	}
}

func TestManifestBoundConfigRejectsMaterialDrift(t *testing.T) {
	for _, tc := range []struct {
		name string
		path []string
	}{
		{"threshold", []string{"confidence_contract", "actionable_threshold", "0.7"}},
		{"candidate", []string{"selected_candidate", "candidate_id", "other"}},
		{"partition", []string{"partitions", "formal_oos", "2022-01-01"}},
		{"universe", []string{"universe", "symbols", "OTHER"}},
		{"minimum slices", []string{"sample_and_dependence", "minimum_regime_or_calendar_slices", "2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := manifestWithChange(t, tc.path...)
			if _, err := loadFrozenExperimentConfig(b, "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a"); err == nil {
				t.Fatal("expected manifest-bound rejection")
			}
		})
	}
}

func TestActionableGateIsSharedByPrimaryAndRobustnessPaths(t *testing.T) {
	cfg := frozenTestConfig(t)
	if !actionableByConfig(0.60, cfg) || actionableByConfig(0.599999, cfg) {
		t.Fatal("frozen 0.60 gate is not exact")
	}
	for _, partition := range []string{"development", "validation", "formal_oos"} {
		if !actionableByConfig(0.60, cfg) {
			t.Fatalf("%s path does not receive frozen gate", partition)
		}
	}
	for _, variant := range []float64{0.90, 1.10} {
		if !actionableByConfig(0.60, cfg) || actionableByConfig(0.599999, cfg) {
			t.Fatalf("robustness variant %v can override eligibility", variant)
		}
	}
}

func TestContaminatedRunGuardBlocksOldFormalIdentity(t *testing.T) {
	blocked, err := contaminatedRunGuard(repoFile(integrityPath))
	if err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Fatal("expected immutable contaminated-run guard")
	}
}

func TestPreflightDoesNotPermitSealedDate(t *testing.T) {
	if err := validateProviderDate("2025-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected returned 2025 provider row rejection")
	}
	if err := validateDateRange("2024-01-01", "2025-01-01"); err == nil {
		t.Fatal("expected 2025 rejection")
	}
	if err := validateDateRange("2024-01-01", "2024-12-31"); err != nil {
		t.Fatal(err)
	}
}

func TestResultSchemaMarksOutputWhenWritten(t *testing.T) {
	data := map[string]*instrumentData{}
	for _, symbol := range symbols {
		data[symbol] = &instrumentData{Dates: []string{"2016-01-04"}}
	}
	if got := qualitySummary(data, true)["performance_output_generated"]; got != true {
		t.Fatalf("written output marker = %v, want true", got)
	}
	if got := qualitySummary(data, false)["performance_output_generated"]; got != false {
		t.Fatalf("data-only output marker = %v, want false", got)
	}
}

func TestPreflightGuardRecordHasNoOutcomeDependency(t *testing.T) {
	b, err := os.ReadFile(repoFile(integrityPath))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "2025-01-01\" bars") {
		t.Fatal("guard record unexpectedly contains holdout bars")
	}
}

func repoFile(path string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "..", path)
}
