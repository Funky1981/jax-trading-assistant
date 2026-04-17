package main

import (
	"testing"

	"jax-trading-assistant/libs/pgmemory"
	"jax-trading-assistant/libs/runtimepolicy"
)

func TestValidateEmbeddingModePolicy_AllowsLocalInDevAndTest(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "dev", mode: "dev"},
		{name: "test", mode: "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEmbeddingModePolicy(runtimepolicy.Mode(tt.mode), "local"); err != nil {
				t.Fatalf("expected local provider to be allowed in %s mode: %v", tt.mode, err)
			}
		})
	}
}

func TestValidateEmbeddingModePolicy_RejectsOpenAIInDevAndTest(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "dev", mode: "dev"},
		{name: "test", mode: "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEmbeddingModePolicy(runtimepolicy.Mode(tt.mode), "openai"); err == nil {
				t.Fatalf("expected openai provider to be rejected in %s mode", tt.mode)
			}
		})
	}
}

func TestValidateEmbeddingModePolicy_AllowsOpenAIOutsideDevAndTest(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "research", mode: "research"},
		{name: "paper", mode: "paper"},
		{name: "live", mode: "live"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEmbeddingModePolicy(runtimepolicy.Mode(tt.mode), "openai"); err != nil {
				t.Fatalf("expected openai provider to be allowed in %s mode: %v", tt.mode, err)
			}
		})
	}
}

func TestValidateEmbeddingModePolicy_RejectsUnknownProvider(t *testing.T) {
	err := validateEmbeddingModePolicy(runtimepolicy.ModeDev, "invalid")
	if err == nil {
		t.Fatal("expected invalid provider error")
	}
}

func TestBuildMemoryStore_LocalProviderDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("EMBEDDING_PROVIDER", "local")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("EMBEDDING_MODEL", "")

	store, provider, err := buildMemoryStore(nil)
	if err != nil {
		t.Fatalf("expected local buildMemoryStore to succeed, got %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if provider != "local" {
		t.Fatalf("expected local provider, got %q", provider)
	}
}

func TestBuildMemoryStore_OpenAIProviderRequiresAPIKey(t *testing.T) {
	t.Setenv("EMBEDDING_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("EMBEDDING_MODEL", "")

	store, provider, err := buildMemoryStore(nil)
	if err == nil {
		t.Fatalf("expected buildMemoryStore to fail without OPENAI_API_KEY, got store=%v provider=%v", store, provider)
	}
}

func TestBuildMemoryStore_RejectsUnknownProvider(t *testing.T) {
	t.Setenv("EMBEDDING_PROVIDER", "invalid")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("EMBEDDING_MODEL", "")

	store, provider, err := buildMemoryStore(nil)
	if err == nil {
		t.Fatalf("expected buildMemoryStore to fail for invalid provider, got store=%v provider=%v", store, provider)
	}
}

func TestRequiredMemorySchemaVersionIsShared(t *testing.T) {
	if pgmemory.RequiredSchemaVersion < 21 {
		t.Fatalf("required memory schema version regressed: %d", pgmemory.RequiredSchemaVersion)
	}
}
