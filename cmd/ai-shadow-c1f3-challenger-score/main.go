package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jax-trading-assistant/internal/modules/aishadow"
)

func main() {
	repositoryRoot := flag.String("repository-root", ".", "repository root containing frozen inputs and accepted Luna evidence")
	terraRun := flag.String("terra-run", "", "completed Terra t1 execution directory")
	terraArtifactIndex := flag.String("terra-artifact-index-sha256", "", "expected Terra t1 artifact-index SHA-256")
	output := flag.String("output", "", "exclusive JSON score output path")
	flag.Parse()
	if strings.TrimSpace(*terraRun) == "" || strings.TrimSpace(*terraArtifactIndex) == "" || strings.TrimSpace(*output) == "" {
		fmt.Fprintln(os.Stderr, "--terra-run, --terra-artifact-index-sha256, and --output are required")
		os.Exit(2)
	}
	if value := strings.TrimSpace(os.Getenv(aishadow.OpenAIDiagnosticInferenceAuthEnv)); value != "" && !strings.EqualFold(value, "false") {
		fmt.Fprintln(os.Stderr, "offline challenger scoring requires hosted inference authorization to be absent or false")
		os.Exit(2)
	}
	root, err := filepath.Abs(*repositoryRoot)
	if err != nil {
		panic(err)
	}
	score, err := aishadow.BuildC1F3TerraChallengerScore(root, *terraRun, strings.ToLower(strings.TrimSpace(*terraArtifactIndex)))
	if err != nil {
		panic(err)
	}
	raw, err := json.MarshalIndent(score, "", "  ")
	if err != nil {
		panic(err)
	}
	path, err := filepath.Abs(*output)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		panic(err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		panic(err)
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		_ = file.Close()
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	fmt.Println(path)
}
