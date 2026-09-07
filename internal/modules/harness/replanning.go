package harness

import (
	"fmt"
	"strings"
	"time"
)

const ResearchReplanContractV1 = "jax.research_replan/v1"

const (
	ReplanApproved        = "REPLAN_APPROVED"
	ResearchSufficient    = "RESEARCH_SUFFICIENT"
	InsufficientEvidence  = "INSUFFICIENT_EVIDENCE"
	ReplanBlocked         = "REPLAN_BLOCKED"
	ReplanBudgetExhausted = "BUDGET_EXHAUSTED"
)

type ResearchGap struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	EvidenceIDs []string `json:"evidence_ids"`
	Priority    int      `json:"priority"`
}

func (gap ResearchGap) Validate() error {
	if !validControlledID(gap.ID) || strings.TrimSpace(gap.Kind) == "" || len(gap.Kind) > 64 || strings.TrimSpace(gap.Description) == "" || len(gap.Description) > 2048 || !schemaStringList(gap.EvidenceIDs) || len(gap.EvidenceIDs) == 0 || gap.Priority < 1 || gap.Priority > 5 {
		return fmt.Errorf("research gap is invalid or unbounded")
	}
	for _, evidenceID := range gap.EvidenceIDs {
		if !validArgumentIdentifier(evidenceID) {
			return fmt.Errorf("research gap has invalid evidence ID")
		}
	}
	return nil
}

type ResearchPlanStep struct {
	StepID      string `json:"step_id"`
	Kind        string `json:"kind"`
	ToolID      string `json:"tool_id"`
	ToolVersion string `json:"tool_version"`
}

func (step ResearchPlanStep) Validate(registry *ControlledToolRegistry, policy ResearchPermissionPolicy) error {
	if !validControlledID(step.StepID) || step.Kind != "TOOL" || !validControlledToolID(step.ToolID) || strings.TrimSpace(step.ToolVersion) == "" {
		return fmt.Errorf("replanned step must be a bounded registered tool step")
	}
	spec, ok := registry.Get(step.ToolID)
	if !ok || spec.Version != step.ToolVersion {
		return fmt.Errorf("replanned step references an unknown or unsupported tool version")
	}
	return policy.Allows(spec)
}

type ResearchReplanPolicy struct {
	ContractVersion string `json:"contract_version"`
	Version         string `json:"version"`
	MaxReplans      int    `json:"max_replans"`
	MaxAddedSteps   int    `json:"max_added_steps"`
}

func (policy ResearchReplanPolicy) Validate() error {
	if policy.ContractVersion != ResearchReplanContractV1 || strings.TrimSpace(policy.Version) == "" || policy.MaxReplans < 0 || policy.MaxReplans > 8 || policy.MaxAddedSteps <= 0 || policy.MaxAddedSteps > 16 {
		return fmt.Errorf("research replan policy is invalid or unbounded")
	}
	return nil
}

type ResearchReplanDecision struct {
	ContractVersion    string             `json:"contract_version"`
	OriginalObjective  string             `json:"original_objective"`
	PreviousPlan       string             `json:"previous_plan_version"`
	NewPlan            string             `json:"new_plan_version"`
	Reason             string             `json:"reason"`
	TriggerEvidenceIDs []string           `json:"trigger_evidence_ids"`
	Gap                ResearchGap        `json:"gap"`
	AddedSteps         []ResearchPlanStep `json:"added_steps"`
	Outcome            string             `json:"outcome"`
	ReplanNumber       int                `json:"replan_number"`
	CreatedAt          time.Time          `json:"created_at"`
}

func (decision ResearchReplanDecision) Validate(registry *ControlledToolRegistry, toolPolicy ResearchPermissionPolicy, state ResearchTaskState, replanPolicy ResearchReplanPolicy) error {
	if err := replanPolicy.Validate(); err != nil {
		return err
	}
	if decision.ContractVersion != ResearchReplanContractV1 || decision.OriginalObjective != state.Objective || strings.TrimSpace(decision.PreviousPlan) != state.PlanVersion || strings.TrimSpace(decision.Reason) == "" || len(decision.Reason) > 2048 || !schemaStringList(decision.TriggerEvidenceIDs) || len(decision.TriggerEvidenceIDs) == 0 || decision.ReplanNumber < 0 || decision.CreatedAt.IsZero() || decision.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("replan decision is invalid or changes the objective")
	}
	if err := decision.Gap.Validate(); err != nil {
		return err
	}
	for _, evidenceID := range decision.TriggerEvidenceIDs {
		if !containsExact(state.EvidenceIDs, evidenceID) {
			return fmt.Errorf("replan trigger evidence is not present in task state")
		}
	}
	for _, evidenceID := range decision.Gap.EvidenceIDs {
		if !containsExact(state.EvidenceIDs, evidenceID) {
			return fmt.Errorf("research gap evidence is not present in task state")
		}
	}
	switch decision.Outcome {
	case ResearchSufficient, InsufficientEvidence, ReplanBlocked, ReplanBudgetExhausted:
		if len(decision.AddedSteps) != 0 || strings.TrimSpace(decision.NewPlan) != "" {
			return fmt.Errorf("non-approved replan outcome cannot add work")
		}
	case ReplanApproved:
		if decision.ReplanNumber != state.ReplanCount+1 || decision.ReplanNumber < 1 || decision.ReplanNumber > replanPolicy.MaxReplans || strings.TrimSpace(decision.NewPlan) == "" || decision.NewPlan == decision.PreviousPlan || len(decision.AddedSteps) == 0 || len(decision.AddedSteps) > replanPolicy.MaxAddedSteps {
			return fmt.Errorf("approved replan exceeds bounded policy")
		}
		seen := make(map[string]struct{}, len(state.PlannedSteps)+len(decision.AddedSteps))
		for _, stepID := range state.PlannedSteps {
			seen[stepID] = struct{}{}
		}
		for _, step := range decision.AddedSteps {
			if _, exists := seen[step.StepID]; exists {
				return fmt.Errorf("replan duplicates an existing step")
			}
			if err := step.Validate(registry, toolPolicy); err != nil {
				return err
			}
			seen[step.StepID] = struct{}{}
		}
	default:
		return fmt.Errorf("unsupported replan outcome %q", decision.Outcome)
	}
	return nil
}

func ProposeResearchReplan(state ResearchTaskState, gap ResearchGap, candidates []ResearchPlanStep, reason string, triggerEvidenceIDs []string, replanNumber int, newPlan string, registry *ControlledToolRegistry, toolPolicy ResearchPermissionPolicy, policy ResearchReplanPolicy, now time.Time) (ResearchReplanDecision, error) {
	if err := state.Validate(); err != nil {
		return ResearchReplanDecision{}, err
	}
	if err := policy.Validate(); err != nil {
		return ResearchReplanDecision{}, err
	}
	decision := ResearchReplanDecision{ContractVersion: ResearchReplanContractV1, OriginalObjective: state.Objective, PreviousPlan: state.PlanVersion, NewPlan: newPlan, Reason: reason, TriggerEvidenceIDs: append([]string(nil), triggerEvidenceIDs...), Gap: gap, AddedSteps: append([]ResearchPlanStep(nil), candidates...), ReplanNumber: replanNumber, CreatedAt: now}
	if replanNumber > policy.MaxReplans {
		decision.Outcome = ReplanBudgetExhausted
	} else if len(candidates) == 0 {
		decision.Outcome = InsufficientEvidence
	} else {
		decision.Outcome = ReplanApproved
	}
	if err := decision.Validate(registry, toolPolicy, state, policy); err != nil {
		return ResearchReplanDecision{}, err
	}
	return decision, nil
}

func ApplyResearchReplan(state ResearchTaskState, decision ResearchReplanDecision, registry *ControlledToolRegistry, toolPolicy ResearchPermissionPolicy, policy ResearchReplanPolicy, now time.Time) (ResearchTaskState, error) {
	if err := decision.Validate(registry, toolPolicy, state, policy); err != nil {
		return ResearchTaskState{}, err
	}
	if decision.Outcome != ReplanApproved {
		return state, nil
	}
	updated := state
	updated.PlannedSteps = append(append([]string(nil), state.PlannedSteps...), stepIDs(decision.AddedSteps)...)
	updated.PlanVersion = decision.NewPlan
	updated.ReplanCount = decision.ReplanNumber
	updated.UnresolvedGaps = removeExact(append([]string(nil), state.UnresolvedGaps...), decision.Gap.ID)
	updated.Status = ResearchTaskRunning
	updated.FailureReason = ""
	updated.CheckpointVersion++
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return ResearchTaskState{}, err
	}
	return updated, nil
}

func stepIDs(steps []ResearchPlanStep) []string {
	ids := make([]string, 0, len(steps))
	for _, step := range steps {
		ids = append(ids, step.StepID)
	}
	return ids
}

func removeExact(values []string, target string) []string {
	result := values[:0]
	for _, value := range values {
		if value != target {
			result = append(result, value)
		}
	}
	return result
}
