package planner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/llmcontext"
)

type fakeModelClient struct {
	pkg      llmcontext.PromptPackage
	response llmcontext.LLMResult
	err      error
}

func (c *fakeModelClient) Complete(_ context.Context, pkg llmcontext.PromptPackage) (llmcontext.LLMResult, error) {
	c.pkg = pkg
	return c.response, c.err
}

func newModelProvider(t *testing.T, response string) (*ModelProvider, *fakeModelClient) {
	t.Helper()
	client := &fakeModelClient{response: llmcontext.LLMResult{
		CorrelationID: "request-123",
		Text:          response,
		InputTokens:   21,
		OutputTokens:  8,
		CachedTokens:  3,
		ActualCostUSD: 0.0012,
	}}
	provider, err := NewModelProvider(client, "litellm", "local-small")
	if err != nil {
		t.Fatal(err)
	}
	return provider, client
}

const validModelJSON = `{"summary":"review context","steps":["inspect evidence","defer execution"],"action":"HOLD","confidence":0.7,"reasoning_notes":"advisory only"}`

func TestModelProviderReturnsValidatedAdvisoryResultAndMetadata(t *testing.T) {
	provider, client := newModelProvider(t, validModelJSON)
	result, err := provider.Generate(context.Background(), Request{Symbol: "AAPL", Context: "bounded context"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "HOLD" || result.Confidence != 0.7 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Inference == nil || result.Inference.Provider != "litellm" || result.Inference.Model != "local-small" || result.Inference.CachedInputTokens != 3 {
		t.Fatalf("missing truthful inference metadata: %+v", result.Inference)
	}
	if client.pkg.Provider != "litellm" || client.pkg.Model != "local-small" || client.pkg.ResponseSchema == "" {
		t.Fatalf("model transport was not configured with provider metadata/schema: %+v", client.pkg)
	}
	if client.pkg.DynamicContext == "" || client.pkg.CacheablePrefix == "" {
		t.Fatal("expected bounded system and request context")
	}
}

func TestModelProviderRejectsMalformedOrUnsafeResponses(t *testing.T) {
	for _, response := range []string{
		`not json`,
		`{"summary":"x","steps":["x"],"action":"APPROVE","confidence":0.5}`,
		`{"summary":"x","steps":["x"],"action":"HOLD","confidence":2}`,
		`{"summary":"x","steps":["x"],"action":"HOLD","confidence":0.5,"unexpected":true}`,
		`{"summary":"x","steps":["x"],"action":"HOLD","confidence":0.5}{"extra":true}`,
	} {
		provider, _ := newModelProvider(t, response)
		if _, err := provider.Generate(context.Background(), Request{Symbol: "AAPL"}); err == nil {
			t.Fatalf("malformed/unsafe response accepted: %s", response)
		}
	}
}

func TestModelProviderFailsClosedOnTransportError(t *testing.T) {
	client := &fakeModelClient{err: errors.New("provider unavailable")}
	provider, err := NewModelProvider(client, "litellm", "local-small")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Generate(context.Background(), Request{Symbol: "AAPL"}); err == nil {
		t.Fatal("provider failure was not returned")
	}
}

func TestModelProviderRejectsUnboundedRequestBeforeTransport(t *testing.T) {
	provider, client := newModelProvider(t, validModelJSON)
	if _, err := provider.Generate(context.Background(), Request{Symbol: "AAPL", Constraints: map[string]any{"large": string(make([]byte, maxRequestBytes))}}); err == nil {
		t.Fatal("unbounded request accepted")
	}
	if client.pkg.Model != "" {
		t.Fatal("transport called for rejected request")
	}
}

func TestModelProviderIdentity(t *testing.T) {
	provider, _ := newModelProvider(t, validModelJSON)
	identity := provider.Identity()
	if identity.Provider != "litellm" || identity.Model != "local-small" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestModelProviderUsesExistingLiteLLMTransportWithoutTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected transport path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		if strings.Contains(string(body), `"tools"`) {
			t.Fatal("planner transport must not send tools")
		}
		w.Header().Set("Content-Type", "application/json")
		response, err := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"message": map[string]string{"content": validModelJSON}}},
			"usage":   map[string]int{"prompt_tokens": 12, "completion_tokens": 7, "total_tokens": 19},
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		_, _ = w.Write(response)
	}))
	defer server.Close()

	client := llmcontext.NewLiteLLMClient(llmcontext.LiteLLMConfig{BaseURL: server.URL, APIKey: "virtual-test-key"})
	provider, err := NewModelProvider(client, "litellm", "local-small")
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Generate(context.Background(), Request{Symbol: "SPY", Context: "bounded"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inference == nil || result.Inference.InputTokens != 12 || result.Inference.OutputTokens != 7 {
		t.Fatalf("transport usage was not retained: %+v", result.Inference)
	}
}
