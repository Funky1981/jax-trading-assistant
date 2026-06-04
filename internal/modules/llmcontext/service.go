package llmcontext

import (
	"context"
	"fmt"
)

type ServiceConfig struct {
	Builder  PromptBuilder
	Router   ModelRouter
	Governor CostGovernor
	Logger   *MemoryUsageLogger
	Provider LLMProviderClient
}

type Service struct {
	builder  PromptBuilder
	router   ModelRouter
	governor CostGovernor
	logger   *MemoryUsageLogger
	provider LLMProviderClient
	gate     EligibilityGate
}

func NewService(config ServiceConfig) Service {
	return Service{
		builder:  config.Builder,
		router:   config.Router,
		governor: config.Governor,
		logger:   config.Logger,
		provider: config.Provider,
		gate:     EligibilityGate{},
	}
}

func (s Service) Execute(ctx context.Context, task LLMTask, eligibility EligibilityInput) (LLMResult, error) {
	if s.provider == nil {
		return LLMResult{}, fmt.Errorf("llm provider required")
	}
	pkg, err := s.builder.BuildPrompt(task)
	if err != nil {
		return LLMResult{}, err
	}
	eligibility.TaskType = task.TaskType
	decision := s.gate.Evaluate(eligibility)
	if !decision.Eligible {
		if s.logger != nil {
			_ = s.logger.RecordPlanned(pkg, CostDecision{Allowed: false, BlockReason: decision.BlockedReason, Reason: decision.Reason})
		}
		return LLMResult{}, fmt.Errorf("llm call blocked: %s", decision.BlockedReason)
	}
	route, err := s.router.SelectRoute(task, BudgetState{})
	if err != nil {
		if s.logger != nil {
			_ = s.logger.RecordPlanned(pkg, CostDecision{Allowed: false, BlockReason: BlockReasonRouteDisabled, Reason: err.Error()})
		}
		return LLMResult{}, err
	}
	pkg.Model = route.ModelAlias
	pkg.Provider = route.Provider
	pkg.EstimatedCostUSD = EstimateCostUSD(pkg, route)
	cost := s.governor.CanRun
	costDecision, err := cost(pkg, route)
	if err != nil {
		return LLMResult{}, err
	}
	if s.logger != nil {
		_ = s.logger.RecordPlanned(pkg, costDecision)
	}
	if !costDecision.Allowed {
		return LLMResult{}, fmt.Errorf("llm call blocked: %s", costDecision.BlockReason)
	}
	result, err := s.provider.Complete(ctx, pkg)
	if err != nil {
		return LLMResult{}, err
	}
	if s.logger != nil {
		_ = s.logger.RecordActual(result)
	}
	return result, nil
}
