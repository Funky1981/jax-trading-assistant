package advancedquant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type authorisedDatasetMetadata struct {
	DatasetID               string  `json:"DatasetID"`
	ContentManifestHash     string  `json:"ContentManifestHash"`
	EventCount              int     `json:"EventCount"`
	IssuerCount             int     `json:"IssuerCount"`
	MaxIssuerShare          float64 `json:"MaxIssuerShare"`
	FinalHoldoutCount       int     `json:"FinalHoldoutCount"`
	FinalHoldout            string  `json:"FinalHoldout"`
	AcceptanceTimestampPass bool    `json:"AcceptanceTimestampPass"`
	EventCoveragePass       bool    `json:"EventCoveragePass"`
	MarketBarValidationPass bool    `json:"MarketBarValidationPass"`
}

// TestPhase12ExitGate is a capability demonstration. It uses explicit fixture
// labels and returns; it is not a claim about HYP-EVENT-001A performance.
func TestPhase12ExitGate(t *testing.T) {
	metadataPath := filepath.Join("..", "..", "..", "data", "datasets", "hyp-event-001a", "dataset-2016-2025-sip-sec-v1", "normalized", "validation.json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("private Phase-12 dataset metadata unavailable in clean CI: %s", metadataPath)
		}
		t.Fatalf("authorised private dataset metadata unavailable: %v", err)
	}
	var metadata authorisedDatasetMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.DatasetID != "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1" || metadata.ContentManifestHash != "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d" || metadata.EventCount != 1059 || metadata.IssuerCount != 60 || metadata.MaxIssuerShare >= .05 || metadata.FinalHoldoutCount != 120 || metadata.FinalHoldout != "SEALED; no outcome performance calculated" || !metadata.AcceptanceTimestampPass || !metadata.EventCoveragePass || !metadata.MarketBarValidationPass {
		t.Fatalf("authorised dataset metadata does not match frozen gate: %+v", metadata)
	}

	protocol := phase12ExitProtocol(t)
	featureValue := .9
	feature, err := NewFeature(Feature{Name: "source_authority", Version: "v1", EventID: "evt_exit", DatasetID: protocol.DatasetID, SourceIdentities: []string{"raw_exit"}, AvailabilityAt: time.Date(2024, 1, 2, 15, 0, 0, 0, time.UTC), CalculatedAt: time.Date(2024, 1, 2, 15, 1, 0, 0, time.UTC), AlgorithmVersion: "fixture-event-time-v1", Status: FeatureKnown, Value: &featureValue})
	if err != nil {
		t.Fatal(err)
	}
	vector, err := NewFeatureVector(FeatureVector{EventID: "evt_exit", DatasetID: protocol.DatasetID, AsOf: feature.AvailabilityAt, Features: []Feature{feature}})
	if err != nil {
		t.Fatal(err)
	}
	if vector.ID == "" {
		t.Fatal("feature vector has no immutable identity")
	}

	observations := phase12ExitObservations()
	baselineValidation, err := ScoreDirectionOnly("exp_exit", observations, PartitionValidation, protocol)
	if err != nil {
		t.Fatal(err)
	}
	conditionedValidation, err := ScoreEvidenceConditioned("exp_exit", observations, PartitionValidation, protocol, .8)
	if err != nil {
		t.Fatal(err)
	}
	if conditionedValidation.MeanCostAdjustedReturn <= baselineValidation.MeanCostAdjustedReturn {
		t.Fatalf("fixture does not demonstrate baseline comparison: baseline=%+v conditioned=%+v", baselineValidation, conditionedValidation)
	}

	factor, err := NewFactorDefinition(FactorDefinition{Name: "event_quality", Version: "v1", Rationale: "registered evidence-quality conditioning", InputFeatureIDs: []string{feature.ID}, Algorithm: "threshold-v1"})
	if err != nil {
		t.Fatal(err)
	}
	trial, err := NewFactorTrial(FactorTrial{ExperimentID: "exp_exit", FactorID: factor.ID, Name: "quality-threshold", HorizonDays: 5, FeatureIDs: []string{feature.ID}, Parameters: map[string]string{"threshold": "0.8"}, CostModelID: "cost_phase11_v1"})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := SelectModel("direction-only", baselineValidation.MeanCostAdjustedReturn, []ModelCandidate{mustModel(t, conditionedValidation.MeanCostAdjustedReturn)}, .001, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if selection.Status != SelectionCandidateSelected {
		t.Fatalf("fixture candidate was not selected on validation: %+v", selection)
	}
	config, err := NewExperimentConfig(ExperimentConfig{ProtocolID: protocol.ID, HypothesisID: protocol.HypothesisID, Target: "5-day SPY-relative return", Baseline: "direction-only", Algorithm: "evidence-quality-threshold-v1", Parameters: map[string]string{"minimum_evidence_quality": ".8"}, Seed: 12})
	if err != nil {
		t.Fatal(err)
	}
	run := &FrozenOOSRun{}
	if err := run.FreezeWithAdmission(config, testAdmission(t, protocol, config)); err != nil {
		t.Fatal(err)
	}
	oos, err := run.ScoreOOS(observations, protocol)
	if err != nil {
		t.Fatal(err)
	}
	if oos.Partition != PartitionOOS || oos.ObservationCount == 0 {
		t.Fatalf("OOS was not evaluated: %+v", oos)
	}
	if _, err := run.ScoreOOS(observations, protocol); err == nil {
		t.Fatal("formal OOS was scored more than once")
	}

	falsifications := completeFalsifications()
	stability, err := EvaluateStability(trial, StabilityCriteria{MinimumObservationsPerPartition: 1, MaximumIssuerShare: .05, RequireCostAdjusted: true}, []MetricResult{mustMetric(t, observations, PartitionDevelopment, protocol), mustMetric(t, observations, PartitionValidation, protocol), mustMetric(t, observations, PartitionOOS, protocol)}, falsifications)
	if err != nil || stability.Status != StabilityStable {
		t.Fatalf("stability gate failed: %+v err=%v", stability, err)
	}
	registry := NewExperimentRegistry()
	record := testRecord(t)
	if err := registry.Register(record); err != nil {
		t.Fatal(err)
	}
	outcome, err := DecideResearchOutcome(record.ID, OutcomeCriteria{MinimumOOSObservations: oos.ObservationCount, BeatBaseline: true, CostAdjusted: true, PlaceboPassed: true, StableAcrossPeriods: true, Reproducible: true, SurvivorshipControlled: false})
	if err != nil || outcome.Status != OutcomeCandidate {
		t.Fatalf("unexpected research-only outcome: %+v err=%v", outcome, err)
	}
	policy := testDriftPolicy(t)
	drift, err := EvaluateDrift(mustModel(t, conditionedValidation.MeanCostAdjustedReturn).ID, policy, time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), .1, .1, oos.MeanCostAdjustedReturn, oos.ObservationCount, 0)
	if err != nil || drift.State != DriftValid {
		t.Fatalf("drift monitoring failed: %+v err=%v", drift, err)
	}
	promotion, err := EvaluatePromotion(PromotionEvidence{HypothesisID: protocol.HypothesisID, ExperimentID: record.ID, DatasetID: protocol.DatasetID, DatasetHash: protocol.DatasetManifestHash, BaselineBeaten: true, Reproducible: true, LeakageReviewPassed: true, CostSensitivityPassed: true, DriftMonitoringDefined: true, FailureBehaviourDefined: true, SurvivorshipControlled: false, FinalHoldoutSealed: true, ForwardPaperDays: 0, ForwardPaperOrders: 0})
	if err != nil || promotion.Status != PromotionClosed || promotion.RecommendationMutationAllowed {
		t.Fatalf("promotion boundary failed: %+v err=%v", promotion, err)
	}
	if _, err := ScoreDirectionOnly("exp_exit", observations, PartitionFinalHoldout, protocol); err == nil {
		t.Fatal("sealed holdout was scored")
	}
}

func phase12ExitProtocol(t *testing.T) ResearchProtocol {
	t.Helper()
	d := func(y, m, day int) time.Time { return time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC) }
	p, err := NewResearchProtocol(ResearchProtocol{HypothesisID: "HYP-EVENT-001A", DatasetID: "hyp-event-001a-sec-8k-alpaca-sip-2016-2025-v1", DatasetManifestHash: "db2b454793e50f28c34c9c2a7d91f798741936528752de7bd27cb52072a4864d", EventFamily: "FORM 8-K", EntryRule: "NEXT_REGULAR_SESSION_OPEN_AFTER_PUBLIC_AVAILABILITY", Benchmark: "SPY", PrimaryHorizonDays: 5, SecondaryHorizonDays: []int{1, 3}, PrimaryMetric: "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN", MetricVersion: "metric_v1", CostModelID: "cost_phase11_v1", FinalHoldoutSealed: true, Windows: []DateWindow{{PartitionDevelopment, d(2016, 1, 1), d(2021, 12, 31)}, {PartitionValidation, d(2022, 1, 1), d(2023, 12, 31)}, {PartitionOOS, d(2024, 1, 1), d(2024, 12, 31)}, {PartitionFinalHoldout, d(2025, 1, 1), d(2025, 12, 31)}}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func phase12ExitObservations() []ResearchObservation {
	base := time.Date(2020, 1, 2, 15, 0, 0, 0, time.UTC)
	makeObservation := func(id string, partition Partition, eventAt time.Time, quality, relative, cost float64, direction int) ResearchObservation {
		return ResearchObservation{EventID: id, IssuerID: id + "_issuer", Partition: partition, EventAt: eventAt, EntryAt: eventAt.Add(24 * time.Hour), ExitAt: eventAt.Add(6 * 24 * time.Hour), PredictedDirection: direction, EvidenceQuality: quality, BenchmarkRelativeReturn: relative, CostAdjustedReturn: cost}
	}
	return []ResearchObservation{
		makeObservation("dev1", PartitionDevelopment, base, .9, .02, .01, 1), makeObservation("dev2", PartitionDevelopment, base.AddDate(1, 0, 0), .2, -.01, -.02, 1),
		makeObservation("val1", PartitionValidation, time.Date(2022, 2, 1, 15, 0, 0, 0, time.UTC), .9, .04, .03, 1), makeObservation("val2", PartitionValidation, time.Date(2023, 2, 1, 15, 0, 0, 0, time.UTC), .2, -.02, -.03, 1),
		makeObservation("oos1", PartitionOOS, time.Date(2024, 2, 1, 15, 0, 0, 0, time.UTC), .9, .03, .02, 1), makeObservation("oos2", PartitionOOS, time.Date(2024, 6, 1, 15, 0, 0, 0, time.UTC), .2, -.01, -.02, 1),
	}
}

func mustModel(t *testing.T, validationMetric float64) ModelCandidate {
	t.Helper()
	model, err := NewModelCandidate(ModelCandidate{Name: "simple_quality", Algorithm: "threshold-v1", ValidationMetric: validationMetric, BaselineMetric: 0, CostAdjusted: true, SelectionPartition: PartitionValidation, ComplexityRank: 1})
	if err != nil {
		t.Fatal(err)
	}
	return model
}
func mustMetric(t *testing.T, observations []ResearchObservation, partition Partition, protocol ResearchProtocol) MetricResult {
	t.Helper()
	result, err := ScoreEvidenceConditioned("exp_exit", observations, partition, protocol, .8)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
