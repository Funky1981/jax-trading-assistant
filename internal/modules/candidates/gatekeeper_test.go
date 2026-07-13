package candidates

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGatekeeperIncompleteCandidateIsBlocked(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.Symbol = ""
	now := time.Date(2026, 7, 12, 9, 30, 0, 0, time.UTC)

	result := EvaluateCandidateGate(candidate, EvidenceScoreSummary{}, now)

	if result.GateStatus != GateStatusIncomplete {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusIncomplete)
	}
	if result.GateReady {
		t.Fatal("incomplete candidate must not be gate-ready")
	}
	if !result.HardReject {
		t.Fatal("incomplete candidate must be a hard reject")
	}
	if !containsString(result.RejectReasons, "structural_fields_missing") {
		t.Fatalf("reject reasons = %#v, want structural_fields_missing", result.RejectReasons)
	}
}

func TestGatekeeperMissingStopLossIsBlocked(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.StopLoss = nil

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if result.GateStatus != GateStatusIncomplete {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusIncomplete)
	}
	if !result.HardReject {
		t.Fatal("candidate without stop loss must be a hard reject")
	}
	if !containsString(result.RejectReasons, "missing_stop_loss_price") {
		t.Fatalf("reject reasons = %#v, want missing_stop_loss_price", result.RejectReasons)
	}
}

func TestGatekeeperMissingCatalystIsBlocked(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.CatalystSummary = ""

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if result.GateStatus != GateStatusIncomplete {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusIncomplete)
	}
	if !result.HardReject {
		t.Fatal("candidate without catalyst must be a hard reject")
	}
	if !containsString(result.RejectReasons, "missing_catalyst_summary") {
		t.Fatalf("reject reasons = %#v, want missing_catalyst_summary", result.RejectReasons)
	}
}

func TestGatekeeperNoEvidenceIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	score := EvidenceScoreSummary{
		CandidateID:       candidate.ID,
		EvidenceStatus:    EvidenceStatusMissing,
		EvidenceReady:     false,
		EvidenceGateReady: false,
	}

	result := EvaluateCandidateGate(candidate, score, time.Now().UTC())

	if result.GateStatus != GateStatusEvidenceMissing {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusEvidenceMissing)
	}
	if result.GateReady {
		t.Fatal("candidate with no evidence must not be gate-ready")
	}
	if result.HardReject {
		t.Fatal("missing evidence should block without hard-rejecting")
	}
}

func TestGatekeeperWeakEvidenceIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	score := sufficientEvidenceScore(candidate.ID)
	score.EvidenceStatus = EvidenceStatusWeak
	score.EvidenceReady = false
	score.EvidenceGateReady = false

	result := EvaluateCandidateGate(candidate, score, time.Now().UTC())

	if result.GateStatus != GateStatusEvidenceWeak {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusEvidenceWeak)
	}
	if result.GateReady {
		t.Fatal("candidate with weak evidence must not be gate-ready")
	}
	if result.HardReject {
		t.Fatal("weak evidence should block without hard-rejecting")
	}
}

func TestGatekeeperMixedEvidenceIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	score := sufficientEvidenceScore(candidate.ID)
	score.EvidenceStatus = EvidenceStatusMixed
	score.EvidenceReady = false
	score.EvidenceGateReady = false
	score.ContradictoryItemCount = 1

	result := EvaluateCandidateGate(candidate, score, time.Now().UTC())

	if result.GateStatus != GateStatusEvidenceMixed {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusEvidenceMixed)
	}
	if result.GateReady {
		t.Fatal("candidate with mixed evidence must not be gate-ready")
	}
	if result.HardReject {
		t.Fatal("mixed evidence should block without hard-rejecting")
	}
}

func TestGatekeeperStaleEvidenceIsBlocked(t *testing.T) {
	candidate := completeStructuredCandidate()
	score := sufficientEvidenceScore(candidate.ID)
	score.EvidenceStatus = EvidenceStatusStale
	score.EvidenceReady = false
	score.EvidenceGateReady = false
	score.StaleItemCount = 1

	result := EvaluateCandidateGate(candidate, score, time.Now().UTC())

	if result.GateStatus != GateStatusEvidenceStale {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusEvidenceStale)
	}
	if !result.HardReject {
		t.Fatal("stale evidence must hard-reject the gate")
	}
}

func TestGatekeeperOnlyContradictoryEvidenceIsBlocked(t *testing.T) {
	candidate := completeStructuredCandidate()
	score := EvidenceScoreSummary{
		CandidateID:            candidate.ID,
		EvidenceItemCount:      1,
		ContradictoryItemCount: 1,
		EvidenceStatus:         EvidenceStatusBlocked,
		EvidenceReady:          false,
		EvidenceGateReady:      false,
	}

	result := EvaluateCandidateGate(candidate, score, time.Now().UTC())

	if result.GateStatus != GateStatusBlocked {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusBlocked)
	}
	if !result.HardReject {
		t.Fatal("only contradictory evidence must hard-reject the gate")
	}
	if !containsString(result.RejectReasons, "evidence_blocked") {
		t.Fatalf("reject reasons = %#v, want evidence_blocked", result.RejectReasons)
	}
}

func TestGatekeeperSufficientEvidenceCanBecomeReadyForRiskReview(t *testing.T) {
	candidate := completeStructuredCandidate()

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if result.GateStatus != GateStatusReadyForRiskReview {
		t.Fatalf("gate status = %q, want %q", result.GateStatus, GateStatusReadyForRiskReview)
	}
	if !result.GateReady {
		t.Fatal("candidate with sufficient evidence should be ready for risk review")
	}
	if result.HardReject {
		t.Fatalf("ready candidate should not be hard-rejected: %#v", result.RejectReasons)
	}
	if result.NextRequiredPhase != NextPhaseRiskReview {
		t.Fatalf("next phase = %q, want %q", result.NextRequiredPhase, NextPhaseRiskReview)
	}
}

func TestGatekeeperNeverApprovesExecutesOrAllowsBrokerExecution(t *testing.T) {
	candidate := completeStructuredCandidate()

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if result.ApprovalGranted {
		t.Fatal("gatekeeper must not approve candidates")
	}
	if result.ExecutionInstructionCreated {
		t.Fatal("gatekeeper must not create execution instructions")
	}
	if result.BrokerExecutionAllowed {
		t.Fatal("gatekeeper must not allow broker execution")
	}
}

func TestGatekeeperRejectsLeverageAboveOne(t *testing.T) {
	candidate := completeStructuredCandidate()
	metadata := json.RawMessage(`{"requestedLeverage":1.25}`)
	candidate.Metadata = &metadata

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if !result.HardReject {
		t.Fatal("leverage above 1.0 must hard-reject the gate")
	}
	if result.GateReady {
		t.Fatal("leveraged candidate must not be gate-ready")
	}
	if !containsString(result.RejectReasons, "leverage_requested_above_1") {
		t.Fatalf("reject reasons = %#v, want leverage_requested_above_1", result.RejectReasons)
	}
}

func TestGatekeeperRejectsPrematureApprovalOrExecutionFlags(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.Status = StatusApproved
	executionID := uuid.New()
	candidate.ExecutionInstructionID = &executionID

	result := EvaluateCandidateGate(candidate, sufficientEvidenceScore(candidate.ID), time.Now().UTC())

	if !result.HardReject {
		t.Fatal("premature approval and execution flags must hard-reject the gate")
	}
	for _, reason := range []string{"approval_granted_too_early", "execution_instruction_created_too_early"} {
		if !containsString(result.RejectReasons, reason) {
			t.Fatalf("reject reasons = %#v, want %q", result.RejectReasons, reason)
		}
	}
	if result.ApprovalGranted || result.ExecutionInstructionCreated || result.BrokerExecutionAllowed {
		t.Fatal("gate result safety flags must remain false even when rejecting unsafe inputs")
	}
}

func sufficientEvidenceScore(candidateID uuid.UUID) EvidenceScoreSummary {
	return EvidenceScoreSummary{
		CandidateID:          candidateID,
		SupportScore:         0.90,
		QualityScore:         0.90,
		FreshnessScore:       1.00,
		OverallEvidenceScore: 0.81,
		EvidenceItemCount:    1,
		SupportingItemCount:  1,
		EvidenceStatus:       EvidenceStatusSufficient,
		EvidenceReady:        true,
		EvidenceGateReady:    true,
	}
}
