package candidates

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestPaperTicketCreationRequiresApprovalEligibilityAndHumanApproval(t *testing.T) {
	candidate := approvalReadyCandidate()
	eligibility := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())

	withoutHumanApproval := CreatePaperTicket(PaperTicketRequest{
		Candidate:   candidate,
		Eligibility: eligibility,
		CreatedAt:   fixedApprovalReviewTime(),
	})
	if withoutHumanApproval.Status != PaperTicketStatusApprovalRequired {
		t.Fatalf("status = %q, want %q", withoutHumanApproval.Status, PaperTicketStatusApprovalRequired)
	}
	if withoutHumanApproval.CanCreateTicket {
		t.Fatal("candidate without human approval must not create a paper ticket")
	}
	if withoutHumanApproval.ExecutionInstructionCreated || withoutHumanApproval.BrokerExecutionAllowed || withoutHumanApproval.LiveTradingAllowed {
		t.Fatal("blocked paper ticket result must keep execution and live-trading flags false")
	}

	withoutEligibility := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          ApprovalEligibilityResult{CandidateID: candidate.ID, ApprovalEligible: false},
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-1",
		SourceApprovalID:     uuid.New(),
		CreatedAt:            fixedApprovalReviewTime(),
	})
	if withoutEligibility.Status != PaperTicketStatusApprovalNotReady {
		t.Fatalf("status = %q, want %q", withoutEligibility.Status, PaperTicketStatusApprovalNotReady)
	}
	if withoutEligibility.CanCreateTicket {
		t.Fatal("candidate without approval eligibility must not create a paper ticket")
	}
}

func TestPaperTicketCreationRejectsFailedRiskReviewAndLeverage(t *testing.T) {
	candidate := approvalReadyCandidate()
	risk := readyRiskReview(candidate)
	risk.RiskReady = false
	risk.RiskStatus = RiskStatusRiskTooHigh
	eligibility := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), risk, fixedApprovalReviewTime())

	result := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          eligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-risk-failed",
		CreatedAt:            fixedApprovalReviewTime(),
	})
	if result.CanCreateTicket {
		t.Fatal("candidate with failed risk review must not create a paper ticket")
	}
	if result.Status != PaperTicketStatusApprovalNotReady {
		t.Fatalf("status = %q, want %q", result.Status, PaperTicketStatusApprovalNotReady)
	}
	if !containsString(result.RejectReasons, "approval_eligibility_not_passed") {
		t.Fatalf("reject reasons = %v, want approval_eligibility_not_passed", result.RejectReasons)
	}

	leveraged := approvalReadyCandidate()
	metadata := json.RawMessage(`{"requestedLeverage":1.1}`)
	leveraged.Metadata = &metadata
	leveragedEligibility := EvaluateApprovalEligibility(leveraged, sufficientEvidenceScore(leveraged.ID), readyRiskGate(leveraged.ID), readyRiskReview(leveraged), fixedApprovalReviewTime())

	leveragedResult := CreatePaperTicket(PaperTicketRequest{
		Candidate:            leveraged,
		Eligibility:          leveragedEligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-leveraged",
		CreatedAt:            fixedApprovalReviewTime(),
	})
	if leveragedResult.CanCreateTicket {
		t.Fatal("leveraged candidate must not create a paper ticket")
	}
	if !containsString(leveragedResult.RejectReasons, "leverage_requested_above_1") {
		t.Fatalf("reject reasons = %v, want leverage_requested_above_1", leveragedResult.RejectReasons)
	}
}

func TestPaperTicketCreationForHumanApprovedEligibleCandidateIsPaperOnly(t *testing.T) {
	candidate := approvalReadyCandidate()
	eligibility := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())
	approvalID := uuid.New()

	result := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          eligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-paper-ready",
		SourceApprovalID:     approvalID,
		CreatedAt:            fixedApprovalReviewTime(),
	})

	if !result.CanCreateTicket {
		t.Fatalf("eligible human-approved candidate should create paper-ticket-ready result: rejects=%v", result.RejectReasons)
	}
	if result.Status != PaperTicketStatusPaperTicketReady {
		t.Fatalf("status = %q, want %q", result.Status, PaperTicketStatusPaperTicketReady)
	}
	if result.PaperTicketID == "" || result.CandidateID != candidate.ID || result.SourceApprovalID == nil || *result.SourceApprovalID != approvalID {
		t.Fatalf("unexpected ticket identity fields: %+v", result)
	}
	if result.EntryPrice != floatPtrValue(candidate.EntryPrice) || result.StopLossPrice != floatPtrValue(candidate.StopLoss) || result.TargetPrice != floatPtrValue(candidate.TakeProfit) {
		t.Fatalf("ticket prices did not copy from candidate: %+v", result)
	}
	if result.PositionSize != floatPtrValue(candidate.PositionSize) || result.MaxNormalLoss != floatPtrValue(candidate.MaxNormalLoss) || result.MaxSlippageAdjustedLoss != floatPtrValue(candidate.MaxSlippageAdjustedLoss) {
		t.Fatalf("ticket risk fields did not copy from candidate: %+v", result)
	}
	if result.RewardRiskRatio != floatPtrValue(candidate.ExpectedRewardRiskRatio) {
		t.Fatalf("reward risk ratio = %v, want %v", result.RewardRiskRatio, floatPtrValue(candidate.ExpectedRewardRiskRatio))
	}
	if !result.PaperOnly {
		t.Fatal("paper ticket result must be paper-only")
	}
	if result.BrokerExecutionAllowed || result.ExecutionInstructionCreated || result.LiveTradingAllowed || result.LeverageAllowed {
		t.Fatalf("paper ticket result must not allow broker/live/leverage/execution: %+v", result)
	}
}

func TestPaperTicketCreationRejectsExistingExecutionInstruction(t *testing.T) {
	candidate := approvalReadyCandidate()
	executionID := uuid.New()
	candidate.ExecutionInstructionID = &executionID
	eligibility := ApprovalEligibilityResult{
		CandidateID:                 candidate.ID,
		ApprovalEligible:            true,
		BrokerExecutionAllowed:      false,
		ExecutionInstructionCreated: false,
		LiveTradingAllowed:          false,
	}

	result := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          eligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-existing-exec",
		CreatedAt:            fixedApprovalReviewTime(),
	})
	if result.CanCreateTicket {
		t.Fatal("candidate with existing execution instruction must not create a paper ticket")
	}
	if !containsString(result.RejectReasons, "execution_instruction_created_too_early") {
		t.Fatalf("reject reasons = %v, want execution_instruction_created_too_early", result.RejectReasons)
	}
}
