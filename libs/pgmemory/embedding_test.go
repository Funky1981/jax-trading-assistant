package pgmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateEmbedderConfig_LocalDoesNotRequireAPIKey(t *testing.T) {
	err := ValidateEmbedderConfig(EmbedderConfig{Provider: EmbeddingProviderLocal})
	if err != nil {
		t.Fatalf("expected local provider to validate without API key, got %v", err)
	}
}

func TestValidateEmbedderConfig_OpenAIRequiresAPIKey(t *testing.T) {
	err := ValidateEmbedderConfig(EmbedderConfig{Provider: EmbeddingProviderOpenAI})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestValidateEmbedderConfig_RejectsUnknownProvider(t *testing.T) {
	err := ValidateEmbedderConfig(EmbedderConfig{Provider: "invalid"})
	if err == nil {
		t.Fatal("expected invalid provider error")
	}
}

func TestLocalEmbedder_IsDeterministicAndDimensioned(t *testing.T) {
	embedder := NewLocalEmbedder()

	a, err := embedder.Embed(context.Background(), "AAPL MACD crossover")
	if err != nil {
		t.Fatalf("first embed failed: %v", err)
	}
	b, err := embedder.Embed(context.Background(), "AAPL MACD crossover")
	if err != nil {
		t.Fatalf("second embed failed: %v", err)
	}
	if len(a) != EmbeddingDimensions || len(b) != EmbeddingDimensions {
		t.Fatalf("unexpected dimension: %d %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("expected deterministic vector; mismatch at %d", i)
		}
	}
}

func TestLocalEmbedder_DistinguishesDifferentTexts(t *testing.T) {
	embedder := NewLocalEmbedder()

	a, err := embedder.Embed(context.Background(), "AAPL MACD crossover")
	if err != nil {
		t.Fatalf("embed A failed: %v", err)
	}
	b, err := embedder.Embed(context.Background(), "TSLA breakdown momentum")
	if err != nil {
		t.Fatalf("embed B failed: %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("dimension mismatch: %d vs %d", len(a), len(b))
	}

	different := false
	for i := range a {
		if a[i] != b[i] {
			different = true
			break
		}
	}
	if !different {
		t.Fatal("expected distinct texts to produce distinct vectors")
	}
}

func TestOpenAIEmbedder_RejectsUnexpectedDimension(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2]}]}`))
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder(OpenAIEmbedderConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "text-embedding-3-small",
	})

	_, err := embedder.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected dimension validation error")
	}
}

func TestNewEmbedder_OpenAIRejectsMissingAPIKey(t *testing.T) {
	_, err := NewEmbedder(EmbedderConfig{Provider: EmbeddingProviderOpenAI})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestNewEmbedder_LocalBuildsWithoutAPIKey(t *testing.T) {
	embedder, err := NewEmbedder(EmbedderConfig{Provider: EmbeddingProviderLocal})
	if err != nil {
		t.Fatalf("expected local embedder, got error: %v", err)
	}
	if embedder == nil {
		t.Fatal("expected non-nil local embedder")
	}
}

func TestValidateOpenAIEmbedderConfig_RequiresValidBaseURL(t *testing.T) {
	err := ValidateOpenAIEmbedderConfig(OpenAIEmbedderConfig{
		APIKey:  "test-key",
		BaseURL: "://bad-url",
	})
	if err == nil || !strings.Contains(err.Error(), "OPENAI_BASE_URL") {
		t.Fatalf("expected invalid base URL error, got %v", err)
	}
}
