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
	lookup         func(string) (string, bool)
	inspectModel   func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error)
	ollamaProvider func(aishadow.Config) aishadow.Provider
	hostedProvider func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider
}

func main() {
	deps := dependencies{
		lookup: os.LookupEnv, inspectModel: aishadow.InspectOllamaModel,
		ollamaProvider: func(config aishadow.Config) aishadow.Provider { return aishadow.NewOllamaClient(config) },
		hostedProvider: func(config aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			return aishadow.NewOpenAIDiagnosticClient(config, nil)
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
	execute := flags.Bool("execute", false, "execute exactly three complete repetitions")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *preflight == *execute {
		return fmt.Errorf("choose exactly one of --preflight or --execute")
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
	default:
		return fmt.Errorf("unsupported diagnostic provider %q", providerName)
	}
	if err != nil {
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
			"manifest_fingerprint": prepared.Plan.ManifestFingerprint,
			"prompt_version":       prepared.Plan.PromptVersion, "output_contract": prepared.Plan.OutputContract,
			"policy_version": prepared.Plan.PolicyVersion, "audit": paths, "audit_sha256": hash,
		})
	}
	var identity aishadow.DiagnosticModelIdentity
	var provider aishadow.Provider
	if providerName == aishadow.OpenAIDiagnosticProvider {
		if !hostedConfig.InferenceExplicitlyAuthorized {
			return fmt.Errorf("hosted inference is not authorized: set %s=true only after WP-00.03C approval", aishadow.OpenAIDiagnosticInferenceAuthEnv)
		}
		identity = aishadow.DiagnosticModelIdentity{Name: config.Model}
		provider = deps.hostedProvider(hostedConfig)
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
