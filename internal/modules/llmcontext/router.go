package llmcontext

import "fmt"

type RoutingConfig struct {
	Routes map[TaskType]ModelRoute
}

func DefaultRoutingConfig() RoutingConfig {
	local := ModelRoute{ModelAlias: "local-small", Provider: "litellm", ProviderModel: "ollama/local-small", Enabled: true}
	return RoutingConfig{Routes: map[TaskType]ModelRoute{
		TaskEventClassification:          local,
		TaskETFMapping:                   local,
		TaskHistoricalSummary:            local,
		TaskEvidenceBundleSummary:        local,
		TaskApprovalSummary:              local,
		TaskPricedInExplanation:          local,
		TaskComplexConflictingNewsReview: {ModelAlias: "disabled", Enabled: false},
		TaskPostTradeReflection:          local,
		TaskCompaction:                   local,
	}}
}

type ModelRouter struct {
	config RoutingConfig
}

func NewModelRouter(config RoutingConfig) ModelRouter {
	return ModelRouter{config: config}
}

func (r ModelRouter) SelectRoute(task LLMTask, _ BudgetState) (ModelRoute, error) {
	route, ok := r.config.Routes[task.TaskType]
	if !ok {
		return ModelRoute{}, fmt.Errorf("no model route for task type %s", task.TaskType)
	}
	if !route.Enabled {
		return ModelRoute{}, fmt.Errorf("%w: %s", ErrRouteDisabled, task.TaskType)
	}
	return route, nil
}

var ErrRouteDisabled = fmt.Errorf("model route disabled")

type CostGovernor struct {
	limits BudgetLimits
}

func NewCostGovernor(limits BudgetLimits) CostGovernor {
	return CostGovernor{limits: limits}
}

func (g CostGovernor) CanRun(pkg PromptPackage, route ModelRoute) (CostDecision, error) {
	if !route.Enabled {
		return CostDecision{Allowed: false, BlockReason: BlockReasonRouteDisabled, Reason: "model route is disabled"}, nil
	}
	if g.limits.PerCallUSD > 0 && pkg.EstimatedCostUSD > g.limits.PerCallUSD {
		return CostDecision{Allowed: false, BlockReason: BlockReasonPerCallBudgetExceeded, Reason: "estimated call cost exceeds per-call budget"}, nil
	}
	return CostDecision{Allowed: true, Reason: "budget available"}, nil
}

func EstimateCostUSD(pkg PromptPackage, route ModelRoute) float64 {
	inputCost := float64(pkg.EstimatedInputTokens) / 1000 * route.InputUSDPer1K
	outputCost := float64(pkg.EstimatedOutputTokens) / 1000 * route.OutputUSDPer1K
	return inputCost + outputCost
}
