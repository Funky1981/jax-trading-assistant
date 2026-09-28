package workflow

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestRecoverySnapshotRoundTripAndCorruptionFailClosed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "create-recovery"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReconciliationRequired(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "recovery", ActorRole: ActorRecovery, IdempotencyKey: "mark-recovery", Now: now}); err != nil {
		t.Fatal(err)
	}
	payload, err := store.ExportSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreSnapshot(payload)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := restored.Get(ctx, workflow.WorkflowID)
	if err != nil || recovered.State != StateReconciliationRequired {
		t.Fatalf("restored state = %#v, %v", recovered, err)
	}
	if _, err := restored.ResolveReconciliation(ctx, ReconciliationRequest{WorkflowID: workflow.WorkflowID, Actor: "operator-1", ActorRole: ActorOperator, IdempotencyKey: "resolve-recovery", Now: now, Cancel: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := RestoreSnapshot([]byte(`{"version":"jax.workflow.durable_snapshot/v999"}`)); err == nil {
		t.Fatal("unsupported snapshot version restored")
	}
	tampered := bytes.Replace(payload, []byte("content_identity"), []byte("tampered_identity"), 1)
	if _, err := RestoreSnapshot(tampered); err == nil {
		t.Fatal("tampered snapshot restored")
	}
}

func TestRecoveryNeverAutoAdvancesToApproval(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "create-safe-recovery"})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.MarkReconciliationRequired(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "recovery", ActorRole: ActorRecovery, IdempotencyKey: "mark-safe-recovery", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if workflow.State != StateReconciliationRequired {
		t.Fatalf("recovery state = %s", workflow.State)
	}
	if _, err := store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "unsafe-resume", Now: now}); err == nil {
		t.Fatal("reconciliation state auto-advanced to confirmation")
	}
}

func TestRestoreWorkflowStateReconstructsDurableAuditForNextCommand(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	original := NewStore()
	wf, err := original.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "restore-single-create"})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = original.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "restore-single-await", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	events, err := original.Events(ctx, wf.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorkflowState(wf, events, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := restored.Replay(ctx, wf.WorkflowID)
	if err != nil || got.State != StateAwaitingHumanConfirmation {
		t.Fatalf("restored replay=%#v err=%v", got, err)
	}
	// Exact command retry after restart resolves from the reconstructed audit
	// idempotency record instead of appending another semantic event.
	retry, err := restored.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "restore-single-await", Now: now})
	if err != nil || retry.Revision != wf.Revision {
		t.Fatalf("restored idempotent retry=%#v err=%v", retry, err)
	}
}
