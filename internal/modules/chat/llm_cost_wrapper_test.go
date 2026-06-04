package chat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/internal/modules/llmcontext"
)

func TestCostManagedLLMClientLogsAndReturnsUnderlyingToolCalls(t *testing.T) {
	base := &recordingLLMClient{
		reply: "answer",
		calls: []harness.ToolCall{{
			Name: "get_signal",
			Args: json.RawMessage(`{"signalId":"sig-1"}`),
		}},
	}
	logger := llmcontext.NewMemoryUsageLogger()
	client := NewCostManagedLLMClient(base, CostManagedConfig{
		Logger: logger,
		Limits: llmcontext.BudgetLimits{PerCallUSD: 1},
	})

	reply, calls, err := client.Complete(context.Background(), []LLMMessage{
		{Role: "system", Content: "static rules"},
		{Role: "user", Content: "show signal"},
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if reply != "answer" || len(calls) != 1 || calls[0].Name != "get_signal" {
		t.Fatalf("unexpected reply/calls: reply=%q calls=%#v", reply, calls)
	}
	if !seenMessageContent(base.seen, "show signal") {
		t.Fatalf("underlying client did not receive user content: %#v", base.seen)
	}
	records := logger.Records()
	if len(records) != 1 {
		t.Fatalf("expected usage record, got %d", len(records))
	}
	if records[0].Blocked {
		t.Fatalf("expected allowed usage record, got %#v", records[0])
	}
}

func TestCostManagedLLMClientBlocksBeforeUnderlyingProvider(t *testing.T) {
	base := &recordingLLMClient{reply: "answer"}
	logger := llmcontext.NewMemoryUsageLogger()
	client := NewCostManagedLLMClient(base, CostManagedConfig{
		Logger:    logger,
		Limits:    llmcontext.BudgetLimits{PerCallUSD: 0.000001},
		Estimator: expensiveEstimator{},
	})

	_, _, err := client.Complete(context.Background(), []LLMMessage{
		{Role: "user", Content: "expensive request"},
	})
	if err == nil {
		t.Fatal("expected budget block")
	}
	if len(base.seen) != 0 {
		t.Fatalf("underlying provider was called despite budget block: %#v", base.seen)
	}
	records := logger.Records()
	if len(records) != 1 || !records[0].Blocked {
		t.Fatalf("expected blocked usage record, got %#v", records)
	}
}

type recordingLLMClient struct {
	reply string
	calls []harness.ToolCall
	seen  []LLMMessage
}

func (c *recordingLLMClient) Complete(_ context.Context, msgs []LLMMessage) (string, []harness.ToolCall, error) {
	c.seen = append([]LLMMessage(nil), msgs...)
	return c.reply, c.calls, nil
}

type expensiveEstimator struct{}

func (expensiveEstimator) Estimate(string) int {
	return 1_000_000
}

func seenMessageContent(msgs []LLMMessage, want string) bool {
	for _, msg := range msgs {
		if strings.Contains(msg.Content, want) {
			return true
		}
	}
	return false
}
