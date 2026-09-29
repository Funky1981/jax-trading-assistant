package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCandidateEconomicPolicyFailsClosedForMissingIdentityOrSizing(t *testing.T) {
	base := candidateEconomicPolicy{PolicyVersion: "policy-v1", IdentityPolicy: "identity-v1", IdentitySource: "reviewed-map",
		SizingPolicyID: "request-policy", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: 1,
		SlippageAllowance: floatPointer(.25),
		Instruments:       map[string]candidateEconomicIdentity{"QQQ": {InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq"}}}
	if _, err := base.build("QQQ", uuid.New(), time.Now()); err != nil {
		t.Fatalf("explicit fixture policy rejected: %v", err)
	}
	missingIssuer := base
	missingIssuer.Instruments = map[string]candidateEconomicIdentity{"QQQ": {InstrumentID: "instrument-test-qqq"}}
	if _, err := missingIssuer.build("QQQ", uuid.New(), time.Now()); err == nil {
		t.Fatal("missing issuer identity must fail closed")
	}
	missingSizing := base
	missingSizing.RiskAllocation = 0
	if _, err := missingSizing.build("QQQ", uuid.New(), time.Now()); err == nil {
		t.Fatal("missing risk allocation must fail closed")
	}
	if _, err := base.build("UNKNOWN", uuid.New(), time.Now()); err == nil {
		t.Fatal("unmapped symbol must not be guessed")
	}
}

func TestLoadCandidateEconomicPolicyRequiresExplicitFile(t *testing.T) {
	t.Setenv(candidateEconomicPolicyEnv, "")
	if _, err := loadCandidateEconomicPolicy(); err == nil {
		t.Fatal("missing policy path must fail closed")
	}
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{"policy_version":"v1","identity_policy_version":"id-v1","identity_source":"reviewed","sizing_policy_id":"s","sizing_policy_version":"v1","risk_allocation":0.01,"requested_leverage":1,"slippage_allowance":0.25,"instruments":{"QQQ":{"instrument_id":"instrument-test-qqq","issuer_id":"issuer-test-qqq"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, path)
	if _, err := loadCandidateEconomicPolicy(); err != nil {
		t.Fatalf("explicit policy rejected: %v", err)
	}
}

func Test02BTechnicalProofCandidatePolicyContainsOnlyVerifiedApprovedCatalog(t *testing.T) {
	path, err := filepath.Abs("../../config/core-readiness-02b-candidate-economic-policy.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, path)
	policy, err := loadCandidateEconomicPolicy()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"SPY", "QQQ", "DIA", "IWM", "XLK", "XLF", "XLE", "SMH", "SOXX", "TLT", "GLD"}
	if policy.PolicyVersion != "core-readiness-02b-technical-proof-v1" || policy.IdentityPolicy != "jax-us-etf-identity-v1" || policy.IdentitySource != "operator-reviewed-official-fund-identity-mapping" || policy.RiskAllocation != .005 || policy.RequestedLeverage != 1 || policy.SlippageAllowance == nil || *policy.SlippageAllowance != .50 || len(policy.Instruments) != len(want) {
		t.Fatalf("unexpected technical proof policy: %+v", policy)
	}
	for _, symbol := range want {
		identity, ok := policy.Instruments[symbol]
		if !ok || identity.InstrumentID != "jax.instrument.us.etf."+strings.ToLower(symbol) || strings.TrimSpace(identity.IssuerID) == "" {
			t.Errorf("missing explicit approved identity for %s: %+v", symbol, identity)
		}
	}
	for _, excluded := range []string{"TQQQ", "SQQQ", "UVXY", "VXX"} {
		if _, ok := policy.Instruments[excluded]; ok {
			t.Errorf("excluded instrument %s appears in proof policy", excluded)
		}
	}
}
