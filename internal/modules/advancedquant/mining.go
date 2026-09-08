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
	TrialContractV1         = "jax.phase12.factor_trial/v1"
	FalsificationContractV1 = "jax.phase12.falsification/v1"
	StabilityContractV1     = "jax.phase12.stability/v1"
)

var requiredFalsifications = []string{
	"SHUFFLED_DIRECTION_LABELS", "CONSTRAINED_SHUFFLED_EVENT_DATES", "PLACEBO_NON_EVENT_DATES",
	"EVIDENCE_QUALITY_PERMUTATION", "STRONGEST_SOURCE_REMOVAL", "ISSUER_CLUSTER_SENSITIVITY",
	"EVENT_DEDUP_CLUSTER_SENSITIVITY", "LIQUIDITY_FILTER_SENSITIVITY", "TRANSACTION_COST_STRESS",
	"REGIME_SPLIT", "TOP_ISSUER_EXCLUSION",
}

type FactorTrial struct {
	ContractVersion string            `json:"contract_version"`
	ID              string            `json:"id"`
	ExperimentID    string            `json:"experiment_id"`
	FactorID        string            `json:"factor_id"`
	Name            string            `json:"name"`
	HorizonDays     int               `json:"horizon_days"`
	FeatureIDs      []string          `json:"feature_ids"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	CostModelID     string            `json:"cost_model_id"`
}

func NewFactorTrial(trial FactorTrial) (FactorTrial, error) {
	trial.ContractVersion = TrialContractV1
	trial.ID = trialID(trial)
	if err := trial.Validate(); err != nil {
		return FactorTrial{}, err
	}
	return trial, nil
}

func (t FactorTrial) Validate() error {
	if t.ContractVersion != TrialContractV1 || !validHashID(t.ID, "trial_") || strings.TrimSpace(t.ExperimentID) == "" || !validHashID(t.FactorID, "factor_") || strings.TrimSpace(t.Name) == "" || t.HorizonDays <= 0 || len(t.FeatureIDs) == 0 || strings.TrimSpace(t.CostModelID) == "" {
		return fmt.Errorf("factor trial has incomplete frozen identity/configuration")
	}
	seen := map[string]struct{}{}
	for _, id := range t.FeatureIDs {
		if !validHashID(id, "feat_") {
			return fmt.Errorf("factor trial contains invalid feature identity")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("factor trial feature identities must be unique")
		}
		seen[id] = struct{}{}
	}
	for key, value := range t.Parameters {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			return fmt.Errorf("factor trial parameters must be non-empty")
		}
	}
	if t.ID != trialID(t) {
		return fmt.Errorf("factor trial identity does not match configuration")
	}
	return nil
}

type TrialLedger struct {
	mu     sync.RWMutex
	trials map[string]FactorTrial
}

func NewTrialLedger() *TrialLedger { return &TrialLedger{trials: map[string]FactorTrial{}} }

func (l *TrialLedger) Add(trial FactorTrial) error {
	if err := trial.Validate(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.trials[trial.ID]; ok && !sameJSON(existing, trial) {
		return fmt.Errorf("factor trial identity collision")
	}
	l.trials[trial.ID] = cloneTrial(trial)
	return nil
}

func (l *TrialLedger) List() []FactorTrial {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]FactorTrial, 0, len(l.trials))
	for _, trial := range l.trials {
		out = append(out, cloneTrial(trial))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type FalsificationResult struct {
	ContractVersion string `json:"contract_version"`
	Name            string `json:"name"`
	Comparable      bool   `json:"comparable"`
	Passed          bool   `json:"passed"`
	Reason          string `json:"reason"`
}

func (f FalsificationResult) Validate() error {
	if f.ContractVersion != FalsificationContractV1 || strings.TrimSpace(f.Name) == "" || strings.TrimSpace(f.Reason) == "" {
		return fmt.Errorf("falsification result requires name, contract and reason")
	}
	for _, required := range requiredFalsifications {
		if required == f.Name {
			return nil
		}
	}
	return fmt.Errorf("unknown falsification %q", f.Name)
}

func ValidateFalsificationSuite(results []FalsificationResult) error {
	seen := map[string]bool{}
	for _, result := range results {
		if err := result.Validate(); err != nil {
			return err
		}
		if seen[result.Name] {
			return fmt.Errorf("duplicate falsification %q", result.Name)
		}
		seen[result.Name] = true
	}
	for _, required := range requiredFalsifications {
		if !seen[required] {
			return fmt.Errorf("missing required falsification %q", required)
		}
	}
	return nil
}

type StabilityStatus string

const (
	StabilityStable       StabilityStatus = "STABLE"
	StabilityUnstable     StabilityStatus = "UNSTABLE"
	StabilityInsufficient StabilityStatus = "INSUFFICIENT_DATA"
)

type StabilityCriteria struct {
	MinimumObservationsPerPartition int     `json:"minimum_observations_per_partition"`
	MaximumIssuerShare              float64 `json:"maximum_issuer_share"`
	RequireCostAdjusted             bool    `json:"require_cost_adjusted"`
}

type StabilityReport struct {
	ContractVersion  string                `json:"contract_version"`
	TrialID          string                `json:"trial_id"`
	Status           StabilityStatus       `json:"status"`
	Criteria         StabilityCriteria     `json:"criteria"`
	PartitionResults []MetricResult        `json:"partition_results"`
	Falsifications   []FalsificationResult `json:"falsifications"`
	Reasons          []string              `json:"reasons"`
}

func EvaluateStability(trial FactorTrial, criteria StabilityCriteria, metrics []MetricResult, falsifications []FalsificationResult) (StabilityReport, error) {
	if err := trial.Validate(); err != nil {
		return StabilityReport{}, err
	}
	if criteria.MinimumObservationsPerPartition <= 0 || !finite(criteria.MaximumIssuerShare) || criteria.MaximumIssuerShare <= 0 || criteria.MaximumIssuerShare > 1 {
		return StabilityReport{}, fmt.Errorf("stability criteria are invalid")
	}
	if err := ValidateFalsificationSuite(falsifications); err != nil {
		return StabilityReport{}, err
	}
	report := StabilityReport{ContractVersion: StabilityContractV1, TrialID: trial.ID, Criteria: criteria, PartitionResults: append([]MetricResult(nil), metrics...), Falsifications: append([]FalsificationResult(nil), falsifications...), Status: StabilityStable}
	seen := map[Partition]bool{}
	for _, metric := range metrics {
		if metric.Partition == PartitionFinalHoldout {
			return StabilityReport{}, fmt.Errorf("final holdout cannot be used for factor stability")
		}
		if metric.ObservationCount < criteria.MinimumObservationsPerPartition {
			report.Status = StabilityInsufficient
			report.Reasons = append(report.Reasons, fmt.Sprintf("%s observation floor not met", metric.Partition))
		}
		seen[metric.Partition] = true
		if criteria.RequireCostAdjusted && !finite(metric.MeanCostAdjustedReturn) {
			report.Status = StabilityInsufficient
			report.Reasons = append(report.Reasons, "cost-adjusted metric missing")
		}
	}
	for _, partition := range []Partition{PartitionDevelopment, PartitionValidation, PartitionOOS} {
		if !seen[partition] {
			report.Status = StabilityInsufficient
			report.Reasons = append(report.Reasons, fmt.Sprintf("%s result missing", partition))
		}
	}
	for _, result := range falsifications {
		if !result.Passed {
			report.Status = StabilityUnstable
			report.Reasons = append(report.Reasons, result.Name+" failed")
		}
	}
	return report, nil
}

func sameJSON(left, right any) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
func trialID(t FactorTrial) string {
	t.ID = ""
	b, _ := json.Marshal(t)
	d := sha256.Sum256(b)
	return "trial_" + hex.EncodeToString(d[:])
}

func cloneTrial(trial FactorTrial) FactorTrial {
	clone := trial
	clone.FeatureIDs = append([]string(nil), trial.FeatureIDs...)
	if trial.Parameters != nil {
		clone.Parameters = make(map[string]string, len(trial.Parameters))
		for key, value := range trial.Parameters {
			clone.Parameters[key] = value
		}
	}
	return clone
}
