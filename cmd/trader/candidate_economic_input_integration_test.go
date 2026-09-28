package main

import (
	"context"
	"testing"
	"time"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"

	"github.com/google/uuid"
)

func TestCandidateEconomicInputPostgresPersistenceReplayAndHistoricalNoBackfill(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	candidateID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO candidate_trades (id, strategy_instance_id, symbol, signal_type, session_date, status, data_provenance) VALUES ($1,$2,'QQQ','BUY',CURRENT_DATE,'awaiting_approval','test-fixture')`, candidateID, uuid.New())
	if err != nil {
		t.Fatalf("insert disposable candidate fixture: %v", err)
	}
	store := candidatesmod.NewStore(pool)
	input, err := candidatesmod.BuildCandidateEconomicInput(candidatesmod.CandidateEconomicInput{
		CandidateID: candidateID, InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq",
		IdentitySource: "fixture-only", IdentityPolicyVersion: "fixture-v1", RiskAllocation: .0125,
		RequestedLeverage: .8, SizingPolicyID: "fixture-risk-request", SizingPolicyVersion: "v2",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PersistCandidateEconomicInput(ctx, input); err != nil {
		t.Fatal(err)
	}
	if err := store.PersistCandidateEconomicInput(ctx, input); err != nil {
		t.Fatalf("identical replay must be idempotent: %v", err)
	}
	loaded, err := store.GetCandidateEconomicInput(ctx, candidateID)
	if err != nil {
		t.Fatal(err)
	}
	if !input.SameEconomicRequest(loaded) || loaded.InstrumentID != "instrument-test-qqq" || loaded.IssuerID != "issuer-test-qqq" || loaded.RiskAllocation != .0125 || loaded.RequestedLeverage != .8 {
		t.Fatalf("persisted economic request differs: %+v", loaded)
	}
	conflict := input
	conflict.IssuerID = "different-test-issuer"
	conflict.ContentIdentity = ""
	conflict, err = candidatesmod.BuildCandidateEconomicInput(conflict, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PersistCandidateEconomicInput(ctx, conflict); err == nil {
		t.Fatal("conflicting replay must fail closed")
	}
	if _, err := pool.Exec(ctx, `UPDATE candidate_economic_inputs SET issuer_id='mutated' WHERE candidate_id=$1`, candidateID); err == nil {
		t.Fatal("economic input update must be rejected")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM candidate_economic_inputs WHERE candidate_id=$1`, candidateID); err == nil {
		t.Fatal("economic input delete must be rejected")
	}

	legacyCandidateID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO candidate_trades (id, strategy_instance_id, symbol, signal_type, session_date, status, data_provenance) VALUES ($1,$2,'QQQ','BUY',CURRENT_DATE,'awaiting_approval','historical-fixture')`, legacyCandidateID, uuid.New())
	if err != nil {
		t.Fatalf("insert historical candidate fixture: %v", err)
	}
	var legacyInputs int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_economic_inputs WHERE candidate_id=$1`, legacyCandidateID).Scan(&legacyInputs); err != nil {
		t.Fatal(err)
	}
	if legacyInputs != 0 {
		t.Fatalf("historical candidate was backfilled: %d rows", legacyInputs)
	}
	if err := store.MarkEconomicInputUnavailable(ctx, legacyCandidateID, "canonical_economic_inputs_unavailable"); err != nil {
		t.Fatal(err)
	}
	var gateStatus, approvalStatus string
	if err := pool.QueryRow(ctx, `SELECT gate_status, approval_status FROM candidate_trades WHERE id=$1`, legacyCandidateID).Scan(&gateStatus, &approvalStatus); err != nil {
		t.Fatal(err)
	}
	if gateStatus != candidatesmod.GateStatusRiskPending || approvalStatus != candidatesmod.ApprovalStatusRiskNotReady {
		t.Fatalf("historical candidate not fail-closed: gate=%s approval=%s", gateStatus, approvalStatus)
	}

	configuredCandidateID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO candidate_trades (id, strategy_instance_id, symbol, signal_type, session_date, status, data_provenance) VALUES ($1,$2,'QQQ','BUY',CURRENT_DATE,'awaiting_approval','test-fixture')`, configuredCandidateID, uuid.New())
	if err != nil {
		t.Fatalf("insert configured candidate fixture: %v", err)
	}
	policy := &candidateEconomicPolicy{PolicyVersion: "fixture-policy-v1", IdentityPolicy: "fixture-identity-v1", IdentitySource: "test-fixture-map",
		SizingPolicyID: "fixture-request", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: .75,
		Instruments: map[string]candidateEconomicIdentity{"QQQ": {InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq"}}}
	promoter := newWorldMonitorOpportunityPromoter(pool)
	promoter.economicPolicy = policy
	_, outcome, err := promoter.reviewCandidateRisk(ctx, configuredCandidateID, uuid.New(), uuid.NullUUID{})
	if err != nil || outcome.ReasonCode != "portfolio_risk_not_performed" {
		t.Fatalf("configured economic input should persist but remain before portfolio risk: outcome=%+v err=%v", outcome, err)
	}
	configuredInput, err := store.GetCandidateEconomicInput(ctx, configuredCandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if configuredInput.InstrumentID != "instrument-test-qqq" || configuredInput.IssuerID != "issuer-test-qqq" || configuredInput.RiskAllocation != .01 || configuredInput.RequestedLeverage != .75 {
		t.Fatalf("configured world-monitor input differs: %+v", configuredInput)
	}
}
