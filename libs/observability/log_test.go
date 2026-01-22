package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogEvent_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	previous := logger.Writer()
	logger.SetOutput(&buf)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})

	ctx := WithRunInfo(context.Background(), RunInfo{
		RunID:  "run-1",
		TaskID: "task-1",
		Symbol: "AAPL",
	})

	LogEvent(ctx, "info", "test_event", map[string]any{
		"input": map[string]any{
			"api_key": "secret",
			"value":   42,
		},
	})

	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		t.Fatal("expected log output")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if payload["event"] != "test_event" {
		t.Fatalf("expected event test_event, got %#v", payload["event"])
	}
	if payload["level"] != "info" {
		t.Fatalf("expected level info, got %#v", payload["level"])
	}
	if payload["run_id"] != "run-1" || payload["task_id"] != "task-1" || payload["symbol"] != "AAPL" {
		t.Fatalf("expected run info fields, got %#v", payload)
	}

	input, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input field to be object, got %#v", payload["input"])
	}
	if input["api_key"] != redactedValue {
		t.Fatalf("expected api_key to be redacted, got %#v", input["api_key"])
	}
}

func TestLogToolStart_RedactsInput(t *testing.T) {
	var buf bytes.Buffer
	previous := logger.Writer()
	logger.SetOutput(&buf)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})

	LogToolStart(context.Background(), "memory", "memory.retain", map[string]any{
		"token": "secret",
		"value": 1,
	})

	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		t.Fatal("expected log output")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	input, ok := payload["input"].(map[string]any)
	if !ok {
		t.Fatalf("expected input field to be object, got %#v", payload["input"])
	}
	if input["token"] != redactedValue {
		t.Fatalf("expected token to be redacted, got %#v", input["token"])
	}
}
