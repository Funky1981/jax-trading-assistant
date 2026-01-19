package app

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	testfixtures "jax-trading-assistant/libs/testing"
)

type fakeMemory struct {
	lastRecallBank string
	lastRetainBank string
	lastRetainItem contracts.MemoryItem
	recallInvoked  bool
	retainInvoked  bool
}

func (m *fakeMemory) Recall(_ context.Context, bank string, _ contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	m.lastRecallBank = bank
	m.recallInvoked = true
	return []contracts.MemoryItem{
		{
			TS:      time.Now().UTC(),
			Type:    "decision",
			Summary: "prior memory",
			Data:    map[string]any{"ok": true},
			Source:  &contracts.MemorySource{System: "test"},
		},
	}, nil
}

func (m *fakeMemory) Retain(_ context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	m.lastRetainBank = bank
	m.lastRetainItem = item
	m.retainInvoked = true
	return "mem_1", nil
}

type fakeAgent struct {
	lastInput PlanInput
}

func (a *fakeAgent) Plan(_ context.Context, input PlanInput) (PlanResult, error) {
	a.lastInput = input
	return PlanResult{
		Summary:        "Plan summary",
		Steps:          []string{"step1", "step2"},
		Action:         "executed",
		Confidence:     0.7,
		ReasoningNotes: "short notes",
	}, nil
}

type fakeTools struct {
	lastPlan PlanResult
}

func (t *fakeTools) Execute(_ context.Context, plan PlanResult) ([]ToolRun, error) {
	t.lastPlan = plan
	return []ToolRun{{Name: "risk.position_size", Success: true}}, nil
}

func TestOrchestrator_Run_RecallPlanExecuteRetain(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}

	orch := NewOrchestrator(memory, agent, tools)

	var constraints map[string]any
	if err := json.Unmarshal(testfixtures.LoadFixture(t, "orchestrator_constraints.json"), &constraints); err != nil {
		t.Fatalf("constraints fixture: %v", err)
	}

	result, err := orch.Run(context.Background(), OrchestrationRequest{
		Bank:        "trade_decisions",
		Symbol:      "AAPL",
		Strategy:    "earnings_gap_v1",
		Constraints: constraints,
		UserContext: "user constraints",
		Tags:        []string{"earnings", "risk-high"},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !memory.recallInvoked || !memory.retainInvoked {
		t.Fatalf("expected recall and retain to be invoked")
	}
	if memory.lastRecallBank != "trade_decisions" {
		t.Fatalf("expected recall bank trade_decisions, got %q", memory.lastRecallBank)
	}
	if agent.lastInput.Symbol != "AAPL" || agent.lastInput.Context != "user constraints" {
		t.Fatalf("expected agent input context merged")
	}
	if len(agent.lastInput.Memories) == 0 {
		t.Fatalf("expected recalled memories passed to agent")
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "risk.position_size" {
		t.Fatalf("unexpected tools: %#v", result.Tools)
	}
	if memory.lastRetainItem.Type != "decision" || memory.lastRetainItem.Symbol != "AAPL" {
		t.Fatalf("unexpected retained item: %#v", memory.lastRetainItem)
	}
	if memory.lastRetainItem.TS.IsZero() {
		t.Fatalf("expected retained item timestamp")
	}
	if memory.lastRetainItem.Summary == "" {
		t.Fatalf("expected retained item summary")
	}
	if len(memory.lastRetainItem.Tags) == 0 {
		t.Fatalf("expected retained item tags")
	}
	if memory.lastRetainItem.Data == nil {
		t.Fatalf("expected retained item data")
	}
	if redacted := observability.RedactValue(memory.lastRetainItem.Data); !reflect.DeepEqual(memory.lastRetainItem.Data, redacted) {
		t.Fatalf("expected retained data to be redacted")
	}
}
