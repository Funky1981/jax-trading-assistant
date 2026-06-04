package llmcontext

import (
	"testing"
	"time"
)

func TestEventClustererMapsDuplicateHeadlinesToCanonicalEvent(t *testing.T) {
	clusterer := EventClusterer{}
	at := time.Date(2026, 5, 13, 13, 31, 0, 0, time.UTC)

	first := clusterer.Add(EventInput{
		ProviderEventID: "reuters-1",
		Headline:        "US CPI hotter than expected in May",
		EventType:       "inflation",
		PrimaryRegion:   "US",
		AffectedTheme:   "macro_rates",
		AffectedETFs:    []string{"SPY", "QQQ", "TLT"},
		Source:          "Reuters",
		Timestamp:       at,
	})
	second := clusterer.Add(EventInput{
		ProviderEventID: "cnbc-1",
		Headline:        "CPI comes in hotter than expected, yields rise",
		EventType:       "inflation",
		PrimaryRegion:   "US",
		AffectedTheme:   "macro_rates",
		AffectedETFs:    []string{"SPY", "TLT"},
		Source:          "CNBC",
		Timestamp:       at.Add(4 * time.Minute),
	})

	if first.CanonicalEventID != second.CanonicalEventID {
		t.Fatalf("expected duplicate canonical id, got %s and %s", first.CanonicalEventID, second.CanonicalEventID)
	}
	if !second.IsDuplicate {
		t.Fatalf("expected second event to be duplicate: %#v", second)
	}
	if second.ClusterSize != 2 {
		t.Fatalf("expected cluster size 2, got %d", second.ClusterSize)
	}
	if !containsString(second.Sources, "Reuters") || !containsString(second.Sources, "CNBC") {
		t.Fatalf("expected both sources, got %v", second.Sources)
	}
}

func TestEventClustererAllowsOneAICallPerCanonicalEvent(t *testing.T) {
	clusterer := EventClusterer{}
	at := time.Date(2026, 5, 13, 13, 31, 0, 0, time.UTC)
	cluster := clusterer.Add(EventInput{
		Headline:      "Fed cuts rates",
		EventType:     "rates",
		PrimaryRegion: "US",
		AffectedTheme: "macro_rates",
		Source:        "Reuters",
		Timestamp:     at,
	})

	if !clusterer.MarkAICall(cluster.CanonicalEventID) {
		t.Fatal("expected first AI call mark to be allowed")
	}
	if clusterer.MarkAICall(cluster.CanonicalEventID) {
		t.Fatal("expected second AI call mark to be rejected")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
