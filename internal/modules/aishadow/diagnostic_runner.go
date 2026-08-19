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
	Version                    string                          `json:"version"`
	EvaluationProfile          string                          `json:"evaluation_profile"`
	DatasetIdentity            string                          `json:"dataset_identity"`
	ManifestVersion            string                          `json:"manifest_version"`
	ManifestFingerprint        string                          `json:"manifest_fingerprint"`
	ManifestFileSHA256         string                          `json:"manifest_file_sha256"`
	FingerprintLockVersion     string                          `json:"fingerprint_lock_version"`
	FingerprintLockFingerprint string                          `json:"fingerprint_lock_fingerprint"`
	FingerprintLockFileSHA256  string                          `json:"fingerprint_lock_file_sha256"`
	FreezeVersion              string                          `json:"freeze_version,omitempty"`
	FreezeFileSHA256           string                          `json:"freeze_file_sha256,omitempty"`
	LabelVersion               string                          `json:"label_version"`
	PromptVersion              string                          `json:"prompt_version"`
	OutputContract             string                          `json:"output_contract"`
	PolicyVersion              string                          `json:"policy_version"`
	CausalConsistencyPolicy    string                          `json:"causal_consistency_policy"`
	CausalAttributionPolicy    string                          `json:"causal_attribution_policy,omitempty"`
	ScoringVersion             string                          `json:"scoring_version,omitempty"`
	TypedLabelVersion          string                          `json:"typed_label_version,omitempty"`
	TypedLabelFileSHA256       string                          `json:"typed_label_file_sha256,omitempty"`
	TypedLabelFingerprint      string                          `json:"typed_label_fingerprint,omitempty"`
	ScoringRubricVersion       string                          `json:"scoring_rubric_version,omitempty"`
	ScoringRubricFileSHA256    string                          `json:"scoring_rubric_file_sha256,omitempty"`
	ScoringRubricFingerprint   string                          `json:"scoring_rubric_fingerprint,omitempty"`
	C1E3ExecutionAuthorization *C1E3ExecutionAuthorizationPlan `json:"c1e3_execution_authorization,omitempty"`
	C1F3FrozenBindings         *C1F3FrozenBindingPlan          `json:"c1f3_frozen_bindings,omitempty"`
	C1F3ExecutionAuthorization *C1F3ExecutionAuthorizationPlan `json:"c1f3_execution_authorization,omitempty"`
	Repetitions                int                             `json:"repetitions"`
	CasesPerRepetition         int                             `json:"cases_per_repetition"`
	ExecutionShape             DiagnosticExecutionShape        `json:"execution_shape"`
	ModelConfiguration         DiagnosticModelConfiguration    `json:"model_configuration"`
	Safety                     DiagnosticSafetyState           `json:"safety"`
	HostedExperiment           *HostedExperimentPlan           `json:"hosted_experiment,omitempty"`
	Events                     []DiagnosticPlanEvent           `json:"events"`
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
	if profile.RequiresTypedAttributionLabels && !isC1F3Profile(profile) {
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
	if isC1F3Profile(profile) {
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
	if profile.isHoldout() && !isC1F3Profile(profile) {
		if strings.TrimSpace(paths.FreezePath) == "" {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic profile %s requires its registered freeze record", profile.Identity)
		}
		loaded, err := LoadDiagnosticFreezeRecord(profile, paths.FreezePath)
		if err != nil {
			return PreparedDiagnostic{}, err
		}
		freeze = &loaded
	} else if !isC1F3Profile(profile) && strings.TrimSpace(paths.FreezePath) != "" {
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
	executionPrompt, executionOutput, executionPolicy := profile.executionVersions()
	plan := DiagnosticPlan{
		Version: DiagnosticReportVersion, EvaluationProfile: profile.Identity, DatasetIdentity: profile.ManifestVersion, ManifestVersion: manifest.Version,
		ManifestFingerprint: manifest.Fingerprint, ManifestFileSHA256: profile.ManifestFileSHA256,
		FingerprintLockVersion: lock.Version, FingerprintLockFingerprint: lock.Fingerprint, FingerprintLockFileSHA256: profile.FingerprintLockFileSHA256,
		FreezeVersion: profile.FreezeVersion, FreezeFileSHA256: profile.FreezeFileSHA256,
		LabelVersion: manifest.LabelVersion, PromptVersion: executionPrompt, OutputContract: executionOutput,
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
	prompt, output, policy := prepared.Profile.executionVersions()
	if prepared.Plan.PromptVersion != prompt || prepared.Plan.OutputContract != output ||
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
	if plan := prepared.Plan.HostedExperiment; plan != nil && prepared.Config.Provider == OpenAIDiagnosticProvider && prepared.Config.Model == OpenAIDiagnosticLunaModel {
		estimated, estimatedErr := parseUSDMicros(plan.EstimatedMaximumRunUSD)
		ceiling, ceilingErr := parseUSDMicros(plan.BudgetCeilingUSD)
		if estimatedErr != nil || ceilingErr != nil || estimated > ceiling {
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
	if !profile.CredentiallessPreflightAllowed || config.InferenceExplicitlyAuthorized || config.C1E3ExecutionAuthorization.OperatorOptIn || config.C1F3ExecutionAuthorization.OperatorOptIn {
		return PreparedDiagnostic{}, fmt.Errorf("frozen profile %s does not permit this local preflight", profile.Identity)
	}
	return prepareHostedDiagnostic(paths, config, safety, false)
}

func prepareHostedDiagnostic(paths DiagnosticPaths, config OpenAIDiagnosticConfig, safety DiagnosticSafetyState, requireCredential bool) (PreparedDiagnostic, error) {
	if requireCredential && !config.APIKey.present() {
		return PreparedDiagnostic{}, fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	if !supportedOpenAIDiagnosticModel(config.Runtime.Model) {
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
	largestInitialBytes, correctiveBytes, estimatedRunCost, err := estimateOpenAIDiagnosticRunMaximum(prepared, config)
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

func estimateOpenAIDiagnosticRunMaximum(prepared PreparedDiagnostic, config OpenAIDiagnosticConfig) (int, int, int64, error) {
	worstInputPrice := config.InputPriceMicrosPerMillion
	if config.CacheWritePriceMicrosPerMillion > worstInputPrice {
		worstInputPrice = config.CacheWritePriceMicrosPerMillion
	}
	largestInitialBytes := 0
	perRepetition := int64(0)
	for _, event := range prepared.Manifest.Events {
		request, err := diagnosticInitialRequest(config, event.Input, prepared.ProxyExposures)
		if err != nil {
			return 0, 0, 0, err
		}
		requestBytes, err := openAIDiagnosticRequestBytes(config, request)
		if err != nil {
			return 0, 0, 0, err
		}
		if requestBytes > largestInitialBytes {
			largestInitialBytes = requestBytes
		}
		perRepetition += tokenCostMicros(estimatedOpenAIInputTokens(config, request, requestBytes), worstInputPrice) +
			tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	}
	// A corrective request can repeat the entire maximum output in both the
	// previous-response field and validation evidence. Four UTF-8 bytes per
	// output token in each field is deliberately conservative for the bounded
	// plain-text JSON contract while retaining the provider's 256-token cap.
	boundedOutputBytes := config.MaxOutputTokens * 4
	corrective, err := diagnosticCorrectiveRequest(config, []string{strings.Repeat("e", boundedOutputBytes)}, strings.Repeat("x", boundedOutputBytes), prepared.ProxyExposures)
	if err != nil {
		return 0, 0, 0, err
	}
	correctiveBytes, err := openAIDiagnosticRequestBytes(config, corrective)
	if err != nil {
		return 0, 0, 0, err
	}
	correctiveCost := tokenCostMicros(estimatedOpenAIInputTokens(config, corrective, correctiveBytes), worstInputPrice) +
		tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	perRepetition += int64(prepared.Profile.CaseCount) * correctiveCost
	return largestInitialBytes, correctiveBytes, perRepetition * int64(prepared.ExecutionShape.EffectiveRepetitions), nil
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
			if isC1F3Profile(prepared.Profile) {
				result, attempts, traces, err = analyseC1FEvent(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
			} else if prepared.Plan.OutputContract == V5SchemaVersion {
				result, attempts, traces, err = analyseV5Event(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
			} else {
				result, attempts, traces, err = analyseEvent(prepared.Config, provider, prepared.Resolver, runID, prepared.Manifest.Version, event.ID, inputFingerprint, event.Input, prepared.ProxyExposures)
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
			for index, attempt := range attempts {
				audit := buildDiagnosticAttemptAudit(runID, repetition, event, attempt, traces[index], prepared.Resolver)
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

func buildDiagnosticAttemptAudit(runID string, repetition int, event DiagnosticEvent, attempt Attempt, trace ProviderTrace, resolver assetresolution.Resolver) DiagnosticAttemptAudit {
	var parsed *StructuredResult
	var v5Parsed *V5StructuredResult
	var typedAttribution *TypedCausalAttribution
	var causalGuard *CausalConsistencyDecision
	var causalAttribution *CausalAttributionDecision
	var effectiveMapping *AssetMapping
	var resolution *PolicyResolution
	if attempt.ValidationStatus == "accepted" {
		if attempt.SchemaVersion == V5SchemaVersion {
			v5Parsed, causalAttribution, resolution, _ = ParseValidateAndApplyV5(trace.Content, event.Input, resolver)
			if v5Parsed != nil {
				attribution := TypedAttributionFromV5(*v5Parsed)
				typedAttribution = &attribution
			}
			if causalAttribution != nil {
				effective := causalAttribution.EffectiveMapping
				effectiveMapping = &effective
			}
		} else {
			parsed, causalGuard, resolution, _ = ParseValidateAndGuard(trace.Content, event.Input, resolver)
		}
	}
	return DiagnosticAttemptAudit{
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
		ModelClassification: parsed, V5RawModelOutput: v5Parsed, TypedAttribution: typedAttribution,
		CausalConsistencyGuard: causalGuard, CausalAttributionPolicy: causalAttribution,
		EffectiveSemanticMapping: effectiveMapping, DeterministicResolution: resolution,
	}
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
