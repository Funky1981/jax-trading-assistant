package llmcontext

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPromptBuilderSeparatesCacheablePrefixAndDynamicContext(t *testing.T) {
	task := LLMTask{
		TaskType:          TaskEvidenceBundleSummary,
		Symbol:            "spy",
		EventID:           "evt-1",
		CandidateID:       "cand-1",
		StrategyID:        "ETF_NEWS_002",
		CorrelationID:     "corr-1",
		EventSummary:      "CPI came in hotter than expected.",
		MarketSnapshot:    "SPY bid 510 ask 510.04",
		EvidenceBundle:    "priced_in=false; confounders=none",
		GuardrailStatus:   "pass",
		RetrievedMemories: []MemoryArtifact{{ID: "mem-1", Summary: "Prior CPI surprises pressured duration ETFs."}},
	}
	builder := NewPromptBuilder(NewStaticPrefixBuilder(), SimpleTokenEstimator{})

	pkg, err := builder.BuildPrompt(task)
	if err != nil {
		t.Fatalf("BuildPrompt returned error: %v", err)
	}

	if pkg.TaskType != TaskEvidenceBundleSummary {
		t.Fatalf("unexpected task type: %s", pkg.TaskType)
	}
	if !strings.Contains(pkg.CacheablePrefix, "advisory-only") {
		t.Fatalf("cacheable prefix missing advisory boundary: %q", pkg.CacheablePrefix)
	}
	if strings.Contains(pkg.CacheablePrefix, "CPI came in hotter") || strings.Contains(pkg.CacheablePrefix, "SPY bid") {
		t.Fatalf("cacheable prefix contains dynamic event data: %q", pkg.CacheablePrefix)
	}
	if !strings.Contains(pkg.DynamicContext, "CPI came in hotter") || !strings.Contains(pkg.DynamicContext, "SPY bid") {
		t.Fatalf("dynamic context missing event/market data: %q", pkg.DynamicContext)
	}
	if !pkg.CacheEligible {
		t.Fatal("evidence bundle summary should be cache eligible")
	}
	if pkg.EstimatedInputTokens <= 0 || pkg.EstimatedOutputTokens <= 0 {
		t.Fatalf("expected token estimates, got input=%d output=%d", pkg.EstimatedInputTokens, pkg.EstimatedOutputTokens)
	}
}

func TestCachePolicyRejectsApprovalAndCurrentMarketTasks(t *testing.T) {
	policy := DefaultCachePolicy()

	for _, taskType := range []TaskType{TaskApprovalSummary, TaskCurrentTradeDecision, TaskBrokerStatus} {
		if policy.Allows(taskType) {
			t.Fatalf("expected %s to be cache blocked", taskType)
		}
	}
	if !policy.Allows(TaskHistoricalSummary) {
		t.Fatal("historical summaries should be cache eligible")
	}
}

func TestLocalFirstRouterDisablesStrongRoutesByDefault(t *testing.T) {
	router := NewModelRouter(DefaultRoutingConfig())

	route, err := router.SelectRoute(LLMTask{TaskType: TaskEventClassification}, BudgetState{})
	if err != nil {
		t.Fatalf("SelectRoute returned error: %v", err)
	}
	if route.ModelAlias != "local-small" || route.Paid {
		t.Fatalf("expected local-small unpaid route, got %#v", route)
	}

	_, err = router.SelectRoute(LLMTask{TaskType: TaskComplexConflictingNewsReview}, BudgetState{})
	if err == nil {
		t.Fatal("expected disabled strong route to return error")
	}
}

func TestCostGovernorBlocksBeforeProviderWhenBudgetExceeded(t *testing.T) {
	governor := NewCostGovernor(BudgetLimits{PerCallUSD: 0.01})
	pkg := PromptPackage{
		TaskType:              TaskEvidenceBundleSummary,
		Model:                 "local-small",
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 1000,
		EstimatedCostUSD:      0.02,
		CorrelationID:         "corr-budget",
	}

	decision, err := governor.CanRun(pkg, ModelRoute{ModelAlias: "local-small", ProviderModel: "ollama/qwen", Enabled: true})
	if err != nil {
		t.Fatalf("CanRun returned error: %v", err)
	}
	if decision.Allowed {
		t.Fatalf("expected budget block, got %#v", decision)
	}
	if decision.BlockReason != BlockReasonPerCallBudgetExceeded {
		t.Fatalf("unexpected block reason: %s", decision.BlockReason)
	}
}

func TestUsageLoggerRecordsPlannedBlockedAndActualCalls(t *testing.T) {
	logger := NewMemoryUsageLogger()
	pkg := PromptPackage{TaskType: TaskEventClassification, Model: "local-small", CorrelationID: "corr-usage", EstimatedCostUSD: 0.001}

	if err := logger.RecordPlanned(pkg, CostDecision{Allowed: false, BlockReason: BlockReasonDailyBudgetExceeded}); err != nil {
		t.Fatalf("RecordPlanned returned error: %v", err)
	}
	if err := logger.RecordActual(LLMResult{CorrelationID: "corr-usage", InputTokens: 10, OutputTokens: 5, ActualCostUSD: 0.0002}); err != nil {
		t.Fatalf("RecordActual returned error: %v", err)
	}

	records := logger.Records()
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if !records[0].Blocked || records[0].BlockReason != BlockReasonDailyBudgetExceeded {
		t.Fatalf("expected blocked planned record, got %#v", records[0])
	}
	if records[0].ActualInputTokens != 10 || records[0].ActualOutputTokens != 5 {
		t.Fatalf("actual usage not recorded: %#v", records[0])
	}
}

func TestNoopProviderReturnsDeterministicResultWithoutNetwork(t *testing.T) {
	provider := NoopProvider{ResponseText: "ok"}
	got, err := provider.Complete(context.Background(), PromptPackage{CorrelationID: "corr-noop"})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if got.Text != "ok" || got.CorrelationID != "corr-noop" {
		t.Fatalf("unexpected noop result: %#v", got)
	}
}

func TestTemplateRendererEnforcesOutputLimits(t *testing.T) {
	rendered := RenderApprovalSummary(ApprovalTemplateData{
		Symbol:          "SPY",
		StrategyName:    "ETF news",
		PaperAction:     "BUY",
		Confidence:      "medium",
		ModelReason:     strings.Repeat("risk-on ", 120),
		PricedInVerdict: "not_priced_in",
		PricedInReason:  strings.Repeat("fresh ", 50),
		Entry:           "510.00",
		StopLoss:        "505.00",
		TakeProfit:      "520.00",
		RiskAmount:      "100.00",
		ExpiresAt:       "2026-06-04T16:00:00Z",
	})

	if CountWords(rendered) > 180 {
		t.Fatalf("approval summary exceeded 180 words: %d\n%s", CountWords(rendered), rendered)
	}
	if !strings.Contains(rendered, "ETF: SPY") || !strings.Contains(rendered, "Decision:") {
		t.Fatalf("approval template missing required fields:\n%s", rendered)
	}
}

func TestCostRollupsCalculatePerCandidateMetrics(t *testing.T) {
	records := []UsageRecord{
		{TaskType: TaskEvidenceBundleSummary, CandidateID: "cand-1", StrategyID: "strat-a", Symbol: "SPY", EstimatedCostUSD: 0.03},
		{TaskType: TaskEventClassification, EventID: "evt-1", StrategyID: "strat-a", Symbol: "SPY", EstimatedCostUSD: 0.01, Blocked: true, BlockReason: BlockReasonEligibilityFailed},
		{TaskType: TaskApprovalSummary, CandidateID: "cand-1", StrategyID: "strat-a", Symbol: "SPY", EstimatedCostUSD: 0.04},
	}

	rollup := RollupCosts(records, "strategy", "strat-a", time.Unix(0, 0), time.Unix(3600, 0))
	if rollup.TotalCostUSD != 0.08 {
		t.Fatalf("unexpected total cost: %.2f", rollup.TotalCostUSD)
	}
	if rollup.CandidateCount != 1 || rollup.EventCount != 1 {
		t.Fatalf("unexpected rollup counts: %#v", rollup)
	}
	if rollup.PaidCallsAvoided != 1 {
		t.Fatalf("expected one paid call avoided, got %d", rollup.PaidCallsAvoided)
	}
}
