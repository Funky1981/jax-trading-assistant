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
	if analytics.SnapshotID != canonical.SnapshotID || analytics.Validate() != nil {
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
	reasons := proposalViolations(current, recommendation.SignedMarketValue, analytics, canonicalPolicy, canonical.Cash.Value)
	if len(reasons) == 0 {
		result.Outcome, result.ReasonCodes, result.Explanations = DecisionAccept, []ReasonCode{ReasonAcceptWithinPolicy}, []string{"recommendation satisfies deterministic Phase-09 policy"}
		return finishDecisionWithValue(result, recommendation.SignedMarketValue)
	}
	maxAllowable := boundedFeasibleMagnitude(current, recommendation.SignedMarketValue, analytics, canonicalPolicy, canonical.Cash.Value)
	if maxAllowable > 0 {
		result.Outcome, result.ReasonCodes, result.Explanations = DecisionAmend, stableReasons(reasons), []string{"requested value was reduced to the deterministic remaining policy capacity"}
		return finishDecisionWithValue(result, math.Copysign(maxAllowable, recommendation.SignedMarketValue))
	}
	return finishDecision(result, rejectCodeForReasons(reasons), "requested recommendation has no remaining policy capacity")
}

func proposalViolations(current, requested float64, analytics ExposureAnalytics, policy RiskPolicy, cash float64) []ReasonCode {
	finalValue := current + requested
	newGross := analytics.GrossExposure - math.Abs(current) + math.Abs(finalValue)
	newNet := analytics.NetExposure - current + finalValue
	reasons := make([]ReasonCode, 0, 2)
	if violatesLimit(policy.MaximumPositionValue, math.Abs(finalValue)) || violatesLimit(policy.MaximumConcentration, math.Abs(finalValue)/analytics.Equity) {
		reasons = append(reasons, ReasonAmendPositionCap)
	}
	if violatesLimit(policy.MaximumGrossExposure, newGross) || violatesLimit(policy.MaximumNetExposure, math.Abs(newNet)) {
		reasons = append(reasons, ReasonAmendRiskBudget)
	}
	if policy.MinimumCash != nil && requested > 0 && cash-requested < *policy.MinimumCash {
		reasons = append(reasons, ReasonAmendRiskBudget)
	}
	return stableReasons(reasons)
}

func boundedFeasibleMagnitude(current, requested float64, analytics ExposureAnalytics, policy RiskPolicy, cash float64) float64 {
	target := math.Abs(requested)
	if target == 0 || len(proposalViolations(current, 0, analytics, policy, cash)) > 0 {
		return 0
	}
	low, high := 0.0, target
	for i := 0; i < 64; i++ {
		mid := (low + high) / 2
		if len(proposalViolations(current, math.Copysign(mid, requested), analytics, policy, cash)) == 0 {
			low = mid
		} else {
			high = mid
		}
	}
	if low <= 1e-9 {
		return 0
	}
	return math.Floor((low+1e-9)*1e6) / 1e6
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
	result.DecisionID = decisionIdentity(result)
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
