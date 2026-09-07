package worldmonitorintelligence

import (
	"strings"
	"testing"
	"time"
)

func TestBuildClustersDeduplicatesAndClustersMultiSourceObservations(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	makeObservation := func(source, id, title string, raw string) Observation {
		return Observation{ID: id, SourceID: source, SourceName: source, EventType: "macro_rates", Title: title, Summary: "Federal Reserve policy meeting decision", SourceURL: "https://example.com/" + id, CollectedAt: now, RawPayload: []byte(raw), Freshness: FreshnessFresh}
	}
	input := []Observation{
		makeObservation("sec", "a", "Federal Reserve policy meeting decision", `{"source":"sec"}`),
		makeObservation("fed", "b", "Federal Reserve announces policy meeting decision", `{"source":"fed"}`),
		makeObservation("sec", "a", "Federal Reserve policy meeting decision", `{"source":"sec"}`),
	}
	clusters, err := BuildClusters(input)
	if err != nil || len(clusters) != 1 || len(clusters[0].Members) != 2 {
		t.Fatalf("clusters=%+v err=%v", clusters, err)
	}
	if clusters[0].ContractVersion != IntelligenceContractVersion || clusters[0].Algorithm != ClusterAlgorithmV1 || !strings.HasPrefix(clusters[0].ID, "ecl_") {
		t.Fatalf("cluster identity/version missing: %+v", clusters[0])
	}
	if len(clusters[0].Unknowns) != 0 {
		t.Fatalf("corroborated cluster has unknowns=%v", clusters[0].Unknowns)
	}
}

func TestBuildClustersRejectsConflictingReplayAndSeparatesUnrelatedEvents(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	base := Observation{ID: "same", SourceID: "source", EventType: "energy_oil", Title: "Oil supply disruption", CollectedAt: now, RawPayload: []byte(`{"v":1}`), Freshness: FreshnessFresh}
	conflict := base
	conflict.RawPayload = []byte(`{"v":2}`)
	if _, err := BuildClusters([]Observation{base, conflict}); err == nil {
		t.Fatal("expected conflicting replay rejection")
	}
	other := base
	other.ID = "other"
	other.EventType = "cyber_outage"
	other.Title = "Payment network outage"
	clusters, err := BuildClusters([]Observation{base, other})
	if err != nil || len(clusters) != 2 {
		t.Fatalf("unrelated clusters=%+v err=%v", clusters, err)
	}
	if len(clusters[0].Members) != 1 || len(clusters[0].Unknowns) != 1 {
		t.Fatalf("single-source unknowns not explicit: %+v", clusters)
	}
}
