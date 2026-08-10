package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/aishadow"
)

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
