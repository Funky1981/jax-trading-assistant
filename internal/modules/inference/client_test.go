package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigFromEnvDefaultsToDeterministic(t *testing.T) {
	cfg, err := ConfigFromEnv(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderDeterministic || cfg.Model != "local-small" {
		t.Fatalf("unexpected default: %+v", cfg)
	}
}

func TestConfigFromEnvRejectsUnknownProvider(t *testing.T) {
	_, err := ConfigFromEnv(func(key string) (string, bool) {
		if key == "JAX_MODEL_PROVIDER" {
			return "unknown", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("unknown provider accepted")
	}
}

func TestOpenAIClientUsesJaxOwnedTransportAndRetainsUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var request Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Tools) != 0 {
			t.Fatal("planner transport must not receive tools")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"req-1","model":"model-actual","choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":12,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":3}}}`))
	}))
	defer server.Close()

	client, err := NewOpenAIClient(Config{Provider: ProviderOpenAI, BaseURL: server.URL, APIKey: "test-key", Model: "model-request"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "bounded"}}, MaxTokens: 10})
	if err != nil {
		t.Fatal(err)
	}
	if response.Provider != ProviderOpenAI || response.Model != "model-actual" || response.RequestID != "req-1" || response.Usage.InputTokens != 12 || response.Usage.OutputTokens != 4 || response.Usage.CachedInputTokens != 3 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestOpenAIClientFailsClosedOnBadResponse(t *testing.T) {
	for _, body := range []string{"not-json", `{"choices":[]}`, `{"choices":[{"message":{"content":""}}]}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		client, err := NewOpenAIClient(Config{Provider: ProviderOpenAI, BaseURL: server.URL, APIKey: "test-key", Model: "test-model"})
		if err != nil {
			server.Close()
			t.Fatal(err)
		}
		if _, err := client.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "x"}}, MaxTokens: 1}); err == nil {
			server.Close()
			t.Fatalf("bad response accepted: %s", body)
		}
		server.Close()
	}
}

func TestOllamaClientUsesLocalEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected Ollama request")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"qwen","message":{"content":"local"},"prompt_eval_count":8,"eval_count":2}`))
	}))
	defer server.Close()
	client, err := NewOllamaClient(Config{Provider: ProviderOllama, BaseURL: server.URL, Model: "qwen"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Complete(context.Background(), Request{Messages: []Message{{Role: "user", Content: "x"}}, MaxTokens: 2})
	if err != nil || response.Provider != ProviderOllama || response.Text != "local" || response.Usage.InputTokens != 8 {
		t.Fatalf("unexpected Ollama response: %+v, err=%v", response, err)
	}
}

func TestRequestValidationRejectsUnboundedOrMalformedRequests(t *testing.T) {
	client := &OpenAIClient{baseURL: "http://invalid", apiKey: "key", model: "model", http: http.DefaultClient}
	for _, request := range []Request{
		{MaxTokens: 1},
		{Messages: []Message{{Role: "user", Content: "x"}}, MaxTokens: 0},
		{Messages: []Message{{Role: "user", Content: strings.Repeat("x", 1<<20+1)}}, MaxTokens: 1},
	} {
		if _, err := client.Complete(context.Background(), request); err == nil {
			t.Fatalf("invalid request accepted: %+v", request)
		}
	}
}
