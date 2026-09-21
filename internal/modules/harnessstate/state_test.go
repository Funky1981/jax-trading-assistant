package harnessstate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/contextbuilder"
	"jax-trading-assistant/internal/modules/harnesscontracts"
)

var fixtureTime = time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)

func fixtureObjective() harnesscontracts.ResearchObjective {
	return harnesscontracts.ResearchObjective{ContractVersion: harnesscontracts.ResearchObjectiveContractV1, ObjectiveID: "objective-harness-03", Version: "v1", Purpose: "Assess a research hypothesis without execution.", Question: "What is supported, contradicted, and still unknown?", CreatedAt: fixtureTime, Constraints: []string{"NO EXECUTION / RESEARCH ONLY", "Use only as-of eligible evidence."}, RequiredOutputs: []string{"evidence assessment"}, CompletionCriteria: []harnesscontracts.CompletionCriterion{{ID: "criterion-1", Description: "Record support, contradiction, and unknowns."}}, PolicyVersions: []harnesscontracts.PolicyReference{{Name: "research-policy", Version: "v1"}}, Status: harnesscontracts.ObjectiveStatusActive}
}

func fixtureEvidence(id, classification string) harnesscontracts.EvidenceReference {
	hash, _ := harnesscontracts.CanonicalHash(map[string]string{"evidence": id})
	observed := fixtureTime.Add(-time.Hour)
	return harnesscontracts.EvidenceReference{ContractVersion: harnesscontracts.EvidenceReferenceContractV1, EvidenceID: id, CanonicalStore: "evidence-store", ProviderIdentity: "provider-fixture", SourceReference: "source:" + id, ContentHash: hash, ObservedAt: &observed, FirstSeenAt: &observed, PublishedAt: &observed, Classification: classification, QualityReference: "quality:fixture", PolicyReference: harnesscontracts.PolicyReference{Name: "evidence-policy", Version: "v1"}, InformationState: "observed"}
}

func fixtureMemory(id string) harnesscontracts.MemoryReference {
	hash, _ := harnesscontracts.CanonicalHash(map[string]string{"memory": id})
	return harnesscontracts.MemoryReference{ContractVersion: harnesscontracts.MemoryReferenceContractV1, MemoryID: id, MemoryType: "prior-case", SourceTaskReference: "task-source", SourceDecisionReference: "decision-source", SourceOutcomeReference: "outcome-source", CreatedAt: fixtureTime, RegimeReference: "regime:unknown", ReliabilityReference: "reliability:unrated", CalibrationReference: "calibration:unrated", ContentHash: hash, Version: "v1"}
}

func fixtureTool(id, status string) harnesscontracts.ToolReference {
	inputHash, _ := harnesscontracts.CanonicalHash(map[string]string{"input": id})
	started := fixtureTime
	completed := fixtureTime.Add(time.Minute)
	tool := harnesscontracts.ToolReference{ContractVersion: harnesscontracts.ToolReferenceContractV1, ToolInvocationID: id, ToolID: "retrieval-fixture", ToolVersion: "v1", Purpose: "Retrieve deterministic offline evidence", InputHash: inputHash, InputReference: "fixture-input:" + id, StartedAt: started, Status: status, Redacted: true}
	switch status {
	case harnesscontracts.ToolStatusFailed:
		tool.CompletedAt = &completed
		tool.FailureClassification = "tool_unavailable"
	case harnesscontracts.ToolStatusSucceeded:
		outputHash, _ := harnesscontracts.CanonicalHash(map[string]string{"output": id})
		tool.CompletedAt = &completed
		tool.OutputHash = outputHash
		tool.OutputReference = "fixture-output:" + id
	}
	return tool
}

func fixtureState() DurableState {
	return DurableState{State: harnesscontracts.TaskState{ContractVersion: harnesscontracts.TaskStateContractV1, TaskID: "task-harness-03", ObjectiveID: "objective-harness-03", ObjectiveVersion: "v1", StateVersion: 1, CurrentStage: "intake", CompletedWork: []harnesscontracts.WorkItem{{ID: "objective-loaded", Description: "Objective and constraints loaded."}}, PendingWork: []harnesscontracts.WorkItem{{ID: "retrieve-support", Description: "Retrieve supporting evidence."}, {ID: "retrieve-counter", Description: "Retrieve counter-evidence."}}, OpenQuestions: []string{"Which factor is causal?"}, KnownUnknowns: []string{"instrument confirmation is missing"}, Constraints: []string{"NO EXECUTION / RESEARCH ONLY", "No future data."}, Decisions: []harnesscontracts.DecisionRecord{{DecisionID: "decision-scope", Summary: "Keep the task research-only.", Reason: "The objective explicitly excludes execution.", DecidedAt: fixtureTime}}, EvidenceReferences: []harnesscontracts.EvidenceReference{fixtureEvidence("evidence-support", harnesscontracts.EvidenceSupporting)}, CounterEvidenceReferences: []harnesscontracts.EvidenceReference{fixtureEvidence("evidence-counter", harnesscontracts.EvidenceContradictory)}, MemoryReferences: []harnesscontracts.MemoryReference{fixtureMemory("memory-prior-case")}, NextAction: "Retrieve the remaining evidence.", CreatedAt: fixtureTime, UpdatedAt: fixtureTime, Status: harnesscontracts.TaskStatusRunning}, UnresolvedContradictions: []string{"support says factor A matters; counter-evidence says factor A is confounded."}, ToolReferences: []harnesscontracts.ToolReference{fixtureTool("tool-evidence", harnesscontracts.ToolStatusSucceeded)}}
}

func newFixture(t *testing.T) (*MemoryStore, harnesscontracts.ResearchObjective, DurableState) {
	t.Helper()
	store := NewMemoryStore()
	objective := fixtureObjective()
	state := fixtureState()
	if _, err := store.CreateTask(context.Background(), CreateTaskRequest{TaskID: state.State.TaskID, Objective: objective, InitialState: state}); err != nil {
		t.Fatal(err)
	}
	return store, objective, state
}

func advanceState(t *testing.T, store *MemoryStore, state DurableState, mutate func(*DurableState), key string) DurableState {
	t.Helper()
	mutate(&state)
	state.State.UpdatedAt = state.State.UpdatedAt.Add(time.Minute)
	result, err := store.AppendStateVersion(context.Background(), state.State.TaskID, state.State.StateVersion, state, key)
	if err != nil {
		t.Fatal(err)
	}
	return result.State
}

func checkpoint(t *testing.T, store *MemoryStore, state DurableState, key string) CheckpointRecord {
	t.Helper()
	record, err := store.CreateCheckpoint(context.Background(), CreateCheckpointRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: key, CreatedAt: state.State.UpdatedAt, ProvenanceReferences: []string{"fixture-run:v1"}})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestHarness03BehaviouralGates(t *testing.T) {
	ctx := context.Background()
	store, objective, initial := newFixture(t)
	firstCheckpoint := checkpoint(t, store, initial, "checkpoint-1")

	t.Run("state creation", func(t *testing.T) {
		task, err := store.LoadTask(ctx, initial.State.TaskID)
		if err != nil || task.LatestVersion != 1 || task.Objective.ObjectiveID != objective.ObjectiveID {
			t.Fatalf("task creation failed: task=%+v err=%v", task, err)
		}
	})
	t.Run("latest state load", func(t *testing.T) {
		state, err := store.LoadLatestState(ctx, initial.State.TaskID)
		if err != nil || state.State.StateVersion != 1 {
			t.Fatalf("latest state failed: %+v %v", state, err)
		}
	})
	t.Run("append version", func(t *testing.T) {
		state := advanceState(t, store, initial, func(s *DurableState) {
			s.State.CurrentStage = "retrieval"
			s.State.NextAction = "Review retrieved evidence."
		}, "state-2")
		if state.State.StateVersion != 2 {
			t.Fatalf("got v%d", state.State.StateVersion)
		}
	})
	t.Run("historical state remains readable", func(t *testing.T) {
		state, err := store.LoadStateVersion(ctx, initial.State.TaskID, 1)
		if err != nil || state.State.CurrentStage != "intake" {
			t.Fatalf("history lost: %+v %v", state, err)
		}
	})
	t.Run("second checkpoint", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		record := checkpoint(t, store, state, "checkpoint-2")
		if record.StateVersion != 2 {
			t.Fatalf("checkpoint version %d", record.StateVersion)
		}
	})
	t.Run("multiple checkpoints list in order", func(t *testing.T) {
		records, err := store.ListCheckpoints(ctx, initial.State.TaskID)
		if err != nil || len(records) != 2 || records[0].StateVersion != 1 || records[1].StateVersion != 2 {
			t.Fatalf("checkpoints=%+v err=%v", records, err)
		}
	})
	t.Run("optimistic concurrency rejects stale writer", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		var winner, loser error
		var wg sync.WaitGroup
		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func(i int) {
				defer wg.Done()
				candidate := clone(state)
				candidate.State.CurrentStage = "race-" + string(rune('a'+i))
				candidate.State.NextAction = "Continue race."
				candidate.State.UpdatedAt = fixtureTime.Add(3 * time.Minute)
				_, err := store.AppendStateVersion(ctx, initial.State.TaskID, 2, candidate, "race-"+string(rune('a'+i)))
				if i == 0 {
					winner = err
				} else {
					loser = err
				}
			}(i)
		}
		wg.Wait()
		if (winner == nil) == (loser == nil) {
			t.Fatalf("expected exactly one race writer to succeed: winner=%v loser=%v", winner, loser)
		}
		conflict := winner
		if conflict == nil {
			conflict = loser
		}
		var versionConflict *VersionConflictError
		if !errors.As(conflict, &versionConflict) {
			t.Fatalf("loser did not receive explicit conflict: %v", conflict)
		}
	})
	t.Run("duplicate state retry is idempotent", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		candidate := clone(state)
		candidate.State.NextAction = "Idempotent next action."
		candidate.State.UpdatedAt = fixtureTime.Add(4 * time.Minute)
		first, err := store.AppendStateVersion(ctx, initial.State.TaskID, state.State.StateVersion, candidate, "same-state")
		if err != nil {
			t.Fatal(err)
		}
		second, err := store.AppendStateVersion(ctx, initial.State.TaskID, state.State.StateVersion, candidate, "same-state")
		if err != nil || second.Applied || first.State.State.StateVersion != second.State.State.StateVersion {
			t.Fatalf("duplicate retry mutated state: first=%+v second=%+v err=%v", first, second, err)
		}
	})
	t.Run("idempotency key content conflict is explicit", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		candidate := clone(state)
		candidate.State.NextAction = "Different content."
		candidate.State.UpdatedAt = fixtureTime.Add(5 * time.Minute)
		if _, err := store.AppendStateVersion(ctx, initial.State.TaskID, state.State.StateVersion, candidate, "same-state"); !errors.Is(err, ErrIdempotencyConflict) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("checkpoint hash is stable across volatile metadata", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		a := checkpoint(t, store, state, "checkpoint-hash-a")
		b, err := store.CreateCheckpoint(ctx, CreateCheckpointRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, CheckpointID: "checkpoint-hash-b", IdempotencyKey: "checkpoint-hash-b", CreatedAt: fixtureTime.Add(99 * time.Hour), ProvenanceReferences: []string{"fixture-run:v1"}})
		if err != nil || a.ContentHash != b.ContentHash {
			t.Fatalf("volatile metadata changed hash: %s %s %v", a.ContentHash, b.ContentHash, err)
		}
	})
	t.Run("checkpoint semantic change changes hash", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		changed := clone(state)
		changed.State.NextAction = "A semantic change."
		changed.State.UpdatedAt = fixtureTime.Add(6 * time.Minute)
		result, err := store.AppendStateVersion(ctx, initial.State.TaskID, state.State.StateVersion, changed, "semantic-change")
		if err != nil {
			t.Fatal(err)
		}
		next := checkpoint(t, store, result.State, "checkpoint-semantic")
		if next.ContentHash == firstCheckpoint.ContentHash {
			t.Fatal("semantic change did not change hash")
		}
	})
	t.Run("checkpoint corruption is detected", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		record := checkpoint(t, store, state, "checkpoint-corrupt")
		store.mu.Lock()
		task := store.tasks[record.TaskID]
		corrupted := task.checkpoints[record.CheckpointID]
		corrupted.Content.NextAction = "tampered"
		task.checkpoints[record.CheckpointID] = corrupted
		store.mu.Unlock()
		if _, err := store.LoadCheckpoint(ctx, record.CheckpointID); !errors.Is(err, ErrIntegrity) {
			t.Fatalf("corruption not detected: %v", err)
		}
	})
	t.Run("resume loads explicit checkpoint", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if err != nil || bundle.State.State.StateVersion != firstCheckpoint.StateVersion || bundle.Task.Objective.ObjectiveID != objective.ObjectiveID || len(bundle.FailureHistory) != 0 {
			t.Fatalf("resume failed: %+v %v", bundle, err)
		}
	})
	t.Run("latest resume uses latest valid checkpoint", func(t *testing.T) {
		list, _ := store.ListCheckpoints(ctx, initial.State.TaskID)
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil || bundle.Checkpoint.CheckpointID != list[len(list)-1].CheckpointID {
			t.Fatalf("latest resume failed: %+v %v", bundle, err)
		}
	})
	t.Run("resume bundle needs no transcript", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if err != nil {
			t.Fatal(err)
		}
		if bundle.State.State.OpenQuestions[0] == "" || bundle.State.State.KnownUnknowns[0] == "" || len(bundle.State.State.Decisions) == 0 {
			t.Fatal("durable resume omitted required state")
		}
	})
	t.Run("constraints survive resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if !contains(bundle.State.State.Constraints, "NO EXECUTION / RESEARCH ONLY") {
			t.Fatal("hard constraint lost")
		}
	})
	t.Run("decision reason survives resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		decision := bundle.State.State.Decisions[0]
		if decision.DecisionID != "decision-scope" || !strings.Contains(decision.Reason, "excludes execution") {
			t.Fatal("decision provenance lost")
		}
	})
	t.Run("evidence references survive resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if len(bundle.State.State.EvidenceReferences) != 1 || bundle.State.State.EvidenceReferences[0].EvidenceID != "evidence-support" {
			t.Fatal("evidence references lost")
		}
	})
	t.Run("counter-evidence references survive resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if len(bundle.State.State.CounterEvidenceReferences) != 1 || bundle.State.State.CounterEvidenceReferences[0].EvidenceID != "evidence-counter" {
			t.Fatal("counter-evidence references lost")
		}
	})
	t.Run("memory references survive resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if len(bundle.State.State.MemoryReferences) != 1 || bundle.State.State.MemoryReferences[0].MemoryID != "memory-prior-case" {
			t.Fatal("memory references lost")
		}
	})
	t.Run("tool references survive resume", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if len(bundle.State.ToolReferences) != 1 || bundle.State.ToolReferences[0].ToolInvocationID != "tool-evidence" {
			t.Fatal("tool references lost")
		}
	})
	t.Run("unknown remains unknown", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if !contains(bundle.State.State.KnownUnknowns, "instrument confirmation is missing") {
			t.Fatal("known unknown was inferred away")
		}
	})
	t.Run("contradiction remains unresolved", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: firstCheckpoint.CheckpointID})
		if len(bundle.State.UnresolvedContradictions) != 1 || !strings.Contains(bundle.State.UnresolvedContradictions[0], "confounded") {
			t.Fatal("contradiction lost")
		}
	})
	t.Run("failed tool state is representable", func(t *testing.T) {
		current, err := store.LoadLatestState(ctx, initial.State.TaskID)
		if err != nil {
			t.Fatal(err)
		}
		state := advanceState(t, store, current, func(s *DurableState) {
			s.ToolReferences = append(s.ToolReferences, fixtureTool("tool-failed", harnesscontracts.ToolStatusFailed))
			s.State.CurrentStage = "retrieval"
			s.State.NextAction = "Retry failed retrieval."
		}, "tool-failure-state")
		result, err := store.RecordFailure(ctx, initial.State.TaskID, state.State.StateVersion, harnesscontracts.FailureState{Attempt: 1, Retryable: true, Classification: "tool_unavailable", Reason: "required retrieval tool unavailable"}, "Retry failed retrieval.", "failure-1")
		if err != nil {
			t.Fatal(err)
		}
		checkpoint(t, store, result.State, "checkpoint-failure")
	})
	t.Run("failure survives restart", func(t *testing.T) {
		snapshot, err := store.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := RestoreMemoryStore(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := fresh.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil || len(bundle.FailureHistory) == 0 || bundle.FailureHistory[len(bundle.FailureHistory)-1].Failure.Classification != "tool_unavailable" {
			t.Fatalf("failure not durable: %+v %v", bundle, err)
		}
	})
	t.Run("failure duplicate is idempotent", func(t *testing.T) {
		result, err := store.RecordFailure(ctx, initial.State.TaskID, 999, harnesscontracts.FailureState{Attempt: 1, Retryable: true, Classification: "tool_unavailable", Reason: "required retrieval tool unavailable"}, "Retry failed retrieval.", "failure-1")
		if err != nil || result.Applied {
			t.Fatalf("duplicate failure was not idempotent: %+v %v", result, err)
		}
	})
	t.Run("retry appends a new state without rewriting failure history", func(t *testing.T) {
		task, _ := store.LoadTask(ctx, initial.State.TaskID)
		result, err := store.RecordRetry(ctx, initial.State.TaskID, task.LatestVersion, "Retry tool now.", "retry-1")
		if err != nil || result.State.State.Status != harnesscontracts.TaskStatusRunning {
			t.Fatalf("retry failed: %+v %v", result, err)
		}
		checkpoint(t, store, result.State, "checkpoint-retry")
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if len(bundle.FailureHistory) != 1 || len(bundle.RetryHistory) != 1 {
			t.Fatalf("operational history lost: %+v", bundle)
		}
	})
	t.Run("retry duplicate is idempotent", func(t *testing.T) {
		task, _ := store.LoadTask(ctx, initial.State.TaskID)
		result, err := store.RecordRetry(ctx, initial.State.TaskID, task.LatestVersion, "Retry tool now.", "retry-1")
		if err != nil || result.Applied {
			t.Fatalf("duplicate retry was not idempotent: %+v %v", result, err)
		}
	})
	t.Run("compaction is derived not canonical", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		before := clone(state)
		compacted, err := store.CompactTask(ctx, CompactRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: "compact-1", CreatedAt: fixtureTime.Add(10 * time.Hour), ProvenanceReferences: []string{"checkpoint-ref:1"}})
		if err != nil || compacted.CanonicalStorage {
			t.Fatalf("invalid compaction: %+v %v", compacted, err)
		}
		after, _ := store.LoadStateVersion(ctx, state.State.TaskID, state.State.StateVersion)
		if !reflect.DeepEqual(before, after) {
			t.Fatal("compaction mutated canonical state")
		}
	})
	t.Run("compaction retains all loss-sensitive fields", func(t *testing.T) {
		records, _ := store.ListCompactions(ctx, initial.State.TaskID)
		if len(records) == 0 {
			t.Fatal("no compaction")
		}
		content := records[len(records)-1].Content
		if !contains(content.Constraints, "NO EXECUTION / RESEARCH ONLY") || len(content.UnresolvedContradictions) != 1 || !contains(content.KnownUnknowns, "instrument confirmation is missing") || len(content.CounterEvidenceReferences) != 1 || len(content.Decisions) != 1 {
			t.Fatalf("compaction omitted protected fields: %+v", content)
		}
	})
	t.Run("compaction duplicate is idempotent", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		first, err := store.CompactTask(ctx, CompactRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: "compact-1", CreatedAt: fixtureTime.Add(11 * time.Hour), ProvenanceReferences: []string{"checkpoint-ref:1"}})
		if err != nil {
			t.Fatal(err)
		}
		if first.CompactionID == "" {
			t.Fatal("missing compaction identity")
		}
	})
	t.Run("multiple compactions preserve contradiction", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		state = advanceState(t, store, state, func(s *DurableState) {
			s.State.CurrentStage = "analysis"
			s.State.NextAction = "Reconcile contradiction."
		}, "more-work")
		checkpoint(t, store, state, "checkpoint-after-more-work")
		second, err := store.CompactTask(ctx, CompactRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: "compact-2", CreatedAt: fixtureTime.Add(12 * time.Hour), ProvenanceReferences: []string{"checkpoint-ref:2"}})
		if err != nil || len(second.Content.UnresolvedContradictions) != 1 {
			t.Fatalf("second compaction lost contradiction: %+v %v", second, err)
		}
	})
	t.Run("resume after repeated compaction returns newest canonical state", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil || bundle.State.State.CurrentStage != "analysis" || len(bundle.State.UnresolvedContradictions) != 1 {
			t.Fatalf("repeated compaction resume failed: %+v %v", bundle, err)
		}
	})
	t.Run("compaction hash changes on semantic change", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		a, err := store.CompactTask(ctx, CompactRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: "compact-semantic-a", CreatedAt: fixtureTime, ProvenanceReferences: []string{"same"}})
		if err != nil {
			t.Fatal(err)
		}
		changed := advanceState(t, store, state, func(s *DurableState) { s.State.NextAction = "Different action." }, "compaction-semantic-state")
		b, err := store.CompactTask(ctx, CompactRequest{TaskID: changed.State.TaskID, StateVersion: changed.State.StateVersion, IdempotencyKey: "compact-semantic-b", CreatedAt: fixtureTime, ProvenanceReferences: []string{"same"}})
		if err != nil || a.ContentHash == b.ContentHash {
			t.Fatalf("semantic compaction change not hashed: %s %s %v", a.ContentHash, b.ContentHash, err)
		}
	})
	t.Run("fresh process snapshot restores only by TaskID", func(t *testing.T) {
		snapshot, err := store.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		fresh, err := RestoreMemoryStore(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := fresh.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil || bundle.Task.TaskID != initial.State.TaskID || bundle.State.State.ObjectiveID != objective.ObjectiveID {
			t.Fatalf("fresh-session recovery failed: %+v %v", bundle, err)
		}
	})
	t.Run("state versions remain append-only after compaction", func(t *testing.T) {
		task, _ := store.LoadTask(ctx, initial.State.TaskID)
		if task.LatestVersion < 5 {
			t.Fatalf("unexpected history version %d", task.LatestVersion)
		}
		for version := 1; version <= task.LatestVersion; version++ {
			if _, err := store.LoadStateVersion(ctx, initial.State.TaskID, version); err != nil {
				t.Fatalf("state v%d unavailable: %v", version, err)
			}
		}
	})
	t.Run("resume checkpoint validates state identity", func(t *testing.T) {
		otherStore := NewMemoryStore()
		otherState := fixtureState()
		otherState.State.TaskID = "task-foreign"
		if _, err := otherStore.CreateTask(ctx, CreateTaskRequest{TaskID: otherState.State.TaskID, Objective: objective, InitialState: otherState}); err != nil {
			t.Fatal(err)
		}
		other := checkpoint(t, otherStore, otherState, "other-checkpoint")
		other.CheckpointID = "foreign-checkpoint"
		other.TaskID = "task-foreign"
		store.mu.Lock()
		store.tasks[initial.State.TaskID].checkpoints[other.CheckpointID] = other
		store.mu.Unlock()
		if _, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID, CheckpointID: other.CheckpointID}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("foreign checkpoint was accepted: %v", err)
		}
	})
	t.Run("invalid missing checkpoint is explicit", func(t *testing.T) {
		if _, err := store.ResumeTask(ctx, ResumeRequest{TaskID: "missing-task"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("secret in state is rejected", func(t *testing.T) {
		secret := clone(initial)
		secret.State.NextAction = "Call api_key=do-not-persist"
		if _, err := store.AppendStateVersion(ctx, initial.State.TaskID, 999, secret, "state-sensitive-case"); !errors.Is(err, ErrSecretMaterial) {
			t.Fatalf("secret was accepted: %v", err)
		}
	})
	t.Run("secret in checkpoint provenance is rejected", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		_, err := store.CreateCheckpoint(ctx, CreateCheckpointRequest{TaskID: state.State.TaskID, StateVersion: state.State.StateVersion, IdempotencyKey: "checkpoint-sensitive", ProvenanceReferences: []string{"authorization: bearer secret"}})
		if !errors.Is(err, ErrSecretMaterial) {
			t.Fatalf("secret provenance was accepted: %v", err)
		}
	})
	t.Run("resume bundle context seam validates", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil {
			t.Fatal(err)
		}
		budget := fixtureBudget()
		policy := contextbuilder.DefaultBuildPolicy()
		policy.Model = harnesscontracts.ModelReference{Provider: "fixture-provider", Model: "fixture-model", Version: "v1"}
		request, err := bundle.BuildRequest(policy, budget, bundle.State.State.UpdatedAt, fixtureEvidenceRetriever{}, fixtureCounterRetriever{}, nil)
		if err != nil || request.CheckpointReference == nil || len(request.ToolReferences) == 0 {
			t.Fatalf("BuildRequest seam failed: %+v %v", request, err)
		}
	})
	t.Run("contextbuilder resumes deterministically", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil {
			t.Fatal(err)
		}
		policy := contextbuilder.DefaultBuildPolicy()
		policy.Model = harnesscontracts.ModelReference{Provider: "fixture-provider", Model: "fixture-model", Version: "v1"}
		request, err := bundle.BuildRequest(policy, fixtureBudget(), bundle.State.State.UpdatedAt, fixtureEvidenceRetriever{}, fixtureCounterRetriever{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		first, err := contextbuilder.Build(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		second, err := contextbuilder.Build(ctx, request)
		if err != nil || first.Package.ContentHash != second.Package.ContentHash {
			t.Fatalf("context was not deterministic: %v", err)
		}
	})
	t.Run("state hash is ordering-independent", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		reversed := clone(state)
		reversed.State.Constraints[0], reversed.State.Constraints[1] = reversed.State.Constraints[1], reversed.State.Constraints[0]
		a, _ := stateHash(state)
		b, _ := stateHash(reversed)
		if a != b {
			t.Fatal("constraint ordering changed canonical state hash")
		}
	})
	t.Run("checkpoint content order is canonical", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		state.State.Decisions = append(state.State.Decisions, harnesscontracts.DecisionRecord{DecisionID: "decision-z", Summary: "Later decision.", Reason: "Fixture reason.", DecidedAt: fixtureTime})
		a := contentFromState(state, objective, []string{"p2", "p1"})
		b := contentFromState(state, objective, []string{"p1", "p2"})
		ha, _ := checkpointContentHash(a)
		hb, _ := checkpointContentHash(b)
		if ha != hb {
			t.Fatal("canonical checkpoint normalization changed with reference order")
		}
	})
	t.Run("failure state validation is strict", func(t *testing.T) {
		bad := clone(initial)
		bad.State.Status = harnesscontracts.TaskStatusFailed
		bad.State.Failure = harnesscontracts.FailureState{}
		if err := bad.Validate(objective); err == nil {
			t.Fatal("failed state without reason accepted")
		}
	})
	t.Run("completed state cannot retain pending work", func(t *testing.T) {
		bad := clone(initial)
		bad.State.Status = harnesscontracts.TaskStatusCompleted
		bad.State.NextAction = ""
		if err := bad.Validate(objective); err == nil {
			t.Fatal("invalid completed state accepted")
		}
	})
	t.Run("memory snapshot is deterministic", func(t *testing.T) {
		a, err := store.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		b, err := store.Snapshot()
		if err != nil || string(a) != string(b) {
			t.Fatal("snapshot bytes changed without state change")
		}
	})
	t.Run("restored snapshot preserves historical state", func(t *testing.T) {
		data, _ := store.Snapshot()
		fresh, err := RestoreMemoryStore(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fresh.LoadStateVersion(ctx, initial.State.TaskID, 1); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("checkpoint list excludes compaction records", func(t *testing.T) {
		checkpoints, _ := store.ListCheckpoints(ctx, initial.State.TaskID)
		compactions, _ := store.ListCompactions(ctx, initial.State.TaskID)
		for _, record := range checkpoints {
			for _, compacted := range compactions {
				if record.CheckpointID == compacted.CompactionID {
					t.Fatal("derived compaction entered checkpoint ledger")
				}
			}
		}
	})
	t.Run("compaction list is historical", func(t *testing.T) {
		records, err := store.ListCompactions(ctx, initial.State.TaskID)
		if err != nil || len(records) < 3 {
			t.Fatalf("compaction history incomplete: %d %v", len(records), err)
		}
	})
	t.Run("tool failure continuation does not repeat completed work", func(t *testing.T) {
		bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if err != nil {
			t.Fatal(err)
		}
		if !containsWork(bundle.State.State.CompletedWork, "objective-loaded") {
			t.Fatal("completed work was repeated/lost")
		}
		if !containsWork(bundle.State.State.PendingWork, "retrieve-support") {
			t.Fatal("pending work was incorrectly completed")
		}
	})
	t.Run("failure history is separate from current running state", func(t *testing.T) {
		bundle, _ := store.ResumeTask(ctx, ResumeRequest{TaskID: initial.State.TaskID})
		if bundle.State.State.Status != harnesscontracts.TaskStatusRunning || len(bundle.FailureHistory) != 1 {
			t.Fatalf("failure/current state boundary lost: %+v", bundle)
		}
	})
	t.Run("checkpoint captures objective identity and version", func(t *testing.T) {
		record, _ := store.LoadCheckpoint(ctx, firstCheckpoint.CheckpointID)
		if record.Content.Objective.ObjectiveID != objective.ObjectiveID || record.Content.Objective.ObjectiveVersion != "v1" {
			t.Fatal("objective identity missing")
		}
	})
	t.Run("checkpoint captures current stage", func(t *testing.T) {
		record, _ := store.LoadCheckpoint(ctx, firstCheckpoint.CheckpointID)
		if record.Content.CurrentStage != "intake" {
			t.Fatal("current stage missing")
		}
	})
	t.Run("checkpoint captures completed and pending work", func(t *testing.T) {
		record, _ := store.LoadCheckpoint(ctx, firstCheckpoint.CheckpointID)
		if len(record.Content.CompletedWork) != 1 || len(record.Content.PendingWork) != 2 {
			t.Fatal("work state missing")
		}
	})
	t.Run("checkpoint captures next action", func(t *testing.T) {
		record, _ := store.LoadCheckpoint(ctx, firstCheckpoint.CheckpointID)
		if record.Content.NextAction == "" {
			t.Fatal("next action missing")
		}
	})
	t.Run("checkpoint captures provenance references", func(t *testing.T) {
		record, _ := store.LoadCheckpoint(ctx, firstCheckpoint.CheckpointID)
		if !contains(record.Content.ProvenanceReferences, "fixture-run:v1") {
			t.Fatal("provenance missing")
		}
	})
	t.Run("state version hash changes with state version", func(t *testing.T) {
		state, _ := store.LoadStateVersion(ctx, initial.State.TaskID, 1)
		state2, _ := store.LoadStateVersion(ctx, initial.State.TaskID, 2)
		a, _ := stateHash(state)
		b, _ := stateHash(state2)
		if a == b {
			t.Fatal("state versions share identity")
		}
	})
	t.Run("unknown task state returns not found", func(t *testing.T) {
		if _, err := store.LoadStateVersion(ctx, initial.State.TaskID, 99999); !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unknown checkpoint returns not found", func(t *testing.T) {
		if _, err := store.LoadCheckpoint(ctx, "missing-checkpoint"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unknown compaction returns not found", func(t *testing.T) {
		if _, err := store.LoadCompaction(ctx, "missing-compaction"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("empty operation identity is rejected", func(t *testing.T) {
		state, _ := store.LoadLatestState(ctx, initial.State.TaskID)
		if _, err := store.AppendStateVersion(ctx, initial.State.TaskID, state.State.StateVersion, state, ""); !errors.Is(err, ErrInvalid) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("checkpoint state version cannot be future", func(t *testing.T) {
		_, err := store.CreateCheckpoint(ctx, CreateCheckpointRequest{TaskID: initial.State.TaskID, StateVersion: 9999, IdempotencyKey: "future-checkpoint"})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("compaction stores references rather than evidence payloads", func(t *testing.T) {
		records, err := store.ListCompactions(ctx, initial.State.TaskID)
		if err != nil || len(records) == 0 {
			t.Fatalf("no compaction records: %v", err)
		}
		for _, reference := range records[0].Content.EvidenceReferences {
			if reference.ContentHash == "" || reference.SourceReference == "" {
				t.Fatal("compaction did not preserve canonical evidence reference")
			}
		}
	})
	t.Run("PAPER-02 firewall has no harnessstate import", func(t *testing.T) {
		_, file, _, _ := runtime.Caller(0)
		root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
		trader := filepath.Join(root, "cmd", "trader")
		var violation string
		_ = filepath.WalkDir(trader, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() {
				return walkErr
			}
			data, _ := os.ReadFile(path)
			if strings.Contains(string(data), "internal/modules/harnessstate") {
				violation = path
			}
			return nil
		})
		if violation != "" {
			t.Fatalf("cmd/trader imports HARNESS-03: %s", violation)
		}
	})
	if t.Failed() {
		t.Logf("fixture task reached v%d", mustTask(t, store, initial.State.TaskID).LatestVersion)
	}
}

func TestHarness03FailureRetryIdempotency(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	objective := fixtureObjective()
	firstState := fixtureState()
	secondState := fixtureState()
	secondState.State.TaskID = "task-harness-03-second"
	if _, err := store.CreateTask(ctx, CreateTaskRequest{TaskID: firstState.State.TaskID, Objective: objective, InitialState: firstState}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(ctx, CreateTaskRequest{TaskID: secondState.State.TaskID, Objective: objective, InitialState: secondState}); err != nil {
		t.Fatal(err)
	}
	failure := harnesscontracts.FailureState{Attempt: 1, Retryable: true, Classification: "tool_unavailable", Reason: "retrieval tool unavailable"}

	first, err := store.RecordFailure(ctx, firstState.State.TaskID, 1, failure, "Retry retrieval.", "shared-failure-key")
	if err != nil || !first.Applied {
		t.Fatalf("initial failure transition failed: %+v %v", first, err)
	}
	duplicate, err := store.RecordFailure(ctx, firstState.State.TaskID, 999, failure, "Retry retrieval.", "shared-failure-key")
	if err != nil || duplicate.Applied {
		t.Fatalf("same failure replay was not idempotent: %+v %v", duplicate, err)
	}
	conflictingFailure := failure
	conflictingFailure.Reason = "different failure reason"
	if _, err := store.RecordFailure(ctx, firstState.State.TaskID, 999, conflictingFailure, "Retry retrieval.", "shared-failure-key"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different failure content was accepted: %v", err)
	}
	if _, err := store.RecordFailure(ctx, firstState.State.TaskID, 999, failure, "Different next action.", "shared-failure-key"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different failure next action was accepted: %v", err)
	}

	second, err := store.RecordFailure(ctx, secondState.State.TaskID, 1, failure, "Retry retrieval.", "shared-failure-key")
	if err != nil || !second.Applied {
		t.Fatalf("same failure key collided across tasks: %+v %v", second, err)
	}

	retry, err := store.RecordRetry(ctx, firstState.State.TaskID, first.State.State.StateVersion, "Retry tool now.", "shared-retry-key")
	if err != nil || !retry.Applied {
		t.Fatalf("initial retry transition failed: %+v %v", retry, err)
	}
	retryDuplicate, err := store.RecordRetry(ctx, firstState.State.TaskID, 999, "Retry tool now.", "shared-retry-key")
	if err != nil || retryDuplicate.Applied {
		t.Fatalf("same retry replay was not idempotent: %+v %v", retryDuplicate, err)
	}
	if _, err := store.RecordRetry(ctx, firstState.State.TaskID, 999, "Different retry action.", "shared-retry-key"); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different retry content was accepted: %v", err)
	}

	secondRetry, err := store.RecordRetry(ctx, secondState.State.TaskID, second.State.State.StateVersion, "Retry tool now.", "shared-retry-key")
	if err != nil || !secondRetry.Applied {
		t.Fatalf("same retry key collided across tasks: %+v %v", secondRetry, err)
	}
}

func fixtureBudget() harnesscontracts.ContextBudget {
	budget := harnesscontracts.ContextBudget{ContractVersion: harnesscontracts.ContextBudgetContractV1, SystemPolicy: 1, Objective: 1, TaskState: 2, Evidence: 2, CounterEvidence: 1, CounterEvidenceReserve: 1, Memory: 0, ToolOutputs: 1, WorkingAllowance: 1, StructuredOutputAllowance: 1, RequireCounterEvidence: true}
	budget.Total = budget.SystemPolicy + budget.Objective + budget.TaskState + budget.Evidence + budget.CounterEvidence + budget.Memory + budget.ToolOutputs + budget.WorkingAllowance + budget.StructuredOutputAllowance
	return budget
}

type fixtureEvidenceRetriever struct{}

func (fixtureEvidenceRetriever) Retrieve(context.Context, contextbuilder.RetrievalRequest) (contextbuilder.EvidenceRetrievalResult, error) {
	ref := fixtureEvidence("evidence-support", harnesscontracts.EvidenceSupporting)
	return contextbuilder.EvidenceRetrievalResult{RetrieverID: "fixture-evidence", RetrieverVersion: "v1", RetrievedAt: fixtureTime, SearchStatus: contextbuilder.SearchPerformed, Candidates: []contextbuilder.EvidenceCandidate{{Reference: ref, Factors: contextbuilder.RankingFactors{ObjectiveRelevance: 100}, EstimatedUnits: 1, DeduplicationKey: ref.EvidenceID, PolicyEligible: true}}}, nil
}

type fixtureCounterRetriever struct{}

func (fixtureCounterRetriever) RetrieveContradictions(context.Context, contextbuilder.RetrievalRequest) (contextbuilder.ContradictionRetrievalResult, error) {
	ref := fixtureEvidence("evidence-counter", harnesscontracts.EvidenceContradictory)
	return contextbuilder.ContradictionRetrievalResult{RetrieverID: "fixture-counter", RetrieverVersion: "v1", RetrievedAt: fixtureTime, SearchStatus: contextbuilder.SearchPerformed, Candidates: []contextbuilder.EvidenceCandidate{{Reference: ref, Factors: contextbuilder.RankingFactors{ObjectiveRelevance: 100}, EstimatedUnits: 1, DeduplicationKey: ref.EvidenceID, PolicyEligible: true}}}, nil
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func containsWork(values []harnesscontracts.WorkItem, wanted string) bool {
	for _, value := range values {
		if value.ID == wanted {
			return true
		}
	}
	return false
}
func mustTask(t *testing.T, store *MemoryStore, taskID string) TaskRecord {
	t.Helper()
	task, err := store.LoadTask(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	return task
}
