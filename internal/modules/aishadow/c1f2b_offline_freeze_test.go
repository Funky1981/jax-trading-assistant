package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateC1F2BOfflineFreezeProvesDefaultDenyPreservationAndBudgets(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path, hash, err := GenerateC1F2BOfflineFreeze(root, t.TempDir(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || !strings.Contains(filepath.ToSlash(path), C1F2BOfflineEvidenceNamespace+"/"+C1F2BWorkPackageIdentity) {
		t.Fatalf("unexpected C1F2B evidence path/hash: %s %s", path, hash)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence C1F2BOfflineFreeze
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Identity != C1F2BWorkPackageIdentity || evidence.AuthorizationMechanism != C1F3ExecutionAuthorizationVersion ||
		evidence.AuthorizationFingerprint != C1F3ExecutionAuthorizationFingerprint() || evidence.AuthorizationSourceSHA256 == "" ||
		evidence.DefaultExecutionAuthorization || evidence.HostedInferenceAuthorized || evidence.CredentialLoaded || evidence.ProviderConstructed ||
		evidence.ProviderContact || evidence.Inference || evidence.DatabaseMutation || evidence.TradingMutation ||
		!evidence.ZeroNetworkPlanningPossible || !evidence.ProviderConstructionRequiresBoth || len(evidence.Profiles) != 2 {
		t.Fatalf("C1F2B offline evidence is incomplete or unsafe: %+v", evidence)
	}
	for _, profile := range evidence.Profiles {
		if profile.ProviderContact || profile.Inference || profile.ExecutionAuthorized || !profile.EvidenceNamespaceCollisionFree ||
			!profile.ProviderInputIsolated || !profile.DefaultExecutionDenied || !strings.Contains(profile.DefaultDenialReason, "--authorize-c1f3-execution") ||
			profile.Repetitions != 1 || profile.RequestedModel != OpenAIDiagnosticLunaModel {
			t.Fatalf("C1F2B profile evidence is incomplete: %+v", profile)
		}
		estimate, estimateErr := parseUSDMicros(profile.EstimatedMaximumRunUSD)
		ceiling, ceilingErr := parseUSDMicros(profile.BudgetCeilingUSD)
		if estimateErr != nil || ceilingErr != nil || estimate > ceiling {
			t.Fatalf("C1F2B conservative estimate exceeds ceiling: %+v", profile)
		}
	}
	for _, frozen := range evidence.FrozenFiles {
		if !frozen.Preserved || frozen.ExpectedSHA256 == "" || frozen.ActualSHA256 != frozen.ExpectedSHA256 {
			t.Fatalf("frozen C1F2B input was not preserved: %+v", frozen)
		}
	}
}

func TestGenerateC1F2BOfflineFreezeRejectsAnyAuthorization(t *testing.T) {
	for _, state := range []struct{ hosted, optIn bool }{{hosted: true}, {optIn: true}, {hosted: true, optIn: true}} {
		if _, _, err := GenerateC1F2BOfflineFreeze(t.TempDir(), t.TempDir(), state.hosted, state.optIn); err == nil {
			t.Fatalf("C1F2B evidence accepted authorization state: %+v", state)
		}
	}
}

func TestWriteC1F2BOfflineFreezeWhenExplicitlyRequested(t *testing.T) {
	outputRoot := os.Getenv("JAX_C1F2B_OFFLINE_EVIDENCE_ROOT")
	if strings.TrimSpace(outputRoot) == "" {
		t.Skip("set JAX_C1F2B_OFFLINE_EVIDENCE_ROOT to write scoped offline evidence")
	}
	root := filepath.Join("..", "..", "..")
	path, hash, err := GenerateC1F2BOfflineFreeze(root, outputRoot, false, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("C1F2B offline evidence: %s sha256=%s", path, hash)
}
