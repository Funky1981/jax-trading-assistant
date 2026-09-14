package main

import (
	"testing"
	"time"
)

func TestRecoveryDataQualityRangeIsFrozen(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	if !validRecoveryRange(start, end, "2026-09-11") {
		t.Fatal("frozen recovery range rejected")
	}
	for _, test := range []struct {
		start, end, asof string
	}{
		{"2024-12-31", "2026-09-11", "2026-09-11"},
		{"2025-01-01", "2026-09-12", "2026-09-11"},
		{"2025-01-01", "2026-09-11", "2026-09-12"},
	} {
		s, _ := time.Parse("2006-01-02", test.start)
		e, _ := time.Parse("2006-01-02", test.end)
		if validRecoveryRange(s, e, test.asof) {
			t.Fatalf("out-of-contract recovery range accepted: %+v", test)
		}
	}
}

func TestRecoveryProviderDateGuardRejectsOutsideBoundary(t *testing.T) {
	previous := recoveryMode
	recoveryMode = true
	t.Cleanup(func() { recoveryMode = previous })
	for _, date := range []string{"2025-01-01", "2026-09-11"} {
		if err := validateProviderDate(date); err != nil {
			t.Fatalf("valid recovery date rejected: %s: %v", date, err)
		}
	}
	for _, date := range []string{"2024-12-31", "2026-09-12", "2026-09-14", "not-a-date"} {
		if err := validateProviderDate(date); err == nil {
			t.Fatalf("out-of-boundary recovery date accepted: %s", date)
		}
	}
}
