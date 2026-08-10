package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"

	"github.com/google/uuid"
)

const DiagnosticReportVersion = "ai-shadow-issuer-diagnostic-report-v2"

type DiagnosticPaths struct {
	ManifestPath        string
	FingerprintLockPath string
	AssetRulesetPath    string
	OutputRoot          string
}

type DiagnosticSafetyState struct {
	RuntimeMode      string  `json:"runtime_mode"`
	AllowLiveTrading bool    `json:"allow_live_trading"`
	ExecutionEnabled bool    `json:"execution_enabled"`
	ExecutionWorker  bool    `json:"execution_worker_enabled"`
	BrokerExecution  bool    `json:"broker_execution_allowed"`
	MaximumLeverage  float64 `json:"maximum_leverage"`
}

type DiagnosticModelConfiguration struct {
	Provider             string  `json:"provider"`
	Model                string  `json:"model"`
	BaseURL              string  `json:"base_url"`
	TimeoutSeconds       int     `json:"timeout_seconds"`
	Temperature          float64 `json:"temperature"`
	Seed                 int64   `json:"seed"`
	Stream               bool    `json:"stream"`
	Think                bool    `json:"think"`
	RetryLimit           int     `json:"retry_limit"`
	ReasoningEffort      string  `json:"reasoning_effort,omitempty"`
	MaxOutputTokens      int     `json:"max_output_tokens,omitempty"`
	StructuredOutputMode string  `json:"structured_output_mode,omitempty"`
}

type HostedExperimentPlan struct {
	ExperimentID                  string            `json:"experiment_id"`
	EvidenceNamespace             string            `json:"evidence_namespace"`
	Endpoint                      string            `json:"endpoint"`
	APIKeyEnvironment             string            `json:"api_key_environment"`
	APIKeyPresent                 bool              `json:"api_key_present"`
	InferenceExplicitlyAuthorized bool              `json:"inference_explicitly_authorized"`
	BudgetCeilingUSD              string            `json:"budget_ceiling_usd"`
	Pricing                       HostedPricingPlan `json:"pricing"`
	BaseRequestCount              int               `json:"base_request_count"`
	MaximumRequestCount           int               `json:"maximum_request_count"`
	EstimatedFirstRequestMaxUSD   string            `json:"estimated_first_request_max_usd"`
	DatabaseMutationAllowed       bool              `json:"database_mutation_allowed"`
	TradingStateMutationAllowed   bool              `json:"trading_state_mutation_allowed"`
}

type DiagnosticPlanEvent struct {
	Position         int    `json:"position"`
	ID               string `json:"id"`
	Category         string `json:"category"`
	InputFingerprint string `json:"input_fingerprint"`
}

type DiagnosticPlan struct {
	Version                    string                       `json:"version"`
	ManifestVersion            string                       `json:"manifest_version"`
	ManifestFingerprint        string                       `json:"manifest_fingerprint"`
	ManifestFileSHA256         string                       `json:"manifest_file_sha256"`
	FingerprintLockVersion     string                       `json:"fingerprint_lock_version"`
	FingerprintLockFingerprint string                       `json:"fingerprint_lock_fingerprint"`
	LabelVersion               string                       `json:"label_version"`
	PromptVersion              string                       `json:"prompt_version"`
	OutputContract             string                       `json:"output_contract"`
	PolicyVersion              string                       `json:"policy_version"`
	Repetitions                int                          `json:"repetitions"`
	CasesPerRepetition         int                          `json:"cases_per_repetition"`
	ModelConfiguration         DiagnosticModelConfiguration `json:"model_configuration"`
	Safety                     DiagnosticSafetyState        `json:"safety"`
	HostedExperiment           *HostedExperimentPlan        `json:"hosted_experiment,omitempty"`
	Events                     []DiagnosticPlanEvent        `json:"events"`
}

type PreparedDiagnostic struct {
	Plan           DiagnosticPlan
	Manifest       DiagnosticManifest
	Lock           DiagnosticFingerprintLock
	Config         Config
	Resolver       assetresolution.Resolver
	ProxyExposures []string
	Paths          DiagnosticPaths
}

type DiagnosticModelIdentity struct {
	Name              string `json:"name"`
	Digest            string `json:"digest"`
	Format            string `json:"format,omitempty"`
	Family            string `json:"family,omitempty"`
	ParameterSize     string `json:"parameter_size,omitempty"`
	QuantizationLevel string `json:"quantization_level,omitempty"`
}

type DiagnosticAttemptAudit struct {
	RunID                   string            `json:"run_id"`
	Repetition              int               `json:"repetition"`
	CaseID                  string            `json:"case_id"`
	Category                string            `json:"category"`
	AttemptNumber           int               `json:"attempt_number"`
	InputFingerprint        string            `json:"input_fingerprint"`
	Provider                string            `json:"provider"`
	ConfiguredModel         string            `json:"configured_model"`
	ModelReportedIdentifier string            `json:"model_reported_identifier,omitempty"`
	PromptVersion           string            `json:"prompt_version"`
	OutputContract          string            `json:"output_contract"`
	PolicyVersion           string            `json:"policy_version"`
	Seed                    int64             `json:"seed"`
	Temperature             float64           `json:"temperature"`
	RequestTimestamp        time.Time         `json:"request_timestamp"`
	ResponseTimestamp       time.Time         `json:"response_timestamp"`
	DurationMS              int64             `json:"duration_ms"`
	RawResponseHash         string            `json:"raw_response_hash"`
	RawResponseBody         string            `json:"raw_response_body"`
	ValidationStatus        string            `json:"validation_status"`
	ValidationErrors        []string          `json:"validation_errors"`
	FailureReason           string            `json:"failure_reason,omitempty"`
	RequestID               string            `json:"request_id,omitempty"`
	ResponseID              string            `json:"response_id,omitempty"`
	ProviderStatus          string            `json:"provider_status,omitempty"`
	Usage                   ProviderUsage     `json:"usage"`
	ModelClassification     *StructuredResult `json:"model_classification,omitempty"`
	DeterministicResolution *PolicyResolution `json:"deterministic_resolution,omitempty"`
}

type DiagnosticCaseRun struct {
	CaseID           string          `json:"case_id"`
	Category         string          `json:"category"`
	InputFingerprint string          `json:"input_fingerprint"`
	Attempts         []Attempt       `json:"attempts"`
	Traces           []ProviderTrace `json:"traces"`
	Result           EventResult     `json:"result"`
}

type DiagnosticAuditPaths struct {
	RunID               string `json:"run_id"`
	Directory           string `json:"directory"`
	Plan                string `json:"plan,omitempty"`
	ReportJSON          string `json:"report_json,omitempty"`
	ReportMarkdown      string `json:"report_markdown,omitempty"`
	Preflight           string `json:"preflight,omitempty"`
	StopRecord          string `json:"stop_record,omitempty"`
	ArtifactIndex       string `json:"artifact_index,omitempty"`
	ArtifactIndexSHA256 string `json:"artifact_index_sha256,omitempty"`
}

func PrepareDiagnostic(paths DiagnosticPaths, config Config, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	if config.MaxEvents != diagnosticEventCount {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic requires JAX_AI_MAX_EVENTS=%d", diagnosticEventCount)
	}
	if safety.RuntimeMode != "paper" || safety.AllowLiveTrading || safety.ExecutionEnabled || safety.ExecutionWorker || safety.BrokerExecution || safety.MaximumLeverage > 1 || safety.MaximumLeverage <= 0 {
		return PreparedDiagnostic{}, fmt.Errorf("unsafe issuer diagnostic state")
	}
	rules, err := assetresolution.LoadRuleset(paths.AssetRulesetPath)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	manifest, err := LoadFrozenDiagnosticManifest(paths.ManifestPath, exposures)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	lock, err := LoadDiagnosticFingerprintLock(paths.FingerprintLockPath)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	if manifest.OutputContract != SchemaVersion || lock.OutputContract != SchemaVersion {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic output contract mismatch")
	}
	if lock.PromptVersion != PromptVersion {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic prompt version mismatch")
	}
	if manifest.PolicyVersion != rules.Version || lock.PolicyVersion != rules.Version {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic policy version mismatch")
	}
	if lock.ManifestFingerprint != manifest.Fingerprint {
		return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic fingerprint lock references a different manifest")
	}

	planEvents := make([]DiagnosticPlanEvent, 0, diagnosticEventCount)
	for index, event := range manifest.Events {
		locked := lock.Events[index]
		if locked.ID != event.ID {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic event order mismatch at position %d: got %s want %s", index+1, event.ID, locked.ID)
		}
		got, err := EventInputFingerprint(event.Input)
		if err != nil {
			return PreparedDiagnostic{}, fmt.Errorf("fingerprint issuer diagnostic event %s: %w", event.ID, err)
		}
		if got != locked.InputFingerprint {
			return PreparedDiagnostic{}, fmt.Errorf("issuer diagnostic event %s input fingerprint changed: got %s want %s", event.ID, got, locked.InputFingerprint)
		}
		planEvents = append(planEvents, DiagnosticPlanEvent{Position: index + 1, ID: event.ID, Category: event.Category, InputFingerprint: got})
	}

	plan := DiagnosticPlan{
		Version: DiagnosticReportVersion, ManifestVersion: manifest.Version,
		ManifestFingerprint: manifest.Fingerprint, ManifestFileSHA256: ExpectedDiagnosticManifestFileSHA256,
		FingerprintLockVersion: lock.Version, FingerprintLockFingerprint: lock.Fingerprint,
		LabelVersion: manifest.LabelVersion, PromptVersion: PromptVersion, OutputContract: SchemaVersion,
		PolicyVersion: rules.Version, Repetitions: diagnosticRepetitionCount, CasesPerRepetition: diagnosticEventCount,
		ModelConfiguration: DiagnosticModelConfiguration{
			Provider: config.Provider, Model: config.Model, BaseURL: config.BaseURL,
			TimeoutSeconds: int(config.Timeout.Seconds()), Temperature: config.Temperature,
			Seed: config.Seed, Stream: false, Think: false, RetryLimit: 1,
		},
		Safety: safety, Events: planEvents,
	}
	return PreparedDiagnostic{Plan: plan, Manifest: manifest, Lock: lock, Config: config, Resolver: resolver, ProxyExposures: exposures, Paths: paths}, nil
}

func PrepareHostedDiagnostic(paths DiagnosticPaths, config OpenAIDiagnosticConfig, safety DiagnosticSafetyState) (PreparedDiagnostic, error) {
	if !config.APIKey.present() {
		return PreparedDiagnostic{}, fmt.Errorf("missing required hosted diagnostic configuration: %s", OpenAIDiagnosticAPIKeyEnv)
	}
	wantNamespace := filepath.Join(OpenAIDiagnosticEvidenceNamespace, config.ExperimentID)
	if !strings.HasSuffix(filepath.Clean(paths.OutputRoot), wantNamespace) {
		return PreparedDiagnostic{}, fmt.Errorf("hosted diagnostic output root must end in isolated namespace %s", filepath.ToSlash(wantNamespace))
	}
	prepared, err := PrepareDiagnostic(paths, config.Runtime, safety)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	firstRequest, err := InitialRequest(prepared.Manifest.Events[0].Input, prepared.ProxyExposures)
	if err != nil {
		return PreparedDiagnostic{}, err
	}
	estimatedInputTokens := len([]byte(firstRequest.System)) + len([]byte(firstRequest.User)) + 1024
	estimatedFirstCost := tokenCostMicros(estimatedInputTokens, config.InputPriceMicrosPerMillion) + tokenCostMicros(config.MaxOutputTokens, config.OutputPriceMicrosPerMillion)
	if estimatedFirstCost > config.BudgetCeilingMicros {
		return PreparedDiagnostic{}, fmt.Errorf("configured hosted request maximum exceeds the experiment budget ceiling")
	}
	prepared.Plan.ModelConfiguration.ReasoningEffort = config.ReasoningEffort
	prepared.Plan.ModelConfiguration.MaxOutputTokens = config.MaxOutputTokens
	prepared.Plan.ModelConfiguration.StructuredOutputMode = OpenAIDiagnosticStructuredOutput
	prepared.Plan.HostedExperiment = &HostedExperimentPlan{
		ExperimentID: config.ExperimentID, EvidenceNamespace: OpenAIDiagnosticEvidenceNamespace + "/" + config.ExperimentID,
		Endpoint: OpenAIDiagnosticEndpoint, APIKeyEnvironment: OpenAIDiagnosticAPIKeyEnv, APIKeyPresent: true,
		InferenceExplicitlyAuthorized: config.InferenceExplicitlyAuthorized,
		BudgetCeilingUSD:              formatUSDMicros(config.BudgetCeilingMicros),
		Pricing: HostedPricingPlan{
			InputUSDPerMillionTokens:       formatUSDMicros(config.InputPriceMicrosPerMillion),
			CachedInputUSDPerMillionTokens: formatUSDMicros(config.CachedInputPriceMicrosPerMillion),
			CacheWriteUSDPerMillionTokens:  formatUSDMicros(config.CacheWritePriceMicrosPerMillion),
			OutputUSDPerMillionTokens:      formatUSDMicros(config.OutputPriceMicrosPerMillion),
			Source:                         "execution-time configuration; re-verify before paid execution",
		},
		BaseRequestCount:            diagnosticEventCount * diagnosticRepetitionCount,
		MaximumRequestCount:         diagnosticEventCount * diagnosticRepetitionCount * 2,
		EstimatedFirstRequestMaxUSD: formatUSDMicros(estimatedFirstCost),
		DatabaseMutationAllowed:     false, TradingStateMutationAllowed: false,
	}
	return prepared, nil
}

func WriteDiagnosticPreflight(prepared PreparedDiagnostic) (DiagnosticAuditPaths, string, error) {
	runID := uuid.NewString()
	dir := filepath.Join(prepared.Paths.OutputRoot, "preflight", runID)
	path := filepath.Join(dir, "preflight.json")
	payload := struct {
		RunID           string         `json:"run_id"`
		Status          string         `json:"status"`
		ProviderContact bool           `json:"provider_contact"`
		OllamaContact   bool           `json:"ollama_contact"`
		Inference       bool           `json:"inference"`
		Plan            DiagnosticPlan `json:"plan"`
	}{runID, "ready", false, false, false, prepared.Plan}
	hash, err := writeExclusiveJSON(path, payload)
	if err != nil {
		return DiagnosticAuditPaths{}, "", err
	}
	return DiagnosticAuditPaths{RunID: runID, Directory: dir, Preflight: path}, hash, nil
}

func ExecuteDiagnostic(prepared PreparedDiagnostic, provider Provider, identity DiagnosticModelIdentity) (DiagnosticRunReport, DiagnosticAuditPaths, error) {
	if provider == nil {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("issuer diagnostic provider is required")
	}
	if prepared.Plan.Repetitions != diagnosticRepetitionCount || prepared.Plan.CasesPerRepetition != diagnosticEventCount {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("issuer diagnostic execution shape changed")
	}
	if identity.Name != prepared.Config.Model || prepared.Config.Provider == "ollama" && strings.TrimSpace(identity.Digest) == "" {
		return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("issuer diagnostic model identity does not match configured model")
	}
	if plan := prepared.Plan.HostedExperiment; plan != nil {
		recorder, ok := provider.(hostedExperimentRecorder)
		if !ok {
			return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("hosted issuer diagnostic provider must expose experiment accounting")
		}
		snapshot := recorder.ExperimentSnapshot()
		if snapshot.ExperimentID != plan.ExperimentID || snapshot.Provider != prepared.Config.Provider ||
			snapshot.RequestedModel != prepared.Config.Model || snapshot.ReasoningEffort != prepared.Plan.ModelConfiguration.ReasoningEffort ||
			snapshot.StructuredOutputMode != prepared.Plan.ModelConfiguration.StructuredOutputMode ||
			snapshot.MaxOutputTokensPerRequest != prepared.Plan.ModelConfiguration.MaxOutputTokens ||
			snapshot.BudgetCeilingUSD != plan.BudgetCeilingUSD || snapshot.Pricing != plan.Pricing {
			return DiagnosticRunReport{}, DiagnosticAuditPaths{}, fmt.Errorf("hosted issuer diagnostic provider does not match the preflight plan")
		}
	}

	runID := uuid.NewString()
	dir := filepath.Join(prepared.Paths.OutputRoot, runID)
	paths := DiagnosticAuditPaths{RunID: runID, Directory: dir, Plan: filepath.Join(dir, "plan.json")}
	planAudit := struct {
		RunID         string                  `json:"run_id"`
		StartedAt     time.Time               `json:"started_at"`
		ModelIdentity DiagnosticModelIdentity `json:"model_identity"`
		Plan          DiagnosticPlan          `json:"plan"`
	}{runID, time.Now().UTC(), identity, prepared.Plan}
	if _, err := writeExclusiveJSON(paths.Plan, planAudit); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if err := copyExclusive(prepared.Paths.ManifestPath, filepath.Join(dir, "manifest.json")); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if err := copyExclusive(prepared.Paths.FingerprintLockPath, filepath.Join(dir, "input-fingerprints.json")); err != nil {
		return DiagnosticRunReport{}, paths, err
	}

	repetitions := make([]DiagnosticRepetitionReport, 0, diagnosticRepetitionCount)
	allRuns := make([][]DiagnosticCaseRun, 0, diagnosticRepetitionCount)
	for repetition := 1; repetition <= diagnosticRepetitionCount; repetition++ {
		caseRuns := make([]DiagnosticCaseRun, 0, diagnosticEventCount)
		for _, event := range prepared.Manifest.Events {
			inputFingerprint, _ := EventInputFingerprint(event.Input)
			result, attempts, traces, err := analyseEvent(
				prepared.Config, provider, prepared.Resolver, runID, DiagnosticManifestVersion,
				event.ID, inputFingerprint, event.Input, prepared.ProxyExposures,
			)
			if err != nil {
				if recorder, ok := provider.(hostedExperimentRecorder); ok {
					paths.StopRecord = filepath.Join(dir, "stop.json")
					stop := struct {
						RunID      string                   `json:"run_id"`
						StoppedAt  time.Time                `json:"stopped_at"`
						StopReason string                   `json:"stop_reason"`
						Experiment HostedExperimentSnapshot `json:"experiment"`
					}{runID, time.Now().UTC(), err.Error(), recorder.ExperimentSnapshot()}
					_, _ = writeExclusiveJSON(paths.StopRecord, stop)
					paths.ArtifactIndex, paths.ArtifactIndexSHA256, _ = writeDiagnosticArtifactIndex(dir)
				}
				return DiagnosticRunReport{}, paths, err
			}
			for index, attempt := range attempts {
				audit := buildDiagnosticAttemptAudit(runID, repetition, event, attempt, traces[index], prepared.Resolver)
				attemptPath := filepath.Join(dir, fmt.Sprintf("repetition-%02d", repetition), fmt.Sprintf("%s-attempt-%02d.json", event.ID, attempt.AttemptNumber))
				if _, err := writeExclusiveJSON(attemptPath, audit); err != nil {
					return DiagnosticRunReport{}, paths, err
				}
			}
			caseRuns = append(caseRuns, DiagnosticCaseRun{CaseID: event.ID, Category: event.Category, InputFingerprint: inputFingerprint, Attempts: attempts, Traces: traces, Result: result})
		}
		allRuns = append(allRuns, caseRuns)
		repetitions = append(repetitions, EvaluateDiagnosticRepetition(repetition, prepared.Manifest, caseRuns, prepared.Resolver))
	}
	report := BuildDiagnosticRunReport(runID, prepared, identity, repetitions, allRuns)
	if recorder, ok := provider.(hostedExperimentRecorder); ok {
		snapshot := recorder.ExperimentSnapshot()
		report.HostedExperiment = &snapshot
	}
	paths.ReportJSON = filepath.Join(dir, "report.json")
	paths.ReportMarkdown = filepath.Join(dir, "report.md")
	if _, err := writeExclusiveJSON(paths.ReportJSON, report); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	if _, err := writeExclusive(filepath.Join(dir, "report.md"), []byte(DiagnosticReportMarkdown(report))); err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	indexPath, indexHash, err := writeDiagnosticArtifactIndex(dir)
	if err != nil {
		return DiagnosticRunReport{}, paths, err
	}
	paths.ArtifactIndex, paths.ArtifactIndexSHA256 = indexPath, indexHash
	return report, paths, nil
}

func buildDiagnosticAttemptAudit(runID string, repetition int, event DiagnosticEvent, attempt Attempt, trace ProviderTrace, resolver assetresolution.Resolver) DiagnosticAttemptAudit {
	var parsed *StructuredResult
	var resolution *PolicyResolution
	if attempt.ValidationStatus == "accepted" {
		parsed, resolution, _ = ParseAndValidate(trace.Content, event.Input, resolver)
	}
	return DiagnosticAttemptAudit{
		RunID: runID, Repetition: repetition, CaseID: event.ID, Category: event.Category,
		AttemptNumber: attempt.AttemptNumber, InputFingerprint: attempt.InputFingerprint,
		Provider: attempt.Provider, ConfiguredModel: attempt.Model, ModelReportedIdentifier: attempt.ModelReportedIdentifier,
		PromptVersion: attempt.PromptVersion, OutputContract: attempt.SchemaVersion, PolicyVersion: resolver.Rules.Version,
		Seed: attempt.Seed, Temperature: attempt.Temperature, RequestTimestamp: attempt.RequestTimestamp,
		ResponseTimestamp: attempt.ResponseTimestamp, DurationMS: attempt.Duration.Milliseconds(),
		RawResponseHash: attempt.RawResponseHash, RawResponseBody: trace.Content,
		ValidationStatus: attempt.ValidationStatus, ValidationErrors: nonNilStrings(attempt.ValidationErrors), FailureReason: attempt.FailureReason,
		RequestID: trace.RequestID, ResponseID: trace.ResponseID, ProviderStatus: trace.Status, Usage: trace.Usage,
		ModelClassification: parsed, DeterministicResolution: resolution,
	}
}

func writeExclusiveJSON(path string, value any) (string, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return writeExclusive(path, append(raw, '\n'))
}

func writeExclusive(path string, raw []byte) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create append-only diagnostic audit %s: %w", path, err)
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func copyExclusive(source, destination string) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	_, err = writeExclusive(destination, raw)
	return err
}

func writeDiagnosticArtifactIndex(dir string) (string, string, error) {
	type artifactHash struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	artifacts := []artifactHash{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() == "artifact-index.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(raw)
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifactHash{Path: filepath.ToSlash(relative), SHA256: hex.EncodeToString(digest[:])})
		return nil
	})
	if err != nil {
		return "", "", err
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	path := filepath.Join(dir, "artifact-index.json")
	hash, err := writeExclusiveJSON(path, struct {
		Version   string         `json:"version"`
		Artifacts []artifactHash `json:"artifacts"`
	}{"ai-shadow-issuer-diagnostic-artifact-index-v1", artifacts})
	return path, hash, err
}
