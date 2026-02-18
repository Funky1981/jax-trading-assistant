package orchestration

import (
	"context"
	"fmt"
	"log"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/dexter"
	"jax-trading-assistant/libs/utcp"
)

// MemoryClientAdapter adapts UTCP MemoryService to MemoryClient interface
type MemoryClientAdapter struct {
	service *utcp.MemoryService
}

// NewMemoryClient creates a new memory client adapter
func NewMemoryClient(memoryServiceURL string) (*MemoryClientAdapter, error) {
	// Create UTCP client configured for HTTP transport to jax-memory service
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

// DexterClientAdapter adapts dexter.Client to DexterClient interface
type DexterClientAdapter struct {
	client *dexter.Client
}

// NewDexterClient creates a new Dexter client adapter
func NewDexterClient(dexterServiceURL string) (*DexterClientAdapter, error) {
	client, err := dexter.New(dexterServiceURL)
	if err != nil {
		return nil, fmt.Errorf("create Dexter client: %w", err)
	}

	return &DexterClientAdapter{client: client}, nil
}

// ResearchCompany performs company research
func (d *DexterClientAdapter) ResearchCompany(ctx context.Context, input dexter.ResearchCompanyInput) (dexter.ResearchCompanyOutput, error) {
	return d.client.ResearchCompany(ctx, input)
}

// CompareCompanies compares multiple companies
func (d *DexterClientAdapter) CompareCompanies(ctx context.Context, input dexter.CompareCompaniesInput) (dexter.CompareCompaniesOutput, error) {
	return d.client.CompareCompanies(ctx, input)
}

// ToolRunnerImpl implements the ToolRunner interface
type ToolRunnerImpl struct {
	dexter *DexterClientAdapter
}

// NewToolRunner creates a new tool runner
func NewToolRunner(dexter *DexterClientAdapter) *ToolRunnerImpl {
	return &ToolRunnerImpl{dexter: dexter}
}

// Execute executes tools based on the AI plan
func (t *ToolRunnerImpl) Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error) {
	// For now, we don't execute tools automatically
	// In the future, this could parse plan.Steps and execute required tools
	log.Printf("tool execution requested for plan: %s", plan.Summary)

	// Return empty tool runs for now
	return []ToolRun{}, nil
}
