package candidates

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApprovalEligibilityBlocksIncompleteCandidate(t *testing.T) {
	candidate := approvalReadyCandidate()
	candidate.StopLoss = nil

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())

	if result.ApprovalEligible {
		t.Fatal("structurally incomplete candidate must not be approval eligible")
	}
	if result.ApprovalStatus != ApprovalStatusStructureIncomplete {
		t.Fatalf("approval status = %q, want %q", result.ApprovalStatus, ApprovalStatusStructureIncomplete)
	}
	if !containsString(result.RejectReasons, "structural_fields_missing") {
		t.Fatalf("reject reasons = %v, want structural_fields_missing", result.RejectReasons)
	}
}

func TestApprovalEligibilityBlocksEvidenceFailures(t *testing.T) {
	tests := []struct {
		name       string
		status     EvidenceStatus
		wantStatus string
		wantReason string
	}{
		{name: "missing", status: EvidenceStatusMissing, wantStatus: ApprovalStatusEvidenceNotReady, wantReason: "evidence_missing"},
		{name: "weak", status: EvidenceStatusWeak, wantStatus: ApprovalStatusEvidenceNotReady, wantReason: "evidence_weak"},
		{name: "mixed", status: EvidenceStatusMixed, wantStatus: ApprovalStatusEvidenceNotReady, wantReason: "evidence_mixed"},
		{name: "stale", status: EvidenceStatusStale, wantStatus: ApprovalStatusEvidenceNotReady, wantReason: "evidence_stale"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			candidate := approvalReadyCandidate()
			evidence := sufficientEvidenceScore(candidate.ID)
			evidence.EvidenceStatus = tc.status
			evidence.EvidenceReady = false
			evidence.EvidenceGateReady = false
			gate := readyRiskGate(candidate.ID)
			if tc.status != EvidenceStatusMissing {
				gate.GateStatus = GateStatusEvidenceWeak
				gate.GateReady = false
			}
			if tc.status == EvidenceStatusStale {
				gate.GateStatus = GateStatusEvidenceStale
				gate.HardReject = true
			}

			result := EvaluateApprovalEligibility(candidate, evidence, gate, readyRiskReview(candidate), fixedApprovalReviewTime())

			if result.ApprovalEligible {
				t.Fatal("candidate with insufficient evidence must not be approval eligible")
			}
			if result.ApprovalStatus != tc.wantStatus {
				t.Fatalf("approval status = %q, want %q", result.ApprovalStatus, tc.wantStatus)
			}
			if !containsString(append(result.RejectReasons, result.WarningReasons...), tc.wantReason) {
				t.Fatalf("reasons = rejects %v warnings %v, want %s", result.RejectReasons, result.WarningReasons, tc.wantReason)
			}
		})
	}
}

func TestApprovalEligibilityBlocksGateNotReady(t *testing.T) {
	candidate := approvalReadyCandidate()
	gate := readyRiskGate(candidate.ID)
	gate.GateReady = false
	gate.GateStatus = GateStatusRiskPending

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), gate, readyRiskReview(candidate), fixedApprovalReviewTime())

	if result.ApprovalEligible {
		t.Fatal("gate-not-ready candidate must not be approval eligible")
	}
	if result.ApprovalStatus != ApprovalStatusGateNotReady {
		t.Fatalf("approval status = %q, want %q", result.ApprovalStatus, ApprovalStatusGateNotReady)
	}
	if !containsString(result.RejectReasons, "gate_not_ready") {
		t.Fatalf("reject reasons = %v, want gate_not_ready", result.RejectReasons)
	}
}

func TestApprovalEligibilityBlocksRiskNotReady(t *testing.T) {
	candidate := approvalReadyCandidate()
	risk := readyRiskReview(candidate)
	risk.RiskReady = false
	risk.RiskStatus = RiskStatusRewardRiskTooLow
	risk.WarningReasons = []string{"reward_risk_below_minimum"}

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), risk, fixedApprovalReviewTime())

	if result.ApprovalEligible {
		t.Fatal("risk-not-ready candidate must not be approval eligible")
	}
	if result.ApprovalStatus != ApprovalStatusRiskNotReady {
		t.Fatalf("approval status = %q, want %q", result.ApprovalStatus, ApprovalStatusRiskNotReady)
	}
	if !containsString(result.WarningReasons, "reward_risk_below_minimum") {
		t.Fatalf("warning reasons = %v, want reward_risk_below_minimum", result.WarningReasons)
	}
}

func TestApprovalEligibilityReadyCandidateBecomesApprovalReviewReady(t *testing.T) {
	candidate := approvalReadyCandidate()

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())

	if !result.ApprovalEligible {
		t.Fatalf("valid candidate should be approval eligible: rejects=%v warnings=%v", result.RejectReasons, result.WarningReasons)
	}
	if result.ApprovalStatus != ApprovalStatusApprovalReviewReady {
		t.Fatalf("approval status = %q, want %q", result.ApprovalStatus, ApprovalStatusApprovalReviewReady)
	}
	if result.NextRequiredPhase != NextPhaseApprovalReview {
		t.Fatalf("next phase = %q, want %q", result.NextRequiredPhase, NextPhaseApprovalReview)
	}
	if result.BrokerExecutionAllowed || result.ExecutionInstructionCreated || result.LiveTradingAllowed {
		t.Fatal("approval eligibility must not allow broker execution, create execution instructions, or allow live trading")
	}
}

func TestApprovalEligibilityRejectsLeverageAboveOne(t *testing.T) {
	candidate := approvalReadyCandidate()
	metadata := json.RawMessage(`{"requestedLeverage":1.1}`)
	candidate.Metadata = &metadata

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())

	if result.ApprovalEligible {
		t.Fatal("leveraged candidate must not be approval eligible")
	}
	if !containsString(result.RejectReasons, "leverage_requested_above_1") {
		t.Fatalf("reject reasons = %v, want leverage_requested_above_1", result.RejectReasons)
	}
}

func TestApprovalEligibilityRejectsPrematureApprovalAndHumanApprovalCannotBypassRisk(t *testing.T) {
	candidate := approvalReadyCandidate()
	candidate.Status = StatusApproved
	risk := readyRiskReview(candidate)
	risk.RiskReady = false
	risk.RiskStatus = RiskStatusGateNotReady

	result := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), risk, fixedApprovalReviewTime())

	if result.ApprovalEligible {
		t.Fatal("premature approval must not bypass risk review")
	}
	for _, reason := range []string{"approval_granted_too_early", "risk_not_ready"} {
		if !containsString(result.RejectReasons, reason) {
			t.Fatalf("reject reasons = %v, want %s", result.RejectReasons, reason)
		}
	}
	if result.BrokerExecutionAllowed || result.ExecutionInstructionCreated || result.LiveTradingAllowed {
		t.Fatal("blocked approval eligibility must keep all execution and live-trading flags false")
	}
}

func approvalReadyCandidate() Candidate {
	candidate := completeStructuredCandidate()
	target := 107.0
	candidate.TakeProfit = &target
	candidate.GateStatus = GateStatusReadyForRiskReview
	candidate.RiskStatus = string(RiskStatusReadyForApprovalReview)
	candidate.ApprovalStatus = ApprovalStatusNotReady
	return candidate
}

func readyRiskReview(candidate Candidate) RiskReviewResult {
	risk := ReviewCandidateRisk(candidate, readyRiskGate(candidate.ID), RiskReviewConfig{
		AccountEquity:          10000,
		MaxRiskPercentPerTrade: 0.01,
		MinRewardRiskRatio:     2.0,
		MaxLeverage:            1.0,
		RequestedLeverage:      1.0,
		Now:                    fixedApprovalReviewTime(),
	})
	return risk
}

func fixedApprovalReviewTime() time.Time {
	return time.Date(2026, 7, 13, 15, 0, 0, 0, time.UTC)
}
