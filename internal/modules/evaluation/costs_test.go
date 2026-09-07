package evaluation

import (
	"math"
	"testing"
	"time"
)

func costPolicyFixture(t *testing.T) CostSlippagePolicy {
	t.Helper()
	policy, err := NewCostSlippagePolicy(CostSlippagePolicy{Version: "v1", CommissionPerOrder: 0.5, CommissionBPS: 10, SpreadBPS: 5, SlippageBPS: 10, MarketImpactBPS: 15, BorrowBPSPerDay: 20, ReferencePriceRule: "NEXT_AVAILABLE_OBSERVATION", AssumptionSource: "Phase 07 deterministic fixture", CreatedAt: time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestApplyHypotheticalTradingCostIsDeterministicAndNonExecution(t *testing.T) {
	policy := costPolicyFixture(t)
	cost, err := ApplyHypotheticalTradingCost(policy, "BUY", 100, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := cost.Validate(policy); err != nil {
		t.Fatal(err)
	}
	if math.Abs(cost.TotalCost-8.5) > 1e-9 || math.Abs(cost.AdjustedPrice-100.3) > 1e-9 {
		t.Fatalf("unexpected deterministic cost: %#v", cost)
	}
	if cost.ExecutionAuthority != "NONE" || cost.CreatesFill {
		t.Fatalf("cost calculation created execution state: %#v", cost)
	}
	second, err := ApplyHypotheticalTradingCost(policy, "BUY", 100, 10, 2)
	if err != nil || second.TotalCost != cost.TotalCost || second.AdjustedPrice != cost.AdjustedPrice {
		t.Fatal("same frozen assumptions did not reproduce the cost")
	}
}

func TestCostSlippagePolicyRejectsZeroFrictionUnlessDiagnostic(t *testing.T) {
	base := CostSlippagePolicy{Version: "v1", ReferencePriceRule: "NEXT_AVAILABLE_OBSERVATION", AssumptionSource: "fixture", CreatedAt: time.Date(2026, 9, 7, 16, 0, 0, 0, time.UTC)}
	if _, err := NewCostSlippagePolicy(base); err == nil {
		t.Fatal("expected silent zero-friction baseline rejection")
	}
	base.DiagnosticBaseline = true
	if _, err := NewCostSlippagePolicy(base); err != nil {
		t.Fatal(err)
	}
}

func TestApplyHypotheticalTradingCostRejectsInvalidInputs(t *testing.T) {
	policy := costPolicyFixture(t)
	if _, err := ApplyHypotheticalTradingCost(policy, "BUY", 100, 10, -1); err == nil {
		t.Fatal("expected negative holding period rejection")
	}
	if _, err := ApplyHypotheticalTradingCost(policy, "BUY", math.NaN(), 10, 0); err == nil {
		t.Fatal("expected non-finite price rejection")
	}
	if _, err := ApplyHypotheticalTradingCost(policy, "HOLD", 100, 10, 0); err == nil {
		t.Fatal("expected unsupported side rejection")
	}
}
