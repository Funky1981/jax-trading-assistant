package harnessstate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresStoreRoundTripAndConcurrency(t *testing.T) {
	dsn := os.Getenv("HARNESS_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("HARNESS_POSTGRES_DSN is not configured")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	migrationPath, err := filepath.Abs(filepath.Join("..", "..", "..", "db", "postgres", "migrations", "000075_harness_durable_task_state.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}

	taskID := fmt.Sprintf("integration-harness-%d", time.Now().UTC().UnixNano())
	defer func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_retry_events WHERE task_id=$1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_failure_events WHERE task_id=$1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_compactions WHERE task_id=$1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_checkpoints WHERE task_id=$1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_task_state_versions WHERE task_id=$1`, taskID)
		_, _ = db.ExecContext(ctx, `DELETE FROM harness_tasks WHERE task_id=$1`, taskID)
	}()
	objective := fixtureObjective()
	state := fixtureState()
	state.State.TaskID = taskID
	store, err := NewPostgresStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTask(ctx, CreateTaskRequest{TaskID: taskID, Objective: objective, InitialState: state}); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadStateVersion(ctx, taskID, 1)
	if err != nil || loaded.State.StateVersion != 1 {
		t.Fatalf("round-trip state failed: %+v %v", loaded, err)
	}
	loaded.State.CurrentStage = "integration-stage"
	loaded.State.NextAction = "Create integration checkpoint."
	loaded.State.UpdatedAt = fixtureTime.Add(time.Minute)
	first, err := store.AppendStateVersion(ctx, taskID, 1, loaded, "integration-state-2")
	if err != nil || !first.Applied {
		t.Fatalf("append failed: %+v %v", first, err)
	}
	if _, err := store.AppendStateVersion(ctx, taskID, 1, loaded, "integration-state-2"); err != nil {
		t.Fatal(err)
	}
	stale := loaded
	stale.State.NextAction = "stale writer"
	stale.State.UpdatedAt = fixtureTime.Add(2 * time.Minute)
	if _, err := store.AppendStateVersion(ctx, taskID, 1, stale, "integration-stale"); err == nil {
		t.Fatal("stale writer was accepted")
	} else {
		var conflict *VersionConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("wrong stale-writer error: %v", err)
		}
	}
	cp, err := store.CreateCheckpoint(ctx, CreateCheckpointRequest{TaskID: taskID, StateVersion: 2, IdempotencyKey: "integration-checkpoint", CreatedAt: fixtureTime.Add(3 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCheckpoint(ctx, cp.CheckpointID); err != nil {
		t.Fatal(err)
	}
	compaction, err := store.CompactTask(ctx, CompactRequest{TaskID: taskID, StateVersion: 2, IdempotencyKey: "integration-compaction", CreatedAt: fixtureTime.Add(4 * time.Minute)})
	if err != nil || compaction.CanonicalStorage {
		t.Fatalf("compaction failed: %+v %v", compaction, err)
	}
	bundle, err := store.ResumeTask(ctx, ResumeRequest{TaskID: taskID})
	if err != nil || bundle.State.State.StateVersion != 2 || bundle.Checkpoint.CheckpointID != cp.CheckpointID {
		t.Fatalf("resume failed: %+v %v", bundle, err)
	}
}
