package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandDefaultsPreserveExistingBenchmarkAndPolicy(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, path := range []string{defaultManifestPath, defaultAssetRulesetPath} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatalf("default command input %s is unavailable: %v", path, err)
		}
	}
	if strings.Contains(defaultManifestPath, "issuer-diagnostic") {
		t.Fatal("48-event diagnostic must not be operationally wired as the command default")
	}
}
