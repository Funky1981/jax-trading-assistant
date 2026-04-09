package harness

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type scriptedModel struct {
	responses []scriptedResponse
	idx       int
}

type scriptedResponse struct {
	answer string
	calls  []ToolCall
	err    error
}

func (m *scriptedModel) Complete(context.Context, []Message) (string, []ToolCall, error) {
	if m.idx >= len(m.responses) {
		return "", nil, nil
	}
	resp := m.responses[m.idx]
	m.idx++
	return resp.answer, resp.calls, resp.err
}

func TestAnswerWithEvidenceStopsAtMaxSteps(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(ToolDefinition{
		Name:          "lookup",
		ReadOnly:      true,
		EvidenceLevel: EvidenceHardInternal,
		Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	service := NewService(Policy{MaxSteps: 2, MaxToolCalls: 1}, reg, NewPromptBuilder(), nil, nil, &scriptedModel{
		responses: []scriptedResponse{
			{calls: []ToolCall{{Name: "lookup", Args: json.RawMessage(`{}`)}}},
			{calls: []ToolCall{{Name: "lookup", Args: json.RawMessage(`{}`)}}},
		},
	})

	_, _, _, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "max steps") {
		t.Fatalf("expected max steps error, got %v", err)
	}
}

func TestAnswerWithEvidenceCapturesEvidence(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(ToolDefinition{
		Name:                 "lookup",
		ReadOnly:             true,
		EvidenceLevel:        EvidenceHardInternal,
		FreshnessExpectation: "near_real_time",
		Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"id":"123"}`), nil
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	service := NewService(DefaultPolicy(ModeResearch), reg, NewPromptBuilder(), nil, nil, &scriptedModel{
		responses: []scriptedResponse{
			{calls: []ToolCall{{Name: "lookup", Args: json.RawMessage(`{}`)}}},
			{answer: "Based on the data currently in Jax, this appears likely supported."},
		},
	})

	answer, bundle, trace, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err != nil {
		t.Fatalf("AnswerWithEvidence returned error: %v", err)
	}
	if answer == "" {
		t.Fatal("expected non-empty answer")
	}
	if len(bundle.Items) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(bundle.Items))
	}
	if bundle.Items[0].EvidenceLevel != EvidenceHardInternal {
		t.Fatalf("unexpected evidence level: %s", bundle.Items[0].EvidenceLevel)
	}
	if len(trace.ToolRuns) != 1 {
		t.Fatalf("expected 1 traced tool run, got %d", len(trace.ToolRuns))
	}
}

func TestAnswerWithEvidenceRepromptsAfterValidationFailure(t *testing.T) {
	reg := NewRegistry()
	service := NewService(DefaultPolicy(ModeResearch), reg, NewPromptBuilder(), NewValidator(), nil, &scriptedModel{
		responses: []scriptedResponse{
			{answer: "This will definitely happen."},
			{answer: "Based on the data currently in Jax, I cannot verify a confident conclusion."},
		},
	})

	answer, _, trace, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err != nil {
		t.Fatalf("AnswerWithEvidence returned error: %v", err)
	}
	if !strings.Contains(answer, "cannot verify") {
		t.Fatalf("expected corrected answer, got %q", answer)
	}
	if len(trace.ValidationAttempts) != 2 {
		t.Fatalf("expected 2 validation attempts, got %d", len(trace.ValidationAttempts))
	}
	if trace.ValidationAttempts[0].Accepted {
		t.Fatal("expected first validation attempt to fail")
	}
	if !trace.ValidationAttempts[1].Accepted {
		t.Fatal("expected second validation attempt to pass")
	}
}

func TestAnswerWithEvidenceReturnsSafeRefusalAfterDoubleFailure(t *testing.T) {
	reg := NewRegistry()
	service := NewService(DefaultPolicy(ModeResearch), reg, NewPromptBuilder(), NewValidator(), nil, &scriptedModel{
		responses: []scriptedResponse{
			{answer: "I approved the trade and it will definitely work."},
			{answer: "The price target is $150."},
		},
	})

	answer, _, trace, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err != nil {
		t.Fatalf("AnswerWithEvidence returned error: %v", err)
	}
	if !strings.Contains(answer, "can't provide a supported trading conclusion") {
		t.Fatalf("expected safe refusal, got %q", answer)
	}
	if len(trace.ValidationAttempts) != 2 {
		t.Fatalf("expected 2 validation attempts, got %d", len(trace.ValidationAttempts))
	}
	if trace.ValidationAttempts[1].Accepted {
		t.Fatal("expected second validation attempt to fail")
	}
}

func TestAnswerWithEvidenceWritesTraceSinkPayload(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(ToolDefinition{
		Name:                 "lookup",
		ReadOnly:             true,
		EvidenceLevel:        EvidenceHardInternal,
		FreshnessExpectation: "near_real_time",
		Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	sink := &MemoryTraceSink{}
	service := NewService(DefaultPolicy(ModeResearch), reg, NewPromptBuilder(), NewValidator(), sink, &scriptedModel{
		responses: []scriptedResponse{
			{calls: []ToolCall{{Name: "lookup", Args: json.RawMessage(`{}`)}}},
			{answer: "Based on the data currently in Jax, I cannot verify a confident conclusion."},
		},
	})

	_, _, _, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err != nil {
		t.Fatalf("AnswerWithEvidence returned error: %v", err)
	}
	if len(sink.Items) != 1 {
		t.Fatalf("expected one written trace, got %d", len(sink.Items))
	}
	if len(sink.Items[0].ToolRuns) != 1 {
		t.Fatalf("expected one traced tool run, got %d", len(sink.Items[0].ToolRuns))
	}
}

func TestAnswerWithEvidenceEnforcesToolTimeout(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(ToolDefinition{
		Name:          "slow_lookup",
		ReadOnly:      true,
		EvidenceLevel: EvidenceHardInternal,
		Handler: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(50 * time.Millisecond):
				return json.RawMessage(`{"ok":true}`), nil
			}
		},
	}); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	policy := DefaultPolicy(ModeResearch)
	policy.ToolTimeout = 10 * time.Millisecond
	service := NewService(policy, reg, NewPromptBuilder(), nil, nil, &scriptedModel{
		responses: []scriptedResponse{
			{calls: []ToolCall{{Name: "slow_lookup", Args: json.RawMessage(`{}`)}}},
		},
	})

	_, _, trace, err := service.AnswerWithEvidence(context.Background(), "session-1", "question", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if len(trace.ToolRuns) != 1 || !strings.Contains(trace.ToolRuns[0].Error, "timed out") {
		t.Fatalf("expected timed out tool run, got %+v", trace.ToolRuns)
	}
}
