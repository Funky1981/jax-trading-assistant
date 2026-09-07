package portfoliorisk

import (
	"context"
	"testing"
	"time"
)

func TestRiskDecisionAuditIsReconstructableAndAppendOnly(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	decision := EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	if err := decision.Validate(); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryRiskDecisionStore()
	if err := store.Save(context.Background(), decision); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Get(context.Background(), decision.DecisionID)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DecisionID != decision.DecisionID || loaded.PolicyID != policy.PolicyID || loaded.PortfolioSnapshotID != canonical.SnapshotID {
		t.Fatalf("audit lost identity: %#v", loaded)
	}
	tampered := decision
	tampered.PolicyID = "rpol_tampered"
	if err := store.Save(context.Background(), tampered); err == nil {
		t.Fatal("audit accepted same identity with changed policy")
	}
}

func TestRiskDecisionValidationRejectsReasonMismatchAndAuthority(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	decision := EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	decision.ReasonCodes = []ReasonCode{ReasonRejectGrossLimit}
	if err := decision.Validate(); err == nil {
		t.Fatal("accept with reject reason validated")
	}
	decision = EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	decision.ExecutionAuthority = "ORDER"
	if err := decision.Validate(); err == nil {
		t.Fatal("unsafe authority validated")
	}
}
