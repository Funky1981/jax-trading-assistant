package portfoliorisk

import (
	"context"
	"testing"
	"time"
)

// TestPhase09ExitHarness is the reproducible Phase-09 gate proof. Fixtures are
// synthetic and frozen: acceptance is a mechanics result, not a profitability
// or real-portfolio claim.
func TestPhase09ExitHarness(t *testing.T) {
	start := fixtureSnapshot()
	start.Positions = append(start.Positions,
		Position{InstrumentID: "NYSE:XYZ", InstrumentResolved: true, Currency: "USD", SignedQuantity: -100, Price: KnownNumber(100, "phase09-fixture"), MarketValue: KnownNumber(-10000, "phase09-fixture"), ValuationAsOf: start.AsOf, PriceSource: "phase09-fixture", Provenance: []string{"fixture:price:xyz"}},
		Position{InstrumentID: "NYSE:LMN", InstrumentResolved: true, Currency: "USD", SignedQuantity: 50, Price: KnownNumber(100, "phase09-fixture"), MarketValue: KnownNumber(5000, "phase09-fixture"), ValuationAsOf: start.AsOf, PriceSource: "phase09-fixture", Provenance: []string{"fixture:price:lmn"}},
	)
	snapshot, err := BuildSnapshot(start)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SnapshotID == "" || analytics.AnalyticsID == "" || policy.PolicyID == "" {
		t.Fatal("canonical identities were not produced")
	}

	accept := EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	if accept.Outcome != DecisionAccept || accept.ReasonCodes[0] != ReasonAcceptWithinPolicy {
		t.Fatalf("accept proof failed: %#v", accept)
	}
	amendInput := recommendation(10000)
	amendInput.InstrumentID = "NYSE:ABC"
	amend := EvaluateRecommendation(amendInput, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if amend.Outcome != DecisionAmend || amend.ResultingValue >= amend.RequestedValue {
		t.Fatalf("amend proof failed: %#v", amend)
	}
	rejectInput := recommendation(5000)
	rejectInput.RequestedLeverage = Limit(2)
	reject := EvaluateRecommendation(rejectInput, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if reject.Outcome != DecisionReject || reject.ReasonCodes[0] != ReasonRejectLeverageLimit {
		t.Fatalf("reject proof failed: %#v", reject)
	}

	proposal, err := CalculatePositionProposal(PositionProposalInput{RecommendationID: accept.RecommendationID, InstrumentID: "NYSE:XYZ", Currency: "USD", SignedMarketValue: 5000, EntryPrice: 100, StopPrice: 90, RiskAllocation: Limit(.01)}, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := BuildStressScenario(StressScenario{Version: "phase09-v1", Name: "frozen correlated decline", DefaultShock: Limit(-.10)})
	if err != nil {
		t.Fatal(err)
	}
	stress, err := CalculateStress(snapshot, analytics, scenario)
	if err != nil {
		t.Fatal(err)
	}
	accept.ProposalID = proposal.ProposalID
	accept.ScenarioID = stress.ScenarioID
	accept.DecisionID = decisionIdentity(accept)
	if err := accept.Validate(); err != nil {
		t.Fatal(err)
	}
	store := NewMemoryRiskDecisionStore()
	if err := store.Save(context.Background(), accept); err != nil {
		t.Fatal(err)
	}

	// No call above is allowed to mutate the frozen source portfolio.
	if snapshot.Positions[0].MarketValue.Value != 20000 || snapshot.Cash.Value != 60000 || proposal.PortfolioMutated {
		t.Fatal("risk evaluation mutated portfolio state")
	}
	if stress.ResultID == "" || stress.ScenarioID == "" || proposal.ProposalID == "" {
		t.Fatal("stress/proposal evidence identity missing")
	}

	unknown := snapshot
	unknown.Cash = UnknownNumber("fixture omitted")
	unknown.SnapshotID = ""
	unknownAnalytics := analytics
	unknownResult := EvaluateRecommendation(recommendation(5000), unknown, unknownAnalytics, policy, stateNow, 10*time.Minute)
	if unknownResult.Outcome == DecisionAccept {
		t.Fatal("materially unknown portfolio silently returned ACCEPT")
	}
	if unknownResult.ReasonCodes[0] != ReasonRejectUnknownExposure {
		t.Fatalf("unknown-state proof=%#v", unknownResult)
	}

	if _, err := BuildStressScenario(StressScenario{Version: "phase09-v1", Name: "corrupt"}); err == nil {
		t.Fatal("invalid scenario negative proof failed")
	}
	if _, err := BuildRiskPolicy(RiskPolicy{Version: "bad", Currency: "USD", MaximumLeverage: Limit(2)}); err == nil {
		t.Fatal("unsafe policy negative proof failed")
	}
}
