package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	C1E2BOfflineEvidenceNamespace = "offline-c1e2b-c1e3-execution-authorization-v1"
	C1E2BWorkPackageIdentity      = "WP-00.03C1E2B"
)

type C1E2BHashedFile struct {
	Identity       string `json:"identity"`
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ActualSHA256   string `json:"actual_sha256"`
	Preserved      bool   `json:"preserved"`
}

type C1E2BPlanningEvidence struct {
	Profile                        string `json:"profile"`
	ExperimentID                   string `json:"experiment_id"`
	EvidenceNamespace              string `json:"evidence_namespace"`
	Cases                          int    `json:"cases"`
	Repetitions                    int    `json:"repetitions"`
	RequestedModel                 string `json:"requested_model"`
	PromptVersion                  string `json:"prompt_version"`
	OutputContract                 string `json:"output_contract"`
	SchemaSHA256                   string `json:"schema_sha256"`
	BudgetCeilingUSD               string `json:"budget_ceiling_usd"`
	EstimatedMaximumRunUSD         string `json:"estimated_maximum_run_usd"`
	ProviderContact                bool   `json:"provider_contact"`
	Inference                      bool   `json:"inference"`
	ExecutionAuthorized            bool   `json:"execution_authorized"`
	EvidenceNamespaceCollisionFree bool   `json:"evidence_namespace_collision_free"`
	DefaultExecutionDenied         bool   `json:"default_execution_denied"`
	DefaultDenialReason            string `json:"default_denial_reason"`
}

type C1E2BOfflineFreeze struct {
	Identity                         string                  `json:"identity"`
	EvidenceNamespace                string                  `json:"evidence_namespace"`
	GeneratedAt                      time.Time               `json:"generated_at"`
	AuthorizationMechanism           string                  `json:"authorization_mechanism"`
	AuthorizationSourceSHA256        string                  `json:"authorization_source_sha256"`
	AuthorizationSourceFiles         []C1E2BHashedFile       `json:"authorization_source_files"`
	DefaultExecutionAuthorization    bool                    `json:"default_execution_authorization"`
	HostedInferenceAuthorized        bool                    `json:"hosted_inference_authorized"`
	CredentialLoaded                 bool                    `json:"credential_loaded"`
	ProviderConstructed              bool                    `json:"provider_constructed"`
	ProviderContact                  bool                    `json:"provider_contact"`
	Inference                        bool                    `json:"inference"`
	DatabaseMutation                 bool                    `json:"database_mutation"`
	TradingMutation                  bool                    `json:"trading_mutation"`
	ZeroNetworkPlanningPossible      bool                    `json:"zero_network_planning_possible"`
	ProviderConstructionRequiresBoth bool                    `json:"provider_construction_requires_both_authorizations"`
	PromptVersion                    string                  `json:"prompt_version"`
	PromptSHA256                     string                  `json:"prompt_sha256"`
	OutputContract                   string                  `json:"output_contract"`
	SchemaSHA256                     string                  `json:"schema_sha256"`
	CausalPolicy                     string                  `json:"causal_policy"`
	CausalPolicySourceSHA256         string                  `json:"causal_policy_source_sha256"`
	ResolverIdentity                 string                  `json:"resolver_identity"`
	ResolverSHA256                   string                  `json:"resolver_sha256"`
	C1DIdentity                      string                  `json:"c1d_identity"`
	C1DSourceSHA256                  string                  `json:"c1d_source_sha256"`
	FrozenFiles                      []C1E2BHashedFile       `json:"frozen_files"`
	Profiles                         []C1E2BPlanningEvidence `json:"profiles"`
}

func GenerateC1E2BOfflineFreeze(repoRoot, outputRoot string, hostedAuthorization, operatorOptIn bool) (string, string, error) {
	if hostedAuthorization || operatorOptIn {
		return "", "", fmt.Errorf("C1E2B offline freeze requires hosted authorization=false and C1E3 operator opt-in=false")
	}
	rulesPath := filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json")
	rulesHash, err := hashOpaqueFile(rulesPath)
	if err != nil {
		return "", "", err
	}

	sourcePaths := []string{
		"internal/modules/aishadow/c1e3_authorization.go",
		"internal/modules/aishadow/diagnostic_runner.go",
		"internal/modules/aishadow/openai_diagnostic.go",
		"cmd/ai-shadow-issuer-diagnostic/main.go",
	}
	sourceFiles := make([]C1E2BHashedFile, 0, len(sourcePaths))
	sourceMaterial := strings.Builder{}
	for _, relative := range sourcePaths {
		hash, hashErr := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(relative)))
		if hashErr != nil {
			return "", "", hashErr
		}
		sourceFiles = append(sourceFiles, C1E2BHashedFile{Identity: "control-plane-source", Path: relative, ActualSHA256: hash, Preserved: true})
		sourceMaterial.WriteString(relative)
		sourceMaterial.WriteByte(0)
		sourceMaterial.WriteString(hash)
		sourceMaterial.WriteByte('\n')
	}
	sourceDigest := sha256.Sum256([]byte(sourceMaterial.String()))

	frozenSpecs := []struct {
		identity string
		path     string
		expected string
	}{
		{"resolver", "config/event-asset-resolution-v1.json", expectedAssetRulesetFileSHA256},
		{"generalization-manifest", "config/ai-shadow-issuer-generalization-holdout-v2.json", "7b22c4c6d72d53d9976df17463bd6116a50ac305008c6d71c5a36f6971091c04"},
		{"generalization-input-lock", "config/ai-shadow-issuer-generalization-holdout-input-fingerprints-v2.json", "c3dc9715e5c7bcc1f8e0cb1020d95e7979a79f4c5748abf0124df5fa76e1cf88"},
		{"generalization-freeze", "config/ai-shadow-issuer-generalization-holdout-freeze-v2.json", "e32eb3ef76a234b5b53db2cea9c011a4e1d85571c6d4b865e1498cd48761d878"},
		{"boundary-manifest", "config/ai-shadow-issuer-boundary-challenge-v2.json", "ae2e15a18e28094c44663bd94bc8f40145e3fd1358ae46e525fca85166ce7578"},
		{"boundary-input-lock", "config/ai-shadow-issuer-boundary-challenge-input-fingerprints-v2.json", "3cced77fbc0d2d229143f379d22365981a668dce6e93174027b0dcfe7a137112"},
		{"boundary-freeze", "config/ai-shadow-issuer-boundary-challenge-freeze-v2.json", "0123286e7e0862961368e85ebea81d3474b49008bd70f85b5c21f7ad75f80dc2"},
		{"generalization-labels", "config/ai-shadow-causal-attribution-labels-generalization-v2-v1.json", "7092398e52df79c375b850e2d408e689ab879fecf1b2d37a63520e2cd98d2855"},
		{"boundary-labels", "config/ai-shadow-causal-attribution-labels-boundary-v2-v1.json", "54be48060bc4c430f19824ada39db4054bbdb93ac46bb698d939ddbf8a61c5d5"},
		{"scoring", "config/ai-shadow-causal-attribution-scoring-c1e3-v1.json", "d9b415d0201657ad7c322e5b3dc9064d4ae4e0f572e5b3232b1b1d66b367066b"},
	}
	frozenFiles := make([]C1E2BHashedFile, 0, len(frozenSpecs))
	for _, spec := range frozenSpecs {
		actual, hashErr := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(spec.path)))
		if hashErr != nil || actual != spec.expected {
			return "", "", fmt.Errorf("frozen C1E2B input hash mismatch for %s", spec.path)
		}
		frozenFiles = append(frozenFiles, C1E2BHashedFile{spec.identity, spec.path, spec.expected, actual, true})
	}

	policyHash, err := hashOpaqueFile(filepath.Join(repoRoot, "internal", "modules", "aishadow", "causal_attribution.go"))
	if err != nil || policyHash != frozenC1EPolicySHA256 {
		return "", "", fmt.Errorf("frozen C1E policy source hash changed")
	}
	c1dHash, err := hashOpaqueFile(filepath.Join(repoRoot, "internal", "modules", "aishadow", "causal_consistency.go"))
	if err != nil || c1dHash != "1f07d7854ef733a55a8172419b060ce5806ebf28fad3ab7ab8392e3cc5bdd895" {
		return "", "", fmt.Errorf("frozen C1D source hash changed")
	}

	plans := []C1E2BPlanningEvidence{}
	for _, profileID := range []string{DiagnosticProfileGeneralizationV2, DiagnosticProfileBoundaryV2} {
		profile, loadErr := LoadDiagnosticEvaluationProfile(profileID)
		if loadErr != nil {
			return "", "", loadErr
		}
		values := c1e2bConfigValues(profile)
		config, loadErr := LoadOpenAIDiagnosticConfigForProfile(mapLookupFromValues(values), profile, false)
		if loadErr != nil {
			return "", "", loadErr
		}
		paths := c1e2bProfilePaths(repoRoot, filepath.Join(outputRoot, "planning"), profile)
		prepared, prepareErr := PrepareHostedDiagnosticPreflight(paths, config, DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1})
		if prepareErr != nil {
			return "", "", prepareErr
		}
		shape := newDiagnosticExecutionShape(profile, 1, true)
		prepared, prepareErr = ApplyDiagnosticExecutionShape(prepared, shape)
		if prepareErr != nil {
			return "", "", prepareErr
		}
		if prepareErr = ValidateDiagnosticExecutionShape(prepared); prepareErr != nil {
			return "", "", prepareErr
		}
		authorization := prepared.Plan.C1E3ExecutionAuthorization
		if authorization == nil || authorization.ExecutionAuthorized || !authorization.BudgetValid || !authorization.EvidenceNamespaceCollisionFree {
			return "", "", fmt.Errorf("C1E2B zero-network planning authorization state is invalid")
		}

		denialConfig := config
		denialConfig.APIKey = APISecret{value: "offline-synthetic-presence-only"}
		_, denialErr := PrepareHostedDiagnostic(paths, denialConfig, DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1})
		if denialErr == nil || !strings.Contains(denialErr.Error(), "--authorize-c1e3-execution") {
			return "", "", fmt.Errorf("C1E2B default execution authorization did not fail closed")
		}
		plans = append(plans, C1E2BPlanningEvidence{
			Profile: profile.Identity, ExperimentID: profile.RequiredExperimentID, EvidenceNamespace: prepared.Plan.HostedExperiment.EvidenceNamespace,
			Cases: profile.CaseCount, Repetitions: 1, RequestedModel: prepared.Config.Model,
			PromptVersion: prepared.Plan.PromptVersion, OutputContract: prepared.Plan.OutputContract, SchemaSHA256: prepared.Plan.HostedExperiment.SchemaSHA256,
			BudgetCeilingUSD: prepared.Plan.HostedExperiment.BudgetCeilingUSD, EstimatedMaximumRunUSD: prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD,
			ProviderContact: false, Inference: false, ExecutionAuthorized: false,
			EvidenceNamespaceCollisionFree: authorization.EvidenceNamespaceCollisionFree,
			DefaultExecutionDenied:         true, DefaultDenialReason: denialErr.Error(),
		})
	}

	rules, err := os.ReadFile(rulesPath)
	if err != nil || rulesHash != expectedAssetRulesetFileSHA256 || len(rules) == 0 {
		return "", "", fmt.Errorf("resolver preservation check failed")
	}
	schemaSHA, err := c1e2bV5SchemaSHA(repoRoot)
	if err != nil {
		return "", "", err
	}
	freeze := C1E2BOfflineFreeze{
		Identity: C1E2BWorkPackageIdentity, EvidenceNamespace: C1E2BOfflineEvidenceNamespace, GeneratedAt: time.Now().UTC(),
		AuthorizationMechanism: C1E3ExecutionAuthorizationVersion, AuthorizationSourceSHA256: hex.EncodeToString(sourceDigest[:]), AuthorizationSourceFiles: sourceFiles,
		DefaultExecutionAuthorization: false, HostedInferenceAuthorized: false, CredentialLoaded: false,
		ProviderConstructed: false, ProviderContact: false, Inference: false, DatabaseMutation: false, TradingMutation: false,
		ZeroNetworkPlanningPossible: true, ProviderConstructionRequiresBoth: true,
		PromptVersion: V5PromptVersion, PromptSHA256: rawHash(v5SystemPrompt), OutputContract: V5SchemaVersion, SchemaSHA256: schemaSHA,
		CausalPolicy: CausalAttributionPolicyVersion, CausalPolicySourceSHA256: policyHash,
		ResolverIdentity: "event-asset-resolution-v1", ResolverSHA256: rulesHash,
		C1DIdentity: CausalConsistencyPolicyVersion, C1DSourceSHA256: c1dHash,
		FrozenFiles: frozenFiles, Profiles: plans,
	}
	raw, err := json.MarshalIndent(freeze, "", "  ")
	if err != nil {
		return "", "", err
	}
	raw = append(raw, '\n')
	directory := filepath.Join(outputRoot, C1E2BOfflineEvidenceNamespace, C1E2BWorkPackageIdentity)
	path := filepath.Join(directory, "offline-preflight.json")
	if _, err := os.Stat(path); err == nil {
		return "", "", fmt.Errorf("C1E2B offline freeze artifact already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return "", "", err
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(raw)
	return path, hex.EncodeToString(digest[:]), nil
}

func c1e2bConfigValues(profile DiagnosticEvaluationProfile) map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": OpenAIDiagnosticProvider, "JAX_AI_MODEL": OpenAIDiagnosticLunaModel,
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_MAX_EVENTS": fmt.Sprint(profile.CaseCount),
		"JAX_AI_EXPERIMENT_ID": profile.RequiredExperimentID, "JAX_AI_REASONING_EFFORT": OpenAIDiagnosticReasoningEffort,
		"JAX_AI_MAX_OUTPUT_TOKENS": "256", "JAX_AI_EXPERIMENT_BUDGET_USD": formatUSDMicros(profile.MaximumBudgetMicros),
		"JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.20", "JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.02",
		"JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS": "0.25", "JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS": "1.20",
		OpenAIDiagnosticContractModeEnv: OpenAIStructuredOutputsMode, OpenAIDiagnosticInferenceAuthEnv: "false",
	}
}

func mapLookupFromValues(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

func c1e2bProfilePaths(repoRoot, outputRoot string, profile DiagnosticEvaluationProfile) DiagnosticPaths {
	return DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        filepath.Join(repoRoot, filepath.FromSlash(profile.ManifestPath)), FingerprintLockPath: filepath.Join(repoRoot, filepath.FromSlash(profile.FingerprintLockPath)),
		FreezePath: filepath.Join(repoRoot, filepath.FromSlash(profile.FreezePath)), AssetRulesetPath: filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json"),
		TypedLabelPath: filepath.Join(repoRoot, filepath.FromSlash(profile.TypedLabelPath)), ScoringRubricPath: filepath.Join(repoRoot, filepath.FromSlash(profile.ScoringRubricPath)),
		OutputRoot: filepath.Join(outputRoot, profile.EvidenceNamespace, profile.RequiredExperimentID),
	}
}

func c1e2bV5SchemaSHA(repoRoot string) (string, error) {
	rules, err := os.ReadFile(filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json"))
	if err != nil || len(rules) == 0 {
		return "", fmt.Errorf("read resolver for v5 schema verification: %w", err)
	}
	loaded, err := assetresolution.LoadRuleset(filepath.Join(repoRoot, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		return "", err
	}
	exposures, err := (assetresolution.Resolver{Rules: loaded}).ProxyExposures()
	if err != nil {
		return "", err
	}
	sha, err := fingerprint(V5OutputSchema(exposures))
	if err != nil || sha != frozenV5SchemaSHA256 {
		return "", fmt.Errorf("frozen v5 schema hash changed")
	}
	return sha, nil
}
