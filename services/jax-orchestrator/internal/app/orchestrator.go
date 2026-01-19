package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
)

type MemoryClient interface {
	Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error)
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

type Agent0 interface {
	Plan(ctx context.Context, input PlanInput) (PlanResult, error)
}

type ToolRunner interface {
	Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error)
}

type Orchestrator struct {
	memory MemoryClient
	agent  Agent0
	tools  ToolRunner
}

func NewOrchestrator(memory MemoryClient, agent Agent0, tools ToolRunner) *Orchestrator {
	return &Orchestrator{memory: memory, agent: agent, tools: tools}
}

type OrchestrationRequest struct {
	Bank        string
	Symbol      string
	Strategy    string
	Constraints map[string]any
	UserContext string
	Tags        []string
}

type PlanInput struct {
	Symbol      string
	Context     string
	Constraints map[string]any
	Memories    []contracts.MemoryItem
}

type PlanResult struct {
	Summary        string
	Steps          []string
	Action         string
	Confidence     float64
	ReasoningNotes string
}

type ToolRun struct {
	Name    string
	Success bool
}

type OrchestrationResult struct {
	Plan  PlanResult
	Tools []ToolRun
}

func (o *Orchestrator) Run(ctx context.Context, req OrchestrationRequest) (OrchestrationResult, error) {
	if o.memory == nil {
		return OrchestrationResult{}, fmt.Errorf("orchestrator: memory client required")
	}
	if o.agent == nil {
		return OrchestrationResult{}, fmt.Errorf("orchestrator: agent required")
	}
	if o.tools == nil {
		return OrchestrationResult{}, fmt.Errorf("orchestrator: tool runner required")
	}
	if strings.TrimSpace(req.Bank) == "" {
		return OrchestrationResult{}, fmt.Errorf("orchestrator: bank is required")
	}
	if strings.TrimSpace(req.Symbol) == "" {
		return OrchestrationResult{}, fmt.Errorf("orchestrator: symbol is required")
	}

	memories, err := o.memory.Recall(ctx, req.Bank, contracts.MemoryQuery{
		Symbol: req.Symbol,
		Limit:  5,
	})
	if err != nil {
		return OrchestrationResult{}, err
	}

	plan, err := o.agent.Plan(ctx, PlanInput{
		Symbol:      req.Symbol,
		Context:     req.UserContext,
		Constraints: req.Constraints,
		Memories:    memories,
	})
	if err != nil {
		return OrchestrationResult{}, err
	}

	toolRuns, err := o.tools.Execute(ctx, plan)
	if err != nil {
		return OrchestrationResult{}, err
	}

	retained := contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Symbol:  req.Symbol,
		Summary: strings.TrimSpace(plan.Summary),
		Tags:    contracts.NormalizeMemoryTags(append(req.Tags, req.Strategy)),
		Data: map[string]any{
			"inputs": req.Constraints,
			"plan": map[string]any{
				"steps":      plan.Steps,
				"action":     plan.Action,
				"confidence": plan.Confidence,
			},
			"reasoning_notes": plan.ReasoningNotes,
			"tools":           toolRuns,
		},
		Source: &contracts.MemorySource{System: "jax-orchestrator"},
	}
	if redacted, ok := observability.RedactValue(retained.Data).(map[string]any); ok {
		retained.Data = redacted
	}
	if retained.Summary == "" {
		retained.Summary = "Decision recorded."
	}
	if err := contracts.ValidateMemoryItem(retained); err != nil {
		return OrchestrationResult{}, err
	}
	if _, err := o.memory.Retain(ctx, req.Bank, retained); err != nil {
		return OrchestrationResult{}, err
	}

	return OrchestrationResult{Plan: plan, Tools: toolRuns}, nil
}
