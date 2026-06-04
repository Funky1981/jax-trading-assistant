package llmcontext

import (
	"fmt"
	"strings"
)

type StaticPrefixBuilder struct{}

func NewStaticPrefixBuilder() StaticPrefixBuilder {
	return StaticPrefixBuilder{}
}

func (StaticPrefixBuilder) Build(taskType TaskType) string {
	return strings.Join([]string{
		"You are Jax, an ETF paper-trading research assistant.",
		"AI output is advisory-only and must never approve trades or place broker orders.",
		"Respect ETF-only, paper-mode, stale-quote, spread, priced-in, approval, and guardrail boundaries.",
		"Return only the requested schema. Do not request live-mode enablement or risk increases.",
		"Task type: " + string(taskType),
	}, "\n")
}

type SimpleTokenEstimator struct{}

func (SimpleTokenEstimator) Estimate(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	words := len(strings.Fields(trimmed))
	if words == 0 {
		return 1
	}
	return words + (len(trimmed) / 24)
}

type PromptBuilder struct {
	prefix    StaticPrefixBuilder
	estimator SimpleTokenEstimator
	policy    CachePolicy
}

func NewPromptBuilder(prefix StaticPrefixBuilder, estimator SimpleTokenEstimator) PromptBuilder {
	return PromptBuilder{prefix: prefix, estimator: estimator, policy: DefaultCachePolicy()}
}

func (b PromptBuilder) BuildPrompt(task LLMTask) (PromptPackage, error) {
	if task.TaskType == "" {
		return PromptPackage{}, fmt.Errorf("task type required")
	}
	prefix := b.prefix.Build(task.TaskType)
	memory := renderMemories(task.RetrievedMemories)
	dynamic := strings.Join(nonEmpty([]string{
		"symbol: " + strings.ToUpper(task.Symbol),
		"event_id: " + task.EventID,
		"candidate_id: " + task.CandidateID,
		"strategy_id: " + task.StrategyID,
		"event_summary: " + task.EventSummary,
		"market_snapshot: " + task.MarketSnapshot,
		"evidence_bundle: " + task.EvidenceBundle,
		"guardrail_status: " + task.GuardrailStatus,
	}), "\n")
	outputTokens := defaultOutputLimit(task.TaskType)
	inputTokens := b.estimator.Estimate(prefix + "\n" + memory + "\n" + dynamic + "\n" + task.ResponseSchema)
	return PromptPackage{
		TaskType:              task.TaskType,
		Provider:              "local",
		Model:                 "local-small",
		CacheablePrefix:       prefix,
		RetrievedMemory:       memory,
		DynamicContext:        dynamic,
		ResponseSchema:        task.ResponseSchema,
		EstimatedInputTokens:  inputTokens,
		EstimatedOutputTokens: outputTokens,
		EstimatedCostUSD:      0,
		CorrelationID:         task.CorrelationID,
		EventID:               task.EventID,
		CandidateID:           task.CandidateID,
		StrategyID:            task.StrategyID,
		Symbol:                strings.ToUpper(task.Symbol),
		CacheEligible:         b.policy.Allows(task.TaskType),
	}, nil
}

func renderMemories(memories []MemoryArtifact) string {
	if len(memories) == 0 {
		return ""
	}
	lines := make([]string, 0, len(memories))
	for _, memory := range memories {
		lines = append(lines, memory.ID+": "+memory.Summary)
	}
	return strings.Join(lines, "\n")
}

func nonEmpty(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if !strings.HasSuffix(value, ": ") {
			out = append(out, value)
		}
	}
	return out
}

func defaultOutputLimit(taskType TaskType) int {
	switch taskType {
	case TaskApprovalSummary:
		return 260
	case TaskPostTradeReflection:
		return 450
	default:
		return 300
	}
}

type CachePolicy struct {
	allowed map[TaskType]bool
}

func DefaultCachePolicy() CachePolicy {
	return CachePolicy{allowed: map[TaskType]bool{
		TaskHistoricalSummary:     true,
		TaskEvidenceBundleSummary: true,
		TaskPricedInExplanation:   true,
		TaskPostTradeReflection:   true,
		TaskCompaction:            true,
	}}
}

func (p CachePolicy) Allows(taskType TaskType) bool {
	return p.allowed[taskType]
}
