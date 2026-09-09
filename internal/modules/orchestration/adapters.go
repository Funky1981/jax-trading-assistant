package orchestration

import (
	"context"
	"fmt"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/utcp"
)

// MemoryClientAdapter adapts UTCP MemoryService to MemoryClient interface
type MemoryClientAdapter struct {
	service *utcp.MemoryService
}

// NewMemoryClient creates a new memory client adapter
func NewMemoryClient(memoryServiceURL string) (*MemoryClientAdapter, error) {
	// Create a UTCP client configured for the research runtime memory surface.
	cfg := utcp.ProvidersConfig{
		Providers: []utcp.ProviderConfig{
			{
				ID:        utcp.MemoryProviderID,
				Transport: "http",
				Endpoint:  memoryServiceURL,
			},
		},
	}

	utcpClient, err := utcp.NewUTCPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create UTCP client: %w", err)
	}

	return &MemoryClientAdapter{
		service: utcp.NewMemoryService(utcpClient),
	}, nil
}

// Recall retrieves memories from the memory service
func (m *MemoryClientAdapter) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	req := contracts.MemoryRecallRequest{
		Bank:  bank,
		Query: query,
	}

	resp, err := m.service.Recall(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Items, nil
}

// Retain stores a memory item
func (m *MemoryClientAdapter) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	req := contracts.MemoryRetainRequest{
		Bank: bank,
		Item: item,
	}

	resp, err := m.service.Retain(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// Agent0ClientAdapter adapts agent0.Client to Agent0Client interface
type Agent0ClientAdapter struct {
	client *agent0.Client
}

// NewAgent0Client creates a new Agent0 client adapter
func NewAgent0Client(agent0ServiceURL string) (*Agent0ClientAdapter, error) {
	client, err := agent0.New(agent0ServiceURL)
	if err != nil {
		return nil, fmt.Errorf("create Agent0 client: %w", err)
	}

	return &Agent0ClientAdapter{client: client}, nil
}

// Plan creates an AI plan
func (a *Agent0ClientAdapter) Plan(ctx context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error) {
	return a.client.Plan(ctx, req)
}

// Execute executes an AI plan
func (a *Agent0ClientAdapter) Execute(ctx context.Context, req agent0.ExecuteRequest) (agent0.ExecuteResponse, error) {
	return a.client.Execute(ctx, req)
}

// ToolRunnerImpl implements the ToolRunner interface
type ToolRunnerImpl struct{}

// NewToolRunner creates a new tool runner
func NewToolRunner() *ToolRunnerImpl {
	return &ToolRunnerImpl{}
}

// Execute executes tools based on the AI plan
func (t *ToolRunnerImpl) Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error) {
	// Tool execution remains a separate, bounded Jax-native capability boundary.
	// This runner intentionally does not infer or execute arbitrary plan steps.
	return []ToolRun{}, nil
}
