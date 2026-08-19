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
)

const (
	C1F2BWorkPackageIdentity      = "WP-00.03C1F2B"
	C1F2BOfflineEvidenceNamespace = "offline-c1f2b-c1f3-execution-authorization-v1"
)

type C1F2BHashedFile struct {
	Identity       string `json:"identity"`
	Path           string `json:"path"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ActualSHA256   string `json:"actual_sha256"`
	Preserved      bool   `json:"preserved"`
}

type C1F2BPlanningEvidence struct {
	Profile                        string                `json:"profile"`
	ProfileFingerprint             string                `json:"profile_fingerprint"`
	ExperimentID                   string                `json:"experiment_id"`
	EvidenceNamespace              string                `json:"evidence_namespace"`
	Cases                          int                   `json:"cases"`
	Repetitions                    int                   `json:"repetitions"`
	RequestedModel                 string                `json:"requested_model"`
	BudgetCeilingUSD               string                `json:"budget_ceiling_usd"`
	EstimatedMaximumRunUSD         string                `json:"estimated_maximum_run_usd"`
	FrozenBindings                 C1F3FrozenBindingPlan `json:"frozen_bindings"`
	ProviderInputIsolated          bool                  `json:"provider_input_isolated"`
	ProviderContact                bool                  `json:"provider_contact"`
	Inference                      bool                  `json:"inference"`
	ExecutionAuthorized            bool                  `json:"execution_authorized"`
	EvidenceNamespaceCollisionFree bool                  `json:"evidence_namespace_collision_free"`
	DefaultExecutionDenied         bool                  `json:"default_execution_denied"`
	DefaultDenialReason            string                `json:"default_denial_reason"`
}

type C1F2BOfflineFreeze struct {
	Identity                         string                  `json:"identity"`
	EvidenceNamespace                string                  `json:"evidence_namespace"`
	GeneratedAt                      time.Time               `json:"generated_at"`
	AuthorizationMechanism           string                  `json:"authorization_mechanism"`
	AuthorizationFingerprint         string                  `json:"authorization_fingerprint"`
	AuthorizationSourceSHA256        string                  `json:"authorization_source_sha256"`
	AuthorizationSourceFiles         []C1F2BHashedFile       `json:"authorization_source_files"`
	ApplicableProfiles               []string                `json:"applicable_profiles"`
	OperatorOptInMechanism           string                  `json:"operator_opt_in_mechanism"`
	GlobalHostedAuthorization        string                  `json:"global_hosted_authorization"`
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
	FrozenFiles                      []C1F2BHashedFile       `json:"frozen_files"`
	Profiles                         []C1F2BPlanningEvidence `json:"profiles"`
}

func GenerateC1F2BOfflineFreeze(repoRoot, outputRoot string, hostedAuthorization, operatorOptIn bool) (string, string, error) {
	if hostedAuthorization || operatorOptIn {
		return "", "", fmt.Errorf("C1F2B offline freeze requires hosted authorization=false and C1F3 operator opt-in=false")
	}

	sourcePaths := []string{
		"internal/modules/aishadow/c1f3_authorization.go",
		"internal/modules/aishadow/diagnostic_profile.go",
		"internal/modules/aishadow/diagnostic_runner.go",
		"internal/modules/aishadow/openai_diagnostic.go",
		"cmd/ai-shadow-issuer-diagnostic/main.go",
	}
	sourceFiles, sourceSHA, err := c1f2bHashSourceSet(repoRoot, sourcePaths)
	if err != nil {
		return "", "", err
	}

	frozenSpecs := []struct{ identity, path, expected string }{
		{"prompt-v6-source", "internal/modules/aishadow/prompt_v6.go", "099dca2b0aecb6a19ae9fc4a2894fbf8c1690bc86f9771eea87533704ddb75a3"},
		{"output-v5-source", "internal/modules/aishadow/prompt_v5.go", "d6049704d911d318361a92d4016d4e18a60771a39924d3b06470aae68660d95c"},
		{"C1F-validator", "internal/modules/aishadow/validation_c1f.go", frozenC1FValidatorSHA256},
		{"C1E-policy", "internal/modules/aishadow/causal_attribution.go", frozenC1EPolicySHA256},
		{"semantic-identity", "internal/modules/aishadow/semantic_identity.go", frozenSemanticIdentitySHA256},
		{"C1F-scoring", "internal/modules/aishadow/scoring_c1f.go", frozenC1FScoringSourceSHA256},
		{"C1F-contract", "internal/modules/aishadow/contract_c1f.go", "93883d45b3a06bb9c782923c1e10f2afe49d3025db6c9590d178f437fda6df20"},
		{"C1F-execution", "internal/modules/aishadow/benchmark_c1f.go", "e05fdb1d1f68be4f6332513b1228dd5b28b16fbf65f49822ced7e71d3c7f8c68"},
		{"C1F2-freeze", "internal/modules/aishadow/c1f2_offline_freeze.go", "7bdf9ef8b36cab3ecfa7636114f716c7829106408ef6d1b4ddab7099121c2d50"},
		{"C1F2A-freeze", "internal/modules/aishadow/c1f2a_freeze.go", "b3ed8dba77772118217fd1f814d40a9c6ce00d6c549343858b40c335ca5aa4d2"},
		{"C1F3-profile-freeze", "internal/modules/aishadow/c1f3_profile.go", "0fe516528f1343ee011721bdc12a29c1d0846af3dc0c32d6d13766dc922c93ab"},
		{"resolver", "config/event-asset-resolution-v1.json", expectedAssetRulesetFileSHA256},
	}
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, loadErr := LoadC1F3EvaluationProfile(identity)
		if loadErr != nil {
			return "", "", loadErr
		}
		frozenSpecs = append(frozenSpecs,
			struct{ identity, path, expected string }{identity + "-manifest", profile.Dataset.ManifestPath, profile.Dataset.ManifestSHA256},
			struct{ identity, path, expected string }{identity + "-input-lock", profile.Dataset.InputLockPath, profile.Dataset.InputLockSHA256},
			struct{ identity, path, expected string }{identity + "-freeze", profile.Dataset.FreezePath, profile.Dataset.FreezeSHA256},
			struct{ identity, path, expected string }{identity + "-typed-sidecar", profile.TypedSidecarPath, profile.TypedSidecarSHA256},
		)
	}
	frozenSpecs = append(frozenSpecs,
		struct{ identity, path, expected string }{"C1F3-scoring-rubric", "config/ai-shadow-causal-attribution-scoring-c1f3-v1.json", "7f795302a4c33c1a7c103294c1cab590a548c9e2cd746ca7908a10dc2a4a65a9"},
		struct{ identity, path, expected string }{"C1F2A-adjudication-rubric", "config/ai-shadow-causal-attribution-adjudication-rubric-c1f2a-v1.json", "bf5548387b1627e1dedc92198705e555721c913df9a316e9b469fd69598dbac3"},
	)
	frozenFiles := make([]C1F2BHashedFile, 0, len(frozenSpecs))
	for _, spec := range frozenSpecs {
		actual, hashErr := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(spec.path)))
		if hashErr != nil || actual != spec.expected {
			return "", "", fmt.Errorf("frozen C1F2B input hash mismatch for %s", spec.path)
		}
		frozenFiles = append(frozenFiles, C1F2BHashedFile{spec.identity, spec.path, spec.expected, actual, true})
	}

	plans := make([]C1F2BPlanningEvidence, 0, 2)
	for _, identity := range []string{C1F3ProfileGeneralization, C1F3ProfileBoundary} {
		profile, loadErr := LoadDiagnosticExecutionProfile(identity)
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
		prepared, prepareErr = ApplyDiagnosticExecutionShape(prepared, newDiagnosticExecutionShape(profile, 1, true))
		if prepareErr != nil {
			return "", "", prepareErr
		}
		authorization := prepared.Plan.C1F3ExecutionAuthorization
		if authorization == nil || authorization.ExecutionAuthorized || !authorization.FrozenBindingsValid || !authorization.BudgetValid ||
			!authorization.EvidenceNamespaceCollisionFree || !authorization.ProviderInputIsolated || prepared.Plan.C1F3FrozenBindings == nil {
			return "", "", fmt.Errorf("C1F2B zero-network planning authorization state is invalid")
		}
		denialConfig := config
		denialConfig.APIKey = APISecret{value: "offline-synthetic-presence-only"}
		_, denialErr := PrepareHostedDiagnostic(paths, denialConfig, DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1})
		if denialErr == nil || !strings.Contains(denialErr.Error(), "--authorize-c1f3-execution") {
			return "", "", fmt.Errorf("C1F2B default execution authorization did not fail closed")
		}
		plans = append(plans, C1F2BPlanningEvidence{
			Profile: profile.Identity, ProfileFingerprint: prepared.Plan.C1F3FrozenBindings.ProfileFingerprint,
			ExperimentID: profile.RequiredExperimentID, EvidenceNamespace: prepared.Plan.HostedExperiment.EvidenceNamespace,
			Cases: profile.CaseCount, Repetitions: 1, RequestedModel: prepared.Config.Model,
			BudgetCeilingUSD: prepared.Plan.HostedExperiment.BudgetCeilingUSD, EstimatedMaximumRunUSD: prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD,
			FrozenBindings: *prepared.Plan.C1F3FrozenBindings, ProviderInputIsolated: true,
			ProviderContact: false, Inference: false, ExecutionAuthorized: false,
			EvidenceNamespaceCollisionFree: authorization.EvidenceNamespaceCollisionFree,
			DefaultExecutionDenied:         true, DefaultDenialReason: denialErr.Error(),
		})
	}

	freeze := C1F2BOfflineFreeze{
		Identity: C1F2BWorkPackageIdentity, EvidenceNamespace: C1F2BOfflineEvidenceNamespace, GeneratedAt: time.Now().UTC(),
		AuthorizationMechanism: C1F3ExecutionAuthorizationVersion, AuthorizationFingerprint: C1F3ExecutionAuthorizationFingerprint(),
		AuthorizationSourceSHA256: sourceSHA, AuthorizationSourceFiles: sourceFiles,
		ApplicableProfiles:     []string{C1F3ProfileGeneralization, C1F3ProfileBoundary},
		OperatorOptInMechanism: "--authorize-c1f3-execution", GlobalHostedAuthorization: OpenAIDiagnosticInferenceAuthEnv + "=true",
		DefaultExecutionAuthorization: false, HostedInferenceAuthorized: false, CredentialLoaded: false,
		ProviderConstructed: false, ProviderContact: false, Inference: false, DatabaseMutation: false, TradingMutation: false,
		ZeroNetworkPlanningPossible: true, ProviderConstructionRequiresBoth: true,
		FrozenFiles: frozenFiles, Profiles: plans,
	}
	raw, err := json.MarshalIndent(freeze, "", "  ")
	if err != nil {
		return "", "", err
	}
	raw = append(raw, '\n')
	directory := filepath.Join(outputRoot, C1F2BOfflineEvidenceNamespace, C1F2BWorkPackageIdentity)
	path := filepath.Join(directory, "offline-preflight.json")
	if _, err := os.Stat(path); err == nil {
		return "", "", fmt.Errorf("C1F2B offline freeze artifact already exists: %s", path)
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

func c1f2bHashSourceSet(repoRoot string, paths []string) ([]C1F2BHashedFile, string, error) {
	files := make([]C1F2BHashedFile, 0, len(paths))
	material := strings.Builder{}
	for _, relative := range paths {
		hash, err := hashOpaqueFile(filepath.Join(repoRoot, filepath.FromSlash(relative)))
		if err != nil {
			return nil, "", err
		}
		files = append(files, C1F2BHashedFile{Identity: "control-plane-source", Path: relative, ActualSHA256: hash, Preserved: true})
		material.WriteString(relative)
		material.WriteByte(0)
		material.WriteString(hash)
		material.WriteByte('\n')
	}
	digest := sha256.Sum256([]byte(material.String()))
	return files, hex.EncodeToString(digest[:]), nil
}
