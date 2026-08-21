package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/aishadow"
)

type commandR3MockProvider struct {
	config  aishadow.OpenAIDiagnosticConfig
	content string
	calls   int
}

func (p *commandR3MockProvider) Complete(aishadow.ProviderRequest) (aishadow.ProviderResponse, error) {
	p.calls++
	return aishadow.ProviderResponse{
		Content: p.content, ModelIdentifier: p.config.Runtime.Model,
		RequestID: fmt.Sprintf("req-r3-offline-%02d", p.calls), ResponseID: fmt.Sprintf("resp-r3-offline-%02d", p.calls), Status: "completed",
	}, nil
}

func (p *commandR3MockProvider) ExperimentSnapshot() aishadow.HostedExperimentSnapshot {
	inputPrice, cachedPrice, cacheWritePrice, outputPrice := "0.20", "0.02", "0.25", "1.20"
	if p.config.Runtime.Model == aishadow.OpenAIDiagnosticTerraModel {
		inputPrice, cachedPrice, cacheWritePrice, outputPrice = "2.00", "0.20", "2.50", "12.00"
	}
	return aishadow.HostedExperimentSnapshot{
		ExperimentID: p.config.ExperimentID, Provider: p.config.Runtime.Provider, RequestedModel: p.config.Runtime.Model,
		ReasoningEffort: p.config.ReasoningEffort, ServiceTier: p.config.ServiceTier(), StructuredOutputMode: p.config.StructuredOutputMode(),
		MaxOutputTokensPerRequest: p.config.MaxOutputTokens, BudgetCeilingUSD: "0.30", RequestCount: p.calls,
		Pricing: aishadow.HostedPricingPlan{
			InputUSDPerMillionTokens: inputPrice, CachedInputUSDPerMillionTokens: cachedPrice, CacheWriteUSDPerMillionTokens: cacheWritePrice,
			OutputUSDPerMillionTokens: outputPrice, Source: aishadow.OpenAIDiagnosticPricingSource,
		},
	}
}

func TestPreflightDoesNotInspectOrInvokeOllama(t *testing.T) {
	values := map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": "ollama", "JAX_AI_MODEL": "ministral-3:8b",
		"JAX_AI_BASE_URL": "http://localhost:11434", "JAX_AI_TIMEOUT_SECONDS": "120",
		"JAX_AI_TEMPERATURE": "0", "JAX_AI_SEED": "20260803", "JAX_AI_MAX_EVENTS": "48",
		"JAX_RUNTIME_MODE": "paper", "ALLOW_LIVE_TRADING": "false", "EXECUTION_ENABLED": "false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED": "false", "BROKER_EXECUTION_ALLOWED": "false", "MAX_LEVERAGE": "1",
	}
	lookup := func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	inspectCalls := 0
	providerCalls := 0
	deps := dependencies{
		lookup: lookup,
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			inspectCalls++
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider {
			providerCalls++
			return nil
		},
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", t.TempDir(),
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if inspectCalls != 0 || providerCalls != 0 {
		t.Fatalf("preflight contacted model dependencies: inspect=%d provider=%d", inspectCalls, providerCalls)
	}
	if !strings.Contains(output.String(), `"ollama_contact": false`) || !strings.Contains(output.String(), `"events": 48`) || !strings.Contains(output.String(), `"repetitions": 3`) {
		t.Fatalf("unexpected preflight output: %s", output.String())
	}
}

func TestHostedPreflightMakesZeroHTTPCallsAndNeverEmitsCredential(t *testing.T) {
	values := hostedCommandValues()
	lookup := func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	inspectCalls := 0
	providerCalls := 0
	deps := dependencies{
		lookup: lookup,
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			inspectCalls++
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider {
			providerCalls++
			return nil
		},
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", outputRoot,
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if inspectCalls != 0 || providerCalls != 0 {
		t.Fatalf("hosted preflight contacted provider dependencies: inspect=%d provider=%d", inspectCalls, providerCalls)
	}
	for _, want := range []string{`"provider_contact": false`, `"provider": "openai"`, `"model": "gpt-5.6-sol"`, `"inference": false`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("hosted preflight output missing %s: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), values[aishadow.OpenAIDiagnosticAPIKeyEnv]) {
		t.Fatalf("credential leaked in command output: %s", output.String())
	}
	err = filepath.Walk(outputRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), values[aishadow.OpenAIDiagnosticAPIKeyEnv]) || strings.Contains(strings.ToLower(string(raw)), "authorization") {
			t.Fatalf("credential material leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLunaOneRepetitionPreflightReportsReadinessWithZeroProviderCalls(t *testing.T) {
	values := hostedCommandValues()
	values["JAX_AI_MODEL"] = aishadow.OpenAIDiagnosticLunaModel
	values[aishadow.DiagnosticRepetitionsEnv] = "1"
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.12"
	values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
	values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.02"
	values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "0.25"
	values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "1.20"
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			t.Fatal("Luna preflight inspected an Ollama model")
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider {
			t.Fatal("Luna preflight constructed an Ollama provider")
			return nil
		},
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", outputRoot,
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("Luna preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{
		`"status": "ready"`, `"provider": "openai"`, `"model": "gpt-5.6-luna"`, `"requested_model": "gpt-5.6-luna"`,
		`"reasoning": "none"`, `"requested_repetitions": 1`, `"effective_repetitions": 1`, `"cases_per_repetition": 48`,
		`"total_planned_cases": 48`, `"provider_contact": false`, `"inference": false`, `"api_key_present": true`,
		`"database_mutation": false`, `"trading_mutation": false`, `"manifest_file_sha256"`, `"fingerprint_lock_fingerprint"`,
		`"budget_configuration"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("Luna preflight output missing %s: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), values[aishadow.OpenAIDiagnosticAPIKeyEnv]) {
		t.Fatalf("Luna credential leaked in preflight output: %s", output.String())
	}
}

func TestLunaStructuredOutputsPreflightReportsContractAndMakesZeroProviderCalls(t *testing.T) {
	values := hostedStructuredOutputsCommandValues()
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			t.Fatal("C1B preflight inspected an Ollama model")
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider {
			t.Fatal("C1B preflight constructed an Ollama provider")
			return nil
		},
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", outputRoot,
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("C1B preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{
		`"status": "ready"`, `"provider": "openai"`, `"model": "gpt-5.6-luna"`, `"reasoning": "none"`,
		`"experiment_id": "WP-00.03C1B"`, `"cell_identity": "WP-00.03C1B"`,
		`"evidence_namespace": "openai-hosted-c1b-structured-outputs-v1/WP-00.03C1B"`,
		`"structured_outputs": true`, `"schema_contract": "ai-shadow-output-v4-issuer-resolution"`,
		`"contract_enforcement": "openai-responses-json-schema-strict"`, `"schema_sha256"`,
		`"service_tier": "default"`,
		`"requested_repetitions": 1`, `"effective_repetitions": 1`, `"cases_per_repetition": 48`, `"total_planned_cases": 48`,
		`"provider_contact": false`, `"inference": false`, `"api_key_present": true`,
		`"database_mutation": false`, `"trading_mutation": false`,
		`"ceiling_usd": "0.20"`, `"input_usd_per_million_tokens": "0.20"`, `"cached_input_usd_per_million_tokens": "0.02"`,
		`"cache_write_usd_per_million_tokens": "0.25"`, `"output_usd_per_million_tokens": "1.20"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("C1B preflight output missing %s: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), values[aishadow.OpenAIDiagnosticAPIKeyEnv]) {
		t.Fatalf("C1B credential leaked in preflight output: %s", output.String())
	}
	err = filepath.Walk(outputRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), values[aishadow.OpenAIDiagnosticAPIKeyEnv]) || strings.Contains(strings.ToLower(string(raw)), "authorization") {
			t.Fatalf("C1B credential material leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHoldoutPreflightsRequireNoCredentialAndConstructNoProvider(t *testing.T) {
	tests := []struct {
		profileID         string
		experimentID      string
		evidenceNamespace string
		manifest          string
		lock              string
		freeze            string
		cases             string
		budget            string
	}{
		{
			profileID: aishadow.DiagnosticProfileGeneralization, experimentID: aishadow.OpenAIGeneralizationExperimentID,
			evidenceNamespace: aishadow.OpenAIGeneralizationEvidenceNamespace, manifest: "ai-shadow-issuer-generalization-holdout-v1.json",
			lock: "ai-shadow-issuer-generalization-holdout-input-fingerprints-v1.json", freeze: "ai-shadow-issuer-generalization-holdout-freeze-v1.json",
			cases: "48", budget: "0.20",
		},
		{
			profileID: aishadow.DiagnosticProfileBoundary, experimentID: aishadow.OpenAIBoundaryExperimentID,
			evidenceNamespace: aishadow.OpenAIBoundaryEvidenceNamespace, manifest: "ai-shadow-issuer-boundary-challenge-v1.json",
			lock: "ai-shadow-issuer-boundary-challenge-input-fingerprints-v1.json", freeze: "ai-shadow-issuer-boundary-challenge-freeze-v1.json",
			cases: "24", budget: "0.10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.profileID, func(t *testing.T) {
			values := hostedStructuredOutputsCommandValues()
			delete(values, aishadow.OpenAIDiagnosticAPIKeyEnv)
			values["JAX_AI_EXPERIMENT_ID"] = tt.experimentID
			values["JAX_AI_MAX_EVENTS"] = tt.cases
			values["JAX_AI_EXPERIMENT_BUDGET_USD"] = tt.budget
			providerCalls := 0
			deps := dependencies{
				lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
					t.Fatal("holdout preflight inspected an Ollama model")
					return aishadow.DiagnosticModelIdentity{}, nil
				},
				ollamaProvider: func(aishadow.Config) aishadow.Provider {
					t.Fatal("holdout preflight constructed an Ollama provider")
					return nil
				},
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
					providerCalls++
					return nil
				},
			}
			root := filepath.Join("..", "..")
			outputRoot := t.TempDir()
			var output bytes.Buffer
			err := run([]string{
				"--preflight", "--evaluation-profile", tt.profileID,
				"--manifest", filepath.Join(root, "config", tt.manifest),
				"--fingerprint-lock", filepath.Join(root, "config", tt.lock),
				"--freeze", filepath.Join(root, "config", tt.freeze),
				"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
				"--output-root", outputRoot,
			}, &output, deps)
			if err != nil {
				t.Fatal(err)
			}
			if providerCalls != 0 {
				t.Fatalf("holdout preflight constructed provider %d times", providerCalls)
			}
			var result struct {
				Status        string                        `json:"status"`
				Profile       string                        `json:"evaluation_profile"`
				Dataset       string                        `json:"dataset_identity"`
				Guard         string                        `json:"causal_consistency_policy"`
				Inference     bool                          `json:"inference"`
				ProviderTouch bool                          `json:"provider_contact"`
				APIKeyPresent bool                          `json:"api_key_present"`
				Audit         aishadow.DiagnosticAuditPaths `json:"audit"`
				AuditSHA256   string                        `json:"audit_sha256"`
			}
			if err := json.Unmarshal(output.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Status != "ready" || result.Profile != tt.profileID || result.Dataset != tt.profileID ||
				result.Guard != aishadow.CausalConsistencyPolicyVersion || result.Inference || result.ProviderTouch || result.APIKeyPresent ||
				result.Audit.RunID == "" || result.AuditSHA256 == "" {
				t.Fatalf("incomplete credentialless holdout preflight: %+v", result)
			}
			if !strings.Contains(filepath.ToSlash(result.Audit.Preflight), tt.evidenceNamespace+"/"+tt.experimentID+"/preflight/") {
				t.Fatalf("preflight evidence namespace is not isolated: %+v", result.Audit)
			}
			raw, err := os.ReadFile(result.Audit.Preflight)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(raw), `"cases_per_repetition": `+tt.cases) ||
				!strings.Contains(string(raw), `"database_mutation_allowed": false`) ||
				!strings.Contains(string(raw), `"trading_state_mutation_allowed": false`) {
				t.Fatalf("preflight evidence lacks frozen shape or safety facts: %s", raw)
			}
		})
	}
}

func TestHostedExecutionRequiresSeparateAuthorizationBeforeProviderCreation(t *testing.T) {
	values := hostedCommandValues()
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider { return nil },
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	err := run([]string{
		"--execute",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", t.TempDir(),
	}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), aishadow.OpenAIDiagnosticInferenceAuthEnv) {
		t.Fatalf("unauthorized hosted execution was not rejected: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("unauthorized hosted execution created provider %d times", providerCalls)
	}
}

func TestC1E3CommandAuthorizationMatrixAndMockedProviderConstruction(t *testing.T) {
	tests := []struct {
		name             string
		profileID        string
		experimentID     string
		cases            string
		budget           string
		hostedAuthorized bool
		operatorOptIn    bool
		wantProvider     int
		wantError        string
	}{
		{name: "default deny", profileID: aishadow.DiagnosticProfileGeneralizationV2, experimentID: aishadow.OpenAIC1E3GeneralizationExperimentID, cases: "48", budget: "0.30", wantError: "--authorize-c1e3-execution"},
		{name: "global authorization alone denied", profileID: aishadow.DiagnosticProfileGeneralizationV2, experimentID: aishadow.OpenAIC1E3GeneralizationExperimentID, cases: "48", budget: "0.30", hostedAuthorized: true, wantError: "--authorize-c1e3-execution"},
		{name: "experiment opt-in alone denied", profileID: aishadow.DiagnosticProfileGeneralizationV2, experimentID: aishadow.OpenAIC1E3GeneralizationExperimentID, cases: "48", budget: "0.30", operatorOptIn: true, wantError: aishadow.OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "generalization fully authorized reaches mock construction", profileID: aishadow.DiagnosticProfileGeneralizationV2, experimentID: aishadow.OpenAIC1E3GeneralizationExperimentID, cases: "48", budget: "0.30", hostedAuthorized: true, operatorOptIn: true, wantProvider: 1, wantError: "provider is required"},
		{name: "boundary fully authorized reaches mock construction", profileID: aishadow.DiagnosticProfileBoundaryV2, experimentID: aishadow.OpenAIC1E3BoundaryExperimentID, cases: "32", budget: "0.20", hostedAuthorized: true, operatorOptIn: true, wantProvider: 1, wantError: "provider is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := c1e3CommandValues(tt.experimentID, tt.cases, tt.budget)
			if tt.hostedAuthorized {
				values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
			}
			providerCalls := 0
			deps := dependencies{
				lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
					providerCalls++
					return nil
				},
			}
			root := filepath.Join("..", "..")
			args := []string{
				"--execute", "--evaluation-profile", tt.profileID,
				"--output-root", t.TempDir(),
			}
			args = append(args, c1e3CommandFrozenPathArgs(t, root, tt.profileID)...)
			if tt.operatorOptIn {
				args = append(args, "--authorize-c1e3-execution")
			}
			err := run(args, &bytes.Buffer{}, deps)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("unexpected C1E3 command result: %v", err)
			}
			if providerCalls != tt.wantProvider {
				t.Fatalf("provider construction calls=%d want=%d", providerCalls, tt.wantProvider)
			}
		})
	}
}

func TestC1E3CredentiallessPreflightReportsDefaultDenyWithoutProviderConstruction(t *testing.T) {
	for _, tt := range []struct {
		profileID, experimentID, cases, budget string
	}{
		{aishadow.DiagnosticProfileGeneralizationV2, aishadow.OpenAIC1E3GeneralizationExperimentID, "48", "0.30"},
		{aishadow.DiagnosticProfileBoundaryV2, aishadow.OpenAIC1E3BoundaryExperimentID, "32", "0.20"},
	} {
		values := c1e3CommandValues(tt.experimentID, tt.cases, tt.budget)
		delete(values, aishadow.OpenAIDiagnosticAPIKeyEnv)
		providerCalls := 0
		deps := dependencies{
			lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
			openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
				providerCalls++
				return nil
			},
		}
		var output bytes.Buffer
		root := filepath.Join("..", "..")
		args := []string{"--preflight", "--evaluation-profile", tt.profileID, "--output-root", t.TempDir()}
		args = append(args, c1e3CommandFrozenPathArgs(t, root, tt.profileID)...)
		err := run(args, &output, deps)
		if err != nil {
			t.Fatal(err)
		}
		if providerCalls != 0 {
			t.Fatalf("C1E3 preflight constructed provider %d times", providerCalls)
		}
		for _, want := range []string{
			`"provider_contact": false`, `"inference": false`, `"execution_authorized": false`,
			`"version": "` + aishadow.C1E3ExecutionAuthorizationVersion + `"`, `"api_key_present": false`,
		} {
			if !strings.Contains(output.String(), want) {
				t.Fatalf("C1E3 preflight output missing %s: %s", want, output.String())
			}
		}
	}
}

func TestC1F3CommandAuthorizationMatrixAndMockedProviderConstruction(t *testing.T) {
	tests := []struct {
		name             string
		profileID        string
		experimentID     string
		cases            string
		budget           string
		hostedAuthorized bool
		operatorOptIn    bool
		wantProvider     int
		wantError        string
	}{
		{name: "default deny", profileID: aishadow.C1F3ProfileGeneralization, experimentID: aishadow.OpenAIC1F3GeneralizationExperimentID, cases: "48", budget: "0.30", wantError: "--authorize-c1f3-execution"},
		{name: "global authorization alone denied", profileID: aishadow.C1F3ProfileGeneralization, experimentID: aishadow.OpenAIC1F3GeneralizationExperimentID, cases: "48", budget: "0.30", hostedAuthorized: true, wantError: "--authorize-c1f3-execution"},
		{name: "C1F3 opt-in alone denied", profileID: aishadow.C1F3ProfileGeneralization, experimentID: aishadow.OpenAIC1F3GeneralizationExperimentID, cases: "48", budget: "0.30", operatorOptIn: true, wantError: aishadow.OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "generalization fully authorized reaches mock constructor", profileID: aishadow.C1F3ProfileGeneralization, experimentID: aishadow.OpenAIC1F3GeneralizationExperimentID, cases: "48", budget: "0.30", hostedAuthorized: true, operatorOptIn: true, wantProvider: 1, wantError: "provider is required"},
		{name: "boundary fully authorized reaches mock constructor", profileID: aishadow.C1F3ProfileBoundary, experimentID: aishadow.OpenAIC1F3BoundaryExperimentID, cases: "32", budget: "0.20", hostedAuthorized: true, operatorOptIn: true, wantProvider: 1, wantError: "provider is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := c1f3CommandValues(tt.experimentID, tt.cases, tt.budget)
			if tt.hostedAuthorized {
				values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
			}
			providerCalls := 0
			deps := dependencies{
				lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
					providerCalls++
					return nil
				},
			}
			root := filepath.Join("..", "..")
			args := []string{"--execute", "--evaluation-profile", tt.profileID, "--output-root", t.TempDir()}
			args = append(args, c1f3CommandFrozenPathArgs(t, root, tt.profileID)...)
			if tt.operatorOptIn {
				args = append(args, "--authorize-c1f3-execution")
			}
			err := run(args, &bytes.Buffer{}, deps)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("unexpected C1F3 command result: %v", err)
			}
			if providerCalls != tt.wantProvider {
				t.Fatalf("mock provider constructor calls=%d want=%d", providerCalls, tt.wantProvider)
			}
		})
	}
}

func TestC1F3CredentiallessPreflightReportsDefaultDenyAndZeroProviderConstruction(t *testing.T) {
	for _, tt := range []struct{ profileID, experimentID, cases, budget string }{
		{aishadow.C1F3ProfileGeneralization, aishadow.OpenAIC1F3GeneralizationExperimentID, "48", "0.30"},
		{aishadow.C1F3ProfileBoundary, aishadow.OpenAIC1F3BoundaryExperimentID, "32", "0.20"},
	} {
		values := c1f3CommandValues(tt.experimentID, tt.cases, tt.budget)
		delete(values, aishadow.OpenAIDiagnosticAPIKeyEnv)
		providerCalls := 0
		deps := dependencies{
			lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
			openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
				providerCalls++
				return nil
			},
		}
		root := filepath.Join("..", "..")
		args := []string{"--preflight", "--evaluation-profile", tt.profileID, "--output-root", t.TempDir()}
		args = append(args, c1f3CommandFrozenPathArgs(t, root, tt.profileID)...)
		var output bytes.Buffer
		if err := run(args, &output, deps); err != nil {
			t.Fatal(err)
		}
		if providerCalls != 0 {
			t.Fatalf("C1F3 preflight constructed provider %d times", providerCalls)
		}
		for _, want := range []string{
			`"provider_contact": false`, `"inference": false`, `"execution_authorized": false`,
			`"version": "` + aishadow.C1F3ExecutionAuthorizationVersion + `"`, `"api_key_present": false`,
			`"provider_input_isolated": true`, `"frozen_bindings_valid": true`,
		} {
			if !strings.Contains(output.String(), want) {
				t.Fatalf("C1F3 preflight output missing %s: %s", want, output.String())
			}
		}
	}
}

func TestC1F3FlagCannotAuthorizeHistoricalProfile(t *testing.T) {
	values := hostedStructuredOutputsCommandValues()
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	err := run([]string{
		"--execute", "--authorize-c1f3-execution", "--output-root", t.TempDir(),
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
	}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("C1F3 opt-in authorized a historical profile: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("historical profile constructed provider %d times", providerCalls)
	}
}

func TestC1F3RepeatabilityCommandAuthorizationAndMockedProviderConstruction(t *testing.T) {
	for _, tt := range []struct {
		name             string
		hostedAuthorized bool
		operatorOptIn    bool
		wantProvider     int
		wantError        string
	}{
		{name: "default deny", wantError: "permanently consumed"},
		{name: "hosted authorization alone", hostedAuthorized: true, wantError: "permanently consumed"},
		{name: "repeatability flag alone", operatorOptIn: true, wantError: "permanently consumed"},
		{name: "formerly exact frozen combination", hostedAuthorized: true, operatorOptIn: true, wantError: "permanently consumed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := c1f3CommandValues(aishadow.C1F3RepeatabilityExperimentID, "48", "0.30")
			if tt.hostedAuthorized {
				values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
			}
			providerCalls := 0
			deps := dependencies{
				lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
					providerCalls++
					return nil
				},
			}
			root := filepath.Join("..", "..")
			args := []string{"--execute", "--evaluation-profile", aishadow.C1F3RepeatabilityProfileIdentity, "--output-root", t.TempDir()}
			args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3RepeatabilityProfileIdentity)...)
			if tt.operatorOptIn {
				args = append(args, "--authorize-c1f3-repeatability")
			}
			err := run(args, &bytes.Buffer{}, deps)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("unexpected repeatability command result: %v", err)
			}
			if providerCalls != tt.wantProvider {
				t.Fatalf("mock provider constructor calls=%d want=%d", providerCalls, tt.wantProvider)
			}
		})
	}
}

func TestC1F3RepeatabilityCredentiallessPreflightIsZeroNetworkDefaultDeny(t *testing.T) {
	values := c1f3CommandValues(aishadow.C1F3RepeatabilityExperimentID, "48", "0.30")
	delete(values, aishadow.OpenAIDiagnosticAPIKeyEnv)
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	args := []string{"--preflight", "--evaluation-profile", aishadow.C1F3RepeatabilityProfileIdentity, "--output-root", t.TempDir()}
	args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3RepeatabilityProfileIdentity)...)
	var output bytes.Buffer
	if err := run(args, &output, deps); err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("repeatability preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{
		`"provider_contact": false`, `"inference": false`, `"execution_authorized": false`, `"api_key_present": false`,
		`"version": "` + aishadow.C1F3RepeatabilityExecutionAuthorizationVersion + `"`, `"provider_input_isolated": true`,
		`"provider_input_matches_original_c1f3": true`, `"baseline_binding_valid": true`, `"runtime_safety_valid": true`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("repeatability preflight output missing %s: %s", want, output.String())
		}
	}
}

func TestC1F3RepeatabilityR3CommandRunsFullMockedC1FProjectionOffline(t *testing.T) {
	values := c1f3CommandValues(aishadow.C1F3RepeatabilityR3ExperimentID, "48", "0.30")
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
	providerConstructors := 0
	var provider *commandR3MockProvider
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(config aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerConstructors++
			provider = &commandR3MockProvider{
				config:  config,
				content: `{"market_relevance":"MEDIUM","mapping_status":"UNRESOLVED","direct_issuer":"","proxy_exposure":"NONE","mapping_confidence":"HIGH","expected_horizon":"MULTI_DAY","likely_direction":"UNCLEAR","catalyst_type":"synthetic fixture","reason":"Synthetic offline policy fixture with no provider inference.","missing_evidence":[],"issuer_attributions":[],"principal_proxy_candidates":[]}`,
			}
			return provider
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	args := []string{
		"--execute", "--authorize-c1f3-repeatability-r3", "--evaluation-profile", aishadow.C1F3RepeatabilityR3ProfileIdentity,
		"--output-root", outputRoot,
	}
	args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3RepeatabilityR3ProfileIdentity)...)
	var output bytes.Buffer
	if err := run(args, &output, deps); err != nil {
		t.Fatal(err)
	}
	if providerConstructors != 1 || provider == nil || provider.calls != 48 {
		t.Fatalf("r3 mocked execution shape changed: constructors=%d calls=%d", providerConstructors, provider.calls)
	}
	var completed struct {
		Status    string                        `json:"status"`
		RunID     string                        `json:"run_id"`
		Artifacts aishadow.DiagnosticAuditPaths `json:"artifacts"`
	}
	if err := json.Unmarshal(output.Bytes(), &completed); err != nil {
		t.Fatal(err)
	}
	if completed.Status != "completed" || completed.RunID == "" || completed.Artifacts.ArtifactIndex == "" {
		t.Fatalf("r3 mocked command did not persist completed evidence: %s", output.String())
	}
	attempts, err := filepath.Glob(filepath.Join(completed.Artifacts.Directory, "repetition-01", "*-attempt-01.json"))
	if err != nil || len(attempts) != 48 {
		t.Fatalf("r3 mocked command persisted %d attempts: %v", len(attempts), err)
	}
	raw, err := os.ReadFile(attempts[0])
	if err != nil {
		t.Fatal(err)
	}
	var audit map[string]any
	if err := json.Unmarshal(raw, &audit); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"v5_raw_model_output", "typed_causal_attribution", "causal_attribution_policy_decision", "effective_semantic_mapping", "deterministic_resolution"} {
		if audit[field] == nil {
			t.Fatalf("r3 mocked attempt omitted %s", field)
		}
	}
}

func TestC1F3RepeatabilityR3CommandAuthorizationMatrix(t *testing.T) {
	for _, tt := range []struct {
		name             string
		hostedAuthorized bool
		operatorOptIn    bool
		want             string
	}{
		{name: "default deny", want: "--authorize-c1f3-repeatability-r3"},
		{name: "hosted authorization alone", hostedAuthorized: true, want: "--authorize-c1f3-repeatability-r3"},
		{name: "r3 flag alone", operatorOptIn: true, want: aishadow.OpenAIDiagnosticInferenceAuthEnv + "=true"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := c1f3CommandValues(aishadow.C1F3RepeatabilityR3ExperimentID, "48", "0.30")
			if tt.hostedAuthorized {
				values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
			}
			providerConstructors := 0
			deps := dependencies{
				lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
					providerConstructors++
					return nil
				},
			}
			root := filepath.Join("..", "..")
			args := []string{"--execute", "--evaluation-profile", aishadow.C1F3RepeatabilityR3ProfileIdentity, "--output-root", t.TempDir()}
			args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3RepeatabilityR3ProfileIdentity)...)
			if tt.operatorOptIn {
				args = append(args, "--authorize-c1f3-repeatability-r3")
			}
			err := run(args, &bytes.Buffer{}, deps)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("unexpected r3 command authorization result: %v", err)
			}
			if providerConstructors != 0 {
				t.Fatalf("default-denied r3 command constructed provider %d times", providerConstructors)
			}
		})
	}
}

func terraChallengerCommandValues() map[string]string {
	values := c1f3CommandValues(aishadow.C1F3TerraChallengerExperimentID, "48", "0.30")
	values["JAX_AI_MODEL"] = aishadow.OpenAIDiagnosticTerraModel
	values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "2.00"
	values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
	values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "2.50"
	values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "12.00"
	return values
}

func TestC1F3TerraChallengerCommandPreflightIsZeroNetworkDefaultDeny(t *testing.T) {
	values := terraChallengerCommandValues()
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	args := []string{"--preflight", "--evaluation-profile", aishadow.C1F3TerraChallengerProfileIdentity, "--output-root", t.TempDir()}
	args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3TerraChallengerProfileIdentity)...)
	var output bytes.Buffer
	if err := run(args, &output, deps); err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("Terra preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{
		`"provider_contact": false`, `"inference": false`, `"execution_authorized": false`,
		`"evaluation_profile": "` + aishadow.C1F3TerraChallengerProfileIdentity + `"`,
		`"model": "` + aishadow.OpenAIDiagnosticTerraModel + `"`, `"reasoning": "none"`,
		`"cases_per_repetition": 48`, `"repetitions": 1`, `"boundary_excluded": true`,
		`"luna_artifacts_isolated": true`, `"runtime_mode": "paper"`,
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("Terra preflight output missing %s: %s", want, output.String())
		}
	}
}

func TestC1F3TerraChallengerCommandAuthorizationIsDistinct(t *testing.T) {
	for _, tt := range []struct {
		name      string
		hosted    bool
		terraFlag bool
		lunaFlag  bool
		want      string
	}{
		{name: "default deny", want: "--authorize-c1f3-terra-challenger-t1"},
		{name: "hosted only", hosted: true, want: "--authorize-c1f3-terra-challenger-t1"},
		{name: "Terra flag only", terraFlag: true, want: aishadow.OpenAIDiagnosticInferenceAuthEnv + "=true"},
		{name: "Luna flag cannot authorize", hosted: true, lunaFlag: true, want: "scoped only"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			values := terraChallengerCommandValues()
			if tt.hosted {
				values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
			}
			providerCalls := 0
			deps := dependencies{
				lookup:         func(key string) (string, bool) { value, ok := values[key]; return value, ok },
				openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider { providerCalls++; return nil },
			}
			root := filepath.Join("..", "..")
			args := []string{"--execute", "--evaluation-profile", aishadow.C1F3TerraChallengerProfileIdentity, "--output-root", t.TempDir()}
			args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3TerraChallengerProfileIdentity)...)
			if tt.terraFlag {
				args = append(args, "--authorize-c1f3-terra-challenger-t1")
			}
			if tt.lunaFlag {
				args = append(args, "--authorize-c1f3-repeatability-r3")
			}
			err := run(args, &bytes.Buffer{}, deps)
			if err == nil || !strings.Contains(err.Error(), tt.want) || providerCalls != 0 {
				t.Fatalf("Terra command authorization did not fail closed: err=%v providers=%d", err, providerCalls)
			}
		})
	}
}

func TestC1F3TerraChallengerCommandRunsOneMockedCellAndScoresOffline(t *testing.T) {
	values := terraChallengerCommandValues()
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
	providerConstructors := 0
	var provider *commandR3MockProvider
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(config aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerConstructors++
			provider = &commandR3MockProvider{
				config:  config,
				content: `{"market_relevance":"MEDIUM","mapping_status":"UNRESOLVED","direct_issuer":"","proxy_exposure":"NONE","mapping_confidence":"HIGH","expected_horizon":"MULTI_DAY","likely_direction":"UNCLEAR","catalyst_type":"synthetic fixture","reason":"Synthetic offline Terra route fixture with no provider inference.","missing_evidence":[],"issuer_attributions":[],"principal_proxy_candidates":[]}`,
			}
			return provider
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	args := []string{
		"--execute", "--authorize-c1f3-terra-challenger-t1", "--evaluation-profile", aishadow.C1F3TerraChallengerProfileIdentity,
		"--output-root", outputRoot,
	}
	args = append(args, c1f3CommandFrozenPathArgs(t, root, aishadow.C1F3TerraChallengerProfileIdentity)...)
	var output bytes.Buffer
	if err := run(args, &output, deps); err != nil {
		t.Fatal(err)
	}
	if providerConstructors != 1 || provider == nil || provider.calls != 48 {
		t.Fatalf("Terra mocked execution shape changed: constructors=%d calls=%d", providerConstructors, provider.calls)
	}
	var completed struct {
		RunID     string                        `json:"run_id"`
		Artifacts aishadow.DiagnosticAuditPaths `json:"artifacts"`
	}
	if err := json.Unmarshal(output.Bytes(), &completed); err != nil {
		t.Fatal(err)
	}
	if completed.RunID == "" || completed.Artifacts.ArtifactIndexSHA256 == "" {
		t.Fatalf("Terra mocked execution did not persist complete evidence: %s", output.String())
	}
	oldAuth := os.Getenv(aishadow.OpenAIDiagnosticInferenceAuthEnv)
	if err := os.Setenv(aishadow.OpenAIDiagnosticInferenceAuthEnv, "false"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv(aishadow.OpenAIDiagnosticInferenceAuthEnv, oldAuth) })
	score, err := aishadow.BuildC1F3TerraChallengerScore(root, completed.Artifacts.Directory, completed.Artifacts.ArtifactIndexSHA256)
	if err != nil {
		t.Fatal(err)
	}
	if score.TerraRunID != completed.RunID || len(score.Cases) != 48 || score.Disposition != aishadow.C1F3TerraWorse || score.TerraProviderSnapshot == nil || score.TerraProviderSnapshot.RequestCount != 48 {
		t.Fatalf("mocked Terra score is incomplete: %+v", score)
	}
}

func TestC1E3OptInCannotAuthorizeAnotherHostedProfile(t *testing.T) {
	values := hostedStructuredOutputsCommandValues()
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "true"
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	err := run([]string{
		"--execute", "--authorize-c1e3-execution", "--output-root", t.TempDir(),
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
	}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), "scoped only") {
		t.Fatalf("C1E3 opt-in authorized another hosted profile: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("wrong profile constructed provider %d times", providerCalls)
	}
}

func TestHostedCommandMissingAPIKeyFailsClosed(t *testing.T) {
	values := hostedCommandValues()
	delete(values, aishadow.OpenAIDiagnosticAPIKeyEnv)
	deps := dependencies{lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok }}
	err := run([]string{"--preflight"}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), aishadow.OpenAIDiagnosticAPIKeyEnv) {
		t.Fatalf("missing hosted API key was not rejected: %v", err)
	}
}

func TestDeepSeekPreflightMakesZeroHTTPCallsAndNeverEmitsCredential(t *testing.T) {
	values := deepSeekCommandValues()
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		inspectModel: func(aishadow.Config) (aishadow.DiagnosticModelIdentity, error) {
			t.Fatal("DeepSeek preflight inspected an Ollama model")
			return aishadow.DiagnosticModelIdentity{}, nil
		},
		ollamaProvider: func(aishadow.Config) aishadow.Provider {
			t.Fatal("DeepSeek preflight constructed an Ollama provider")
			return nil
		},
		openAIProvider: func(aishadow.OpenAIDiagnosticConfig) aishadow.Provider {
			t.Fatal("DeepSeek preflight constructed an OpenAI provider")
			return nil
		},
		deepSeekProvider: func(aishadow.DeepSeekDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	outputRoot := t.TempDir()
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", outputRoot,
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("DeepSeek preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{`"provider_contact": false`, `"provider": "deepseek"`, `"model": "deepseek-v4-pro"`, `"inference": false`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("DeepSeek preflight output missing %s: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), values[aishadow.DeepSeekDiagnosticAPIKeyEnv]) {
		t.Fatalf("DeepSeek credential leaked in command output: %s", output.String())
	}
	err = filepath.Walk(outputRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return walkErr
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(raw), values[aishadow.DeepSeekDiagnosticAPIKeyEnv]) || strings.Contains(strings.ToLower(string(raw)), "authorization") {
			t.Fatalf("DeepSeek credential material leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeepSeekOneRepetitionPreflightMakesZeroProviderCalls(t *testing.T) {
	values := deepSeekCommandValues()
	values[aishadow.DiagnosticRepetitionsEnv] = "1"
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		deepSeekProvider: func(aishadow.DeepSeekDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	var output bytes.Buffer
	err := run([]string{
		"--preflight",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", t.TempDir(),
	}, &output, deps)
	if err != nil {
		t.Fatal(err)
	}
	if providerCalls != 0 {
		t.Fatalf("one-repetition preflight constructed provider %d times", providerCalls)
	}
	for _, want := range []string{`"requested_repetitions": 1`, `"effective_repetitions": 1`, `"total_planned_cases": 48`, `"inference": false`} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("one-repetition preflight output missing %s: %s", want, output.String())
		}
	}
}

func TestInvalidRepetitionSelectionFailsBeforeProviderConstruction(t *testing.T) {
	values := deepSeekCommandValues()
	values[aishadow.DiagnosticRepetitionsEnv] = "2"
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		deepSeekProvider: func(aishadow.DeepSeekDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	err := run([]string{"--preflight"}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), aishadow.DiagnosticRepetitionsEnv) {
		t.Fatalf("invalid repetition selection was not rejected: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("invalid repetition selection constructed provider %d times", providerCalls)
	}
}

func TestDeepSeekExecutionRequiresSeparateAuthorizationBeforeProviderCreation(t *testing.T) {
	values := deepSeekCommandValues()
	providerCalls := 0
	deps := dependencies{
		lookup: func(key string) (string, bool) { value, ok := values[key]; return value, ok },
		deepSeekProvider: func(aishadow.DeepSeekDiagnosticConfig) aishadow.Provider {
			providerCalls++
			return nil
		},
	}
	root := filepath.Join("..", "..")
	err := run([]string{
		"--execute",
		"--manifest", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-manifest-v1.json"),
		"--fingerprint-lock", filepath.Join(root, "config", "ai-shadow-issuer-diagnostic-input-fingerprints-v1.json"),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
		"--output-root", t.TempDir(),
	}, &bytes.Buffer{}, deps)
	if err == nil || !strings.Contains(err.Error(), aishadow.OpenAIDiagnosticInferenceAuthEnv) {
		t.Fatalf("unauthorized DeepSeek execution was not rejected: %v", err)
	}
	if providerCalls != 0 {
		t.Fatalf("unauthorized DeepSeek execution constructed provider %d times", providerCalls)
	}
}

func hostedCommandValues() map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": "openai", "JAX_AI_MODEL": aishadow.OpenAIDiagnosticModel,
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_MAX_EVENTS": "48", aishadow.OpenAIDiagnosticAPIKeyEnv: "sk-test-only-do-not-use",
		"JAX_AI_EXPERIMENT_ID": "A1", "JAX_AI_REASONING_EFFORT": "none", "JAX_AI_MAX_OUTPUT_TOKENS": "256",
		"JAX_AI_EXPERIMENT_BUDGET_USD": "1.00", "JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS": "5.00",
		"JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.50", "JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS": "6.25",
		"JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS": "30.00", aishadow.OpenAIDiagnosticInferenceAuthEnv: "false",
		"JAX_RUNTIME_MODE": "paper", "ALLOW_LIVE_TRADING": "false", "EXECUTION_ENABLED": "false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED": "false", "BROKER_EXECUTION_ALLOWED": "false", "MAX_LEVERAGE": "1",
	}
}

func hostedStructuredOutputsCommandValues() map[string]string {
	values := hostedCommandValues()
	values["JAX_AI_MODEL"] = aishadow.OpenAIDiagnosticLunaModel
	values["JAX_AI_EXPERIMENT_ID"] = aishadow.OpenAIStructuredOutputsExperimentID
	values[aishadow.OpenAIDiagnosticContractModeEnv] = aishadow.OpenAIStructuredOutputsMode
	values[aishadow.DiagnosticRepetitionsEnv] = "1"
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = "0.20"
	values["JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.20"
	values["JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS"] = "0.02"
	values["JAX_AI_CACHE_WRITE_PRICE_USD_PER_MILLION_TOKENS"] = "0.25"
	values["JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS"] = "1.20"
	return values
}

func c1e3CommandValues(experimentID, cases, budget string) map[string]string {
	values := hostedStructuredOutputsCommandValues()
	values["JAX_AI_EXPERIMENT_ID"] = experimentID
	values["JAX_AI_MAX_EVENTS"] = cases
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = budget
	values[aishadow.DiagnosticRepetitionsEnv] = "1"
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "false"
	return values
}

func c1f3CommandValues(experimentID, cases, budget string) map[string]string {
	values := hostedStructuredOutputsCommandValues()
	values["JAX_AI_EXPERIMENT_ID"] = experimentID
	values["JAX_AI_MAX_EVENTS"] = cases
	values["JAX_AI_EXPERIMENT_BUDGET_USD"] = budget
	values[aishadow.DiagnosticRepetitionsEnv] = "1"
	values[aishadow.OpenAIDiagnosticInferenceAuthEnv] = "false"
	return values
}

func c1e3CommandFrozenPathArgs(t *testing.T, root, profileID string) []string {
	t.Helper()
	profile, err := aishadow.LoadDiagnosticEvaluationProfile(profileID)
	if err != nil {
		t.Fatal(err)
	}
	return []string{
		"--manifest", filepath.Join(root, filepath.FromSlash(profile.ManifestPath)),
		"--fingerprint-lock", filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		"--freeze", filepath.Join(root, filepath.FromSlash(profile.FreezePath)),
		"--typed-labels", filepath.Join(root, filepath.FromSlash(profile.TypedLabelPath)),
		"--scoring-rubric", filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath)),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
	}
}

func c1f3CommandFrozenPathArgs(t *testing.T, root, profileID string) []string {
	t.Helper()
	profile, err := aishadow.LoadDiagnosticExecutionProfile(profileID)
	if err != nil {
		t.Fatal(err)
	}
	return []string{
		"--manifest", filepath.Join(root, filepath.FromSlash(profile.ManifestPath)),
		"--fingerprint-lock", filepath.Join(root, filepath.FromSlash(profile.FingerprintLockPath)),
		"--freeze", filepath.Join(root, filepath.FromSlash(profile.FreezePath)),
		"--typed-labels", filepath.Join(root, filepath.FromSlash(profile.TypedLabelPath)),
		"--scoring-rubric", filepath.Join(root, filepath.FromSlash(profile.ScoringRubricPath)),
		"--asset-ruleset-file", filepath.Join(root, "config", "event-asset-resolution-v1.json"),
	}
}

func deepSeekCommandValues() map[string]string {
	return map[string]string{
		"JAX_AI_SHADOW_ENABLED": "true", "JAX_AI_PROVIDER": "deepseek", "JAX_AI_MODEL": aishadow.DeepSeekDiagnosticModel,
		"JAX_AI_TIMEOUT_SECONDS": "120", "JAX_AI_MAX_EVENTS": "48", aishadow.DeepSeekDiagnosticAPIKeyEnv: "ds-test-only-do-not-use",
		"JAX_AI_EXPERIMENT_ID": "A1", "JAX_AI_REASONING_EFFORT": "none", aishadow.DeepSeekDiagnosticThinkingModeEnv: "disabled",
		"JAX_AI_MAX_OUTPUT_TOKENS": "256", "JAX_AI_EXPERIMENT_BUDGET_USD": "0.10",
		"JAX_AI_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.435", "JAX_AI_CACHED_INPUT_PRICE_USD_PER_MILLION_TOKENS": "0.003625",
		"JAX_AI_OUTPUT_PRICE_USD_PER_MILLION_TOKENS": "0.87", aishadow.OpenAIDiagnosticInferenceAuthEnv: "false",
		"JAX_RUNTIME_MODE": "paper", "ALLOW_LIVE_TRADING": "false", "EXECUTION_ENABLED": "false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED": "false", "BROKER_EXECUTION_ALLOWED": "false", "MAX_LEVERAGE": "1",
	}
}
