package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func withRepoRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestRecoveryManifestFullConformance(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}
	if m.Candidate.ActionableConfidenceThreshold != 0.60 {
		t.Fatalf("threshold = %v", m.Candidate.ActionableConfidenceThreshold)
	}
	if m.Provider.RequestDateGuard == "" || !m.Provider.RawBytesBeforeNormalization {
		t.Fatal("provider material fields not bound")
	}
	if m.NoPostResultSalvage.DateExtension || m.NoPostResultSalvage.ThresholdTuning || !m.NoPostResultSalvage.NewExperimentRequired {
		t.Fatal("post-result salvage contract weakened")
	}
	if m.PromotionGates.UnknownMaterialStatus != "FAIL_CLOSED" || m.PromotionGates.SecondarySignPermutation != "NON_BLOCKING_INFORMATIONAL" {
		t.Fatal("promotion gate semantics changed")
	}
}

func TestOutcomeFreeContractAuditExecutes(t *testing.T) {
	withRepoRoot(t)
	if err := runContractAudit(); err != nil {
		t.Fatal(err)
	}
}

func TestRecoveryDateGuardIsExact(t *testing.T) {
	for _, date := range []string{"2025-01-01", "2026-01-02", "2026-09-11"} {
		if !recoveryDateAllowed(date) {
			t.Fatalf("expected allowed: %s", date)
		}
	}
	for _, date := range []string{"2024-12-31", "2026-09-12", "2026-09-14", "not-a-date"} {
		if recoveryDateAllowed(date) {
			t.Fatalf("expected rejected: %s", date)
		}
	}
}

func TestContaminatedRunGuardRemainsPermanent(t *testing.T) {
	withRepoRoot(t)
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := validateContaminatedGuard(m); err != nil {
		t.Fatal(err)
	}
}

func TestReadinessOutputContainsNoPerformanceFields(t *testing.T) {
	b, err := json.Marshal(readinessResult{})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"return", "pnl", "sharpe", "sortino", "drawdown", "win_rate", "placebo_effect"} {
		if containsFold(string(b), forbidden) {
			t.Fatalf("readiness schema exposes forbidden performance field %q: %s", forbidden, string(b))
		}
	}
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := range sub {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
