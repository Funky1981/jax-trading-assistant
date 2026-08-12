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

func TestLoadDiagnosticRepetitionSelectionFailsClosed(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		supplied bool
		want     int
		wantErr  bool
	}{
		{name: "default", want: 3},
		{name: "explicit one", value: "1", supplied: true, want: 1},
		{name: "explicit three", value: "3", supplied: true, want: 3},
		{name: "zero", value: "0", supplied: true, wantErr: true},
		{name: "negative", value: "-1", supplied: true, wantErr: true},
		{name: "two", value: "2", supplied: true, wantErr: true},
		{name: "above three", value: "4", supplied: true, wantErr: true},
		{name: "malformed", value: "one", supplied: true, wantErr: true},
		{name: "empty", value: "", supplied: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shape, err := LoadDiagnosticRepetitionSelection(func(key string) (string, bool) {
				if key != DiagnosticRepetitionsEnv {
					t.Fatalf("unexpected environment lookup: %s", key)
				}
				return tt.value, tt.supplied
			})
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "exactly 1 or 3") {
					t.Fatalf("unsupported selection was not rejected: shape=%+v err=%v", shape, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if shape.RequestedRepetitions != tt.want || shape.EffectiveRepetitions != tt.want ||
				shape.TotalPlannedCases != tt.want*48 || shape.OverrideSupplied != tt.supplied {
				t.Fatalf("unexpected execution shape: %+v", shape)
			}
		})
	}
}

func TestPrepareDiagnosticVerifiesFrozenCompletenessAndSymbolicIDs(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.Repetitions != 3 || prepared.Plan.CasesPerRepetition != 48 ||
		prepared.Plan.ExecutionShape.RequestedRepetitions != 3 || prepared.Plan.ExecutionShape.EffectiveRepetitions != 3 ||
		prepared.Plan.ExecutionShape.OverrideSupplied || prepared.Plan.ExecutionShape.TotalPlannedCases != 144 {
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

func TestPrepareHostedDiagnosticLocksA1BudgetNamespaceAndNoMutation(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	config := openAITestConfig()
	prepared, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	plan := prepared.Plan.HostedExperiment
	if plan == nil || plan.ExperimentID != "A1" || plan.EvidenceNamespace != "openai-hosted-a1-v1/A1" || !plan.APIKeyPresent {
		t.Fatalf("hosted experiment identity is incomplete: %+v", plan)
	}
	if plan.DatabaseMutationAllowed || plan.TradingStateMutationAllowed || plan.BudgetCeilingUSD != "1.00" {
		t.Fatalf("hosted safety or budget plan is wrong: %+v", plan)
	}
	if prepared.Plan.Repetitions != 3 || prepared.Plan.ExecutionShape.TotalPlannedCases != 144 || plan.BaseRequestCount != 144 || plan.MaximumRequestCount != 288 {
		t.Fatalf("hosted default execution shape changed: plan=%+v execution=%+v", plan, prepared.Plan.ExecutionShape)
	}
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.ManifestFileSHA256 != ExpectedDiagnosticManifestFileSHA256 || prepared.Plan.PromptVersion != PromptVersion || prepared.Plan.OutputContract != SchemaVersion {
		t.Fatalf("hosted plan changed frozen benchmark identity: %+v", prepared.Plan)
	}
	if prepared.Plan.ModelConfiguration.Provider != OpenAIDiagnosticProvider || prepared.Plan.ModelConfiguration.Model != OpenAIDiagnosticModel || prepared.Plan.ModelConfiguration.StructuredOutputMode != OpenAIDiagnosticStructuredOutput {
		t.Fatalf("hosted provider plan is wrong: %+v", prepared.Plan.ModelConfiguration)
	}

	unsafePaths := diagnosticTestPaths(t)
	unsafePaths.OutputRoot = filepath.Join(t.TempDir(), "ai-shadow-issuer")
	if _, err := PrepareHostedDiagnostic(unsafePaths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "isolated namespace") {
		t.Fatalf("baseline evidence namespace was not rejected: %v", err)
	}
}

func TestPrepareLunaDiagnosticRecordsOneRepetitionBudgetAndFrozenIdentity(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	config := openAILunaTestConfig()
	prepared, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Plan.ModelConfiguration.Model != OpenAIDiagnosticLunaModel || prepared.Plan.Repetitions != 3 || prepared.Plan.ExecutionShape.TotalPlannedCases != 144 {
		t.Fatalf("Luna default selection or identity changed: %+v", prepared.Plan)
	}
	shape, err := LoadDiagnosticRepetitionSelection(func(string) (string, bool) { return "1", true })
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, shape)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiagnosticExecutionShape(prepared); err != nil {
		t.Fatal(err)
	}
	plan := prepared.Plan.HostedExperiment
	if plan == nil || plan.BaseRequestCount != 48 || plan.MaximumRequestCount != 96 || plan.BudgetCeilingUSD != "0.12" ||
		plan.Pricing.InputUSDPerMillionTokens != "0.20" || plan.Pricing.CachedInputUSDPerMillionTokens != "0.02" ||
		plan.Pricing.CacheWriteUSDPerMillionTokens != "0.25" || plan.Pricing.OutputUSDPerMillionTokens != "1.20" ||
		plan.LargestFrozenInitialRequestBytes <= 0 || plan.ConservativeCorrectiveRequestBytes <= 0 {
		t.Fatalf("Luna execution plan is incomplete: %+v", plan)
	}
	estimated, err := parseUSDMicros(plan.EstimatedMaximumRunUSD)
	if err != nil || estimated <= 0 || estimated > config.BudgetCeilingMicros {
		t.Fatalf("Luna complete-run estimate=%s ceiling=%s err=%v", plan.EstimatedMaximumRunUSD, plan.BudgetCeilingUSD, err)
	}
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.ManifestFileSHA256 != ExpectedDiagnosticManifestFileSHA256 ||
		prepared.Plan.FingerprintLockFingerprint != prepared.Lock.Fingerprint || prepared.Plan.PromptVersion != PromptVersion || prepared.Plan.OutputContract != SchemaVersion {
		t.Fatalf("Luna plan changed frozen benchmark identities: %+v", prepared.Plan)
	}
}

func TestLunaDefaultThreeRepetitionPlanRetains144CasesButFailsInsufficientBudget(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	prepared, err := PrepareHostedDiagnostic(paths, openAILunaTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Plan.ExecutionShape.TotalPlannedCases != 144 {
		t.Fatalf("default hosted execution shape changed: %+v", prepared.Plan.ExecutionShape)
	}
	if err := ValidateDiagnosticExecutionShape(prepared); err == nil || !strings.Contains(err.Error(), "complete-run estimate") {
		t.Fatalf("underfunded three-repetition Luna plan was not rejected: %v", err)
	}
}

func TestPrepareLunaStructuredOutputsCellLocksContractNamespaceAndOneRepetition(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), OpenAIStructuredOutputsEvidenceNamespace, OpenAIStructuredOutputsExperimentID)
	config := openAIStructuredOutputsTestConfig()
	prepared, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiagnosticExecutionShape(prepared); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("C1B default three-repetition shape did not fail closed: %v", err)
	}
	shape, err := LoadDiagnosticRepetitionSelection(func(string) (string, bool) { return "1", true })
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, shape)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDiagnosticExecutionShape(prepared); err != nil {
		t.Fatal(err)
	}
	plan := prepared.Plan.HostedExperiment
	model := prepared.Plan.ModelConfiguration
	if plan == nil || plan.ExperimentID != OpenAIStructuredOutputsExperimentID || plan.CellIdentity != OpenAIStructuredOutputsExperimentID ||
		plan.EvidenceNamespace != OpenAIStructuredOutputsEvidenceNamespace+"/"+OpenAIStructuredOutputsExperimentID ||
		!plan.StructuredOutputs || plan.SchemaContract != SchemaVersion || len(plan.SchemaSHA256) != 64 ||
		plan.ContractEnforcement != string(OpenAIOutputContractStrictJSONSchema) || plan.BaseRequestCount != 48 || plan.MaximumRequestCount != 96 {
		t.Fatalf("C1B hosted plan is incomplete: %+v", plan)
	}
	if plan.ServiceTier != OpenAIStructuredOutputsServiceTier {
		t.Fatalf("C1B service tier is not pinned: %+v", plan)
	}
	if !model.StructuredOutputs || model.StructuredOutputMode != OpenAIStructuredOutputsMode || model.SchemaContract != SchemaVersion || model.ServiceTier != OpenAIStructuredOutputsServiceTier ||
		model.ContractEnforcement != string(OpenAIOutputContractStrictJSONSchema) {
		t.Fatalf("C1B model contract is ambiguous: %+v", model)
	}
	if plan.Pricing.InputUSDPerMillionTokens != "0.20" || plan.Pricing.CachedInputUSDPerMillionTokens != "0.02" ||
		plan.Pricing.CacheWriteUSDPerMillionTokens != "0.25" || plan.Pricing.OutputUSDPerMillionTokens != "1.20" || plan.BudgetCeilingUSD != "0.20" {
		t.Fatalf("C1B review-time pricing inputs are wrong: %+v", plan.Pricing)
	}
	estimated, err := parseUSDMicros(plan.EstimatedMaximumRunUSD)
	if err != nil || estimated <= 0 || estimated > config.BudgetCeilingMicros || plan.EstimatedMaximumRunUSD != "0.145073" ||
		plan.LargestFrozenInitialRequestBytes != 3785 || plan.ConservativeCorrectiveRequestBytes != 3829 {
		t.Fatalf("C1B complete-run estimate=%s ceiling=%s err=%v", plan.EstimatedMaximumRunUSD, plan.BudgetCeilingUSD, err)
	}
	plan.EstimatedMaximumRunUSD = "0.200001"
	if err := ValidateDiagnosticExecutionShape(prepared); err == nil || !strings.Contains(err.Error(), "cannot accommodate") {
		t.Fatalf("C1B did not fail closed above the hard ceiling: %v", err)
	}
	plan.EstimatedMaximumRunUSD = "0.145073"
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.ManifestFileSHA256 != ExpectedDiagnosticManifestFileSHA256 ||
		prepared.Plan.FingerprintLockFingerprint != prepared.Lock.Fingerprint || prepared.Plan.PromptVersion != PromptVersion || prepared.Plan.OutputContract != SchemaVersion {
		t.Fatalf("C1B changed frozen benchmark identities: %+v", prepared.Plan)
	}
	unsafePaths := diagnosticTestPaths(t)
	unsafePaths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	if _, err := PrepareHostedDiagnostic(unsafePaths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "isolated namespace") {
		t.Fatalf("C1B accepted the prior A1 namespace: %v", err)
	}
}

func TestPrepareDeepSeekDiagnosticLocksA1ModeBudgetNamespaceAndNoMutation(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), DeepSeekDiagnosticEvidenceNamespace, DeepSeekDiagnosticExperimentID)
	config := deepSeekTestConfig()
	prepared, err := PrepareDeepSeekDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	plan := prepared.Plan.HostedExperiment
	if plan == nil || plan.ExperimentID != "A1" || plan.EvidenceNamespace != "deepseek-hosted-a1-v1/A1" || !plan.APIKeyPresent {
		t.Fatalf("DeepSeek experiment identity is incomplete: %+v", plan)
	}
	if plan.DatabaseMutationAllowed || plan.TradingStateMutationAllowed || plan.InferenceExplicitlyAuthorized || plan.BudgetCeilingUSD != "0.10" {
		t.Fatalf("DeepSeek safety, authorization, or budget plan is wrong: %+v", plan)
	}
	if prepared.Plan.Repetitions != 3 || prepared.Plan.ExecutionShape.TotalPlannedCases != 144 || plan.BaseRequestCount != 144 || plan.MaximumRequestCount != 288 {
		t.Fatalf("DeepSeek default execution shape changed: plan=%+v execution=%+v", plan, prepared.Plan.ExecutionShape)
	}
	if prepared.Plan.ManifestFingerprint != ExpectedDiagnosticManifestFingerprint || prepared.Plan.ManifestFileSHA256 != ExpectedDiagnosticManifestFileSHA256 ||
		prepared.Plan.FingerprintLockFingerprint == "" || prepared.Plan.PromptVersion != PromptVersion || prepared.Plan.OutputContract != SchemaVersion {
		t.Fatalf("DeepSeek plan changed frozen benchmark identity: %+v", prepared.Plan)
	}
	model := prepared.Plan.ModelConfiguration
	if model.Provider != DeepSeekDiagnosticProvider || model.Model != DeepSeekDiagnosticModel || model.ThinkingMode != DeepSeekDiagnosticThinkingMode ||
		model.Think || model.StructuredOutputMode != DeepSeekDiagnosticStructuredOutput || model.RetryLimit != 1 {
		t.Fatalf("DeepSeek provider plan is wrong: %+v", model)
	}

	unsafePaths := diagnosticTestPaths(t)
	unsafePaths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	if _, err := PrepareDeepSeekDiagnostic(unsafePaths, config, diagnosticTestSafety()); err == nil || !strings.Contains(err.Error(), "isolated namespace") {
		t.Fatalf("OpenAI evidence namespace was not rejected: %v", err)
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
	if !strings.Contains(string(raw), `"ollama_contact": false`) || !strings.Contains(string(raw), `"inference": false`) ||
		!strings.Contains(string(raw), `"requested_repetitions": 3`) || !strings.Contains(string(raw), `"effective_repetitions": 3`) ||
		!strings.Contains(string(raw), `"total_planned_cases": 144`) || !strings.Contains(string(raw), `"issuer-diag-048"`) {
		t.Fatalf("preflight audit is incomplete: %s", raw)
	}
	if _, err := writeExclusiveJSON(paths.Preflight, prepared.Plan); err == nil {
		t.Fatal("append-only audit file was overwritten")
	}
}

func TestOneRepetitionPlanAndPreflightRecordFortyEightCases(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), DeepSeekDiagnosticEvidenceNamespace, DeepSeekDiagnosticExperimentID)
	prepared, err := PrepareDeepSeekDiagnostic(paths, deepSeekTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	shape, err := LoadDiagnosticRepetitionSelection(func(key string) (string, bool) { return "1", true })
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, shape)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Plan.Repetitions != 1 || prepared.Plan.ExecutionShape.TotalPlannedCases != 48 ||
		prepared.Plan.HostedExperiment.BaseRequestCount != 48 || prepared.Plan.HostedExperiment.MaximumRequestCount != 96 {
		t.Fatalf("one-repetition hosted plan is wrong: %+v", prepared.Plan)
	}
	pathsWritten, _, err := WriteDiagnosticPreflight(prepared)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(pathsWritten.Preflight)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"requested_repetitions": 1`, `"effective_repetitions": 1`, `"total_planned_cases": 48`, `"experiment_id": "A1"`, `"provider": "deepseek"`, `"model": "deepseek-v4-pro"`, `"inference_explicitly_authorized": false`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("one-repetition preflight missing %s: %s", want, raw)
		}
	}
}

func TestHostedPreflightRejectsInferenceAuthorization(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), DeepSeekDiagnosticEvidenceNamespace, DeepSeekDiagnosticExperimentID)
	config := deepSeekTestConfig()
	config.InferenceExplicitlyAuthorized = true
	prepared, err := PrepareDeepSeekDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := WriteDiagnosticPreflight(prepared); err == nil || !strings.Contains(err.Error(), "authorization to remain false") {
		t.Fatalf("authorized hosted preflight was not rejected: %v", err)
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
	if paths.ArtifactIndex == "" || paths.ArtifactIndexSHA256 == "" {
		t.Fatalf("artifact hashes were not indexed: %+v", paths)
	}
	if _, err := writeExclusiveJSON(paths.ArtifactIndex, report); err == nil {
		t.Fatal("append-only artifact index was overwritten")
	}
}

func TestExecuteDiagnosticUsesExplicitOneRepetition(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	shape, err := LoadDiagnosticRepetitionSelection(func(key string) (string, bool) { return "1", true })
	if err != nil {
		t.Fatal(err)
	}
	prepared, err = ApplyDiagnosticExecutionShape(prepared, shape)
	if err != nil {
		t.Fatal(err)
	}
	provider := &diagnosticConstantProvider{content: unresolvedJSON()}
	report, _, err := ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: "test-model", Digest: "test-digest"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 48 || len(report.Repetitions) != 1 || report.Repetitions[0].Contract.TotalEvents != 48 {
		t.Fatalf("one-repetition execution shape changed: calls=%d report=%+v", provider.calls, report)
	}
}

func TestExecuteDiagnosticRejectsPlanRuntimeShapeMismatchBeforeInference(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	prepared.Plan.Repetitions = 1
	provider := &diagnosticConstantProvider{content: unresolvedJSON()}
	_, _, err = ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: "test-model", Digest: "test-digest"})
	if err == nil || !strings.Contains(err.Error(), "does not match validated runtime selection") {
		t.Fatalf("execution-shape mismatch was not rejected: %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("execution-shape mismatch contacted provider %d times", provider.calls)
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

func TestHostedBudgetStopWritesAppendOnlyStopRecordWithoutHTTP(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), OpenAIDiagnosticEvidenceNamespace, OpenAIDiagnosticExperimentID)
	config := openAITestConfig()
	prepared, err := PrepareHostedDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	doer := &queuedHTTPDoer{}
	provider := NewOpenAIDiagnosticClient(config, doer)
	provider.budget.ceilingMicros = 1
	_, audit, err := ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: OpenAIDiagnosticModel})
	if err == nil || !strings.Contains(err.Error(), "budget rejected request") {
		t.Fatalf("budget stop was not returned: %v", err)
	}
	if doer.calls != 0 {
		t.Fatalf("budget-stopped execution made %d HTTP calls", doer.calls)
	}
	if audit.StopRecord == "" || audit.ArtifactIndex == "" || audit.ArtifactIndexSHA256 == "" {
		t.Fatalf("budget stop evidence is incomplete: %+v", audit)
	}
	raw, err := os.ReadFile(audit.StopRecord)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"stop_reason"`) || !strings.Contains(string(raw), `"budget_rejection_count": 1`) {
		t.Fatalf("budget stop record is incomplete: %s", raw)
	}
	if _, err := writeExclusiveJSON(audit.StopRecord, prepared.Plan); err == nil {
		t.Fatal("append-only hosted stop record was overwritten")
	}
}

func TestDeepSeekBudgetStopWritesAppendOnlyStopRecordWithoutHTTP(t *testing.T) {
	paths := diagnosticTestPaths(t)
	paths.OutputRoot = filepath.Join(t.TempDir(), DeepSeekDiagnosticEvidenceNamespace, DeepSeekDiagnosticExperimentID)
	config := deepSeekTestConfig()
	prepared, err := PrepareDeepSeekDiagnostic(paths, config, diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	doer := &queuedHTTPDoer{}
	provider := NewDeepSeekDiagnosticClient(config, doer)
	provider.budget.ceilingMicros = 1
	_, audit, err := ExecuteDiagnostic(prepared, provider, DiagnosticModelIdentity{Name: DeepSeekDiagnosticModel})
	if err == nil || !strings.Contains(err.Error(), "budget rejected request") {
		t.Fatalf("DeepSeek budget stop was not returned: %v", err)
	}
	if doer.calls != 0 || audit.StopRecord == "" || audit.ArtifactIndex == "" || audit.ArtifactIndexSHA256 == "" {
		t.Fatalf("DeepSeek no-network stop evidence is incomplete: calls=%d audit=%+v", doer.calls, audit)
	}
	if _, err := writeExclusiveJSON(audit.StopRecord, prepared.Plan); err == nil {
		t.Fatal("append-only DeepSeek stop record was overwritten")
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

func TestDiagnosticAttemptAuditCarriesDeepSeekFingerprintFinishAndCacheEvidence(t *testing.T) {
	prepared, err := PrepareDiagnostic(diagnosticTestPaths(t), diagnosticTestConfig(), diagnosticTestSafety())
	if err != nil {
		t.Fatal(err)
	}
	event := prepared.Manifest.Events[0]
	attempt := Attempt{
		AttemptNumber: 1, Provider: DeepSeekDiagnosticProvider, Model: DeepSeekDiagnosticModel,
		ModelReportedIdentifier: DeepSeekDiagnosticModel, PromptVersion: PromptVersion,
		SchemaVersion: SchemaVersion, ValidationStatus: "accepted",
	}
	trace := ProviderTrace{
		AttemptNumber: 1, Content: directJSON("Apple"), ModelIdentifier: DeepSeekDiagnosticModel,
		RequestID: "req", ResponseID: "resp", Status: "completed", SystemFingerprint: "fp_test", FinishReason: "stop",
		Usage: ProviderUsage{InputTokens: 100, CachedTokens: 20, CacheMissTokens: 80, OutputTokens: 10, TotalTokens: 110},
	}
	audit := buildDiagnosticAttemptAudit("run", 1, event, attempt, trace, prepared.Resolver)
	if audit.SystemFingerprint != "fp_test" || audit.FinishReason != "stop" || audit.Usage.CacheMissTokens != 80 ||
		audit.RequestID != "req" || audit.ResponseID != "resp" || audit.ModelReportedIdentifier != DeepSeekDiagnosticModel {
		t.Fatalf("DeepSeek attempt evidence was not carried into the append-only audit: %+v", audit)
	}
}
