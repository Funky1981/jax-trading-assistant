package exploratorypaper

import (
	"context"
	"errors"
	"jax-trading-assistant/internal/testsupport"
	"math"
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
	t.Cleanup(func() {
		pool.Close()
	})
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
	persistedEntryOrder, ok := venue.Snapshot().Orders[order.OrderID]
	if !ok {
		t.Fatalf("filled entry order %s was not present in venue snapshot", order.OrderID)
	}
	assertOrderReconcilesWithFills(t, persistedEntryOrder, fills)
	position, err := OpenApprovedPosition("position-pg-"+suffix, thesis, binding, fills[0].FilledAt, fills[0].Price)
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgresStore(pool)
	lifecycleID, err := store.SaveApprovedEntry(ctx, binding.CandidateID, thesis, binding, approval, position, EntrySnapshot{Order: persistedEntryOrder, Fill: fills[0], Ledger: account})
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
	persistedExitOrder, ok := venue.Snapshot().Orders[exitOrder.OrderID]
	if !ok {
		t.Fatalf("filled exit order %s was not present in venue snapshot", exitOrder.OrderID)
	}
	assertOrderReconcilesWithFills(t, persistedExitOrder, exitFills)
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
	rejectionDecision := ExitDecision{Action: ExitNow, Reason: ExitRiskKill}
	rejectionReview := restored.Reviews[2]
	rejectionRecommendationID := exitRecommendationIdentity(rejectionReview, rejectionDecision)
	if err := store.PersistExitRecommendation(ctx, position.PositionID, rejectionDecision, rejectionReview); err != nil {
		t.Fatal(err)
	}
	exitRejection := postgresExitRejection(t, rejectionRecommendationID, "pg-reject-"+suffix, "SHORT", exitAt, position.PositionID, rejectionReview.ReviewID, binding.WorkflowID, binding.PaperIntentID)
	if err := store.PersistExitDecision(ctx, position.PositionID, exitRejection); err != nil {
		t.Fatalf("durable exit rejection was not persisted: %v", err)
	}
	if err := store.PersistExitDecision(ctx, position.PositionID, exitRejection); err != nil {
		t.Fatalf("same durable exit rejection replay was not idempotent: %v", err)
	}
	if sourceRejection, err := store.ApproveExit(ctx, restored, rejectionDecision, ReviewObservation{}); err != nil || sourceRejection.Workflow.State != workflow.StateHumanRejected {
		t.Fatalf("durable exit rejection source = %#v, err=%v", sourceRejection, err)
	}
	outcome, err := BuildOutcomeFromFills(restored.Position, binding, fills[0], exitFills[0], ExitThesisInvalidated, 2, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PersistOutcome(ctx, position.PositionID, ExitSnapshot{Order: persistedExitOrder, Fill: exitFills[0], Ledger: exitAccount, Approval: &exitApproval}, outcome); err != nil {
		t.Fatal(err)
	}
	finalStore := NewPostgresStore(pool)
	finalRecord, err := finalStore.Get(ctx, position.PositionID)
	approvedReview := finalRecord.Reviews[1]
	rejectedReview := finalRecord.Reviews[2]
	if err != nil || finalRecord.Outcome == nil || finalRecord.Outcome.ExitFillID != exitFills[0].FillID || finalRecord.Position.State != StateClosed || approvedReview.Status != "COMPLETED" || approvedReview.ExitApproval == nil || rejectedReview.Status != "EXIT_REJECTED" || rejectedReview.ExitApproval == nil {
		t.Fatalf("final outcome was not restored: record=%#v err=%v", finalRecord, err)
	}
}

type scopedPaperArtifact struct {
	account  papertrading.PaperAccount
	order    papertrading.PaperOrder
	fill     papertrading.PaperFill
	approval ApprovalSnapshot
}

func TestPostgresPaperVenueRestoreIsAccountScoped(t *testing.T) {
	databaseURL := testsupport.PostgresDSN(t, "PAPER01B_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
	})
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	suffix := now.Format("20060102150405.000000")
	accountA := "scope-account-a-" + suffix
	accountB := "scope-account-b-" + suffix
	artifactA := persistScopedPaperArtifact(t, ctx, pool, accountA, "scope-a-"+suffix, now)
	artifactB := persistScopedPaperArtifact(t, ctx, pool, accountB, "scope-b-"+suffix, now.Add(time.Minute))
	store := NewPostgresStore(pool)
	costModel := papertrading.DefaultCostModel()
	contract := papertrading.DefaultPaperCapabilityContract()

	venueA, err := store.RestorePaperVenueForAccount(ctx, accountA, contract, costModel)
	if err != nil {
		t.Fatal(err)
	}
	snapshotA := venueA.Snapshot()
	if len(snapshotA.Orders) != 1 || len(snapshotA.Fills) != 1 {
		t.Fatalf("account A restore orders=%d fills=%d", len(snapshotA.Orders), len(snapshotA.Fills))
	}
	if _, ok := snapshotA.Orders[artifactB.order.OrderID]; ok {
		t.Fatal("account A venue restored account B order")
	}
	if _, ok := snapshotA.Fills[artifactB.fill.FillID]; ok {
		t.Fatal("account A venue restored account B fill")
	}
	if result := papertrading.Reconcile(papertrading.ReconciliationInput{VenueSnapshot: snapshotA, Account: artifactA.account}, now.Add(2*time.Minute)); result.Status != papertrading.ReconciliationClean {
		t.Fatalf("account A reconciliation=%#v", result)
	}

	venueB, err := store.RestorePaperVenueForAccount(ctx, accountB, contract, costModel)
	if err != nil {
		t.Fatal(err)
	}
	snapshotB := venueB.Snapshot()
	if len(snapshotB.Orders) != 1 || len(snapshotB.Fills) != 1 {
		t.Fatalf("account B restore orders=%d fills=%d", len(snapshotB.Orders), len(snapshotB.Fills))
	}
	if _, ok := snapshotB.Orders[artifactA.order.OrderID]; ok {
		t.Fatal("account B venue restored account A order")
	}
	if _, ok := snapshotB.Fills[artifactA.fill.FillID]; ok {
		t.Fatal("account B venue restored account A fill")
	}
	if result := papertrading.Reconcile(papertrading.ReconciliationInput{VenueSnapshot: snapshotB, Account: artifactB.account}, now.Add(2*time.Minute)); result.Status != papertrading.ReconciliationClean {
		t.Fatalf("account B reconciliation=%#v", result)
	}

	unownedOrder := artifactA.order
	unownedOrder.OrderID = "unowned-order-" + suffix
	unownedOrder.PaperIntentID = "unowned-intent-" + suffix
	_, err = pool.Exec(ctx, `
		INSERT INTO paper_orders (order_id,contract_version,venue_id,environment,paper_intent_id,workflow_id,recommendation_id,risk_decision_id,confirmation_id,instrument_id,direction,quantity,remaining_quantity,filled_quantity,order_type,limit_price,cost_model_id,created_at,activates_at,status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
	`, unownedOrder.OrderID, unownedOrder.ContractVersion, unownedOrder.VenueID, string(unownedOrder.Environment), unownedOrder.PaperIntentID, unownedOrder.WorkflowID, unownedOrder.RecommendationID, unownedOrder.RiskDecisionID, unownedOrder.ConfirmationID, unownedOrder.InstrumentID, unownedOrder.Direction, unownedOrder.Quantity, unownedOrder.RemainingQuantity, unownedOrder.FilledQuantity, string(unownedOrder.OrderType), nullableFloat(unownedOrder.LimitPrice), unownedOrder.CostModelID, unownedOrder.CreatedAt, unownedOrder.ActivatesAt, string(unownedOrder.Status))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM paper_orders WHERE order_id=$1`, unownedOrder.OrderID)
	})
	if _, err := store.RestorePaperVenueForAccount(ctx, accountA, contract, costModel); err == nil {
		t.Fatal("unowned paper order was silently ignored during account-scoped restore")
	}
}

func persistScopedPaperArtifact(t *testing.T, ctx context.Context, pool *pgxpool.Pool, accountID, suffix string, now time.Time) scopedPaperArtifact {
	t.Helper()
	approval := postgresApproval(t, suffix, "LONG", now)
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), papertrading.DefaultCostModel())
	if err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: approval.Workflow, PaperIntent: approval.PaperIntent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: papertrading.DefaultCostModel(), InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: papertrading.OrderMarket, CreatedAt: now, IdempotencyKey: suffix})
	if err != nil {
		t.Fatal(err)
	}
	fills, err := venue.ProcessTick(papertrading.MarketTick{TickID: "tick-" + suffix, InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(time.Second), ReceivedAt: now.Add(time.Second), Session: papertrading.SessionOpen, Source: "account-scope-test"})
	if err != nil || len(fills) != 1 {
		t.Fatalf("scoped artifact fills=%#v err=%v", fills, err)
	}
	ledger, err := papertrading.NewPaperLedger(accountID, "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	account, err := ledger.ApplyFill(fills[0])
	if err != nil {
		t.Fatal(err)
	}
	persistedOrder, ok := venue.Snapshot().Orders[order.OrderID]
	if !ok {
		t.Fatalf("scoped artifact order %s was not present after fill", order.OrderID)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistWorkflowSnapshot(ctx, tx, approval); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := persistPaperArtifacts(ctx, tx, persistedOrder, fills[0], account); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return scopedPaperArtifact{account: account, order: persistedOrder, fill: fills[0], approval: approval}
}

func assertOrderReconcilesWithFills(t *testing.T, order papertrading.PaperOrder, fills []papertrading.PaperFill) {
	t.Helper()
	var durableFillQuantity float64
	for _, fill := range fills {
		if fill.OrderID != order.OrderID {
			t.Fatalf("fill %s references order %s, want %s", fill.FillID, fill.OrderID, order.OrderID)
		}
		durableFillQuantity += fill.Quantity
	}
	if math.Abs(order.FilledQuantity-durableFillQuantity) > 1e-9 {
		t.Fatalf("order %s filled quantity=%v, durable fill quantity=%v", order.OrderID, order.FilledQuantity, durableFillQuantity)
	}
	if math.Abs(order.RemainingQuantity+order.FilledQuantity-order.Quantity) > 1e-9 {
		t.Fatalf("order %s quantity identity failed: quantity=%v filled=%v remaining=%v", order.OrderID, order.Quantity, order.FilledQuantity, order.RemainingQuantity)
	}
	switch {
	case order.FilledQuantity == 0 && order.Status != papertrading.OrderNew && order.Status != papertrading.OrderActive:
		t.Fatalf("order %s status=%s is invalid for an unfilled order", order.OrderID, order.Status)
	case order.RemainingQuantity == 0 && order.Status != papertrading.OrderFilled:
		t.Fatalf("order %s status=%s is invalid for a fully filled order", order.OrderID, order.Status)
	case order.FilledQuantity > 0 && order.RemainingQuantity > 0 && order.Status != papertrading.OrderPartiallyFilled:
		t.Fatalf("order %s status=%s is invalid for a partially filled order", order.OrderID, order.Status)
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

func postgresExitRejection(t *testing.T, recommendationID, suffix, direction string, now time.Time, positionID, reviewID, entryWorkflowID, entryPaperIntentID string) ApprovalSnapshot {
	t.Helper()
	risk := acceptedRisk(t, recommendationID, now, 1200)
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
	confirmation, err = confirmation.WithDecision("operator-1", workflow.ConfirmationReject)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = workflowStore.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: workflow.ActorHuman, IdempotencyKey: suffix + "-reject", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	events, err := workflowStore.Events(context.Background(), wf.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	return ApprovalSnapshot{Workflow: wf, Events: events, ExitBinding: &ExitApprovalBinding{PositionID: positionID, ReviewID: reviewID, RecommendationID: recommendationID, EntryWorkflowID: entryWorkflowID, EntryPaperIntentID: entryPaperIntentID}}
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
