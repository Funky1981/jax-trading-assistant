package candidates

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestPersistedPaperTicketRequiresSuccessfulBoundaryResult(t *testing.T) {
	candidate := approvalReadyCandidate()
	blocked := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          ApprovalEligibilityResult{CandidateID: candidate.ID, ApprovalEligible: false},
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-blocked",
		SourceApprovalID:     uuid.New(),
		CreatedAt:            fixedApprovalReviewTime(),
	})

	_, err := NewPersistedPaperTicket(candidate, sufficientEvidenceScore(candidate.ID), ApprovalEligibilityResult{}, blocked)
	if err == nil {
		t.Fatal("blocked paper ticket boundary result must not become persisted paper ticket")
	}
}

func TestPersistedPaperTicketCopiesReviewFieldsAndSafetyDefaults(t *testing.T) {
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

	ticket, err := NewPersistedPaperTicket(candidate, sufficientEvidenceScore(candidate.ID), eligibility, result)
	if err != nil {
		t.Fatalf("new persisted paper ticket: %v", err)
	}

	if ticket.Status != PaperTicketStatusPaperTicketCreated {
		t.Fatalf("status = %q, want %q", ticket.Status, PaperTicketStatusPaperTicketCreated)
	}
	if ticket.PaperTicketID != result.PaperTicketID || ticket.CandidateID != candidate.ID || ticket.SourceApprovalID == nil || *ticket.SourceApprovalID != approvalID {
		t.Fatalf("unexpected identity fields: %+v", ticket)
	}
	if ticket.Symbol != candidate.Symbol || ticket.Direction != candidate.Direction || ticket.SetupType != candidate.SetupType || ticket.CatalystSummary != candidate.CatalystSummary {
		t.Fatalf("trade plan fields not copied from candidate: %+v", ticket)
	}
	if ticket.EvidenceStatus != string(EvidenceStatusSufficient) || ticket.GateStatus != candidate.GateStatus || ticket.RiskStatus != candidate.RiskStatus || ticket.ApprovalStatus != eligibility.ApprovalStatus {
		t.Fatalf("review statuses not copied: %+v", ticket)
	}
	if !ticket.PaperOnly || ticket.BrokerExecutionAllowed || ticket.ExecutionInstructionCreated || ticket.LiveTradingAllowed || ticket.LeverageAllowed {
		t.Fatalf("persisted paper ticket safety defaults are unsafe: %+v", ticket)
	}
}

func TestPaperTicketReviewJSONStaysPaperReviewOnly(t *testing.T) {
	candidate := approvalReadyCandidate()
	eligibility := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())
	result := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          eligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  "approval-paper-ready",
		SourceApprovalID:     uuid.New(),
		CreatedAt:            fixedApprovalReviewTime(),
	})
	ticket, err := NewPersistedPaperTicket(candidate, sufficientEvidenceScore(candidate.ID), eligibility, result)
	if err != nil {
		t.Fatalf("new persisted paper ticket: %v", err)
	}

	payload, err := json.Marshal(ticket.ReviewModel())
	if err != nil {
		t.Fatalf("marshal review model: %v", err)
	}
	forbidden := []string{
		"brokerExecutionAllowed",
		"executionInstructionCreated",
		"liveTradingAllowed",
		"leverageAllowed",
		"executionReady",
		"autoExecutionEnabled",
	}
	for _, key := range forbidden {
		if jsonContainsKey(payload, key) {
			t.Fatalf("review model JSON exposed forbidden key %q: %s", key, string(payload))
		}
	}
}

func jsonContainsKey(payload []byte, key string) bool {
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return false
	}
	_, ok := object[key]
	return ok
}
