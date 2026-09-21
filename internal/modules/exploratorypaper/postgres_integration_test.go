package exploratorypaper

import (
	"context"
	"errors"
	"jax-trading-assistant/internal/testsupport"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/workflow"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This test is opt-in because package tests must remain runnable without a
// local database. CI/review execution sets PAPER01B_DATABASE_URL after the
// normal migration service is healthy.
func TestPostgresRestartRestoresExploratoryLifecycle(t *testing.T) {
	databaseURL := testsupport.PostgresDSN(t, "PAPER01B_DATABASE_URL")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	suffix := now.Format("20060102150405.000000")
	thesis := fixtureThesis()
	thesis.ThesisID = "thesis-pg-" + suffix
	thesis.EventID = "event-pg-" + suffix
	thesis.CreatedAt, thesis.CandidateGeneratedAt, thesis.EventTimestamp = now, now, now.Add(-time.Hour)
	approval := postgresApproval(t, "pg-"+suffix, "LONG", now)
	intent := approval.PaperIntent
	binding := fixtureBinding(thesis)
	binding.CandidateID = "candidate-pg-" + suffix
	binding.WorkflowID, binding.PaperIntentID, binding.BoundAt = approval.Workflow.WorkflowID, intent.IntentID, now
	binding.ThesisContentHash, binding.EvidenceSetHash = ThesisContentHash(thesis), EvidenceSetFingerprint(append(append([]EvidenceReference{}, thesis.Evidence...), thesis.CounterEvidence...))
	costModel := papertrading.DefaultCostModel()
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), costModel)
	if err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: approval.Workflow, PaperIntent: intent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costModel, InstrumentID: thesis.InstrumentID, Quantity: 10, ReferencePrice: 100, OrderType: papertrading.OrderMarket, CreatedAt: now, IdempotencyKey: "pg-entry-" + suffix})
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := papertrading.NewPaperLedger("account-pg-"+suffix, "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	fills, err := venue.ProcessTick(papertrading.MarketTick{TickID: "tick-pg-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(time.Second), ReceivedAt: now.Add(time.Second), Session: papertrading.SessionOpen, Source: "paper-integration"})
	if err != nil || len(fills) != 1 {
		t.Fatalf("fills=%#v err=%v", fills, err)
	}
	account, err := ledger.ApplyFill(fills[0])
	if err != nil {
		t.Fatal(err)
	}
	position, err := OpenApprovedPosition("position-pg-"+suffix, thesis, binding, fills[0].FilledAt, fills[0].Price)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresStore(pool)
	lifecycleID, err := store.SaveApprovedEntry(ctx, binding.CandidateID, thesis, binding, approval, position, EntrySnapshot{Order: order, Fill: fills[0], Ledger: account})
	if err != nil {
		t.Fatal(err)
	}
	calendar := SessionCalendar{Sessions: map[string]bool{now.Format("2006-01-02"): true, now.AddDate(0, 0, 1).Format("2006-01-02"): true, now.AddDate(0, 0, 2).Format("2006-01-02"): true, now.AddDate(0, 0, 3).Format("2006-01-02"): true, now.AddDate(0, 0, 4).Format("2006-01-02"): true}, Timezone: "UTC", OpenTime: "00:00", CloseTime: "23:59"}
	schedule, err := BuildReviewSchedule(position, calendar)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PersistReviewSchedule(ctx, position.PositionID, schedule); err != nil {
		t.Fatal(err)
	}
	if err := store.PersistReviewSchedule(ctx, position.PositionID, schedule); err != nil {
		t.Fatal(err)
	}

	// Recreate the store to model a process restart. Get must restore and
	// validate the durable workflow/paper identities, not only JSON state.
	store = NewPostgresStore(pool)
	restored, err := store.Get(ctx, position.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.LifecycleID != lifecycleID || restored.Thesis.Thesis.ThesisID != thesis.ThesisID || restored.Binding.WorkflowID != approval.Workflow.WorkflowID || restored.Binding.PaperIntentID != intent.IntentID || restored.EntryOrder.OrderID != order.OrderID || restored.EntryFill.FillID != fills[0].FillID || restored.Position.PositionID != position.PositionID || len(restored.Reviews) != 5 {
		t.Fatalf("restored lifecycle lost identity: %#v", restored)
	}
	if err := store.RecordReviewUnavailable(ctx, position.PositionID, 1, now.Add(2*time.Hour), "integration fixture has no required observation"); err != nil {
		t.Fatal(err)
	}
	if err := store.RecordReviewUnavailable(ctx, position.PositionID, 1, now.Add(3*time.Hour), "duplicate scheduler delivery"); err != nil {
		t.Fatal(err)
	}

	evidence := RelevantEvidence{Reference: EvidenceReference{EvidenceID: "evidence-pg-" + suffix, SourceID: "source-pg", SourceURL: "https://example.test/pg", Quality: "high", ObservedAt: now}, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, Signal: EvidenceInvalidates, Reason: "restart integration invalidation"}
	if _, err := store.ApplyEvidence(ctx, position.PositionID, evidence, now.Add(time.Hour), thesis.PolicyVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyEvidence(ctx, position.PositionID, evidence, now.Add(2*time.Hour), thesis.PolicyVersion); err != nil {
		t.Fatal(err)
	}
	conflict := evidence
	conflict.Reason = "conflicting replay"
	if _, err := store.ApplyEvidence(ctx, position.PositionID, conflict, now.Add(3*time.Hour), thesis.PolicyVersion); err == nil {
		t.Fatal("conflicting same-ID evidence replay was accepted")
	}

	exitAt := now.Add(2 * time.Hour)
	exitDecision := ExitDecision{Action: ExitNow, Reason: ExitThesisInvalidated}
	if len(restored.Reviews) == 0 {
		t.Fatal("restored lifecycle has no review for durable exit approval")
	}
	review := restored.Reviews[1]
	exitRecommendationID := exitRecommendationIdentity(review, exitDecision)
	if err := store.PersistExitRecommendation(ctx, position.PositionID, exitDecision, review); err != nil {
		t.Fatal(err)
	}
	exitApproval := postgresExitApproval(t, exitRecommendationID, "pg-exit-"+suffix, "SHORT", exitAt, position.PositionID, review.ReviewID, binding.WorkflowID, binding.PaperIntentID)
	if _, err := store.ApproveExit(ctx, restored, exitDecision, ReviewObservation{}); !errors.Is(err, ErrHumanApprovalPending) {
		t.Fatalf("exit approval source before operator decision err=%v, want ErrHumanApprovalPending", err)
	}
	exitOrder, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: exitApproval.Workflow, PaperIntent: exitApproval.PaperIntent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costModel, InstrumentID: thesis.InstrumentID, Quantity: 10, ReferencePrice: 104, OrderType: papertrading.OrderMarket, CreatedAt: exitAt, IdempotencyKey: "pg-exit-" + suffix})
	if err != nil {
		t.Fatal(err)
	}
	exitFills, err := venue.ProcessTick(papertrading.MarketTick{TickID: "tick-pg-exit-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 103, Ask: 104, Last: 104, AvailableQuantity: 10, Timestamp: exitAt.Add(time.Second), ReceivedAt: exitAt.Add(time.Second), Session: papertrading.SessionOpen, Source: "paper-integration"})
	if err != nil || len(exitFills) != 1 {
		t.Fatalf("exit fills=%#v err=%v", exitFills, err)
	}
	exitAccount, err := ledger.ApplyFill(exitFills[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.Position.Close(exitFills[0].FilledAt); err != nil {
		t.Fatal(err)
	}
	if err := store.PersistExitApproval(ctx, position.PositionID, review, exitApproval); err != nil {
		t.Fatal(err)
	}
	if err := store.PersistExitApproval(ctx, position.PositionID, review, exitApproval); err != nil {
		t.Fatalf("same durable exit approval replay was not idempotent: %v", err)
	}
	if sourceApproval, err := store.ApproveExit(ctx, restored, exitDecision, ReviewObservation{}); err != nil || sourceApproval.Workflow.WorkflowID != exitApproval.Workflow.WorkflowID {
		t.Fatalf("durable exit approval source = %#v, err=%v", sourceApproval, err)
	}
	outcome, err := BuildOutcomeFromFills(restored.Position, binding, fills[0], exitFills[0], ExitThesisInvalidated, 2, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PersistOutcome(ctx, position.PositionID, ExitSnapshot{Order: exitOrder, Fill: exitFills[0], Ledger: exitAccount, Approval: &exitApproval}, outcome); err != nil {
		t.Fatal(err)
	}
	finalStore := NewPostgresStore(pool)
	finalRecord, err := finalStore.Get(ctx, position.PositionID)
	approvedReview := finalRecord.Reviews[1]
	if err != nil || finalRecord.Outcome == nil || finalRecord.Outcome.ExitFillID != exitFills[0].FillID || finalRecord.Position.State != StateClosed || approvedReview.Status != "COMPLETED" || approvedReview.ExitApproval == nil {
		t.Fatalf("final outcome was not restored: record=%#v err=%v", finalRecord, err)
	}
}

func postgresApproval(t *testing.T, suffix, direction string, now time.Time) ApprovalSnapshot {
	return postgresApprovalForRecommendation(t, "recommendation-"+suffix, suffix, direction, now)
}

func postgresExitApproval(t *testing.T, recommendationID, suffix, direction string, now time.Time, positionID, reviewID, entryWorkflowID, entryPaperIntentID string) ApprovalSnapshot {
	approval := postgresApprovalForRecommendation(t, recommendationID, suffix, direction, now)
	approval.ExitBinding = &ExitApprovalBinding{
		PositionID:         positionID,
		ReviewID:           reviewID,
		RecommendationID:   recommendationID,
		EntryWorkflowID:    entryWorkflowID,
		EntryPaperIntentID: entryPaperIntentID,
	}
	return approval
}

func postgresApprovalForRecommendation(t *testing.T, recommendationID, suffix, direction string, now time.Time) ApprovalSnapshot {
	t.Helper()
	value := 1000.0
	if direction == "SHORT" {
		value = 1200
	}
	risk := acceptedRisk(t, recommendationID, now, value)
	workflowStore := workflow.NewStore()
	wf, err := workflowStore.Create(context.Background(), workflow.CreateRequest{RiskDecision: risk, Now: now, IdempotencyKey: suffix + "-create"})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = workflowStore.RequestHumanConfirmation(context.Background(), workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: suffix + "-await", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := workflow.NewConfirmation(wf, "AAPL", direction, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("operator-1", workflow.ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = workflowStore.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: workflow.ActorHuman, IdempotencyKey: suffix + "-approve", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wf, intent, err := workflowStore.CreatePaperIntent(context.Background(), workflow.PaperIntentRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: suffix + "-intent", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	events, err := workflowStore.Events(context.Background(), wf.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	return ApprovalSnapshot{Workflow: wf, Events: events, PaperIntent: intent}
}
