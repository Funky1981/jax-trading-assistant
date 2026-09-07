package worldmonitorintelligence

import (
	"testing"
	"time"
)

func TestExtractEntitiesAndGeographiesIsDeterministicAndExplicit(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	cluster := EventCluster{ContractVersion: IntelligenceContractVersion, Algorithm: ClusterAlgorithmV1, ID: "ecl_test", EventType: "macro_rates", Members: []CanonicalObservation{{ID: "a", Title: "Federal Reserve policy decision affects United States markets", Summary: "The Fed publishes its statement.", CollectedAt: now}}}
	result := ExtractEntities(cluster)
	if result.Algorithm != EntityExtractionAlgorithmV1 || len(result.Entities) != 1 || result.Entities[0].ID != "ent_federal_reserve" {
		t.Fatalf("entities=%+v", result)
	}
	if len(result.Geographies) != 1 || result.Geographies[0].ID != "geo_united_states" {
		t.Fatalf("geographies=%+v", result)
	}
	if len(result.Unknowns) != 0 {
		t.Fatalf("unexpected unknowns=%v", result.Unknowns)
	}
}

func TestExtractEntitiesDoesNotInventUnsupportedNames(t *testing.T) {
	result := ExtractEntities(EventCluster{ContractVersion: IntelligenceContractVersion, Algorithm: ClusterAlgorithmV1, Members: []CanonicalObservation{{Title: "Unrecognised local authority announces incident"}}})
	if len(result.Entities) != 0 || len(result.Geographies) != 0 || len(result.Unknowns) != 2 {
		t.Fatalf("unsupported extraction=%+v", result)
	}
}
