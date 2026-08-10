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
	defaultDiagnosticManifestPath = "config/ai-shadow-issuer-diagnostic-manifest-v1.json"
	defaultFingerprintLockPath    = "config/ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"
	defaultAssetRulesetPath       = "config/event-asset-resolution-v1.json"
	defaultDiagnosticOutputRoot   = ".runtime/diagnostics/ai-shadow-issuer"
	defaultHostedOutputRoot       = ".runtime/diagnostics/ai-shadow-issuer-hosted"
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
	manifestPath := flags.String("manifest", defaultDiagnosticManifestPath, "frozen issuer diagnostic manifest")
	fingerprintLockPath := flags.String("fingerprint-lock", defaultFingerprintLockPath, "frozen per-event input fingerprint lock")
	assetRulesetPath := flags.String("asset-ruleset-file", defaultAssetRulesetPath, "deterministic issuer resolution policy")
	outputRoot := flags.String("output-root", "", "append-only diagnostic audit root (provider-specific default when omitted)")
	preflight := flags.Bool("preflight", false, "perform all non-Ollama checks and write an audit artifact")
	execute := flags.Bool("execute", false, "execute the validated diagnostic repetition selection (default 3; explicit 1 allowed)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *preflight == *execute {
		return fmt.Errorf("choose exactly one of --preflight or --execute")
	}
	executionShape, err := aishadow.LoadDiagnosticRepetitionSelection(deps.lookup)
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
	root := *outputRoot
	var prepared aishadow.PreparedDiagnostic
	var config aishadow.Config
	var hostedConfig aishadow.OpenAIDiagnosticConfig
	var deepSeekConfig aishadow.DeepSeekDiagnosticConfig
	paths := aishadow.DiagnosticPaths{
		ManifestPath: *manifestPath, FingerprintLockPath: *fingerprintLockPath,
		AssetRulesetPath: *assetRulesetPath,
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
		hostedConfig, err = aishadow.LoadOpenAIDiagnosticConfig(deps.lookup)
		if root == "" {
			root = defaultHostedOutputRoot
		}
		paths.OutputRoot = filepath.Join(root, aishadow.OpenAIDiagnosticEvidenceNamespace, aishadow.OpenAIDiagnosticExperimentID)
		if err == nil {
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
		return encoder.Encode(map[string]any{
			"status": "ready", "inference": false, "provider_contact": false, "ollama_contact": false,
			"provider": prepared.Plan.ModelConfiguration.Provider, "model": prepared.Plan.ModelConfiguration.Model,
			"events": prepared.Plan.CasesPerRepetition, "repetitions": prepared.Plan.Repetitions,
			"requested_repetitions": prepared.Plan.ExecutionShape.RequestedRepetitions,
			"effective_repetitions": prepared.Plan.ExecutionShape.EffectiveRepetitions,
			"total_planned_cases":   prepared.Plan.ExecutionShape.TotalPlannedCases,
			"manifest_fingerprint":  prepared.Plan.ManifestFingerprint,
			"prompt_version":        prepared.Plan.PromptVersion, "output_contract": prepared.Plan.OutputContract,
			"policy_version": prepared.Plan.PolicyVersion, "audit": paths, "audit_sha256": hash,
		})
	}
	var identity aishadow.DiagnosticModelIdentity
	var provider aishadow.Provider
	if providerName == aishadow.OpenAIDiagnosticProvider {
		if !hostedConfig.InferenceExplicitlyAuthorized {
			return fmt.Errorf("hosted inference is not authorized: %s must be true under separate architecture approval", aishadow.OpenAIDiagnosticInferenceAuthEnv)
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
