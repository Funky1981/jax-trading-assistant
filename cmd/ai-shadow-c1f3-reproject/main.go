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
	repositoryRoot := flag.String("repository-root", ".", "repository root containing the frozen C1F3 inputs and immutable execution evidence")
	outputRoot := flag.String("output-root", "", "append-only derived evidence root; defaults to the registered C1F3A namespace")
	flag.Parse()
	if strings.EqualFold(strings.TrimSpace(os.Getenv(aishadow.OpenAIDiagnosticInferenceAuthEnv)), "true") {
		fmt.Fprintln(os.Stderr, "C1F3 offline reprojection requires hosted inference authorization to be absent or false")
		os.Exit(1)
	}
	root, err := filepath.Abs(*repositoryRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	request := aishadow.DefaultC1F3ReprojectionRequest(root)
	if strings.TrimSpace(*outputRoot) != "" {
		request.OutputRoot, err = filepath.Abs(*outputRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	indexPath, indexSHA, evidence, err := aishadow.WriteC1F3Reprojection(request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result := struct {
		Version             string `json:"version"`
		ReprojectionSHA256  string `json:"reprojection_fingerprint"`
		ArtifactIndexPath   string `json:"artifact_index_path"`
		ArtifactIndexSHA256 string `json:"artifact_index_sha256"`
	}{evidence.Version, evidence.Fingerprint, indexPath, indexSHA}
	raw, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(raw))
}
