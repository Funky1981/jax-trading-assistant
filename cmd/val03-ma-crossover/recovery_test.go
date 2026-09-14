package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func loadRecoveryTestManifest(t *testing.T) recoveryManifest {
	t.Helper()
	b, err := os.ReadFile(repoFile(recoveryManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	var m recoveryManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRecoveryManifestIsDistinctAndParentHashBound(t *testing.T) {
	m := loadRecoveryTestManifest(t)
	if err := validateRecoveryManifest(m); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(repoFile(manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	if got := sha256Hex(b); got != parentManifestSHA {
		t.Fatalf("parent manifest changed: %s", got)
	}
	if m.ContractID == "jax.val-02.candidate-preregistration" || m.Parent.ManifestSHA256 != parentManifestSHA {
		t.Fatal("recovery identity is not distinct and parent-bound")
	}
}

func TestRecoveryBoundaryAndPostBoundaryRejection(t *testing.T) {
	for _, date := range []string{"2025-01-01", "2026-09-11"} {
		if !recoveryDateAllowed(date) {
			t.Fatalf("expected recovery date allowed: %s", date)
		}
	}
	for _, date := range []string{"2024-12-31", "2026-09-12", "2026-09-14", "2026-09-15"} {
		if recoveryDateAllowed(date) {
			t.Fatalf("expected recovery date rejected: %s", date)
		}
	}
	m := loadRecoveryTestManifest(t)
	if m.Boundary.End == "2026-09-14" || m.Boundary.ExtensionAllowed {
		t.Fatal("recovery boundary permits incomplete/post-result extension")
	}
}

func TestRecoverySampleFloorsAreTypedFromParentContract(t *testing.T) {
	b, err := os.ReadFile(repoFile(manifestPath))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	cfg, err := loadFrozenExperimentConfig(b, hex.EncodeToString(h[:]))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DevelopmentSampleFloor != 90 || cfg.ValidationSampleFloor != 30 || cfg.OOSSampleFloor != 30 {
		t.Fatalf("sample floors = development:%d validation:%d oos:%d", cfg.DevelopmentSampleFloor, cfg.ValidationSampleFloor, cfg.OOSSampleFloor)
	}
	m := loadRecoveryTestManifest(t)
	if m.SampleFloors.Development != cfg.DevelopmentSampleFloor || m.SampleFloors.Validation != cfg.ValidationSampleFloor || m.SampleFloors.RecoveryOOS != cfg.OOSSampleFloor {
		t.Fatal("recovery sample floors do not inherit parent values")
	}
}

func TestRecoveryRetainsSafetyAndProviderContract(t *testing.T) {
	m := loadRecoveryTestManifest(t)
	if m.Provider.Name != "Alpaca" || m.Provider.Feed != "SIP" || m.Provider.Timeframe != "1Day" || m.Provider.Fallback != "NONE" || m.Provider.PaidSpendUSD != 0 {
		t.Fatalf("unexpected provider contract: %+v", m.Provider)
	}
	if m.Safety.ExecutionAuthority != "NONE" || m.Safety.CreatesFill || m.Candidate.PrimaryClaim != "BULLISH_LONG_ONLY" {
		t.Fatal("recovery safety or primary claim changed")
	}
}
