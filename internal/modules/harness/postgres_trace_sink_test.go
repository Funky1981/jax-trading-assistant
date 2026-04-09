package harness

import (
	"encoding/json"
	"testing"
)

func TestRedactTraceScrubsSensitiveFields(t *testing.T) {
	trace := Trace{
		Question:    "Authorization: Bearer secret-token",
		FinalAnswer: "token=abc123",
		ToolRuns: []ToolRun{
			{
				Call: ToolCall{
					Name: "lookup",
					Args: json.RawMessage(`{"api_key":"secret","symbol":"AAPL"}`),
				},
				Result: json.RawMessage(`{"password":"hunter2","ok":true}`),
			},
		},
	}

	redacted := redactTrace(trace)
	if redacted.Question != "[redacted]" {
		t.Fatalf("expected question redacted, got %q", redacted.Question)
	}
	if redacted.FinalAnswer != "[redacted]" {
		t.Fatalf("expected final answer redacted, got %q", redacted.FinalAnswer)
	}
	if string(redacted.ToolRuns[0].Call.Args) != `{"api_key":"[redacted]","symbol":"AAPL"}` {
		t.Fatalf("unexpected redacted args: %s", string(redacted.ToolRuns[0].Call.Args))
	}
	if string(redacted.ToolRuns[0].Result) != `{"ok":true,"password":"[redacted]"}` {
		t.Fatalf("unexpected redacted result: %s", string(redacted.ToolRuns[0].Result))
	}
}
