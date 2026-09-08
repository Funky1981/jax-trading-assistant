package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ProtocolContractV1   = "jax.phase12.research_protocol/v1"
	ExperimentContractV1 = "jax.phase12.experiment/v1"
	MetricContractV1     = "jax.phase12.metric/v1"
)

type Partition string

const (
	PartitionDevelopment  Partition = "DEVELOPMENT"
	PartitionValidation   Partition = "VALIDATION"
	PartitionOOS          Partition = "OOS"
	PartitionFinalHoldout Partition = "FINAL_HOLDOUT"
)

type DateWindow struct {
	Name  Partition `json:"name"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (w DateWindow) Validate() error {
	if w.Name != PartitionDevelopment && w.Name != PartitionValidation && w.Name != PartitionOOS && w.Name != PartitionFinalHoldout {
		return fmt.Errorf("unsupported partition %q", w.Name)
	}
	if w.Start.IsZero() || w.End.IsZero() || w.Start.Location() != time.UTC || w.End.Location() != time.UTC || w.End.Before(w.Start) {
		return fmt.Errorf("partition %s requires an ordered UTC date window", w.Name)
	}
	return nil
}

type ResearchProtocol struct {
	ContractVersion      string       `json:"contract_version"`
	ID                   string       `json:"id"`
	HypothesisID         string       `json:"hypothesis_id"`
	DatasetID            string       `json:"dataset_id"`
	DatasetManifestHash  string       `json:"dataset_manifest_hash"`
	EventFamily          string       `json:"event_family"`
	EntryRule            string       `json:"entry_rule"`
	Benchmark            string       `json:"benchmark"`
	PrimaryHorizonDays   int          `json:"primary_horizon_days"`
	SecondaryHorizonDays []int        `json:"secondary_horizon_days"`
	PrimaryMetric        string       `json:"primary_metric"`
	MetricVersion        string       `json:"metric_version"`
	CostModelID          string       `json:"cost_model_id"`
	Windows              []DateWindow `json:"windows"`
	FinalHoldoutSealed   bool         `json:"final_holdout_sealed"`
}

func NewResearchProtocol(protocol ResearchProtocol) (ResearchProtocol, error) {
	protocol.ContractVersion = ProtocolContractV1
	protocol.ID = protocolID(protocol)
	if err := protocol.Validate(); err != nil {
		return ResearchProtocol{}, err
	}
	return protocol, nil
}

func (p ResearchProtocol) Validate() error {
	if p.ContractVersion != ProtocolContractV1 || !validHashID(p.ID, "protocol_") || strings.TrimSpace(p.HypothesisID) == "" || strings.TrimSpace(p.DatasetID) == "" || !validSHA256(p.DatasetManifestHash) || p.EventFamily != "FORM 8-K" || strings.TrimSpace(p.EntryRule) == "" || p.Benchmark != "SPY" || p.PrimaryHorizonDays != 5 || len(p.SecondaryHorizonDays) != 2 || p.PrimaryMetric != "5_DAY_BENCHMARK_RELATIVE_DIRECTIONAL_RETURN" || strings.TrimSpace(p.MetricVersion) == "" || strings.TrimSpace(p.CostModelID) == "" || len(p.Windows) != 4 || !p.FinalHoldoutSealed {
		return fmt.Errorf("research protocol does not match the frozen HYP-EVENT-001A contract")
	}
	seen := map[Partition]bool{}
	for _, window := range p.Windows {
		if err := window.Validate(); err != nil {
			return err
		}
		if seen[window.Name] {
			return fmt.Errorf("duplicate research partition %s", window.Name)
		}
		seen[window.Name] = true
	}
	for _, required := range []Partition{PartitionDevelopment, PartitionValidation, PartitionOOS, PartitionFinalHoldout} {
		if !seen[required] {
			return fmt.Errorf("required research partition %s is missing", required)
		}
	}
	ordered := append([]DateWindow(nil), p.Windows...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start.Before(ordered[j].Start) })
	for i := 1; i < len(ordered); i++ {
		if !ordered[i].Start.After(ordered[i-1].End) {
			return fmt.Errorf("research partitions overlap or are not strictly ordered")
		}
	}
	for _, horizon := range p.SecondaryHorizonDays {
		if horizon <= 0 || horizon == p.PrimaryHorizonDays {
			return fmt.Errorf("secondary horizons must be positive and distinct")
		}
	}
	if p.ID != protocolID(p) {
		return fmt.Errorf("research protocol ID does not match immutable contents")
	}
	return nil
}

// ResearchObservation is an already-labelled, event-time-safe observation.
// The framework accepts labels but never creates a label from a return.
type ResearchObservation struct {
	EventID                 string    `json:"event_id"`
	IssuerID                string    `json:"issuer_id"`
	Partition               Partition `json:"partition"`
	EventAt                 time.Time `json:"event_at"`
	EntryAt                 time.Time `json:"entry_at"`
	ExitAt                  time.Time `json:"exit_at"`
	PredictedDirection      int       `json:"predicted_direction"`
	EvidenceQuality         float64   `json:"evidence_quality"`
	BenchmarkRelativeReturn float64   `json:"benchmark_relative_return"`
	CostAdjustedReturn      float64   `json:"cost_adjusted_return"`
}

func (o ResearchObservation) Validate(protocol ResearchProtocol) error {
	if strings.TrimSpace(o.EventID) == "" || strings.TrimSpace(o.IssuerID) == "" || o.Partition == PartitionFinalHoldout || (o.Partition != PartitionDevelopment && o.Partition != PartitionValidation && o.Partition != PartitionOOS) || o.EventAt.IsZero() || o.EntryAt.IsZero() || o.ExitAt.IsZero() || o.EventAt.Location() != time.UTC || o.EntryAt.Location() != time.UTC || o.ExitAt.Location() != time.UTC || !o.EntryAt.After(o.EventAt) || !o.ExitAt.After(o.EntryAt) || (o.PredictedDirection != -1 && o.PredictedDirection != 1) || !finite(o.EvidenceQuality) || o.EvidenceQuality < 0 || o.EvidenceQuality > 1 || !finite(o.BenchmarkRelativeReturn) || !finite(o.CostAdjustedReturn) {
		return fmt.Errorf("research observation is invalid, non-UTC, future-leaking or unlabeled")
	}
	if !observationInWindow(o, protocol) {
		return fmt.Errorf("observation does not belong to its declared protocol partition")
	}
	for _, window := range protocol.Windows {
		if window.Name == o.Partition && dateOnly(o.ExitAt).After(dateOnly(window.End)) {
			return fmt.Errorf("observation outcome crosses its partition boundary")
		}
	}
	return nil
}

type ExperimentConfig struct {
	ContractVersion string            `json:"contract_version"`
	ID              string            `json:"id"`
	ProtocolID      string            `json:"protocol_id"`
	HypothesisID    string            `json:"hypothesis_id"`
	FeatureIDs      []string          `json:"feature_ids"`
	Target          string            `json:"target"`
	Baseline        string            `json:"baseline"`
	Algorithm       string            `json:"algorithm"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	Seed            int64             `json:"seed"`
}

func NewExperimentConfig(config ExperimentConfig) (ExperimentConfig, error) {
	config.ContractVersion = ExperimentContractV1
	config.ID = experimentID(config)
	if err := config.Validate(); err != nil {
		return ExperimentConfig{}, err
	}
	return config, nil
}

func (c ExperimentConfig) Validate() error {
	if c.ContractVersion != ExperimentContractV1 || !validHashID(c.ID, "exp_") || strings.TrimSpace(c.ProtocolID) == "" || strings.TrimSpace(c.HypothesisID) == "" || strings.TrimSpace(c.Target) == "" || strings.TrimSpace(c.Baseline) == "" || strings.TrimSpace(c.Algorithm) == "" {
		return fmt.Errorf("experiment configuration identity and frozen fields are required")
	}
	for key, value := range c.Parameters {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return fmt.Errorf("experiment parameters must be non-empty")
		}
		if key == "minimum_evidence_quality" {
			threshold, err := strconv.ParseFloat(value, 64)
			if err != nil || !finite(threshold) || threshold < 0 || threshold > 1 {
				return fmt.Errorf("minimum evidence quality must be a finite number within [0,1]")
			}
		}
	}
	if c.ID != experimentID(c) {
		return fmt.Errorf("experiment identity does not match immutable configuration")
	}
	return nil
}

type MetricResult struct {
	ContractVersion        string    `json:"contract_version"`
	ExperimentID           string    `json:"experiment_id"`
	Partition              Partition `json:"partition"`
	ObservationCount       int       `json:"observation_count"`
	MeanSignedReturn       float64   `json:"mean_signed_return"`
	MeanCostAdjustedReturn float64   `json:"mean_cost_adjusted_return"`
	HitRate                float64   `json:"hit_rate"`
}

// ScoreDirectionOnly is deliberately small and deterministic. Evidence
// conditioning is implemented by ScoreEvidenceConditioned with a frozen
// threshold; neither function accesses final-holdout observations.
func ScoreDirectionOnly(experimentID string, observations []ResearchObservation, partition Partition, protocol ResearchProtocol) (MetricResult, error) {
	return score(experimentID, observations, partition, protocol, func(_ ResearchObservation) bool { return true })
}

func ScoreEvidenceConditioned(experimentID string, observations []ResearchObservation, partition Partition, protocol ResearchProtocol, minimumQuality float64) (MetricResult, error) {
	if !finite(minimumQuality) || minimumQuality < 0 || minimumQuality > 1 {
		return MetricResult{}, fmt.Errorf("evidence threshold must be finite and within [0,1]")
	}
	return score(experimentID, observations, partition, protocol, func(o ResearchObservation) bool { return o.EvidenceQuality >= minimumQuality })
}

func score(experimentID string, observations []ResearchObservation, partition Partition, protocol ResearchProtocol, include func(ResearchObservation) bool) (MetricResult, error) {
	if err := protocol.Validate(); err != nil {
		return MetricResult{}, err
	}
	if strings.TrimSpace(experimentID) == "" || partition == PartitionFinalHoldout {
		return MetricResult{}, fmt.Errorf("score requires an experiment and non-holdout partition")
	}
	result := MetricResult{ContractVersion: MetricContractV1, ExperimentID: experimentID, Partition: partition}
	for _, observation := range observations {
		if observation.Partition != partition {
			continue
		}
		if err := observation.Validate(protocol); err != nil {
			return MetricResult{}, err
		}
		if !include(observation) {
			continue
		}
		result.ObservationCount++
		signed := float64(observation.PredictedDirection) * observation.BenchmarkRelativeReturn
		result.MeanSignedReturn += signed
		result.MeanCostAdjustedReturn += float64(observation.PredictedDirection) * observation.CostAdjustedReturn
		if signed > 0 {
			result.HitRate++
		}
	}
	if result.ObservationCount == 0 {
		return result, fmt.Errorf("no eligible observations for %s", partition)
	}
	denominator := float64(result.ObservationCount)
	result.MeanSignedReturn /= denominator
	result.MeanCostAdjustedReturn /= denominator
	result.HitRate /= denominator
	return result, nil
}

// FrozenOOSRun enforces one-way OOS use: once frozen, no feature/configuration
// change is accepted, and ScoreOOS can execute only once.
type FrozenOOSRun struct {
	mu        sync.Mutex
	Config    ExperimentConfig
	Frozen    bool
	OOSScored bool
	Result    MetricResult
}

func (r *FrozenOOSRun) Freeze(config ExperimentConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Frozen {
		return fmt.Errorf("OOS configuration is already frozen")
	}
	r.Config = config
	r.Frozen = true
	return nil
}

func (r *FrozenOOSRun) ScoreOOS(observations []ResearchObservation, protocol ResearchProtocol) (MetricResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.Frozen {
		return MetricResult{}, fmt.Errorf("OOS configuration must be frozen before scoring")
	}
	if r.OOSScored {
		return MetricResult{}, fmt.Errorf("formal OOS may be scored only once")
	}
	if err := protocol.Validate(); err != nil || protocol.ID != r.Config.ProtocolID {
		return MetricResult{}, fmt.Errorf("OOS protocol does not match the frozen experiment configuration")
	}
	result, err := ScoreEvidenceConditioned(r.Config.ID, observations, PartitionOOS, protocol, qualityThreshold(r.Config))
	if err != nil {
		return MetricResult{}, err
	}
	r.Result = result
	r.OOSScored = true
	return result, nil
}

func qualityThreshold(config ExperimentConfig) float64 {
	if value, ok := config.Parameters["minimum_evidence_quality"]; ok {
		threshold, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return -1
		}
		return threshold
	}
	return 0
}

func observationInWindow(o ResearchObservation, p ResearchProtocol) bool {
	for _, window := range p.Windows {
		if window.Name == o.Partition {
			return !o.EventAt.Before(window.Start) && !o.EventAt.After(window.End)
		}
	}
	return false
}

func dateOnly(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func protocolID(p ResearchProtocol) string {
	p.ID = ""
	b, _ := json.Marshal(p)
	d := sha256.Sum256(b)
	return "protocol_" + hex.EncodeToString(d[:])
}
func experimentID(c ExperimentConfig) string {
	c.ID = ""
	b, _ := json.Marshal(c)
	d := sha256.Sum256(b)
	return "exp_" + hex.EncodeToString(d[:])
}
func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}
