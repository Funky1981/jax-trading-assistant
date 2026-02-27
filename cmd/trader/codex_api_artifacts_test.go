package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTestingArtifactMappings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		gate            string
		testType        string
		wantDir         string
		wantPrimaryFile string
	}{
		{gate: "Gate1", testType: "data_recon", wantDir: "gate1", wantPrimaryFile: "summary.md"},
		{gate: "Gate5", testType: "pnl_recon", wantDir: "gate5", wantPrimaryFile: "pnl_recon.md"},
		{gate: "Gate6", testType: "failure_suite", wantDir: "gate6", wantPrimaryFile: "report.md"},
		{gate: "Gate7", testType: "flatten_proof", wantDir: "gate7", wantPrimaryFile: "proof.md"},
	}
	for _, tc := range cases {
		if got := testingArtifactDir(tc.gate, tc.testType); got != tc.wantDir {
			t.Fatalf("testingArtifactDir(%q, %q) = %q, want %q", tc.gate, tc.testType, got, tc.wantDir)
		}
		if got := testingPrimaryArtifactFile(tc.testType); got != tc.wantPrimaryFile {
			t.Fatalf("testingPrimaryArtifactFile(%q) = %q, want %q", tc.testType, got, tc.wantPrimaryFile)
		}
	}
}

func TestWriteTestingArtifactReportCreatesExpectedFiles(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	datePath := time.Now().UTC().Format("2006-01-02")
	cases := []struct {
		testType       string
		extraArtifacts []string
	}{
		{testType: "data_recon", extraArtifacts: []string{"recon.csv", "summary.md"}},
		{testType: "pnl_recon", extraArtifacts: []string{"fills.csv", "corrections.csv", "pnl_recon.md"}},
		{testType: "failure_suite", extraArtifacts: []string{"report.md"}},
		{testType: "flatten_proof", extraArtifacts: []string{"proof.md", "violations.csv"}},
	}

	for _, tc := range cases {
		summary := map[string]any{
			"status": "passed",
			"commands": []map[string]any{
				{
					"command":    "echo test",
					"status":     "passed",
					"exitCode":   0,
					"durationMs": 1,
					"output":     "ok",
				},
			},
		}
		artifactPath := writeTestingArtifactReport(nil, "GateX", tc.testType, summary)
		artifactFile := filepath.FromSlash(strings.TrimPrefix(artifactPath, "/"))
		if _, err := os.Stat(artifactFile); err != nil {
			t.Fatalf("primary artifact missing for %q: %v", tc.testType, err)
		}
		artifactDir := filepath.Join("reports", testingArtifactDir("GateX", tc.testType), datePath)
		for _, rel := range tc.extraArtifacts {
			if _, err := os.Stat(filepath.Join(artifactDir, rel)); err != nil {
				t.Fatalf("extra artifact %q missing for %q: %v", rel, tc.testType, err)
			}
		}
	}
}
