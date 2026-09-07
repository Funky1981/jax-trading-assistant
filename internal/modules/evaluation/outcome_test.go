package evaluation

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestTrackRecommendationOutcomeRecordsObservedWindowsWithoutExecution(t *testing.T) {
	caseFile := replayFixture(t)
	decision := caseFile.DecisionAt
	observations := make([]OutcomeObservation, 0, 20)
	for index := 1; index <= 20; index++ {
		price := 100.0 + float64(index-1)/2
		if index == 3 {
			price = 98
		}
		benchmark := 200.0 + float64(index)/2
		observations = append(observations, OutcomeObservation{At: decision.Add(time.Duration(index) * time.Hour), Price: price, BenchmarkPrice: &benchmark})
	}
	recordedAt := decision.Add(21 * time.Hour)
	outcome, err := TrackRecommendationOutcome(caseFile, observations, nil, "fixture-market-observations", recordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.Validate(caseFile); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(outcome.ID, "outcome_") || outcome.RecommendationID != caseFile.Recommendation.ID || outcome.OriginalThesis != caseFile.Recommendation.Thesis {
		t.Fatalf("outcome did not preserve original recommendation identity: %#v", outcome)
	}
	if len(outcome.Windows) != 4 || !outcome.Windows[3].Complete {
		t.Fatalf("expected four windows with a complete twenty-observation window: %#v", outcome.Windows)
	}
	if math.Abs(*outcome.Windows[3].Return-0.095) > 1e-9 {
		t.Fatalf("unexpected twenty-observation return: %v", *outcome.Windows[3].Return)
	}
	if *outcome.Windows[3].MaximumFavourableExcursion <= 0 || *outcome.Windows[3].MaximumAdverseExcursion >= 0 {
		t.Fatalf("expected excursion bounds: %#v", outcome.Windows[3])
	}
}

func TestTrackRecommendationOutcomeRejectsInvalidObservationTimeline(t *testing.T) {
	caseFile := replayFixture(t)
	if _, err := TrackRecommendationOutcome(caseFile, []OutcomeObservation{{At: caseFile.DecisionAt, Price: 100}}, nil, "fixture", caseFile.DecisionAt.Add(time.Hour)); err == nil {
		t.Fatal("expected pre-decision observation rejection")
	}
	first := caseFile.DecisionAt.Add(2 * time.Hour)
	if _, err := TrackRecommendationOutcome(caseFile, []OutcomeObservation{{At: first, Price: 100}, {At: first, Price: 101}}, nil, "fixture", caseFile.DecisionAt.Add(3*time.Hour)); err == nil {
		t.Fatal("expected non-increasing observation rejection")
	}
}

func TestTrackRecommendationOutcomeDoesNotInventMissingWindows(t *testing.T) {
	caseFile := replayFixture(t)
	observations := []OutcomeObservation{{At: caseFile.DecisionAt.Add(time.Hour), Price: 100}}
	outcome, err := TrackRecommendationOutcome(caseFile, observations, nil, "fixture", caseFile.DecisionAt.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, window := range outcome.Windows {
		if window.Horizon != OutcomeNextObservation && window.Complete {
			t.Fatalf("unexpected complete horizon with insufficient observations: %#v", window)
		}
		if window.Horizon != OutcomeNextObservation && window.Return != nil {
			t.Fatalf("missing horizon received synthetic return: %#v", window)
		}
	}
}

func TestTrackRecommendationOutcomeInvalidationUsesOnlyKnownObservations(t *testing.T) {
	caseFile := replayFixture(t)
	invalidationAt := caseFile.DecisionAt.Add(3 * time.Hour)
	observations := []OutcomeObservation{{At: caseFile.DecisionAt.Add(time.Hour), Price: 100}, {At: caseFile.DecisionAt.Add(2 * time.Hour), Price: 95}, {At: caseFile.DecisionAt.Add(4 * time.Hour), Price: 80}}
	outcome, err := TrackRecommendationOutcome(caseFile, observations, &invalidationAt, "fixture", caseFile.DecisionAt.Add(5*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	window := outcome.Windows[len(outcome.Windows)-1]
	if window.Horizon != OutcomeInvalidation || !window.Complete || window.EndAt != observations[1].At || window.EndPrice != 95 {
		t.Fatalf("invalidation window used unavailable observations: %#v", window)
	}
}

func TestRecommendationOutcomeRejectsMutationOfOriginalRecommendation(t *testing.T) {
	caseFile := replayFixture(t)
	observations := []OutcomeObservation{{At: caseFile.DecisionAt.Add(time.Hour), Price: 100}}
	outcome, err := TrackRecommendationOutcome(caseFile, observations, nil, "fixture", caseFile.DecisionAt.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	caseFile.Recommendation.Thesis = "mutated after decision"
	if err := outcome.Validate(caseFile); err == nil {
		t.Fatal("expected mutated recommendation to invalidate outcome")
	}
}
