package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	DriftPolicyContractV1 = "jax.phase12.drift_policy/v1"
	DriftReportContractV1 = "jax.phase12.drift_report/v1"
	RetrainingContractV1  = "jax.phase12.rolling_retraining/v1"
)

type DriftState string

const (
	DriftResearch DriftState = "RESEARCH"
	DriftValid    DriftState = "VALID"
	DriftDegraded DriftState = "DEGRADED"
	DriftRejected DriftState = "REJECTED"
	DriftRetired  DriftState = "RETIRED"
)

type DriftPolicy struct {
	ContractVersion     string  `json:"contract_version"`
	ID                  string  `json:"id"`
	Version             string  `json:"version"`
	MaxFeatureDrift     float64 `json:"max_feature_drift"`
	MaxPredictionDrift  float64 `json:"max_prediction_drift"`
	MinimumPerformance  float64 `json:"minimum_performance"`
	RetireAfterDegraded int     `json:"retire_after_degraded"`
}

func NewDriftPolicy(policy DriftPolicy) (DriftPolicy, error) {
	policy.ContractVersion = DriftPolicyContractV1
	policy.ID = driftPolicyID(policy)
	if err := policy.Validate(); err != nil {
		return DriftPolicy{}, err
	}
	return policy, nil
}

func (p DriftPolicy) Validate() error {
	if p.ContractVersion != DriftPolicyContractV1 || !validHashID(p.ID, "driftpolicy_") || strings.TrimSpace(p.Version) == "" || !finite(p.MaxFeatureDrift) || !finite(p.MaxPredictionDrift) || !finite(p.MinimumPerformance) || p.MaxFeatureDrift < 0 || p.MaxPredictionDrift < 0 || p.MinimumPerformance < -1 || p.MinimumPerformance > 1 || p.RetireAfterDegraded <= 0 || p.ID != driftPolicyID(p) {
		return fmt.Errorf("drift policy is invalid")
	}
	return nil
}

type DriftReport struct {
	ContractVersion string     `json:"contract_version"`
	ModelID         string     `json:"model_id"`
	PolicyID        string     `json:"policy_id"`
	ObservedAt      time.Time  `json:"observed_at"`
	FeatureDrift    float64    `json:"feature_drift"`
	PredictionDrift float64    `json:"prediction_drift"`
	Performance     float64    `json:"performance"`
	SampleCount     int        `json:"sample_count"`
	PriorDegraded   int        `json:"prior_degraded"`
	State           DriftState `json:"state"`
	Reason          string     `json:"reason"`
}

func EvaluateDrift(modelID string, policy DriftPolicy, observedAt time.Time, featureDrift, predictionDrift, performance float64, sampleCount, priorDegraded int) (DriftReport, error) {
	if err := policy.Validate(); err != nil {
		return DriftReport{}, err
	}
	if !validHashID(modelID, "model_") || observedAt.IsZero() || observedAt.Location() != time.UTC || !finite(featureDrift) || !finite(predictionDrift) || !finite(performance) || featureDrift < 0 || predictionDrift < 0 || sampleCount < 0 || priorDegraded < 0 {
		return DriftReport{}, fmt.Errorf("drift observation is invalid")
	}
	report := DriftReport{ContractVersion: DriftReportContractV1, ModelID: modelID, PolicyID: policy.ID, ObservedAt: observedAt, FeatureDrift: featureDrift, PredictionDrift: predictionDrift, Performance: performance, SampleCount: sampleCount, PriorDegraded: priorDegraded, State: DriftValid, Reason: "within registered drift policy"}
	if sampleCount == 0 {
		report.State = DriftDegraded
		report.Reason = "insufficient monitoring observations"
	} else if featureDrift > policy.MaxFeatureDrift || predictionDrift > policy.MaxPredictionDrift || performance < policy.MinimumPerformance {
		report.State = DriftDegraded
		report.Reason = "monitoring threshold breached"
		if priorDegraded+1 >= policy.RetireAfterDegraded {
			report.State = DriftRejected
			report.Reason = "repeated degradation exceeded retirement threshold"
		}
	}
	return report, nil
}

type RollingRetrainingWindow struct {
	ContractVersion string            `json:"contract_version"`
	ID              string            `json:"id"`
	ModelID         string            `json:"model_id"`
	DatasetID       string            `json:"dataset_id"`
	FeatureVersion  string            `json:"feature_version"`
	TrainStart      time.Time         `json:"train_start"`
	TrainEnd        time.Time         `json:"train_end"`
	EvaluateStart   time.Time         `json:"evaluate_start"`
	EvaluateEnd     time.Time         `json:"evaluate_end"`
	Parameters      map[string]string `json:"parameters,omitempty"`
}

func NewRollingRetrainingWindow(window RollingRetrainingWindow) (RollingRetrainingWindow, error) {
	window.ContractVersion = RetrainingContractV1
	window.ID = retrainingID(window)
	if err := window.Validate(); err != nil {
		return RollingRetrainingWindow{}, err
	}
	return window, nil
}

func (w RollingRetrainingWindow) Validate() error {
	if w.ContractVersion != RetrainingContractV1 || !validHashID(w.ID, "retrain_") || !validHashID(w.ModelID, "model_") || strings.TrimSpace(w.DatasetID) == "" || strings.TrimSpace(w.FeatureVersion) == "" || w.TrainStart.IsZero() || w.TrainEnd.IsZero() || w.EvaluateStart.IsZero() || w.EvaluateEnd.IsZero() || w.TrainStart.Location() != time.UTC || w.TrainEnd.Location() != time.UTC || w.EvaluateStart.Location() != time.UTC || w.EvaluateEnd.Location() != time.UTC || w.TrainEnd.Before(w.TrainStart) || !w.EvaluateEnd.After(w.EvaluateStart) || !w.EvaluateStart.After(w.TrainEnd) || w.ID != retrainingID(w) {
		return fmt.Errorf("rolling retraining window is invalid or overlaps evaluation data")
	}
	return nil
}

func driftPolicyID(p DriftPolicy) string {
	p.ID = ""
	b, _ := json.Marshal(p)
	d := sha256.Sum256(b)
	return "driftpolicy_" + hex.EncodeToString(d[:])
}
func retrainingID(w RollingRetrainingWindow) string {
	w.ID = ""
	b, _ := json.Marshal(w)
	d := sha256.Sum256(b)
	return "retrain_" + hex.EncodeToString(d[:])
}
