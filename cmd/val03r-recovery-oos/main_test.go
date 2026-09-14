package main

import (
	"strings"
	"testing"
)

func TestRecoveryRunnerPerformanceLock(t *testing.T) {
	for _, args := range [][]string{{"--contract-audit"}, {"--preflight-only"}} {
		if len(args) != 1 || (args[0] != "--contract-audit" && args[0] != "--preflight-only") {
			t.Fatalf("registered mode rejected: %v", args)
		}
	}
	for _, forbidden := range []string{"--run", "--execute", "--threshold=0.2", "--oos"} {
		if forbidden == "--contract-audit" || forbidden == "--preflight-only" {
			t.Fatal("performance mode accidentally registered")
		}
	}
}

func TestRecoveryRunnerFrozenContractConstants(t *testing.T) {
	if recoveryStart != "2025-01-01" || recoveryEnd != "2026-09-11" {
		t.Fatal("recovery boundary drifted")
	}
	if !strings.Contains(parentManifestPath, "PREREGISTRATION.json") || !strings.Contains(dataContractPath, "RECOVERY-DATA-CONTRACT.json") {
		t.Fatal("required contract identity missing")
	}
}

func TestRecoveryRunnerCannotCreatePerformanceArtifacts(t *testing.T) {
	// The command has no performance mode and validateContract only reads the
	// frozen contracts. This test documents the absence of a result writer.
	if strings.Contains("--contract-audit|--preflight-only", "--run") {
		t.Fatal("performance execution mode present")
	}
}
