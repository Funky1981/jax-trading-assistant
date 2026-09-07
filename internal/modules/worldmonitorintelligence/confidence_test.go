package worldmonitorintelligence

import (
	"testing"
	"time"
)

func TestAssessConfidenceRequiresIndependentSourceCorroboration(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{Members: []CanonicalObservation{
		{ID: "a", SourceID: "sec", PublishedAt: &now, Freshness: FreshnessFresh},
		{ID: "b", SourceID: "fed", ObservedAt: &now, Freshness: FreshnessFresh},
	}}
	assessment, err := AssessConfidence(cluster)
	if err != nil || assessment.Corroboration != CorroborationCorroborated || assessment.IndependentSourceCount != 2 || assessment.Score < 0.75 || assessment.Band != "HIGH" {
		t.Fatalf("assessment=%+v err=%v", assessment, err)
	}
	if len(assessment.Unknowns) != 0 {
		t.Fatalf("unexpected unknowns=%v", assessment.Unknowns)
	}
}

func TestAssessConfidenceExposesStaleAndMissingTimeUnknowns(t *testing.T) {
	assessment, err := AssessConfidence(EventCluster{Members: []CanonicalObservation{{ID: "a", SourceID: "one", Freshness: FreshnessStale}}})
	if err != nil || assessment.Corroboration != CorroborationUncorroborated || assessment.Band != "LOW" || len(assessment.Unknowns) < 3 {
		t.Fatalf("assessment=%+v err=%v", assessment, err)
	}
}
