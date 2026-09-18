// Package harnesscontracts contains the versioned, dependency-light contracts
// shared by future Jax harness implementations.
//
// This package intentionally contains no model provider, retrieval, memory,
// evaluator, persistence, HTTP, broker, or execution code. It is not imported
// by cmd/trader. The pre-existing internal/modules/harness package is a
// separate advisory/chat implementation and is intentionally not changed by
// HARNESS-01.
package harnesscontracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ResearchObjectiveContractV1        = "jax.harness.research_objective/v1"
	TaskStateContractV1                = "jax.harness.task_state/v1"
	ContextPackageContractV1           = "jax.harness.context_package/v1"
	ContextBudgetContractV1            = "jax.harness.context_budget/v1"
	EvidenceReferenceContractV1        = "jax.harness.evidence_reference/v1"
	MemoryReferenceContractV1          = "jax.harness.memory_reference/v1"
	ToolReferenceContractV1            = "jax.harness.tool_reference/v1"
	CheckpointContractV1               = "jax.harness.checkpoint/v1"
	ContextProvenanceContractV1        = "jax.harness.context_provenance/v1"
	EvaluatorInputContractV1           = "jax.harness.evaluator_input/v1"
	CompactionRecordContractV1         = "jax.harness.compaction_record/v1"
	StructuredResearchOutputContractV1 = "jax.harness.structured_output/v1"
	EvaluatorResultContractV1          = "jax.harness.evaluator_result/v1"
	HarnessRunContractV1               = "jax.harness.run/v1"
	EngineeringObjectiveContractV1     = "jax.harness.engineering_objective/v1"
	RepositoryStateContractV1          = "jax.harness.repository_state/v1"
	DocumentationMapContractV1         = "jax.harness.documentation_map/v1"
	VerificationPlanContractV1         = "jax.harness.verification_plan/v1"
	EngineeringCheckpointContractV1    = "jax.harness.engineering_checkpoint/v1"
	CleanExitReportContractV1          = "jax.harness.clean_exit/v1"
	EngineeringHarnessRunContractV1    = "jax.harness.engineering_run/v1"
	CanonicalHashAlgorithmSHA256       = "sha256"
)

const (
	ObjectiveStatusDraft     = "DRAFT"
	ObjectiveStatusActive    = "ACTIVE"
	ObjectiveStatusCompleted = "COMPLETED"
	ObjectiveStatusAbandoned = "ABANDONED"

	TaskStatusRunning   = "RUNNING"
	TaskStatusBlocked   = "BLOCKED"
	TaskStatusCompleted = "COMPLETED"
	TaskStatusFailed    = "FAILED"
	TaskStatusAbandoned = "ABANDONED"

	EvidenceCandidate      = "candidate"
	EvidenceSupporting     = "supporting"
	EvidenceContradictory  = "contradictory"
	EvidenceInvalidating   = "invalidating"
	EvidenceUnknownMissing = "unknown_missing"

	ToolStatusRequested = "REQUESTED"
	ToolStatusSucceeded = "SUCCEEDED"
	ToolStatusFailed    = "FAILED"
	ToolStatusSkipped   = "SKIPPED"

	HarnessRunRunning   = "RUNNING"
	HarnessRunSucceeded = "SUCCEEDED"
	HarnessRunFailed    = "FAILED"
	HarnessRunAbstained = "ABSTAINED"

	EvaluatorPass    = "PASS"
	EvaluatorRevise  = "REVISE"
	EvaluatorAbstain = "ABSTAIN"

	OutputStatusDraft      = "DRAFT"
	OutputStatusComplete   = "COMPLETE"
	OutputStatusIncomplete = "INCOMPLETE"
	OutputStatusAbstain    = "ABSTAIN"

	RepositoryWorktreeClean   = "CLEAN"
	RepositoryWorktreeDirty   = "DIRTY"
	RepositoryWorktreeUnknown = "UNKNOWN"

	CleanExitPass    = "PASS"
	CleanExitBlocked = "BLOCKED"

	VerificationPass    = "PASS"
	VerificationFail    = "FAIL"
	VerificationSkipped = "SKIPPED"
)

var (
	identifierPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	versionPattern       = regexp.MustCompile(`^v[1-9][0-9]*$`)
	shaPattern           = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
	canonicalHashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// PolicyReference identifies a versioned policy without copying its content.
type PolicyReference struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (reference PolicyReference) Validate() error {
	if !validReference(reference.Name) || !validVersion(reference.Version) {
		return fmt.Errorf("policy reference is malformed")
	}
	return nil
}

type ObjectiveReference struct {
	ObjectiveID      string `json:"objective_id"`
	ObjectiveVersion string `json:"objective_version"`
}

func (reference ObjectiveReference) Validate() error {
	if !validIdentifier(reference.ObjectiveID) || !validVersion(reference.ObjectiveVersion) {
		return fmt.Errorf("objective reference is malformed")
	}
	return nil
}

type TaskStateReference struct {
	TaskID       string `json:"task_id"`
	StateVersion int    `json:"state_version"`
}

func (reference TaskStateReference) Validate() error {
	if !validIdentifier(reference.TaskID) || reference.StateVersion < 1 {
		return fmt.Errorf("task-state reference is malformed")
	}
	return nil
}

type ArtifactReference struct {
	ReferenceID string `json:"reference_id"`
	Contract    string `json:"contract"`
	ContentHash string `json:"content_hash"`
}

func (reference ArtifactReference) Validate() error {
	if !validIdentifier(reference.ReferenceID) || !validReference(reference.Contract) || !validCanonicalHash(reference.ContentHash) {
		return fmt.Errorf("artifact reference is malformed")
	}
	return nil
}

type ModelReference struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Version  string `json:"version"`
}

func (reference ModelReference) Validate() error {
	if !validReference(reference.Provider) || !validReference(reference.Model) || !validReference(reference.Version) {
		return fmt.Errorf("model reference is malformed")
	}
	return nil
}

type CompletionCriterion struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func (criterion CompletionCriterion) Validate() error {
	if !validIdentifier(criterion.ID) || !boundedText(criterion.Description, 4096) {
		return fmt.Errorf("completion criterion is malformed")
	}
	return nil
}

type ResearchObjective struct {
	ContractVersion    string                `json:"contract_version"`
	ObjectiveID        string                `json:"objective_id"`
	Version            string                `json:"version"`
	Purpose            string                `json:"purpose"`
	Question           string                `json:"question"`
	CreatedAt          time.Time             `json:"created_at"`
	Constraints        []string              `json:"constraints"`
	RequiredOutputs    []string              `json:"required_outputs"`
	CompletionCriteria []CompletionCriterion `json:"completion_criteria"`
	PolicyVersions     []PolicyReference     `json:"policy_versions"`
	Status             string                `json:"status"`
}

func NewResearchObjective(objective ResearchObjective) (ResearchObjective, error) {
	if objective.ContractVersion == "" {
		objective.ContractVersion = ResearchObjectiveContractV1
	}
	if objective.Status == "" {
		objective.Status = ObjectiveStatusDraft
	}
	if err := objective.Validate(); err != nil {
		return ResearchObjective{}, err
	}
	return objective, nil
}

func (objective ResearchObjective) Validate() error {
	if objective.ContractVersion != ResearchObjectiveContractV1 || !validIdentifier(objective.ObjectiveID) || !validVersion(objective.Version) || !boundedText(objective.Purpose, 4096) || !boundedText(objective.Question, 8192) || !validUTCTime(objective.CreatedAt) || !validStringList(objective.Constraints, 64, 4096) || len(objective.RequiredOutputs) == 0 || !validStringList(objective.RequiredOutputs, 32, 256) || len(objective.CompletionCriteria) == 0 || len(objective.CompletionCriteria) > 64 || !validPolicyReferences(objective.PolicyVersions) {
		return fmt.Errorf("research objective is incomplete or malformed")
	}
	seen := map[string]struct{}{}
	for _, criterion := range objective.CompletionCriteria {
		if err := criterion.Validate(); err != nil {
			return err
		}
		if _, exists := seen[criterion.ID]; exists {
			return fmt.Errorf("completion criteria contain duplicate ID %q", criterion.ID)
		}
		seen[criterion.ID] = struct{}{}
	}
	switch objective.Status {
	case ObjectiveStatusDraft, ObjectiveStatusActive, ObjectiveStatusCompleted, ObjectiveStatusAbandoned:
		return nil
	default:
		return fmt.Errorf("unsupported research objective status %q", objective.Status)
	}
}

type WorkItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

func (item WorkItem) Validate() error {
	if !validIdentifier(item.ID) || !boundedText(item.Description, 4096) {
		return fmt.Errorf("work item is malformed")
	}
	return nil
}

type DecisionRecord struct {
	DecisionID string    `json:"decision_id"`
	Summary    string    `json:"summary"`
	Reason     string    `json:"reason"`
	DecidedAt  time.Time `json:"decided_at"`
}

func (decision DecisionRecord) Validate() error {
	if !validIdentifier(decision.DecisionID) || !boundedText(decision.Summary, 4096) || !boundedText(decision.Reason, 8192) || !validUTCTime(decision.DecidedAt) {
		return fmt.Errorf("decision record is malformed")
	}
	return nil
}

type FailureState struct {
	Attempt        int    `json:"attempt"`
	Retryable      bool   `json:"retryable"`
	Classification string `json:"classification"`
	Reason         string `json:"reason"`
}

func (failure FailureState) Validate() error {
	if failure.Attempt < 0 || failure.Attempt > 1000 {
		return fmt.Errorf("failure state is malformed")
	}
	if strings.TrimSpace(failure.Classification) == "" && strings.TrimSpace(failure.Reason) == "" {
		return nil
	}
	if !boundedText(failure.Classification, 256) || !boundedText(failure.Reason, 8192) {
		return fmt.Errorf("failure state is malformed")
	}
	return nil
}

type TaskState struct {
	ContractVersion           string               `json:"contract_version"`
	TaskID                    string               `json:"task_id"`
	ObjectiveID               string               `json:"objective_id"`
	ObjectiveVersion          string               `json:"objective_version"`
	StateVersion              int                  `json:"state_version"`
	CurrentStage              string               `json:"current_stage"`
	CompletedWork             []WorkItem           `json:"completed_work"`
	PendingWork               []WorkItem           `json:"pending_work"`
	OpenQuestions             []string             `json:"open_questions"`
	KnownUnknowns             []string             `json:"known_unknowns"`
	Constraints               []string             `json:"constraints"`
	Decisions                 []DecisionRecord     `json:"decisions"`
	EvidenceReferences        []EvidenceReference  `json:"evidence_references"`
	CounterEvidenceReferences []EvidenceReference  `json:"counter_evidence_references"`
	MemoryReferences          []MemoryReference    `json:"memory_references"`
	Failure                   FailureState         `json:"failure"`
	Checkpoint                *CheckpointReference `json:"checkpoint_reference,omitempty"`
	NextAction                string               `json:"next_action"`
	CreatedAt                 time.Time            `json:"created_at"`
	UpdatedAt                 time.Time            `json:"updated_at"`
	Status                    string               `json:"status"`
}

func (state TaskState) Validate() error {
	if state.ContractVersion != TaskStateContractV1 || !validIdentifier(state.TaskID) || !validIdentifier(state.ObjectiveID) || !validVersion(state.ObjectiveVersion) || state.StateVersion < 1 || strings.TrimSpace(state.CurrentStage) == "" || !validUTCTime(state.CreatedAt) || !validUTCTime(state.UpdatedAt) || state.UpdatedAt.Before(state.CreatedAt) || !validStringList(state.OpenQuestions, 128, 4096) || !validStringList(state.KnownUnknowns, 128, 4096) || !validStringList(state.Constraints, 128, 4096) || !validStringList(state.NextActionList(), 1, 4096) || !validStringList(stateFailureReasonList(state), 1, 8192) {
		return fmt.Errorf("task state is incomplete or malformed")
	}
	if len(state.CompletedWork) > 256 || len(state.PendingWork) > 256 || len(state.Decisions) > 256 || len(state.EvidenceReferences) > 512 || len(state.CounterEvidenceReferences) > 512 || len(state.MemoryReferences) > 512 {
		return fmt.Errorf("task state is unbounded")
	}
	switch state.Status {
	case TaskStatusRunning, TaskStatusBlocked, TaskStatusCompleted, TaskStatusFailed, TaskStatusAbandoned:
	default:
		return fmt.Errorf("unsupported task state status %q", state.Status)
	}
	if err := state.Failure.Validate(); err != nil {
		return err
	}
	if (state.Status == TaskStatusBlocked || state.Status == TaskStatusFailed) != (strings.TrimSpace(state.Failure.Reason) != "") {
		return fmt.Errorf("task failure state and failure reason must agree")
	}
	if state.Status == TaskStatusCompleted && len(state.PendingWork) > 0 {
		return fmt.Errorf("completed task cannot have pending work")
	}
	if state.Status != TaskStatusCompleted && strings.TrimSpace(state.NextAction) == "" {
		return fmt.Errorf("non-terminal task requires next action")
	}
	seen := map[string]string{}
	for _, item := range append(append([]WorkItem{}, state.CompletedWork...), state.PendingWork...) {
		if err := item.Validate(); err != nil {
			return err
		}
		if previous, exists := seen[item.ID]; exists {
			return fmt.Errorf("work item %q appears in both %s and current state", item.ID, previous)
		}
		if containsWorkItem(state.CompletedWork, item.ID) {
			seen[item.ID] = "completed work"
		} else {
			seen[item.ID] = "pending work"
		}
	}
	for _, decision := range state.Decisions {
		if err := decision.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range state.EvidenceReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range state.CounterEvidenceReferences {
		if err := reference.ValidateCounterEvidence(); err != nil {
			return err
		}
	}
	for _, reference := range state.MemoryReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	if state.Checkpoint != nil {
		if err := state.Checkpoint.Validate(); err != nil {
			return err
		}
		if state.Checkpoint.TaskID != state.TaskID || state.Checkpoint.StateVersion > state.StateVersion {
			return fmt.Errorf("checkpoint reference does not belong to task state")
		}
	}
	return nil
}

func (state TaskState) NextActionList() []string {
	if strings.TrimSpace(state.NextAction) == "" {
		return nil
	}
	return []string{state.NextAction}
}

func stateFailureReasonList(state TaskState) []string {
	if strings.TrimSpace(state.Failure.Reason) == "" {
		return nil
	}
	return []string{state.Failure.Reason}
}

type ContextBudget struct {
	ContractVersion           string `json:"contract_version"`
	SystemPolicy              int64  `json:"system_policy"`
	Objective                 int64  `json:"objective"`
	TaskState                 int64  `json:"task_state"`
	Evidence                  int64  `json:"evidence"`
	CounterEvidence           int64  `json:"counter_evidence"`
	CounterEvidenceReserve    int64  `json:"counter_evidence_reserve"`
	Memory                    int64  `json:"memory"`
	ToolOutputs               int64  `json:"tool_outputs"`
	WorkingAllowance          int64  `json:"working_allowance"`
	StructuredOutputAllowance int64  `json:"structured_output_allowance"`
	Total                     int64  `json:"total"`
	RequireCounterEvidence    bool   `json:"require_counter_evidence"`
}

func (budget ContextBudget) Validate() error {
	if budget.ContractVersion != ContextBudgetContractV1 {
		return fmt.Errorf("unsupported context budget contract")
	}
	values := []int64{budget.SystemPolicy, budget.Objective, budget.TaskState, budget.Evidence, budget.CounterEvidence, budget.CounterEvidenceReserve, budget.Memory, budget.ToolOutputs, budget.WorkingAllowance, budget.StructuredOutputAllowance, budget.Total}
	for _, value := range values {
		if value < 0 {
			return fmt.Errorf("context budget cannot contain negative values")
		}
	}
	// CounterEvidenceReserve is a protected subset of CounterEvidence, not an
	// additional budget class and must not be double-counted in Total.
	sum, ok := safeSum([]int64{budget.SystemPolicy, budget.Objective, budget.TaskState, budget.Evidence, budget.CounterEvidence, budget.Memory, budget.ToolOutputs, budget.WorkingAllowance, budget.StructuredOutputAllowance})
	if !ok || sum != budget.Total {
		return fmt.Errorf("context budget total is inconsistent")
	}
	if budget.CounterEvidenceReserve > budget.CounterEvidence {
		return fmt.Errorf("counter-evidence reserve exceeds counter-evidence budget")
	}
	if budget.RequireCounterEvidence && (budget.CounterEvidence == 0 || budget.CounterEvidenceReserve == 0) {
		return fmt.Errorf("required counter-evidence must have a non-zero reserve")
	}
	return nil
}

type EvidenceReference struct {
	ContractVersion  string          `json:"contract_version"`
	EvidenceID       string          `json:"evidence_id"`
	CanonicalStore   string          `json:"canonical_store"`
	ProviderIdentity string          `json:"provider_identity"`
	SourceReference  string          `json:"source_reference"`
	ContentHash      string          `json:"content_hash,omitempty"`
	ObservedAt       *time.Time      `json:"observed_at,omitempty"`
	FirstSeenAt      *time.Time      `json:"first_seen_at,omitempty"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	Classification   string          `json:"classification"`
	QualityReference string          `json:"quality_reference"`
	PolicyReference  PolicyReference `json:"policy_reference"`
	InformationState string          `json:"information_state"`
}

func (reference EvidenceReference) Validate() error {
	if reference.ContractVersion != EvidenceReferenceContractV1 || !validIdentifier(reference.EvidenceID) || !validReference(reference.CanonicalStore) || !validReference(reference.ProviderIdentity) || !validReference(reference.SourceReference) || !validReference(reference.QualityReference) || !validReference(reference.InformationState) || !validOptionalCanonicalHash(reference.ContentHash) || !validOptionalUTCTime(reference.ObservedAt) || !validOptionalUTCTime(reference.FirstSeenAt) || !validOptionalUTCTime(reference.PublishedAt) || reference.PolicyReference.Validate() != nil {
		return fmt.Errorf("evidence reference is malformed")
	}
	switch reference.Classification {
	case EvidenceCandidate, EvidenceSupporting, EvidenceContradictory, EvidenceInvalidating, EvidenceUnknownMissing:
		return nil
	default:
		return fmt.Errorf("unsupported evidence classification %q", reference.Classification)
	}
}

func (reference EvidenceReference) ValidateCounterEvidence() error {
	if err := reference.Validate(); err != nil {
		return err
	}
	if reference.Classification != EvidenceContradictory && reference.Classification != EvidenceInvalidating && reference.Classification != EvidenceUnknownMissing {
		return fmt.Errorf("counter-evidence must be contradictory, invalidating, or unknown/missing")
	}
	return nil
}

type MemoryReference struct {
	ContractVersion         string    `json:"contract_version"`
	MemoryID                string    `json:"memory_id"`
	MemoryType              string    `json:"memory_type"`
	SourceTaskReference     string    `json:"source_task_reference"`
	SourceDecisionReference string    `json:"source_decision_reference"`
	SourceOutcomeReference  string    `json:"source_outcome_reference"`
	CreatedAt               time.Time `json:"created_at"`
	RegimeReference         string    `json:"regime_reference"`
	ReliabilityReference    string    `json:"reliability_reference"`
	CalibrationReference    string    `json:"calibration_reference"`
	ContentHash             string    `json:"content_hash"`
	Version                 string    `json:"version"`
}

func (reference MemoryReference) Validate() error {
	if reference.ContractVersion != MemoryReferenceContractV1 || !validIdentifier(reference.MemoryID) || !validReference(reference.MemoryType) || !validReference(reference.SourceTaskReference) || !validReference(reference.SourceDecisionReference) || !validReference(reference.SourceOutcomeReference) || !validUTCTime(reference.CreatedAt) || !validReference(reference.RegimeReference) || !validReference(reference.ReliabilityReference) || !validReference(reference.CalibrationReference) || !validCanonicalHash(reference.ContentHash) || !validVersion(reference.Version) {
		return fmt.Errorf("memory reference is malformed")
	}
	return nil
}

type ToolReference struct {
	ContractVersion            string     `json:"contract_version"`
	ToolInvocationID           string     `json:"tool_invocation_id"`
	ToolID                     string     `json:"tool_id"`
	ToolVersion                string     `json:"tool_version"`
	Purpose                    string     `json:"purpose"`
	InputHash                  string     `json:"input_hash"`
	InputReference             string     `json:"input_reference"`
	OutputHash                 string     `json:"output_hash,omitempty"`
	OutputReference            string     `json:"output_reference,omitempty"`
	StartedAt                  time.Time  `json:"started_at"`
	CompletedAt                *time.Time `json:"completed_at,omitempty"`
	Status                     string     `json:"status"`
	FailureClassification      string     `json:"failure_classification,omitempty"`
	CanonicalArtifactReference string     `json:"canonical_artifact_reference,omitempty"`
	Redacted                   bool       `json:"redacted"`
}

func (reference ToolReference) Validate() error {
	if reference.ContractVersion != ToolReferenceContractV1 || !validIdentifier(reference.ToolInvocationID) || !validReference(reference.ToolID) || !validReference(reference.ToolVersion) || !boundedText(reference.Purpose, 4096) || !validCanonicalHash(reference.InputHash) || !validReference(reference.InputReference) || !validOptionalCanonicalHash(reference.OutputHash) || !validOptionalReference(reference.OutputReference) || !validUTCTime(reference.StartedAt) || !validOptionalUTCTime(reference.CompletedAt) || !validOptionalReference(reference.CanonicalArtifactReference) || containsSecretMaterial(reference.InputReference) || containsSecretMaterial(reference.OutputReference) || containsSecretMaterial(reference.CanonicalArtifactReference) {
		return fmt.Errorf("tool reference is malformed or contains secret material")
	}
	switch reference.Status {
	case ToolStatusRequested:
		if reference.CompletedAt != nil {
			return fmt.Errorf("requested tool cannot have completion time")
		}
	case ToolStatusSucceeded:
		if reference.CompletedAt == nil || !validCanonicalHash(reference.OutputHash) || !validReference(reference.OutputReference) || reference.FailureClassification != "" {
			return fmt.Errorf("successful tool reference is incomplete")
		}
	case ToolStatusFailed:
		if reference.CompletedAt == nil || strings.TrimSpace(reference.FailureClassification) == "" {
			return fmt.Errorf("failed tool reference is incomplete")
		}
	case ToolStatusSkipped:
		if strings.TrimSpace(reference.FailureClassification) == "" {
			return fmt.Errorf("skipped tool reference requires classification")
		}
	default:
		return fmt.Errorf("unsupported tool status %q", reference.Status)
	}
	return nil
}

type CheckpointReference struct {
	ContractVersion string    `json:"contract_version"`
	CheckpointID    string    `json:"checkpoint_id"`
	TaskID          string    `json:"task_id"`
	StateVersion    int       `json:"state_version"`
	ContentHash     string    `json:"content_hash"`
	CreatedAt       time.Time `json:"created_at"`
}

func (reference CheckpointReference) Validate() error {
	if reference.ContractVersion != CheckpointContractV1 || !validIdentifier(reference.CheckpointID) || !validIdentifier(reference.TaskID) || reference.StateVersion < 1 || !validCanonicalHash(reference.ContentHash) || !validUTCTime(reference.CreatedAt) {
		return fmt.Errorf("checkpoint reference is malformed")
	}
	return nil
}

type Checkpoint struct {
	ContractVersion          string              `json:"contract_version"`
	CheckpointID             string              `json:"checkpoint_id"`
	Objective                ObjectiveReference  `json:"objective"`
	TaskState                TaskStateReference  `json:"task_state"`
	CurrentStage             string              `json:"current_stage"`
	ImportantDecisions       []DecisionRecord    `json:"important_decisions"`
	UnresolvedContradictions []string            `json:"unresolved_contradictions"`
	OpenQuestions            []string            `json:"open_questions"`
	KnownUnknowns            []string            `json:"known_unknowns"`
	EvidenceReferences       []EvidenceReference `json:"evidence_references"`
	NextAction               string              `json:"next_action"`
	Failure                  FailureState        `json:"failure"`
	Provenance               ContextProvenance   `json:"provenance"`
	ContentHash              string              `json:"content_hash"`
	CreatedAt                time.Time           `json:"created_at"`
}

func (checkpoint Checkpoint) Validate() error {
	if checkpoint.ContractVersion != CheckpointContractV1 || !validIdentifier(checkpoint.CheckpointID) || checkpoint.Objective.Validate() != nil || checkpoint.TaskState.Validate() != nil || strings.TrimSpace(checkpoint.CurrentStage) == "" || !validStringList(checkpoint.UnresolvedContradictions, 128, 4096) || !validStringList(checkpoint.OpenQuestions, 128, 4096) || !validStringList(checkpoint.KnownUnknowns, 128, 4096) || !boundedText(checkpoint.NextAction, 4096) || !validUTCTime(checkpoint.CreatedAt) || checkpoint.Provenance.Validate() != nil || !validCanonicalHash(checkpoint.ContentHash) {
		return fmt.Errorf("checkpoint is incomplete or malformed")
	}
	if err := checkpoint.Failure.Validate(); err != nil {
		return err
	}
	for _, decision := range checkpoint.ImportantDecisions {
		if err := decision.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range checkpoint.EvidenceReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ContextProvenance struct {
	ContractVersion           string             `json:"contract_version"`
	Objective                 ObjectiveReference `json:"objective"`
	TaskState                 TaskStateReference `json:"task_state"`
	ContextPackageID          string             `json:"context_package_id"`
	ContextPackageHash        string             `json:"context_package_hash,omitempty"`
	SelectedEvidenceIDs       []string           `json:"selected_evidence_ids"`
	CounterEvidenceIDs        []string           `json:"counter_evidence_ids"`
	SelectedMemoryIDs         []string           `json:"selected_memory_ids"`
	ToolInvocationIDs         []string           `json:"tool_invocation_ids"`
	Model                     ModelReference     `json:"model"`
	PromptVersion             string             `json:"prompt_version"`
	PolicyVersions            []PolicyReference  `json:"policy_versions"`
	StructuredOutputReference *ArtifactReference `json:"structured_output_reference,omitempty"`
	EvaluationReference       *ArtifactReference `json:"evaluation_reference,omitempty"`
	AssemblyAlgorithmVersion  string             `json:"assembly_algorithm_version"`
	AssembledAt               time.Time          `json:"assembled_at"`
}

func (provenance ContextProvenance) Validate() error {
	if provenance.ContractVersion != ContextProvenanceContractV1 || provenance.Objective.Validate() != nil || provenance.TaskState.Validate() != nil || !validIdentifier(provenance.ContextPackageID) || !validOptionalCanonicalHash(provenance.ContextPackageHash) || !validStringList(provenance.SelectedEvidenceIDs, 512, 256) || !validStringList(provenance.CounterEvidenceIDs, 512, 256) || !validStringList(provenance.SelectedMemoryIDs, 512, 256) || !validStringList(provenance.ToolInvocationIDs, 512, 256) || provenance.Model.Validate() != nil || !validReference(provenance.PromptVersion) || !validPolicyReferences(provenance.PolicyVersions) || !validReference(provenance.AssemblyAlgorithmVersion) || !validUTCTime(provenance.AssembledAt) {
		return fmt.Errorf("context provenance is incomplete or malformed")
	}
	if provenance.StructuredOutputReference != nil {
		if err := provenance.StructuredOutputReference.Validate(); err != nil {
			return err
		}
	}
	if provenance.EvaluationReference != nil {
		if err := provenance.EvaluationReference.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type ContextPackageAudit struct {
	BuilderVersion      string    `json:"builder_version"`
	OmittedEvidenceIDs  []string  `json:"omitted_evidence_ids"`
	DeferredEvidenceIDs []string  `json:"deferred_evidence_ids"`
	OmittedMemoryIDs    []string  `json:"omitted_memory_ids"`
	SelectionReasons    []string  `json:"selection_reasons"`
	AssembledAt         time.Time `json:"assembled_at"`
}

func (audit ContextPackageAudit) Validate() error {
	if !validReference(audit.BuilderVersion) || !validStringList(audit.OmittedEvidenceIDs, 512, 256) || !validStringList(audit.DeferredEvidenceIDs, 512, 256) || !validStringList(audit.OmittedMemoryIDs, 512, 256) || !validStringList(audit.SelectionReasons, 2048, 4096) || !validUTCTime(audit.AssembledAt) {
		return fmt.Errorf("context package audit metadata is malformed")
	}
	return nil
}

type ContextPackage struct {
	ContractVersion     string               `json:"contract_version"`
	ContextPackageID    string               `json:"context_package_id"`
	TaskID              string               `json:"task_id"`
	TaskStateVersion    int                  `json:"task_state_version"`
	ObjectiveID         string               `json:"objective_id"`
	ObjectiveVersion    string               `json:"objective_version"`
	CreatedAt           time.Time            `json:"created_at"`
	ModelPurpose        string               `json:"model_purpose"`
	TokenBudget         ContextBudget        `json:"token_budget"`
	Constraints         []string             `json:"constraints"`
	PolicyVersions      []PolicyReference    `json:"policy_versions"`
	SelectedEvidence    []EvidenceReference  `json:"selected_evidence"`
	CounterEvidence     []EvidenceReference  `json:"counter_evidence"`
	SelectedMemory      []MemoryReference    `json:"selected_memory"`
	CurrentState        TaskState            `json:"current_state"`
	OpenQuestions       []string             `json:"open_questions"`
	KnownUnknowns       []string             `json:"known_unknowns"`
	Assumptions         []string             `json:"assumptions"`
	ToolReferences      []ToolReference      `json:"tool_references"`
	CheckpointReference *CheckpointReference `json:"checkpoint_reference,omitempty"`
	Provenance          ContextProvenance    `json:"provenance"`
	Audit               ContextPackageAudit  `json:"audit"`
	ContentHash         string               `json:"content_hash"`
}

func NewContextPackage(pkg ContextPackage) (ContextPackage, error) {
	if pkg.ContractVersion == "" {
		pkg.ContractVersion = ContextPackageContractV1
	}
	hash, err := pkg.ComputeContentHash()
	if err != nil {
		return ContextPackage{}, err
	}
	pkg.ContentHash = hash
	pkg.Provenance.ContextPackageID = pkg.ContextPackageID
	pkg.Provenance.ContextPackageHash = hash
	if err := pkg.Validate(); err != nil {
		return ContextPackage{}, err
	}
	return pkg, nil
}

func (pkg ContextPackage) Validate() error {
	if pkg.ContractVersion != ContextPackageContractV1 || !validIdentifier(pkg.ContextPackageID) || !validIdentifier(pkg.TaskID) || pkg.TaskStateVersion < 1 || !validIdentifier(pkg.ObjectiveID) || !validVersion(pkg.ObjectiveVersion) || !validUTCTime(pkg.CreatedAt) || !boundedText(pkg.ModelPurpose, 4096) || !validStringList(pkg.Constraints, 128, 4096) || !validPolicyReferences(pkg.PolicyVersions) || !validStringList(pkg.OpenQuestions, 128, 4096) || !validStringList(pkg.KnownUnknowns, 128, 4096) || !validStringList(pkg.Assumptions, 128, 4096) || !validCanonicalHash(pkg.ContentHash) || pkg.TokenBudget.Validate() != nil || pkg.CurrentState.Validate() != nil || pkg.Provenance.Validate() != nil || pkg.Audit.Validate() != nil {
		return fmt.Errorf("context package is incomplete or malformed")
	}
	if pkg.CurrentState.TaskID != pkg.TaskID || pkg.CurrentState.StateVersion != pkg.TaskStateVersion || pkg.CurrentState.ObjectiveID != pkg.ObjectiveID || pkg.CurrentState.ObjectiveVersion != pkg.ObjectiveVersion {
		return fmt.Errorf("context package identity does not match current state")
	}
	for _, reference := range pkg.SelectedEvidence {
		if err := reference.Validate(); err != nil {
			return err
		}
		if reference.Classification != EvidenceCandidate && reference.Classification != EvidenceSupporting {
			return fmt.Errorf("selected evidence must be candidate or supporting")
		}
	}
	for _, reference := range pkg.CounterEvidence {
		if err := reference.ValidateCounterEvidence(); err != nil {
			return err
		}
	}
	for _, reference := range pkg.SelectedMemory {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range pkg.ToolReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	if pkg.CheckpointReference != nil {
		if err := pkg.CheckpointReference.Validate(); err != nil {
			return err
		}
	}
	expected, err := pkg.ComputeContentHash()
	if err != nil || expected != pkg.ContentHash {
		return fmt.Errorf("context package content hash mismatch")
	}
	if pkg.Provenance.ContextPackageID != pkg.ContextPackageID || (pkg.Provenance.ContextPackageHash != "" && pkg.Provenance.ContextPackageHash != pkg.ContentHash) {
		return fmt.Errorf("context provenance does not match package identity")
	}
	return nil
}

func (pkg ContextPackage) ComputeContentHash() (string, error) {
	input := canonicalContextPackage(pkg)
	return canonicalHash(input)
}

type canonicalContextPackageInput struct {
	ContractVersion     string               `json:"contract_version"`
	TaskID              string               `json:"task_id"`
	TaskStateVersion    int                  `json:"task_state_version"`
	ObjectiveID         string               `json:"objective_id"`
	ObjectiveVersion    string               `json:"objective_version"`
	ModelPurpose        string               `json:"model_purpose"`
	TokenBudget         ContextBudget        `json:"token_budget"`
	Constraints         []string             `json:"constraints"`
	PolicyVersions      []PolicyReference    `json:"policy_versions"`
	SelectedEvidence    []EvidenceReference  `json:"selected_evidence"`
	CounterEvidence     []EvidenceReference  `json:"counter_evidence"`
	SelectedMemory      []MemoryReference    `json:"selected_memory"`
	CurrentState        TaskState            `json:"current_state"`
	OpenQuestions       []string             `json:"open_questions"`
	KnownUnknowns       []string             `json:"known_unknowns"`
	Assumptions         []string             `json:"assumptions"`
	ToolReferences      []ToolReference      `json:"tool_references"`
	CheckpointReference *CheckpointReference `json:"checkpoint_reference,omitempty"`
}

func canonicalContextPackage(pkg ContextPackage) canonicalContextPackageInput {
	return canonicalContextPackageInput{
		ContractVersion:     pkg.ContractVersion,
		TaskID:              pkg.TaskID,
		TaskStateVersion:    pkg.TaskStateVersion,
		ObjectiveID:         pkg.ObjectiveID,
		ObjectiveVersion:    pkg.ObjectiveVersion,
		ModelPurpose:        pkg.ModelPurpose,
		TokenBudget:         pkg.TokenBudget,
		Constraints:         sortedStrings(pkg.Constraints),
		PolicyVersions:      sortedPolicies(pkg.PolicyVersions),
		SelectedEvidence:    sortedEvidence(pkg.SelectedEvidence),
		CounterEvidence:     sortedEvidence(pkg.CounterEvidence),
		SelectedMemory:      sortedMemory(pkg.SelectedMemory),
		CurrentState:        canonicalTaskState(pkg.CurrentState),
		OpenQuestions:       sortedStrings(pkg.OpenQuestions),
		KnownUnknowns:       sortedStrings(pkg.KnownUnknowns),
		Assumptions:         sortedStrings(pkg.Assumptions),
		ToolReferences:      sortedTools(pkg.ToolReferences),
		CheckpointReference: pkg.CheckpointReference,
	}
}

type ResearchClaim struct {
	ClaimID            string   `json:"claim_id"`
	Text               string   `json:"text"`
	EvidenceIDs        []string `json:"evidence_ids"`
	CounterEvidenceIDs []string `json:"counter_evidence_ids"`
	CausalReasoning    string   `json:"causal_reasoning"`
}

// EvaluatorInput is the immutable reference bundle supplied to an independent
// evaluator. It carries no execution authority and does not run evaluation.
type EvaluatorInput struct {
	ContractVersion    string             `json:"contract_version"`
	InputID            string             `json:"input_id"`
	Objective          ObjectiveReference `json:"objective"`
	TaskState          TaskStateReference `json:"task_state"`
	ContextPackage     ArtifactReference  `json:"context_package"`
	StructuredOutput   ArtifactReference  `json:"structured_output"`
	ContextProvenance  ContextProvenance  `json:"context_provenance"`
	CreatedAt          time.Time          `json:"created_at"`
	ExecutionAuthority string             `json:"execution_authority"`
}

func (input EvaluatorInput) Validate() error {
	if input.ContractVersion != EvaluatorInputContractV1 || !validIdentifier(input.InputID) || input.Objective.Validate() != nil || input.TaskState.Validate() != nil || input.ContextPackage.Validate() != nil || input.StructuredOutput.Validate() != nil || input.ContextProvenance.Validate() != nil || !validUTCTime(input.CreatedAt) || input.ExecutionAuthority != "NONE" {
		return fmt.Errorf("evaluator input is incomplete or execution-capable")
	}
	if input.ContextPackage.ReferenceID != input.ContextProvenance.ContextPackageID || input.TaskState != input.ContextProvenance.TaskState || input.Objective != input.ContextProvenance.Objective {
		return fmt.Errorf("evaluator input references do not agree")
	}
	return nil
}

// ToolInvocationRecord is the durable invocation/result reference contract.
// The alias preserves one validation boundary while allowing later audit
// layers to use the more explicit record terminology.
type ToolInvocationRecord = ToolReference

// CompactionRecord records what a future compaction operation retained. It is
// not the canonical task state, evidence store, or compaction algorithm.
type CompactionRecord struct {
	ContractVersion          string             `json:"contract_version"`
	CompactionID             string             `json:"compaction_id"`
	TaskStateBefore          TaskStateReference `json:"task_state_before"`
	TaskStateAfter           TaskStateReference `json:"task_state_after"`
	RetainedEvidenceIDs      []string           `json:"retained_evidence_ids"`
	RetainedContradictionIDs []string           `json:"retained_contradiction_ids"`
	RetainedOpenQuestions    []string           `json:"retained_open_questions"`
	RetainedKnownUnknowns    []string           `json:"retained_known_unknowns"`
	RetainedDecisionIDs      []string           `json:"retained_decision_ids"`
	NextAction               string             `json:"next_action"`
	Provenance               ContextProvenance  `json:"provenance"`
	AlgorithmVersion         string             `json:"algorithm_version"`
	CreatedAt                time.Time          `json:"created_at"`
	ContentHash              string             `json:"content_hash"`
	CanonicalStorage         bool               `json:"canonical_storage"`
}

func (record CompactionRecord) Validate() error {
	if record.ContractVersion != CompactionRecordContractV1 || !validIdentifier(record.CompactionID) || record.TaskStateBefore.Validate() != nil || record.TaskStateAfter.Validate() != nil || record.TaskStateAfter.StateVersion <= record.TaskStateBefore.StateVersion || !validStringList(record.RetainedEvidenceIDs, 512, 256) || !validStringList(record.RetainedContradictionIDs, 512, 256) || !validStringList(record.RetainedOpenQuestions, 128, 4096) || !validStringList(record.RetainedKnownUnknowns, 128, 4096) || !validStringList(record.RetainedDecisionIDs, 256, 256) || !boundedText(record.NextAction, 4096) || record.Provenance.Validate() != nil || !validReference(record.AlgorithmVersion) || !validUTCTime(record.CreatedAt) || !validCanonicalHash(record.ContentHash) || record.CanonicalStorage {
		return fmt.Errorf("compaction record is incomplete, canonical, or malformed")
	}
	if record.TaskStateAfter.TaskID != record.TaskStateBefore.TaskID || record.Provenance.TaskState != record.TaskStateAfter {
		return fmt.Errorf("compaction record task-state references do not agree")
	}
	return nil
}

func (claim ResearchClaim) Validate() error {
	if !validIdentifier(claim.ClaimID) || !boundedText(claim.Text, 8192) || !validStringList(claim.EvidenceIDs, 128, 256) || !validStringList(claim.CounterEvidenceIDs, 128, 256) || !boundedText(claim.CausalReasoning, 8192) {
		return fmt.Errorf("research claim is malformed")
	}
	return nil
}

type StructuredResearchOutput struct {
	ContractVersion           string              `json:"contract_version"`
	OutputID                  string              `json:"output_id"`
	TaskID                    string              `json:"task_id"`
	ContextPackageID          string              `json:"context_package_id"`
	Answer                    string              `json:"answer"`
	Claims                    []ResearchClaim     `json:"claims"`
	EvidenceReferences        []EvidenceReference `json:"evidence_references"`
	CounterEvidenceReferences []EvidenceReference `json:"counter_evidence_references"`
	Assumptions               []string            `json:"assumptions"`
	KnownUnknowns             []string            `json:"known_unknowns"`
	Confidence                *float64            `json:"confidence,omitempty"`
	Uncertainty               string              `json:"uncertainty"`
	InvalidationConditions    []string            `json:"invalidation_conditions"`
	RecommendedNextAction     string              `json:"recommended_next_research_action"`
	CompletionStatus          string              `json:"completion_status"`
}

func (output StructuredResearchOutput) Validate() error {
	if output.ContractVersion != StructuredResearchOutputContractV1 || !validIdentifier(output.OutputID) || !validIdentifier(output.TaskID) || !validIdentifier(output.ContextPackageID) || !boundedText(output.Answer, 32768) || !validStringList(output.Assumptions, 128, 4096) || !validStringList(output.KnownUnknowns, 128, 4096) || !boundedText(output.Uncertainty, 8192) || !validStringList(output.InvalidationConditions, 128, 4096) || !boundedText(output.RecommendedNextAction, 4096) {
		return fmt.Errorf("structured research output is incomplete or malformed")
	}
	if output.Confidence != nil && (*output.Confidence < 0 || *output.Confidence > 1 || math.IsNaN(*output.Confidence) || math.IsInf(*output.Confidence, 0)) {
		return fmt.Errorf("structured output confidence is invalid")
	}
	switch output.CompletionStatus {
	case OutputStatusDraft, OutputStatusComplete, OutputStatusIncomplete, OutputStatusAbstain:
	default:
		return fmt.Errorf("unsupported structured output completion status %q", output.CompletionStatus)
	}
	for _, claim := range output.Claims {
		if err := claim.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range output.EvidenceReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	for _, reference := range output.CounterEvidenceReferences {
		if err := reference.ValidateCounterEvidence(); err != nil {
			return err
		}
	}
	return nil
}

type EvaluatorChecks struct {
	ObjectiveSatisfied          bool     `json:"objective_satisfied"`
	EvidenceSufficient          bool     `json:"evidence_sufficient"`
	EvidenceQualityAcceptable   bool     `json:"evidence_quality_acceptable"`
	ContradictionsAddressed     bool     `json:"contradictions_addressed"`
	CausalChainSupported        bool     `json:"causal_chain_supported"`
	UnsupportedClaims           []string `json:"unsupported_claims"`
	ConfidenceSupported         bool     `json:"confidence_supported"`
	UncertaintyRepresented      bool     `json:"uncertainty_represented"`
	InvalidationDefined         bool     `json:"invalidation_defined"`
	PolicyCompliant             bool     `json:"policy_compliant"`
	PrematureCompletionDetected bool     `json:"premature_completion_detected"`
}

type EvaluatorReason struct {
	Code       string `json:"code"`
	Detail     string `json:"detail"`
	Correction string `json:"correction,omitempty"`
}

func (reason EvaluatorReason) Validate() error {
	if !validIdentifier(reason.Code) || !boundedText(reason.Detail, 8192) || !boundedText(reason.Correction, 8192) {
		return fmt.Errorf("evaluator reason is malformed")
	}
	return nil
}

type EvaluatorResult struct {
	ContractVersion     string            `json:"contract_version"`
	ResultID            string            `json:"result_id"`
	TaskID              string            `json:"task_id"`
	ContextPackageID    string            `json:"context_package_id"`
	Decision            string            `json:"decision"`
	Checks              EvaluatorChecks   `json:"checks"`
	Reasons             []EvaluatorReason `json:"reasons"`
	RequiredCorrections []string          `json:"required_corrections"`
	EvaluatorVersion    string            `json:"evaluator_version"`
	EvaluatedAt         time.Time         `json:"evaluated_at"`
	ExecutionAuthority  string            `json:"execution_authority"`
}

func (result EvaluatorResult) Validate() error {
	if result.ContractVersion != EvaluatorResultContractV1 || !validIdentifier(result.ResultID) || !validIdentifier(result.TaskID) || !validIdentifier(result.ContextPackageID) || !validReference(result.EvaluatorVersion) || !validUTCTime(result.EvaluatedAt) || result.ExecutionAuthority != "NONE" || !validStringList(result.Checks.UnsupportedClaims, 128, 4096) || !validStringList(result.RequiredCorrections, 128, 8192) {
		return fmt.Errorf("evaluator result is incomplete or execution-capable")
	}
	switch result.Decision {
	case EvaluatorPass:
		if !result.Checks.ObjectiveSatisfied || !result.Checks.EvidenceSufficient || !result.Checks.EvidenceQualityAcceptable || !result.Checks.ContradictionsAddressed || !result.Checks.CausalChainSupported || len(result.Checks.UnsupportedClaims) != 0 || !result.Checks.ConfidenceSupported || !result.Checks.UncertaintyRepresented || !result.Checks.InvalidationDefined || !result.Checks.PolicyCompliant || result.Checks.PrematureCompletionDetected || len(result.RequiredCorrections) != 0 {
			return fmt.Errorf("evaluator PASS does not satisfy all research checks")
		}
	case EvaluatorRevise, EvaluatorAbstain:
		if len(result.Reasons) == 0 {
			return fmt.Errorf("evaluator %s requires structured reasons", result.Decision)
		}
	default:
		return fmt.Errorf("unsupported evaluator decision %q", result.Decision)
	}
	for _, reason := range result.Reasons {
		if err := reason.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type TokenUsage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

func (usage TokenUsage) Validate() error {
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.ReasoningTokens < 0 {
		return fmt.Errorf("token usage cannot be negative")
	}
	return nil
}

type HarnessRun struct {
	ContractVersion    string             `json:"contract_version"`
	HarnessRunID       string             `json:"harness_run_id"`
	TaskID             string             `json:"task_id"`
	Objective          ObjectiveReference `json:"objective"`
	ContextPackage     ArtifactReference  `json:"context_package"`
	Model              ModelReference     `json:"model"`
	StartedAt          time.Time          `json:"started_at"`
	EndedAt            *time.Time         `json:"ended_at,omitempty"`
	ToolReferences     []ToolReference    `json:"tool_references"`
	StructuredOutput   *ArtifactReference `json:"structured_output,omitempty"`
	EvaluatorResult    *ArtifactReference `json:"evaluator_result,omitempty"`
	RevisionCount      int                `json:"revision_count"`
	TokenUsage         TokenUsage         `json:"token_usage"`
	LatencyMillis      int64              `json:"latency_millis"`
	CostUSD            *float64           `json:"cost_usd,omitempty"`
	Status             string             `json:"status"`
	Failure            *FailureState      `json:"failure,omitempty"`
	ExecutionAuthority string             `json:"execution_authority"`
}

func (run HarnessRun) Validate() error {
	if run.ContractVersion != HarnessRunContractV1 || !validIdentifier(run.HarnessRunID) || !validIdentifier(run.TaskID) || run.Objective.Validate() != nil || run.ContextPackage.Validate() != nil || run.Model.Validate() != nil || !validUTCTime(run.StartedAt) || !validOptionalUTCTime(run.EndedAt) || run.RevisionCount < 0 || run.LatencyMillis < 0 || run.ExecutionAuthority != "NONE" || run.TokenUsage.Validate() != nil {
		return fmt.Errorf("harness run is incomplete or execution-capable")
	}
	if run.EndedAt != nil && run.EndedAt.Before(run.StartedAt) {
		return fmt.Errorf("harness run ended before it started")
	}
	if run.CostUSD != nil && (*run.CostUSD < 0 || math.IsNaN(*run.CostUSD) || math.IsInf(*run.CostUSD, 0)) {
		return fmt.Errorf("harness run cost is invalid")
	}
	for _, reference := range run.ToolReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	switch run.Status {
	case HarnessRunRunning, HarnessRunSucceeded, HarnessRunFailed, HarnessRunAbstained:
	default:
		return fmt.Errorf("unsupported harness run status %q", run.Status)
	}
	if (run.Status == HarnessRunFailed) != (run.Failure != nil) {
		return fmt.Errorf("harness run failure state does not match status")
	}
	if run.Failure != nil {
		if err := run.Failure.Validate(); err != nil {
			return err
		}
	}
	if run.StructuredOutput != nil {
		if err := run.StructuredOutput.Validate(); err != nil {
			return err
		}
	}
	if run.EvaluatorResult != nil {
		if err := run.EvaluatorResult.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type EngineeringObjective struct {
	ContractVersion    string    `json:"contract_version"`
	ObjectiveID        string    `json:"objective_id"`
	Version            string    `json:"version"`
	Purpose            string    `json:"purpose"`
	Request            string    `json:"request"`
	Scope              []string  `json:"scope"`
	Exclusions         []string  `json:"exclusions"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	CreatedAt          time.Time `json:"created_at"`
	Status             string    `json:"status"`
}

func (objective EngineeringObjective) Validate() error {
	if objective.ContractVersion != EngineeringObjectiveContractV1 || !validIdentifier(objective.ObjectiveID) || !validVersion(objective.Version) || !boundedText(objective.Purpose, 4096) || !boundedText(objective.Request, 16384) || !validStringList(objective.Scope, 128, 4096) || !validStringList(objective.Exclusions, 128, 4096) || len(objective.AcceptanceCriteria) == 0 || !validStringList(objective.AcceptanceCriteria, 128, 4096) || !validUTCTime(objective.CreatedAt) {
		return fmt.Errorf("engineering objective is incomplete or malformed")
	}
	if objective.Status != ObjectiveStatusDraft && objective.Status != ObjectiveStatusActive && objective.Status != ObjectiveStatusCompleted && objective.Status != ObjectiveStatusAbandoned {
		return fmt.Errorf("unsupported engineering objective status %q", objective.Status)
	}
	return nil
}

type RepositoryState struct {
	ContractVersion           string    `json:"contract_version"`
	RepositoryHeadSHA         string    `json:"repository_head_sha"`
	StartingSHA               string    `json:"starting_sha"`
	FrozenExperimentalCodeSHA string    `json:"frozen_experimental_code_sha,omitempty"`
	Branch                    string    `json:"branch"`
	WorktreeStatus            string    `json:"worktree_status"`
	CurrentPackageReference   string    `json:"current_package_reference"`
	CapturedAt                time.Time `json:"captured_at"`
}

func (state RepositoryState) Validate() error {
	if state.ContractVersion != RepositoryStateContractV1 || !validSHA(state.RepositoryHeadSHA) || !validSHA(state.StartingSHA) || !validOptionalSHA(state.FrozenExperimentalCodeSHA) || !validReference(state.Branch) || !validReference(state.CurrentPackageReference) || !validUTCTime(state.CapturedAt) {
		return fmt.Errorf("repository state is incomplete or malformed")
	}
	switch state.WorktreeStatus {
	case RepositoryWorktreeClean, RepositoryWorktreeDirty, RepositoryWorktreeUnknown:
		return nil
	default:
		return fmt.Errorf("unsupported repository worktree status %q", state.WorktreeStatus)
	}
}

type DocumentReference struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

func (reference DocumentReference) Validate() error {
	if !validDocumentPath(reference.Path) || !validReference(reference.Role) {
		return fmt.Errorf("document reference is malformed")
	}
	return nil
}

type DocumentationMap struct {
	ContractVersion             string              `json:"contract_version"`
	AuthorityChain              []DocumentReference `json:"authority_chain"`
	SelectedDocuments           []DocumentReference `json:"selected_documents"`
	ExcludedHistoricalDocuments []DocumentReference `json:"excluded_historical_documents"`
	CurrentPackageReference     string              `json:"current_package_reference"`
	GeneratedAt                 time.Time           `json:"generated_at"`
}

func (mapping DocumentationMap) Validate() error {
	if mapping.ContractVersion != DocumentationMapContractV1 || len(mapping.AuthorityChain) == 0 || len(mapping.AuthorityChain) > 64 || len(mapping.SelectedDocuments) == 0 || len(mapping.SelectedDocuments) > 256 || !validReference(mapping.CurrentPackageReference) || !validUTCTime(mapping.GeneratedAt) {
		return fmt.Errorf("documentation map is incomplete or malformed")
	}
	for _, references := range [][]DocumentReference{mapping.AuthorityChain, mapping.SelectedDocuments, mapping.ExcludedHistoricalDocuments} {
		for _, reference := range references {
			if err := reference.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

type VerificationCommand struct {
	VerificationID          string   `json:"verification_id"`
	Command                 string   `json:"command"`
	Purpose                 string   `json:"purpose"`
	ApplicableScopes        []string `json:"applicable_scopes"`
	Required                bool     `json:"required"`
	EnvironmentRequirements []string `json:"environment_requirements"`
	ExpectedEvidence        []string `json:"expected_evidence"`
	TypicalCostClass        string   `json:"typical_cost_class"`
	KnownFailurePolicy      string   `json:"known_failure_policy"`
	TrustedSource           string   `json:"trusted_source"`
}

func (command VerificationCommand) Validate() error {
	if !validIdentifier(command.VerificationID) || !boundedText(command.Command, 4096) || !boundedText(command.Purpose, 4096) || !validStringList(command.ApplicableScopes, 32, 256) || !validStringList(command.EnvironmentRequirements, 64, 1024) || !validStringList(command.ExpectedEvidence, 64, 1024) || !validReference(command.TypicalCostClass) || !boundedText(command.KnownFailurePolicy, 4096) || command.TrustedSource != "repository-policy" {
		return fmt.Errorf("verification command is incomplete or untrusted")
	}
	if strings.ContainsAny(command.Command, "\r\n") {
		return fmt.Errorf("verification command cannot contain newlines")
	}
	return nil
}

type VerificationPlan struct {
	ContractVersion string                `json:"contract_version"`
	PlanID          string                `json:"plan_id"`
	Version         string                `json:"version"`
	Commands        []VerificationCommand `json:"commands"`
	CreatedAt       time.Time             `json:"created_at"`
}

func (plan VerificationPlan) Validate() error {
	if plan.ContractVersion != VerificationPlanContractV1 || !validIdentifier(plan.PlanID) || !validVersion(plan.Version) || len(plan.Commands) == 0 || len(plan.Commands) > 64 || !validUTCTime(plan.CreatedAt) {
		return fmt.Errorf("verification plan is incomplete or malformed")
	}
	seen := map[string]struct{}{}
	for _, command := range plan.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
		if _, exists := seen[command.VerificationID]; exists {
			return fmt.Errorf("verification plan contains duplicate command")
		}
		seen[command.VerificationID] = struct{}{}
	}
	return nil
}

func DefaultVerificationRegistry() VerificationPlan {
	return VerificationPlan{
		ContractVersion: VerificationPlanContractV1,
		PlanID:          "jax-repository-verification",
		Version:         "v1",
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Commands: []VerificationCommand{
			{VerificationID: "git-diff-check", Command: "git diff --check", Purpose: "Detect whitespace errors in the scoped diff.", ApplicableScopes: []string{"docs", "go", "all"}, Required: true, ExpectedEvidence: []string{"clean diff-check exit"}, TypicalCostClass: "low", KnownFailurePolicy: "Fail the package until corrected.", TrustedSource: "repository-policy"},
			{VerificationID: "go-test-all", Command: "go test ./...", Purpose: "Run the repository Go test suite.", ApplicableScopes: []string{"go", "contracts", "all"}, Required: true, ExpectedEvidence: []string{"Go test output"}, TypicalCostClass: "high", KnownFailurePolicy: "Report unrelated pre-existing failures without changing unrelated runtime code.", TrustedSource: "repository-policy"},
			{VerificationID: "go-verify-standard", Command: ".\\scripts\\go-verify.ps1 -Mode standard", Purpose: "Run repository standard Go formatting and test verification.", ApplicableScopes: []string{"go", "contracts"}, Required: false, ExpectedEvidence: []string{"standard verification output"}, TypicalCostClass: "high", KnownFailurePolicy: "Report and investigate failures.", TrustedSource: "repository-policy"},
			{VerificationID: "go-verify-full", Command: ".\\scripts\\go-verify.ps1 -Mode full", Purpose: "Run full repository Go verification.", ApplicableScopes: []string{"core-contracts", "cross-boundary", "all"}, Required: false, ExpectedEvidence: []string{"full verification output"}, TypicalCostClass: "very-high", KnownFailurePolicy: "Do not hide failures.", TrustedSource: "repository-policy"},
			{VerificationID: "trader-import-boundary", Command: ".\\scripts\\check-trader-imports.ps1", Purpose: "Check the cmd/trader research import boundary.", ApplicableScopes: []string{"architecture", "go", "all"}, Required: true, ExpectedEvidence: []string{"boundary check output"}, TypicalCostClass: "medium", KnownFailurePolicy: "Stop on a boundary violation.", TrustedSource: "repository-policy"},
			{VerificationID: "golden-check", Command: ".\\scripts\\golden-check.ps1 -Mode verify", Purpose: "Run the repository golden/replay verification.", ApplicableScopes: []string{"behavior-sensitive", "all"}, Required: false, ExpectedEvidence: []string{"golden verification output"}, TypicalCostClass: "high", KnownFailurePolicy: "Run when behavior-sensitive scope is changed; report failures.", TrustedSource: "repository-policy"},
		},
	}
}

type EngineeringCheckpoint struct {
	ContractVersion   string               `json:"contract_version"`
	CheckpointID      string               `json:"checkpoint_id"`
	Objective         EngineeringObjective `json:"objective"`
	Repository        RepositoryState      `json:"repository"`
	Documentation     DocumentationMap     `json:"documentation"`
	Verification      VerificationPlan     `json:"verification"`
	CompletedWork     []string             `json:"completed_work"`
	PendingWork       []string             `json:"pending_work"`
	Decisions         []DecisionRecord     `json:"decisions"`
	KnownFailures     []string             `json:"known_failures"`
	NextAction        string               `json:"next_action"`
	CheckpointVersion int                  `json:"checkpoint_version"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

func (checkpoint EngineeringCheckpoint) Validate() error {
	if checkpoint.ContractVersion != EngineeringCheckpointContractV1 || !validIdentifier(checkpoint.CheckpointID) || checkpoint.Objective.Validate() != nil || checkpoint.Repository.Validate() != nil || checkpoint.Documentation.Validate() != nil || checkpoint.Verification.Validate() != nil || !validStringList(checkpoint.CompletedWork, 256, 4096) || !validStringList(checkpoint.PendingWork, 256, 4096) || !validStringList(checkpoint.KnownFailures, 128, 8192) || !boundedText(checkpoint.NextAction, 4096) || checkpoint.CheckpointVersion < 1 || !validUTCTime(checkpoint.CreatedAt) || !validUTCTime(checkpoint.UpdatedAt) || checkpoint.UpdatedAt.Before(checkpoint.CreatedAt) {
		return fmt.Errorf("engineering checkpoint is incomplete or malformed")
	}
	for _, decision := range checkpoint.Decisions {
		if err := decision.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type CheckRecord struct {
	CheckID           string `json:"check_id"`
	Status            string `json:"status"`
	EvidenceReference string `json:"evidence_reference"`
}

func (record CheckRecord) Validate() error {
	if !validIdentifier(record.CheckID) || !validReference(record.EvidenceReference) {
		return fmt.Errorf("check record is malformed")
	}
	switch record.Status {
	case VerificationPass, VerificationFail, VerificationSkipped:
		return nil
	default:
		return fmt.Errorf("unsupported check status %q", record.Status)
	}
}

type SkippedCheck struct {
	CheckID string `json:"check_id"`
	Reason  string `json:"reason"`
}

func (check SkippedCheck) Validate() error {
	if !validIdentifier(check.CheckID) || !boundedText(check.Reason, 4096) {
		return fmt.Errorf("skipped check is malformed")
	}
	return nil
}

type CIResult struct {
	Workflow string `json:"workflow"`
	RunID    string `json:"run_id"`
	SHA      string `json:"sha"`
	Result   string `json:"result"`
}

func (result CIResult) Validate() error {
	if !boundedText(result.Workflow, 256) || !validIdentifier(result.RunID) || !validSHA(result.SHA) || result.Result != VerificationPass {
		return fmt.Errorf("CI result is malformed or unsuccessful")
	}
	return nil
}

type CleanExitReport struct {
	ContractVersion        string         `json:"contract_version"`
	ReportID               string         `json:"report_id"`
	StartingSHA            string         `json:"starting_sha"`
	EndingSHA              string         `json:"ending_sha"`
	Branch                 string         `json:"branch"`
	ExpectedDiffScope      []string       `json:"expected_diff_scope"`
	ActualDiffScope        []string       `json:"actual_diff_scope"`
	RuntimeIsolationStatus string         `json:"runtime_isolation_status"`
	Checks                 []CheckRecord  `json:"checks"`
	SkippedChecks          []SkippedCheck `json:"skipped_checks"`
	KnownUnrelatedFailures []string       `json:"known_unrelated_failures"`
	Commit                 string         `json:"commit"`
	PushStatus             string         `json:"push_status"`
	OriginEquality         bool           `json:"origin_equality"`
	DivergenceAhead        int            `json:"divergence_ahead"`
	DivergenceBehind       int            `json:"divergence_behind"`
	ExactSHACI             []CIResult     `json:"exact_sha_ci"`
	HandoverComplete       bool           `json:"handover_complete"`
	OpenBlockers           []string       `json:"open_blockers"`
	Result                 string         `json:"result"`
}

func (report CleanExitReport) Validate() error {
	if report.ContractVersion != CleanExitReportContractV1 || !validIdentifier(report.ReportID) || !validSHA(report.StartingSHA) || !validSHA(report.EndingSHA) || !validReference(report.Branch) || !validStringList(report.ExpectedDiffScope, 128, 256) || !validStringList(report.ActualDiffScope, 256, 256) || report.RuntimeIsolationStatus != VerificationPass || len(report.Checks) == 0 || !validReference(report.Commit) || report.PushStatus != "PUSHED" || report.DivergenceAhead < 0 || report.DivergenceBehind < 0 || !validStringList(report.KnownUnrelatedFailures, 128, 8192) || !validStringList(report.OpenBlockers, 128, 8192) {
		return fmt.Errorf("clean-exit report is incomplete")
	}
	switch report.Result {
	case CleanExitBlocked:
		return nil
	case CleanExitPass:
		if !report.OriginEquality || report.DivergenceAhead != 0 || report.DivergenceBehind != 0 || !report.HandoverComplete || len(report.OpenBlockers) != 0 || len(report.ExactSHACI) == 0 {
			return fmt.Errorf("successful clean exit is missing mandatory evidence")
		}
	default:
		return fmt.Errorf("unsupported clean-exit result %q", report.Result)
	}
	for _, check := range report.Checks {
		if err := check.Validate(); err != nil {
			return err
		}
		if report.Result == CleanExitPass && check.Status == VerificationFail {
			return fmt.Errorf("successful clean exit cannot contain a failed check")
		}
	}
	for _, check := range report.SkippedChecks {
		if err := check.Validate(); err != nil {
			return err
		}
	}
	for _, result := range report.ExactSHACI {
		if err := result.Validate(); err != nil {
			return err
		}
		if result.SHA != report.EndingSHA {
			return fmt.Errorf("exact-SHA CI result does not match ending SHA")
		}
	}
	return nil
}

type EngineeringHarnessRun struct {
	ContractVersion    string          `json:"contract_version"`
	RunID              string          `json:"run_id"`
	ObjectiveID        string          `json:"objective_id"`
	Repository         RepositoryState `json:"repository"`
	CheckpointID       string          `json:"checkpoint_id"`
	VerificationPlanID string          `json:"verification_plan_id"`
	CleanExitReportID  string          `json:"clean_exit_report_id,omitempty"`
	StartedAt          time.Time       `json:"started_at"`
	EndedAt            *time.Time      `json:"ended_at,omitempty"`
	Status             string          `json:"status"`
	Failure            *FailureState   `json:"failure,omitempty"`
}

func (run EngineeringHarnessRun) Validate() error {
	if run.ContractVersion != EngineeringHarnessRunContractV1 || !validIdentifier(run.RunID) || !validIdentifier(run.ObjectiveID) || run.Repository.Validate() != nil || !validIdentifier(run.CheckpointID) || !validIdentifier(run.VerificationPlanID) || !validOptionalIdentifier(run.CleanExitReportID) || !validUTCTime(run.StartedAt) || !validOptionalUTCTime(run.EndedAt) {
		return fmt.Errorf("engineering harness run is incomplete")
	}
	if run.EndedAt != nil && run.EndedAt.Before(run.StartedAt) {
		return fmt.Errorf("engineering harness run ended before it started")
	}
	switch run.Status {
	case HarnessRunRunning, HarnessRunSucceeded, HarnessRunFailed, HarnessRunAbstained:
	default:
		return fmt.Errorf("unsupported engineering harness run status %q", run.Status)
	}
	if (run.Status == HarnessRunFailed) != (run.Failure != nil) {
		return fmt.Errorf("engineering harness failure state does not match status")
	}
	if run.Failure != nil {
		return run.Failure.Validate()
	}
	return nil
}

func canonicalHash(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("canonical encode: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

// CanonicalHash returns the versioned hash representation used by contracts
// that need stable content identity. JSON object keys are ordered by the
// encoding/json package; callers must sort semantically unordered slices
// before calling it. Contract-specific canonicalizers do that explicitly.
func CanonicalHash(value any) (string, error) { return canonicalHash(value) }

// IsCanonicalHash reports whether value has the canonical SHA-256 form.
func IsCanonicalHash(value string) bool { return validCanonicalHash(value) }

func validCanonicalHash(value string) bool { return canonicalHashPattern.MatchString(value) }

func validOptionalCanonicalHash(value string) bool { return value == "" || validCanonicalHash(value) }

func validSHA(value string) bool { return shaPattern.MatchString(value) }

func validOptionalSHA(value string) bool { return value == "" || validSHA(value) }

func validIdentifier(value string) bool { return identifierPattern.MatchString(value) }

func validOptionalIdentifier(value string) bool { return value == "" || validIdentifier(value) }

func validVersion(value string) bool { return versionPattern.MatchString(value) }

func validReference(value string) bool {
	return validIdentifier(value) && !containsSecretMaterial(value)
}

func validOptionalReference(value string) bool { return value == "" || validReference(value) }

func validUTCTime(value time.Time) bool { return !value.IsZero() && value.Location() == time.UTC }

func validOptionalUTCTime(value *time.Time) bool { return value == nil || validUTCTime(*value) }

func boundedText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= max
}

func validStringList(values []string, maxItems, maxItemLength int) bool {
	if len(values) > maxItems {
		return false
	}
	for _, value := range values {
		if !boundedText(value, maxItemLength) {
			return false
		}
	}
	return true
}

func validPolicyReferences(values []PolicyReference) bool {
	if len(values) > 64 {
		return false
	}
	for _, value := range values {
		if value.Validate() != nil {
			return false
		}
	}
	return true
}

func validDocumentPath(path string) bool {
	if strings.Contains(path, "..") || strings.ContainsAny(path, "\r\n") {
		return false
	}
	return strings.HasPrefix(path, "Docs/") || strings.HasPrefix(path, "skills/") || path == "AGENTS.md"
}

func containsSecretMaterial(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api_key", "apikey", "password", "authorization", "cookie", "database_url", "dsn=", "broker_credential", "client_secret", "refresh_token", "token="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func safeSum(values []int64) (int64, bool) {
	var total int64
	for _, value := range values {
		if value < 0 || total > math.MaxInt64-value {
			return 0, false
		}
		total += value
	}
	return total, true
}

func containsWorkItem(items []WorkItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func sortedStrings(values []string) []string {
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return copyValues
}

func sortedPolicies(values []PolicyReference) []PolicyReference {
	copyValues := append([]PolicyReference(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool {
		return copyValues[i].Name+"/"+copyValues[i].Version < copyValues[j].Name+"/"+copyValues[j].Version
	})
	return copyValues
}

func sortedEvidence(values []EvidenceReference) []EvidenceReference {
	copyValues := append([]EvidenceReference(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i].EvidenceID < copyValues[j].EvidenceID })
	return copyValues
}

func sortedMemory(values []MemoryReference) []MemoryReference {
	copyValues := append([]MemoryReference(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i].MemoryID < copyValues[j].MemoryID })
	return copyValues
}

func sortedTools(values []ToolReference) []ToolReference {
	copyValues := append([]ToolReference(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i].ToolInvocationID < copyValues[j].ToolInvocationID })
	return copyValues
}

func canonicalTaskState(state TaskState) TaskState {
	state.OpenQuestions = sortedStrings(state.OpenQuestions)
	state.KnownUnknowns = sortedStrings(state.KnownUnknowns)
	state.Constraints = sortedStrings(state.Constraints)
	state.Decisions = append([]DecisionRecord(nil), state.Decisions...)
	sort.Slice(state.Decisions, func(i, j int) bool { return state.Decisions[i].DecisionID < state.Decisions[j].DecisionID })
	state.EvidenceReferences = sortedEvidence(state.EvidenceReferences)
	state.CounterEvidenceReferences = sortedEvidence(state.CounterEvidenceReferences)
	state.MemoryReferences = sortedMemory(state.MemoryReferences)
	return state
}
