package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jax-trading-assistant/internal/modules/aishadow"
	"jax-trading-assistant/internal/modules/assetresolution"
)

func main() {
	manifestPath := flag.String("manifest", "config/ai-shadow-issuer-diagnostic-manifest-v1.json", "original frozen 48-case diagnostic manifest")
	rulesPath := flag.String("asset-rules", "config/event-asset-resolution-v1.json", "deterministic asset rules")
	runRoot := flag.String("run-root", ".runtime/diagnostics/ai-shadow-issuer-hosted/openai-hosted-c1b-structured-outputs-v1/WP-00.03C1B", "accepted C1B run root")
	outputPath := flag.String("output", ".runtime/diagnostics/ai-shadow-issuer-guard-replay-c1d1.json", "offline replay report path")
	flag.Parse()
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("JAX_AI_HOSTED_INFERENCE_AUTHORIZED")), "false") {
		fatalf("offline replay requires JAX_AI_HOSTED_INFERENCE_AUTHORIZED=false")
	}
	rules, err := assetresolution.LoadRuleset(*rulesPath)
	if err != nil {
		fatalf("load asset rules: %v", err)
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		fatalf("load proxy exposures: %v", err)
	}
	manifest, err := aishadow.LoadFrozenDiagnosticManifest(*manifestPath, exposures)
	if err != nil {
		fatalf("load original frozen manifest: %v", err)
	}
	reports := make([]aishadow.DiagnosticRunReport, 0, len(aishadow.AcceptedC1BRunIDs))
	for _, runID := range aishadow.AcceptedC1BRunIDs {
		raw, readErr := os.ReadFile(filepath.Join(*runRoot, runID, "report.json"))
		if readErr != nil {
			fatalf("read accepted run %s: %v", runID, readErr)
		}
		var report aishadow.DiagnosticRunReport
		if err := json.Unmarshal(raw, &report); err != nil {
			fatalf("decode accepted run %s: %v", runID, err)
		}
		reports = append(reports, report)
	}
	replay, err := aishadow.BuildCausalConsistencyReplay(manifest, reports, resolver)
	if err != nil {
		fatalf("build offline replay: %v", err)
	}
	raw, err := json.MarshalIndent(replay, "", "  ")
	if err != nil {
		fatalf("encode offline replay: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fatalf("create output directory: %v", err)
	}
	if err := os.WriteFile(*outputPath, append(raw, '\n'), 0o644); err != nil {
		fatalf("write offline replay: %v", err)
	}
	fmt.Println(*outputPath)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
