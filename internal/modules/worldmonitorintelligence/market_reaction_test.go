package worldmonitorintelligence

import (
	"testing"
	"time"
)

func TestCorrelateMarketReactionIsReadOnlyAndUsesNearestPoints(t *testing.T) {
	eventAt := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{EventType: "macro_rates", Members: []CanonicalObservation{{ID: "event", ObservedAt: &eventAt}}}
	baseline := 0.01
	points := []MarketPoint{{Instrument: "SPY", At: eventAt.Add(-5 * time.Minute), Price: 100}, {Instrument: "SPY", At: eventAt.Add(5 * time.Minute), Price: 102}, {Instrument: "SPY", At: eventAt.Add(30 * time.Minute), Price: 103}}
	result, err := CorrelateMarketReaction(cluster, points, map[string]float64{"SPY": baseline}, 15*time.Minute)
	if err != nil || len(result) != 1 || result[0].State != ReactionPositive || result[0].BeforePrice != 100 || result[0].AfterPrice != 102 || result[0].AbnormalReturn == nil || *result[0].AbnormalReturn < 0.009 {
		t.Fatalf("reaction=%+v err=%v", result, err)
	}
}

func TestCorrelateMarketReactionMakesMissingBaselineAndPointsVisible(t *testing.T) {
	eventAt := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{Members: []CanonicalObservation{{ID: "event", CollectedAt: eventAt}}}
	result, err := CorrelateMarketReaction(cluster, []MarketPoint{{Instrument: "SPY", At: eventAt.Add(5 * time.Minute), Price: 100}}, nil, time.Hour)
	if err != nil || len(result) != 1 || result[0].State != ReactionUnknown || len(result[0].Unknowns) == 0 {
		t.Fatalf("missing evidence reaction=%+v err=%v", result, err)
	}
}

func TestLinkPlausibleInstrumentsRemainsTaxonomyBound(t *testing.T) {
	links := LinkPlausibleInstruments(EventCluster{EventType: "energy_oil"}, EntityExtraction{})
	if len(links) != 2 || links[0].Instrument != "USO" || links[1].Instrument != "XLE" {
		t.Fatalf("links=%+v", links)
	}
}
