package portfoliorisk

import (
	"testing"
	"time"
)

func fixtureAnalytics(t *testing.T) (PortfolioSnapshot, ExposureAnalytics, RiskPolicy) {
	t.Helper()
	snapshot := fixtureSnapshot()
	policy, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, analytics, policy
}

func recommendation(value float64) RecommendationRiskInput {
	return RecommendationRiskInput{RecommendationID: "rec-phase09-1", InstrumentID: "NYSE:XYZ", Currency: "USD", SignedMarketValue: value, RiskAllocation: Limit(.01), RequestedLeverage: Limit(1), Confidence: Limit(.9), ExecutionAuthority: "NONE", QuantResultIDs: []string{"quant-result-frozen-1"}}
}

func TestRiskDecisionAcceptsWithinPolicy(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	result := EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	if result.Outcome != DecisionAccept || len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != ReasonAcceptWithinPolicy {
		t.Fatalf("decision=%#v", result)
	}
	if result.DecisionID == "" || result.ExecutionAuthority != "NONE" {
		t.Fatalf("decision identity/safety lost: %#v", result)
	}
}

func TestRiskDecisionAmendsToDeterministicCapacity(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	input := recommendation(10000)
	input.InstrumentID = "NYSE:ABC"
	result := EvaluateRecommendation(input, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if result.Outcome != DecisionAmend || result.ResultingValue != 5000 {
		t.Fatalf("decision=%#v", result)
	}
	if len(result.ReasonCodes) == 0 || result.ReasonCodes[0] != ReasonAmendPositionCap {
		t.Fatalf("reason codes=%v", result.ReasonCodes)
	}
}

func TestRiskDecisionRejectsFailuresAndConfidenceCannotOverride(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	tests := []struct {
		name           string
		mutate         func(*RecommendationRiskInput)
		mutateSnapshot func(*PortfolioSnapshot)
		want           ReasonCode
	}{
		{"execution authority", func(r *RecommendationRiskInput) { r.ExecutionAuthority = "APPROVE" }, nil, ReasonRejectExecutionAuthority},
		{"leverage above one", func(r *RecommendationRiskInput) { r.RequestedLeverage = Limit(2) }, nil, ReasonRejectLeverageLimit},
		{"unknown risk allocation", func(r *RecommendationRiskInput) { r.RiskAllocation = nil }, nil, ReasonRejectUnknownExposure},
		{"stale portfolio", nil, func(s *PortfolioSnapshot) {
			s.AsOf = stateNow.Add(-time.Hour)
			s.CapturedAt = stateNow.Add(-time.Hour + time.Minute)
			s.Positions[0].ValuationAsOf = s.AsOf
		}, ReasonRejectStalePortfolio},
		{"unknown cash", nil, func(s *PortfolioSnapshot) { s.Cash = UnknownNumber("omitted") }, ReasonRejectUnknownExposure},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := recommendation(5000)
			if tt.mutate != nil {
				tt.mutate(&input)
			}
			candidate := snapshot
			if tt.mutateSnapshot != nil {
				tt.mutateSnapshot(&candidate)
			}
			result := EvaluateRecommendation(input, candidate, analytics, policy, stateNow, 10*time.Minute)
			if result.Outcome != DecisionReject || result.ReasonCodes[0] != tt.want {
				t.Fatalf("decision=%#v", result)
			}
		})
	}
	// A high model confidence does not change a deterministic rejection.
	input := recommendation(5000)
	input.InstrumentID = "NYSE:ABC"
	input.SignedMarketValue = 100000
	input.Confidence = Limit(1)
	result := EvaluateRecommendation(input, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if result.Outcome == DecisionAccept {
		t.Fatal("confidence overrode risk limits")
	}
}
