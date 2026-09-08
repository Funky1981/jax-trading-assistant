package papertrading

import (
	"testing"
	"time"
)

func TestAcceleratedSoakProtocolIsExplicitlyNotRealForwardEvidence(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	run, err := NewSoakRun(DefaultSoakProtocol(), start, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Record(SoakObservation{At: start.Add(90 * 24 * time.Hour), Recommendations: 1, EligiblePaperIntents: 1, Orders: 1, Fills: 1, PartialFills: 1, Crashes: 1}); err != nil {
		t.Fatal(err)
	}
	if err := run.Record(SoakObservation{At: start.Add(180 * 24 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := run.Complete(start.Add(180 * 24 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if run.Status != SoakCompleted || run.EvidenceClass != AcceleratedSyntheticSoak || run.ActualForwardDuration != 0 || run.RunID == "" {
		t.Fatalf("accelerated soak = %#v", run)
	}
}

func TestSoakFailsOnSafetyOrReconciliationViolation(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	run, err := NewSoakRun(DefaultSoakProtocol(), start, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Record(SoakObservation{At: start.Add(time.Hour), LivePathActivated: true}); err != nil {
		t.Fatal(err)
	}
	if run.Status != SoakFailed || len(run.FailureReasons) != 1 || run.FailureReasons[0] != "LIVE_PATH_ACTIVATED" {
		t.Fatalf("unsafe soak status = %#v", run)
	}
}
