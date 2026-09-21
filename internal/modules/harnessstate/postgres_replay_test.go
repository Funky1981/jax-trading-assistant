package harnessstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
	"jax-trading-assistant/internal/testsupport"
)

func TestPostgresSameKeyConcurrentCommands(t *testing.T) {
	db, err := sql.Open("pgx", testsupport.PostgresDSN(t, "HARNESS_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(16)
	ctx := context.Background()
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"failure", "retry", "checkpoint", "compaction"} {
		t.Run(operation, func(t *testing.T) {
			taskID := fmt.Sprintf("integration-concurrent-%s-%d", operation, time.Now().UnixNano())
			state := fixtureState()
			state.State.TaskID = taskID
			if _, err := store.CreateTask(ctx, CreateTaskRequest{TaskID: taskID, Objective: fixtureObjective(), InitialState: state}); err != nil {
				t.Fatal(err)
			}
			failure := harnesscontracts.FailureState{Classification: "tool_unavailable", Reason: "fixture retryable persistence failure", Retryable: true, Attempt: 1}
			if operation == "retry" {
				if _, err := store.RecordFailure(ctx, taskID, 1, failure, "Retry fixture operation.", "initial-failure"); err != nil {
					t.Fatal(err)
				}
			}
			run := func(conflict bool) (bool, string, error) {
				next := "Continue fixture work."
				provenance := []string{"fixture:original"}
				if conflict {
					next = "Different semantic request."
					provenance = []string{"fixture:changed"}
				}
				switch operation {
				case "failure":
					r, err := store.RecordFailure(ctx, taskID, 1, failure, next, "same-key")
					return r.Applied, fmt.Sprint(r.State.State.StateVersion), err
				case "retry":
					r, err := store.RecordRetry(ctx, taskID, 2, next, "same-key")
					return r.Applied, fmt.Sprint(r.State.State.StateVersion), err
				case "checkpoint":
					r, err := store.CreateCheckpoint(ctx, CreateCheckpointRequest{TaskID: taskID, StateVersion: 1, IdempotencyKey: "same-key", ProvenanceReferences: provenance, CreatedAt: fixtureTime})
					return false, r.CheckpointID, err
				default:
					r, err := store.CompactTask(ctx, CompactRequest{TaskID: taskID, StateVersion: 1, IdempotencyKey: "same-key", ProvenanceReferences: provenance, CreatedAt: fixtureTime})
					return false, r.CompactionID, err
				}
			}
			type result struct {
				applied  bool
				identity string
				err      error
			}
			results := make(chan result, 12)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for range 12 {
				wg.Add(1)
				go func() { defer wg.Done(); <-start; applied, id, err := run(false); results <- result{applied, id, err} }()
			}
			close(start)
			wg.Wait()
			close(results)
			applied, identity := 0, ""
			for r := range results {
				if r.err != nil {
					t.Errorf("same-key concurrent replay failed: %v", r.err)
					continue
				}
				if r.applied {
					applied++
				}
				if identity == "" {
					identity = r.identity
				}
				if r.identity != identity {
					t.Errorf("replay returned another identity")
				}
			}
			if (operation == "failure" || operation == "retry") && applied != 1 {
				t.Errorf("applied=%d, want exactly one", applied)
			}
			latest, err := store.LoadLatestState(ctx, taskID)
			if err != nil {
				t.Fatalf("load final durable state: %v", err)
			}
			wantVersion := 1
			if operation == "failure" {
				wantVersion = 2
			}
			if operation == "retry" {
				wantVersion = 3
			}
			if latest.State.StateVersion != wantVersion {
				t.Errorf("final state version=%d, want %d", latest.State.StateVersion, wantVersion)
			}
			table := map[string]string{
				"failure":    "harness_failure_events",
				"retry":      "harness_retry_events",
				"checkpoint": "harness_checkpoints",
				"compaction": "harness_compactions",
			}[operation]
			var rows int
			if err := db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE task_id=$1 AND state_version=$2", table), taskID, wantVersion).Scan(&rows); err != nil {
				t.Fatalf("count durable %s rows: %v", operation, err)
			}
			if rows != 1 {
				t.Errorf("durable %s rows=%d, want exactly one", operation, rows)
			}
			if _, _, err := run(true); !errors.Is(err, ErrIdempotencyConflict) {
				t.Errorf("changed semantics: %v", err)
			}
		})
	}
}
