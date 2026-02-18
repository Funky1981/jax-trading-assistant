package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/libs/agent0"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/dexter"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategies"
)

// MemoryClient interface for memory operations
type MemoryClient interface {
	Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error)
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

// Agent0Client interface for AI planning
type Agent0Client interface {
	Plan(ctx context.Context, req agent0.PlanRequest) (agent0.PlanResponse, error)
	Execute(ctx context.Context, req agent0.ExecuteRequest) (agent0.ExecuteResponse, error)
}

// DexterClient interface for research operations
type DexterClient interface {
	ResearchCompany(ctx context.Context, input dexter.ResearchCompanyInput) (dexter.ResearchCompanyOutput, error)
	CompareCompanies(ctx context.Context, input dexter.CompareCompaniesInput) (dexter.CompareCompaniesOutput, error)
}

// ToolRunner interface for executing tools based on AI plans
type ToolRunner interface {
	Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error)
}

// Service provides orchestration functionality
type Service struct {
	memory     MemoryClient
	agent      Agent0Client
	dexter     DexterClient
	tools      ToolRunner
	strategies *strategies.Registry
}

// NewService creates a new orchestration service
func NewService(memory MemoryClient, agent Agent0Client, tools ToolRunner, strategyRegistry *strategies.Registry) *Service {
	return &Service{
		memory:     memory,
		agent:      agent,
		tools:      tools,
		strategies: strategyRegistry,
	}
}

// WithDexter adds Dexter research capabilities
func (s *Service) WithDexter(dexter DexterClient) *Service {
	s.dexter = dexter
	return s
}

// OrchestrationRequest defines the input for orchestration
type OrchestrationRequest struct {
	Bank            string
	Symbol          string
	Strategy        string
	Constraints     map[string]any
	UserContext     string
	Tags            []string
	ResearchQueries []string // Optional: Questions for Dexter research
}

// PlanInput contains inputs for Agent0 planning
type PlanInput struct {
	Symbol      string
	Context     string
	Constraints map[string]any
	Memories    []contracts.MemoryItem
	Signals     []strategies.Signal
}

// PlanResult contains the AI plan result
type PlanResult struct {
	Summary        string
	Steps          []string
	Action         string
	Confidence     float64
	ReasoningNotes string
}

// ToolRun represents a tool execution result
type ToolRun struct {
	Name    string
	Success bool
}

// OrchestrationResult contains the orchestration output
type OrchestrationResult struct {
	Plan  PlanResult
	Tools []ToolRun
}

// Orchestrate executes the full orchestration pipeline
func (s *Service) Orchestrate(ctx context.Context, req OrchestrationRequest) (OrchestrationResult, error) {
	startTime := time.Now()
	var runErr error
	defer func() {
		duration := time.Since(startTime)
		observability.RecordOrchestrationRun(ctx, duration, 7, runErr)
	}()

	// Validate dependencies
	if s.memory == nil {
		runErr = fmt.Errorf("orchestrator: memory client required")
		return OrchestrationResult{}, runErr
	}
	if s.agent == nil {
		runErr = fmt.Errorf("orchestrator: agent required")
		return OrchestrationResult{}, runErr
	}
	if s.tools == nil {
		runErr = fmt.Errorf("orchestrator: tool runner required")
		return OrchestrationResult{}, runErr
	}
	if strings.TrimSpace(req.Bank) == "" {
		runErr = fmt.Errorf("orchestrator: bank is required")
		return OrchestrationResult{}, runErr
	}
	if strings.TrimSpace(req.Symbol) == "" {
		runErr = fmt.Errorf("orchestrator: symbol is required")
		return OrchestrationResult{}, runErr
	}

	// 1. Recall relevant memories
	memories, err := s.memory.Recall(ctx, req.Bank, contracts.MemoryQuery{
		Symbol: req.Symbol,
		Limit:  5,
	})
	if err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}
	observability.RecordRecall(ctx, "hindsight", "recall", err)

	// 2. Get strategy signals (if strategies enabled)
	var signals []strategies.Signal
	if s.strategies != nil && req.Strategy != "" {
		strategy, err := s.strategies.Get(req.Strategy)
		if err == nil {
			// Extract analysis input from constraints
			analysisInput := extractAnalysisInput(req.Symbol, req.Constraints)
			signal, err := strategy.Analyze(ctx, analysisInput)
			if err == nil && signal.Type != strategies.SignalHold {
				signals = append(signals, signal)
				observability.RecordStrategySignal(ctx, req.Strategy, string(signal.Type), signal.Confidence)
			}
		}
	}

	// 3. Dexter research (if enabled and queries provided)
	var research *dexter.ResearchCompanyOutput
	if s.dexter != nil && len(req.ResearchQueries) > 0 {
		researchStart := time.Now()
		res, err := s.dexter.ResearchCompany(ctx, dexter.ResearchCompanyInput{
			Ticker:    req.Symbol,
			Questions: req.ResearchQueries,
		})
		observability.RecordResearchQuery(ctx, "dexter", time.Since(researchStart), err)
		if err == nil {
			research = &res
		}
	}

	// 4. Build context for Agent0
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
	if research != nil {
		contextBuilder.WriteString("\n\nDexter research:\n")
		contextBuilder.WriteString(fmt.Sprintf("Summary: %s\n", research.Summary))
		if len(research.KeyPoints) > 0 {
			contextBuilder.WriteString("Key points:\n")
			for i, point := range research.KeyPoints {
				contextBuilder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, point))
			}
		}
		if len(research.Metrics) > 0 {
			contextBuilder.WriteString(fmt.Sprintf("Metrics: %v\n", research.Metrics))
		}
	}

	agentMemories := []agent0.Memory{}
	for _, mem := range memories {
		agentMemories = append(agentMemories, agent0.Memory{
			Summary: mem.Summary,
			Type:    mem.Type,
			Symbol:  mem.Symbol,
			Tags:    mem.Tags,
			Data:    mem.Data,
		})
	}

	planReq := agent0.PlanRequest{
		Context:     contextBuilder.String(),
		Symbol:      req.Symbol,
		Constraints: req.Constraints,
		Memories:    agentMemories,
	}

	planStart := time.Now()
	agentPlan, err := s.agent.Plan(ctx, planReq)
	if err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}
	observability.RecordAgent0Plan(ctx, time.Since(planStart), len(agentPlan.Steps), agentPlan.Confidence, nil)

	// 5. Build plan result
	plan := PlanResult{
		Summary:        agentPlan.Summary,
		Steps:          agentPlan.Steps,
		Action:         agentPlan.Action,
		Confidence:     agentPlan.Confidence,
		ReasoningNotes: agentPlan.ReasoningNotes,
	}

	// 6. Execute tools based on plan
	toolRuns, err := s.tools.Execute(ctx, plan)
	if err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}

	// 7. Retain decision to memory
	retainedData := map[string]any{
		"inputs": req.Constraints,
		"plan": map[string]any{
			"steps":      plan.Steps,
			"action":     plan.Action,
			"confidence": plan.Confidence,
		},
		"reasoning_notes": plan.ReasoningNotes,
		"tools":           toolRuns,
		"signals":         summarizeSignals(signals),
	}
	if research != nil {
		retainedData["research"] = map[string]any{
			"summary":    research.Summary,
			"key_points": research.KeyPoints,
			"metrics":    research.Metrics,
		}
	}

	retained := contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Symbol:  req.Symbol,
		Summary: strings.TrimSpace(plan.Summary),
		Tags:    contracts.NormalizeMemoryTags(append(req.Tags, req.Strategy)),
		Data:    retainedData,
		Source:  &contracts.MemorySource{System: "jax-orchestrator"},
	}
	if redacted, ok := observability.RedactValue(retained.Data).(map[string]any); ok {
		retained.Data = redacted
	}
	if retained.Summary == "" {
		retained.Summary = "Decision recorded."
	}
	if err := contracts.ValidateMemoryItem(retained); err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}
	if _, err := s.memory.Retain(ctx, req.Bank, retained); err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}
	observability.RecordRetain(ctx, req.Bank, nil)

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
