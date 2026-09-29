package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalPortfolioRiskPolicyRequiresExplicitValidFile(t *testing.T) {
	t.Setenv(portfolioRiskPolicyFileEnv, "")
	if _, err := loadCanonicalPortfolioRiskPolicy(); err == nil {
		t.Fatal("missing risk policy was accepted")
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{"currency":"USD","maximum_leverage":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(portfolioRiskPolicyFileEnv, path)
	if _, err := loadCanonicalPortfolioRiskPolicy(); err == nil {
		t.Fatal("policy missing identity and contract required fields was accepted")
	}
	if err := os.WriteFile(path, []byte(`{"version":"portfolio-v1","currency":"USD","maximum_leverage":1,"maximum_position_value":20000}`), 0600); err != nil {
		t.Fatal(err)
	}
	policy, err := loadCanonicalPortfolioRiskPolicy()
	if err != nil || policy.PolicyID == "" || policy.MaximumLeverage == nil || *policy.MaximumLeverage > 1 {
		t.Fatalf("valid explicit policy=%#v err=%v", policy, err)
	}
	if err := os.WriteFile(path, []byte(`{"version":"portfolio-v1","currency":"USD","maximum_leverage":1.1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCanonicalPortfolioRiskPolicy(); err == nil {
		t.Fatal("policy above 1x was accepted")
	}
}

func Test02BTechnicalProofPortfolioRiskPolicyUsesOnlySupportedConstraints(t *testing.T) {
	path, err := filepath.Abs("../../config/core-readiness-02b-portfolio-risk-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(portfolioRiskPolicyFileEnv, path)
	policy, err := loadCanonicalPortfolioRiskPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if policy.ContractVersion != "jax.portfolio.risk_policy/v1" || policy.Version != "core-readiness-02b-technical-proof-v1" || policy.Currency != "USD" || policy.MaximumPositionValue == nil || *policy.MaximumPositionValue != 50000 || policy.MaximumRiskAllocation == nil || *policy.MaximumRiskAllocation != .02 || policy.MaximumLeverage == nil || *policy.MaximumLeverage != 1 || policy.PolicyID == "" {
		t.Fatalf("unexpected 02B portfolio risk policy: %+v", policy)
	}
	if policy.MaximumConcentration != nil || policy.MaximumGrossExposure != nil || policy.MaximumNetExposure != nil || policy.MinimumCash != nil || policy.MaximumStressLoss != nil {
		t.Fatalf("unsupported optional risk dimensions were invented: %+v", policy)
	}
}
