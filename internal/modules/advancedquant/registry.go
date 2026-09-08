package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

const (
	RegistryContractV1 = "jax.phase12.experiment_registry/v1"
	OutcomeContractV1  = "jax.phase12.research_outcome/v1"
)

type ExperimentRecord struct {
	ContractVersion string            `json:"contract_version"`
	ID              string            `json:"id"`
	HypothesisID    string            `json:"hypothesis_id"`
	DatasetID       string            `json:"dataset_id"`
	DatasetHash     string            `json:"dataset_hash"`
	FeatureIDs      []string          `json:"feature_ids"`
	Target          string            `json:"target"`
	Horizons        []int             `json:"horizons"`
	Partitions      []Partition       `json:"partitions"`
	Baseline        string            `json:"baseline"`
	Algorithm       string            `json:"algorithm"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	Seed            int64             `json:"seed"`
	SoftwareVersion string            `json:"software_version"`
	CostModelID     string            `json:"cost_model_id"`
}

func NewExperimentRecord(record ExperimentRecord) (ExperimentRecord, error) {
	record.ContractVersion = RegistryContractV1
	record.ID = registryExperimentID(record)
	if err := record.Validate(); err != nil {
		return ExperimentRecord{}, err
	}
	return record, nil
}

func (r ExperimentRecord) Validate() error {
	if r.ContractVersion != RegistryContractV1 || !validHashID(r.ID, "regexp_") || strings.TrimSpace(r.HypothesisID) == "" || strings.TrimSpace(r.DatasetID) == "" || !validSHA256(r.DatasetHash) || len(r.FeatureIDs) == 0 || strings.TrimSpace(r.Target) == "" || len(r.Horizons) == 0 || len(r.Partitions) != 3 || strings.TrimSpace(r.Baseline) == "" || strings.TrimSpace(r.Algorithm) == "" || strings.TrimSpace(r.SoftwareVersion) == "" || strings.TrimSpace(r.CostModelID) == "" {
		return fmt.Errorf("experiment registry record is incomplete")
	}
	seen := map[string]bool{}
	for _, id := range r.FeatureIDs {
		if !validHashID(id, "feat_") || seen[id] {
			return fmt.Errorf("experiment feature identities are invalid or duplicated")
		}
		seen[id] = true
	}
	seen = map[string]bool{}
	for _, partition := range r.Partitions {
		if partition == PartitionFinalHoldout || (partition != PartitionDevelopment && partition != PartitionValidation && partition != PartitionOOS) || seen[string(partition)] {
			return fmt.Errorf("registry cannot include final holdout or duplicate partitions")
		}
		seen[string(partition)] = true
	}
	if r.ID != registryExperimentID(r) {
		return fmt.Errorf("registry experiment identity does not match contents")
	}
	return nil
}

type ExperimentRegistry struct {
	mu      sync.RWMutex
	records map[string]ExperimentRecord
}

func NewExperimentRegistry() *ExperimentRegistry {
	return &ExperimentRegistry{records: map[string]ExperimentRecord{}}
}

func (r *ExperimentRegistry) Register(record ExperimentRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.records[record.ID]; ok && !sameJSON(existing, record) {
		return fmt.Errorf("experiment registry identity collision")
	}
	r.records[record.ID] = cloneExperimentRecord(record)
	return nil
}

func (r *ExperimentRegistry) List() []ExperimentRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ExperimentRecord, 0, len(r.records))
	for _, record := range r.records {
		out = append(out, cloneExperimentRecord(record))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type ResearchOutcomeStatus string

const (
	OutcomeRejected     ResearchOutcomeStatus = "HYPOTHESIS_REJECTED"
	OutcomeNoRobustEdge ResearchOutcomeStatus = "NO_ROBUST_EDGE"
	OutcomeInsufficient ResearchOutcomeStatus = "INSUFFICIENT_DATA"
	OutcomeUnstable     ResearchOutcomeStatus = "UNSTABLE"
	OutcomeCostRemoved  ResearchOutcomeStatus = "COSTS_REMOVE_EFFECT"
	OutcomeCandidate    ResearchOutcomeStatus = "PROMISING_RESEARCH_CANDIDATE_NOT_PROMOTED"
)

type OutcomeCriteria struct {
	MinimumOOSObservations int  `json:"minimum_oos_observations"`
	BeatBaseline           bool `json:"beat_baseline"`
	CostAdjusted           bool `json:"cost_adjusted"`
	PlaceboPassed          bool `json:"placebo_passed"`
	StableAcrossPeriods    bool `json:"stable_across_periods"`
	Reproducible           bool `json:"reproducible"`
	SurvivorshipControlled bool `json:"survivorship_controlled"`
	FinalHoldoutUsed       bool `json:"final_holdout_used"`
}

type ResearchOutcome struct {
	ContractVersion string                `json:"contract_version"`
	ID              string                `json:"id"`
	ExperimentID    string                `json:"experiment_id"`
	Status          ResearchOutcomeStatus `json:"status"`
	Criteria        OutcomeCriteria       `json:"criteria"`
	Reason          string                `json:"reason"`
}

func DecideResearchOutcome(experimentID string, criteria OutcomeCriteria) (ResearchOutcome, error) {
	if strings.TrimSpace(experimentID) == "" || criteria.MinimumOOSObservations < 0 || criteria.FinalHoldoutUsed {
		return ResearchOutcome{}, fmt.Errorf("outcome requires experiment and valid sample floor without holdout use")
	}
	outcome := ResearchOutcome{ContractVersion: OutcomeContractV1, ExperimentID: experimentID, Criteria: criteria, Status: OutcomeCandidate, Reason: "candidate remains research-only; promotion is separately gated"}
	switch {
	case criteria.FinalHoldoutUsed:
		outcome.Status, outcome.Reason = OutcomeRejected, "final holdout was used; experiment is contaminated"
	case criteria.MinimumOOSObservations == 0:
		outcome.Status, outcome.Reason = OutcomeInsufficient, "no formal OOS observations"
	case !criteria.Reproducible:
		outcome.Status, outcome.Reason = OutcomeRejected, "result is not reproducible"
	case !criteria.CostAdjusted:
		outcome.Status, outcome.Reason = OutcomeCostRemoved, "cost-adjusted result is unavailable"
	case !criteria.BeatBaseline:
		outcome.Status, outcome.Reason = OutcomeNoRobustEdge, "candidate did not beat the registered baseline"
	case !criteria.PlaceboPassed || !criteria.StableAcrossPeriods:
		outcome.Status, outcome.Reason = OutcomeUnstable, "placebo or cross-period stability requirement failed"
	case !criteria.SurvivorshipControlled:
		outcome.Status, outcome.Reason = OutcomeCandidate, "result is promising but survivorship limitation closes promotion"
	}
	outcome.ID = outcomeID(outcome)
	return outcome, nil
}

func (o ResearchOutcome) Validate() error {
	if o.ContractVersion != OutcomeContractV1 || !validHashID(o.ID, "outcome12_") || strings.TrimSpace(o.ExperimentID) == "" || strings.TrimSpace(o.Reason) == "" || o.Criteria.FinalHoldoutUsed {
		return fmt.Errorf("research outcome is invalid or uses sealed holdout")
	}
	if o.Status != OutcomeRejected && o.Status != OutcomeNoRobustEdge && o.Status != OutcomeInsufficient && o.Status != OutcomeUnstable && o.Status != OutcomeCostRemoved && o.Status != OutcomeCandidate {
		return fmt.Errorf("unsupported research outcome status")
	}
	if o.ID != outcomeID(o) {
		return fmt.Errorf("research outcome identity does not match result")
	}
	return nil
}

func registryExperimentID(r ExperimentRecord) string {
	r.ID = ""
	b, _ := json.Marshal(r)
	d := sha256.Sum256(b)
	return "regexp_" + hex.EncodeToString(d[:])
}
func outcomeID(o ResearchOutcome) string {
	o.ID = ""
	b, _ := json.Marshal(o)
	d := sha256.Sum256(b)
	return "outcome12_" + hex.EncodeToString(d[:])
}

func cloneExperimentRecord(record ExperimentRecord) ExperimentRecord {
	clone := record
	clone.FeatureIDs = append([]string(nil), record.FeatureIDs...)
	clone.Horizons = append([]int(nil), record.Horizons...)
	clone.Partitions = append([]Partition(nil), record.Partitions...)
	if record.Parameters != nil {
		clone.Parameters = make(map[string]string, len(record.Parameters))
		for key, value := range record.Parameters {
			clone.Parameters[key] = value
		}
	}
	return clone
}
