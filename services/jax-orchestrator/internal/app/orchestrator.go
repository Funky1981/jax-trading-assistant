package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategies"
)

type MemoryClient interface {
	Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error)
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

type Agent0Client interface {
	Plan(ctx context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error)
	Execute(ctx context.Context, req agent0.ExecuteRequest) (agent0.ExecuteResponse, error)
}

type ToolRunner interface {
	Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error)
}

type Orchestrator struct {
	memory     MemoryClient
	agent      Agent0Client
	tools      ToolRunner
	strategies *strategies.Registry
}

func NewOrchestrator(memory MemoryClient, agent Agent0Client, tools ToolRunner, strategyRegistry *strategies.Registry) *Orchestrator {
	return &Orchestrator{
		memory:     memory,
		agent:      agent,
		tools:      tools,
		strategies: strategyRegistry,
	}
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
	Signals     []strategies.Signal // Strategy signals from analysis
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

	// 1. Recall relevant memories
	memories, err := o.memory.Recall(ctx, req.Bank, contracts.MemoryQuery{
		Symbol: req.Symbol,
		Limit:  5,
	})
	if err != nil {
		return OrchestrationResult{}, err
	}

	// 2. Get strategy signals (if strategies enabled)
	var signals []strategies.Signal
	if o.strategies != nil && req.Strategy != "" {
		strategy, err := o.strategies.Get(req.Strategy)
		if err == nil {
			// Extract analysis input from constraints
			analysisInput := extractAnalysisInput(req.Symbol, req.Constraints)
			signal, err := strategy.Analyze(ctx, analysisInput)
			if err == nil && signal.Type != strategies.SignalHold {
				signals = append(signals, signal)
			}
		}
	}

	// 3. Build context for Agent0
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString(req.UserContext)
	if len(memories) > 0 {
		contextBuilder.WriteString("\n\nRecalled memories:\n")
		for i, mem := range memories {
			contextBuilder.WriteString(fmt.Sprintf("%d. %s (type=%s)\n", i+1, mem.Summary, mem.Type))
		}
	}
	if len(signals) > 0 {
		contextBuilder.WriteString("\n\nStrategy signals:\n")
		for i, sig := range signals {
			contextBuilder.WriteString(fmt.Sprintf("%d. %s: %s at %.2f (confidence: %.2f)\n", 
				i+1, sig.Symbol, sig.Type, sig.EntryPrice, sig.Confidence))
		}
	}

	// 4. Agent0 planning with enhanced context
	agentMemories := make([]agent0.Memory, len(memories))
	for i, mem := range memories {
		agentMemories[i] = agent0.Memory{
			Summary: mem.Summary,
			Type:    mem.Type,
			Symbol:  mem.Symbol,
			Tags:    mem.Tags,
			Data:    mem.Data,
		}
	}

	planReq := agent0.PlanRequest{
		Task:        fmt.Sprintf("Analyze trading opportunity for %s", req.Symbol),
		Context:     contextBuilder.String(),
		Symbol:      req.Symbol,
		Constraints: req.Constraints,
		Memories:    agentMemories,
	}

	agentPlan, err := o.agent.Plan(ctx, planReq)
	if err != nil {
		return OrchestrationResult{}, err
	}

	plan := PlanResult{
		Summary:        agentPlan.Summary,
		Steps:          agentPlan.Steps,
		Action:         agentPlan.Action,
		Confidence:     agentPlan.Confidence,
		ReasoningNotes: agentPlan.ReasoningNotes,
	}

	// 5. Execute tools based on plan
	toolRuns, err := o.tools.Execute(ctx, plan)
	if err != nil {
		return OrchestrationResult{}, err
	}

	// 6. Retain decision to memory
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
			"signals":         summarizeSignals(signals),
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

// extractAnalysisInput converts orchestration constraints to strategy analysis input
func extractAnalysisInput(symbol string, constraints map[string]any) strategies.AnalysisInput {
	input := strategies.AnalysisInput{
		Symbol: symbol,
	}

	if price, ok := constraints["price"].(float64); ok {
		input.Price = price
	}
	if rsi, ok := constraints["rsi"].(float64); ok {
		input.RSI = rsi
	}
	if atr, ok := constraints["atr"].(float64); ok {
		input.ATR = atr
	}
	if trend, ok := constraints["market_trend"].(string); ok {
		input.MarketTrend = trend
	}
	if volume, ok := constraints["volume"].(int64); ok {
		input.Volume = volume
	}
	if avgVolume, ok := constraints["avg_volume"].(int64); ok {
		input.AvgVolume20 = avgVolume
	}

	// MACD extraction
	if macdData, ok := constraints["macd"].(map[string]interface{}); ok {
		if value, ok := macdData["value"].(float64); ok {
			input.MACD.Value = value
		}
		if signal, ok := macdData["signal"].(float64); ok {
			input.MACD.Signal = signal
		}
		if histogram, ok := macdData["histogram"].(float64); ok {
			input.MACD.Histogram = histogram
		}
	}

	// Moving averages
	if sma20, ok := constraints["sma20"].(float64); ok {
		input.SMA20 = sma20
	}
	if sma50, ok := constraints["sma50"].(float64); ok {
		input.SMA50 = sma50
	}
	if sma200, ok := constraints["sma200"].(float64); ok {
		input.SMA200 = sma200
	}

	return input
}

// summarizeSignals creates a summary of strategy signals for memory retention
func summarizeSignals(signals []strategies.Signal) []map[string]interface{} {
	summaries := make([]map[string]interface{}, len(signals))
	for i, sig := range signals {
		summaries[i] = map[string]interface{}{
			"type":       string(sig.Type),
			"symbol":     sig.Symbol,
			"confidence": sig.Confidence,
			"entry":      sig.EntryPrice,
			"stop":       sig.StopLoss,
			"reason":     sig.Reason,
		}
	}
	return summaries
}

