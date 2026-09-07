package portfoliorisk

import (
	"testing"
	"time"
)

func TestPositionProposalIsBoundedAndNonExecutable(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	input := PositionProposalInput{RecommendationID: "rec-proposal-1", InstrumentID: "NYSE:XYZ", Currency: "USD", SignedMarketValue: 50000, EntryPrice: 100, StopPrice: 90, RiskAllocation: Limit(.01)}
	proposal, err := CalculatePositionProposal(input, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.ProposedValue != 10000 || proposal.ProposedQuantity != 100 || proposal.CapitalRequired != 10000 {
		t.Fatalf("unexpected proposal=%#v", proposal)
	}
	if proposal.ExecutionAuthority != "NONE" || proposal.PortfolioMutated {
		t.Fatalf("proposal crossed execution boundary: %#v", proposal)
	}
	if proposal.ProposalID == "" {
		t.Fatal("proposal identity missing")
	}
}

func TestPositionProposalRespectsCashAndUnknownState(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	input := PositionProposalInput{RecommendationID: "rec-proposal-2", InstrumentID: "NYSE:XYZ", Currency: "USD", SignedMarketValue: 50000, EntryPrice: 100, StopPrice: 90, RiskAllocation: Limit(.50)}
	proposal, err := CalculatePositionProposal(input, snapshot, analytics, policy, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.CapReason != "maximum position value" || proposal.ProposedValue != 25000 {
		t.Fatalf("position cap not applied: %#v", proposal)
	}
	snapshot.Cash = UnknownNumber("omitted")
	if _, err := CalculatePositionProposal(input, snapshot, analytics, policy, stateNow, 10*time.Minute); err == nil {
		t.Fatal("unknown cash produced proposal")
	}
	snapshot = fixtureSnapshot()
	snapshot.Cash = KnownNumber(5000, "fixture")
	analytics, _ = CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if _, err := CalculatePositionProposal(input, snapshot, analytics, policy, stateNow, 10*time.Minute); err == nil {
		t.Fatal("minimum cash violation produced proposal")
	}
}

func TestPositionProposalRejectsMissingSizingInputs(t *testing.T) {
	snapshot, analytics, policy := fixtureAnalytics(t)
	input := PositionProposalInput{RecommendationID: "r", InstrumentID: "NYSE:XYZ", Currency: "USD", SignedMarketValue: 1000, EntryPrice: 100, StopPrice: 100}
	if _, err := CalculatePositionProposal(input, snapshot, analytics, policy, stateNow, 10*time.Minute); err == nil {
		t.Fatal("equal entry/stop accepted")
	}
}
