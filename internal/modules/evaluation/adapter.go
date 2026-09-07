package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const BacktestAdapterAssessmentContractV1 = "jax.backtest_adapter_assessment/v1"

const (
	AdapterSelected = "SELECTED"
	AdapterRejected = "REJECTED"
	AdapterDeferred = "DEFERRED"
)

type BacktestAdapterCandidate struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Version                string `json:"version"`
	Language               string `json:"language"`
	License                string `json:"license"`
	EventDriven            bool   `json:"event_driven"`
	PointInTimeSafe        bool   `json:"point_in_time_safe"`
	TransactionCostSupport bool   `json:"transaction_cost_support"`
	Reproducible           bool   `json:"reproducible"`
	CustomDataSupport      bool   `json:"custom_data_support"`
	OperationalBurden      string `json:"operational_burden"`
	IntegrationBurden      string `json:"integration_burden"`
	DecisionStatus         string `json:"decision_status"`
	AssessmentNotes        string `json:"assessment_notes"`
}

func NewBacktestAdapterCandidate(candidate BacktestAdapterCandidate) (BacktestAdapterCandidate, error) {
	candidate.ID = deriveAdapterCandidateID(candidate)
	if err := candidate.Validate(); err != nil {
		return BacktestAdapterCandidate{}, err
	}
	return candidate, nil
}

func (candidate BacktestAdapterCandidate) Validate() error {
	if !validIdentity("adapter_", candidate.ID) || strings.TrimSpace(candidate.Name) == "" || strings.TrimSpace(candidate.Version) == "" || strings.TrimSpace(candidate.Language) == "" || strings.TrimSpace(candidate.License) == "" || strings.TrimSpace(candidate.OperationalBurden) == "" || strings.TrimSpace(candidate.IntegrationBurden) == "" || strings.TrimSpace(candidate.AssessmentNotes) == "" {
		return fmt.Errorf("backtest adapter candidate requires version, license, burden and assessment metadata")
	}
	switch candidate.DecisionStatus {
	case AdapterSelected, AdapterRejected, AdapterDeferred:
	default:
		return fmt.Errorf("unsupported backtest adapter decision status %q", candidate.DecisionStatus)
	}
	if candidate.ID != deriveAdapterCandidateID(candidate) {
		return fmt.Errorf("backtest adapter candidate ID does not match its content")
	}
	return nil
}

type BacktestAdapterAssessment struct {
	ContractVersion      string                     `json:"contract_version"`
	ID                   string                     `json:"id"`
	EvaluatedAt          time.Time                  `json:"evaluated_at"`
	Scope                string                     `json:"scope"`
	CandidateIDs         []string                   `json:"candidate_ids"`
	Candidates           []BacktestAdapterCandidate `json:"candidates"`
	SelectedCandidateID  string                     `json:"selected_candidate_id"`
	SelectionRationale   string                     `json:"selection_rationale"`
	RejectedAlternatives []string                   `json:"rejected_alternatives"`
	ExecutionAuthority   string                     `json:"execution_authority"`
	InferenceUsed        bool                       `json:"inference_used"`
}

func NewBacktestAdapterAssessment(evaluatedAt time.Time, scope string, candidates []BacktestAdapterCandidate, selectedID, rationale string) (BacktestAdapterAssessment, error) {
	assessment := BacktestAdapterAssessment{ContractVersion: BacktestAdapterAssessmentContractV1, EvaluatedAt: evaluatedAt, Scope: scope, Candidates: append([]BacktestAdapterCandidate(nil), candidates...), SelectedCandidateID: selectedID, SelectionRationale: rationale, ExecutionAuthority: "NONE", InferenceUsed: false}
	for _, candidate := range candidates {
		assessment.CandidateIDs = append(assessment.CandidateIDs, candidate.ID)
		if candidate.ID != selectedID {
			assessment.RejectedAlternatives = append(assessment.RejectedAlternatives, candidate.ID)
		}
	}
	sort.Strings(assessment.CandidateIDs)
	sort.Strings(assessment.RejectedAlternatives)
	assessment.ID = deriveAdapterAssessmentID(assessment)
	if err := assessment.Validate(); err != nil {
		return BacktestAdapterAssessment{}, err
	}
	return assessment, nil
}

func (assessment BacktestAdapterAssessment) Validate() error {
	if assessment.ContractVersion != BacktestAdapterAssessmentContractV1 || !validIdentity("assessment_", assessment.ID) || assessment.EvaluatedAt.IsZero() || assessment.EvaluatedAt.Location() != time.UTC || strings.TrimSpace(assessment.Scope) == "" || strings.TrimSpace(assessment.SelectedCandidateID) == "" || strings.TrimSpace(assessment.SelectionRationale) == "" || assessment.ExecutionAuthority != "NONE" || assessment.InferenceUsed || len(assessment.Candidates) == 0 || !strictSortedUnique(assessment.CandidateIDs) || !strictSortedUnique(assessment.RejectedAlternatives) {
		return fmt.Errorf("backtest adapter assessment requires deterministic bounded selection metadata")
	}
	if len(assessment.CandidateIDs) != len(assessment.Candidates) {
		return fmt.Errorf("backtest adapter assessment candidate identities are incomplete")
	}
	seen := map[string]struct{}{}
	selected := false
	for _, candidate := range assessment.Candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
		if _, exists := seen[candidate.ID]; exists {
			return fmt.Errorf("backtest adapter candidates must be unique")
		}
		seen[candidate.ID] = struct{}{}
		if candidate.ID == assessment.SelectedCandidateID {
			selected = true
			if candidate.DecisionStatus != AdapterSelected {
				return fmt.Errorf("selected adapter must be marked SELECTED")
			}
			if !candidate.EventDriven || !candidate.PointInTimeSafe || !candidate.TransactionCostSupport || !candidate.Reproducible || !candidate.CustomDataSupport {
				return fmt.Errorf("selected adapter must support event-driven point-in-time reproducible custom-data evaluation with costs")
			}
		} else if candidate.DecisionStatus == AdapterSelected {
			return fmt.Errorf("only the selected adapter may be marked SELECTED")
		}
	}
	if !selected || !sameAdapterStrings(assessment.CandidateIDs, mapKeys(seen)) || !sameAdapterStrings(assessment.RejectedAlternatives, rejectIDs(assessment.Candidates, assessment.SelectedCandidateID)) || assessment.ID != deriveAdapterAssessmentID(assessment) {
		return fmt.Errorf("backtest adapter assessment has inconsistent candidate identity or selection")
	}
	return nil
}

func sameAdapterStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func rejectIDs(candidates []BacktestAdapterCandidate, selected string) []string {
	ids := make([]string, 0, len(candidates)-1)
	for _, candidate := range candidates {
		if candidate.ID != selected {
			ids = append(ids, candidate.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func deriveAdapterCandidateID(candidate BacktestAdapterCandidate) string {
	candidate.ID = ""
	seed, _ := json.Marshal(candidate)
	digest := sha256.Sum256(seed)
	return "adapter_" + hex.EncodeToString(digest[:])
}

func deriveAdapterAssessmentID(assessment BacktestAdapterAssessment) string {
	assessment.ID = ""
	seed, _ := json.Marshal(assessment)
	digest := sha256.Sum256(seed)
	return "assessment_" + hex.EncodeToString(digest[:])
}
