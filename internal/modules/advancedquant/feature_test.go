package advancedquant

import (
	"testing"
	"time"
)

func testFeature(t *testing.T, status FeatureStatus, value *float64) Feature {
	t.Helper()
	at := time.Date(2022, 1, 3, 14, 0, 0, 0, time.UTC)
	f, err := NewFeature(Feature{Name: "source_authority", Version: "v1", EventID: "evt_1", DatasetID: "data_1", SourceIdentities: []string{"raw_1"}, AvailabilityAt: at, CalculatedAt: at.Add(time.Minute), AlgorithmVersion: "alg_v1", Status: status, Value: value, UnknownReason: map[FeatureStatus]string{FeatureKnown: "", FeatureUnknown: "not supplied", FeatureNotEligible: "future source"}[status]})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFeatureContractBindsKnowabilityAndProvenance(t *testing.T) {
	value := 1.0
	f := testFeature(t, FeatureKnown, &value)
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	vector, err := NewFeatureVector(FeatureVector{EventID: "evt_1", DatasetID: "data_1", AsOf: f.AvailabilityAt, Features: []Feature{f}})
	if err != nil {
		t.Fatal(err)
	}
	if err := vector.Validate(); err != nil {
		t.Fatal(err)
	}
	store := NewFeatureStore()
	if err := store.Put(f); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(f.ID)
	if err != nil || got.ID != f.ID {
		t.Fatalf("stored feature lookup failed: got=%+v err=%v", got, err)
	}
	*got.Value = 99
	gotAgain, err := store.Get(f.ID)
	if err != nil || *gotAgain.Value != 1 {
		t.Fatalf("store exposed mutable feature value: %+v err=%v", gotAgain, err)
	}
}

func TestFeatureContractRejectsFutureUnknownAndDuplicateInputs(t *testing.T) {
	value := 2.0
	f := testFeature(t, FeatureKnown, &value)
	vector, err := NewFeatureVector(FeatureVector{EventID: "evt_1", DatasetID: "data_1", AsOf: f.AvailabilityAt.Add(-time.Second), Features: []Feature{f}})
	if err == nil || vector.ID != "" {
		t.Fatal("feature available after vector as-of was accepted")
	}
	unknown := testFeature(t, FeatureUnknown, nil)
	if unknown.Value != nil || unknown.UnknownReason == "" {
		t.Fatal("unknown feature lost explicit unknown semantics")
	}
	factor, err := NewFactorDefinition(FactorDefinition{Name: "event_quality", Version: "v1", Rationale: "registered event-time evidence quality", InputFeatureIDs: []string{f.ID, unknown.ID}, Algorithm: "weighted-v1"})
	if err != nil {
		t.Fatal(err)
	}
	factor.InputFeatureIDs = append(factor.InputFeatureIDs, f.ID)
	if err := factor.Validate(); err == nil {
		t.Fatal("factor accepted duplicate feature input")
	}
}

func TestFeatureIdentityDetectsTampering(t *testing.T) {
	value := 1.0
	f := testFeature(t, FeatureKnown, &value)
	f.Name = "future_return"
	if err := f.Validate(); err == nil {
		t.Fatal("tampered feature was accepted")
	}
}
