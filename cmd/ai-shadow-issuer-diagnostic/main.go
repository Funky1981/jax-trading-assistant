package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"jax-trading-assistant/internal/modules/aishadow"
	"jax-trading-assistant/internal/modules/eventdecisions"
)

const (
	defaultDiagnosticManifestPath = "config/ai-shadow-issuer-diagnostic-manifest-v1.json"
	defaultFingerprintLockPath    = "config/ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"
	defaultAssetRulesetPath       = "config/event-asset-resolution-v1.json"
	defaultDiagnosticOutputRoot   = ".runtime/diagnostics/ai-shadow-issuer"
)

type dependencies struct {
	lookup       func(string) (string, bool)
	inspectModel func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error)
	provider     func(aishadow.Config) aishadow.Provider
}

func main() {
	deps := dependencies{
		lookup: os.LookupEnv, inspectModel: aishadow.InspectOllamaModel,
		provider: func(config aishadow.Config) aishadow.Provider { return aishadow.NewOllamaClient(config) },
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
	outputRoot := flags.String("output-root", defaultDiagnosticOutputRoot, "ignored append-only diagnostic audit root")
	preflight := flags.Bool("preflight", false, "perform all non-Ollama checks and write an audit artifact")
	execute := flags.Bool("execute", false, "execute exactly three complete repetitions")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *preflight == *execute {
		return fmt.Errorf("choose exactly one of --preflight or --execute")
	}
	config, err := aishadow.LoadConfig(deps.lookup)
	if err != nil {
		return err
	}
	safety, err := eventdecisions.ReadSafetyState(deps.lookup)
	if err != nil {
		return err
	}
	prepared, err := aishadow.PrepareDiagnostic(aishadow.DiagnosticPaths{
		ManifestPath: *manifestPath, FingerprintLockPath: *fingerprintLockPath,
		AssetRulesetPath: *assetRulesetPath, OutputRoot: *outputRoot,
	}, config, aishadow.DiagnosticSafetyState{
		RuntimeMode: safety.RuntimeMode, AllowLiveTrading: safety.AllowLiveTrading,
		ExecutionEnabled: safety.ExecutionEnabled, ExecutionWorker: safety.ExecutionWorker,
		BrokerExecution: safety.BrokerExecution, MaximumLeverage: safety.MaximumLeverage,
	})
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
			"status": "ready", "inference": false, "ollama_contact": false,
			"events": prepared.Plan.CasesPerRepetition, "repetitions": prepared.Plan.Repetitions,
			"manifest_fingerprint": prepared.Plan.ManifestFingerprint,
			"prompt_version":       prepared.Plan.PromptVersion, "output_contract": prepared.Plan.OutputContract,
			"policy_version": prepared.Plan.PolicyVersion, "audit": paths, "audit_sha256": hash,
		})
	}
	identity, err := deps.inspectModel(config)
	if err != nil {
		return err
	}
	report, paths, err := aishadow.ExecuteDiagnostic(prepared, deps.provider(config), identity)
	if err != nil {
		return err
	}
	return encoder.Encode(map[string]any{"status": "completed", "run_id": report.RunID, "repetitions": len(report.Repetitions), "artifacts": paths})
}
