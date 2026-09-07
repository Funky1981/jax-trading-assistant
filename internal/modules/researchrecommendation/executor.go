package researchrecommendation

import (
	"errors"
	"fmt"
)

var ErrResearchRetryLimit = errors.New("structured research retry limit exhausted")

// StructuredResearchProvider is deliberately injected at the boundary. The
// contract package does not select a hosted provider or perform network I/O.
type StructuredResearchProvider interface {
	GenerateStructuredResearch(ContextPlan) (StructuredResearchOutput, error)
}

// BudgetAwareResearchProvider can expose a preflight cost classification. A
// caller that uses a paid provider must pass through this budget gate before
// any generation attempt is made.
type BudgetAwareResearchProvider interface {
	StructuredResearchProvider
	Paid() bool
	EstimatedCostUSD(ContextPlan) float64
}

type ResearchExecutionStats struct {
	Attempts           int    `json:"attempts"`
	ValidationFailures int    `json:"validation_failures"`
	ProviderFailures   int    `json:"provider_failures"`
	RetryLimit         int    `json:"retry_limit"`
	LastError          string `json:"last_error,omitempty"`
}

func ExecuteStructuredResearch(plan ContextPlan, packet EvidencePacket, provider StructuredResearchProvider) (StructuredResearchOutput, ResearchExecutionStats, error) {
	stats := ResearchExecutionStats{RetryLimit: plan.Budget.MaxRetries}
	if err := packet.Validate(); err != nil {
		return StructuredResearchOutput{}, stats, err
	}
	if err := plan.Validate(); err != nil {
		return StructuredResearchOutput{}, stats, err
	}
	if plan.Oversize {
		return StructuredResearchOutput{}, stats, fmt.Errorf("structured research cannot execute an oversize context plan")
	}
	if provider == nil {
		return StructuredResearchOutput{}, stats, fmt.Errorf("structured research provider is required")
	}
	if budgeted, ok := provider.(BudgetAwareResearchProvider); ok {
		estimatedCost := budgeted.EstimatedCostUSD(plan)
		if !finite(estimatedCost) || estimatedCost < 0 || (budgeted.Paid() && estimatedCost > plan.Budget.MaxEstimatedCostUSD) || (budgeted.Paid() && plan.Budget.MaxEstimatedCostUSD == 0) {
			return StructuredResearchOutput{}, stats, fmt.Errorf("paid structured research provider fails the explicit cost budget gate")
		}
	}

	for attempt := 0; attempt <= plan.Budget.MaxRetries; attempt++ {
		stats.Attempts++
		output, err := provider.GenerateStructuredResearch(plan)
		if err != nil {
			stats.ProviderFailures++
			stats.LastError = err.Error()
			continue
		}
		// RetryCount is execution-owned provenance. A provider cannot under-report
		// attempts by returning a stale or fabricated retry value.
		output.Inference.RetryCount = attempt
		if err := ValidateStructuredResearchOutput(plan, packet, output); err != nil {
			stats.ValidationFailures++
			stats.LastError = err.Error()
			continue
		}
		return output, stats, nil
	}
	return StructuredResearchOutput{}, stats, fmt.Errorf("%w: attempts=%d validation_failures=%d provider_failures=%d last_error=%s", ErrResearchRetryLimit, stats.Attempts, stats.ValidationFailures, stats.ProviderFailures, stats.LastError)
}
