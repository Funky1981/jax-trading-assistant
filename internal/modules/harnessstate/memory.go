package harnessstate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
)

type idempotentState struct {
	Hash    string
	Version int
}

type memoryTask struct {
	record         TaskRecord
	states         map[int]DurableState
	stateHashes    map[int]string
	stateKeys      map[string]idempotentState
	checkpoints    map[string]CheckpointRecord
	checkpointKeys map[string]string
	compactions    map[string]CompactionRecord
	compactionKeys map[string]string
	failures       []FailureEvent
	retries        []RetryEvent
}

// MemoryStore is a deterministic contract store. It uses the same append,
// conflict, idempotency, integrity, and history rules as PostgresStore and is
// intended for offline fixtures and fast tests.
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]*memoryTask
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{tasks: make(map[string]*memoryTask)} }

func (s *MemoryStore) CreateTask(_ context.Context, request CreateTaskRequest) (TaskRecord, error) {
	request, err := normalizeCreateTask(request)
	if err != nil {
		return TaskRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[request.TaskID]; exists {
		return TaskRecord{}, ErrAlreadyExists
	}
	now := request.InitialState.State.UpdatedAt
	record := TaskRecord{ContractVersion: ContractVersion, TaskID: request.TaskID, Objective: clone(request.Objective), LatestVersion: 1, CreatedAt: request.InitialState.State.CreatedAt, UpdatedAt: now}
	stateHash, err := stateHash(request.InitialState)
	if err != nil {
		return TaskRecord{}, err
	}
	s.tasks[request.TaskID] = &memoryTask{record: record, states: map[int]DurableState{1: clone(request.InitialState)}, stateHashes: map[int]string{1: stateHash}, stateKeys: make(map[string]idempotentState), checkpoints: make(map[string]CheckpointRecord), checkpointKeys: make(map[string]string), compactions: make(map[string]CompactionRecord), compactionKeys: make(map[string]string)}
	return clone(record), nil
}

func (s *MemoryStore) LoadTask(_ context.Context, taskID string) (TaskRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return TaskRecord{}, ErrNotFound
	}
	return clone(task.record), nil
}

func (s *MemoryStore) AppendStateVersion(_ context.Context, taskID string, expectedVersion int, state DurableState, idempotencyKey string) (AppendStateResult, error) {
	if err := validateOperationKey(idempotencyKey); err != nil {
		return AppendStateResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return AppendStateResult{}, ErrNotFound
	}
	state = prepareNextState(state, task.record.Objective, taskID, expectedVersion+1)
	if err := state.Validate(task.record.Objective); err != nil {
		return AppendStateResult{}, wrapValidation(err)
	}
	hash, err := stateHash(state)
	if err != nil {
		return AppendStateResult{}, err
	}
	if prior, exists := task.stateKeys[idempotencyKey]; exists {
		if prior.Hash != hash {
			return AppendStateResult{}, ErrIdempotencyConflict
		}
		return AppendStateResult{State: clone(task.states[prior.Version]), Applied: false}, nil
	}
	if expectedVersion != task.record.LatestVersion {
		return AppendStateResult{}, &VersionConflictError{TaskID: taskID, ExpectedVersion: expectedVersion, CurrentVersion: task.record.LatestVersion}
	}
	if state.State.StateVersion != expectedVersion+1 {
		return AppendStateResult{}, fmt.Errorf("%w: next state version must be %d", ErrInvalid, expectedVersion+1)
	}
	task.states[state.State.StateVersion] = clone(state)
	task.stateHashes[state.State.StateVersion] = hash
	task.stateKeys[idempotencyKey] = idempotentState{Hash: hash, Version: state.State.StateVersion}
	task.record.LatestVersion = state.State.StateVersion
	task.record.UpdatedAt = state.State.UpdatedAt
	return AppendStateResult{State: clone(state), Applied: true}, nil
}

func prepareNextState(state DurableState, objective harnesscontracts.ResearchObjective, taskID string, version int) DurableState {
	state = clone(state)
	if state.State.ContractVersion == "" {
		state.State.ContractVersion = harnesscontracts.TaskStateContractV1
	}
	state.State.TaskID = taskID
	state.State.ObjectiveID = objective.ObjectiveID
	state.State.ObjectiveVersion = objective.Version
	state.State.StateVersion = version
	if state.State.UpdatedAt.IsZero() {
		state.State.UpdatedAt = time.Now().UTC()
	}
	if state.State.CreatedAt.IsZero() {
		state.State.CreatedAt = state.State.UpdatedAt
	}
	return state
}

func (s *MemoryStore) LoadStateVersion(_ context.Context, taskID string, version int) (DurableState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return DurableState{}, ErrNotFound
	}
	state, ok := task.states[version]
	if !ok {
		return DurableState{}, ErrNotFound
	}
	if err := verifyState(state, task.record.Objective, task.stateHashes[version]); err != nil {
		return DurableState{}, err
	}
	return clone(state), nil
}

func (s *MemoryStore) LoadLatestState(ctx context.Context, taskID string) (DurableState, error) {
	task, err := s.LoadTask(ctx, taskID)
	if err != nil {
		return DurableState{}, err
	}
	return s.LoadStateVersion(ctx, taskID, task.LatestVersion)
}

func (s *MemoryStore) CreateCheckpoint(_ context.Context, request CreateCheckpointRequest) (CheckpointRecord, error) {
	if err := validateOperationKey(request.IdempotencyKey); err != nil {
		return CheckpointRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[request.TaskID]
	if !ok {
		return CheckpointRecord{}, ErrNotFound
	}
	stateVersion := request.StateVersion
	if stateVersion == 0 {
		stateVersion = task.record.LatestVersion
	}
	state, ok := task.states[stateVersion]
	if !ok {
		return CheckpointRecord{}, ErrNotFound
	}
	content := contentFromState(state, task.record.Objective, request.ProvenanceReferences)
	if err := content.Validate(); err != nil {
		return CheckpointRecord{}, wrapValidation(err)
	}
	hash, err := checkpointContentHash(content)
	if err != nil {
		return CheckpointRecord{}, err
	}
	if priorID, exists := task.checkpointKeys[request.IdempotencyKey]; exists {
		prior := task.checkpoints[priorID]
		if prior.ContentHash != hash {
			return CheckpointRecord{}, ErrIdempotencyConflict
		}
		return clone(prior), nil
	}
	checkpointID := request.CheckpointID
	if checkpointID == "" {
		checkpointID = generatedRecordID("checkpoint", hash, request.IdempotencyKey)
	}
	if _, exists := task.checkpoints[checkpointID]; exists {
		return CheckpointRecord{}, ErrAlreadyExists
	}
	createdAt := request.CreatedAt.UTC()
	if request.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	record := CheckpointRecord{ContractVersion: CheckpointRecordVersion, CheckpointID: checkpointID, TaskID: request.TaskID, StateVersion: stateVersion, Content: content, ContentHash: hash, CreatedAt: createdAt}
	if err := record.Validate(); err != nil {
		return CheckpointRecord{}, err
	}
	task.checkpoints[checkpointID] = clone(record)
	task.checkpointKeys[request.IdempotencyKey] = checkpointID
	return clone(record), nil
}

func (s *MemoryStore) LoadCheckpoint(_ context.Context, checkpointID string) (CheckpointRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, task := range s.tasks {
		if record, ok := task.checkpoints[checkpointID]; ok {
			if err := record.Validate(); err != nil {
				return CheckpointRecord{}, err
			}
			return clone(record), nil
		}
	}
	return CheckpointRecord{}, ErrNotFound
}

func (s *MemoryStore) ListCheckpoints(_ context.Context, taskID string) ([]CheckpointRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}
	result := make([]CheckpointRecord, 0, len(task.checkpoints))
	for _, record := range task.checkpoints {
		result = append(result, clone(record))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].StateVersion != result[j].StateVersion {
			return result[i].StateVersion < result[j].StateVersion
		}
		return result[i].CheckpointID < result[j].CheckpointID
	})
	return result, nil
}

func (s *MemoryStore) RecordFailure(ctx context.Context, taskID string, expectedVersion int, failure harnesscontracts.FailureState, nextAction, idempotencyKey string) (AppendStateResult, error) {
	if failure.Reason == "" {
		return AppendStateResult{}, fmt.Errorf("%w: failure reason is required", ErrInvalid)
	}
	s.mu.RLock()
	if task, ok := s.tasks[taskID]; ok {
		for _, event := range task.failures {
			if event.EventID == "failure-"+idempotencyKey {
				state := task.states[event.StateVersion]
				s.mu.RUnlock()
				return AppendStateResult{State: clone(state), Applied: false}, nil
			}
		}
	}
	s.mu.RUnlock()
	state, err := s.LoadLatestState(ctx, taskID)
	if err != nil {
		return AppendStateResult{}, err
	}
	state.State.Status = harnesscontracts.TaskStatusFailed
	state.State.Failure = failure
	state.State.NextAction = nextAction
	result, err := s.AppendStateVersion(ctx, taskID, expectedVersion, state, idempotencyKey)
	if err != nil {
		return AppendStateResult{}, err
	}
	if result.Applied {
		s.mu.Lock()
		s.tasks[taskID].failures = append(s.tasks[taskID].failures, FailureEvent{ContractVersion: FailureRecordVersion, EventID: "failure-" + idempotencyKey, TaskID: taskID, StateVersion: result.State.State.StateVersion, Failure: failure, CreatedAt: result.State.State.UpdatedAt})
		s.mu.Unlock()
	}
	return result, nil
}

func (s *MemoryStore) RecordRetry(ctx context.Context, taskID string, expectedVersion int, nextAction, idempotencyKey string) (AppendStateResult, error) {
	s.mu.RLock()
	if task, ok := s.tasks[taskID]; ok {
		for _, event := range task.retries {
			if event.EventID == "retry-"+idempotencyKey {
				state := task.states[event.StateVersion]
				s.mu.RUnlock()
				return AppendStateResult{State: clone(state), Applied: false}, nil
			}
		}
	}
	s.mu.RUnlock()
	state, err := s.LoadLatestState(ctx, taskID)
	if err != nil {
		return AppendStateResult{}, err
	}
	attempt := state.State.Failure.Attempt + 1
	state.State.Status = harnesscontracts.TaskStatusRunning
	state.State.Failure = harnesscontracts.FailureState{}
	state.State.NextAction = nextAction
	result, err := s.AppendStateVersion(ctx, taskID, expectedVersion, state, idempotencyKey)
	if err != nil {
		return AppendStateResult{}, err
	}
	if result.Applied {
		s.mu.Lock()
		s.tasks[taskID].retries = append(s.tasks[taskID].retries, RetryEvent{ContractVersion: RetryRecordVersion, EventID: "retry-" + idempotencyKey, TaskID: taskID, StateVersion: result.State.State.StateVersion, Attempt: attempt, NextAction: nextAction, CreatedAt: result.State.State.UpdatedAt})
		s.mu.Unlock()
	}
	return result, nil
}

func (s *MemoryStore) ResumeTask(ctx context.Context, request ResumeRequest) (ResumeBundle, error) {
	task, err := s.LoadTask(ctx, request.TaskID)
	if err != nil {
		return ResumeBundle{}, err
	}
	var checkpoint CheckpointRecord
	if request.CheckpointID != "" {
		checkpoint, err = s.LoadCheckpoint(ctx, request.CheckpointID)
	} else {
		checkpoints, listErr := s.ListCheckpoints(ctx, request.TaskID)
		if listErr != nil {
			return ResumeBundle{}, listErr
		}
		if len(checkpoints) == 0 {
			return ResumeBundle{}, ErrNotFound
		}
		checkpoint = checkpoints[len(checkpoints)-1]
	}
	if err != nil {
		return ResumeBundle{}, err
	}
	if checkpoint.TaskID != request.TaskID {
		return ResumeBundle{}, fmt.Errorf("%w: checkpoint belongs to another task", ErrInvalid)
	}
	state, err := s.LoadStateVersion(ctx, request.TaskID, checkpoint.StateVersion)
	if err != nil {
		return ResumeBundle{}, err
	}
	if err := compareCheckpointState(checkpoint.Content, state, task.Objective); err != nil {
		return ResumeBundle{}, err
	}
	s.mu.RLock()
	failures := make([]FailureEvent, 0, len(s.tasks[request.TaskID].failures))
	for _, event := range s.tasks[request.TaskID].failures {
		if event.StateVersion <= checkpoint.StateVersion {
			failures = append(failures, clone(event))
		}
	}
	retries := make([]RetryEvent, 0, len(s.tasks[request.TaskID].retries))
	for _, event := range s.tasks[request.TaskID].retries {
		if event.StateVersion <= checkpoint.StateVersion {
			retries = append(retries, clone(event))
		}
	}
	s.mu.RUnlock()
	bundle := ResumeBundle{ContractVersion: ContractVersion, Task: task, State: state, Checkpoint: checkpoint, FailureHistory: failures, RetryHistory: retries, NextAction: state.State.NextAction}
	if err := bundle.Validate(); err != nil {
		return ResumeBundle{}, err
	}
	return bundle, nil
}

func compareCheckpointState(content CheckpointContent, state DurableState, objective harnesscontracts.ResearchObjective) error {
	want := contentFromState(state, objective, content.ProvenanceReferences)
	if stringMustJSON(canonicalCheckpointContent(content)) != stringMustJSON(canonicalCheckpointContent(want)) {
		return fmt.Errorf("%w: checkpoint does not match canonical state version", ErrIntegrity)
	}
	return nil
}

func stringMustJSON(value any) string { data, _ := json.Marshal(value); return string(data) }

func (s *MemoryStore) CompactTask(_ context.Context, request CompactRequest) (CompactionRecord, error) {
	if err := validateOperationKey(request.IdempotencyKey); err != nil {
		return CompactionRecord{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[request.TaskID]
	if !ok {
		return CompactionRecord{}, ErrNotFound
	}
	stateVersion := request.StateVersion
	if stateVersion == 0 {
		stateVersion = task.record.LatestVersion
	}
	state, ok := task.states[stateVersion]
	if !ok {
		return CompactionRecord{}, ErrNotFound
	}
	algorithm := request.AlgorithmVersion
	if algorithm == "" {
		algorithm = CompactionAlgorithmV1
	}
	content := contentFromState(state, task.record.Objective, request.ProvenanceReferences)
	if err := content.Validate(); err != nil {
		return CompactionRecord{}, wrapValidation(err)
	}
	hash, err := compactionContentHash(algorithm, content)
	if err != nil {
		return CompactionRecord{}, err
	}
	if priorID, exists := task.compactionKeys[request.IdempotencyKey]; exists {
		prior := task.compactions[priorID]
		if prior.ContentHash != hash {
			return CompactionRecord{}, ErrIdempotencyConflict
		}
		return clone(prior), nil
	}
	id := request.CompactionID
	if id == "" {
		id = generatedRecordID("compaction", hash, request.IdempotencyKey)
	}
	if _, exists := task.compactions[id]; exists {
		return CompactionRecord{}, ErrAlreadyExists
	}
	createdAt := request.CreatedAt.UTC()
	if request.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	record := CompactionRecord{ContractVersion: CompactionRecordVersion, CompactionID: id, TaskID: request.TaskID, SourceStateVersion: stateVersion, Content: content, AlgorithmVersion: algorithm, CanonicalStorage: false, ContentHash: hash, CreatedAt: createdAt}
	if err := record.Validate(); err != nil {
		return CompactionRecord{}, err
	}
	task.compactions[id] = clone(record)
	task.compactionKeys[request.IdempotencyKey] = id
	return clone(record), nil
}

func (s *MemoryStore) LoadCompaction(_ context.Context, compactionID string) (CompactionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, task := range s.tasks {
		if record, ok := task.compactions[compactionID]; ok {
			if err := record.Validate(); err != nil {
				return CompactionRecord{}, err
			}
			return clone(record), nil
		}
	}
	return CompactionRecord{}, ErrNotFound
}

func (s *MemoryStore) ListCompactions(_ context.Context, taskID string) ([]CompactionRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}
	result := make([]CompactionRecord, 0, len(task.compactions))
	for _, record := range task.compactions {
		result = append(result, clone(record))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceStateVersion != result[j].SourceStateVersion {
			return result[i].SourceStateVersion < result[j].SourceStateVersion
		}
		return result[i].CompactionID < result[j].CompactionID
	})
	return result, nil
}

func verifyState(state DurableState, objective harnesscontracts.ResearchObjective, expectedHash string) error {
	if err := state.Validate(objective); err != nil {
		return err
	}
	hash, err := stateHash(state)
	if err != nil || hash != expectedHash {
		return fmt.Errorf("%w: task state hash mismatch", ErrIntegrity)
	}
	return nil
}

func stateHash(state DurableState) (string, error) {
	state = canonicalState(state)
	return harnesscontracts.CanonicalHash(state)
}

func canonicalState(state DurableState) DurableState {
	state = clone(state)
	sort.Slice(state.State.CompletedWork, func(i, j int) bool { return state.State.CompletedWork[i].ID < state.State.CompletedWork[j].ID })
	sort.Slice(state.State.PendingWork, func(i, j int) bool { return state.State.PendingWork[i].ID < state.State.PendingWork[j].ID })
	sort.Slice(state.State.Decisions, func(i, j int) bool { return state.State.Decisions[i].DecisionID < state.State.Decisions[j].DecisionID })
	sort.Slice(state.State.EvidenceReferences, func(i, j int) bool {
		return state.State.EvidenceReferences[i].EvidenceID < state.State.EvidenceReferences[j].EvidenceID
	})
	sort.Slice(state.State.CounterEvidenceReferences, func(i, j int) bool {
		return state.State.CounterEvidenceReferences[i].EvidenceID < state.State.CounterEvidenceReferences[j].EvidenceID
	})
	sort.Slice(state.State.MemoryReferences, func(i, j int) bool {
		return state.State.MemoryReferences[i].MemoryID < state.State.MemoryReferences[j].MemoryID
	})
	sort.Slice(state.ToolReferences, func(i, j int) bool {
		return state.ToolReferences[i].ToolInvocationID < state.ToolReferences[j].ToolInvocationID
	})
	sort.Strings(state.State.Constraints)
	sort.Strings(state.State.OpenQuestions)
	sort.Strings(state.State.KnownUnknowns)
	sort.Strings(state.UnresolvedContradictions)
	return state
}

// Snapshot and Restore let tests model a process boundary without coupling
// them to a database. The production process-boundary implementation is the
// PostgreSQL store below.
type memorySnapshot struct {
	Tasks []memorySnapshotTask `json:"tasks"`
}
type memorySnapshotTask struct {
	Record         TaskRecord                  `json:"record"`
	States         map[int]DurableState        `json:"states"`
	StateHashes    map[int]string              `json:"state_hashes"`
	StateKeys      map[string]idempotentState  `json:"state_keys"`
	Checkpoints    map[string]CheckpointRecord `json:"checkpoints"`
	CheckpointKeys map[string]string           `json:"checkpoint_keys"`
	Compactions    map[string]CompactionRecord `json:"compactions"`
	CompactionKeys map[string]string           `json:"compaction_keys"`
	Failures       []FailureEvent              `json:"failures"`
	Retries        []RetryEvent                `json:"retries"`
}

func (s *MemoryStore) Snapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := memorySnapshot{}
	for _, task := range s.tasks {
		snapshot.Tasks = append(snapshot.Tasks, memorySnapshotTask{task.record, task.states, task.stateHashes, task.stateKeys, task.checkpoints, task.checkpointKeys, task.compactions, task.compactionKeys, task.failures, task.retries})
	}
	sort.Slice(snapshot.Tasks, func(i, j int) bool { return snapshot.Tasks[i].Record.TaskID < snapshot.Tasks[j].Record.TaskID })
	return json.Marshal(snapshot)
}

func RestoreMemoryStore(data []byte) (*MemoryStore, error) {
	var snapshot memorySnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	store := NewMemoryStore()
	for _, item := range snapshot.Tasks {
		task := &memoryTask{record: item.Record, states: item.States, stateHashes: item.StateHashes, stateKeys: item.StateKeys, checkpoints: item.Checkpoints, checkpointKeys: item.CheckpointKeys, compactions: item.Compactions, compactionKeys: item.CompactionKeys, failures: item.Failures, retries: item.Retries}
		if task.states == nil || task.stateHashes == nil || task.stateKeys == nil || task.checkpoints == nil || task.checkpointKeys == nil || task.compactions == nil || task.compactionKeys == nil {
			return nil, fmt.Errorf("%w: incomplete memory snapshot", ErrIntegrity)
		}
		store.tasks[item.Record.TaskID] = task
	}
	return store, nil
}
