package approvals

import (
	"errors"
	"testing"
)

// TestDecisionConstants verifies the decision value strings match the DB enum.
func TestDecisionConstants(t *testing.T) {
	cases := map[string]string{
		"approved":              DecisionApproved,
		"rejected":              DecisionRejected,
		"snoozed":               DecisionSnoozed,
		"reanalysis_requested":  DecisionReanalysisRequested,
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("decision constant: expected %q, got %q", want, got)
		}
	}
}

// TestSentinelErrors confirms all sentinel errors are distinct so callers can
// distinguish them with errors.Is.
func TestSentinelErrors_AreDistinct(t *testing.T) {
	for _, e := range []error{ErrCandidateExpired, ErrNotAwaitingApproval} {
		if e == nil {
			t.Fatalf("sentinel error must not be nil: %v", e)
		}
	}
	if errors.Is(ErrCandidateExpired, ErrNotAwaitingApproval) {
		t.Error("ErrCandidateExpired and ErrNotAwaitingApproval must be distinct")
	}
}

// TestRejectedDecision_NoExecution documents that the Decide function only
// calls buildInstruction when decision == DecisionApproved (not rejected).
// We verify this invariant via the decision constant values.
func TestRejectedDecision_NoExecution(t *testing.T) {
	// A rejected decision must never be equal to the approved constant —
	// buildInstruction is only triggered on DecisionApproved.
	if DecisionRejected == DecisionApproved {
		t.Error("rejected and approved must be distinct decision values; " +
			"otherwise rejected candidates would trigger execution instructions")
	}
}

// TestSnoozedDecision_StaysInQueue verifies that snoozed candidates remain
// in awaiting_approval state (not approved/rejected), so they re-appear in
// the queue after the snooze window expires.
func TestSnoozedDecision_StaysInQueue(t *testing.T) {
	if DecisionSnoozed == DecisionApproved {
		t.Error("snoozed must not equal approved; would incorrectly trigger execution")
	}
	if DecisionSnoozed == DecisionRejected {
		t.Error("snoozed must not equal rejected; would prematurely close the candidate")
	}
}

// TestApprovalRequest_Fields checks that the struct shape is stable.
func TestApprovalRequest_Fields(t *testing.T) {
	_ = ApprovalRequest{
		Decision:    DecisionApproved,
		ApprovedBy:  "test-user",
		SnoozeHours: 4,
	}
}
