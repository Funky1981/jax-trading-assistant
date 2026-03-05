package agent0

import (
	"context"
	"strings"
)

// MockClient implements a mock Agent0 client for testing
type MockClient struct {
	PlanFunc    func(ctx context.Context, req PlanRequest) (PlanResponse, error)
	ExecuteFunc func(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error)
	HealthFunc  func(ctx context.Context) error
}

func (m *MockClient) Plan(ctx context.Context, req PlanRequest) (PlanResponse, error) {
	if m.PlanFunc != nil {
		return m.PlanFunc(ctx, req)
	}
	summary := "Mock planner produced a HOLD recommendation."
	if symbol := strings.TrimSpace(req.Symbol); symbol != "" {
		summary = "Mock planner produced a HOLD recommendation for " + symbol + "."
	}
	return PlanResponse{
		Summary:        summary,
		Steps:          []string{"collect_context", "evaluate_constraints", "propose_action"},
		Action:         "HOLD",
		Confidence:     0.5,
		ReasoningNotes: "deterministic mock response",
	}, nil
}

func (m *MockClient) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, req)
	}
	return ExecuteResponse{
		ToolCalls: []ToolCall{},
		Success:   true,
		Summary:   "No tools executed (mock executor).",
	}, nil
}

func (m *MockClient) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}
