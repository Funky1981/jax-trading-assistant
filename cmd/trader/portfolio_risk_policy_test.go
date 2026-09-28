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
