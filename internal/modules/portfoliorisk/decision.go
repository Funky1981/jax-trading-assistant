package portfoliorisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"
)

const RiskDecisionAlgorithmV1 = "jax.portfolio.risk_decision/v1"

type DecisionOutcome string

const (
	DecisionAccept DecisionOutcome = "ACCEPT"
	DecisionAmend  DecisionOutcome = "AMEND"
	DecisionReject DecisionOutcome = "REJECT"
)

type ReasonCode string

const (
	ReasonAcceptWithinPolicy          ReasonCode = "ACCEPT_WITHIN_POLICY"
	ReasonAmendPositionCap            ReasonCode = "AMEND_POSITION_CAP"
	ReasonAmendRiskBudget             ReasonCode = "AMEND_RISK_BUDGET"
	ReasonRejectMissingPortfolio      ReasonCode = "REJECT_MISSING_PORTFOLIO"
	ReasonRejectStalePortfolio        ReasonCode = "REJECT_STALE_PORTFOLIO"
	ReasonRejectUnknownExposure       ReasonCode = "REJECT_UNKNOWN_EXPOSURE"
	ReasonRejectInvalidPolicy         ReasonCode = "REJECT_INVALID_POLICY"
	ReasonRejectUnsupportedCurrency   ReasonCode = "REJECT_UNSUPPORTED_CURRENCY"
	ReasonRejectConcentrationLimit    ReasonCode = "REJECT_CONCENTRATION_LIMIT"
	ReasonRejectGrossLimit            ReasonCode = "REJECT_GROSS_EXPOSURE_LIMIT"
	ReasonRejectNetLimit              ReasonCode = "REJECT_NET_EXPOSURE_LIMIT"
	ReasonRejectInsufficientCapital   ReasonCode = "REJECT_INSUFFICIENT_CAPITAL"
	ReasonRejectLeverageLimit         ReasonCode = "REJECT_LEVERAGE_LIMIT"
	ReasonRejectInvalidRecommendation ReasonCode = "REJECT_INVALID_RECOMMENDATION"
	ReasonRejectExecutionAuthority    ReasonCode = "REJECT_EXECUTION_AUTHORITY"
	ReasonRejectNoCapacity            ReasonCode = "REJECT_NO_REMAINING_CAPACITY"
)

type RecommendationRiskInput struct {
	RecommendationID   string   `json:"recommendation_id"`
	InstrumentID       string   `json:"instrument_id"`
	Currency           string   `json:"currency"`
	SignedMarketValue  float64  `json:"signed_market_value"`
	RiskAllocation     *float64 `json:"risk_allocation,omitempty"`
	RequestedLeverage  *float64 `json:"requested_leverage,omitempty"`
	Confidence         *float64 `json:"confidence,omitempty"`
	ExecutionAuthority string   `json:"execution_authority"`
	QuantResultIDs     []string `json:"quant_result_ids"`
}

type RiskDecision struct {
	DecisionID          string          `json:"decision_id"`
	Algorithm           string          `json:"algorithm"`
	Outcome             DecisionOutcome `json:"outcome"`
	RecommendationID    string          `json:"recommendation_id"`
	PortfolioSnapshotID string          `json:"portfolio_snapshot_id"`
	AnalyticsID         string          `json:"analytics_id"`
	PolicyID            string          `json:"policy_id"`
	EvaluatedAt         time.Time       `json:"evaluated_at"`
	ReasonCodes         []ReasonCode    `json:"reason_codes"`
	Explanations        []string        `json:"explanations"`
	RequestedValue      float64         `json:"requested_value"`
	ResultingValue      float64         `json:"resulting_value"`
	ExecutionAuthority  string          `json:"execution_authority"`
	QuantResultIDs      []string        `json:"quant_result_ids,omitempty"`
	ScenarioID          string          `json:"scenario_id,omitempty"`
	ProposalID          string          `json:"proposal_id,omitempty"`
}

// EvaluateRecommendation is deterministic policy arithmetic. Model confidence
// is carried for provenance only and cannot override a failed rule.
func EvaluateRecommendation(recommendation RecommendationRiskInput, snapshot PortfolioSnapshot, analytics ExposureAnalytics, policy RiskPolicy, evaluatedAt time.Time, maxAge time.Duration) RiskDecision {
	result := RiskDecision{Algorithm: RiskDecisionAlgorithmV1, Outcome: DecisionReject, EvaluatedAt: evaluatedAt.UTC(), RecommendationID: recommendation.RecommendationID, PortfolioSnapshotID: snapshot.SnapshotID, AnalyticsID: analytics.AnalyticsID, PolicyID: policy.PolicyID, RequestedValue: recommendation.SignedMarketValue, ResultingValue: recommendation.SignedMarketValue, ExecutionAuthority: "NONE", QuantResultIDs: append([]string(nil), recommendation.QuantResultIDs...)}
	if recommendation.ExecutionAuthority != "NONE" {
		return finishDecision(result, ReasonRejectExecutionAuthority, "Phase-09 risk evaluation cannot grant execution authority")
	}
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return finishDecision(result, ReasonRejectMissingPortfolio, err.Error())
	}
	result.PortfolioSnapshotID = canonical.SnapshotID
	canonicalPolicy, err := BuildRiskPolicy(policy)
	if err != nil {
		return finishDecision(result, ReasonRejectInvalidPolicy, err.Error())
	}
	result.PolicyID = canonicalPolicy.PolicyID
	if canonical.Currency != canonicalPolicy.Currency || strings.ToUpper(strings.TrimSpace(recommendation.Currency)) != canonical.Currency {
		return finishDecision(result, ReasonRejectUnsupportedCurrency, "recommendation, portfolio and policy currencies must match")
	}
	freshness := canonical.AssessFreshness(evaluatedAt, maxAge)
	if freshness.Status == FreshnessStale {
		return finishDecision(result, ReasonRejectStalePortfolio, freshness.Reason)
	}
	if freshness.Status != FreshnessFresh {
		return finishDecision(result, ReasonRejectUnknownExposure, freshness.Reason)
	}
	if strings.TrimSpace(recommendation.RecommendationID) == "" || strings.TrimSpace(recommendation.InstrumentID) == "" || !finiteNonZero(recommendation.SignedMarketValue) {
		return finishDecision(result, ReasonRejectInvalidRecommendation, "recommendation identity, instrument and signed value are required")
	}
	if recommendation.RiskAllocation == nil || !finiteLimit(recommendation.RiskAllocation) || *recommendation.RiskAllocation < 0 {
		return finishDecision(result, ReasonRejectUnknownExposure, "risk allocation is unknown or invalid")
	}
	if recommendation.RequestedLeverage == nil || !finiteLimit(recommendation.RequestedLeverage) || *recommendation.RequestedLeverage <= 0 || *recommendation.RequestedLeverage > 1 {
		return finishDecision(result, ReasonRejectLeverageLimit, "requested leverage must be explicit and no greater than 1x")
	}
	if *recommendation.RiskAllocation > valueOrZero(canonicalPolicy.MaximumRiskAllocation) && canonicalPolicy.MaximumRiskAllocation != nil {
		return finishDecision(result, ReasonRejectLeverageLimit, "risk allocation exceeds policy")
	}
	if analytics.SnapshotID != canonical.SnapshotID || analytics.Algorithm == "" {
		return finishDecision(result, ReasonRejectUnknownExposure, "analytics identity does not match portfolio snapshot")
	}
	if analytics.Equity <= 0 || !finite(analytics.GrossExposure) || !finite(analytics.NetExposure) {
		return finishDecision(result, ReasonRejectUnknownExposure, "analytics denominator or exposure is unknown")
	}
	current := 0.0
	for _, line := range analytics.Lines {
		if line.InstrumentID == recommendation.InstrumentID {
			current = line.MarketValue
			break
		}
	}
	proposedFinal := current + recommendation.SignedMarketValue
	maxAllowable := math.Abs(recommendation.SignedMarketValue)
	reasons := make([]ReasonCode, 0, 2)
	// Calculate the maximum safe signed magnitude for a same-direction proposal.
	if recommendation.SignedMarketValue > 0 {
		maxAllowable = math.Min(maxAllowable, capRemaining(canonicalPolicy.MaximumPositionValue, math.Abs(current)))
		maxAllowable = math.Min(maxAllowable, capRemaining(canonicalPolicy.MaximumConcentration, math.Abs(current), analytics.Equity))
		maxAllowable = math.Min(maxAllowable, capRemaining(canonicalPolicy.MaximumGrossExposure, analytics.GrossExposure-math.Abs(current)))
		maxAllowable = math.Min(maxAllowable, capRemaining(canonicalPolicy.MaximumNetExposure, analytics.NetExposure-current))
		if canonical.Cash.Known && canonicalPolicy.MinimumCash != nil {
			maxAllowable = math.Min(maxAllowable, math.Max(0, canonical.Cash.Value-*canonicalPolicy.MinimumCash))
		}
	}
	newGross := analytics.GrossExposure - math.Abs(current) + math.Abs(proposedFinal)
	newNet := analytics.NetExposure - current + proposedFinal
	newConcentration := math.Abs(proposedFinal) / analytics.Equity
	if violatesLimit(canonicalPolicy.MaximumPositionValue, math.Abs(proposedFinal)) || violatesLimit(canonicalPolicy.MaximumConcentration, newConcentration) {
		reasons = append(reasons, ReasonAmendPositionCap)
	}
	if violatesLimit(canonicalPolicy.MaximumGrossExposure, newGross) || violatesLimit(canonicalPolicy.MaximumNetExposure, math.Abs(newNet)) {
		reasons = append(reasons, ReasonAmendRiskBudget)
	}
	if canonicalPolicy.MinimumCash != nil && recommendation.SignedMarketValue > 0 && canonical.Cash.Value-recommendation.SignedMarketValue < *canonicalPolicy.MinimumCash {
		reasons = append(reasons, ReasonAmendRiskBudget)
	}
	if len(reasons) == 0 {
		result.Outcome, result.ReasonCodes, result.Explanations = DecisionAccept, []ReasonCode{ReasonAcceptWithinPolicy}, []string{"recommendation satisfies deterministic Phase-09 policy"}
		return finishDecisionWithValue(result, recommendation.SignedMarketValue)
	}
	if maxAllowable > 0 && maxAllowable < math.Abs(recommendation.SignedMarketValue) {
		if recommendation.SignedMarketValue < 0 {
			maxAllowable = -maxAllowable
		}
		result.Outcome, result.ReasonCodes, result.Explanations = DecisionAmend, stableReasons(reasons), []string{"requested value was reduced to the deterministic remaining policy capacity"}
		return finishDecisionWithValue(result, maxAllowable)
	}
	return finishDecision(result, rejectCodeForReasons(reasons), "requested recommendation has no remaining policy capacity")
}

func capRemaining(limit *float64, used float64, extra ...float64) float64 {
	if limit == nil {
		return math.Inf(1)
	}
	if len(extra) > 0 {
		used = used + 0
		return math.Max(0, *limit*extra[0]-used)
	}
	return math.Max(0, *limit-used)
}
func valueOrZero(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
func violatesLimit(limit *float64, observed float64) bool {
	return limit != nil && observed > *limit+1e-9
}
func stableReasons(reasons []ReasonCode) []ReasonCode {
	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return reasons
}
func rejectCodeForReasons(reasons []ReasonCode) ReasonCode {
	for _, reason := range reasons {
		if reason == ReasonAmendRiskBudget {
			return ReasonRejectInsufficientCapital
		}
	}
	if len(reasons) > 0 && reasons[0] == ReasonAmendPositionCap {
		return ReasonRejectConcentrationLimit
	}
	return ReasonRejectNoCapacity
}
func finishDecision(result RiskDecision, code ReasonCode, explanation string) RiskDecision {
	result.Outcome = DecisionReject
	result.ReasonCodes = []ReasonCode{code}
	result.Explanations = []string{explanation}
	return finishDecisionWithValue(result, result.ResultingValue)
}
func finishDecisionWithValue(result RiskDecision, value float64) RiskDecision {
	result.ResultingValue = value
	copyResult := result
	copyResult.DecisionID = ""
	copyResult.EvaluatedAt = time.Time{}
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	result.DecisionID = "rdec_" + hex.EncodeToString(digest[:])
	return result
}

func decisionIdentity(result RiskDecision) string {
	copyResult := result
	copyResult.DecisionID = ""
	copyResult.EvaluatedAt = time.Time{}
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	return "rdec_" + hex.EncodeToString(digest[:])
}
