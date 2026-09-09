package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/audit"
	"jax-trading-assistant/internal/modules/planner"
	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/strategies"
)

// MemoryClient interface for memory operations
type MemoryClient interface {
	Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error)
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

// ToolRunner interface for executing tools based on AI plans
type ToolRunner interface {
	Execute(ctx context.Context, plan PlanResult) ([]ToolRun, error)
}

// Service provides orchestration functionality
type Service struct {
	memory     MemoryClient
	planner    planner.Planner
	tools      ToolRunner
	strategies *strategies.Registry
	audit      *audit.Service
}

// NewService creates a new orchestration service
func NewService(memory MemoryClient, p planner.Planner, tools ToolRunner, strategyRegistry *strategies.Registry) *Service {
	return &Service{
		memory:     memory,
		planner:    p,
		tools:      tools,
		strategies: strategyRegistry,
	}
}

func (s *Service) WithAudit(auditSvc *audit.Service) *Service {
	s.audit = auditSvc
	return s
}

// OrchestrationRequest defines the input for orchestration
type OrchestrationRequest struct {
	Bank        string
	Symbol      string
	Strategy    string
	Constraints map[string]any
	UserContext string
	Tags        []string
}

// PlanInput and PlanResult are compatibility aliases for the Jax-owned
// planner contract.
type PlanInput = planner.Request
type PlanResult = planner.Result

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
	if s.planner == nil {
		runErr = fmt.Errorf("orchestrator: planner required")
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
	observability.RecordRecall(ctx, "memory", "recall", err)

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

	// 3. Build bounded context for the Jax-owned planner.
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
	planReq := planner.Request{
		Context:     contextBuilder.String(),
		Symbol:      req.Symbol,
		Constraints: req.Constraints,
		Memories:    memories,
		Signals:     signals,
	}

	planStart := time.Now()
	plannerResult, err := s.planner.Plan(ctx, planReq)
	if err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}
	observability.RecordPlannerPlan(ctx, time.Since(planStart), len(plannerResult.Steps), plannerResult.Confidence, nil)

	if s.audit != nil {
		flowID := observability.FlowIDFromContext(ctx)
		runInfo := observability.RunInfoFromContext(ctx)
		valid, trace := audit.ValidatePlanShape(plannerResult.Summary, plannerResult.Action, plannerResult.Confidence, plannerResult.Steps)
		decisionID, _ := s.audit.LogAIDecision(ctx, audit.AIDecisionRecord{
			RunID:       runInfo.RunID,
			FlowID:      flowID,
			Role:        "planner",
			Provider:    "jax-planner",
			Model:       planner.ContractVersion,
			Prompt:      map[string]any{"context": planReq.Context, "constraints": planReq.Constraints, "symbol": planReq.Symbol},
			Response:    map[string]any{"summary": plannerResult.Summary, "steps": plannerResult.Steps, "action": plannerResult.Action, "confidence": plannerResult.Confidence, "reasoning": plannerResult.ReasoningNotes},
			SchemaValid: valid,
			Decision:    plannerResult.Action,
			Reasoning:   plannerResult.ReasoningNotes,
			RuleTrace:   trace,
		})
		_ = s.audit.LogAIAcceptance(ctx, decisionID, valid, "schema_validator", "plan schema validation", trace)
		if !valid {
			runErr = fmt.Errorf("planner result failed schema validation")
			return OrchestrationResult{}, runErr
		}
	}

	// 4. Build plan result
	plan := plannerResult

	// 5. Execute tools based on plan
	toolRuns, err := s.tools.Execute(ctx, plan)
	if err != nil {
		runErr = err
		return OrchestrationResult{}, err
	}

	// 6. Retain decision to memory
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
	retained := contracts.MemoryItem{
		TS:      time.Now().UTC(),
		Type:    "decision",
		Symbol:  req.Symbol,
		Summary: strings.TrimSpace(plan.Summary),
		Tags:    contracts.NormalizeMemoryTags(append(req.Tags, req.Strategy)),
		Data:    retainedData,
		Source:  &contracts.MemorySource{System: "jax-research"},
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
