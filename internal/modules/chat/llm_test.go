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
