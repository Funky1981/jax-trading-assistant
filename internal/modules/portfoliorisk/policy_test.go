package portfoliorisk

import (
	"errors"
	"math"
	"testing"
)

func fixturePolicy() RiskPolicy {
	return RiskPolicy{Version: "phase09-fixture-v1", Currency: "USD", MaximumPositionValue: Limit(25000), MaximumConcentration: Limit(.25), MaximumGrossExposure: Limit(90000), MaximumNetExposure: Limit(60000), MaximumRiskAllocation: Limit(.02), MinimumCash: Limit(10000), MaximumLeverage: Limit(1), MaximumStressLoss: Limit(.20)}
}

func TestRiskPolicyIsVersionedAndStable(t *testing.T) {
	a, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if a.PolicyID == "" || a.PolicyID != b.PolicyID || a.ContractVersion != RiskPolicyContractVersion {
		t.Fatalf("unstable policy identity: %#v %#v", a, b)
	}
	changed := fixturePolicy()
	changed.MaximumConcentration = Limit(.20)
	c, err := BuildRiskPolicy(changed)
	if err != nil {
		t.Fatal(err)
	}
	if c.PolicyID == a.PolicyID {
		t.Fatal("changing a limit did not change policy identity")
	}
}

func TestRiskPolicyRejectsMissingOrUnsafeLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*RiskPolicy)
	}{
		{"missing leverage", func(p *RiskPolicy) { p.MaximumLeverage = nil }},
		{"leverage above safety ceiling", func(p *RiskPolicy) { p.MaximumLeverage = Limit(1.01) }},
		{"unsupported currency", func(p *RiskPolicy) { p.Currency = "XYZ" }},
		{"nan limit", func(p *RiskPolicy) { p.MaximumConcentration = Limit(math.NaN()) }},
		{"negative cash", func(p *RiskPolicy) { p.MinimumCash = Limit(-1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := fixturePolicy()
			tt.mutate(&policy)
			if _, err := BuildRiskPolicy(policy); err == nil {
				t.Fatal("unsafe policy was accepted")
			}
		})
	}
}

func TestPolicyIdentityClaimCannotBeReusedForChangedContent(t *testing.T) {
	policy, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	policy.MaximumGrossExposure = Limit(1)
	if _, err := BuildRiskPolicy(policy); err == nil || !errors.Is(err, ErrPolicyIdentity) {
		t.Fatalf("error=%v, want policy identity error", err)
	}
}
