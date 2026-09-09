package orchestration

import (
	"context"
	"fmt"
	"os"
	"strings"

	"jax-trading-assistant/internal/modules/llmcontext"
	"jax-trading-assistant/internal/modules/planner"
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

// NewPlanner creates the deterministic local Jax-owned advisory planner.
func NewPlanner() (*planner.Service, error) {
	return planner.NewService(planner.DeterministicProvider{})
}

// NewConfiguredPlanner selects the deterministic provider by default. The
// model-backed route is explicit and reuses the existing Jax LiteLLM
// transport; a failed model route never falls back silently to deterministic
// output.
func NewConfiguredPlanner(lookup func(string) (string, bool)) (*planner.Service, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	provider := strings.ToLower(strings.TrimSpace(lookupValue(lookup, "JAX_PLANNER_PROVIDER")))
	switch provider {
	case "", "deterministic", "offline":
		return NewPlanner()
	case "litellm":
		baseURL := strings.TrimSpace(lookupValue(lookup, "AI_GATEWAY_BASE_URL"))
		apiKey := strings.TrimSpace(lookupValue(lookup, "AI_GATEWAY_API_KEY"))
		if baseURL == "" || apiKey == "" {
			return nil, fmt.Errorf("planner: litellm provider requires AI_GATEWAY_BASE_URL and AI_GATEWAY_API_KEY")
		}
		model := strings.TrimSpace(lookupValue(lookup, "JAX_PLANNER_MODEL"))
		if model == "" {
			model = strings.TrimSpace(lookupValue(lookup, "AI_DEFAULT_MODEL"))
		}
		if model == "" {
			model = "local-small"
		}
		client := llmcontext.NewLiteLLMClient(llmcontext.LiteLLMConfig{BaseURL: baseURL, APIKey: apiKey})
		modelProvider, err := planner.NewModelProvider(client, "litellm", model)
		if err != nil {
			return nil, err
		}
		return planner.NewService(modelProvider)
	default:
		return nil, fmt.Errorf("planner: unknown JAX_PLANNER_PROVIDER %q", provider)
	}
}

func lookupValue(lookup func(string) (string, bool), key string) string {
	value, _ := lookup(key)
	return value
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
