package harnessstate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
)

// PostgresStore is the durable implementation. It expects migration
// 000075_harness_durable_task_state to have been applied by the repository's
// normal migration runner.
type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: nil database", ErrInvalid)
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) CreateTask(ctx context.Context, request CreateTaskRequest) (TaskRecord, error) {
	request, err := normalizeCreateTask(request)
	if err != nil {
		return TaskRecord{}, err
	}
	payload, err := json.Marshal(request.InitialState)
	if err != nil {
		return TaskRecord{}, err
	}
	stateHashValue, err := stateHash(request.InitialState)
	if err != nil {
		return TaskRecord{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TaskRecord{}, err
	}
	defer tx.Rollback()
	record := TaskRecord{ContractVersion: ContractVersion, TaskID: request.TaskID, Objective: clone(request.Objective), LatestVersion: 1, CreatedAt: request.InitialState.State.CreatedAt, UpdatedAt: request.InitialState.State.UpdatedAt}
	objectivePayload, err := json.Marshal(record.Objective)
	if err != nil {
		return TaskRecord{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO harness_tasks (task_id, objective, latest_state_version, created_at, updated_at) VALUES ($1,$2,$3,$4,$5)`, request.TaskID, objectivePayload, 1, record.CreatedAt, record.UpdatedAt); err != nil {
		return TaskRecord{}, mapDatabaseError(err, ErrAlreadyExists)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO harness_task_state_versions (task_id, state_version, state_hash, payload, idempotency_key, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, request.TaskID, 1, stateHashValue, payload, "create-"+request.TaskID, request.InitialState.State.UpdatedAt); err != nil {
		return TaskRecord{}, err
	}
	if err = tx.Commit(); err != nil {
		return TaskRecord{}, err
	}
	return record, nil
}

func (s *PostgresStore) LoadTask(ctx context.Context, taskID string) (TaskRecord, error) {
	var record TaskRecord
	var objective []byte
	err := s.db.QueryRowContext(ctx, `SELECT task_id, objective, latest_state_version, created_at, updated_at FROM harness_tasks WHERE task_id=$1`, taskID).Scan(&record.TaskID, &objective, &record.LatestVersion, &record.CreatedAt, &record.UpdatedAt)
	if err == sql.ErrNoRows {
		return TaskRecord{}, ErrNotFound
	}
	if err != nil {
		return TaskRecord{}, err
	}
	if err := json.Unmarshal(objective, &record.Objective); err != nil {
		return TaskRecord{}, fmt.Errorf("%w: objective JSON: %v", ErrIntegrity, err)
	}
	record.ContractVersion = ContractVersion
	return record, nil
}

func (s *PostgresStore) AppendStateVersion(ctx context.Context, taskID string, expectedVersion int, state DurableState, idempotencyKey string) (AppendStateResult, error) {
	if err := validateOperationKey(idempotencyKey); err != nil {
		return AppendStateResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AppendStateResult{}, err
	}
	defer tx.Rollback()
	var current int
	var objectivePayload []byte
	if err = tx.QueryRowContext(ctx, `SELECT latest_state_version, objective FROM harness_tasks WHERE task_id=$1 FOR UPDATE`, taskID).Scan(&current, &objectivePayload); err == sql.ErrNoRows {
		return AppendStateResult{}, ErrNotFound
	} else if err != nil {
		return AppendStateResult{}, err
	}
	var objective harnesscontracts.ResearchObjective
	if err = json.Unmarshal(objectivePayload, &objective); err != nil {
		return AppendStateResult{}, fmt.Errorf("%w: objective JSON: %v", ErrIntegrity, err)
	}
	state = prepareNextState(state, objective, taskID, expectedVersion+1)
	if err = state.Validate(objective); err != nil {
		return AppendStateResult{}, wrapValidation(err)
	}
	hash, err := stateHash(state)
	if err != nil {
		return AppendStateResult{}, err
	}
	var existingVersion int
	var existingHash string
	var existingPayload []byte
	lookupErr := tx.QueryRowContext(ctx, `SELECT state_version, state_hash, payload FROM harness_task_state_versions WHERE task_id=$1 AND idempotency_key=$2`, taskID, idempotencyKey).Scan(&existingVersion, &existingHash, &existingPayload)
	if lookupErr == nil {
		if existingHash != hash {
			return AppendStateResult{}, ErrIdempotencyConflict
		}
		var existing DurableState
		if err := json.Unmarshal(existingPayload, &existing); err != nil {
			return AppendStateResult{}, fmt.Errorf("%w: state JSON: %v", ErrIntegrity, err)
		}
		if err := tx.Commit(); err != nil {
			return AppendStateResult{}, err
		}
		return AppendStateResult{State: existing, Applied: false}, nil
	}
	if lookupErr != sql.ErrNoRows {
		return AppendStateResult{}, lookupErr
	}
	if expectedVersion != current {
		return AppendStateResult{}, &VersionConflictError{TaskID: taskID, ExpectedVersion: expectedVersion, CurrentVersion: current}
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return AppendStateResult{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO harness_task_state_versions (task_id, state_version, state_hash, payload, idempotency_key, created_at) VALUES ($1,$2,$3,$4,$5,$6)`, taskID, expectedVersion+1, hash, payload, idempotencyKey, state.State.UpdatedAt); err != nil {
		return AppendStateResult{}, mapDatabaseError(err, ErrIdempotencyConflict)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE harness_tasks SET latest_state_version=$2, updated_at=$3 WHERE task_id=$1`, taskID, expectedVersion+1, state.State.UpdatedAt); err != nil {
		return AppendStateResult{}, err
	}
	if err = tx.Commit(); err != nil {
		return AppendStateResult{}, err
	}
	return AppendStateResult{State: state, Applied: true}, nil
}

func (s *PostgresStore) LoadStateVersion(ctx context.Context, taskID string, version int) (DurableState, error) {
	var payload []byte
	var storedHash string
	var objectivePayload []byte
	err := s.db.QueryRowContext(ctx, `SELECT v.payload, v.state_hash, t.objective FROM harness_task_state_versions v JOIN harness_tasks t ON t.task_id=v.task_id WHERE v.task_id=$1 AND v.state_version=$2`, taskID, version).Scan(&payload, &storedHash, &objectivePayload)
	if err == sql.ErrNoRows {
		return DurableState{}, ErrNotFound
	}
	if err != nil {
		return DurableState{}, err
	}
	var state DurableState
	var objective harnesscontracts.ResearchObjective
	if err := json.Unmarshal(payload, &state); err != nil {
		return DurableState{}, fmt.Errorf("%w: state JSON: %v", ErrIntegrity, err)
	}
	if err := json.Unmarshal(objectivePayload, &objective); err != nil {
		return DurableState{}, fmt.Errorf("%w: objective JSON: %v", ErrIntegrity, err)
	}
	if err := verifyState(state, objective, storedHash); err != nil {
		return DurableState{}, err
	}
	return state, nil
}

func (s *PostgresStore) LoadLatestState(ctx context.Context, taskID string) (DurableState, error) {
	task, err := s.LoadTask(ctx, taskID)
	if err != nil {
		return DurableState{}, err
	}
	return s.LoadStateVersion(ctx, taskID, task.LatestVersion)
}

func (s *PostgresStore) CreateCheckpoint(ctx context.Context, request CreateCheckpointRequest) (CheckpointRecord, error) {
	if err := validateOperationKey(request.IdempotencyKey); err != nil {
		return CheckpointRecord{}, err
	}
	task, err := s.LoadTask(ctx, request.TaskID)
	if err != nil {
		return CheckpointRecord{}, err
	}
	version := request.StateVersion
	if version == 0 {
		version = task.LatestVersion
	}
	state, err := s.LoadStateVersion(ctx, request.TaskID, version)
	if err != nil {
		return CheckpointRecord{}, err
	}
	content := contentFromState(state, task.Objective, request.ProvenanceReferences)
	if err := content.Validate(); err != nil {
		return CheckpointRecord{}, wrapValidation(err)
	}
	hash, err := checkpointContentHash(content)
	if err != nil {
		return CheckpointRecord{}, err
	}
	var existingID, existingHash string
	lookupErr := s.db.QueryRowContext(ctx, `SELECT checkpoint_id, content_hash FROM harness_checkpoints WHERE task_id=$1 AND idempotency_key=$2`, request.TaskID, request.IdempotencyKey).Scan(&existingID, &existingHash)
	if lookupErr == nil {
		if existingHash != hash {
			return CheckpointRecord{}, ErrIdempotencyConflict
		}
		return s.LoadCheckpoint(ctx, existingID)
	}
	if lookupErr != sql.ErrNoRows {
		return CheckpointRecord{}, lookupErr
	}
	id := request.CheckpointID
	if id == "" {
		id = generatedRecordID("checkpoint", hash, request.IdempotencyKey)
	}
	createdAt := request.CreatedAt.UTC()
	if request.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	record := CheckpointRecord{ContractVersion: CheckpointRecordVersion, CheckpointID: id, TaskID: request.TaskID, StateVersion: version, Content: content, ContentHash: hash, CreatedAt: createdAt}
	if err := record.Validate(); err != nil {
		return CheckpointRecord{}, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return CheckpointRecord{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO harness_checkpoints (checkpoint_id, task_id, state_version, content_hash, payload, idempotency_key, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, request.TaskID, version, hash, payload, request.IdempotencyKey, createdAt)
	if err != nil {
		return CheckpointRecord{}, mapDatabaseError(err, ErrAlreadyExists)
	}
	return record, nil
}

func (s *PostgresStore) LoadCheckpoint(ctx context.Context, checkpointID string) (CheckpointRecord, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM harness_checkpoints WHERE checkpoint_id=$1`, checkpointID).Scan(&payload)
	if err == sql.ErrNoRows {
		return CheckpointRecord{}, ErrNotFound
	}
	if err != nil {
		return CheckpointRecord{}, err
	}
	var record CheckpointRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return CheckpointRecord{}, fmt.Errorf("%w: checkpoint JSON: %v", ErrIntegrity, err)
	}
	if err := record.Validate(); err != nil {
		return CheckpointRecord{}, err
	}
	return record, nil
}

func (s *PostgresStore) ListCheckpoints(ctx context.Context, taskID string) ([]CheckpointRecord, error) {
	if _, err := s.LoadTask(ctx, taskID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM harness_checkpoints WHERE task_id=$1 ORDER BY state_version, checkpoint_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []CheckpointRecord{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var record CheckpointRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("%w: checkpoint JSON: %v", ErrIntegrity, err)
		}
		if err := record.Validate(); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *PostgresStore) RecordFailure(ctx context.Context, taskID string, expectedVersion int, failure harnesscontracts.FailureState, nextAction, idempotencyKey string) (AppendStateResult, error) {
	if failure.Reason == "" {
		return AppendStateResult{}, fmt.Errorf("%w: failure reason is required", ErrInvalid)
	}
	var priorVersion int
	lookupErr := s.db.QueryRowContext(ctx, `SELECT state_version FROM harness_failure_events WHERE event_id=$1`, "failure-"+idempotencyKey).Scan(&priorVersion)
	if lookupErr == nil {
		state, err := s.LoadStateVersion(ctx, taskID, priorVersion)
		if err != nil {
			return AppendStateResult{}, err
		}
		return AppendStateResult{State: state, Applied: false}, nil
	}
	if lookupErr != sql.ErrNoRows {
		return AppendStateResult{}, lookupErr
	}
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
		event := FailureEvent{ContractVersion: FailureRecordVersion, EventID: "failure-" + idempotencyKey, TaskID: taskID, StateVersion: result.State.State.StateVersion, Failure: failure, CreatedAt: result.State.State.UpdatedAt}
		payload, _ := json.Marshal(event)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO harness_failure_events (event_id, task_id, state_version, payload, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (event_id) DO NOTHING`, event.EventID, taskID, event.StateVersion, payload, event.CreatedAt)
	}
	return result, nil
}

func (s *PostgresStore) RecordRetry(ctx context.Context, taskID string, expectedVersion int, nextAction, idempotencyKey string) (AppendStateResult, error) {
	var priorVersion int
	lookupErr := s.db.QueryRowContext(ctx, `SELECT state_version FROM harness_retry_events WHERE event_id=$1`, "retry-"+idempotencyKey).Scan(&priorVersion)
	if lookupErr == nil {
		state, err := s.LoadStateVersion(ctx, taskID, priorVersion)
		if err != nil {
			return AppendStateResult{}, err
		}
		return AppendStateResult{State: state, Applied: false}, nil
	}
	if lookupErr != sql.ErrNoRows {
		return AppendStateResult{}, lookupErr
	}
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
		event := RetryEvent{ContractVersion: RetryRecordVersion, EventID: "retry-" + idempotencyKey, TaskID: taskID, StateVersion: result.State.State.StateVersion, Attempt: attempt, NextAction: nextAction, CreatedAt: result.State.State.UpdatedAt}
		payload, _ := json.Marshal(event)
		_, _ = s.db.ExecContext(ctx, `INSERT INTO harness_retry_events (event_id, task_id, state_version, payload, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (event_id) DO NOTHING`, event.EventID, taskID, event.StateVersion, payload, event.CreatedAt)
	}
	return result, nil
}

func (s *PostgresStore) ResumeTask(ctx context.Context, request ResumeRequest) (ResumeBundle, error) {
	task, err := s.LoadTask(ctx, request.TaskID)
	if err != nil {
		return ResumeBundle{}, err
	}
	var checkpoint CheckpointRecord
	if request.CheckpointID != "" {
		checkpoint, err = s.LoadCheckpoint(ctx, request.CheckpointID)
	} else {
		list, listErr := s.ListCheckpoints(ctx, request.TaskID)
		if listErr != nil {
			return ResumeBundle{}, listErr
		}
		if len(list) == 0 {
			return ResumeBundle{}, ErrNotFound
		}
		checkpoint = list[len(list)-1]
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
	failures, retries, err := s.loadEvents(ctx, request.TaskID, checkpoint.StateVersion)
	if err != nil {
		return ResumeBundle{}, err
	}
	bundle := ResumeBundle{ContractVersion: ContractVersion, Task: task, State: state, Checkpoint: checkpoint, FailureHistory: failures, RetryHistory: retries, NextAction: state.State.NextAction}
	if err := bundle.Validate(); err != nil {
		return ResumeBundle{}, err
	}
	return bundle, nil
}

func (s *PostgresStore) loadEvents(ctx context.Context, taskID string, maxStateVersion int) ([]FailureEvent, []RetryEvent, error) {
	failureRows, err := s.db.QueryContext(ctx, `SELECT payload FROM harness_failure_events WHERE task_id=$1 AND state_version <= $2 ORDER BY state_version, event_id`, taskID, maxStateVersion)
	if err != nil {
		return nil, nil, err
	}
	var failures []FailureEvent
	for failureRows.Next() {
		var payload []byte
		if err := failureRows.Scan(&payload); err != nil {
			failureRows.Close()
			return nil, nil, err
		}
		var event FailureEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			failureRows.Close()
			return nil, nil, fmt.Errorf("%w: failure JSON: %v", ErrIntegrity, err)
		}
		failures = append(failures, event)
	}
	if err := failureRows.Close(); err != nil {
		return nil, nil, err
	}
	retryRows, err := s.db.QueryContext(ctx, `SELECT payload FROM harness_retry_events WHERE task_id=$1 AND state_version <= $2 ORDER BY state_version, event_id`, taskID, maxStateVersion)
	if err != nil {
		return nil, nil, err
	}
	defer retryRows.Close()
	var retries []RetryEvent
	for retryRows.Next() {
		var payload []byte
		if err := retryRows.Scan(&payload); err != nil {
			return nil, nil, err
		}
		var event RetryEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, nil, fmt.Errorf("%w: retry JSON: %v", ErrIntegrity, err)
		}
		retries = append(retries, event)
	}
	return failures, retries, retryRows.Err()
}

func (s *PostgresStore) CompactTask(ctx context.Context, request CompactRequest) (CompactionRecord, error) {
	if err := validateOperationKey(request.IdempotencyKey); err != nil {
		return CompactionRecord{}, err
	}
	task, err := s.LoadTask(ctx, request.TaskID)
	if err != nil {
		return CompactionRecord{}, err
	}
	version := request.StateVersion
	if version == 0 {
		version = task.LatestVersion
	}
	state, err := s.LoadStateVersion(ctx, request.TaskID, version)
	if err != nil {
		return CompactionRecord{}, err
	}
	algorithm := request.AlgorithmVersion
	if algorithm == "" {
		algorithm = CompactionAlgorithmV1
	}
	content := contentFromState(state, task.Objective, request.ProvenanceReferences)
	if err := content.Validate(); err != nil {
		return CompactionRecord{}, wrapValidation(err)
	}
	hash, err := compactionContentHash(algorithm, content)
	if err != nil {
		return CompactionRecord{}, err
	}
	var existingID, existingHash string
	lookupErr := s.db.QueryRowContext(ctx, `SELECT compaction_id, content_hash FROM harness_compactions WHERE task_id=$1 AND idempotency_key=$2`, request.TaskID, request.IdempotencyKey).Scan(&existingID, &existingHash)
	if lookupErr == nil {
		if existingHash != hash {
			return CompactionRecord{}, ErrIdempotencyConflict
		}
		return s.LoadCompaction(ctx, existingID)
	}
	if lookupErr != sql.ErrNoRows {
		return CompactionRecord{}, lookupErr
	}
	id := request.CompactionID
	if id == "" {
		id = generatedRecordID("compaction", hash, request.IdempotencyKey)
	}
	createdAt := request.CreatedAt.UTC()
	if request.CreatedAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	record := CompactionRecord{ContractVersion: CompactionRecordVersion, CompactionID: id, TaskID: request.TaskID, SourceStateVersion: version, Content: content, AlgorithmVersion: algorithm, CanonicalStorage: false, ContentHash: hash, CreatedAt: createdAt}
	if err := record.Validate(); err != nil {
		return CompactionRecord{}, err
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return CompactionRecord{}, err
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO harness_compactions (compaction_id, task_id, state_version, content_hash, payload, idempotency_key, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, request.TaskID, version, hash, payload, request.IdempotencyKey, createdAt); err != nil {
		return CompactionRecord{}, mapDatabaseError(err, ErrAlreadyExists)
	}
	return record, nil
}

func (s *PostgresStore) LoadCompaction(ctx context.Context, compactionID string) (CompactionRecord, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM harness_compactions WHERE compaction_id=$1`, compactionID).Scan(&payload)
	if err == sql.ErrNoRows {
		return CompactionRecord{}, ErrNotFound
	}
	if err != nil {
		return CompactionRecord{}, err
	}
	var record CompactionRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return CompactionRecord{}, fmt.Errorf("%w: compaction JSON: %v", ErrIntegrity, err)
	}
	if err := record.Validate(); err != nil {
		return CompactionRecord{}, err
	}
	return record, nil
}

func (s *PostgresStore) ListCompactions(ctx context.Context, taskID string) ([]CompactionRecord, error) {
	if _, err := s.LoadTask(ctx, taskID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM harness_compactions WHERE task_id=$1 ORDER BY state_version, compaction_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []CompactionRecord
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var record CompactionRecord
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, fmt.Errorf("%w: compaction JSON: %v", ErrIntegrity, err)
		}
		if err := record.Validate(); err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SourceStateVersion != result[j].SourceStateVersion {
			return result[i].SourceStateVersion < result[j].SourceStateVersion
		}
		return result[i].CompactionID < result[j].CompactionID
	})
	return result, rows.Err()
}

func mapDatabaseError(err error, fallback error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", fallback, err)
}
