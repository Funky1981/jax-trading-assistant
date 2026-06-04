package chat

import (
	"encoding/json"
	"testing"
)

func TestDecodeAssistantContent_String(t *testing.T) {
	got, err := decodeAssistantContent(json.RawMessage(`"hello"`))
	if err != nil {
		t.Fatalf("decodeAssistantContent returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}

func TestDecodeAssistantContent_Parts(t *testing.T) {
	got, err := decodeAssistantContent(json.RawMessage(`[{"type":"output_text","text":"first"},{"type":"output_text","text":"second"}]`))
	if err != nil {
		t.Fatalf("decodeAssistantContent returned error: %v", err)
	}
	if got != "first\nsecond" {
		t.Fatalf("unexpected joined content: %q", got)
	}
}

func TestDecodeToolCalls(t *testing.T) {
	calls, err := decodeToolCalls([]openAIToolCallResponse{
		{Function: struct {
			Name      string "json:\"name\""
			Arguments string "json:\"arguments\""
		}{
			Name:      "get_signal",
			Arguments: `{"signalId":"abc"}`,
		}},
	})
	if err != nil {
		t.Fatalf("decodeToolCalls returned error: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "get_signal" {
		t.Fatalf("unexpected tool name: %s", calls[0].Name)
	}
	if string(calls[0].Args) != `{"signalId":"abc"}` {
		t.Fatalf("unexpected args: %s", string(calls[0].Args))
	}
}

func TestOpenAIToolDefsMatchesSharedCatalog(t *testing.T) {
	defs := openAIToolDefs()
	if len(defs) != 18 {
		t.Fatalf("expected 18 tool defs, got %d", len(defs))
	}
	if defs[0].Type != "function" {
		t.Fatalf("unexpected type: %s", defs[0].Type)
	}
}

func TestNewOpenAIChatClientFromEnvPrefersAIGateway(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_BASE_URL", "")
	t.Setenv("OPENAI_MODEL", "")
	t.Setenv("AI_GATEWAY_BASE_URL", "http://home-server:4000")
	t.Setenv("AI_GATEWAY_API_KEY", "virtual-key")
	t.Setenv("AI_DEFAULT_MODEL", "local-small")

	client := NewOpenAIChatClientFromEnv()
	if client == nil {
		t.Fatal("expected gateway-backed client")
	}
	if client.baseURL != "http://home-server:4000" || client.apiKey != "virtual-key" || client.model != "local-small" {
		t.Fatalf("unexpected client config: %#v", client)
	}
}

func TestNewOpenAIChatClientFromEnvBlocksDirectProviderByDefault(t *testing.T) {
	t.Setenv("AI_GATEWAY_BASE_URL", "")
	t.Setenv("AI_GATEWAY_API_KEY", "")
	t.Setenv("AI_ALLOW_DIRECT_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "direct-key")
	t.Setenv("OPENAI_BASE_URL", "")

	if client := NewOpenAIChatClientFromEnv(); client != nil {
		t.Fatalf("expected direct provider to be blocked by default, got %#v", client)
	}
}

func TestNewOpenAIChatClientFromEnvAllowsDirectProviderWhenExplicit(t *testing.T) {
	t.Setenv("AI_GATEWAY_BASE_URL", "")
	t.Setenv("AI_GATEWAY_API_KEY", "")
	t.Setenv("AI_ALLOW_DIRECT_PROVIDER", "true")
	t.Setenv("OPENAI_API_KEY", "direct-key")
	t.Setenv("OPENAI_BASE_URL", "https://api.openai.com")
	t.Setenv("OPENAI_MODEL", "gpt-test")

	client := NewOpenAIChatClientFromEnv()
	if client == nil {
		t.Fatal("expected explicitly allowed direct provider client")
	}
	if client.baseURL != "https://api.openai.com" || client.model != "gpt-test" {
		t.Fatalf("unexpected direct provider client config: %#v", client)
	}
}
