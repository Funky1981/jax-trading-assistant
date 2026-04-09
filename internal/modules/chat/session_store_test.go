package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessageJSONIncludesTraceID(t *testing.T) {
	traceID := "trace-123"
	msg := Message{
		Role:    RoleAssistant,
		Content: "answer",
		TraceID: &traceID,
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	if !strings.Contains(string(raw), `"traceId":"trace-123"`) {
		t.Fatalf("expected traceId in json, got %s", string(raw))
	}
}
