package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/harness"
	"jax-trading-assistant/internal/modules/llmcontext"
)

type CostManagedConfig struct {
	Logger    *llmcontext.MemoryUsageLogger
	Limits    llmcontext.BudgetLimits
	Estimator llmcontext.TokenEstimator
}

type costManagedLLMClient struct {
	base      LLMClient
	service   llmcontext.Service
	estimator llmcontext.TokenEstimator
	provider  *chatProviderAdapter
}

func NewCostManagedLLMClient(base LLMClient, config CostManagedConfig) LLMClient {
	if base == nil {
		return nil
	}
	estimator := config.Estimator
	if estimator == nil {
		estimator = llmcontext.SimpleTokenEstimator{}
	}
	logger := config.Logger
	if logger == nil {
		logger = llmcontext.NewMemoryUsageLogger()
	}
	provider := &chatProviderAdapter{base: base}
	route := llmcontext.ModelRoute{
		ModelAlias:     "local-small",
		Provider:       "litellm",
		ProviderModel:  "chat-gateway",
		Enabled:        true,
		InputUSDPer1K:  0.001,
		OutputUSDPer1K: 0.001,
	}
	service := llmcontext.NewService(llmcontext.ServiceConfig{
		Builder: llmcontext.NewPromptBuilder(llmcontext.NewStaticPrefixBuilder(), estimator),
		Router: llmcontext.NewModelRouter(llmcontext.RoutingConfig{Routes: map[llmcontext.TaskType]llmcontext.ModelRoute{
			llmcontext.TaskApprovalSummary: route,
		}}),
		Governor: llmcontext.NewCostGovernor(config.Limits),
		Logger:   logger,
		Provider: provider,
	})
	return &costManagedLLMClient{base: base, service: service, estimator: estimator, provider: provider}
}

func (c *costManagedLLMClient) Complete(ctx context.Context, msgs []LLMMessage) (string, []harness.ToolCall, error) {
	c.provider.messages = append([]LLMMessage(nil), msgs...)
	task := llmcontext.LLMTask{
		TaskType:       llmcontext.TaskApprovalSummary,
		CorrelationID:  fmt.Sprintf("chat-%d", time.Now().UnixNano()),
		EventSummary:   renderLLMMessages(msgs),
		ResponseSchema: "advisory chat response with optional tool calls",
	}
	result, err := c.service.Execute(ctx, task, permissiveChatEligibility())
	if err != nil {
		return "", nil, err
	}
	return result.Text, append([]harness.ToolCall(nil), c.provider.calls...), nil
}

type chatProviderAdapter struct {
	base     LLMClient
	messages []LLMMessage
	calls    []harness.ToolCall
}

func (p *chatProviderAdapter) Complete(ctx context.Context, pkg llmcontext.PromptPackage) (llmcontext.LLMResult, error) {
	reply, calls, err := p.base.Complete(ctx, p.messages)
	if err != nil {
		return llmcontext.LLMResult{}, err
	}
	p.calls = append([]harness.ToolCall(nil), calls...)
	return llmcontext.LLMResult{
		CorrelationID: pkg.CorrelationID,
		Text:          reply,
		InputTokens:   pkg.EstimatedInputTokens,
		OutputTokens:  llmcontext.SimpleTokenEstimator{}.Estimate(reply),
	}, nil
}

func renderLLMMessages(msgs []LLMMessage) string {
	var b strings.Builder
	for _, msg := range msgs {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(msg.Role)
		b.WriteString(": ")
		b.WriteString(msg.Content)
	}
	return b.String()
}

func permissiveChatEligibility() llmcontext.EligibilityInput {
	return llmcontext.EligibilityInput{
		TaskType:              llmcontext.TaskApprovalSummary,
		EventID:               "chat",
		Symbol:                "SPY",
		StrategyID:            "chat-assistant",
		CandidateID:           "chat",
		EvidenceBundleID:      "chat-history",
		EventExists:           true,
		EventRecent:           true,
		EventTradeable:        true,
		SourceQualityOK:       true,
		SymbolAllowlisted:     true,
		AssetTypeETF:          true,
		PlainVanillaETF:       true,
		PaperMode:             true,
		QuoteFresh:            true,
		SpreadAcceptable:      true,
		MarketSessionOK:       true,
		ETFMappingExists:      true,
		PricedInVerdict:       llmcontext.PricedInVerdictNotPricedIn,
		ConfounderAnalysisOK:  true,
		EvidenceBundlePresent: true,
		BudgetAvailable:       true,
		ModelRouteEnabled:     true,
		RequestedModelRoute:   "local-small",
	}
}
