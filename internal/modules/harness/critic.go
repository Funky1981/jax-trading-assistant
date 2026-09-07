package harness

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const ResearchCriticPolicyContractV1 = "jax.research_critic_policy/v1"
const ResearchCritiqueContractV1 = "jax.research_critique/v1"

const (
	CriticUnsupportedClaim          = "UNSUPPORTED_CLAIM"
	CriticEvidenceOmission          = "EVIDENCE_OMISSION"
	CriticContradiction             = "CONTRADICTION"
	CriticStaleEvidence             = "STALE_EVIDENCE"
	CriticWeakSourceIndependence    = "WEAK_SOURCE_INDEPENDENCE"
	CriticUnknown                   = "UNKNOWN"
	CriticOverconfidence            = "OVERCONFIDENCE"
	CriticMissingCounterCase        = "MISSING_COUNTER_CASE"
	CriticRecommendationImplication = "RECOMMENDATION_IMPLICATION"
)

const (
	CriticActionDowngrade           = "DOWNGRADE"
	CriticActionAddUnknown          = "ADD_UNKNOWN"
	CriticActionRequestResearch     = "REQUEST_RESEARCH"
	CriticActionRetainContradiction = "RETAIN_CONTRADICTION"
)

type ResearchCriticPolicy struct {
	ContractVersion string `json:"contract_version"`
	Version         string `json:"version"`
	MaxCycles       int    `json:"max_cycles"`
	MaxFindings     int    `json:"max_findings"`
}

func (policy ResearchCriticPolicy) Validate() error {
	if policy.ContractVersion != ResearchCriticPolicyContractV1 || strings.TrimSpace(policy.Version) == "" || policy.MaxCycles <= 0 || policy.MaxCycles > 3 || policy.MaxFindings <= 0 || policy.MaxFindings > 32 {
		return fmt.Errorf("critic policy is invalid or unbounded")
	}
	return nil
}

type CriticFinding struct {
	ID           string       `json:"id"`
	Category     string       `json:"category"`
	Severity     string       `json:"severity"`
	Description  string       `json:"description"`
	EvidenceIDs  []string     `json:"evidence_ids"`
	Action       string       `json:"action"`
	RequestedGap *ResearchGap `json:"requested_gap,omitempty"`
}

func (finding CriticFinding) Validate(state ResearchTaskState) error {
	if !validControlledID(finding.ID) || strings.TrimSpace(finding.Description) == "" || len(finding.Description) > 2048 || !schemaStringList(finding.EvidenceIDs) || len(finding.EvidenceIDs) > 16 {
		return fmt.Errorf("critic finding is invalid or unbounded")
	}
	switch finding.Category {
	case CriticUnsupportedClaim, CriticEvidenceOmission, CriticContradiction, CriticStaleEvidence, CriticWeakSourceIndependence, CriticUnknown, CriticOverconfidence, CriticMissingCounterCase, CriticRecommendationImplication:
	default:
		return fmt.Errorf("unsupported critic finding category %q", finding.Category)
	}
	switch finding.Severity {
	case "LOW", "MEDIUM", "HIGH":
	default:
		return fmt.Errorf("unsupported critic finding severity %q", finding.Severity)
	}
	switch finding.Action {
	case CriticActionDowngrade, CriticActionAddUnknown, CriticActionRequestResearch, CriticActionRetainContradiction:
	default:
		return fmt.Errorf("unsupported critic finding action %q", finding.Action)
	}
	for _, evidenceID := range finding.EvidenceIDs {
		if !containsExact(state.EvidenceIDs, evidenceID) {
			return fmt.Errorf("critic finding references evidence outside task state")
		}
	}
	if finding.Action == CriticActionRequestResearch {
		if finding.RequestedGap == nil {
			return fmt.Errorf("research-request critic finding requires a gap")
		}
		if err := finding.RequestedGap.Validate(); err != nil {
			return err
		}
		for _, evidenceID := range finding.RequestedGap.EvidenceIDs {
			if !containsExact(state.EvidenceIDs, evidenceID) {
				return fmt.Errorf("requested research gap references evidence outside task state")
			}
		}
	} else if finding.RequestedGap != nil {
		return fmt.Errorf("only a research-request finding may contain a gap")
	}
	return nil
}

type ResearchCritique struct {
	ContractVersion        string          `json:"contract_version"`
	TaskID                 string          `json:"task_id"`
	Cycle                  int             `json:"cycle"`
	InputReportSHA256      string          `json:"input_report_sha256"`
	Findings               []CriticFinding `json:"findings"`
	ImprovedReport         string          `json:"improved_report"`
	AddedUnknowns          []string        `json:"added_unknowns"`
	RetainedContradictions []string        `json:"retained_contradictions"`
	RequestedGaps          []ResearchGap   `json:"requested_gaps"`
	Downgraded             bool            `json:"downgraded"`
	Outcome                string          `json:"outcome"`
	ExecutionAuthority     string          `json:"execution_authority"`
	CreatedAt              time.Time       `json:"created_at"`
}

func (critique ResearchCritique) Validate(state ResearchTaskState, policy ResearchCriticPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if critique.ContractVersion != ResearchCritiqueContractV1 || critique.TaskID != state.TaskID || critique.Cycle != state.CriticCycles+1 || critique.Cycle > policy.MaxCycles || !validSHA256Usage(critique.InputReportSHA256) || critique.InputReportSHA256 != reportDigest(state.CurrentReport) || len(critique.Findings) > policy.MaxFindings || len(critique.ImprovedReport) > 128*1024 || len(critique.AddedUnknowns) > 32 || len(critique.RetainedContradictions) > 64 || !schemaStringListOrEmpty(critique.RetainedContradictions) || len(critique.RequestedGaps) > 16 || critique.ExecutionAuthority != "NONE" || strings.TrimSpace(critique.Outcome) == "" || critique.CreatedAt.IsZero() || critique.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("critic result is invalid, stale or exceeds bounds")
	}
	seen := make(map[string]struct{}, len(critique.Findings))
	for _, finding := range critique.Findings {
		if _, exists := seen[finding.ID]; exists {
			return fmt.Errorf("critic finding is duplicated")
		}
		seen[finding.ID] = struct{}{}
		if err := finding.Validate(state); err != nil {
			return err
		}
	}
	for _, unknown := range critique.AddedUnknowns {
		if strings.TrimSpace(unknown) == "" || len(unknown) > 2048 {
			return fmt.Errorf("critic unknown is invalid or unbounded")
		}
	}
	for _, contradiction := range state.Contradictions {
		if !containsExact(critique.RetainedContradictions, contradiction) {
			return fmt.Errorf("critic dropped an existing contradiction")
		}
	}
	for _, gap := range critique.RequestedGaps {
		if err := gap.Validate(); err != nil {
			return err
		}
		for _, evidenceID := range gap.EvidenceIDs {
			if !containsExact(state.EvidenceIDs, evidenceID) {
				return fmt.Errorf("critic gap references evidence outside task state")
			}
		}
	}
	if critique.Downgraded && len(critique.Findings) == 0 {
		return fmt.Errorf("critic cannot downgrade without a finding")
	}
	return nil
}

func RunBoundedCritique(state ResearchTaskState, findings []CriticFinding, improvedReport string, unknowns []string, retainedContradictions []string, requestedGaps []ResearchGap, downgraded bool, outcome string, policy ResearchCriticPolicy, now time.Time) (ResearchCritique, error) {
	if err := state.Validate(); err != nil {
		return ResearchCritique{}, err
	}
	critique := ResearchCritique{ContractVersion: ResearchCritiqueContractV1, TaskID: state.TaskID, Cycle: state.CriticCycles + 1, InputReportSHA256: reportDigest(state.CurrentReport), Findings: append([]CriticFinding(nil), findings...), ImprovedReport: improvedReport, AddedUnknowns: append([]string(nil), unknowns...), RetainedContradictions: append([]string(nil), retainedContradictions...), RequestedGaps: append([]ResearchGap(nil), requestedGaps...), Downgraded: downgraded, Outcome: outcome, ExecutionAuthority: "NONE", CreatedAt: now}
	if err := critique.Validate(state, policy); err != nil {
		return ResearchCritique{}, err
	}
	return critique, nil
}

func ApplyBoundedCritique(state ResearchTaskState, critique ResearchCritique, policy ResearchCriticPolicy, now time.Time) (ResearchTaskState, error) {
	if err := critique.Validate(state, policy); err != nil {
		return ResearchTaskState{}, err
	}
	updated := state
	updated.CriticCycles = critique.Cycle
	if strings.TrimSpace(critique.ImprovedReport) != "" {
		updated.CurrentReport = critique.ImprovedReport
	}
	updated.UnresolvedGaps = append([]string(nil), state.UnresolvedGaps...)
	for _, gap := range critique.RequestedGaps {
		if !containsExact(updated.UnresolvedGaps, gap.ID) {
			updated.UnresolvedGaps = append(updated.UnresolvedGaps, gap.ID)
		}
	}
	updated.Contradictions = append([]string(nil), critique.RetainedContradictions...)
	for _, unknown := range critique.AddedUnknowns {
		if !containsExact(updated.Contradictions, unknown) {
			updated.Contradictions = append(updated.Contradictions, unknown)
		}
	}
	sort.Strings(updated.UnresolvedGaps)
	sort.Strings(updated.Contradictions)
	updated.CheckpointVersion++
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return ResearchTaskState{}, err
	}
	return updated, nil
}

func reportDigest(report string) string {
	digest := sha256.Sum256([]byte(report))
	return hex.EncodeToString(digest[:])
}
