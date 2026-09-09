package advancedquant

import (
	"fmt"
	"sort"
	"strings"
)

const PopulationContractV1 = "jax.phase12.analysis_population/v1"

type ExclusionReason string

const (
	ExclusionClassifierResultMissing ExclusionReason = "CLASSIFIER_RESULT_MISSING"
	ExclusionNonDirectionalNeutral   ExclusionReason = "NON_DIRECTIONAL_NEUTRAL"
	ExclusionClassifierAbstention    ExclusionReason = "CLASSIFIER_ABSTENTION"
	ExclusionEvidenceThreshold       ExclusionReason = "EVIDENCE_THRESHOLD_NOT_MET"
	ExclusionMarketUnavailable       ExclusionReason = "MARKET_DATA_UNAVAILABLE"
	ExclusionSessionIncomplete       ExclusionReason = "SESSION_WINDOW_INCOMPLETE"
	ExclusionBenchmarkUnavailable    ExclusionReason = "BENCHMARK_UNAVAILABLE"
	ExclusionSymbolUnavailable       ExclusionReason = "SYMBOL_MAPPING_UNAVAILABLE"
	ExclusionCorporateAction         ExclusionReason = "CORPORATE_ACTION_UNRESOLVED"
	ExclusionDuplicateEvent          ExclusionReason = "DUPLICATE_EVENT"
	ExclusionOther                   ExclusionReason = "OTHER_TYPED_REASON"
)

type AnalysisPopulation string

const (
	ClassifiedEventPopulation     AnalysisPopulation = "CLASSIFIED_EVENT_POPULATION"
	DirectionalEventPopulation    AnalysisPopulation = "DIRECTIONAL_EVENT_POPULATION"
	EvidenceConditionedPopulation AnalysisPopulation = "EVIDENCE_CONDITIONED_POPULATION"
	ReturnEligiblePopulation      AnalysisPopulation = "RETURN_ELIGIBLE_POPULATION"
	AnalysisPopulationFinal       AnalysisPopulation = "ANALYSIS_POPULATION"
)

// PopulationRecord describes one source event at every downstream population
// boundary. It prevents classifier abstention and market-data failure from
// collapsing into one generic skipped count.
type PopulationRecord struct {
	EventID             string            `json:"event_id"`
	IssuerID            string            `json:"issuer_id"`
	Partition           Partition         `json:"partition"`
	ClassifierState     string            `json:"classifier_state"`
	Classified          bool              `json:"classified"`
	Directional         bool              `json:"directional"`
	EvidenceConditioned bool              `json:"evidence_conditioned"`
	ReturnEligible      bool              `json:"return_eligible"`
	AnalysisIncluded    bool              `json:"analysis_included"`
	ExclusionReasons    []ExclusionReason `json:"exclusion_reasons,omitempty"`
}

func (r PopulationRecord) Validate() error {
	if strings.TrimSpace(r.EventID) == "" || strings.TrimSpace(r.IssuerID) == "" || (r.Partition != PartitionDevelopment && r.Partition != PartitionValidation && r.Partition != PartitionOOS) {
		return fmt.Errorf("population record identity is invalid")
	}
	if r.Directional && !r.Classified {
		return fmt.Errorf("directional population requires classification")
	}
	if r.EvidenceConditioned && !r.Directional {
		return fmt.Errorf("evidence-conditioned population requires direction")
	}
	if r.AnalysisIncluded && !r.ReturnEligible {
		return fmt.Errorf("analysis population requires return eligibility")
	}
	seen := map[ExclusionReason]struct{}{}
	for _, reason := range r.ExclusionReasons {
		if _, exists := seen[reason]; reason == "" || exists {
			return fmt.Errorf("population exclusion reason is empty, generic or duplicated")
		}
		seen[reason] = struct{}{}
	}
	return nil
}

type PopulationCounts struct {
	SourceEvents         int `json:"source_events"`
	ClassifiedEvents     int `json:"classified_events"`
	DirectionalEvents    int `json:"directional_events"`
	EvidenceConditioned  int `json:"evidence_conditioned_events"`
	ReturnEligibleEvents int `json:"return_eligible_events"`
	AnalysisEvents       int `json:"analysis_events"`
}

type PopulationRates struct {
	ClassifiedRate          float64 `json:"classified_rate"`
	DirectionalRate         float64 `json:"directional_rate"`
	EvidenceConditionedRate float64 `json:"evidence_conditioned_rate"`
	ReturnEligibleRate      float64 `json:"return_eligible_rate"`
	AnalysisRate            float64 `json:"analysis_rate"`
}

type PopulationSummary struct {
	ContractVersion             string                        `json:"contract_version"`
	Counts                      PopulationCounts              `json:"counts"`
	ClassifierDistribution      map[string]int                `json:"classifier_distribution"`
	ExclusionCounts             map[ExclusionReason]int       `json:"exclusion_counts"`
	RatesByPartition            map[Partition]PopulationRates `json:"rates_by_partition"`
	RatesByDirection            map[string]PopulationRates    `json:"rates_by_direction"`
	RatesByEvidenceState        map[string]PopulationRates    `json:"rates_by_evidence_state"`
	SelectionBiasReviewRequired bool                          `json:"selection_bias_review_required"`
	ComparisonPopulationNote    string                        `json:"comparison_population_note"`
}

func BuildPopulationSummary(records []PopulationRecord) (PopulationSummary, error) {
	summary := PopulationSummary{ContractVersion: PopulationContractV1, ClassifierDistribution: map[string]int{}, ExclusionCounts: map[ExclusionReason]int{}, RatesByPartition: map[Partition]PopulationRates{}, RatesByDirection: map[string]PopulationRates{}, RatesByEvidenceState: map[string]PopulationRates{}, ComparisonPopulationNote: "Evidence-conditioned performance is a selected subset of directional events; it is not a same-population comparison unless explicitly constructed."}
	partitionCounts := map[Partition]PopulationCounts{}
	directionCounts := map[string]PopulationCounts{}
	evidenceCounts := map[string]PopulationCounts{}
	seen := map[string]struct{}{}
	for _, record := range records {
		if err := record.Validate(); err != nil {
			return PopulationSummary{}, err
		}
		if _, ok := seen[record.EventID]; ok {
			return PopulationSummary{}, fmt.Errorf("duplicate population event %s", record.EventID)
		}
		seen[record.EventID] = struct{}{}
		incrementPopulation(&summary.Counts, record)
		partitionCount := partitionCounts[record.Partition]
		incrementPopulation(&partitionCount, record)
		partitionCounts[record.Partition] = partitionCount
		directionCount := directionCounts[record.ClassifierState]
		incrementPopulation(&directionCount, record)
		directionCounts[record.ClassifierState] = directionCount
		evidenceState := "NOT_EVIDENCE_CONDITIONED"
		if record.EvidenceConditioned {
			evidenceState = "EVIDENCE_CONDITIONED"
		}
		evidenceCount := evidenceCounts[evidenceState]
		incrementPopulation(&evidenceCount, record)
		evidenceCounts[evidenceState] = evidenceCount
		summary.ClassifierDistribution[record.ClassifierState]++
		for _, reason := range record.ExclusionReasons {
			summary.ExclusionCounts[reason]++
		}
		if record.EvidenceConditioned && record.Directional && len(record.ExclusionReasons) > 0 {
			summary.SelectionBiasReviewRequired = true
		}
	}
	if summary.Counts.EvidenceConditioned > summary.Counts.DirectionalEvents || summary.Counts.AnalysisEvents > summary.Counts.ReturnEligibleEvents {
		return PopulationSummary{}, fmt.Errorf("population counts violate nesting")
	}
	if summary.Counts.EvidenceConditioned != summary.Counts.DirectionalEvents {
		summary.SelectionBiasReviewRequired = true
	}
	for partition, counts := range partitionCounts {
		summary.RatesByPartition[partition] = ratesFromCounts(counts)
	}
	for direction, counts := range directionCounts {
		summary.RatesByDirection[direction] = ratesFromCounts(counts)
	}
	for state, counts := range evidenceCounts {
		summary.RatesByEvidenceState[state] = ratesFromCounts(counts)
	}
	return summary, nil
}

func ratesFromCounts(counts PopulationCounts) PopulationRates {
	if counts.SourceEvents == 0 {
		return PopulationRates{}
	}
	denominator := float64(counts.SourceEvents)
	return PopulationRates{ClassifiedRate: float64(counts.ClassifiedEvents) / denominator, DirectionalRate: float64(counts.DirectionalEvents) / denominator, EvidenceConditionedRate: float64(counts.EvidenceConditioned) / denominator, ReturnEligibleRate: float64(counts.ReturnEligibleEvents) / denominator, AnalysisRate: float64(counts.AnalysisEvents) / denominator}
}

func incrementPopulation(counts *PopulationCounts, record PopulationRecord) {
	counts.SourceEvents++
	if record.Classified {
		counts.ClassifiedEvents++
	}
	if record.Directional {
		counts.DirectionalEvents++
	}
	if record.EvidenceConditioned {
		counts.EvidenceConditioned++
	}
	if record.ReturnEligible {
		counts.ReturnEligibleEvents++
	}
	if record.AnalysisIncluded {
		counts.AnalysisEvents++
	}
}

func (s PopulationSummary) Validate() error {
	if s.ContractVersion != PopulationContractV1 || s.Counts.SourceEvents < 0 || s.Counts.ClassifiedEvents > s.Counts.SourceEvents || s.Counts.DirectionalEvents > s.Counts.ClassifiedEvents || s.Counts.EvidenceConditioned > s.Counts.DirectionalEvents || s.Counts.AnalysisEvents > s.Counts.ReturnEligibleEvents || strings.TrimSpace(s.ComparisonPopulationNote) == "" {
		return fmt.Errorf("population summary is invalid")
	}
	if !s.SelectionBiasReviewRequired && s.Counts.EvidenceConditioned != s.Counts.DirectionalEvents {
		return fmt.Errorf("subset selection must require bias review")
	}
	return nil
}

func SortedExclusionReasons(values map[ExclusionReason]int) []ExclusionReason {
	out := make([]ExclusionReason, 0, len(values))
	for reason := range values {
		out = append(out, reason)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
