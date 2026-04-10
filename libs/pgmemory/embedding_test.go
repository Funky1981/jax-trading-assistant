package pgmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateOpenAIEmbedderConfig_RequiresAPIKey(t *testing.T) {
	err := ValidateOpenAIEmbedderConfig(OpenAIEmbedderConfig{})
	if err == nil {
		t.Fatal("expected missing API key error")
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
