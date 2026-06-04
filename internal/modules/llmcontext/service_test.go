package llmcontext

import (
	"context"
	"testing"
)

func TestServiceBlocksIneligibleCallBeforeProvider(t *testing.T) {
	provider := &countingProvider{}
	logger := NewMemoryUsageLogger()
	service := NewService(ServiceConfig{
		Builder:  NewPromptBuilder(NewStaticPrefixBuilder(), SimpleTokenEstimator{}),
		Router:   NewModelRouter(DefaultRoutingConfig()),
		Governor: NewCostGovernor(BudgetLimits{PerCallUSD: 1}),
		Logger:   logger,
		Provider: provider,
	})

	_, err := service.Execute(context.Background(), LLMTask{
		TaskType:      TaskApprovalSummary,
		Symbol:        "SPY",
		CorrelationID: "corr-block",
	}, validEligibilityInput(func(in *EligibilityInput) {
		in.SymbolAllowlisted = false
	}))
	if err == nil {
		t.Fatal("expected blocked call to return error")
	}
	if provider.calls != 0 {
		t.Fatalf("provider was called %d times", provider.calls)
	}
	records := logger.Records()
	if len(records) != 1 || !records[0].Blocked || records[0].BlockReason != BlockReasonSymbolNotAllowlisted {
		t.Fatalf("blocked call was not logged correctly: %#v", records)
	}
}

func TestServiceRunsEligibilityRoutingBudgetLoggingThenProvider(t *testing.T) {
	provider := &countingProvider{result: LLMResult{Text: "ok", InputTokens: 7, OutputTokens: 3, ActualCostUSD: 0.001}}
	logger := NewMemoryUsageLogger()
	service := NewService(ServiceConfig{
		Builder:  NewPromptBuilder(NewStaticPrefixBuilder(), SimpleTokenEstimator{}),
		Router:   NewModelRouter(DefaultRoutingConfig()),
		Governor: NewCostGovernor(BudgetLimits{PerCallUSD: 1}),
		Logger:   logger,
		Provider: provider,
	})

	result, err := service.Execute(context.Background(), LLMTask{
		TaskType:      TaskEvidenceBundleSummary,
		Symbol:        "SPY",
		EventID:       "evt-1",
		CandidateID:   "cand-1",
		StrategyID:    "strat-1",
		CorrelationID: "corr-run",
		EventSummary:  "event",
	}, validEligibilityInput(func(in *EligibilityInput) {
		in.TaskType = TaskEvidenceBundleSummary
	}))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Text != "ok" || provider.calls != 1 {
		t.Fatalf("unexpected provider result/calls: result=%#v calls=%d", result, provider.calls)
	}
	records := logger.Records()
	if len(records) != 1 || records[0].Blocked || records[0].ActualInputTokens != 7 || records[0].ActualOutputTokens != 3 {
		t.Fatalf("usage was not logged correctly: %#v", records)
	}
}

type countingProvider struct {
	calls  int
	result LLMResult
}

func (p *countingProvider) Complete(_ context.Context, pkg PromptPackage) (LLMResult, error) {
	p.calls++
	result := p.result
	result.CorrelationID = pkg.CorrelationID
	return result, nil
}
