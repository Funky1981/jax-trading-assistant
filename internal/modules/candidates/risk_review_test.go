package candidates

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReviewCandidateRiskRejectsCandidateThatIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	gate := readyRiskGate(candidate.ID)
	gate.GateReady = false
	gate.GateStatus = GateStatusEvidenceWeak

	result := ReviewCandidateRisk(candidate, gate, RiskReviewConfig{AccountEquity: 10000})

	if result.RiskStatus != RiskStatusGateNotReady {
		t.Fatalf("risk status = %q, want %q", result.RiskStatus, RiskStatusGateNotReady)
	}
	if result.RiskReady {
		t.Fatal("risk review must not be ready when trust gate is not ready")
	}
	if !containsString(result.RejectReasons, "gate_not_ready") {
		t.Fatalf("reject reasons = %v, want gate_not_ready", result.RejectReasons)
	}
}

func TestReviewCandidateRiskRejectsInvalidTradePlanInputs(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Candidate)
		wantStatus RiskStatus
		wantReason string
	}{
		{
			name: "missing entry",
			mutate: func(c *Candidate) {
				c.EntryPrice = nil
			},
			wantStatus: RiskStatusInvalidTradePlan,
			wantReason: "missing_entry_price",
		},
		{
			name: "missing stop loss",
			mutate: func(c *Candidate) {
				c.StopLoss = nil
			},
			wantStatus: RiskStatusInvalidStop,
			wantReason: "missing_stop_loss_price",
		},
		{
			name: "missing target",
			mutate: func(c *Candidate) {
				c.TakeProfit = nil
			},
			wantStatus: RiskStatusInvalidTradePlan,
			wantReason: "missing_target_price",
		},
		{
			name: "invalid long stop",
			mutate: func(c *Candidate) {
				entry := 100.0
				stop := 100.0
				c.EntryPrice = &entry
				c.StopLoss = &stop
				c.Direction = "long"
			},
			wantStatus: RiskStatusInvalidStop,
			wantReason: "invalid_long_stop",
		},
		{
			name: "invalid short stop",
			mutate: func(c *Candidate) {
				entry := 100.0
				stop := 99.0
				c.EntryPrice = &entry
				c.StopLoss = &stop
				c.Direction = "short"
				c.SignalType = "SELL"
			},
			wantStatus: RiskStatusInvalidStop,
			wantReason: "invalid_short_stop",
		},
		{
			name: "negative slippage",
			mutate: func(c *Candidate) {
				slippage := -0.01
				c.SlippageAllowance = &slippage
			},
			wantStatus: RiskStatusInvalidSlippage,
			wantReason: "negative_slippage_allowance",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			candidate := completeStructuredCandidate()
			tc.mutate(&candidate)

			result := ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{AccountEquity: 10000})

			if result.RiskStatus != tc.wantStatus {
				t.Fatalf("risk status = %q, want %q", result.RiskStatus, tc.wantStatus)
			}
			if result.RiskReady {
				t.Fatal("invalid trade plan must not be risk ready")
			}
			if !containsString(result.RejectReasons, tc.wantReason) {
				t.Fatalf("reject reasons = %v, want %s", result.RejectReasons, tc.wantReason)
			}
		})
	}
}

func TestReviewCandidateRiskSizesPositionWithSlippageAdjustedRisk(t *testing.T) {
	candidate := completeStructuredCandidate()
	entry := 100.0
	stop := 98.0
	target := 106.0
	slippage := 0.50
	candidate.EntryPrice = &entry
	candidate.StopLoss = &stop
	candidate.TakeProfit = &target
	candidate.SlippageAllowance = &slippage

	result := ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{
		AccountEquity:          10000,
		MaxRiskPercentPerTrade: 0.01,
		MinRewardRiskRatio:     2.0,
		MaxLeverage:            1.0,
		RequestedLeverage:      1.0,
		Now:                    fixedRiskReviewTime(),
	})

	if result.RiskStatus != RiskStatusReadyForApprovalReview {
		t.Fatalf("risk status = %q, want %q; rejects=%v warnings=%v", result.RiskStatus, RiskStatusReadyForApprovalReview, result.RejectReasons, result.WarningReasons)
	}
	if !result.RiskReady {
		t.Fatal("valid slippage-adjusted risk review should be ready for approval review")
	}
	assertFloatEqual(t, result.StopDistance, 2.0)
	assertFloatEqual(t, result.SlippageAdjustedStopDistance, 2.5)
	assertFloatEqual(t, result.MaxAllowedLoss, 100.0)
	assertFloatEqual(t, result.PositionSize, 40.0)
	assertFloatEqual(t, result.MaxNormalLoss, 80.0)
	assertFloatEqual(t, result.MaxSlippageAdjustedLoss, 100.0)
	assertFloatEqual(t, result.RewardAmount, 240.0)
	assertFloatEqual(t, result.RewardRiskRatio, 2.4)

	if result.MaxSlippageAdjustedLoss > result.MaxAllowedLoss {
		t.Fatalf("slippage-adjusted loss %.4f exceeds allowed loss %.4f", result.MaxSlippageAdjustedLoss, result.MaxAllowedLoss)
	}
	if result.ApprovalGranted || result.ExecutionInstructionCreated || result.BrokerExecutionAllowed {
		t.Fatal("risk review must not approve, create execution instructions, or allow broker execution")
	}
}

func TestReviewCandidateRiskHigherSlippageProducesSmallerPositionSize(t *testing.T) {
	lowSlippageCandidate := completeStructuredCandidate()
	highSlippageCandidate := completeStructuredCandidate()
	entry := 100.0
	stop := 98.0
	target := 106.0
	lowSlippage := 0.10
	highSlippage := 1.00
	lowSlippageCandidate.EntryPrice = &entry
	lowSlippageCandidate.StopLoss = &stop
	lowSlippageCandidate.TakeProfit = &target
	lowSlippageCandidate.SlippageAllowance = &lowSlippage
	highSlippageCandidate.EntryPrice = &entry
	highSlippageCandidate.StopLoss = &stop
	highSlippageCandidate.TakeProfit = &target
	highSlippageCandidate.SlippageAllowance = &highSlippage

	cfg := RiskReviewConfig{AccountEquity: 10000, MaxRiskPercentPerTrade: 0.01, MinRewardRiskRatio: 1.5}
	low := ReviewCandidateRisk(lowSlippageCandidate, readyRiskGate(lowSlippageCandidate.ID), cfg)
	high := ReviewCandidateRisk(highSlippageCandidate, readyRiskGate(highSlippageCandidate.ID), cfg)

	if !(high.PositionSize < low.PositionSize) {
		t.Fatalf("higher slippage position size = %.4f, low slippage position size = %.4f; want smaller", high.PositionSize, low.PositionSize)
	}
}

func TestReviewCandidateRiskRejectsLeverageAboveOne(t *testing.T) {
	candidate := completeStructuredCandidate()

	result := ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{
		AccountEquity:          10000,
		MaxRiskPercentPerTrade: 0.01,
		MaxLeverage:            1.25,
		RequestedLeverage:      1.0,
	})

	if result.RiskStatus != RiskStatusLeverageBlocked {
		t.Fatalf("risk status = %q, want %q", result.RiskStatus, RiskStatusLeverageBlocked)
	}
	if !containsString(result.RejectReasons, "max_leverage_above_1") {
		t.Fatalf("reject reasons = %v, want max_leverage_above_1", result.RejectReasons)
	}

	result = ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{
		AccountEquity:          10000,
		MaxRiskPercentPerTrade: 0.01,
		MaxLeverage:            1.0,
		RequestedLeverage:      1.10,
	})

	if result.RiskStatus != RiskStatusLeverageBlocked {
		t.Fatalf("risk status = %q, want %q", result.RiskStatus, RiskStatusLeverageBlocked)
	}
	if !containsString(result.RejectReasons, "requested_leverage_above_1") {
		t.Fatalf("reject reasons = %v, want requested_leverage_above_1", result.RejectReasons)
	}
}

func TestReviewCandidateRiskBlocksLowRewardRiskWithoutHardReject(t *testing.T) {
	candidate := completeStructuredCandidate()
	entry := 100.0
	stop := 98.0
	target := 102.5
	slippage := 0.50
	candidate.EntryPrice = &entry
	candidate.StopLoss = &stop
	candidate.TakeProfit = &target
	candidate.SlippageAllowance = &slippage

	result := ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{
		AccountEquity:          10000,
		MaxRiskPercentPerTrade: 0.01,
		MinRewardRiskRatio:     2.0,
	})

	if result.RiskStatus != RiskStatusRewardRiskTooLow {
		t.Fatalf("risk status = %q, want %q", result.RiskStatus, RiskStatusRewardRiskTooLow)
	}
	if result.RiskReady {
		t.Fatal("low reward/risk must block approval readiness")
	}
	if len(result.RejectReasons) != 0 {
		t.Fatalf("low reward/risk should warn, not hard reject; rejects=%v", result.RejectReasons)
	}
	if !containsString(result.WarningReasons, "reward_risk_below_minimum") {
		t.Fatalf("warning reasons = %v, want reward_risk_below_minimum", result.WarningReasons)
	}
}

func readyRiskGate(candidateID uuid.UUID) GateResult {
	return GateResult{
		CandidateID:                 candidateID,
		EvaluatedAt:                 fixedRiskReviewTime(),
		GateStatus:                  GateStatusReadyForRiskReview,
		GateReady:                   true,
		NextRequiredPhase:           NextPhaseRiskReview,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		ApprovalGranted:             false,
	}
}

func fixedRiskReviewTime() time.Time {
	return time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
}

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("got %.6f, want %.6f", got, want)
	}
}
