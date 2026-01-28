package agent0

import (
	"context"
	"errors"
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
	return PlanResponse{}, errors.New("mock plan not implemented")
}

func (m *MockClient) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, req)
	}
	return ExecuteResponse{}, errors.New("mock execute not implemented")
}

func (m *MockClient) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}
