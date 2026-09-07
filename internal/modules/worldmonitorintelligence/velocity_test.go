package worldmonitorintelligence

import (
	"testing"
	"time"
)

func TestAssessVelocityDetectsBaselineDeviation(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{Members: []CanonicalObservation{
		{ID: "a", CollectedAt: now.Add(-10 * time.Minute)},
		{ID: "b", CollectedAt: now.Add(-20 * time.Minute)},
		{ID: "c", CollectedAt: now.Add(-30 * time.Minute)},
	}}
	history := []time.Time{now.Add(-25 * time.Hour)}
	signal, err := AssessVelocity(cluster, history, now, time.Hour, 24*time.Hour)
	if err != nil || signal.State != VelocityAbnormal || signal.WindowEventCount != 3 || signal.BaselineEventCount != 1 || signal.RateRatio <= 3 {
		t.Fatalf("signal=%+v err=%v", signal, err)
	}
}

func TestAssessVelocityDoesNotEmitInfinityForZeroBaseline(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{Members: []CanonicalObservation{{ID: "new", CollectedAt: now}}}
	signal, err := AssessVelocity(cluster, nil, now, time.Hour, 24*time.Hour)
	if err != nil || signal.State != VelocityAbnormal || signal.RateRatio != 999 || len(signal.Unknowns) == 0 {
		t.Fatalf("zero baseline signal=%+v err=%v", signal, err)
	}
}
