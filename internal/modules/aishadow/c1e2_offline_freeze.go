package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1E2OfflineEvidenceNamespace = "offline-c1e2-causal-attribution-v1"
	C1E2WorkPackageIdentity      = "WP-00.03C1E2"
)

type C1E2FrozenFile struct {
	Identity       string `json:"identity"`
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ActualSHA256   string `json:"actual_sha256"`
	Preserved      bool   `json:"preserved"`
}

type C1E2FrozenProfile struct {
	DatasetIdentity   string `json:"dataset_identity"`
	ExperimentID      string `json:"experiment_id"`
	EvidenceNamespace string `json:"evidence_namespace"`
	CaseCount         int    `json:"case_count"`
	PromptVersion     string `json:"prompt_version"`
	OutputContract    string `json:"output_contract"`
	CausalPolicy      string `json:"causal_policy"`
	ResolverPolicy    string `json:"resolver_policy"`
	Repetitions       int    `json:"initial_repetitions"`
	MaximumBudgetUSD  string `json:"maximum_budget_usd"`
	Executed          bool   `json:"executed"`
}

type C1E2OfflineFreeze struct {
	Identity                      string              `json:"identity"`
	EvidenceNamespace             string              `json:"evidence_namespace"`
	GeneratedAt                   time.Time           `json:"generated_at"`
	ProviderContact               bool                `json:"provider_contact"`
	Inference                     bool                `json:"inference"`
	DatabaseMutation              bool                `json:"database_mutation"`
	TradingMutation               bool                `json:"trading_mutation"`
	HostedInferenceAuthorized     bool                `json:"hosted_inference_authorized"`
	PromptVersion                 string              `json:"prompt_version"`
	PromptSHA256                  string              `json:"prompt_sha256"`
	OutputContract                string              `json:"output_contract"`
	OpenAISchemaName              string              `json:"openai_schema_name"`
	SchemaSHA256                  string              `json:"schema_sha256"`
	CausalAttributionPolicy       string              `json:"causal_attribution_policy"`
	CausalAttributionSourceSHA256 string              `json:"causal_attribution_source_sha256"`
	ResolverPolicy                string              `json:"resolver_policy"`
	ResolverSHA256                string              `json:"resolver_sha256"`
	C1DPolicy                     string              `json:"c1d_policy"`
	C1DSourceSHA256               string              `json:"c1d_source_sha256"`
	C1DActive                     bool                `json:"c1d_active"`
	ScoringVersion                string              `json:"scoring_version"`
	C1E3ProfilesConfigured        bool                `json:"c1e3_profiles_configured"`
	C1E3ProfilesExecuted          bool                `json:"c1e3_profiles_executed"`
	V2ContentsInspected           bool                `json:"v2_contents_inspected"`
	V2Files                       []C1E2FrozenFile    `json:"v2_frozen_files"`
	Profiles                      []C1E2FrozenProfile `json:"c1e3_profiles"`
}

// GenerateC1E2OfflineFreeze hashes v2 files as opaque byte streams. It does
// not decode their JSON and therefore cannot expose cases, labels, or wording.
func GenerateC1E2OfflineFreeze(repoRoot, outputRoot string, hostedAuthorization bool) (string, string, error) {
	if hostedAuthorization {
		return "", "", fmt.Errorf("C1E2 offline freeze requires hosted inference authorization=false")
	}
	rulesPath := filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json")
	rules, err := assetresolution.LoadRuleset(rulesPath)
	if err != nil {
		return "", "", err
	}
	if rules.Version != "event-asset-resolution-v1" {
		return "", "", fmt.Errorf("unexpected resolver identity %q", rules.Version)
	}
	exposures, err := (assetresolution.Resolver{Rules: rules}).ProxyExposures()
	if err != nil {
		return "", "", err
	}
	schemaSHA, err := fingerprint(V5OutputSchema(exposures))
	if err != nil {
		return "", "", err
	}

	v2Files := []C1E2FrozenFile{}
	for _, profileID := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, loadErr := LoadDiagnosticEvaluationProfile(profileID)
		if loadErr != nil {
			return "", "", loadErr
		}
		for _, file := range []struct{ identity, path, expected string }{
			{profile.ManifestVersion, profile.ManifestPath, profile.ManifestFileSHA256},
			{profile.FingerprintLockVersion, profile.FingerprintLockPath, profile.FingerprintLockFileSHA256},
			{profile.FreezeVersion, profile.FreezePath, profile.FreezeFileSHA256},
		} {
			actual, hashErr := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(file.path)))
			if hashErr != nil {
				return "", "", hashErr
			}
			if actual != file.expected {
				return "", "", fmt.Errorf("frozen v2 metadata hash mismatch for %s: got %s want %s", file.path, actual, file.expected)
			}
			v2Files = append(v2Files, C1E2FrozenFile{file.identity, filepath.ToSlash(file.path), file.expected, actual, true})
		}
	}

	policySourceSHA, err := hashOpaqueFile(filepath.Join(repoRoot, "internal", "modules", "aishadow", "causal_attribution.go"))
	if err != nil {
		return "", "", err
	}
	c1dSourceSHA, err := hashOpaqueFile(filepath.Join(repoRoot, "internal", "modules", "aishadow", "causal_consistency.go"))
	if err != nil {
		return "", "", err
	}
	resolverSHA, err := hashOpaqueFile(rulesPath)
	if err != nil {
		return "", "", err
	}

	profiles := []C1E2FrozenProfile{}
	for _, profileID := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, _ := LoadDiagnosticEvaluationProfile(profileID)
		prompt, output, policy := profile.executionVersions()
		if err := ValidateContractRoute(prompt, output, policy); err != nil {
			return "", "", err
		}
		profiles = append(profiles, C1E2FrozenProfile{
			DatasetIdentity: profile.Identity, ExperimentID: profile.RequiredExperimentID,
			EvidenceNamespace: profile.EvidenceNamespace, CaseCount: profile.CaseCount,
			PromptVersion: prompt, OutputContract: output, CausalPolicy: policy,
			ResolverPolicy: rules.Version, Repetitions: profile.DefaultRepetitions,
			MaximumBudgetUSD: formatUSDMicros(profile.MaximumBudgetMicros), Executed: false,
		})
	}

	freeze := C1E2OfflineFreeze{
		Identity: C1E2WorkPackageIdentity, EvidenceNamespace: C1E2OfflineEvidenceNamespace,
		GeneratedAt: time.Now().UTC(), ProviderContact: false, Inference: false,
		DatabaseMutation: false, TradingMutation: false, HostedInferenceAuthorized: false,
		PromptVersion: V5PromptVersion, PromptSHA256: rawHash(v5SystemPrompt),
		OutputContract: V5SchemaVersion, OpenAISchemaName: V5OpenAISchemaName, SchemaSHA256: schemaSHA,
		CausalAttributionPolicy: CausalAttributionPolicyVersion, CausalAttributionSourceSHA256: policySourceSHA,
		ResolverPolicy: rules.Version, ResolverSHA256: resolverSHA,
		C1DPolicy: CausalConsistencyPolicyVersion, C1DSourceSHA256: c1dSourceSHA, C1DActive: false,
		ScoringVersion:         CausalAttributionScoringVersion,
		C1E3ProfilesConfigured: true, C1E3ProfilesExecuted: false, V2ContentsInspected: false,
		V2Files: v2Files, Profiles: profiles,
	}
	raw, err := json.MarshalIndent(freeze, "", "  ")
	if err != nil {
		return "", "", err
	}
	raw = append(raw, '\n')
	directory := filepath.Join(outputRoot, C1E2OfflineEvidenceNamespace, C1E2WorkPackageIdentity)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", "", err
	}
	path := filepath.Join(directory, "offline-preflight.json")
	if _, err := os.Stat(path); err == nil {
		return "", "", fmt.Errorf("C1E2 offline freeze artifact already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(raw)
	return path, hex.EncodeToString(digest[:]), nil
}

func hashOpaqueFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
