package aishadow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type diagnosticConstantProvider struct {
	content  string
	calls    int
	requests []ProviderRequest
}

func (p *diagnosticConstantProvider) Complete(request ProviderRequest) (ProviderResponse, error) {
	p.calls++
	p.requests = append(p.requests, request)
	return ProviderResponse{Content: p.content, ModelIdentifier: "test-model"}, nil
}

func diagnosticTestConfig() Config {
	return Config{Enabled: true, Provider: "ollama", Model: "test-model", BaseURL: "http://localhost:11434", Timeout: 120 * time.Second, Temperature: 0, Seed: 20260803, MaxEvents: 48}
}

func diagnosticTestSafety() DiagnosticSafetyState {
	return DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1}
}

func diagnosticTestPaths(t *testing.T) DiagnosticPaths {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	return DiagnosticPaths{
		ManifestPath:        filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		FingerprintLockPath: filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		AssetRulesetPath:    filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		OutputRoot:          t.TempDir(),
	}
}

func TestPrepareDiagnosticVerifiesFrozenCompletenessAndSymbolicIDs(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.Repetitions != 3 || prepared.Plan.CasesPerRepetition != 48 {
		t.Fatalf("unexpected diagnostic plan: %+v", prepared.Plan)
	}
	if len(prepared.Plan.Events) != 48 || prepared.Plan.Events[0].ID != "issuer-diag-001" || prepared.Plan.Events[47].ID != "issuer-diag-048" {
		t.Fatalf("symbolic diagnostic IDs or event order changed: %+v", prepared.Plan.Events)
	}
	for index, event := range prepared.Manifest.Events {
		got, err := EventInputFingerprint(event.Input)
		if err != nil || got != prepared.Lock.Events[index].InputFingerprint {
			t.Fatalf("event %s fingerprint mismatch: got=%s err=%v", event.ID, got, err)
		}
	}
}

func TestPrepareDiagnosticRejectsManifestAndInputFingerprintMismatch(t *testing.T) {
	paths := diagnosticTestPaths(t)
	manifestRaw, err := os.ReadFile(paths.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	paths.ManifestPath = filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(paths.ManifestPath, append(manifestRaw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareDiagnostic(paths, diagnosticTestConfig(), diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "file hash changed") {
		t.Fatalf("changed frozen manifest was not rejected: %v", err)
	}

	paths = diagnosticTestPaths(t)
	lock, err := LoadDiagnosticFingerprintLock(paths.FingerprintLockPath)
	if err != nil {
		t.Fatal(err)
	}
	lock.Events[0].InputFingerprint = strings.Repeat("0", 64)
	lock.Fingerprint, err = diagnosticFingerprintLockFingerprint(lock)
	if err != nil {
		t.Fatal(err)
	}
	paths.FingerprintLockPath = filepath.Join(t.TempDir(), "lock.json")
	raw, _ := json.Marshal(lock)
	if err := os.WriteFile(paths.FingerprintLockPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareDiagnostic(paths, diagnosticTestConfig(), diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "input fingerprint changed") {
		t.Fatalf("changed EventInput fingerprint was not rejected: %v", err)
	}
}

func TestPrepareDiagnosticFailsClosedOnUnsafeStateAndChangedCaseLimit(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		safety DiagnosticSafetyState
		want   string
	}{
		{name: "wrong case limit", config: func() Config { value := diagnosticTestConfig(); value.MaxEvents = 47; return value }(), safety: diagnosticTestSafety(), want: "JAX_AI_MAX_EVENTS=48"},
		{name: "live runtime", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "live", MaximumLeverage: 1}, want: "unsafe issuer diagnostic state"},
		{name: "live trading allowed", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "paper", AllowLiveTrading: true, MaximumLeverage: 1}, want: "unsafe issuer diagnostic state"},
		{name: "execution enabled", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "paper", ExecutionEnabled: true, MaximumLeverage: 1}, want: "unsafe issuer diagnostic state"},
		{name: "execution worker enabled", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "paper", ExecutionWorker: true, MaximumLeverage: 1}, want: "unsafe issuer diagnostic state"},
		{name: "broker execution allowed", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "paper", BrokerExecution: true, MaximumLeverage: 1}, want: "unsafe issuer diagnostic state"},
		{name: "leverage above one", config: diagnosticTestConfig(), safety: DiagnosticSafetyState{RuntimeMode: "paper", MaximumLeverage: 1.01}, want: "unsafe issuer diagnostic state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PrepareDiagnostic(diagnosticTestPaths(t), tt.config, tt.safety)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unsafe or changed execution shape was not rejected: %v", err)
			}
		})
	}
}

func TestDiagnosticPreflightWritesIsolatedAuditWithoutProvider(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	paths, hash, err := WriteDiagnosticPreflight(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || !strings.HasPrefix(paths.Preflight, prepared.Paths.OutputRoot) {
		t.Fatalf("unexpected preflight audit: paths=%+v hash=%s", paths, hash)
	}
	raw, err := os.ReadFile(paths.Preflight)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"ollama_contact": false`) || !strings.Contains(string(raw), `"inference": false`) || !strings.Contains(string(raw), `"issuer-diag-048"`) {
		t.Fatalf("preflight audit is incomplete: %s", raw)
	}
	if _, err := writeExclusiveJSON(paths.Preflight, prepared.Plan); err == nil {
		t.Fatal("append-only audit file was overwritten")
	}
}

func TestExecuteDiagnosticUsesThreeRepetitionsAndFileAuditOnly(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	provider := &diagnosticConstantProvider{content: unresolvedJSON()}
	report, paths, err := ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: "test-model", Digest: "test-digest"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 48*3 || len(report.Repetitions) != 3 {
		t.Fatalf("execution shape changed: calls=%d repetitions=%d", provider.calls, len(report.Repetitions))
	}
	for index, repetition := range report.Repetitions {
		if repetition.Repetition != index+1 || repetition.Contract.TotalEvents != 48 || repetition.Contract.FinalValidOutputs != 48 {
			t.Fatalf("unexpected repetition report: %+v", repetition)
		}
	}
	if report.Repeatability.SemanticClassificationStable != 48 || report.Repeatability.EventsIdenticalAllRuns != 48 {
		t.Fatalf("unexpected repeatability: %+v", report.Repeatability)
	}
	for repetition := 1; repetition <= 3; repetition++ {
		path := filepath.Join(paths.Directory, fmt.Sprintf("repetition-%02d", repetition), "issuer-diag-001-attempt-01.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), `"raw_response_body"`) || !strings.Contains(string(raw), `"repetition": `+fmt.Sprint(repetition)) {
			t.Fatalf("attempt audit is incomplete: %s", raw)
		}
	}
	if _, err := os.Stat(paths.ReportJSON); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.ReportMarkdown); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteDiagnosticRejectsUnpinnedModelBeforeInference(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	provider := &diagnosticConstantProvider{content: unresolvedJSON()}
	_, _, err = ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: "different-model", Digest: "test-digest"})
	if err == nil || !strings.Contains(err.Error(), "model identity does not match") {
		t.Fatalf("unpinned model identity was not rejected: %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("model identity failure contacted inference provider %d times", provider.calls)
	}
}

func TestDiagnosticAdjudicatedComparisonAndV4ResolutionReuse(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	event := prepared.Manifest.Events[0]
	fingerprint, _ := EventInputFingerprint(event.Input)
	provider := &diagnosticConstantProvider{content: directJSON("Apple")}
	result, attempts, traces, err := analyseEvent(prepared.Config, provider, prepared.Resolver, "run", DiagnosticManifestVersion, event.ID, fingerprint, event.Input, prepared.ProxyExposures)
	if err != nil {
		t.Fatal(err)
	}
	if result.Resolution == nil || result.Resolution.ResolvedTicker != "AAPL" || len(attempts) != 1 {
		t.Fatalf("committed v4 validation/resolution was not reused: %+v", result)
	}
	if len(provider.requests) != 1 || strings.Contains(provider.requests[0].User, "adjudicated_label") || strings.Contains(provider.requests[0].User, event.Label.DirectIssuer) {
		t.Fatalf("diagnostic answer leaked into model input: %+v", provider.requests)
	}
	manifest := DiagnosticManifest{Events: []DiagnosticEvent{event}}
	report := EvaluateDiagnosticRepetition(1, manifest, []DiagnosticCaseRun{{CaseID: event.ID, Category: event.Category, InputFingerprint: fingerprint, Attempts: attempts, Traces: traces, Result: result}}, prepared.Resolver)
	if report.Issuer.TruePositives != 1 || report.Issuer.FalseNegatives != 0 || !report.Cases[0].SemanticCorrect || !report.Cases[0].ResolutionCorrect {
		t.Fatalf("adjudicated comparison failed: %+v", report)
	}
}
