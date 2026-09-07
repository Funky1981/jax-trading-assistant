package workflow

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestConcurrentHumanDecisionsHaveExactlyOneAuthoritativeWinner(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "concurrent-create"})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "concurrent-request", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	approve, err := NewConfirmation(workflow, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	approve, err = approve.WithDecision("operator-approve", ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	reject, err := NewConfirmation(workflow, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	reject, err = reject.WithDecision("operator-reject", ConfirmationReject)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		workflow Workflow
		err      error
	}
	results := make(chan result, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		value, callErr := store.Confirm(ctx, HumanConfirmationRequest{WorkflowID: workflow.WorkflowID, Confirmation: approve, Actor: "operator-approve", ActorRole: ActorHuman, IdempotencyKey: "concurrent-approve", Now: now})
		results <- result{workflow: value, err: callErr}
	}()
	go func() {
		defer group.Done()
		value, callErr := store.Confirm(ctx, HumanConfirmationRequest{WorkflowID: workflow.WorkflowID, Confirmation: reject, Actor: "operator-reject", ActorRole: ActorHuman, IdempotencyKey: "concurrent-reject", Now: now})
		results <- result{workflow: value, err: callErr}
	}()
	group.Wait()
	close(results)

	winners := 0
	losers := 0
	for value := range results {
		if value.err == nil {
			winners++
			if value.workflow.State != StateHumanApproved && value.workflow.State != StateHumanRejected {
				t.Fatalf("winning state = %s", value.workflow.State)
			}
		} else {
			losers++
		}
	}
	if winners != 1 || losers != 1 {
		t.Fatalf("concurrent decision results = winners %d, losers %d", winners, losers)
	}
	events, err := store.Events(ctx, workflow.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("audit events = %d, want exactly one decision event", len(events))
	}
}

func TestConcurrentPaperIntentRetriesAreIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "concurrent-intent-create"})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.RequestHumanConfirmation(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "concurrent-intent-request", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := NewConfirmation(workflow, "AAPL", "LONG", now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("operator-1", ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err = store.Confirm(ctx, HumanConfirmationRequest{WorkflowID: workflow.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: ActorHuman, IdempotencyKey: "concurrent-intent-approve", Now: now})
	if err != nil {
		t.Fatal(err)
	}

	request := PaperIntentRequest{WorkflowID: workflow.WorkflowID, Actor: "workflow-system", ActorRole: ActorSystem, IdempotencyKey: "concurrent-intent-create-paper", Now: now}
	results := make(chan struct {
		intent PaperIntent
		err    error
	}, 2)
	var group sync.WaitGroup
	group.Add(2)
	for range 2 {
		go func() {
			defer group.Done()
			_, intent, callErr := store.CreatePaperIntent(ctx, request)
			results <- struct {
				intent PaperIntent
				err    error
			}{intent: intent, err: callErr}
		}()
	}
	group.Wait()
	close(results)
	var identity string
	for value := range results {
		if value.err != nil {
			t.Fatal(value.err)
		}
		if identity == "" {
			identity = value.intent.IntentID
		} else if value.intent.IntentID != identity {
			t.Fatalf("idempotent intent identities differ: %s and %s", identity, value.intent.IntentID)
		}
	}
	events, err := store.Events(ctx, workflow.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("audit events = %d, want one paper-intent event", len(events))
	}
}
