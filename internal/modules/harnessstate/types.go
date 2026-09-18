// Package harnessstate owns the research-side durable task-state seam for
// HARNESS-03. It is deliberately independent from cmd/trader, PAPER-02, and
// the canonical evidence/memory stores: it persists references to those
// records, never their payloads.
package harnessstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/contextbuilder"
	"jax-trading-assistant/internal/modules/harnesscontracts"
)

const (
	ContractVersion         = "jax.harness.durable_task_state/v1"
	CheckpointRecordVersion = "jax.harness.durable_checkpoint/v1"
	CompactionRecordVersion = "jax.harness.structured_compaction/v1"
	FailureRecordVersion    = "jax.harness.failure_event/v1"
	RetryRecordVersion      = "jax.harness.retry_event/v1"
	CompactionAlgorithmV1   = "structured-compaction-v1"
)

var (
	ErrNotFound            = errors.New("harness state not found")
	ErrAlreadyExists       = errors.New("harness state already exists")
	ErrIntegrity           = errors.New("harness state integrity failure")
	ErrSecretMaterial      = errors.New("secret material is not permitted in harness state")
	ErrIdempotencyConflict = errors.New("harness idempotency key was reused with different content")
	ErrInvalid             = errors.New("invalid harness state")
	operationKeyPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
	secretKeyPattern       = regexp.MustCompile(`(?i)(api[_-]?key|authorization|cookie|password|secret|database[_-]?url|broker[_-]?(credential|token)|access[_-]?token|refresh[_-]?token)`)
	secretValuePattern     = regexp.MustCompile(`(?i)(api[_-]?key|authorization|cookie|password|secret|bearer)\s*[:=]|postgres(?:ql)?://[^\s]+:[^\s]+@`)
)

// VersionConflictError is returned when a writer used a stale expected
// version. Callers can inspect both versions and retry from the loaded state.
type VersionConflictError struct {
	TaskID          string
	ExpectedVersion int
	CurrentVersion  int
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("task %s version conflict: expected v%d, current v%d", e.TaskID, e.ExpectedVersion, e.CurrentVersion)
}

// DurableState adds the two pieces of resumability metadata that the original
// HARNESS-01 TaskState intentionally kept outside the retrieval contract.
// Everything else remains in harnesscontracts.TaskState.
type DurableState struct {
	State                    harnesscontracts.TaskState       `json:"state"`
	UnresolvedContradictions []string                         `json:"unresolved_contradictions,omitempty"`
	ToolReferences           []harnesscontracts.ToolReference `json:"tool_references,omitempty"`
}

type TaskRecord struct {
	ContractVersion string                             `json:"contract_version"`
	TaskID          string                             `json:"task_id"`
	Objective       harnesscontracts.ResearchObjective `json:"objective"`
	LatestVersion   int                                `json:"latest_version"`
	CreatedAt       time.Time                          `json:"created_at"`
	UpdatedAt       time.Time                          `json:"updated_at"`
}

type CreateTaskRequest struct {
	TaskID       string
	Objective    harnesscontracts.ResearchObjective `json:"objective"`
	InitialState DurableState                       `json:"initial_state"`
}

type AppendStateResult struct {
	State   DurableState `json:"state"`
	Applied bool         `json:"applied"`
}

type CheckpointContent struct {
	Objective                 harnesscontracts.ObjectiveReference  `json:"objective"`
	TaskState                 harnesscontracts.TaskStateReference  `json:"task_state"`
	CurrentStage              string                               `json:"current_stage"`
	CompletedWork             []harnesscontracts.WorkItem          `json:"completed_work"`
	PendingWork               []harnesscontracts.WorkItem          `json:"pending_work"`
	Constraints               []string                             `json:"constraints"`
	Decisions                 []harnesscontracts.DecisionRecord    `json:"decisions"`
	OpenQuestions             []string                             `json:"open_questions"`
	KnownUnknowns             []string                             `json:"known_unknowns"`
	EvidenceReferences        []harnesscontracts.EvidenceReference `json:"evidence_references"`
	CounterEvidenceReferences []harnesscontracts.EvidenceReference `json:"counter_evidence_references"`
	MemoryReferences          []harnesscontracts.MemoryReference   `json:"memory_references"`
	ToolReferences            []harnesscontracts.ToolReference     `json:"tool_references"`
	UnresolvedContradictions  []string                             `json:"unresolved_contradictions"`
	Failure                   harnesscontracts.FailureState        `json:"failure"`
	NextAction                string                               `json:"next_action"`
	ProvenanceReferences      []string                             `json:"provenance_references"`
}

type CheckpointRecord struct {
	ContractVersion string            `json:"contract_version"`
	CheckpointID    string            `json:"checkpoint_id"`
	TaskID          string            `json:"task_id"`
	StateVersion    int               `json:"state_version"`
	Content         CheckpointContent `json:"content"`
	ContentHash     string            `json:"content_hash"`
	CreatedAt       time.Time         `json:"created_at"`
}

type CreateCheckpointRequest struct {
	TaskID               string    `json:"task_id"`
	StateVersion         int       `json:"state_version"`
	CheckpointID         string    `json:"checkpoint_id,omitempty"`
	IdempotencyKey       string    `json:"idempotency_key"`
	CreatedAt            time.Time `json:"created_at"`
	ProvenanceReferences []string  `json:"provenance_references,omitempty"`
}

type CompactionRecord struct {
	ContractVersion    string            `json:"contract_version"`
	CompactionID       string            `json:"compaction_id"`
	TaskID             string            `json:"task_id"`
	SourceStateVersion int               `json:"source_state_version"`
	Content            CheckpointContent `json:"content"`
	AlgorithmVersion   string            `json:"algorithm_version"`
	CanonicalStorage   bool              `json:"canonical_storage"`
	ContentHash        string            `json:"content_hash"`
	CreatedAt          time.Time         `json:"created_at"`
}

type CompactRequest struct {
	TaskID               string    `json:"task_id"`
	StateVersion         int       `json:"state_version"`
	CompactionID         string    `json:"compaction_id,omitempty"`
	IdempotencyKey       string    `json:"idempotency_key"`
	AlgorithmVersion     string    `json:"algorithm_version,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	ProvenanceReferences []string  `json:"provenance_references,omitempty"`
}

type FailureEvent struct {
	ContractVersion string                        `json:"contract_version"`
	EventID         string                        `json:"event_id"`
	TaskID          string                        `json:"task_id"`
	StateVersion    int                           `json:"state_version"`
	Failure         harnesscontracts.FailureState `json:"failure"`
	CreatedAt       time.Time                     `json:"created_at"`
}

type RetryEvent struct {
	ContractVersion string    `json:"contract_version"`
	EventID         string    `json:"event_id"`
	TaskID          string    `json:"task_id"`
	StateVersion    int       `json:"state_version"`
	Attempt         int       `json:"attempt"`
	NextAction      string    `json:"next_action"`
	CreatedAt       time.Time `json:"created_at"`
}

type ResumeRequest struct {
	TaskID       string `json:"task_id"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
}

type ResumeBundle struct {
	ContractVersion string           `json:"contract_version"`
	Task            TaskRecord       `json:"task"`
	State           DurableState     `json:"state"`
	Checkpoint      CheckpointRecord `json:"checkpoint"`
	FailureHistory  []FailureEvent   `json:"failure_history"`
	RetryHistory    []RetryEvent     `json:"retry_history"`
	NextAction      string           `json:"next_action"`
}

// Store is the complete research-side persistence seam. Implementations must
// preserve historical state versions and make retry identities explicit.
type Store interface {
	CreateTask(context.Context, CreateTaskRequest) (TaskRecord, error)
	LoadTask(context.Context, string) (TaskRecord, error)
	AppendStateVersion(context.Context, string, int, DurableState, string) (AppendStateResult, error)
	LoadStateVersion(context.Context, string, int) (DurableState, error)
	LoadLatestState(context.Context, string) (DurableState, error)
	CreateCheckpoint(context.Context, CreateCheckpointRequest) (CheckpointRecord, error)
	LoadCheckpoint(context.Context, string) (CheckpointRecord, error)
	ListCheckpoints(context.Context, string) ([]CheckpointRecord, error)
	RecordFailure(context.Context, string, int, harnesscontracts.FailureState, string, string) (AppendStateResult, error)
	RecordRetry(context.Context, string, int, string, string) (AppendStateResult, error)
	ResumeTask(context.Context, ResumeRequest) (ResumeBundle, error)
	CompactTask(context.Context, CompactRequest) (CompactionRecord, error)
	LoadCompaction(context.Context, string) (CompactionRecord, error)
	ListCompactions(context.Context, string) ([]CompactionRecord, error)
}

func (state DurableState) Validate(objective harnesscontracts.ResearchObjective) error {
	if err := state.State.Validate(); err != nil {
		return fmt.Errorf("task state: %w", err)
	}
	if state.State.ObjectiveID != objective.ObjectiveID || state.State.ObjectiveVersion != objective.Version {
		return fmt.Errorf("task state objective identity does not match objective")
	}
	if len(state.UnresolvedContradictions) > 128 || len(state.ToolReferences) > 128 {
		return fmt.Errorf("durable state is unbounded")
	}
	for _, value := range state.UnresolvedContradictions {
		if strings.TrimSpace(value) == "" || len(value) > 4096 {
			return fmt.Errorf("unresolved contradiction is malformed")
		}
	}
	for _, reference := range state.ToolReferences {
		if err := reference.Validate(); err != nil {
			return err
		}
	}
	return rejectSecrets(state)
}

func (content CheckpointContent) Validate() error {
	if err := content.Objective.Validate(); err != nil {
		return err
	}
	if err := content.TaskState.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(content.CurrentStage) == "" || len(content.CurrentStage) > 4096 || strings.TrimSpace(content.NextAction) == "" || len(content.NextAction) > 4096 {
		return fmt.Errorf("checkpoint stage or next action is malformed")
	}
	if len(content.CompletedWork) > 256 || len(content.PendingWork) > 256 || len(content.Decisions) > 256 || len(content.OpenQuestions) > 128 || len(content.KnownUnknowns) > 128 || len(content.Constraints) > 128 || len(content.EvidenceReferences) > 512 || len(content.CounterEvidenceReferences) > 512 || len(content.MemoryReferences) > 512 || len(content.ToolReferences) > 128 || len(content.UnresolvedContradictions) > 128 || len(content.ProvenanceReferences) > 128 {
		return fmt.Errorf("checkpoint content is unbounded")
	}
	for _, item := range append(append([]harnesscontracts.WorkItem{}, content.CompletedWork...), content.PendingWork...) {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	for _, item := range content.Decisions {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	for _, ref := range content.EvidenceReferences {
		if err := ref.Validate(); err != nil {
			return err
		}
	}
	for _, ref := range content.CounterEvidenceReferences {
		if err := ref.ValidateCounterEvidence(); err != nil {
			return err
		}
	}
	for _, ref := range content.MemoryReferences {
		if err := ref.Validate(); err != nil {
			return err
		}
	}
	for _, ref := range content.ToolReferences {
		if err := ref.Validate(); err != nil {
			return err
		}
	}
	for _, value := range append(append(append(append([]string{}, content.Constraints...), content.OpenQuestions...), content.KnownUnknowns...), content.UnresolvedContradictions...) {
		if strings.TrimSpace(value) == "" || len(value) > 8192 {
			return fmt.Errorf("checkpoint text is malformed")
		}
	}
	if err := content.Failure.Validate(); err != nil {
		return err
	}
	return rejectSecrets(content)
}

func (record CheckpointRecord) Validate() error {
	if record.ContractVersion != CheckpointRecordVersion || record.TaskID == "" || record.CheckpointID == "" || record.StateVersion < 1 || !harnesscontracts.IsCanonicalHash(record.ContentHash) || record.CreatedAt.IsZero() || record.CreatedAt.Location() != time.UTC || record.Content.TaskState.TaskID != record.TaskID || record.Content.TaskState.StateVersion != record.StateVersion {
		return fmt.Errorf("checkpoint record is malformed")
	}
	if err := record.Content.Validate(); err != nil {
		return err
	}
	expected, err := checkpointContentHash(record.Content)
	if err != nil || expected != record.ContentHash {
		return fmt.Errorf("%w: checkpoint content hash mismatch", ErrIntegrity)
	}
	return nil
}

func (record CompactionRecord) Validate() error {
	if record.ContractVersion != CompactionRecordVersion || record.TaskID == "" || record.CompactionID == "" || record.SourceStateVersion < 1 || record.AlgorithmVersion == "" || record.CanonicalStorage || !harnesscontracts.IsCanonicalHash(record.ContentHash) || record.CreatedAt.IsZero() || record.CreatedAt.Location() != time.UTC || record.Content.TaskState.TaskID != record.TaskID || record.Content.TaskState.StateVersion != record.SourceStateVersion {
		return fmt.Errorf("compaction record is malformed")
	}
	if err := record.Content.Validate(); err != nil {
		return err
	}
	expected, err := compactionContentHash(record.AlgorithmVersion, record.Content)
	if err != nil || expected != record.ContentHash {
		return fmt.Errorf("%w: compaction content hash mismatch", ErrIntegrity)
	}
	return nil
}

func (bundle ResumeBundle) Validate() error {
	if bundle.ContractVersion != ContractVersion || bundle.Task.TaskID == "" || bundle.State.State.TaskID != bundle.Task.TaskID || bundle.State.State.StateVersion != bundle.Checkpoint.StateVersion || bundle.Checkpoint.TaskID != bundle.Task.TaskID || bundle.NextAction != bundle.State.State.NextAction {
		return fmt.Errorf("resume bundle identity mismatch")
	}
	if err := bundle.Task.Objective.Validate(); err != nil {
		return err
	}
	if err := bundle.State.Validate(bundle.Task.Objective); err != nil {
		return err
	}
	if err := bundle.Checkpoint.Validate(); err != nil {
		return err
	}
	return nil
}

// BuildRequest is the only integration seam HARNESS-03 exposes to
// ContextBuilder. Retrieval remains caller-owned and no model is invoked.
func (bundle ResumeBundle) BuildRequest(budget contextbuilder.BuildPolicy, tokenBudget harnesscontracts.ContextBudget, referenceTime time.Time, evidence contextbuilder.EvidenceRetriever, contradictions contextbuilder.ContradictionRetriever, memory contextbuilder.MemoryRetriever) (contextbuilder.BuildRequest, error) {
	request := contextbuilder.BuildRequest{
		Objective:              bundle.Task.Objective,
		TaskState:              bundle.State.State,
		Budget:                 tokenBudget,
		Policy:                 budget,
		ReferenceTime:          referenceTime,
		EvidenceRetriever:      evidence,
		ContradictionRetriever: contradictions,
		MemoryRetriever:        memory,
		ToolReferences:         append([]harnesscontracts.ToolReference(nil), bundle.State.ToolReferences...),
		CheckpointReference:    &harnesscontracts.CheckpointReference{ContractVersion: harnesscontracts.CheckpointContractV1, CheckpointID: bundle.Checkpoint.CheckpointID, TaskID: bundle.Task.TaskID, StateVersion: bundle.Checkpoint.StateVersion, ContentHash: bundle.Checkpoint.ContentHash, CreatedAt: bundle.Checkpoint.CreatedAt},
	}
	if err := request.Validate(); err != nil {
		return contextbuilder.BuildRequest{}, err
	}
	return request, nil
}

func checkpointContentHash(content CheckpointContent) (string, error) {
	return harnesscontracts.CanonicalHash(canonicalCheckpointContent(content))
}

func compactionContentHash(algorithm string, content CheckpointContent) (string, error) {
	return harnesscontracts.CanonicalHash(struct {
		Algorithm string            `json:"algorithm"`
		Content   CheckpointContent `json:"content"`
	}{algorithm, canonicalCheckpointContent(content)})
}

func canonicalCheckpointContent(content CheckpointContent) CheckpointContent {
	copyContent := content
	copyContent.CompletedWork = append([]harnesscontracts.WorkItem(nil), content.CompletedWork...)
	copyContent.PendingWork = append([]harnesscontracts.WorkItem(nil), content.PendingWork...)
	copyContent.Decisions = append([]harnesscontracts.DecisionRecord(nil), content.Decisions...)
	copyContent.EvidenceReferences = append([]harnesscontracts.EvidenceReference(nil), content.EvidenceReferences...)
	copyContent.CounterEvidenceReferences = append([]harnesscontracts.EvidenceReference(nil), content.CounterEvidenceReferences...)
	copyContent.MemoryReferences = append([]harnesscontracts.MemoryReference(nil), content.MemoryReferences...)
	copyContent.ToolReferences = append([]harnesscontracts.ToolReference(nil), content.ToolReferences...)
	copyContent.Constraints = append([]string(nil), content.Constraints...)
	copyContent.OpenQuestions = append([]string(nil), content.OpenQuestions...)
	copyContent.KnownUnknowns = append([]string(nil), content.KnownUnknowns...)
	copyContent.UnresolvedContradictions = append([]string(nil), content.UnresolvedContradictions...)
	copyContent.ProvenanceReferences = append([]string(nil), content.ProvenanceReferences...)
	sort.Slice(copyContent.CompletedWork, func(i, j int) bool { return copyContent.CompletedWork[i].ID < copyContent.CompletedWork[j].ID })
	sort.Slice(copyContent.PendingWork, func(i, j int) bool { return copyContent.PendingWork[i].ID < copyContent.PendingWork[j].ID })
	sort.Slice(copyContent.Decisions, func(i, j int) bool { return copyContent.Decisions[i].DecisionID < copyContent.Decisions[j].DecisionID })
	sort.Slice(copyContent.EvidenceReferences, func(i, j int) bool {
		return copyContent.EvidenceReferences[i].EvidenceID < copyContent.EvidenceReferences[j].EvidenceID
	})
	sort.Slice(copyContent.CounterEvidenceReferences, func(i, j int) bool {
		return copyContent.CounterEvidenceReferences[i].EvidenceID < copyContent.CounterEvidenceReferences[j].EvidenceID
	})
	sort.Slice(copyContent.MemoryReferences, func(i, j int) bool {
		return copyContent.MemoryReferences[i].MemoryID < copyContent.MemoryReferences[j].MemoryID
	})
	sort.Slice(copyContent.ToolReferences, func(i, j int) bool {
		return copyContent.ToolReferences[i].ToolInvocationID < copyContent.ToolReferences[j].ToolInvocationID
	})
	sort.Strings(copyContent.Constraints)
	sort.Strings(copyContent.OpenQuestions)
	sort.Strings(copyContent.KnownUnknowns)
	sort.Strings(copyContent.UnresolvedContradictions)
	sort.Strings(copyContent.ProvenanceReferences)
	return copyContent
}

func contentFromState(state DurableState, objective harnesscontracts.ResearchObjective, provenance []string) CheckpointContent {
	return CheckpointContent{
		Objective:                 harnesscontracts.ObjectiveReference{ObjectiveID: objective.ObjectiveID, ObjectiveVersion: objective.Version},
		TaskState:                 harnesscontracts.TaskStateReference{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion},
		CurrentStage:              state.State.CurrentStage,
		CompletedWork:             append([]harnesscontracts.WorkItem(nil), state.State.CompletedWork...),
		PendingWork:               append([]harnesscontracts.WorkItem(nil), state.State.PendingWork...),
		Constraints:               append([]string(nil), state.State.Constraints...),
		Decisions:                 append([]harnesscontracts.DecisionRecord(nil), state.State.Decisions...),
		OpenQuestions:             append([]string(nil), state.State.OpenQuestions...),
		KnownUnknowns:             append([]string(nil), state.State.KnownUnknowns...),
		EvidenceReferences:        append([]harnesscontracts.EvidenceReference(nil), state.State.EvidenceReferences...),
		CounterEvidenceReferences: append([]harnesscontracts.EvidenceReference(nil), state.State.CounterEvidenceReferences...),
		MemoryReferences:          append([]harnesscontracts.MemoryReference(nil), state.State.MemoryReferences...),
		ToolReferences:            append([]harnesscontracts.ToolReference(nil), state.ToolReferences...),
		UnresolvedContradictions:  append([]string(nil), state.UnresolvedContradictions...),
		Failure:                   state.State.Failure,
		NextAction:                state.State.NextAction,
		ProvenanceReferences:      append([]string(nil), provenance...),
	}
}

func normalizeCreateTask(request CreateTaskRequest) (CreateTaskRequest, error) {
	if request.TaskID == "" {
		return CreateTaskRequest{}, fmt.Errorf("%w: task ID is required", ErrInvalid)
	}
	if err := request.Objective.Validate(); err != nil {
		return CreateTaskRequest{}, fmt.Errorf("%w: objective: %v", ErrInvalid, err)
	}
	if request.InitialState.State.ContractVersion == "" {
		request.InitialState.State.ContractVersion = harnesscontracts.TaskStateContractV1
	}
	if request.InitialState.State.TaskID == "" {
		request.InitialState.State.TaskID = request.TaskID
	}
	if request.InitialState.State.ObjectiveID == "" {
		request.InitialState.State.ObjectiveID = request.Objective.ObjectiveID
	}
	if request.InitialState.State.ObjectiveVersion == "" {
		request.InitialState.State.ObjectiveVersion = request.Objective.Version
	}
	if request.InitialState.State.StateVersion == 0 {
		request.InitialState.State.StateVersion = 1
	}
	if request.InitialState.State.Status == "" {
		request.InitialState.State.Status = harnesscontracts.TaskStatusRunning
	}
	now := time.Now().UTC()
	if request.InitialState.State.CreatedAt.IsZero() {
		request.InitialState.State.CreatedAt = now
	}
	if request.InitialState.State.UpdatedAt.IsZero() {
		request.InitialState.State.UpdatedAt = request.InitialState.State.CreatedAt
	}
	if err := request.InitialState.Validate(request.Objective); err != nil {
		return CreateTaskRequest{}, wrapValidation(err)
	}
	if request.InitialState.State.StateVersion != 1 {
		return CreateTaskRequest{}, fmt.Errorf("%w: initial state must be version 1", ErrInvalid)
	}
	return request, nil
}

func validateOperationKey(value string) error {
	if !operationKeyPattern.MatchString(value) || containsSensitiveString(value) {
		return fmt.Errorf("%w: malformed operation/idempotency key", ErrInvalid)
	}
	return nil
}

func wrapValidation(err error) error {
	if errors.Is(err, ErrSecretMaterial) {
		return fmt.Errorf("%w: %v", ErrSecretMaterial, err)
	}
	return fmt.Errorf("%w: %v", ErrInvalid, err)
}

func rejectSecrets(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%w: cannot inspect payload: %v", ErrInvalid, err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("%w: cannot inspect payload: %v", ErrInvalid, err)
	}
	if walkSecret(decoded, "") {
		return ErrSecretMaterial
	}
	return nil
}

func walkSecret(value any, key string) bool {
	switch item := value.(type) {
	case map[string]any:
		for childKey, childValue := range item {
			if secretKeyPattern.MatchString(childKey) {
				if text, ok := childValue.(string); ok && strings.TrimSpace(text) != "" {
					return true
				}
			}
			if walkSecret(childValue, childKey) {
				return true
			}
		}
	case []any:
		for _, child := range item {
			if walkSecret(child, key) {
				return true
			}
		}
	case string:
		if secretValuePattern.MatchString(item) {
			return true
		}
	}
	return false
}

func containsSensitiveString(value string) bool {
	return secretKeyPattern.MatchString(value) || secretValuePattern.MatchString(value)
}

func generatedRecordID(prefix, hash, operationKey string) string {
	// The content hash gives semantic identity; the operation key keeps two
	// intentional checkpoints of identical content distinct while preserving
	// retry identity through the idempotency map/column.
	return prefix + "-" + hash[len("sha256:"):len("sha256:")+16] + "-" + operationKey
}

func clone[T any](value T) T {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		panic(err)
	}
	return result
}
