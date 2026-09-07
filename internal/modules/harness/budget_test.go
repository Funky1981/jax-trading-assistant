package harness

import (
	"context"
	"strings"
	"testing"
	"time"
)

func budgetFixture(t *testing.T) *BudgetController {
	t.Helper()
	budget, err := NewResearchBudget(ResearchBudget{Version: "v1", MaxWallClock: time.Minute, ToolTimeout: time.Second, MaxSteps: 2, MaxToolCalls: 2, MaxModelCalls: 2, MaxRetries: 1, MaxInputTokens: 100, MaxOutputTokens: 50, MaxReasoningTokens: 20, MaxEstimatedCostUSD: 1, MaxActualCostUSD: 1, MaximumModelTier: BudgetTierLocal, RequireKnownCost: true})
	if err != nil {
		t.Fatal(err)
	}
	controller, err := NewBudgetController(budget)
	if err != nil {
		t.Fatal(err)
	}
	return controller
}

func TestBudgetControllerEnforcesAllResearchLimits(t *testing.T) {
	controller := budgetFixture(t)
	if err := controller.AdvanceStep(); err != nil {
		t.Fatal(err)
	}
	if err := controller.ReserveToolCall(); err != nil {
		t.Fatal(err)
	}
	if err := controller.ReserveRetry(); err != nil {
		t.Fatal(err)
	}
	if err := controller.ReserveModelCall(BudgetTierLocal, 20, 10, 5, 0.2); err != nil {
		t.Fatal(err)
	}
	usage, err := NormalizeProviderUsage(ProviderUsage{Provider: "fixture", Model: "local", InputTokens: intPointer(20), OutputTokens: intPointer(10), ReasoningTokens: intPointer(5), ReportedCostUSD: floatPointer(0.2), FinishStatus: "stop", RawPayloadSHA256: strings.Repeat("a", 64)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordUsage(usage); err != nil {
		t.Fatal(err)
	}
	if state := controller.State(); state.ModelCalls != 1 || state.ActualCostUSD != 0.2 {
		t.Fatalf("unexpected budget state: %#v", state)
	}
	if err := controller.AdvanceStep(); err != nil {
		t.Fatal(err)
	}
	if err := controller.AdvanceStep(); err == nil {
		t.Fatal("expected max-step exhaustion")
	}
	if err := controller.ReserveRetry(); err == nil {
		t.Fatal("expected retry exhaustion")
	}
}

func TestBudgetControllerRejectsModelTierAndCostOverruns(t *testing.T) {
	controller := budgetFixture(t)
	if err := controller.ReserveModelCall(BudgetTierCheapHosted, 1, 1, 1, 0); err == nil {
		t.Fatal("expected model tier ceiling rejection")
	}
	if err := controller.ReserveModelCall(BudgetTierLocal, 101, 1, 1, 0); err == nil {
		t.Fatal("expected token budget rejection")
	}
	if err := controller.ReserveModelCall(BudgetTierLocal, 1, 1, 1, 1.1); err == nil {
		t.Fatal("expected estimated cost rejection")
	}
	ambiguous, err := NormalizeProviderUsage(ProviderUsage{Provider: "fixture", Model: "local", RawPayloadSHA256: strings.Repeat("b", 64)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.RecordUsage(ambiguous); err == nil {
		t.Fatal("expected strict unknown-cost rejection")
	}
}

func TestBudgetControllerRestoresConsumedStateAndHonorsTaskDeadline(t *testing.T) {
	budget, err := NewResearchBudget(ResearchBudget{Version: "resume", MaxWallClock: time.Minute, ToolTimeout: time.Second, MaxSteps: 2, MaxToolCalls: 2, MaxModelCalls: 1, MaxRetries: 1, MaxInputTokens: 10, MaxOutputTokens: 10, MaxReasoningTokens: 10, MaximumModelTier: BudgetTierLocal, MaxEstimatedCostUSD: 1, MaxActualCostUSD: 1})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC()
	controller, err := NewBudgetControllerWithState(budget, BudgetState{Steps: 1, ToolCalls: 1}, started)
	if err != nil {
		t.Fatal(err)
	}
	if err := controller.ReserveToolCall(); err != nil {
		t.Fatal(err)
	}
	if err := controller.ReserveToolCall(); err == nil {
		t.Fatal("restored tool-call budget was not enforced")
	}
	if _, err := NewBudgetControllerWithState(budget, BudgetState{ToolCalls: 3}, started); err == nil {
		t.Fatal("over-budget restored state was accepted")
	}
	expired, err := NewBudgetControllerWithState(budget, BudgetState{}, started.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := expired.Context(context.Background()); err == nil {
		t.Fatal("restored task deadline was not enforced")
	}
}

func TestNormalizeProviderUsagePreservesReportedAndDerivedCost(t *testing.T) {
	pricing := PricingSnapshot{Provider: "fixture", Model: "local", EffectiveAt: time.Date(2026, 9, 7, 18, 0, 0, 0, time.UTC), InputUSDPer1K: 1, CachedInputUSDPer1K: 0.1, OutputUSDPer1K: 2, ReasoningUSDPer1K: 3, Currency: "USD"}
	input, cached, output, reasoning := 100, 20, 10, 5
	usage, err := NormalizeProviderUsage(ProviderUsage{Provider: "fixture", Model: "local", InputTokens: &input, CachedInputTokens: &cached, OutputTokens: &output, ReasoningTokens: &reasoning, FinishStatus: "stop", RawPayloadSHA256: strings.Repeat("c", 64)}, &pricing)
	if err != nil {
		t.Fatal(err)
	}
	if usage.CostStatus != BudgetCostKnown || usage.ActualCostUSD != 0.117 {
		t.Fatalf("unexpected derived usage: %#v", usage)
	}
	reported := 0.77
	usage, err = NormalizeProviderUsage(ProviderUsage{Provider: "fixture", Model: "local", InputTokens: &input, OutputTokens: &output, ReasoningTokens: &reasoning, ReportedCostUSD: &reported, RawPayloadSHA256: strings.Repeat("d", 64)}, nil)
	if err != nil || usage.CostStatus != BudgetCostKnown || usage.ActualCostUSD != reported {
		t.Fatalf("reported usage was not preserved: %#v err=%v", usage, err)
	}
}

func TestBudgetContextHonoursCancellation(t *testing.T) {
	controller := budgetFixture(t)
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := controller.Context(parent); err == nil {
		t.Fatal("expected cancelled parent rejection")
	}
}

func intPointer(value int) *int           { return &value }
func floatPointer(value float64) *float64 { return &value }
