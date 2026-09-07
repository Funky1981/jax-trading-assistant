package portfoliorisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

const PositionProposalAlgorithmV1 = "jax.portfolio.position_proposal/v1"

// PositionProposal is descriptive arithmetic only. Deliberately absent are
// venue, route, order type, time-in-force, broker and execution fields.
type PositionProposal struct {
	ProposalID         string    `json:"proposal_id"`
	Algorithm          string    `json:"algorithm"`
	RecommendationID   string    `json:"recommendation_id"`
	SnapshotID         string    `json:"snapshot_id"`
	PolicyID           string    `json:"policy_id"`
	InstrumentID       string    `json:"instrument_id"`
	Currency           string    `json:"currency"`
	EntryPrice         float64   `json:"entry_price"`
	StopPrice          float64   `json:"stop_price"`
	SizingMethod       string    `json:"sizing_method"`
	RiskBudget         float64   `json:"risk_budget"`
	RequestedValue     float64   `json:"requested_value"`
	ProposedValue      float64   `json:"proposed_value"`
	ProposedQuantity   float64   `json:"proposed_quantity"`
	MaximumQuantity    float64   `json:"maximum_quantity"`
	CapitalRequired    float64   `json:"capital_required"`
	RemainingCash      float64   `json:"remaining_cash"`
	CapReason          string    `json:"cap_reason,omitempty"`
	EvaluatedAt        time.Time `json:"evaluated_at"`
	ExecutionAuthority string    `json:"execution_authority"`
	PortfolioMutated   bool      `json:"portfolio_mutated"`
}

type PositionProposalInput struct {
	RecommendationID  string
	InstrumentID      string
	Currency          string
	SignedMarketValue float64
	EntryPrice        float64
	StopPrice         float64
	RiskAllocation    *float64
}

func CalculatePositionProposal(input PositionProposalInput, snapshot PortfolioSnapshot, analytics ExposureAnalytics, policy RiskPolicy, evaluatedAt time.Time, maxAge time.Duration) (PositionProposal, error) {
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return PositionProposal{}, err
	}
	canonicalPolicy, err := BuildRiskPolicy(policy)
	if err != nil {
		return PositionProposal{}, err
	}
	if evaluatedAt.IsZero() || input.RecommendationID == "" || input.InstrumentID == "" || input.Currency != canonical.Currency || input.Currency != canonicalPolicy.Currency {
		return PositionProposal{}, fmt.Errorf("proposal requires matching identities and currencies")
	}
	if !finitePositive(input.EntryPrice) || !finitePositive(input.StopPrice) || input.EntryPrice == input.StopPrice {
		return PositionProposal{}, fmt.Errorf("proposal requires distinct positive entry and stop prices")
	}
	if input.RiskAllocation == nil || !finiteLimit(input.RiskAllocation) || *input.RiskAllocation <= 0 || *input.RiskAllocation > 1 {
		return PositionProposal{}, fmt.Errorf("proposal requires explicit risk allocation in (0,1]")
	}
	if analytics.SnapshotID != canonical.SnapshotID || analytics.Equity <= 0 {
		return PositionProposal{}, fmt.Errorf("proposal analytics identity or equity is unknown")
	}
	if freshness := canonical.AssessFreshness(evaluatedAt, maxAge); freshness.Status != FreshnessFresh {
		return PositionProposal{}, fmt.Errorf("proposal cannot use %s portfolio: %s", freshness.Status, freshness.Reason)
	}
	if !canonical.Cash.Known {
		return PositionProposal{}, fmt.Errorf("%w: cash is required for a capital-safe proposal", ErrUnknownMaterialState)
	}
	unitRisk := math.Abs(input.EntryPrice - input.StopPrice)
	riskBudget := canonical.Equity.Value * *input.RiskAllocation
	riskQuantity := riskBudget / unitRisk
	maximumQuantity := riskQuantity
	capReason := "risk budget"
	if canonicalPolicy.MaximumPositionValue != nil {
		if cap := *canonicalPolicy.MaximumPositionValue / input.EntryPrice; cap < maximumQuantity {
			maximumQuantity, capReason = cap, "maximum position value"
		}
	}
	if canonicalPolicy.MaximumConcentration != nil {
		if cap := (*canonicalPolicy.MaximumConcentration * canonical.Equity.Value) / input.EntryPrice; cap < maximumQuantity {
			maximumQuantity, capReason = cap, "maximum concentration"
		}
	}
	if input.SignedMarketValue > 0 && canonicalPolicy.MinimumCash != nil {
		if cap := math.Max(0, (canonical.Cash.Value-*canonicalPolicy.MinimumCash)/input.EntryPrice); cap < maximumQuantity {
			maximumQuantity, capReason = cap, "minimum cash"
		}
	}
	requestedQuantity := math.Abs(input.SignedMarketValue) / input.EntryPrice
	if requestedQuantity == 0 {
		requestedQuantity = maximumQuantity
	}
	proposedQuantity := math.Min(requestedQuantity, maximumQuantity)
	if !finitePositive(proposedQuantity) {
		return PositionProposal{}, fmt.Errorf("proposal has no positive known capacity")
	}
	signedQuantity := proposedQuantity
	if input.SignedMarketValue < 0 {
		signedQuantity = -signedQuantity
	}
	proposedValue := signedQuantity * input.EntryPrice
	capitalRequired := 0.0
	if proposedValue > 0 {
		capitalRequired = proposedValue
	}
	result := PositionProposal{Algorithm: PositionProposalAlgorithmV1, RecommendationID: input.RecommendationID, SnapshotID: canonical.SnapshotID, PolicyID: canonicalPolicy.PolicyID, InstrumentID: input.InstrumentID, Currency: input.Currency, EntryPrice: input.EntryPrice, StopPrice: input.StopPrice, SizingMethod: "equity_risk_fraction_capped_by_policy_and_known_cash", RiskBudget: riskBudget, RequestedValue: input.SignedMarketValue, ProposedValue: proposedValue, ProposedQuantity: signedQuantity, MaximumQuantity: maximumQuantity, CapitalRequired: capitalRequired, RemainingCash: canonical.Cash.Value - capitalRequired, CapReason: capReason, EvaluatedAt: evaluatedAt.UTC(), ExecutionAuthority: "NONE", PortfolioMutated: false}
	result.ProposalID = proposalIdentity(result)
	return result, nil
}

func proposalIdentity(result PositionProposal) string {
	copyResult := result
	copyResult.ProposalID = ""
	copyResult.EvaluatedAt = time.Time{}
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	return "pprop_" + hex.EncodeToString(digest[:])
}
