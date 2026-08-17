package aishadow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateC1E2BOfflineFreezeProvesDefaultDenyAndPreservation(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	path, hash, err := GenerateC1E2BOfflineFreeze(root, t.TempDir(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || !strings.Contains(filepath.ToSlash(path), C1E2BOfflineEvidenceNamespace+"/"+C1E2BWorkPackageIdentity) {
		t.Fatalf("unexpected C1E2B evidence path/hash: %s %s", path, hash)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var evidence C1E2BOfflineFreeze
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Identity != C1E2BWorkPackageIdentity || evidence.AuthorizationMechanism != C1E3ExecutionAuthorizationVersion ||
		evidence.AuthorizationSourceSHA256 == "" || evidence.DefaultExecutionAuthorization || evidence.HostedInferenceAuthorized ||
		evidence.CredentialLoaded || evidence.ProviderConstructed || evidence.ProviderContact || evidence.Inference ||
		evidence.DatabaseMutation || evidence.TradingMutation || !evidence.ZeroNetworkPlanningPossible ||
		!evidence.ProviderConstructionRequiresBoth || len(evidence.Profiles) != 2 {
		t.Fatalf("C1E2B offline evidence is incomplete or unsafe: %+v", evidence)
	}
	for _, profile := range evidence.Profiles {
		if profile.ProviderContact || profile.Inference || profile.ExecutionAuthorized || !profile.EvidenceNamespaceCollisionFree ||
			!profile.DefaultExecutionDenied || !strings.Contains(profile.DefaultDenialReason, "--authorize-c1e3-execution") ||
			profile.PromptVersion != V5PromptVersion || profile.OutputContract != V5SchemaVersion || profile.SchemaSHA256 != frozenV5SchemaSHA256 {
			t.Fatalf("C1E2B profile evidence is incomplete: %+v", profile)
		}
	}
	for _, frozen := range evidence.FrozenFiles {
		if !frozen.Preserved || frozen.ExpectedSHA256 == "" || frozen.ActualSHA256 != frozen.ExpectedSHA256 {
			t.Fatalf("frozen input was not preserved: %+v", frozen)
		}
	}
}

func TestGenerateC1E2BOfflineFreezeRejectsAnyAuthorization(t *testing.T) {
	for _, test := range []struct {
		hosted bool
		optIn  bool
	}{{hosted: true}, {optIn: true}, {hosted: true, optIn: true}} {
		if _, _, err := GenerateC1E2BOfflineFreeze(t.TempDir(), t.TempDir(), test.hosted, test.optIn); err == nil {
			t.Fatalf("C1E2B evidence accepted authorization state: %+v", test)
		}
	}
}
