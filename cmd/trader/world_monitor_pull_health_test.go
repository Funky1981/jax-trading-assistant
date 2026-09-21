package main

import (
	"errors"
	"testing"
	"time"
)

func TestWorldMonitorPullHealthRetainsFailureAndLastSuccess(t *testing.T) {
	at := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	var health worldMonitorPullHealth
	health.failure(at, errors.New("provider unavailable"))
	health.failure(at.Add(time.Minute), errors.New("provider timeout"))
	if health.ConsecutiveFailures != 2 || !health.LastSuccess.IsZero() || !health.LastFailure.Equal(at.Add(time.Minute)) || health.LastError != "provider timeout" {
		t.Fatalf("failure state = %#v", health)
	}
	health.success(at.Add(2*time.Minute), 42)
	if health.ConsecutiveFailures != 0 || health.LastCursor != 42 || !health.LastSuccess.Equal(at.Add(2*time.Minute)) || health.LastError != "" {
		t.Fatalf("success state = %#v", health)
	}
}
