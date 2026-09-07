package harness

import (
	"context"
	"testing"
)

func TestPhase08ExitHarnessDemonstratesBoundedDurableResearch(t *testing.T) {
	proof, err := RunPhase08ExitHarness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := proof.Validate(); err != nil {
		t.Fatal(err)
	}
	if proof.Comparison.Agent.UnsupportedClaimRate >= proof.Comparison.Baseline.UnsupportedClaimRate || proof.Comparison.Agent.CostUSD != 0 {
		t.Fatalf("evaluation did not show bounded quality improvement without spend: %#v", proof.Comparison)
	}
}

func TestResearchRoutingIsExplicitAndDoesNotSilentlyEscalate(t *testing.T) {
	policy := DefaultResearchModelRoutingPolicy()
	route, err := policy.Select("research_gap_analysis")
	if err != nil || route.Tier != BudgetTierLocal || route.Model != "local-research-v1" {
		t.Fatalf("unexpected explicit route: %#v err=%v", route, err)
	}
	if _, err := policy.Select("unknown_research_task"); err == nil {
		t.Fatal("expected unknown task route denial")
	}
}
