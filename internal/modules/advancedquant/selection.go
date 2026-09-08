package advancedquant

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const SelectionContractV1 = "jax.phase12.model_selection/v1"

type SelectionStatus string

const (
	SelectionBaselineWins      SelectionStatus = "ADVANCED_MODEL_NOT_JUSTIFIED"
	SelectionCandidateSelected SelectionStatus = "RESEARCH_CANDIDATE_SELECTED"
	SelectionRejected          SelectionStatus = "REJECTED"
)

type ModelCandidate struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Algorithm          string            `json:"algorithm"`
	Parameters         map[string]string `json:"parameters,omitempty"`
	ValidationMetric   float64           `json:"validation_metric"`
	BaselineMetric     float64           `json:"baseline_metric"`
	CostAdjusted       bool              `json:"cost_adjusted"`
	SelectionPartition Partition         `json:"selection_partition"`
	ComplexityRank     int               `json:"complexity_rank"`
}

func NewModelCandidate(candidate ModelCandidate) (ModelCandidate, error) {
	candidate.ID = modelID(candidate)
	if err := candidate.Validate(); err != nil {
		return ModelCandidate{}, err
	}
	return candidate, nil
}

func (c ModelCandidate) Validate() error {
	if !validHashID(c.ID, "model_") || strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Algorithm) == "" || !finite(c.ValidationMetric) || !finite(c.BaselineMetric) || !c.CostAdjusted || c.SelectionPartition != PartitionValidation || c.ComplexityRank < 0 {
		return fmt.Errorf("model candidate requires frozen validation-only, cost-adjusted evidence")
	}
	if c.ID != modelID(c) {
		return fmt.Errorf("model candidate identity does not match configuration")
	}
	return nil
}

type ModelSelectionResult struct {
	ContractVersion          string          `json:"contract_version"`
	SelectionID              string          `json:"selection_id"`
	Status                   SelectionStatus `json:"status"`
	BaselineName             string          `json:"baseline_name"`
	SelectedModelID          string          `json:"selected_model_id,omitempty"`
	CandidateIDs             []string        `json:"candidate_ids"`
	SelectionPartition       Partition       `json:"selection_partition"`
	MinimumImprovement       float64         `json:"minimum_improvement"`
	EnsembleRequested        bool            `json:"ensemble_requested"`
	ComplementarityValidated bool            `json:"complementarity_validated"`
	Reason                   string          `json:"reason"`
}

func SelectModel(baselineName string, baselineMetric float64, candidates []ModelCandidate, minimumImprovement float64, ensembleRequested, complementarityValidated bool) (ModelSelectionResult, error) {
	if strings.TrimSpace(baselineName) == "" || !finite(baselineMetric) || !finite(minimumImprovement) || minimumImprovement < 0 || len(candidates) == 0 {
		return ModelSelectionResult{}, fmt.Errorf("selection requires baseline, candidate and finite improvement threshold")
	}
	if ensembleRequested && !complementarityValidated {
		return ModelSelectionResult{}, fmt.Errorf("ensemble requires validated complementary value")
	}
	result := ModelSelectionResult{ContractVersion: SelectionContractV1, BaselineName: baselineName, SelectionPartition: PartitionValidation, MinimumImprovement: minimumImprovement, EnsembleRequested: ensembleRequested, ComplementarityValidated: complementarityValidated, Status: SelectionBaselineWins, Reason: "baseline remains adequate; advanced model is not justified", CandidateIDs: make([]string, 0, len(candidates))}
	var selected *ModelCandidate
	for index := range candidates {
		candidate := candidates[index]
		if err := candidate.Validate(); err != nil {
			return ModelSelectionResult{}, err
		}
		result.CandidateIDs = append(result.CandidateIDs, candidate.ID)
		if candidate.ValidationMetric >= baselineMetric+minimumImprovement && (selected == nil || candidate.ValidationMetric > selected.ValidationMetric || candidate.ValidationMetric == selected.ValidationMetric && candidate.ComplexityRank < selected.ComplexityRank) {
			selected = &candidate
		}
	}
	if selected != nil {
		result.Status = SelectionCandidateSelected
		result.SelectedModelID = selected.ID
		result.Reason = "candidate exceeds baseline on frozen validation data; OOS remains unscored until configuration freeze"
	}
	result.SelectionID = selectionID(result)
	return result, nil
}

func (r ModelSelectionResult) Validate() error {
	if r.ContractVersion != SelectionContractV1 || !validHashID(r.SelectionID, "select_") || strings.TrimSpace(r.BaselineName) == "" || r.SelectionPartition != PartitionValidation || !finite(r.MinimumImprovement) || r.MinimumImprovement < 0 || len(r.CandidateIDs) == 0 || strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("model selection result is incomplete or uses an invalid partition")
	}
	if r.EnsembleRequested && !r.ComplementarityValidated {
		return fmt.Errorf("unvalidated ensemble selection")
	}
	if r.Status == SelectionCandidateSelected && !validHashID(r.SelectedModelID, "model_") {
		return fmt.Errorf("selected model identity is missing")
	}
	if r.Status != SelectionBaselineWins && r.Status != SelectionCandidateSelected && r.Status != SelectionRejected {
		return fmt.Errorf("unsupported model selection status")
	}
	if r.SelectionID != selectionID(r) {
		return fmt.Errorf("selection identity does not match result")
	}
	return nil
}

func modelID(c ModelCandidate) string {
	c.ID = ""
	b, _ := json.Marshal(c)
	d := sha256.Sum256(b)
	return "model_" + hex.EncodeToString(d[:])
}
func selectionID(r ModelSelectionResult) string {
	r.SelectionID = ""
	b, _ := json.Marshal(r)
	d := sha256.Sum256(b)
	return "select_" + hex.EncodeToString(d[:])
}
