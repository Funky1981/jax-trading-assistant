package orchestration

import (
	"context"
	"testing"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/strategies"
)

// fakeMemory implements MemoryClient interface for testing
type fakeMemory struct {
	lastRecallBank string
	lastRetainBank string
	lastRetainItem contracts.MemoryItem
	recallInvoked  bool
	retainInvoked  bool
}

func (m *fakeMemory) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	m.recallInvoked = true
	m.lastRecallBank = bank
	return []contracts.MemoryItem{
		{
			Summary: "Previous trade on AAPL",
			Type:    "decision",
			Symbol:  query.Symbol,
			Tags:    []string{"trade"},
		},
	}, nil
}

func (m *fakeMemory) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	m.retainInvoked = true
	m.lastRetainBank = bank
	m.lastRetainItem = item
	return "mem-123", nil
}

// fakeAgent implements Agent0Client interface for testing
type fakeAgent struct {
	lastPlanRequest agent0.PlanRequest
}

func (a *fakeAgent) Plan(ctx context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error) {
	a.lastPlanRequest = req
	return agent0.PlanResponse{
		Summary:        "Hold position on " + req.Symbol,
		Steps:          []string{"Analyze market", "Review position"},
		Action:         "hold",
		Confidence:     0.75,
		ReasoningNotes: "Market conditions stable",
	}, nil
}

func (a *fakeAgent) Execute(ctx context.Context, req agent0.ExecuteRequest) (agent0.ExecuteResponse, error) {
	return agent0.ExecuteResponse{
		Success:   true,
		Summary:   "Executed",
		ToolCalls: []agent0.ToolCall{},
	}, nil
}

// fakeTools implements ToolRunner interface for testing
type fakeTools struct {
	executeCalled bool
}

func (t *fakeTools) Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error) {
	t.executeCalled = true
	return []ToolRun{}, nil
}

func TestService_BasicOrchestration(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	service := NewService(memory, agent, tools, registry)

	result, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:        "trade_decisions",
		Symbol:      "AAPL",
		UserContext: "Analyzing AAPL for potential entry",
		Tags:        []string{"analysis"},
		Constraints: map[string]any{"price": 150.0},
	})

	if err != nil {
		t.Fatalf("orchestrate failed: %v", err)
	}

	// Verify memory recall was invoked
	if !memory.recallInvoked {
		t.Error("expected memory recall to be invoked")
	}
	if memory.lastRecallBank != "trade_decisions" {
		t.Errorf("expected recall bank 'trade_decisions', got '%s'", memory.lastRecallBank)
	}

	// Verify agent plan was invoked
	if agent.lastPlanRequest.Symbol != "AAPL" {
		t.Errorf("expected symbol AAPL, got %s", agent.lastPlanRequest.Symbol)
	}

	// Verify plan result
	if result.Plan.Action != "hold" {
		t.Errorf("expected action 'hold', got '%s'", result.Plan.Action)
	}
	if result.Plan.Confidence != 0.75 {
		t.Errorf("expected confidence 0.75, got %.2f", result.Plan.Confidence)
	}

	// Verify tools were executed
	if !tools.executeCalled {
		t.Error("expected tools to be executed")
	}

	// Verify memory retain was invoked
	if !memory.retainInvoked {
		t.Error("expected memory retain to be invoked")
	}
	if memory.lastRetainItem.Symbol != "AAPL" {
		t.Errorf("expected retained symbol AAPL, got %s", memory.lastRetainItem.Symbol)
	}
}

func TestService_RequiresMemoryClient(t *testing.T) {
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	service := NewService(nil, agent, tools, registry)

	_, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:   "trade_decisions",
		Symbol: "AAPL",
	})

	if err == nil {
		t.Error("expected error when memory client is nil")
	}
	if !containsStr(err.Error(), "memory client required") {
		t.Errorf("expected 'memory client required' error, got: %v", err)
	}
}

func TestService_RequiresAgentClient(t *testing.T) {
	memory := &fakeMemory{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	service := NewService(memory, nil, tools, registry)

	_, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:   "trade_decisions",
		Symbol: "AAPL",
	})

	if err == nil {
		t.Error("expected error when agent client is nil")
	}
	if !containsStr(err.Error(), "agent required") {
		t.Errorf("expected 'agent required' error, got: %v", err)
	}
}

func TestService_RequiresToolRunner(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	registry := strategies.NewRegistry()

	service := NewService(memory, agent, nil, registry)

	_, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:   "trade_decisions",
		Symbol: "AAPL",
	})

	if err == nil {
		t.Error("expected error when tool runner is nil")
	}
	if !containsStr(err.Error(), "tool runner required") {
		t.Errorf("expected 'tool runner required' error, got: %v", err)
	}
}

func TestService_RequiresBank(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	service := NewService(memory, agent, tools, registry)

	_, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:   "",
		Symbol: "AAPL",
	})

	if err == nil {
		t.Error("expected error when bank is empty")
	}
	if !containsStr(err.Error(), "bank is required") {
		t.Errorf("expected 'bank is required' error, got: %v", err)
	}
}

func TestService_RequiresSymbol(t *testing.T) {
	memory := &fakeMemory{}
	agent := &fakeAgent{}
	tools := &fakeTools{}
	registry := strategies.NewRegistry()

	service := NewService(memory, agent, tools, registry)

	_, err := service.Orchestrate(context.Background(), OrchestrationRequest{
		Bank:   "trade_decisions",
		Symbol: "",
	})

	if err == nil {
		t.Error("expected error when symbol is empty")
	}
	if !containsStr(err.Error(), "symbol is required") {
		t.Errorf("expected 'symbol is required' error, got: %v", err)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(s) > len(substr)+1 && s[1:len(s)-1] != s[1 : len(s)-1][:len(s)-2] && findSubstr(s, substr)))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
