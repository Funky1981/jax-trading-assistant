package researchrecommendation

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fixtureResearchProvider struct {
	outputs []StructuredResearchOutput
	errors  []error
	calls   int
}

type paidFixtureResearchProvider struct {
	fixtureResearchProvider
	estimated float64
}

func (provider *paidFixtureResearchProvider) Paid() bool { return true }

func (provider *paidFixtureResearchProvider) EstimatedCostUSD(ContextPlan) float64 {
	return provider.estimated
}

func (provider *fixtureResearchProvider) GenerateStructuredResearch(ContextPlan) (StructuredResearchOutput, error) {
	index := provider.calls
	provider.calls++
	if index < len(provider.errors) && provider.errors[index] != nil {
		return StructuredResearchOutput{}, provider.errors[index]
	}
	if index >= len(provider.outputs) {
		return StructuredResearchOutput{}, fmt.Errorf("fixture provider exhausted")
	}
	return provider.outputs[index], nil
}

func TestExecuteStructuredResearchValidatesAndBoundsRetries(t *testing.T) {
	packet, plan := researchFixture(t)
	invalid := validResearchOutput(t, packet, plan)
	invalid.ThesisEvidenceIDs = []string{"epi_" + strings.Repeat("f", 64)}
	valid := validResearchOutput(t, packet, plan)
	provider := &fixtureResearchProvider{outputs: []StructuredResearchOutput{invalid, valid}}
	output, stats, err := ExecuteStructuredResearch(plan, packet, provider)
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || stats.Attempts != 2 || stats.ValidationFailures != 1 || output.Inference.RetryCount != 1 {
		t.Fatalf("retry provenance/boundary incorrect: calls=%d stats=%+v output=%+v", provider.calls, stats, output.Inference)
	}
}

func TestExecuteStructuredResearchFailsClosedAtRetryLimit(t *testing.T) {
	packet, plan := researchFixture(t)
	provider := &fixtureResearchProvider{errors: []error{errors.New("provider unavailable"), errors.New("provider unavailable")}}
	_, stats, err := ExecuteStructuredResearch(plan, packet, provider)
	if !errors.Is(err, ErrResearchRetryLimit) || stats.Attempts != plan.Budget.MaxRetries+1 || stats.ProviderFailures != stats.Attempts {
		t.Fatalf("retry limit was not enforced: stats=%+v err=%v", stats, err)
	}
}

func TestExecuteStructuredResearchRejectsOversizeAndPaidBudgetViolations(t *testing.T) {
	packet, plan := researchFixture(t)
	plan.Oversize = true
	plan.OversizeDecision = "ABSTAIN_NO_TRADE_CONTEXT_BUDGET"
	if _, _, err := ExecuteStructuredResearch(plan, packet, &fixtureResearchProvider{}); err == nil {
		t.Fatal("oversize research plan executed")
	}
	paid := &paidFixtureResearchProvider{estimated: plan.Budget.MaxEstimatedCostUSD + 1}
	plan.Oversize = false
	plan.OversizeDecision = "NONE"
	if _, _, err := ExecuteStructuredResearch(plan, packet, paid); err == nil {
		t.Fatal("paid provider exceeded its explicit cost budget")
	}
}
