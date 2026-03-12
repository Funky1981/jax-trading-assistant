package candidates

import (
	"errors"
	"testing"
)

// TestStatusConstants verifies the lifecycle status strings match the DB enum.
// If these drift, migrations and queries will silently mismatch.
func TestStatusConstants(t *testing.T) {
	cases := map[string]string{
		"detected":          StatusDetected,
		"qualified":         StatusQualified,
		"blocked":           StatusBlocked,
		"awaiting_approval": StatusAwaitingApproval,
		"approved":          StatusApproved,
		"rejected":          StatusRejected,
		"expired":           StatusExpired,
		"submitted":         StatusSubmitted,
		"filled":            StatusFilled,
		"cancelled":         StatusCancelled,
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("status constant: expected %q, got %q", want, got)
		}
	}
}

// TestErrDuplicateCandidate_IsSentinel ensures the error can be reliably
// detected with errors.Is by callers (e.g. the watcher suppressing duplicates).
func TestErrDuplicateCandidate_IsSentinel(t *testing.T) {
	if ErrDuplicateCandidate == nil {
		t.Fatal("ErrDuplicateCandidate must not be nil")
	}
	if !errors.Is(ErrDuplicateCandidate, ErrDuplicateCandidate) {
		t.Error("ErrDuplicateCandidate must satisfy errors.Is with itself")
	}
}

// TestProposalRequest_Fields sanity-checks that the ProposalRequest struct
// has the expected shape (fields used by the watcher).
func TestProposalRequest_Fields(t *testing.T) {
	// Verify the struct can be fully populated — will fail to compile if fields are removed.
	conf := 0.8
	reasoning := "test"
	_ = ProposalRequest{
		StrategyInstanceID: [16]byte{},
		Symbol:             "AAPL",
		SignalType:         "BUY",
		Confidence:         &conf,
		Reasoning:          &reasoning,
		DataProvenance:     "unit-test",
	}
}

// TestCandidate_BlockedNeverAwaiting documents the invariant: a blocked
// candidate must never appear in the approval queue.
// This is enforced by the Store.UpdateStatus transition guard (status=blocked
// has no path to awaiting_approval). We verify the constant is distinct so
// a WHERE status='awaiting_approval' query can never match a blocked row.
func TestCandidate_BlockedNeverAwaiting(t *testing.T) {
	if StatusBlocked == StatusAwaitingApproval {
		t.Error("blocked and awaiting_approval must be distinct status values")
	}
}

// TestCandidate_ExpiredNeverAwaiting documents that an expired candidate
// cannot sneak back into the approval queue.
func TestCandidate_ExpiredNeverAwaiting(t *testing.T) {
	if StatusExpired == StatusAwaitingApproval {
		t.Error("expired and awaiting_approval must be distinct status values")
	}
}
