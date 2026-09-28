package candidates

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildCandidateEconomicInputIdentityIsDeterministicAndValidates(t *testing.T) {
	candidateID := uuid.New()
	input := CandidateEconomicInput{CandidateID: candidateID, InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq",
		IdentitySource: "fixture", IdentityPolicyVersion: "fixture-v1", RiskAllocation: 0.01, RequestedLeverage: 1,
		SizingPolicyID: "fixture-sizing", SizingPolicyVersion: "v1"}
	first, err := BuildCandidateEconomicInput(input, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildCandidateEconomicInput(input, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if !first.SameEconomicRequest(second) || first.CreatedAt.Equal(second.CreatedAt) {
		t.Fatalf("identity must be stable across replay while created_at remains audit time: first=%+v second=%+v", first, second)
	}
	input.IssuerID = ""
	if _, err := BuildCandidateEconomicInput(input, time.Now()); err == nil {
		t.Fatal("missing issuer identity must fail closed")
	}
}

func TestBuildCandidateEconomicInputRejectsConflictingIdentity(t *testing.T) {
	input := CandidateEconomicInput{CandidateID: uuid.New(), InstrumentID: "i", IssuerID: "e", IdentitySource: "fixture",
		IdentityPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: 1, SizingPolicyID: "p", SizingPolicyVersion: "v1"}
	built, err := BuildCandidateEconomicInput(input, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	built.IssuerID = "different"
	if _, err := BuildCandidateEconomicInput(built, time.Now()); err == nil {
		t.Fatal("mutated content identity must be rejected")
	}
}

func TestCandidateEconomicInputIdentityBindsExplicitSlippageAllowance(t *testing.T) {
	allowance := .25
	input := CandidateEconomicInput{CandidateID: uuid.New(), InstrumentID: "i", IssuerID: "e", IdentitySource: "fixture",
		IdentityPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: 1, SlippageAllowance: &allowance,
		SizingPolicyID: "p", SizingPolicyVersion: "v1"}
	built, err := BuildCandidateEconomicInput(input, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	changedAllowance := .5
	built.SlippageAllowance = &changedAllowance
	if _, err := BuildCandidateEconomicInput(built, time.Now().UTC()); err == nil {
		t.Fatal("changed slippage allowance must invalidate the economic content identity")
	}
	negative := -0.01
	input.SlippageAllowance = &negative
	input.ContentIdentity = ""
	if _, err := BuildCandidateEconomicInput(input, time.Now().UTC()); err == nil {
		t.Fatal("negative slippage allowance must fail closed")
	}
}
