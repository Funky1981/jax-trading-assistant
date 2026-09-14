package main

import "testing"

func TestValidateDateRangeRejectsSealedHoldout(t *testing.T) {
	if err := validateDateRange("2016-01-01", "2025-01-01"); err == nil {
		t.Fatal("expected sealed-holdout rejection")
	}
	if err := validateDateRange("2016-01-01", "2024-12-31"); err != nil {
		t.Fatalf("unexpected pre-holdout rejection: %v", err)
	}
}

func TestGeometryValidityIsStrictlyPreEntry(t *testing.T) {
	for _, tc := range []struct {
		stop, open, target float64
		want               bool
	}{
		{10, 11, 12, true},
		{10, 10, 12, false},
		{10, 12, 12, false},
		{12, 11, 10, false},
	} {
		if got := geometryValid(tc.stop, tc.open, tc.target); got != tc.want {
			t.Fatalf("geometryValid(%v,%v,%v)=%v, want %v", tc.stop, tc.open, tc.target, got, tc.want)
		}
	}
}

func TestBaseCommissionMinimumAndMaximum(t *testing.T) {
	minimum := netReturn(100, 101, 1, 0, 0, 0)
	if minimum >= 0 {
		t.Fatalf("minimum commission arithmetic should charge both legs, got %v", minimum)
	}
	maximum := netReturn(0.01, 0.02, 1000000, 0, 0, 0)
	if maximum <= 0 {
		t.Fatalf("expected positive result under capped commission, got %v", maximum)
	}
}

func TestSeedDerivationIsDeterministicAndDomainSeparated(t *testing.T) {
	manifest := "manifest-test"
	seedA := seedFor(manifest, "|instrument-year-bootstrap-v2|")
	seedB := seedFor(manifest, "|instrument-year-bootstrap-v2|")
	if seedA != seedB {
		t.Fatal("seed derivation is not deterministic")
	}
	if seedA == seedFor(manifest, "|timestamp-placebo-v1|") {
		t.Fatal("seed domains unexpectedly collide")
	}
}
