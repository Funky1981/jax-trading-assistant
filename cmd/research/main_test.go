package main

import "testing"

func TestBuildMemoryStore_RequiresEmbeddingConfig(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("EMBEDDING_MODEL", "")

	store, err := buildMemoryStore(nil)
	if err == nil {
		t.Fatalf("expected buildMemoryStore to fail without OPENAI_API_KEY, got store=%v", store)
	}
}
