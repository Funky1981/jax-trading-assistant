package worldmonitorintelligence

import (
	"fmt"
	"sort"
	"time"
)

type IntelligenceResult struct {
	ContractVersion       string
	Cluster               EventCluster
	Entities              EntityExtraction
	Confidence            ConfidenceAssessment
	Velocity              VelocitySignal
	PlausibleInstruments  []InstrumentLink
	MarketReactions       []MarketReaction
	AdapterEvaluations    []AdapterEvaluation
	Unknowns              []string
	TradeCandidateCreated bool
}

type PipelineInput struct {
	Observations   []Observation
	History        []time.Time
	MarketPoints   []MarketPoint
	Baselines      map[string]float64
	Now            time.Time
	VelocityWindow time.Duration
	BaselineWindow time.Duration
	ReactionWindow time.Duration
	AdapterSpecs   []AdapterSpec
}

// Analyze assembles the Phase-04 evidence path. It deliberately has no return
// type or port for candidate creation: the result is an intelligence read
// model with explicit uncertainty and a permanently false candidate flag.
func Analyze(input PipelineInput) ([]IntelligenceResult, error) {
	if input.Now.IsZero() || input.VelocityWindow <= 0 || input.BaselineWindow <= 0 || input.ReactionWindow <= 0 {
		return nil, fmt.Errorf("analysis reference time and positive windows are required")
	}
	clusters, err := BuildClusters(input.Observations)
	if err != nil {
		return nil, err
	}
	adapters, err := EvaluateEvidenceAdapters(input.AdapterSpecs)
	if err != nil {
		return nil, err
	}
	results := make([]IntelligenceResult, 0, len(clusters))
	for _, cluster := range clusters {
		entities := ExtractEntities(cluster)
		confidence, err := AssessConfidence(cluster)
		if err != nil {
			return nil, err
		}
		velocity, err := AssessVelocity(cluster, input.History, input.Now, input.VelocityWindow, input.BaselineWindow)
		if err != nil {
			return nil, err
		}
		reactions, err := CorrelateMarketReaction(cluster, input.MarketPoints, input.Baselines, input.ReactionWindow)
		if err != nil {
			return nil, err
		}
		unknowns := append([]string{}, cluster.Unknowns...)
		unknowns = append(unknowns, entities.Unknowns...)
		unknowns = append(unknowns, confidence.Unknowns...)
		unknowns = append(unknowns, velocity.Unknowns...)
		for _, reaction := range reactions {
			unknowns = append(unknowns, reaction.Unknowns...)
		}
		for _, adapter := range adapters {
			unknowns = append(unknowns, adapter.Unknowns...)
		}
		sort.Strings(unknowns)
		results = append(results, IntelligenceResult{ContractVersion: IntelligenceContractVersion, Cluster: cluster, Entities: entities, Confidence: confidence, Velocity: velocity, PlausibleInstruments: LinkPlausibleInstruments(cluster, entities), MarketReactions: reactions, AdapterEvaluations: adapters, Unknowns: uniqueStrings(unknowns), TradeCandidateCreated: false})
	}
	sort.Slice(results, func(left, right int) bool { return results[left].Cluster.ID < results[right].Cluster.ID })
	return results, nil
}
