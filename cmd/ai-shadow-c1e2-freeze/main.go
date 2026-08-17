package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/aishadow"
)

func main() {
	repoRoot := flag.String("repo-root", ".", "repository root")
	outputRoot := flag.String("output-root", ".runtime/ai-shadow", "offline evidence root")
	flag.Parse()
	if strings.TrimSpace(os.Getenv(aishadow.OpenAIDiagnosticInferenceAuthEnv)) != "false" {
		fmt.Fprintln(os.Stderr, aishadow.OpenAIDiagnosticInferenceAuthEnv+" must explicitly equal false")
		os.Exit(1)
	}
	path, hash, err := aishadow.GenerateC1E2OfflineFreeze(*repoRoot, *outputRoot, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("offline_preflight=%s\nsha256=%s\n", path, hash)
}
