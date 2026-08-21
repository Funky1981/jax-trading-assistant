package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"

	"github.com/google/uuid"
)

const (
	DiagnosticReportVersion  = "ai-shadow-issuer-diagnostic-report-v2"
	DiagnosticRepetitionsEnv = "JAX_AI_DIAGNOSTIC_REPETITIONS"
)

type DiagnosticPaths struct {
	EvaluationProfileID string
	ManifestPath        string
	FingerprintLockPath string
	FreezePath          string
	AssetRulesetPath    string
	TypedLabelPath      string
	ScoringRubricPath   string
	OutputRoot          string
}

type DiagnosticSafetyState struct {
	RuntimeMode      string  `json:"runtime_mode"`
	AllowLiveTrading bool    `json:"allow_live_trading"`
	ExecutionEnabled bool    `json:"execution_enabled"`
	ExecutionWorker  bool    `json:"execution_worker_enabled"`
	BrokerExecution  bool    `json:"broker_execution_allowed"`
	MaximumLeverage  float64 `json:"maximum_leverage"`
}

type DiagnosticModelConfiguration struct {
	Provider             string  `json:"provider"`
	Model                string  `json:"model"`
	BaseURL              string  `json:"base_url"`
	TimeoutSeconds       int     `json:"timeout_seconds"`
	Temperature          float64 `json:"temperature"`
	Seed                 int64   `json:"seed"`
	Stream               bool    `json:"stream"`
	Think                bool    `json:"think"`
	RetryLimit           int     `json:"retry_limit"`
	ReasoningEffort      string  `json:"reasoning_effort,omitempty"`
	MaxOutputTokens      int     `json:"max_output_tokens,omitempty"`
	StructuredOutputMode string  `json:"structured_output_mode,omitempty"`
	StructuredOutputs    bool    `json:"structured_outputs,omitempty"`
	SchemaContract       string  `json:"schema_contract,omitempty"`
	ContractEnforcement  string  `json:"contract_enforcement,omitempty"`
	ServiceTier          string  `json:"service_tier,omitempty"`
	ThinkingMode         string  `json:"thinking_mode,omitempty"`
}

type HostedExperimentPlan struct {
	ExperimentID                       string            `json:"experiment_id"`
	CellIdentity                       string            `json:"cell_identity,omitempty"`
	EvidenceNamespace                  string            `json:"evidence_namespace"`
	SchemaContract                     string            `json:"schema_contract,omitempty"`
	SchemaSHA256                       string            `json:"schema_sha256,omitempty"`
	StructuredOutputs                  bool              `json:"structured_outputs,omitempty"`
	ContractEnforcement                string            `json:"contract_enforcement,omitempty"`
	ServiceTier                        string            `json:"service_tier,omitempty"`
	Endpoint                           string            `json:"endpoint"`
	APIKeyEnvironment                  string            `json:"api_key_environment"`
	APIKeyPresent                      bool              `json:"api_key_present"`
	InferenceExplicitlyAuthorized      bool              `json:"inference_explicitly_authorized"`
	BudgetCeilingUSD                   string            `json:"budget_ceiling_usd"`
	Pricing                            HostedPricingPlan `json:"pricing"`
	BaseRequestCount                   int               `json:"base_request_count"`
	MaximumRequestCount                int               `json:"maximum_request_count"`
	EstimatedFirstRequestMaxUSD        string            `json:"estimated_first_request_max_usd"`
	EstimatedInitialRunUSD             string            `json:"estimated_initial_run_usd,omitempty"`
	LargestFrozenInitialRequestBytes   int               `json:"largest_frozen_initial_request_bytes,omitempty"`
	ConservativeCorrectiveRequestBytes int               `json:"conservative_corrective_request_bytes,omitempty"`
	EstimatedMaximumRunUSD             string            `json:"estimated_maximum_run_usd,omitempty"`
	DatabaseMutationAllowed            bool              `json:"database_mutation_allowed"`
	TradingStateMutationAllowed        bool              `json:"trading_state_mutation_allowed"`
}

type DiagnosticPlanEvent struct {
	Position         int    `json:"position"`
	ID               string `json:"id"`
	Category         string `json:"category"`
	InputFingerprint string `json:"input_fingerprint"`
}

type DiagnosticPlan struct {
	Version                                   string                                         `json:"version"`
	EvaluationProfile                         string                                         `json:"evaluation_profile"`
	DatasetIdentity                           string                                         `json:"dataset_identity"`
	ManifestVersion                           string                                         `json:"manifest_version"`
	ManifestFingerprint                       string                                         `json:"manifest_fingerprint"`
	ManifestFileSHA256                        string                                         `json:"manifest_file_sha256"`
	FingerprintLockVersion                    string                                         `json:"fingerprint_lock_version"`
	FingerprintLockFingerprint                string                                         `json:"fingerprint_lock_fingerprint"`
	FingerprintLockFileSHA256                 string                                         `json:"fingerprint_lock_file_sha256"`
	FreezeVersion                             string                                         `json:"freeze_version,omitempty"`
	FreezeFileSHA256                          string                                         `json:"freeze_file_sha256,omitempty"`
	LabelVersion                              string                                         `json:"label_version"`
	PromptVersion                             string                                         `json:"prompt_version"`
	OutputContract                            string                                         `json:"output_contract"`
	ExecutionRoute                            diagnosticExecutionRoute                       `json:"execution_route"`
	ValidatorVersion                          string                                         `json:"validator_version"`
	PolicyVersion                             string                                         `json:"policy_version"`
	CausalConsistencyPolicy                   string                                         `json:"causal_consistency_policy"`
	CausalAttributionPolicy                   string                                         `json:"causal_attribution_policy,omitempty"`
	ScoringVersion                            string                                         `json:"scoring_version,omitempty"`
	TypedLabelVersion                         string                                         `json:"typed_label_version,omitempty"`
	TypedLabelFileSHA256                      string                                         `json:"typed_label_file_sha256,omitempty"`
	TypedLabelFingerprint                     string                                         `json:"typed_label_fingerprint,omitempty"`
	ScoringRubricVersion                      string                                         `json:"scoring_rubric_version,omitempty"`
	ScoringRubricFileSHA256                   string                                         `json:"scoring_rubric_file_sha256,omitempty"`
	ScoringRubricFingerprint                  string                                         `json:"scoring_rubric_fingerprint,omitempty"`
	C1E3ExecutionAuthorization                *C1E3ExecutionAuthorizationPlan                `json:"c1e3_execution_authorization,omitempty"`
	C1F3FrozenBindings                        *C1F3FrozenBindingPlan                         `json:"c1f3_frozen_bindings,omitempty"`
	C1F3ExecutionAuthorization                *C1F3ExecutionAuthorizationPlan                `json:"c1f3_execution_authorization,omitempty"`
	C1F3RepeatabilityFrozenBindings           *C1F3RepeatabilityFrozenBindingPlan            `json:"c1f3_repeatability_frozen_bindings,omitempty"`
	C1F3RepeatabilityExecutionAuthorization   *C1F3RepeatabilityExecutionAuthorizationPlan   `json:"c1f3_repeatability_execution_authorization,omitempty"`
	C1F3RepeatabilityR3ExecutionAuthorization *C1F3RepeatabilityR3ExecutionAuthorizationPlan `json:"c1f3_repeatability_r3_execution_authorization,omitempty"`
	C1F3TerraChallengerFrozenBindings         *C1F3TerraChallengerFrozenBindingPlan          `json:"c1f3_terra_challenger_frozen_bindings,omitempty"`
	C1F3TerraChallengerExecutionAuthorization *C1F3TerraChallengerExecutionAuthorizationPlan `json:"c1f3_terra_challenger_execution_authorization,omitempty"`
	Repetitions                               int                                            `json:"repetitions"`
	CasesPerRepetition                        int                                            `json:"cases_per_repetition"`
	ExecutionShape                            DiagnosticExecutionShape                       `json:"execution_shape"`
	ModelConfiguration                        DiagnosticModelConfiguration                   `json:"model_configuration"`
	Safety                                    DiagnosticSafetyState                          `json:"safety"`
	HostedExperiment                          *HostedExperimentPlan                          `json:"hosted_experiment,omitempty"`
	Events                                    []DiagnosticPlanEvent                          `json:"events"`
}

type DiagnosticExecutionShape struct {
	RepetitionEnvironment string `json:"repetition_environment"`
	OverrideSupplied      bool   `json:"override_supplied"`
	RequestedRepetitions  int    `json:"requested_repetitions"`
	EffectiveRepetitions  int    `json:"effective_repetitions"`
	CasesPerRepetition    int    `json:"cases_per_repetition"`
	TotalPlannedCases     int    `json:"total_planned_cases"`
}

type PreparedDiagnostic struct {
	Profile        DiagnosticEvaluationProfile
	Plan           DiagnosticPlan
	Manifest       DiagnosticManifest
	Lock           DiagnosticFingerprintLock
	Freeze         *DiagnosticFreezeRecord
	Config         Config
	Resolver       assetresolution.Resolver
	ProxyExposures []string
	Paths          DiagnosticPaths
	ExecutionShape DiagnosticExecutionShape
}

type DiagnosticModelIdentity struct {
	Name              string `json:"name"`
	Digest            string `json:"digest"`
	Format            string `json:"format,omitempty"`
	Family            string `json:"family,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
}

type DiagnosticAttemptAudit struct {
	RunID                    string                     `json:"run_id"`
	Repetition               int                        `json:"repetition"`
	CaseID                   string                     `json:"case_id"`
	Category                 string                     `json:"category"`
	AttemptNumber            int                        `json:"attempt_number"`
	InputFingerprint         string                     `json:"input_fingerprint"`
	Provider                 string                     `json:"provider"`
	ConfiguredModel          string                     `json:"configured_model"`
	ModelReportedIdentifier  string                     `json:"model_reported_identifier,omitempty"`
	PromptVersion            string                     `json:"prompt_version"`
	OutputContract           string                     `json:"output_contract"`
	PolicyVersion            string                     `json:"policy_version"`
	Seed                     int64                      `json:"seed"`
	Temperature              float64                    `json:"temperature"`
	RequestTimestamp         time.Time                  `json:"request_timestamp"`
	ResponseTimestamp        time.Time                  `json:"response_timestamp"`
	DurationMS               int64                      `json:"duration_ms"`
	RawResponseHash          string                     `json:"raw_response_hash"`
	RawResponseBody          string                     `json:"raw_response_body"`
	ValidationStatus         string                     `json:"validation_status"`
	ValidationErrors         []string                   `json:"validation_errors"`
	FailureReason            string                     `json:"failure_reason,omitempty"`
	RequestID                string                     `json:"request_id,omitempty"`
	ResponseID               string                     `json:"response_id,omitempty"`
	ProviderStatus           string                     `json:"provider_status,omitempty"`
	SystemFingerprint        string                     `json:"system_fingerprint,omitempty"`
	FinishReason             string                     `json:"finish_reason,omitempty"`
	Usage                    ProviderUsage              `json:"usage"`
	ModelClassification      *StructuredResult          `json:"model_classification,omitempty"`
	V5RawModelOutput         *V5StructuredResult        `json:"v5_raw_model_output,omitempty"`
	TypedAttribution         *TypedCausalAttribution    `json:"typed_causal_attribution,omitempty"`
	CausalConsistencyGuard   *CausalConsistencyDecision `json:"causal_consistency_guard,omitempty"`
	CausalAttributionPolicy  *CausalAttributionDecision `json:"causal_attribution_policy_decision,omitempty"`
	EffectiveSemanticMapping *AssetMapping              `json:"effective_semantic_mapping,omitempty"`
	DeterministicResolution  *PolicyResolution          `json:"deterministic_resolution,omitempty"`
}

type DiagnosticCaseRun struct {
	CaseID           string          `json:"case_id"`
	Category         string          `json:"category"`
	InputFingerprint string          `json:"input_fingerprint"`
	Attempts         []Attempt       `json:"attempts"`
	Traces           []ProviderTrace `json:"traces"`
	Result           EventResult     `json:"result"`
}

type DiagnosticAuditPaths struct {
	RunID               string `json:"run_id"`
	Directory           string `json:"directory"`
	Plan                string `json:"plan,omitempty"`
	ReportJSON          string `json:"report_json,omitempty"`
	ReportMarkdown      string `json:"report_markdown,omitempty"`
	Preflight           string `json:"preflight,omitempty"`
	StopRecord          string `json:"stop_record,omitempty"`
	ArtifactIndex       string `json:"artifact_index,omitempty"`
	ArtifactIndexSHA256 string `json:"artifact_index_sha256,omitempty"`
}

func PrepareDiagnostic(paths DiagnosticPaths, config Config, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	profileID := paths.EvaluationProfileID
	if profileID == "" {
		profileID = DiagnosticProfileOriginal
	}
	profile, err := LoadDiagnosticExecutionProfile(profileID)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	executionContract, err := diagnosticExecutionContractForProfile(profile)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	if profile.RequiresTypedAttributionLabels && !usesC1F3SemanticStack(profile) {
		if _, err := LoadFrozenTypedLabelSidecarForProfile(profile, paths.TypedLabelPath); err != nil {
			return PreparedDiagnostic{}, err
		}
		if _, err := LoadFrozenC1E3ScoringRubric(profile, paths.ScoringRubricPath); err != nil {
			return PreparedDiagnostic{}, err
		}
	}
	if config.MaxEvents != profile.CaseCount {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s requires JAX_AI_MAX_EVENTS=%d", profile.Identity, profile.CaseCount)
	}
	if profile.RequiredProvider != "" && config.Provider != profile.RequiredProvider {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s requires provider %s", profile.Identity, profile.RequiredProvider)
	}
	if safety.RuntimeMode != "paper" || safety.AllowLiveTrading || safety.ExecutionEnabled || safety.ExecutionWorker || safety.BrokerExecution || safety.MaximumLeverage > 1 || safety.MaximumLeverage <= 0 {
		return PreparedDiagnostic{}, fmt.Errorf("unsafe issuer diagnostic state")
	}
	if profile.isHoldout() {
		policyHash, err := diagnosticFileSHA256(paths.AssetRulesetPath)
		if err != nil {
			return PreparedDiagnostic{}, err
		}
		if policyHash != expectedAssetRulesetFileSHA256 {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s resolver policy hash changed: got %s want %s", profile.Identity, policyHash, expectedAssetRulesetFileSHA256)
		}
	}
	rules, err := assetresolution.LoadRuleset(paths.AssetRulesetPath)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	if isC1E3Profile(profile) {
		if err := validateC1E3FrozenSemanticSources(paths, rules.Version, exposures); err != nil {
			return PreparedDiagnostic{}, err
		}
	}
	var manifest DiagnosticManifest
	var lock DiagnosticFingerprintLock
	var freeze *DiagnosticFreezeRecord
	var c1f3Bindings *C1F3FrozenBindingPlan
	var repeatabilityBindings *C1F3RepeatabilityFrozenBindingPlan
	var terraChallengerBindings *C1F3TerraChallengerFrozenBindingPlan
	if isC1F3TerraChallengerProfile(profile) {
		loadedManifest, loadedLock, loadedFreeze, semanticBindings, bindings, loadErr := loadC1F3TerraChallengerExecutionInputs(paths, profile, resolver, exposures)
		if loadErr != nil {
			return PreparedDiagnostic{}, loadErr
		}
		manifest, lock, freeze = loadedManifest, loadedLock, &loadedFreeze
		c1f3Bindings = &semanticBindings
		terraChallengerBindings = &bindings
	} else if isC1F3RepeatabilityProfile(profile) {
		loadedManifest, loadedLock, loadedFreeze, semanticBindings, bindings, loadErr := loadC1F3RepeatabilityExecutionInputs(paths, profile, resolver, exposures)
		if loadErr != nil {
			return PreparedDiagnostic{}, loadErr
		}
		manifest, lock, freeze = loadedManifest, loadedLock, &loadedFreeze
		c1f3Bindings = &semanticBindings
		repeatabilityBindings = &bindings
	} else if isC1F3Profile(profile) {
		loadedManifest, loadedLock, loadedFreeze, bindings, loadErr := loadC1F3ExecutionInputs(paths, profile, resolver, exposures)
		if loadErr != nil {
			return PreparedDiagnostic{}, loadErr
		}
		manifest, lock, freeze = loadedManifest, loadedLock, &loadedFreeze
		c1f3Bindings = &bindings
	} else {
		manifest, err = LoadFrozenDiagnosticManifestForProfile(profile, paths.ManifestPath, exposures)
		if err != nil {
			return PreparedDiagnostic{}, err
		}
		lock, err = LoadDiagnosticFingerprintLockForProfile(profile, paths.FingerprintLockPath)
		if err != nil {
			return PreparedDiagnostic{}, err
		}
		if manifest.OutputContract != SchemaVersion || lock.OutputContract != SchemaVersion {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic output contract mismatch")
		}
		if lock.PromptVersion != PromptVersion {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic prompt version mismatch")
		}
	}
	if manifest.PolicyVersion != rules.Version || lock.PolicyVersion != rules.Version {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic policy version mismatch")
	}
	if lock.ManifestFingerprint != manifest.Fingerprint {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic fingerprint lock references a different manifest")
	}
	if profile.isHoldout() && !usesC1F3SemanticStack(profile) {
		if strings.TrimSpace(paths.FreezePath) == "" {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s requires its registered freeze record", profile.Identity)
		}
		loaded, err := LoadDiagnosticFreezeRecord(profile, paths.FreezePath)
		if err != nil {
			return PreparedDiagnostic{}, err
		}
		freeze = &loaded
	} else if !usesC1F3SemanticStack(profile) && strings.TrimSpace(paths.FreezePath) != "" {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s does not accept a freeze record", profile.Identity)
	}

	planEvents := make([]DiagnosticPlanEvent, 0, profile.CaseCount)
	for index, event := range manifest.Events {
		locked := lock.Events[index]
		if locked.ID != event.ID {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic event order mismatch at position %d: got %s want %s", index+1, event.ID, locked.ID)
		}
		got, err := EventInputFingerprint(event.Input)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("fingerprint issuer diagnostic event %s: %w", event.ID, err)
		}
		if got != locked.InputFingerprint {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic event %s input fingerprint changed: got %s want %s", event.ID, got, locked.InputFingerprint)
		}
		planEvents = append(planEvents, DiagnosticPlanEvent{Position: index + 1, ID: event.ID, Category: event.Category, InputFingerprint: got})
	}

	executionShape := newDiagnosticExecutionShape(profile, profile.DefaultRepetitions, false)
	executionPrompt, executionOutput, executionPolicy := executionContract.Prompt, executionContract.Output, executionContract.Policy
	plan := DiagnosticPlan{
		Version: DiagnosticReportVersion, EvaluationProfile: profile.Identity, DatasetIdentity: profile.ManifestVersion, ManifestVersion: manifest.Version,
		ManifestFingerprint: manifest.Fingerprint, ManifestFileSHA256: profile.ManifestFileSHA256,
		FingerprintLockVersion: lock.Version, FingerprintLockFingerprint: lock.Fingerprint, FingerprintLockFileSHA256: profile.FingerprintLockFileSHA256,
		FreezeVersion: profile.FreezeVersion, FreezeFileSHA256: profile.FreezeFileSHA256,
		LabelVersion: manifest.LabelVersion, PromptVersion: executionPrompt, OutputContract: executionOutput,
		ExecutionRoute: executionContract.Route, ValidatorVersion: executionContract.Validator,
		PolicyVersion: rules.Version,
		Repetitions:   executionShape.EffectiveRepetitions, CasesPerRepetition: profile.CaseCount,
		ExecutionShape: executionShape,
		ModelConfiguration: DiagnosticModelConfiguration{
			Provider: config.Provider, Model: config.Model, BaseURL: config.BaseURL,
			TimeoutSeconds: int(config.Timeout.Seconds()), Temperature: config.Temperature,
			Seed: config.Seed, Stream: false, Think: false, RetryLimit: 1,
		},
		Safety: safety, Events: planEvents,
	}
	if executionOutput == V5SchemaVersion {
		plan.CausalAttributionPolicy = executionPolicy
		plan.ScoringVersion = profile.ScoringVersion
		plan.TypedLabelVersion = profile.TypedLabelVersion
		plan.TypedLabelFileSHA256 = profile.TypedLabelFileSHA256
		plan.TypedLabelFingerprint = profile.TypedLabelFingerprint
		plan.ScoringRubricVersion = profile.ScoringRubricVersion
		plan.ScoringRubricFileSHA256 = profile.ScoringRubricFileSHA256
		plan.ScoringRubricFingerprint = profile.ScoringRubricFingerprint
	} else {
		plan.CausalConsistencyPolicy = executionPolicy
	}
	plan.C1F3FrozenBindings = c1f3Bindings
	plan.C1F3RepeatabilityFrozenBindings = repeatabilityBindings
	plan.C1F3TerraChallengerFrozenBindings = terraChallengerBindings
	return PreparedDiagnostic{Profile: profile, Plan: plan, Manifest: manifest, Lock: lock, Freeze: freeze, Config: config, Resolver: resolver, ProxyExposures: exposures, Paths: paths, ExecutionShape: executionShape}, nil
}

func LoadDiagnosticRepetitionSelection(lookup func(string) (string, bool)) (DiagnosticExecutionShape, error) {
	profile, _ := LoadDiagnosticEvaluationProfile(DiagnosticProfileOriginal)
	shape, err := LoadDiagnosticRepetitionSelectionForProfile(lookup, profile)
	if err != nil {
		return DiagnosticExecutionShape{}, fmt.Errorf("%s must be exactly 1 or %d", DiagnosticRepetitionsEnv, diagnosticRepetitionCount)
	}
	return shape, nil
}

func LoadDiagnosticRepetitionSelectionForProfile(lookup func(string) (string, bool), profile DiagnosticEvaluationProfile) (DiagnosticExecutionShape, error) {
	raw, supplied := lookup(DiagnosticRepetitionsEnv)
	if !supplied {
		return newDiagnosticExecutionShape(profile, profile.DefaultRepetitions, false), nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || !profile.permitsRepetitions(value) {
		return DiagnosticExecutionShape{}, fmt.Errorf("%s is not permitted for frozen profile %s", DiagnosticRepetitionsEnv, profile.Identity)
	}
	return newDiagnosticExecutionShape(profile, value, true), nil
}

func ApplyDiagnosticExecutionShape(prepared PreparedDiagnostic, shape DiagnosticExecutionShape) (PreparedDiagnostic, error) {
	if err := validateDiagnosticExecutionShapeValue(shape, prepared.Profile); err != nil {
		return PreparedDiagnostic{}, err
	}
	prepared.ExecutionShape = shape
	prepared.Plan.ExecutionShape = shape
	prepared.Plan.Repetitions = shape.EffectiveRepetitions
	prepared.Plan.CasesPerRepetition = shape.CasesPerRepetition
	if prepared.Plan.HostedExperiment != nil {
		previousRepetitions := prepared.Plan.HostedExperiment.BaseRequestCount / prepared.Profile.CaseCount
		prepared.Plan.HostedExperiment.BaseRequestCount = shape.TotalPlannedCases
		prepared.Plan.HostedExperiment.MaximumRequestCount = shape.TotalPlannedCases * 2
		if estimate, err := parseUSDMicros(prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD); err == nil && previousRepetitions > 0 {
			prepared.Plan.HostedExperiment.EstimatedMaximumRunUSD = formatUSDMicros(estimate / int64(previousRepetitions) * int64(shape.EffectiveRepetitions))
		}
		if estimate, err := parseUSDMicros(prepared.Plan.HostedExperiment.EstimatedInitialRunUSD); err == nil && previousRepetitions > 0 {
			prepared.Plan.HostedExperiment.EstimatedInitialRunUSD = formatUSDMicros(estimate / int64(previousRepetitions) * int64(shape.EffectiveRepetitions))
		}
	}
	return prepared, nil
}

func ValidateDiagnosticExecutionShape(prepared PreparedDiagnostic) error {
	if err := validateDiagnosticExecutionShapeValue(prepared.ExecutionShape, prepared.Profile); err != nil {
		return err
	}
	shape := prepared.ExecutionShape
	if prepared.Plan.ExecutionShape != shape || prepared.Plan.Repetitions != shape.EffectiveRepetitions ||
		prepared.Plan.CasesPerRepetition != shape.CasesPerRepetition || prepared.Plan.EvaluationProfile != prepared.Profile.Identity ||
		len(prepared.Plan.Events) != prepared.Profile.CaseCount ||
		len(prepared.Manifest.Events) != prepared.Profile.CaseCount {
		return fmt.Errorf("issuer diagnostic execution shape does not match validated runtime selection")
	}
	contract, err := diagnosticExecutionContractForProfile(prepared.Profile)
	if err != nil {
		return err
	}
	prompt, output, policy := contract.Prompt, contract.Output, contract.Policy
	if prepared.Plan.PromptVersion != prompt || prepared.Plan.OutputContract != output ||
		prepared.Plan.ExecutionRoute != contract.Route || prepared.Plan.ValidatorVersion != contract.Validator ||
		(output == V5SchemaVersion && (prepared.Plan.CausalAttributionPolicy != policy || prepared.Plan.CausalConsistencyPolicy != "")) ||
		(output != V5SchemaVersion && prepared.Plan.CausalConsistencyPolicy != policy) {
		return fmt.Errorf("issuer diagnostic execution contract does not match the frozen profile")
	}
	if plan := prepared.Plan.HostedExperiment; plan != nil &&
		(plan.BaseRequestCount != shape.TotalPlannedCases || plan.MaximumRequestCount != shape.TotalPlannedCases*2) {
		return fmt.Errorf("hosted issuer diagnostic request plan does not match validated execution shape")
	}
	if plan := prepared.Plan.HostedExperiment; plan != nil && plan.ExperimentID == OpenAIStructuredOutputsExperimentID && shape.EffectiveRepetitions != 1 {
		return fmt.Errorf("%s requires exactly one diagnostic repetition", OpenAIStructuredOutputsExperimentID)
	}
	if prepared.Profile.RequiredExperimentID != "" && (prepared.Plan.HostedExperiment == nil || prepared.Plan.HostedExperiment.ExperimentID != prepared.Profile.RequiredExperimentID) {
		return fmt.Errorf("issuer diagnostic profile %s requires experiment %s", prepared.Profile.Identity, prepared.Profile.RequiredExperimentID)
	}
	if plan := prepared.Plan.HostedExperiment; plan != nil && prepared.Config.Provider == OpenAIDiagnosticProvider && (prepared.Config.Model == OpenAIDiagnosticLunaModel || prepared.Config.Model == OpenAIDiagnosticTerraModel) {
		estimate := plan.EstimatedMaximumRunUSD
		if prepared.Config.Model == OpenAIDiagnosticTerraModel {
			estimate = plan.EstimatedFirstRequestMaxUSD
		}
		estimated, estimatedErr := parseUSDMicros(estimate)
		ceiling, ceilingErr := parseUSDMicros(plan.BudgetCeilingUSD)
		if estimatedErr != nil || ceilingErr != nil || estimated > ceiling {
			if prepared.Config.Model == OpenAIDiagnosticTerraModel {
				return fmt.Errorf("configured hosted model budget ceiling cannot accommodate the frozen per-request maximum")
			}
			return fmt.Errorf("configured Luna budget ceiling cannot accommodate the conservative complete-run estimate")
		}
	}
	return nil
}

func newDiagnosticExecutionShape(profile DiagnosticEvaluationProfile, repetitions int, overrideSupplied bool) DiagnosticExecutionShape {
	return DiagnosticExecutionShape{
		RepetitionEnvironment: DiagnosticRepetitionsEnv,
		OverrideSupplied:      overrideSupplied,
		RequestedRepetitions:  repetitions,
		EffectiveRepetitions:  repetitions,
		CasesPerRepetition:    profile.CaseCount,
		TotalPlannedCases:     repetitions * profile.CaseCount,
	}
}

func validateDiagnosticExecutionShapeValue(shape DiagnosticExecutionShape, profile DiagnosticEvaluationProfile) error {
	if shape.RepetitionEnvironment != DiagnosticRepetitionsEnv ||
		shape.RequestedRepetitions != shape.EffectiveRepetitions ||
		!profile.permitsRepetitions(shape.EffectiveRepetitions) ||
		shape.CasesPerRepetition != profile.CaseCount ||
		shape.TotalPlannedCases != shape.EffectiveRepetitions*profile.CaseCount {
		return fmt.Errorf("issuer diagnostic execution shape is not permitted for frozen profile %s", profile.Identity)
	}
	if !shape.OverrideSupplied && shape.EffectiveRepetitions != profile.DefaultRepetitions {
		return fmt.Errorf("issuer diagnostic default execution shape for profile %s must remain %d repetitions", profile.Identity, profile.DefaultRepetitions)
	}
	return nil
}

func PrepareHostedDiagnostic(paths DiagnosticPaths, config OpenAIDiagnosticConfig, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	return prepareHostedDiagnostic(paths, config, safety, true)
}

func PrepareHostedDiagnosticPreflight(paths DiagnosticPaths, config OpenAIDiagnosticConfig, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	profileID := paths.EvaluationProfileID
	if profileID == "" {
		profileID = DiagnosticProfileOriginal
	}
	profile, err := LoadDiagnosticExecutionProfile(profileID)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	if !profile.CredentiallessPreflightAllowed || config.InferenceExplicitlyAuthorized || config.C1E3ExecutionAuthorization.OperatorOptIn || config.C1F3ExecutionAuthorization.OperatorOptIn || config.C1F3RepeatabilityExecutionAuthorization.OperatorOptIn || config.C1F3RepeatabilityR3ExecutionAuthorization.OperatorOptIn || config.C1F3TerraChallengerExecutionAuthorization.OperatorOptIn {
		return PreparedDiagnostic{}, fmt.Errorf("frozen profile %s does not permit this local preflight", profile.Identity)
	}
	return prepareHostedDiagnostic(paths, config, safety, false)
}

func prepareHostedDiagnostic(paths DiagnosticPaths, config OpenAIDiagnosticConfig, safety DiagnosticSafetyState, requireCredential bool) (PreparedDiagnostic, error) {
	if requireCredential && !config.APIKey.present() {
		return PreparedDiagnostic{}, fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !supportedOpenAIDiagnosticModel(config.Runtime.Model) && !(config.Runtime.Model == OpenAIDiagnosticTerraModel && config.ExperimentID == C1F3TerraChallengerExperimentID) {
		return PreparedDiagnostic{}, fmt.Errorf("unsupported OpenAI diagnostic model %q", config.Runtime.Model)
	}
	if err := validateOpenAIExperimentCell(config.ExperimentID, config.Runtime.Model, config.OutputContractMode); err != nil {
		return PreparedDiagnostic{}, err
	}
	wantNamespace := filepath.Join(config.EvidenceNamespace(), config.ExperimentID)
	if !strings.HasSuffix(filepath.Clean(paths.OutputRoot), wantNamespace) {
		return PreparedDiagnostic{}, fmt.Errorf("hosted diagnostic output root must end in isolated namespace %s", filepath.ToSlash(wantNamespace))
	}
	prepared, err := PrepareDiagnostic(paths, config.Runtime, safety)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	if prepared.Profile.RequiredProvider != "" && (prepared.Profile.RequiredProvider != config.Runtime.Provider ||
		prepared.Profile.RequiredModel != config.Runtime.Model || prepared.Profile.RequiredExperimentID != config.ExperimentID ||
		prepared.Profile.RequiredOutputContractMode != config.OutputContractMode || prepared.Profile.EvidenceNamespace != config.EvidenceNamespace()) {
		return PreparedDiagnostic{}, fmt.Errorf("hosted configuration does not match frozen profile %s", prepared.Profile.Identity)
	}
	if err := validateC1E3AuthorizationScope(prepared.Profile, config); err != nil {
		return PreparedDiagnostic{}, err
	}
	if err := validateC1F3AuthorizationScope(prepared.Profile, config); err != nil {
		return PreparedDiagnostic{}, err
	}
	if err := validateC1F3RepeatabilityAuthorizationScope(prepared.Profile, config); err != nil {
		return PreparedDiagnostic{}, err
	}
	if err := validateC1F3RepeatabilityR3AuthorizationScope(prepared.Profile, config); err != nil {
		return PreparedDiagnostic{}, err
	}
	if err := validateC1F3TerraChallengerAuthorizationScope(prepared.Profile, config); err != nil {
		return PreparedDiagnostic{}, err
	}
	collisionFree := true
	if isC1E3Profile(prepared.Profile) {
		collisionFree, err = c1e3EvidenceNamespaceCollisionFree(paths.OutputRoot)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("inspect C1E3 evidence namespace: %w", err)
		}
		if !collisionFree {
			return PreparedDiagnostic{}, fmt.Errorf("C1E3 evidence namespace already contains execution evidence")
		}
	}
	if isC1F3Profile(prepared.Profile) {
		collisionFree, err = c1f3EvidenceNamespaceCollisionFree(paths.OutputRoot)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("inspect C1F3 evidence namespace: %w", err)
		}
		if !collisionFree {
			return PreparedDiagnostic{}, fmt.Errorf("C1F3 evidence namespace already contains execution evidence")
		}
	}
	if isC1F3RepeatabilityProfile(prepared.Profile) {
		collisionFree, err = c1f3EvidenceNamespaceCollisionFree(paths.OutputRoot)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("inspect C1F3 repeatability evidence namespace: %w", err)
		}
		if !collisionFree {
			return PreparedDiagnostic{}, fmt.Errorf("C1F3 repeatability evidence namespace already contains execution evidence")
		}
	}
	if isC1F3TerraChallengerProfile(prepared.Profile) {
		collisionFree, err = c1f3EvidenceNamespaceCollisionFree(paths.OutputRoot)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("inspect Terra challenger evidence namespace: %w", err)
		}
		if !collisionFree {
			return PreparedDiagnostic{}, fmt.Errorf("Terra challenger evidence namespace already contains execution evidence")
		}
	}
	firstRequest, err := diagnosticInitialRequest(config, prepared.Manifest.Events[0].Input, prepared.ProxyExposures)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	firstWireBytes, err := openAIDiagnosticRequestBytes(config, firstRequest)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	estimatedInputTokens := estimatedOpenAIInputTokens(config, firstRequest, firstWireBytes)
	worstInputPrice := config.InputPriceMicrosPerMillion
	if config.CacheWritePriceMicrosPerMillion > worstInputPrice {
		worstInputPrice = config.CacheWritePriceMicrosPerMillion
	}
	estimatedFirstCost := tokenCostMicros(estimatedInputTokens, worstInputPrice) + tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	if estimatedFirstCost > config.BudgetCeilingMicros {
		return PreparedDiagnostic{}, fmt.Errorf("configured hosted request maximum exceeds the experiment budget ceiling")
	}
	prepared.Plan.ModelConfiguration.ReasoningEffort = config.ReasoningEffort
	prepared.Plan.ModelConfiguration.MaxOutputTokens = config.MaxOutputTokens
	prepared.Plan.ModelConfiguration.StructuredOutputMode = config.StructuredOutputMode()
	prepared.Plan.ModelConfiguration.StructuredOutputs = config.StructuredOutputsEnabled()
	prepared.Plan.ModelConfiguration.SchemaContract = prepared.Plan.OutputContract
	prepared.Plan.ModelConfiguration.ContractEnforcement = string(config.OutputContractMode)
	prepared.Plan.ModelConfiguration.ServiceTier = config.ServiceTier()
	schemaSHA256, err := providerRequestSchemaSHA256(firstRequest.Schema)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	largestInitialBytes, correctiveBytes, estimatedInitialRunCost, estimatedRunCost, err := estimateOpenAIDiagnosticRunMaximum(prepared, config)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	prepared.Plan.HostedExperiment = &HostedExperimentPlan{
		ExperimentID: config.ExperimentID, CellIdentity: config.ExperimentID, EvidenceNamespace: config.EvidenceNamespace() + "/" + config.ExperimentID,
		SchemaContract: prepared.Plan.OutputContract, SchemaSHA256: schemaSHA256, StructuredOutputs: config.StructuredOutputsEnabled(), ContractEnforcement: string(config.OutputContractMode),
		ServiceTier: config.ServiceTier(),
		Endpoint:    OpenAIDiagnosticEndpoint, APIKeyEnvironment: OpenAIDiagnosticAPIKeyEnv, APIKeyPresent: config.APIKey.present(),
		InferenceExplicitlyAuthorized: config.InferenceExplicitlyAuthorized,
		BudgetCeilingUSD:              formatUSDMicros(config.BudgetCeilingMicros),
		Pricing: HostedPricingPlan{
			InputUSDPerMillionTokens:       formatUSDMicros(config.InputPriceMicrosPerMillion),
			CachedInputUSDPerMillionTokens: formatUSDMicros(config.CachedInputPriceMicrosPerMillion),
			CacheWriteUSDPerMillionTokens:  formatUSDMicros(config.CacheWritePriceMicrosPerMillion),
			OutputUSDPerMillionTokens:      formatUSDMicros(config.OutputPriceMicrosPerMillion),
			Source:                         OpenAIDiagnosticPricingSource,
		},
		BaseRequestCount:                   prepared.ExecutionShape.TotalPlannedCases,
		MaximumRequestCount:                prepared.ExecutionShape.TotalPlannedCases * 2,
		EstimatedFirstRequestMaxUSD:        formatUSDMicros(estimatedFirstCost),
		EstimatedInitialRunUSD:             formatUSDMicros(estimatedInitialRunCost),
		LargestFrozenInitialRequestBytes:   largestInitialBytes,
		ConservativeCorrectiveRequestBytes: correctiveBytes,
		EstimatedMaximumRunUSD:             formatUSDMicros(estimatedRunCost),
		DatabaseMutationAllowed:            false, TradingStateMutationAllowed: false,
	}
	if isC1E3Profile(prepared.Profile) {
		budgetValid := estimatedRunCost <= config.BudgetCeilingMicros
		authorized := requireCredential && config.C1E3ExecutionAuthorization.OperatorOptIn && config.InferenceExplicitlyAuthorized && config.APIKey.present() && budgetValid && collisionFree
		prepared.Plan.C1E3ExecutionAuthorization = &C1E3ExecutionAuthorizationPlan{
			Version: C1E3ExecutionAuthorizationVersion, OperatorOptIn: config.C1E3ExecutionAuthorization.OperatorOptIn,
			HostedInferenceAuthorized: config.InferenceExplicitlyAuthorized, CredentialPresent: config.APIKey.present(),
			FrozenInputsValid: true, BudgetValid: budgetValid, EvidenceNamespaceCollisionFree: collisionFree,
			ExecutionAuthorized: authorized,
		}
		if requireCredential {
			if err := validateC1E3ExecutionAuthorization(prepared, config); err != nil {
				return PreparedDiagnostic{}, err
			}
		}
	}
	if isC1F3Profile(prepared.Profile) {
		isolationErr := validateC1F3ProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures)
		budgetValid := estimatedRunCost <= config.BudgetCeilingMicros
		authorized := requireCredential && config.C1F3ExecutionAuthorization.OperatorOptIn && config.InferenceExplicitlyAuthorized && config.APIKey.present() && budgetValid && collisionFree && isolationErr == nil
		prepared.Plan.C1F3ExecutionAuthorization = &C1F3ExecutionAuthorizationPlan{
			Version: C1F3ExecutionAuthorizationVersion, AuthorizationFingerprint: C1F3ExecutionAuthorizationFingerprint(),
			OperatorOptIn: config.C1F3ExecutionAuthorization.OperatorOptIn, HostedInferenceAuthorized: config.InferenceExplicitlyAuthorized,
			CredentialPresent: config.APIKey.present(), FrozenBindingsValid: prepared.Plan.C1F3FrozenBindings != nil,
			BudgetValid: budgetValid, EvidenceNamespaceCollisionFree: collisionFree, ProviderInputIsolated: isolationErr == nil,
			ExecutionAuthorized: authorized,
		}
		if isolationErr != nil {
			return PreparedDiagnostic{}, isolationErr
		}
		if requireCredential {
			if err := validateC1F3ExecutionAuthorization(prepared, config); err != nil {
				return PreparedDiagnostic{}, err
			}
		}
	}
	if isC1F3RepeatabilityR2Profile(prepared.Profile) {
		isolationErr := validateC1F3RepeatabilityProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures)
		budgetValid := estimatedRunCost <= config.BudgetCeilingMicros && config.BudgetCeilingMicros <= 300_000
		bindings := prepared.Plan.C1F3RepeatabilityFrozenBindings
		baselineValid := bindings != nil && bindings.Baseline == frozenC1F3RepeatabilityBaseline()
		scoringValid := bindings != nil && bindings.ComparisonScoring.Identity == C1F3RepeatabilityScoringVersion && bindings.ComparisonScoring.FileSHA256 == C1F3RepeatabilityScoringFileSHA256
		runtimeSafetyValid := prepared.Plan.Safety.RuntimeMode == "paper" && !prepared.Plan.Safety.AllowLiveTrading && !prepared.Plan.Safety.ExecutionEnabled && !prepared.Plan.Safety.ExecutionWorker && !prepared.Plan.Safety.BrokerExecution && prepared.Plan.Safety.MaximumLeverage > 0 && prepared.Plan.Safety.MaximumLeverage <= 1
		authorized := requireCredential && config.C1F3RepeatabilityExecutionAuthorization.OperatorOptIn && config.InferenceExplicitlyAuthorized && config.APIKey.present() && budgetValid && collisionFree && isolationErr == nil && baselineValid && scoringValid && runtimeSafetyValid
		prepared.Plan.C1F3RepeatabilityExecutionAuthorization = &C1F3RepeatabilityExecutionAuthorizationPlan{
			Version: C1F3RepeatabilityExecutionAuthorizationVersion, AuthorizationFingerprint: C1F3RepeatabilityExecutionAuthorizationFingerprint(),
			OperatorOptIn: config.C1F3RepeatabilityExecutionAuthorization.OperatorOptIn, HostedInferenceAuthorized: config.InferenceExplicitlyAuthorized,
			CredentialPresent: config.APIKey.present(), FrozenBindingsValid: bindings != nil, BaselineBindingValid: baselineValid,
			RepeatabilityScoringValid: scoringValid, BudgetValid: budgetValid, EvidenceNamespaceCollisionFree: collisionFree,
			ProviderInputIsolated: isolationErr == nil, ProviderInputMatchesC1F3: isolationErr == nil, RuntimeSafetyValid: runtimeSafetyValid,
			ExecutionAuthorized: authorized,
		}
		if isolationErr != nil {
			return PreparedDiagnostic{}, isolationErr
		}
		if requireCredential {
			if err := validateC1F3RepeatabilityExecutionAuthorization(prepared, config); err != nil {
				return PreparedDiagnostic{}, err
			}
		}
	}
	if isC1F3RepeatabilityR3Profile(prepared.Profile) {
		isolationErr := validateC1F3RepeatabilityProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures)
		budgetValid := estimatedRunCost <= config.BudgetCeilingMicros && config.BudgetCeilingMicros <= 300_000
		bindings := prepared.Plan.C1F3RepeatabilityFrozenBindings
		baselineValid := bindings != nil && bindings.Baseline == frozenC1F3RepeatabilityBaseline()
		scoringValid := bindings != nil && bindings.ComparisonScoring.Identity == C1F3RepeatabilityScoringVersion && bindings.ComparisonScoring.FileSHA256 == C1F3RepeatabilityScoringFileSHA256
		runtimeSafetyValid := prepared.Plan.Safety.RuntimeMode == "paper" && !prepared.Plan.Safety.AllowLiveTrading && !prepared.Plan.Safety.ExecutionEnabled && !prepared.Plan.Safety.ExecutionWorker && !prepared.Plan.Safety.BrokerExecution && prepared.Plan.Safety.MaximumLeverage > 0 && prepared.Plan.Safety.MaximumLeverage <= 1
		authorized := requireCredential && config.C1F3RepeatabilityR3ExecutionAuthorization.OperatorOptIn && config.InferenceExplicitlyAuthorized && config.APIKey.present() && budgetValid && collisionFree && isolationErr == nil && baselineValid && scoringValid && runtimeSafetyValid
		prepared.Plan.C1F3RepeatabilityR3ExecutionAuthorization = &C1F3RepeatabilityR3ExecutionAuthorizationPlan{
			Version: C1F3RepeatabilityR3ExecutionAuthorizationVersion, AuthorizationFingerprint: C1F3RepeatabilityR3ExecutionAuthorizationFingerprint(),
			OperatorOptIn: config.C1F3RepeatabilityR3ExecutionAuthorization.OperatorOptIn, HostedInferenceAuthorized: config.InferenceExplicitlyAuthorized,
			CredentialPresent: config.APIKey.present(), FrozenBindingsValid: bindings != nil, BaselineBindingValid: baselineValid,
			RepeatabilityScoringValid: scoringValid, BudgetValid: budgetValid, EvidenceNamespaceCollisionFree: collisionFree,
			ProviderInputIsolated: isolationErr == nil, ProviderInputMatchesC1F3: isolationErr == nil, R2ResponseIsolated: isolationErr == nil, RuntimeSafetyValid: runtimeSafetyValid,
			ExecutionAuthorized: authorized,
		}
		if isolationErr != nil {
			return PreparedDiagnostic{}, isolationErr
		}
		if requireCredential {
			if err := validateC1F3RepeatabilityR3ExecutionAuthorization(prepared, config); err != nil {
				return PreparedDiagnostic{}, err
			}
		}
	}
	if isC1F3TerraChallengerProfile(prepared.Profile) {
		isolationErr := validateC1F3TerraProviderInputIsolation(prepared.Manifest, config, prepared.ProxyExposures)
		budgetValid := estimatedFirstCost <= config.BudgetCeilingMicros && config.BudgetCeilingMicros <= 300_000
		bindings := prepared.Plan.C1F3TerraChallengerFrozenBindings
		lunaValid := bindings != nil && bindings.AcceptedLuna.RunID == C1F3TerraAcceptedLunaRunID && bindings.AcceptedLuna.ArtifactIndexSHA256 == C1F3TerraAcceptedLunaArtifactIndexSHA256 && bindings.AcceptedLuna.RawResponseCount == 48 && bindings.AcceptedLuna.EvidenceUntouched
		rubricValid := bindings != nil && bindings.DecisionRubric.Identity == C1F3TerraChallengerRubricVersion && bindings.DecisionRubric.FileSHA256 == C1F3TerraChallengerRubricFileSHA256 && bindings.DecisionRubric.SemanticFingerprint == C1F3TerraChallengerRubricFingerprint
		runtimeSafetyValid := prepared.Plan.Safety.RuntimeMode == "paper" && !prepared.Plan.Safety.AllowLiveTrading && !prepared.Plan.Safety.ExecutionEnabled && !prepared.Plan.Safety.ExecutionWorker && !prepared.Plan.Safety.BrokerExecution && prepared.Plan.Safety.MaximumLeverage > 0 && prepared.Plan.Safety.MaximumLeverage <= 1
		authorized := requireCredential && config.C1F3TerraChallengerExecutionAuthorization.OperatorOptIn && config.InferenceExplicitlyAuthorized && config.APIKey.present() && budgetValid && collisionFree && isolationErr == nil && lunaValid && rubricValid && runtimeSafetyValid
		prepared.Plan.C1F3TerraChallengerExecutionAuthorization = &C1F3TerraChallengerExecutionAuthorizationPlan{
			Version: C1F3TerraChallengerExecutionAuthorizationVersion, AuthorizationFingerprint: C1F3TerraChallengerExecutionAuthorizationFingerprint(),
			OperatorOptIn: config.C1F3TerraChallengerExecutionAuthorization.OperatorOptIn, HostedInferenceAuthorized: config.InferenceExplicitlyAuthorized,
			CredentialPresent: config.APIKey.present(), FrozenBindingsValid: bindings != nil, LunaPreservationValid: lunaValid,
			DecisionRubricValid: rubricValid, BudgetValid: budgetValid, EvidenceNamespaceCollisionFree: collisionFree,
			ProviderInputIsolated: isolationErr == nil, OnlyModelVariableChanged: isolationErr == nil, BoundaryExcluded: bindings != nil && bindings.BoundaryExcluded,
			RuntimeSafetyValid: runtimeSafetyValid, ExecutionAuthorized: authorized,
		}
		if isolationErr != nil {
			return PreparedDiagnostic{}, isolationErr
		}
		if requireCredential {
			if err := validateC1F3TerraChallengerExecutionAuthorization(prepared, config); err != nil {
				return PreparedDiagnostic{}, err
			}
		}
	}
	return prepared, nil
}

func diagnosticInitialRequest(config OpenAIDiagnosticConfig, input EventInput, proxyExposures []string) (ProviderRequest, error) {
	if config.PromptVersion == V6PromptVersion {
		return V6InitialRequest(input, proxyExposures)
	}
	if config.OutputContract == V5SchemaVersion {
		return V5InitialRequest(input, proxyExposures)
	}
	return InitialRequest(input, proxyExposures)
}

func diagnosticCorrectiveRequest(config OpenAIDiagnosticConfig, validationErrors []string, previous string, proxyExposures []string) (ProviderRequest, error) {
	if config.PromptVersion == V6PromptVersion {
		return V6CorrectiveRequest(validationErrors, previous, proxyExposures)
	}
	if config.OutputContract == V5SchemaVersion {
		return V5CorrectiveRequest(validationErrors, previous, proxyExposures)
	}
	return CorrectiveRequest(validationErrors, previous, proxyExposures)
}

func estimateOpenAIDiagnosticRunMaximum(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) (int, int, int64, int64, error) {
	worstInputPrice := config.InputPriceMicrosPerMillion
	if config.CacheWritePriceMicrosPerMillion > worstInputPrice {
		worstInputPrice = config.CacheWritePriceMicrosPerMillion
	}
	largestInitialBytes := 0
	perRepetition := int64(0)
	for _, event := range prepared.Manifest.Events {
		request, err := diagnosticInitialRequest(config, event.Input, prepared.ProxyExposures)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		requestBytes, err := openAIDiagnosticRequestBytes(config, request)
		if err != nil {
			return 0, 0, 0, 0, err
		}
		if requestBytes > largestInitialBytes {
			largestInitialBytes = requestBytes
		}
		perRepetition += tokenCostMicros(estimatedOpenAIInputTokens(config, request, requestBytes), worstInputPrice) +
			tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	}
	initialRunCost := perRepetition * int64(prepared.ExecutionShape.EffectiveRepetitions)
	// A corrective request can repeat the entire maximum output in both the
	// previous-response field and validation evidence. Four UTF-8 bytes per
	// output token in each field is deliberately conservative for the bounded
	// plain-text JSON contract while retaining the provider's 256-token cap.
	boundedOutputBytes := config.MaxOutputTokens * 4
	corrective, err := diagnosticCorrectiveRequest(config, []string{strings.Repeat("e", boundedOutputBytes)}, strings.Repeat("x", boundedOutputBytes), prepared.ProxyExposures)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	correctiveBytes, err := openAIDiagnosticRequestBytes(config, corrective)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	correctiveCost := tokenCostMicros(estimatedOpenAIInputTokens(config, corrective, correctiveBytes), worstInputPrice) +
		tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	perRepetition += int64(prepared.Profile.CaseCount) * correctiveCost
	return largestInitialBytes, correctiveBytes, initialRunCost, perRepetition * int64(prepared.ExecutionShape.EffectiveRepetitions), nil
}

func openAIDiagnosticRequestBytes(config OpenAIDiagnosticConfig, request ProviderRequest) (int, error) {
	if !config.StructuredOutputsEnabled() {
		return len([]byte(request.System)) + len([]byte(request.User)), nil
	}
	raw, err := marshalOpenAIDiagnosticRequest(config, request)
	if err != nil {
		return 0, err
	}
	return len(raw), nil
}

func providerRequestSchemaSHA256(schema map[string]any) (string, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("marshal canonical output schema: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func PrepareDeepSeekDiagnostic(paths DiagnosticPaths, config DeepSeekDiagnosticConfig, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	if !config.APIKey.present() {
		return PreparedDiagnostic{}, fmt.Errorf("missing required DeepSeek diagnostic configuration: %s", DeepSeekDiagnosticAPIKeyEnv)
	}
	wantNamespace := filepath.Join(DeepSeekDiagnosticEvidenceNamespace, config.ExperimentID)
	if !strings.HasSuffix(filepath.Clean(paths.OutputRoot), wantNamespace) {
		return PreparedDiagnostic{}, fmt.Errorf("DeepSeek diagnostic output root must end in isolated namespace %s", filepath.ToSlash(wantNamespace))
	}
	prepared, err := PrepareDiagnostic(paths, config.Runtime, safety)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	firstRequest, err := InitialRequest(prepared.Manifest.Events[0].Input, prepared.ProxyExposures)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	estimatedInputTokens := len([]byte(firstRequest.System)) + len([]byte(firstRequest.User)) + 1024
	estimatedFirstCost := tokenCostMicros(estimatedInputTokens, config.CacheMissPriceMicrosPerMillion) + tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	if estimatedFirstCost > config.BudgetCeilingMicros {
		return PreparedDiagnostic{}, fmt.Errorf("configured DeepSeek request maximum exceeds the experiment budget ceiling")
	}
	prepared.Plan.ModelConfiguration.ReasoningEffort = config.ReasoningEffort
	prepared.Plan.ModelConfiguration.ThinkingMode = config.ThinkingMode
	prepared.Plan.ModelConfiguration.MaxOutputTokens = config.MaxOutputTokens
	prepared.Plan.ModelConfiguration.StructuredOutputMode = DeepSeekDiagnosticStructuredOutput
	prepared.Plan.HostedExperiment = &HostedExperimentPlan{
		ExperimentID: config.ExperimentID, EvidenceNamespace: DeepSeekDiagnosticEvidenceNamespace + "/" + config.ExperimentID,
		Endpoint: DeepSeekDiagnosticEndpoint, APIKeyEnvironment: DeepSeekDiagnosticAPIKeyEnv, APIKeyPresent: true,
		InferenceExplicitlyAuthorized: config.InferenceExplicitlyAuthorized,
		BudgetCeilingUSD:              formatUSDMicros(config.BudgetCeilingMicros),
		Pricing: HostedPricingPlan{
			InputUSDPerMillionTokens:          formatUSDMicros(config.CacheMissPriceMicrosPerMillion),
			CachedInputUSDPerMillionTokens:    formatUSDMicros(config.CacheHitPriceMicrosPerMillion),
			CacheMissInputUSDPerMillionTokens: formatUSDMicros(config.CacheMissPriceMicrosPerMillion),
			OutputUSDPerMillionTokens:         formatUSDMicros(config.OutputPriceMicrosPerMillion),
			Source:                            "execution-time configuration; re-verify before paid execution",
		},
		BaseRequestCount:            prepared.ExecutionShape.TotalPlannedCases,
		MaximumRequestCount:         prepared.ExecutionShape.TotalPlannedCases * 2,
		EstimatedFirstRequestMaxUSD: formatUSDMicros(estimatedFirstCost),
		DatabaseMutationAllowed:     false, TradingStateMutationAllowed: false,
	}
	return prepared, nil
}

func WriteDiagnosticPreflight(prepared PreparedDiagnostic) (DiagnosticAuditPaths, string, error) {
	if err := ValidateDiagnosticExecutionShape(prepared); err != nil {
		return DiagnosticAuditPaths{}, "", err
	}
	if prepared.Plan.HostedExperiment != nil && prepared.Plan.HostedExperiment.InferenceExplicitlyAuthorized {
		return DiagnosticAuditPaths{}, "", fmt.Errorf("hosted diagnostic preflight requires inference authorization to remain false")
	}
	runID := uuid.NewString()
	dir := filepath.Join(prepared.Paths.OutputRoot, "preflight", runID)
	path := filepath.Join(dir, "preflight.json")
	payload := struct {
		RunID           string         `json:"run_id"`
		Status          string         `json:"status"`
		ProviderContact bool           `json:"provider_contact"`
		OllamaContact   bool           `json:"ollama_contact"`
		Inference       bool           `json:"inference"`
		Plan            DiagnosticPlan `json:"plan"`
	}{runID, "ready", false, false, false, prepared.Plan}
	hash, err := writeExclusiveJSON(path, payload)
	if err != nil {
		return DiagnosticAuditPaths{}, "", err
	}
	return DiagnosticAuditPaths{RunID: runID, Directory: dir, Preflight: path}, hash, nil
}

func ExecuteDiagnostic(prepared PreparedDiagnostic, provider Provider, identity DiagnosticModelIdentity) (DiagnosticRunReport, DiagnosticAuditPaths, error) {
	if err := ValidateDiagnosticExecutionShape(prepared); err != nil {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, err
	}
	executionContract, err := diagnosticExecutionContractForProfile(prepared.Profile)
	if err != nil {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, err
	}
	if provider == nil {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("issuer diagnostic provider is required")
	}
	if identity.Name != prepared.Config.Model || prepared.Config.Provider == "ollama" && strings.TrimSpace(identity.Digest) == "" {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("issuer diagnostic model identity does not match configured model")
	}
	if plan := prepared.Plan.HostedExperiment; plan != nil {
		recorder, ok := provider.(hostedExperimentRecorder)
		if !ok {
			return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("hosted issuer diagnostic provider must expose experiment accounting")
		}
		snapshot := recorder.ExperimentSnapshot()
		if snapshot.ExperimentID != plan.ExperimentID || snapshot.Provider != prepared.Config.Provider ||
			snapshot.RequestedModel != prepared.Config.Model || snapshot.ReasoningEffort != prepared.Plan.ModelConfiguration.ReasoningEffort ||
			snapshot.ThinkingMode != prepared.Plan.ModelConfiguration.ThinkingMode ||
			snapshot.StructuredOutputMode != prepared.Plan.ModelConfiguration.StructuredOutputMode ||
			snapshot.MaxOutputTokensPerRequest != prepared.Plan.ModelConfiguration.MaxOutputTokens ||
			snapshot.BudgetCeilingUSD != plan.BudgetCeilingUSD || snapshot.Pricing != plan.Pricing {
			return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("hosted issuer diagnostic provider does not match the preflight plan")
		}
	}

	runID := uuid.NewString()
	dir := filepath.Join(prepared.Paths.OutputRoot, runID)
	paths := DiagnosticAuditPaths{RunID: runID, Directory: dir, Plan: filepath.Join(dir, "plan.json")}
	planAudit := struct {
		RunID         string                  `json:"run_id"`
		StartedAt     time.Time               `json:"started_at"`
		ModelIdentity DiagnosticModelIdentity `json:"model_identity"`
		Plan          DiagnosticPlan          `json:"plan"`
	}{runID, time.Now().UTC(), identity, prepared.Plan}
	if _, err := writeExclusiveJSON(paths.Plan, planAudit); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if err := copyExclusive(prepared.Paths.ManifestPath, filepath.Join(dir, "manifest.json")); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if err := copyExclusive(prepared.Paths.FingerprintLockPath, filepath.Join(dir, "input-fingerprints.json")); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if prepared.Paths.FreezePath != "" {
		if err := copyExclusive(prepared.Paths.FreezePath, filepath.Join(dir, "freeze.json")); err != nil {
			return DiagnosticRunReport{}, paths, err
		}
	}

	repetitionCount := prepared.ExecutionShape.EffectiveRepetitions
	repetitions := make([]DiagnosticRepetitionReport, 0, repetitionCount)
	allRuns := make([][]DiagnosticCaseRun, 0, repetitionCount)
	for repetition := 1; repetition <= repetitionCount; repetition++ {
		caseRuns := make([]DiagnosticCaseRun, 0, prepared.Profile.CaseCount)
		for _, event := range prepared.Manifest.Events {
			inputFingerprint, _ := EventInputFingerprint(event.Input)
			var result EventResult
			var attempts []Attempt
			var traces []ProviderTrace
			var err error
			switch executionContract.Route {
			case diagnosticRouteC1F3, diagnosticRouteC1FRepeatabilityR2, diagnosticRouteC1FRepeatabilityR3, diagnosticRouteC1FTerraChallenger:
				result, attempts, traces, err = analyseC1FEvent(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
			case diagnosticRouteHistoricalC1EV5:
				result, attempts, traces, err = analyseV5Event(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
			case diagnosticRouteHistoricalV4:
				result, attempts, traces, err = analyseEvent(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
			default:
				err = fmt.Errorf("unsupported diagnostic execution route %q", executionContract.Route)
			}
			if err != nil {
				if recorder, ok := provider.(hostedExperimentRecorder); ok {
					paths.StopRecord = filepath.Join(dir, "stop.json")
					stop := struct {
						RunID      string                   `json:"run_id"`
						StoppedAt  time.Time                `json:"stopped_at"`
						StopReason string                   `json:"stop_reason"`
						Experiment HostedExperimentSnapshot `json:"experiment"`
					}{runID, time.Now().UTC(), err.Error(), recorder.ExperimentSnapshot()}
					_, _ = writeExclusiveJSON(paths.StopRecord, stop)
					paths.ArtifactIndex, paths.ArtifactIndexSHA256, _ = writeDiagnosticArtifactIndex(dir)
				}
				return DiagnosticRunReport{}, paths, err
			}
			if err = attachDiagnosticResultProjection(prepared.Profile, result, attempts); err != nil {
				return DiagnosticRunReport{}, paths, err
			}
			for index, attempt := range attempts {
				audit, err := buildDiagnosticAttemptAudit(prepared.Profile, runID, repetition, event, attempt, traces[index], prepared.Resolver)
				if err != nil {
					return DiagnosticRunReport{}, paths, err
				}
				attemptPath := filepath.Join(dir, fmt.Sprintf("repetition-%02d", repetition), fmt.Sprintf("%s-attempt-%02d.json", event.ID, attempt.AttemptNumber))
				if _, err := writeExclusiveJSON(attemptPath, audit); err != nil {
					return DiagnosticRunReport{}, paths, err
				}
			}
			caseRuns = append(caseRuns, DiagnosticCaseRun{CaseID: event.ID, Category: event.Category, InputFingerprint: inputFingerprint, Attempts: attempts, Traces: traces, Result: result})
		}
		allRuns = append(allRuns, caseRuns)
		repetitions = append(repetitions, EvaluateDiagnosticRepetition(repetition, prepared.Manifest, caseRuns, prepared.Resolver))
	}
	report := BuildDiagnosticRunReport(runID, prepared, identity, repetitions, allRuns)
	if recorder, ok := provider.(hostedExperimentRecorder); ok {
		snapshot := recorder.ExperimentSnapshot()
		report.HostedExperiment = &snapshot
	}
	paths.ReportJSON = filepath.Join(dir, "report.json")
	paths.ReportMarkdown = filepath.Join(dir, "report.md")
	if _, err := writeExclusiveJSON(paths.ReportJSON, report); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if _, err := writeExclusive(filepath.Join(dir, "report.md"), []byte(DiagnosticReportMarkdown(report))); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	indexPath, indexHash, err := writeDiagnosticArtifactIndex(dir)
	if err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	paths.ArtifactIndex, paths.ArtifactIndexSHA256 = indexPath, indexHash
	return report, paths, nil
}

type diagnosticProjectionRoute = diagnosticExecutionRoute

const (
	diagnosticProjectionV4                 diagnosticProjectionRoute = diagnosticRouteHistoricalV4
	diagnosticProjectionV5                 diagnosticProjectionRoute = diagnosticRouteHistoricalC1EV5
	diagnosticProjectionC1F                diagnosticProjectionRoute = diagnosticRouteC1F3
	diagnosticProjectionC1FRepeatabilityR2 diagnosticProjectionRoute = diagnosticRouteC1FRepeatabilityR2
	diagnosticProjectionC1FRepeatabilityR3 diagnosticProjectionRoute = diagnosticRouteC1FRepeatabilityR3
	diagnosticProjectionC1FTerraChallenger diagnosticProjectionRoute = diagnosticRouteC1FTerraChallenger
)

type diagnosticAttemptProjection struct {
	route             diagnosticProjectionRoute
	parsed            *StructuredResult
	v5Parsed          *V5StructuredResult
	typedAttribution  *TypedCausalAttribution
	causalGuard       *CausalConsistencyDecision
	causalAttribution *CausalAttributionDecision
	effectiveMapping  *AssetMapping
	resolution        *PolicyResolution
}

func newV4DiagnosticAttemptProjection(parsed *StructuredResult, guard *CausalConsistencyDecision, resolution *PolicyResolution) *diagnosticAttemptProjection {
	return &diagnosticAttemptProjection{route: diagnosticProjectionV4, parsed: parsed, causalGuard: guard, resolution: resolution}
}

func newV5DiagnosticAttemptProjection(parsed *V5StructuredResult, decision *CausalAttributionDecision, resolution *PolicyResolution) *diagnosticAttemptProjection {
	return newTypedDiagnosticAttemptProjection(diagnosticProjectionV5, parsed, decision, resolution)
}

func newC1FDiagnosticAttemptProjection(route diagnosticProjectionRoute, parsed *V5StructuredResult, decision *CausalAttributionDecision, resolution *PolicyResolution) *diagnosticAttemptProjection {
	return newTypedDiagnosticAttemptProjection(route, parsed, decision, resolution)
}

func newTypedDiagnosticAttemptProjection(route diagnosticProjectionRoute, parsed *V5StructuredResult, decision *CausalAttributionDecision, resolution *PolicyResolution) *diagnosticAttemptProjection {
	projection := &diagnosticAttemptProjection{route: route, v5Parsed: parsed, causalAttribution: decision, resolution: resolution}
	if parsed != nil {
		attribution := TypedAttributionFromV5(*parsed)
		projection.typedAttribution = &attribution
	}
	if decision != nil {
		effective := decision.EffectiveMapping
		projection.effectiveMapping = &effective
	}
	return projection
}

func attachDiagnosticResultProjection(profile DiagnosticEvaluationProfile, result EventResult, attempts []Attempt) error {
	if len(attempts) == 0 {
		return fmt.Errorf("diagnostic result has no attempt evidence")
	}
	final := &attempts[len(attempts)-1]
	if final.ValidationStatus != "accepted" {
		return nil
	}
	route, err := diagnosticProjectionRouteForProfile(profile)
	if err != nil {
		return err
	}
	switch route {
	case diagnosticProjectionV4:
		final.projection = newV4DiagnosticAttemptProjection(result.Parsed, result.CausalGuard, result.Resolution)
	case diagnosticProjectionV5:
		final.projection = newV5DiagnosticAttemptProjection(result.V5Parsed, result.CausalAttribution, result.Resolution)
	case diagnosticProjectionC1F, diagnosticProjectionC1FRepeatabilityR2, diagnosticProjectionC1FRepeatabilityR3, diagnosticProjectionC1FTerraChallenger:
		final.projection = newC1FDiagnosticAttemptProjection(route, result.V5Parsed, result.CausalAttribution, result.Resolution)
	default:
		return fmt.Errorf("unsupported diagnostic result projection route %q", route)
	}
	return nil
}

func diagnosticProjectionRouteForProfile(profile DiagnosticEvaluationProfile) (diagnosticProjectionRoute, error) {
	contract, err := diagnosticExecutionContractForProfile(profile)
	if err != nil {
		return "", fmt.Errorf("diagnostic evidence projection route: %w", err)
	}
	return contract.Route, nil
}

func buildDiagnosticAttemptAudit(profile DiagnosticEvaluationProfile, runID string, repetition int, event DiagnosticEvent, attempt Attempt, trace ProviderTrace, resolver assetresolution.Resolver) (DiagnosticAttemptAudit, error) {
	route, err := diagnosticProjectionRouteForProfile(profile)
	if err != nil {
		return DiagnosticAttemptAudit{}, err
	}
	contract, err := diagnosticExecutionContractForProfile(profile)
	if err != nil {
		return DiagnosticAttemptAudit{}, err
	}
	prompt, output := contract.Prompt, contract.Output
	if attempt.PromptVersion != prompt || attempt.SchemaVersion != output {
		return DiagnosticAttemptAudit{}, fmt.Errorf("diagnostic attempt route does not match frozen profile %q", profile.Identity)
	}
	if trace.AttemptNumber != 0 && trace.AttemptNumber != attempt.AttemptNumber {
		return DiagnosticAttemptAudit{}, fmt.Errorf("diagnostic attempt trace identity mismatch")
	}
	if rawHash(trace.Content) != attempt.RawResponseHash {
		return DiagnosticAttemptAudit{}, fmt.Errorf("diagnostic attempt raw response hash mismatch")
	}
	projection := attempt.projection
	if attempt.ValidationStatus == "accepted" {
		if projection == nil || projection.route != route {
			return DiagnosticAttemptAudit{}, fmt.Errorf("accepted diagnostic attempt has no projection for frozen route %q", route)
		}
		if route == diagnosticProjectionV4 && (projection.parsed == nil || projection.causalGuard == nil || projection.resolution == nil) {
			return DiagnosticAttemptAudit{}, fmt.Errorf("accepted historical v4 diagnostic projection is incomplete")
		}
		if route != diagnosticProjectionV4 && (projection.v5Parsed == nil || projection.typedAttribution == nil || projection.causalAttribution == nil || projection.effectiveMapping == nil || projection.resolution == nil) {
			return DiagnosticAttemptAudit{}, fmt.Errorf("accepted typed diagnostic projection is incomplete")
		}
	} else if projection != nil {
		return DiagnosticAttemptAudit{}, fmt.Errorf("rejected diagnostic attempt unexpectedly carries an accepted projection")
	}
	if projection == nil {
		projection = &diagnosticAttemptProjection{}
	}
	audit := DiagnosticAttemptAudit{
		RunID: runID, Repetition: repetition, CaseID: event.ID, Category: event.Category,
		AttemptNumber: attempt.AttemptNumber, InputFingerprint: attempt.InputFingerprint,
		Provider: attempt.Provider, ConfiguredModel: attempt.Model, ModelReportedIdentifier: attempt.ModelReportedIdentifier,
		PromptVersion: attempt.PromptVersion, OutputContract: attempt.SchemaVersion, PolicyVersion: resolver.Rules.Version,
		Seed: attempt.Seed, Temperature: attempt.Temperature, RequestTimestamp: attempt.RequestTimestamp,
		ResponseTimestamp: attempt.ResponseTimestamp, DurationMS: attempt.Duration.Milliseconds(),
		RawResponseHash: attempt.RawResponseHash, RawResponseBody: trace.Content,
		ValidationStatus: attempt.ValidationStatus, ValidationErrors: nonNilStrings(attempt.ValidationErrors), FailureReason: attempt.FailureReason,
		RequestID: trace.RequestID, ResponseID: trace.ResponseID, ProviderStatus: trace.Status,
		SystemFingerprint: trace.SystemFingerprint, FinishReason: trace.FinishReason, Usage: trace.Usage,
		ModelClassification: projection.parsed, V5RawModelOutput: projection.v5Parsed, TypedAttribution: projection.typedAttribution,
		CausalConsistencyGuard: projection.causalGuard, CausalAttributionPolicy: projection.causalAttribution,
		EffectiveSemanticMapping: projection.effectiveMapping, DeterministicResolution: projection.resolution,
	}
	return audit, nil
}

func writeExclusiveJSON(path string, value any) (string, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return writeExclusive(path, append(raw, '\n'))
}

func writeExclusive(path string, raw []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create append-only diagnostic audit %s: %w", path, err)
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func copyExclusive(source, destination string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	_, err = writeExclusive(destination, raw)
	return err
}

func writeDiagnosticArtifactIndex(dir string) (string, string, error) {
	type artifactHash struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	artifacts := []artifactHash{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "artifact-index.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifactHash{Path: filepath.ToSlash(relative), SHA256: hex.EncodeToString(digest[:])})
		return nil
	})
	if err != nil {
		return "", "", err
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	path := filepath.Join(dir, "artifact-index.json")
	hash, err := writeExclusiveJSON(path, struct {
		Version   string         `json:"version"`
		Artifacts []artifactHash `json:"artifacts"`
	}{"ai-shadow-issuer-diagnostic-artifact-index-v1", artifacts})
	return path, hash, err
}
