package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"jax-trading-assistant/internal/modules/aishadow"
	"jax-trading-assistant/internal/modules/eventdecisions"
)

const (
	defaultAssetRulesetPath     = "config/event-asset-resolution-v1.json"
	defaultDiagnosticOutputRoot = ".runtime/diagnostics/ai-shadow-issuer"
	defaultHostedOutputRoot     = ".runtime/diagnostics/ai-shadow-issuer-hosted"
)

type dependencies struct {
	lookup           func(string) (string, bool)
	inspectModel     func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error)
	ollamaProvider   func(aishadow.Config) aishadow.Provider
	openAIProvider   func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider
	deepSeekProvider func(aishadow.DeepSeekDiagnosticConfig) aishadow.Provider
}

func main() {
	deps := dependencies{
		lookup: os.LookupEnv, inspectModel: aishadow.InspectOllamaModel,
		ollamaProvider: func(config aishadow.Config) aishadow.Provider { return aishadow.NewOllamaClient(config) },
		openAIProvider: func(config aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			return aishadow.NewOpenAIDiagnosticClient(config, nil)
		},
		deepSeekProvider: func(config aishadow.DeepSeekDiagnosticConfig) aishadow.Provider {
			return aishadow.NewDeepSeekDiagnosticClient(config, nil)
		},
	}
	if err := run(os.Args[1:], os.Stdout, deps); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer, deps dependencies) error {
	flags := flag.NewFlagSet("ai-shadow-issuer-diagnostic", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	evaluationProfileID := flags.String("evaluation-profile", aishadow.DiagnosticProfileOriginal, "registered frozen issuer diagnostic evaluation profile")
	manifestPath := flags.String("manifest", "", "frozen issuer diagnostic manifest (registered profile default when omitted)")
	fingerprintLockPath := flags.String("fingerprint-lock", "", "frozen per-event input fingerprint lock (registered profile default when omitted)")
	freezePath := flags.String("freeze", "", "frozen evaluation metadata (registered holdout profile default when omitted)")
	typedLabelPath := flags.String("typed-labels", "", "frozen typed causal-attribution labels (registered C1E3 profile default when omitted)")
	scoringRubricPath := flags.String("scoring-rubric", "", "frozen typed scoring rubric (registered C1E3 profile default when omitted)")
	assetRulesetPath := flags.String("asset-ruleset-file", defaultAssetRulesetPath, "deterministic issuer resolution policy")
	outputRoot := flags.String("output-root", "", "append-only diagnostic audit root (provider-specific default when omitted)")
	preflight := flags.Bool("preflight", false, "perform all non-Ollama checks and write an audit artifact")
	execute := flags.Bool("execute", false, "execute the validated diagnostic repetition selection (default 3; explicit 1 allowed)")
	authorizeC1E3Execution := flags.Bool("authorize-c1e3-execution", false, "explicitly authorize provider construction for a registered frozen C1E3 profile")
	authorizeC1F3Execution := flags.Bool("authorize-c1f3-execution", false, "explicitly authorize provider construction for one of the two registered frozen C1F3 profiles")
	authorizeC1F3Repeatability := flags.Bool("authorize-c1f3-repeatability", false, "explicitly authorize provider construction only for the frozen C1F3 Generalization repeatability cell")
	authorizeC1F3RepeatabilityR3 := flags.Bool("authorize-c1f3-repeatability-r3", false, "explicitly authorize provider construction only for the frozen C1F3 Generalization r3 replacement repeatability cell")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *preflight == *execute {
		return fmt.Errorf("choose exactly one of --preflight or --execute")
	}
	profile, err := aishadow.LoadDiagnosticExecutionProfile(*evaluationProfileID)
	if err != nil {
		return err
	}
	if *authorizeC1E3Execution && !*execute {
		return fmt.Errorf("--authorize-c1e3-execution is valid only with --execute")
	}
	if *authorizeC1F3Execution && !*execute {
		return fmt.Errorf("--authorize-c1f3-execution is valid only with --execute")
	}
	if *authorizeC1F3Repeatability && !*execute {
		return fmt.Errorf("--authorize-c1f3-repeatability is valid only with --execute")
	}
	if *authorizeC1F3RepeatabilityR3 && !*execute {
		return fmt.Errorf("--authorize-c1f3-repeatability-r3 is valid only with --execute")
	}
	if *manifestPath == "" {
		*manifestPath = profile.ManifestPath
	}
	if *fingerprintLockPath == "" {
		*fingerprintLockPath = profile.FingerprintLockPath
	}
	if *freezePath == "" {
		*freezePath = profile.FreezePath
	}
	if *typedLabelPath == "" {
		*typedLabelPath = profile.TypedLabelPath
	}
	if *scoringRubricPath == "" {
		*scoringRubricPath = profile.ScoringRubricPath
	}
	executionShape, err := aishadow.LoadDiagnosticRepetitionSelectionForProfile(deps.lookup, profile)
	if err != nil {
		return err
	}
	safety, err := eventdecisions.ReadSafetyState(deps.lookup)
	if err != nil {
		return err
	}
	providerValue, ok := deps.lookup("JAX_AI_PROVIDER")
	if !ok || strings.TrimSpace(providerValue) == "" {
		return fmt.Errorf("missing required AI shadow configuration: JAX_AI_PROVIDER")
	}
	providerName := strings.ToLower(strings.TrimSpace(providerValue))
	if profile.RequiredProvider != "" && providerName != profile.RequiredProvider {
		return fmt.Errorf("frozen profile %s requires provider %s", profile.Identity, profile.RequiredProvider)
	}
	root := *outputRoot
	var prepared aishadow.PreparedDiagnostic
	var config aishadow.Config
	var hostedConfig aishadow.OpenAIDiagnosticConfig
	var deepSeekConfig aishadow.DeepSeekDiagnosticConfig
	paths := aishadow.DiagnosticPaths{
		EvaluationProfileID: profile.Identity,
		ManifestPath:        *manifestPath, FingerprintLockPath: *fingerprintLockPath,
		FreezePath: *freezePath, AssetRulesetPath: *assetRulesetPath,
		TypedLabelPath: *typedLabelPath, ScoringRubricPath: *scoringRubricPath,
	}
	diagnosticSafety := aishadow.DiagnosticSafetyState{
		RuntimeMode: safety.RuntimeMode, AllowLiveTrading: safety.AllowLiveTrading,
		ExecutionEnabled: safety.ExecutionEnabled, ExecutionWorker: safety.ExecutionWorker,
		BrokerExecution: safety.BrokerExecution, MaximumLeverage: safety.MaximumLeverage,
	}
	switch providerName {
	case "ollama":
		config, err = aishadow.LoadConfig(deps.lookup)
		if root == "" {
			root = defaultDiagnosticOutputRoot
		}
		paths.OutputRoot = root
		if err == nil {
			prepared, err = aishadow.PrepareDiagnostic(paths, config, diagnosticSafety)
		}
	case aishadow.OpenAIDiagnosticProvider:
		requireCredential := !*preflight || !profile.CredentiallessPreflightAllowed
		hostedConfig, err = aishadow.LoadOpenAIDiagnosticConfigForProfile(deps.lookup, profile, requireCredential)
		if err == nil {
			hostedConfig.C1E3ExecutionAuthorization = aishadow.NewC1E3ExecutionAuthorization(*authorizeC1E3Execution)
			hostedConfig.C1F3ExecutionAuthorization = aishadow.NewC1F3ExecutionAuthorization(*authorizeC1F3Execution)
			hostedConfig.C1F3RepeatabilityExecutionAuthorization = aishadow.NewC1F3RepeatabilityExecutionAuthorization(*authorizeC1F3Repeatability)
			hostedConfig.C1F3RepeatabilityR3ExecutionAuthorization = aishadow.NewC1F3RepeatabilityR3ExecutionAuthorization(*authorizeC1F3RepeatabilityR3)
		}
		if root == "" {
			root = defaultHostedOutputRoot
		}
		if err == nil {
			paths.OutputRoot = filepath.Join(root, hostedConfig.EvidenceNamespace(), hostedConfig.ExperimentID)
		}
		if err == nil && *preflight && profile.CredentiallessPreflightAllowed {
			config = hostedConfig.Runtime
			prepared, err = aishadow.PrepareHostedDiagnosticPreflight(paths, hostedConfig, diagnosticSafety)
		} else if err == nil {
			config = hostedConfig.Runtime
			prepared, err = aishadow.PrepareHostedDiagnostic(paths, hostedConfig, diagnosticSafety)
		}
	case aishadow.DeepSeekDiagnosticProvider:
		deepSeekConfig, err = aishadow.LoadDeepSeekDiagnosticConfig(deps.lookup)
		if root == "" {
			root = defaultHostedOutputRoot
		}
		paths.OutputRoot = filepath.Join(root, aishadow.DeepSeekDiagnosticEvidenceNamespace, aishadow.DeepSeekDiagnosticExperimentID)
		if err == nil {
			config = deepSeekConfig.Runtime
			prepared, err = aishadow.PrepareDeepSeekDiagnostic(paths, deepSeekConfig, diagnosticSafety)
		}
	default:
		return fmt.Errorf("unsupported diagnostic provider %q", providerName)
	}
	if err != nil {
		return err
	}
	prepared, err = aishadow.ApplyDiagnosticExecutionShape(prepared, executionShape)
	if err != nil {
		return err
	}
	if err := aishadow.ValidateDiagnosticExecutionShape(prepared); err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if *preflight {
		paths, hash, err := aishadow.WriteDiagnosticPreflight(prepared)
		if err != nil {
			return err
		}
		result := map[string]any{
			"status": "ready", "inference": false, "provider_contact": false, "ollama_contact": false,
			"evaluation_profile": prepared.Plan.EvaluationProfile, "dataset_identity": prepared.Plan.DatasetIdentity,
			"causal_consistency_policy": prepared.Plan.CausalConsistencyPolicy,
			"provider":                  prepared.Plan.ModelConfiguration.Provider, "model": prepared.Plan.ModelConfiguration.Model,
			"requested_model": prepared.Plan.ModelConfiguration.Model, "reasoning": prepared.Plan.ModelConfiguration.ReasoningEffort,
			"events": prepared.Plan.CasesPerRepetition, "cases_per_repetition": prepared.Plan.CasesPerRepetition, "repetitions": prepared.Plan.Repetitions,
			"requested_repetitions":        prepared.Plan.ExecutionShape.RequestedRepetitions,
			"effective_repetitions":        prepared.Plan.ExecutionShape.EffectiveRepetitions,
			"total_planned_cases":          prepared.Plan.ExecutionShape.TotalPlannedCases,
			"manifest_fingerprint":         prepared.Plan.ManifestFingerprint,
			"manifest_file_sha256":         prepared.Plan.ManifestFileSHA256,
			"fingerprint_lock_version":     prepared.Plan.FingerprintLockVersion,
			"fingerprint_lock_fingerprint": prepared.Plan.FingerprintLockFingerprint,
			"prompt_version":               prepared.Plan.PromptVersion, "output_contract": prepared.Plan.OutputContract,
			"policy_version":    prepared.Plan.PolicyVersion,
			"database_mutation": false, "trading_mutation": false,
			"audit": paths, "audit_sha256": hash,
		}
		if hosted := prepared.Plan.HostedExperiment; hosted != nil {
			result["api_key_present"] = hosted.APIKeyPresent
			result["experiment_id"] = hosted.ExperimentID
			result["cell_identity"] = hosted.CellIdentity
			result["evidence_namespace"] = hosted.EvidenceNamespace
			result["structured_outputs"] = hosted.StructuredOutputs
			result["schema_contract"] = hosted.SchemaContract
			result["schema_sha256"] = hosted.SchemaSHA256
			result["contract_enforcement"] = hosted.ContractEnforcement
			if hosted.ServiceTier != "" {
				result["service_tier"] = hosted.ServiceTier
			}
			result["budget_configuration"] = map[string]any{
				"ceiling_usd": hosted.BudgetCeilingUSD, "pricing": hosted.Pricing,
				"estimated_maximum_run_usd": hosted.EstimatedMaximumRunUSD,
			}
			if authorization := prepared.Plan.C1E3ExecutionAuthorization; authorization != nil {
				result["c1e3_execution_authorization"] = authorization
				result["execution_authorized"] = authorization.ExecutionAuthorized
			}
			if authorization := prepared.Plan.C1F3ExecutionAuthorization; authorization != nil {
				result["c1f3_execution_authorization"] = authorization
				result["execution_authorized"] = authorization.ExecutionAuthorized
			}
			if authorization := prepared.Plan.C1F3RepeatabilityExecutionAuthorization; authorization != nil {
				result["c1f3_repeatability_execution_authorization"] = authorization
				result["execution_authorized"] = authorization.ExecutionAuthorized
				result["repeatability_frozen_bindings"] = prepared.Plan.C1F3RepeatabilityFrozenBindings
			}
			if authorization := prepared.Plan.C1F3RepeatabilityR3ExecutionAuthorization; authorization != nil {
				result["c1f3_repeatability_r3_execution_authorization"] = authorization
				result["execution_authorized"] = authorization.ExecutionAuthorized
				result["repeatability_frozen_bindings"] = prepared.Plan.C1F3RepeatabilityFrozenBindings
				result["execution_route"] = prepared.Plan.ExecutionRoute
				result["validator_version"] = prepared.Plan.ValidatorVersion
			}
		}
		return encoder.Encode(result)
	}
	var identity aishadow.DiagnosticModelIdentity
	var provider aishadow.Provider
	if providerName == aishadow.OpenAIDiagnosticProvider {
		if !hostedConfig.InferenceExplicitlyAuthorized {
			return fmt.Errorf("hosted inference is not authorized: %s must be true under separate architecture approval", aishadow.OpenAIDiagnosticInferenceAuthEnv)
		}
		if err := aishadow.RevalidateOpenAIProviderConstruction(prepared, hostedConfig); err != nil {
			return err
		}
		identity = aishadow.DiagnosticModelIdentity{Name: config.Model}
		provider = deps.openAIProvider(hostedConfig)
	} else if providerName == aishadow.DeepSeekDiagnosticProvider {
		if !deepSeekConfig.InferenceExplicitlyAuthorized {
			return fmt.Errorf("hosted inference is not authorized: %s must be true under separate architecture approval", aishadow.OpenAIDiagnosticInferenceAuthEnv)
		}
		identity = aishadow.DiagnosticModelIdentity{Name: config.Model}
		provider = deps.deepSeekProvider(deepSeekConfig)
	} else {
		identity, err = deps.inspectModel(config)
		if err != nil {
			return err
		}
		provider = deps.ollamaProvider(config)
	}
	report, auditPaths, err := aishadow.ExecuteDiagnostic(prepared, provider, identity)
	if err != nil {
		return err
	}
	return encoder.Encode(map[string]any{"status": "completed", "run_id": report.RunID, "repetitions": len(report.Repetitions), "artifacts": auditPaths})
}
