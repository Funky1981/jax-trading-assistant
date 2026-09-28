package main

import (
	"os"
	"path/filepath"
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
