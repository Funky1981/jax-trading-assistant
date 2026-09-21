package main

import (
	"testing"
	"time"
)

func TestExploratoryWorkerHealthTracksFailuresAndLastSuccess(t *testing.T) {
	at := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	var health exploratoryWorkerHealth
	health.entryFailure(at, nil)
	health.entryFailure(at.Add(time.Minute), nil)
	if health.entryConsecutiveFailures != 2 || !health.lastEntrySuccess.IsZero() {
		t.Fatalf("entry failure state = %#v", health)
	}
	health.entrySuccess(at.Add(2 * time.Minute))
	if health.entryConsecutiveFailures != 0 || !health.lastEntrySuccess.Equal(at.Add(2*time.Minute)) {
		t.Fatalf("entry success state = %#v", health)
	}
	health.reviewFailure(at, nil)
	health.reviewSuccess(at.Add(3 * time.Minute))
	if health.reviewConsecutiveFailures != 0 || !health.lastReviewSuccess.Equal(at.Add(3*time.Minute)) {
		t.Fatalf("review state = %#v", health)
	}
}
