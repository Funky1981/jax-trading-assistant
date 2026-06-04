package llmcontext

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLiteLLMClientSendsPromptPackageThroughChatCompletions(t *testing.T) {
	var gotAuth string
	var gotBody struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"summary ok"}}],
			"usage":{"prompt_tokens":12,"completion_tokens":4,"total_tokens":16}
		}`))
	}))
	defer server.Close()

	client := NewLiteLLMClient(LiteLLMConfig{
		BaseURL: server.URL,
		APIKey:  "virtual-key",
		HTTP:    server.Client(),
	})
	result, err := client.Complete(context.Background(), PromptPackage{
		Model:                 "local-small",
		CacheablePrefix:       "static rules",
		RetrievedMemory:       "memory",
		DynamicContext:        "event data",
		ResponseSchema:        `{"type":"object"}`,
		EstimatedOutputTokens: 99,
		CorrelationID:         "corr-litellm",
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if gotAuth != "Bearer virtual-key" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if gotBody.Model != "local-small" || gotBody.MaxTokens != 99 {
		t.Fatalf("unexpected request body: %#v", gotBody)
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Role != "system" || gotBody.Messages[1].Role != "user" {
		t.Fatalf("unexpected messages: %#v", gotBody.Messages)
	}
	if result.Text != "summary ok" || result.InputTokens != 12 || result.OutputTokens != 4 || result.CorrelationID != "corr-litellm" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestLiteLLMClientRequiresVirtualKey(t *testing.T) {
	client := NewLiteLLMClient(LiteLLMConfig{BaseURL: "http://example.test"})
	_, err := client.Complete(context.Background(), PromptPackage{Model: "local-small"})
	if err == nil {
		t.Fatal("expected missing API key to fail")
	}
}
