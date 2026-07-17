package candidates

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type RiskStatus string

const (
	RiskStatusBlocked                RiskStatus = "blocked"
	RiskStatusGateNotReady           RiskStatus = "gate_not_ready"
	RiskStatusInvalidTradePlan       RiskStatus = "invalid_trade_plan"
	RiskStatusInvalidStop            RiskStatus = "invalid_stop"
	RiskStatusInvalidSlippage        RiskStatus = "invalid_slippage"
	RiskStatusRiskTooHigh            RiskStatus = "risk_too_high"
	RiskStatusRewardRiskTooLow       RiskStatus = "reward_risk_too_low"
	RiskStatusLeverageBlocked        RiskStatus = "leverage_blocked"
	RiskStatusReadyForApprovalReview RiskStatus = "ready_for_approval_review"
	defaultAccountEquity                        = 10000.0
	defaultMaxRiskPercentPerTrade               = 0.01
	defaultMinRewardRiskRatio                   = 2.0
	defaultMaxLeverage                          = 1.0
)

type RiskReviewConfig struct {
	AccountEquity          float64
	MaxRiskPercentPerTrade float64
	MinRewardRiskRatio     float64
	MaxLeverage            float64
	RequestedLeverage      float64
	Now                    time.Time
}

type RiskReviewResult struct {
	CandidateID                  string     `json:"candidateId"`
	EvaluatedAt                  time.Time  `json:"evaluatedAt"`
	RiskStatus                   RiskStatus `json:"riskStatus"`
	RiskReady                    bool       `json:"riskReady"`
	PositionSize                 float64    `json:"positionSize"`
	EntryPrice                   float64    `json:"entryPrice"`
	StopLossPrice                float64    `json:"stopLossPrice"`
	TargetPrice                  float64    `json:"targetPrice"`
	StopDistance                 float64    `json:"stopDistance"`
	SlippageAllowance            float64    `json:"slippageAllowance"`
	SlippageAdjustedStopDistance float64    `json:"slippageAdjustedStopDistance"`
	MaxNormalLoss                float64    `json:"maxNormalLoss"`
	MaxSlippageAdjustedLoss      float64    `json:"maxSlippageAdjustedLoss"`
	RewardAmount                 float64    `json:"rewardAmount"`
	RewardRiskRatio              float64    `json:"rewardRiskRatio"`
	AccountEquity                float64    `json:"accountEquity"`
	MaxRiskPercent               float64    `json:"maxRiskPercent"`
	MaxAllowedLoss               float64    `json:"maxAllowedLoss"`
	MinRewardRiskRatio           float64    `json:"minRewardRiskRatio"`
	MaxLeverage                  float64    `json:"maxLeverage"`
	RequestedLeverage            float64    `json:"requestedLeverage"`
	RejectReasons                []string   `json:"rejectReasons"`
	WarningReasons               []string   `json:"warningReasons"`
	NextRequiredPhase            string     `json:"nextRequiredPhase"`
	BrokerExecutionAllowed       bool       `json:"brokerExecutionAllowed"`
	ExecutionInstructionCreated  bool       `json:"executionInstructionCreated"`
	ApprovalGranted              bool       `json:"approvalGranted"`
}

func ReviewCandidateRisk(candidate Candidate, gate GateResult, cfg RiskReviewConfig) RiskReviewResult {
	cfg = normalizedRiskReviewConfig(cfg)
	evaluatedAt := cfg.Now
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}

	result := RiskReviewResult{
		CandidateID:                 candidate.ID.String(),
		EvaluatedAt:                 evaluatedAt,
		RiskStatus:                  RiskStatusBlocked,
		AccountEquity:               roundFloat(cfg.AccountEquity),
		MaxRiskPercent:              roundFloat(cfg.MaxRiskPercentPerTrade),
		MinRewardRiskRatio:          roundFloat(cfg.MinRewardRiskRatio),
		MaxLeverage:                 roundFloat(cfg.MaxLeverage),
		RequestedLeverage:           roundFloat(cfg.RequestedLeverage),
		NextRequiredPhase:           NextPhaseRiskReview,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		ApprovalGranted:             false,
	}

	if gate.BrokerExecutionAllowed {
		return hardRiskReject(result, RiskStatusBlocked, NextPhaseStop, "broker_execution_allowed_too_early")
	}
	if gate.ExecutionInstructionCreated || candidate.ExecutionInstructionID != nil || candidate.TradeID != nil {
		return hardRiskReject(result, RiskStatusBlocked, NextPhaseStop, "execution_instruction_created_too_early")
	}
	if gate.ApprovalGranted || candidate.Status == StatusApproved || strings.EqualFold(candidate.ApprovalStatus, "approved") ||
		(candidate.LatestApproval != nil && strings.EqualFold(candidate.LatestApproval.Decision, "approved")) {
		return hardRiskReject(result, RiskStatusBlocked, NextPhaseStop, "approval_granted_too_early")
	}

	if cfg.MaxLeverage > 1.0 {
		return hardRiskReject(result, RiskStatusLeverageBlocked, NextPhaseStop, "max_leverage_above_1")
	}
	if cfg.RequestedLeverage > 1.0 || leverageRequestedAboveOne(candidate) {
		return hardRiskReject(result, RiskStatusLeverageBlocked, NextPhaseStop, "requested_leverage_above_1")
	}

	entry, ok := positivePrice(candidate.EntryPrice)
	if !ok {
		return hardRiskReject(result, RiskStatusInvalidTradePlan, NextPhaseCandidateRepair, "missing_entry_price")
	}
	stop, ok := positivePrice(candidate.StopLoss)
	if !ok {
		return hardRiskReject(result, RiskStatusInvalidStop, NextPhaseCandidateRepair, "missing_stop_loss_price")
	}
	target, ok := positivePrice(candidate.TakeProfit)
	if !ok {
		return hardRiskReject(result, RiskStatusInvalidTradePlan, NextPhaseCandidateRepair, "missing_target_price")
	}

	structural := ValidateStructuralCompleteness(candidate)
	if !structural.StructurallyComplete {
		result.RiskStatus = RiskStatusInvalidTradePlan
		result.NextRequiredPhase = NextPhaseCandidateRepair
		result.RejectReasons = appendUnique(result.RejectReasons, structural.RejectReasons...)
		for _, missing := range structural.MissingFields {
			result.RejectReasons = appendUnique(result.RejectReasons, "missing_"+missing)
		}
		return result
	}

	if !gate.GateReady || gate.GateStatus != GateStatusReadyForRiskReview {
		return hardRiskReject(result, RiskStatusGateNotReady, gate.NextRequiredPhase, "gate_not_ready")
	}

	slippage, ok := nonNegativeAmount(candidate.SlippageAllowance)
	if !ok {
		return hardRiskReject(result, RiskStatusInvalidSlippage, NextPhaseCandidateRepair, "negative_slippage_allowance")
	}

	result.EntryPrice = entry.InexactFloat64()
	result.StopLossPrice = stop.InexactFloat64()
	result.TargetPrice = target.InexactFloat64()
	result.SlippageAllowance = slippage.InexactFloat64()

	stopDistance, reason := stopDistanceForDirection(candidate, entry, stop)
	if !reasonIsEmpty(reason) {
		return hardRiskReject(result, RiskStatusInvalidStop, NextPhaseCandidateRepair, reason)
	}
	adjustedStopDistance := stopDistance.Add(slippage)
	if !adjustedStopDistance.GreaterThan(decimal.Zero) {
		return hardRiskReject(result, RiskStatusInvalidSlippage, NextPhaseCandidateRepair, "slippage_adjusted_stop_distance_invalid")
	}

	maxAllowedLoss := decimal.NewFromFloat(cfg.AccountEquity).Mul(decimal.NewFromFloat(cfg.MaxRiskPercentPerTrade))
	positionSize := maxAllowedLoss.Div(adjustedStopDistance)
	maxNormalLoss := positionSize.Mul(stopDistance)
	maxSlippageAdjustedLoss := positionSize.Mul(adjustedStopDistance)

	result.StopDistance = decimalToFloat(stopDistance)
	result.SlippageAdjustedStopDistance = decimalToFloat(adjustedStopDistance)
	result.MaxAllowedLoss = decimalToFloat(maxAllowedLoss)
	result.PositionSize = decimalToFloat(positionSize)
	result.MaxNormalLoss = decimalToFloat(maxNormalLoss)
	result.MaxSlippageAdjustedLoss = decimalToFloat(maxSlippageAdjustedLoss)

	if maxSlippageAdjustedLoss.Round(6).GreaterThan(maxAllowedLoss.Round(6)) {
		return hardRiskReject(result, RiskStatusRiskTooHigh, NextPhaseRiskReview, "slippage_adjusted_loss_above_allowed")
	}

	rewardAmount, reason := rewardForDirection(candidate, entry, target, positionSize)
	if !reasonIsEmpty(reason) {
		return hardRiskReject(result, RiskStatusInvalidTradePlan, NextPhaseCandidateRepair, reason)
	}
	result.RewardAmount = decimalToFloat(rewardAmount)
	result.RewardRiskRatio = decimalToFloat(rewardAmount.Div(maxSlippageAdjustedLoss))

	if decimal.NewFromFloat(result.RewardRiskRatio).LessThan(decimal.NewFromFloat(cfg.MinRewardRiskRatio)) {
		result.RiskStatus = RiskStatusRewardRiskTooLow
		result.RiskReady = false
		result.NextRequiredPhase = NextPhaseRiskReview
		result.WarningReasons = appendUnique(result.WarningReasons, "reward_risk_below_minimum")
		return result
	}

	result.RiskStatus = RiskStatusReadyForApprovalReview
	result.RiskReady = true
	result.NextRequiredPhase = "approval_review"
	return result
}

func normalizedRiskReviewConfig(cfg RiskReviewConfig) RiskReviewConfig {
	if cfg.AccountEquity <= 0 {
		cfg.AccountEquity = defaultAccountEquity
	}
	if cfg.MaxRiskPercentPerTrade <= 0 {
		cfg.MaxRiskPercentPerTrade = defaultMaxRiskPercentPerTrade
	}
	if cfg.MinRewardRiskRatio <= 0 {
		cfg.MinRewardRiskRatio = defaultMinRewardRiskRatio
	}
	if cfg.MaxLeverage <= 0 {
		cfg.MaxLeverage = defaultMaxLeverage
	}
	if cfg.RequestedLeverage <= 0 {
		cfg.RequestedLeverage = 1.0
	}
	return cfg
}

func positivePrice(value *float64) (decimal.Decimal, bool) {
	if value == nil || *value <= 0 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(*value), true
}

func nonNegativeAmount(value *float64) (decimal.Decimal, bool) {
	if value == nil {
		return decimal.Zero, true
	}
	if *value < 0 {
		return decimal.Zero, false
	}
	return decimal.NewFromFloat(*value), true
}

func stopDistanceForDirection(candidate Candidate, entry, stop decimal.Decimal) (decimal.Decimal, string) {
	switch normalizedDirection(candidate) {
	case "short":
		if stop.LessThanOrEqual(entry) {
			return decimal.Zero, "invalid_short_stop"
		}
		return stop.Sub(entry), ""
	default:
		if stop.GreaterThanOrEqual(entry) {
			return decimal.Zero, "invalid_long_stop"
		}
		return entry.Sub(stop), ""
	}
}

func rewardForDirection(candidate Candidate, entry, target, positionSize decimal.Decimal) (decimal.Decimal, string) {
	switch normalizedDirection(candidate) {
	case "short":
		if target.GreaterThanOrEqual(entry) {
			return decimal.Zero, "invalid_short_target"
		}
		return entry.Sub(target).Mul(positionSize), ""
	default:
		if target.LessThanOrEqual(entry) {
			return decimal.Zero, "invalid_long_target"
		}
		return target.Sub(entry).Mul(positionSize), ""
	}
}

func normalizedDirection(candidate Candidate) string {
	direction := strings.ToLower(strings.TrimSpace(candidate.Direction))
	if direction == "" {
		direction = directionFromSignalType(candidate.SignalType)
	}
	if direction == "sell" {
		return "short"
	}
	return direction
}

func hardRiskReject(result RiskReviewResult, status RiskStatus, nextPhase string, reason string) RiskReviewResult {
	result.RiskStatus = status
	result.RiskReady = false
	if strings.TrimSpace(nextPhase) != "" {
		result.NextRequiredPhase = nextPhase
	}
	result.RejectReasons = appendUnique(result.RejectReasons, reason)
	return result
}

func reasonIsEmpty(reason string) bool {
	return strings.TrimSpace(reason) == ""
}

func decimalToFloat(value decimal.Decimal) float64 {
	return roundFloat(value.InexactFloat64())
}

func roundFloat(value float64) float64 {
	rounded, _ := decimal.NewFromFloat(value).Round(6).Float64()
	return rounded
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
