package ejlayer

import (
	"testing"

	domain "jax-trading-assistant/internal/domain/ejlayer"
)

// TestShouldAbstainLowConfidence verifies abstention when confidence is too low
func TestShouldAbstainLowConfidence(t *testing.T) {
	if !domain.ShouldAbstain(0.1, 0.8) {
		t.Error("expected abstain with confidence=0.1")
	}
}

// TestShouldAbstainDepletedBudget verifies abstention when uncertainty budget is depleted
func TestShouldAbstainDepletedBudget(t *testing.T) {
	if !domain.ShouldAbstain(0.8, 0.1) {
		t.Error("expected abstain with budget=0.1")
	}
}

// TestShouldNotAbstainGoodConditions verifies no abstention under healthy conditions
func TestShouldNotAbstainGoodConditions(t *testing.T) {
	if domain.ShouldAbstain(0.8, 0.8) {
		t.Error("expected no abstain with confidence=0.8, budget=0.8")
	}
}

// TestComputeUncertaintyBudget_NoFactors verifies full budget when no factors present
func TestComputeUncertaintyBudget_NoFactors(t *testing.T) {
	b := domain.ComputeUncertaintyBudget(nil)
	if b != 1.0 {
		t.Errorf("expected 1.0, got %f", b)
	}
}

// TestComputeUncertaintyBudget_FullyDrained verifies zero budget when fully drained
func TestComputeUncertaintyBudget_FullyDrained(t *testing.T) {
	factors := []domain.UncertaintyFactor{
		{Name: "volatility", Weight: 1.0, Score: 1.0},
	}
	b := domain.ComputeUncertaintyBudget(factors)
	if b != 0.0 {
		t.Errorf("expected 0.0 for fully drained, got %f", b)
	}
}

// TestDeriveDominance verifies majority dominance is returned
func TestDeriveDominance(t *testing.T) {
	eps := []*domain.Episode{
		{ContextDominance: domain.DomTechnical},
		{ContextDominance: domain.DomTechnical},
		{ContextDominance: domain.DomVolatility},
	}
	got := deriveDominance(eps)
	if got != domain.DomTechnical {
		t.Errorf("expected DomTechnical, got %s", got)
	}
}

// TestDeriveDominance_Empty verifies unclear returned for empty slice
func TestDeriveDominance_Empty(t *testing.T) {
	got := deriveDominance(nil)
	if got != domain.DomUnclear {
		t.Errorf("expected DomUnclear for empty, got %s", got)
	}
}

// TestWeightedConfidenceNoHistory verifies neutral result with no history
func TestWeightedConfidenceNoHistory(t *testing.T) {
	c := domain.WeightedConfidence(nil)
	if c != 0.5 {
		t.Errorf("expected 0.5 for no history, got %f", c)
	}
}
