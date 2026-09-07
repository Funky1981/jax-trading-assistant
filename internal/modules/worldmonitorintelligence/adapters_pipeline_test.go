package worldmonitorintelligence

import (
	"testing"
	"time"
)

func TestEvaluateEvidenceAdaptersPreservesDistinctExternalLimitations(t *testing.T) {
	evaluations, err := EvaluateEvidenceAdapters(DefaultAdapterSpecs())
	if err != nil || len(evaluations) != 3 {
		t.Fatalf("evaluations=%+v err=%v", evaluations, err)
	}
	byKind := map[AdapterKind]AdapterEvaluation{}
	for _, evaluation := range evaluations {
		byKind[evaluation.Kind] = evaluation
		if !evaluation.ReadOnly || !evaluation.CanCorroborate || evaluation.Algorithm != AdapterEvaluationAlgorithmV1 {
			t.Fatalf("unsafe adapter evaluation=%+v", evaluation)
		}
	}
	if byKind[AdapterACLED].Status != AdapterDeferred || byKind[AdapterAIS].Status != AdapterEvaluatedReadOnly || len(byKind[AdapterAIS].Unknowns) == 0 || len(byKind[AdapterPredictionMarket].Unknowns) == 0 {
		t.Fatalf("adapter limitations lost: %+v", byKind)
	}
	if err := (AdapterEvidence{AdapterID: "adapter_ais_v1", ExternalID: "ais-1", SchemaVersion: "ais-vessel/v1", RawPayload: []byte(`{"mmsi":1}`)}).Validate(); err != nil {
		t.Fatalf("valid adapter evidence rejected: %v", err)
	}
	if err := (AdapterEvidence{AdapterID: "adapter_ais_v1", ExternalID: "ais-1", SchemaVersion: "ais-vessel/v1"}).Validate(); err == nil {
		t.Fatal("missing raw adapter evidence accepted")
	}
}

func TestPhase04ExitConditionReplayedMultiSourceEvent(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	publication := now.Add(-10 * time.Minute)
	makeObservation := func(source, id, title, raw string) Observation {
		return Observation{ID: id, SourceID: source, SourceName: source, EventType: "macro_rates", Title: title, Summary: "Federal Reserve policy meeting decision affects United States markets", SourceURL: "https://" + source + ".example/event/" + id, PublishedAt: &publication, ObservedAt: &publication, CollectedAt: now.Add(-5 * time.Minute), ProviderRev: "world-monitor-events/v1", RawPayload: []byte(raw), Freshness: FreshnessFresh}
	}
	input := PipelineInput{
		Observations: []Observation{
			makeObservation("sec", "evt-sec", "Federal Reserve policy meeting decision", `{"source":"sec","bytes":1}`),
			makeObservation("fed", "evt-fed", "Federal Reserve announces policy meeting decision", `{"source":"fed","bytes":2}`),
			makeObservation("sec", "evt-sec", "Federal Reserve policy meeting decision", `{"source":"sec","bytes":1}`),
		},
		History: []time.Time{now.Add(-25 * time.Hour)}, Now: now,
		VelocityWindow: time.Hour, BaselineWindow: 24 * time.Hour, ReactionWindow: 15 * time.Minute,
		MarketPoints: []MarketPoint{{Instrument: "TLT", At: now.Add(-10 * time.Minute), Price: 100}, {Instrument: "TLT", At: now.Add(10 * time.Minute), Price: 101}},
		Baselines:    map[string]float64{"TLT": 0.002}, AdapterSpecs: DefaultAdapterSpecs(),
	}
	results, err := Analyze(input)
	if err != nil || len(results) != 1 {
		t.Fatalf("results=%+v err=%v", results, err)
	}
	result := results[0]
	if len(result.Cluster.Members) != 2 || result.Confidence.Corroboration != CorroborationCorroborated || len(result.Entities.Entities) != 1 || len(result.Entities.Geographies) != 1 || len(result.PlausibleInstruments) == 0 || len(result.MarketReactions) != 1 || result.TradeCandidateCreated {
		t.Fatalf("exit condition result=%+v", result)
	}
	if len(result.Unknowns) == 0 {
		t.Fatal("replayed corroborated event did not preserve explicit unknowns")
	}
}
