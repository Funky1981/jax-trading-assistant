package main

import (
	"bytes"
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
		provider: func(aishadow.Config) aishadow.Provider {
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
