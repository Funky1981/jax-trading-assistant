package exploratorypaper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This exercises the real exploratory Runtime and PostgreSQL order/ledger
// persistence. It is restricted by testsupport to a disposable test database.
func TestPostgresRuntimePersistsPendingEntryAcrossRestartAndFillsOncePostLatency(t *testing.T) {
	databaseURL := testsupport.PostgresDSN(t, "PAPER01B_DATABASE_URL")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC().Truncate(time.Microsecond).Add(-2 * time.Second)
	approvalAt := base.Add(100 * time.Millisecond)
	activationAt := approvalAt.Add(time.Second)
	executionAt := base.Add(1200 * time.Millisecond)
	suffix := fmt.Sprintf("pending-%d", base.UnixNano())
	accountID := "account-" + suffix
	thesis := fixtureThesis()
	thesis.ThesisID, thesis.EventID = "thesis-"+suffix, "event-"+suffix
	thesis.CreatedAt, thesis.CandidateGeneratedAt, thesis.EventTimestamp = base, base, base.Add(-time.Hour)
	for i := range thesis.Evidence {
		thesis.Evidence[i].ObservedAt = base
	}
	for i := range thesis.CounterEvidence {
		thesis.CounterEvidence[i].ObservedAt = base
	}
	input := reviewedCandidateInput(base)
	input.EventID, input.IssuerID, input.InstrumentID = thesis.EventID, thesis.IssuerID, thesis.InstrumentID
	input.EventCategory, input.EventTimestamp, input.GeneratedAt = thesis.EventCategory, thesis.EventTimestamp, thesis.CandidateGeneratedAt
	input.Evidence = thesis.Evidence
	input.ReviewedEvidence.EvidenceSetFingerprint = EvidenceSetFingerprint(input.Evidence)
	input.ReviewedEvidence.ReviewedAt = base
	input.SourceURL = thesis.Evidence[0].SourceURL
	input.CausalMechanism, input.QuantContext, input.RiskAssessment = thesis.ExpectedMechanism, thesis.QuantTechnicalContext, thesis.RiskAssessment
	input.TechnicalConfirmation, input.Direction = "confirmed supporting trend", thesis.Direction
	approval := postgresApproval(t, suffix, "LONG", approvalAt)
	workflowSnapshot, intent := approval.Workflow, approval.PaperIntent
	risk := acceptedRisk(t, workflowSnapshot.RecommendationID, approvalAt, workflowSnapshot.ResultingValue)
	ledger, err := papertrading.NewPaperLedger(accountID, "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	openSessions := map[string]bool{}
	for day := 0; day <= 10; day++ {
		openSessions[base.AddDate(0, 0, day).Format("2006-01-02")] = true
	}
	calendar := SessionCalendar{Sessions: openSessions, Timezone: "UTC", OpenTime: "00:00", CloseTime: "23:59"}
	originalTick := papertrading.MarketTick{TickID: "approval-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 99.9, Ask: 100.1, Last: 100, AvailableQuantity: 10, Timestamp: base, LastProviderAt: base, ReceivedAt: base, AsOf: base, RequireTemporalProvenance: true, Session: papertrading.SessionOpen, Source: "alpaca"}
	entry := EntryRequest{CandidateID: "candidate-" + suffix, Candidate: input, Evidence: *input.ReviewedEvidence, Thesis: thesis, RiskDecision: risk, Approval: approval, PositionID: "position-" + suffix, Quantity: 10, Tick: originalTick, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: papertrading.DefaultCostModel(), Ledger: ledger.Snapshot(), Calendar: calendar, Now: approvalAt}
	store := NewPostgresStoreForAccount(pool, accountID)
	venue, err := papertrading.NewPaperVenue(entry.Venue, entry.CostModel)
	if err != nil {
		t.Fatal(err)
	}
	current := approvalAt
	runtime := &Runtime{Mode: "PAPER", AccountID: accountID, Store: store, Entries: runtimeEntrySource{entries: []EntryRequest{entry}}, Reviews: runtimeReviewSource{}, Execution: &runtimeExecutionSource{}, Venue: venue, CostModel: entry.CostModel, Now: func() time.Time { return current }}
	if err := runtime.RunEntryCycle(ctx); err != nil {
		t.Fatalf("initial pending pass: %v", err)
	}
	if len(venue.Snapshot().Orders) != 1 || len(venue.Snapshot().Fills) != 0 {
		t.Fatalf("initial pass orders=%d fills=%d", len(venue.Snapshot().Orders), len(venue.Snapshot().Fills))
	}
	order, ok := venue.OrderForIntent(intent.IntentID)
	if !ok || !order.CreatedAt.Equal(approvalAt) || !order.ActivatesAt.Equal(activationAt) || order.CreatedAt.Before(workflowSnapshot.Confirmation.ConfirmedAt) {
		t.Fatalf("pending order time/identity: %#v", order)
	}
	var persistedOrderCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_orders WHERE paper_intent_id=$1`, intent.IntentID).Scan(&persistedOrderCount); err != nil || persistedOrderCount != 1 {
		t.Fatalf("durable pending order count=%d err=%v", persistedOrderCount, err)
	}
	if originalTick.Timestamp != base || originalTick.LastProviderAt != base || originalTick.ReceivedAt != base || originalTick.AsOf != base {
		t.Fatal("approval observation provenance changed")
	}

	// Simulate process death before activation: reconstruct the runtime venue
	// entirely from the durable pending order, with the same immutable queue item.
	venue, err = store.RestorePaperVenueForAccount(ctx, accountID, entry.Venue, entry.CostModel)
	if err != nil {
		t.Fatalf("restore pending order: %v", err)
	}
	if len(venue.Snapshot().Orders) != 1 || len(venue.Snapshot().Fills) != 0 {
		t.Fatal("restart did not restore one pending order and zero fills")
	}
	current = activationAt
	runtime = &Runtime{Mode: "PAPER", AccountID: accountID, Store: store, Entries: runtimeEntrySource{entries: []EntryRequest{entry}}, Reviews: runtimeReviewSource{}, Execution: &runtimeExecutionSource{ticks: []papertrading.MarketTick{originalTick}}, Venue: venue, CostModel: entry.CostModel, Now: func() time.Time { return current }}
	if err := runtime.RunEntryCycle(ctx); err != nil {
		t.Fatalf("approval observation after activation: %v", err)
	}
	if len(venue.Snapshot().Fills) != 0 {
		t.Fatal("approval-time observation filled the post-latency order")
	}
	executionTick := papertrading.MarketTick{TickID: "execution-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 100.9, Ask: 101.1, Last: 101, AvailableQuantity: 10, Timestamp: executionAt, LastProviderAt: executionAt, ReceivedAt: executionAt, AsOf: executionAt, RequireTemporalProvenance: true, Session: papertrading.SessionOpen, Source: "alpaca"}
	current = executionAt
	runtime = &Runtime{Mode: "PAPER", AccountID: accountID, Store: store, Entries: runtimeEntrySource{entries: []EntryRequest{entry}}, Reviews: runtimeReviewSource{}, Execution: &runtimeExecutionSource{ticks: []papertrading.MarketTick{executionTick}}, Venue: venue, CostModel: entry.CostModel, Now: func() time.Time { return current }}
	if err := runtime.RunEntryCycle(ctx); err != nil {
		t.Fatalf("post-activation pass: %v", err)
	}
	record, found, err := store.FindByCandidate(ctx, entry.CandidateID)
	if err != nil || !found {
		t.Fatalf("filled lifecycle found=%t err=%v", found, err)
	}
	if record.EntryOrder.OrderID != order.OrderID || record.EntryFill.OrderID != order.OrderID || record.EntryFill.FilledAt.Before(activationAt) || record.EntryFill.FilledAt.Before(executionAt) {
		t.Fatalf("entry fill causality/order mismatch: order=%#v fill=%#v", record.EntryOrder, record.EntryFill)
	}
	provenance := record.EntryFill.MarketProvenance
	if provenance == nil || provenance.Provider != "alpaca" || !provenance.StrictTemporal || provenance.QuoteProviderAt != executionAt || provenance.TradeProviderAt != executionAt || provenance.ReceivedAt != executionAt || provenance.AsOf != executionAt || provenance.AvailableAt != executionAt {
		t.Fatalf("execution provenance did not round-trip independently: %#v", provenance)
	}
	if originalTick.Timestamp != base || originalTick.LastProviderAt != base || originalTick.ReceivedAt != base || originalTick.AsOf != base {
		t.Fatal("approved queue evidence was rewritten during execution")
	}
	if err := runtime.RunEntryCycle(ctx); err != nil {
		t.Fatalf("entry replay: %v", err)
	}
	var orders, fills, ledgerEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_orders WHERE paper_intent_id=$1`, intent.IntentID).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_fills WHERE order_id=$1`, order.OrderID).Scan(&fills); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_ledger_events WHERE order_id=$1`, order.OrderID).Scan(&ledgerEvents); err != nil {
		t.Fatal(err)
	}
	if orders != 1 || fills != 1 || ledgerEvents != 1 {
		t.Fatalf("replay duplicated durable artifacts: orders=%d fills=%d ledger=%d", orders, fills, ledgerEvents)
	}
	restored, err := store.RestorePaperVenueForAccount(ctx, accountID, entry.Venue, entry.CostModel)
	if err != nil {
		t.Fatalf("restore completed order: %v", err)
	}
	if got := restored.Snapshot().Fills[record.EntryFill.FillID].MarketProvenance; got == nil || *got != *provenance {
		t.Fatalf("restored venue fill provenance=%#v want=%#v", got, provenance)
	}

	// The same worker path must keep an approved exit pending until a distinct
	// quote is available after its own latency, then persist outcome atomically.
	if err != nil || len(record.Reviews) == 0 {
		t.Fatalf("load scheduled review record=%#v err=%v", record, err)
	}
	var review Review
	for _, candidate := range record.Reviews {
		if candidate.ScheduledAt.After(record.Position.EntryAt) {
			review = candidate
			break
		}
	}
	if review.ReviewID == "" {
		t.Fatal("no scheduled review occurs after entry fill")
	}
	exitApprovalAt := review.ScheduledAt.Add(100 * time.Millisecond)
	exitDecision := ExitDecision{Action: ExitNow, Reason: ExitThesisInvalidated}
	store.now = func() time.Time { return exitApprovalAt }
	if err := store.PersistExitRecommendation(ctx, record.Position.PositionID, exitDecision, review); err != nil {
		t.Fatal(err)
	}
	record, err = store.Get(ctx, entry.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range record.Reviews {
		if candidate.ReviewID == review.ReviewID {
			review = candidate
			break
		}
	}
	exitApproval := postgresExitApproval(t, review.ExitRecommendationID, suffix+"-exit", "SHORT", exitApprovalAt, record.Position.PositionID, review.ReviewID, record.Binding.WorkflowID, record.Binding.PaperIntentID)
	if err := store.PersistExitApproval(ctx, record.Position.PositionID, review, exitApproval); err != nil {
		t.Fatal(err)
	}
	record, err = store.Get(ctx, entry.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range record.Reviews {
		if candidate.ReviewID == review.ReviewID {
			review = candidate
			break
		}
	}
	ledgerAfterEntry, err := papertrading.RestorePaperLedger(entry.Ledger)
	if err != nil {
		t.Fatal(err)
	}
	accountAfterEntry, err := ledgerAfterEntry.ApplyFill(record.EntryFill)
	if err != nil {
		t.Fatal(err)
	}
	reviewTick := papertrading.MarketTick{TickID: "review-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 98.9, Ask: 99.1, Last: 99, AvailableQuantity: 10, Timestamp: exitApprovalAt, ReceivedAt: exitApprovalAt, Session: papertrading.SessionOpen, Source: "review-candle"}
	reviewEvidence := RelevantEvidence{Reference: EvidenceReference{EvidenceID: "exit-evidence-" + suffix, SourceID: "source-" + suffix, SourceURL: "https://example.test/exit", Quality: "high", ObservedAt: exitApprovalAt}, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, Signal: EvidenceInvalidates, Reason: "integration exit invalidation"}
	reviewObservation := ReviewObservation{Tick: reviewTick, Price: 99, PriceSource: "review-candle", Evidence: []RelevantEvidence{reviewEvidence}, Calendar: calendar, Ledger: accountAfterEntry, Path: []PriceObservation{{ObservationID: reviewTick.TickID, At: exitApprovalAt, Price: 99, Source: "review-candle"}}, QuoteMode: "MODELED_CANDLE_CLOSE", LiquidityMode: "MODELED_POSITION_CAPACITY", ObservedAt: exitApprovalAt, ReceivedAt: exitApprovalAt, MarketTimeframe: "1m", MarketAsOf: exitApprovalAt, MarketProvenance: "persisted-test-candle", ThesisInvalidated: true}
	exitActivationAt := exitApprovalAt.Add(time.Second)
	exitExecutionAt := exitApprovalAt.Add(1200 * time.Millisecond)
	exitTick := papertrading.MarketTick{TickID: "exit-execution-" + suffix, InstrumentID: thesis.InstrumentID, Bid: 98.9, Ask: 99.1, Last: 99, AvailableQuantity: 10, Timestamp: exitExecutionAt, LastProviderAt: exitExecutionAt, ReceivedAt: exitExecutionAt, AsOf: exitExecutionAt, RequireTemporalProvenance: true, Session: papertrading.SessionOpen, Source: "alpaca"}
	current = exitApprovalAt
	runtime = &Runtime{Mode: "PAPER", AccountID: accountID, Store: store, Entries: runtimeEntrySource{}, Reviews: availableReviewSource{observation: reviewObservation}, Execution: &runtimeExecutionSource{}, ExitApprover: store, Venue: restored, CostModel: entry.CostModel, Now: func() time.Time { return current }}
	if err := runtime.RunDueReviews(ctx); err != nil {
		t.Fatalf("approved exit pending pass: %v", err)
	}
	exitOrder, ok := restored.OrderForIntent(exitApproval.PaperIntent.IntentID)
	if !ok || !exitOrder.ActivatesAt.Equal(exitActivationAt) || len(restored.Snapshot().Fills) != 1 {
		t.Fatalf("exit did not remain durable/pending: order=%#v", exitOrder)
	}
	venue, err = store.RestorePaperVenueForAccount(ctx, accountID, entry.Venue, entry.CostModel)
	if err != nil {
		t.Fatalf("restore pending exit after restart: %v", err)
	}
	current = exitExecutionAt
	runtime = &Runtime{Mode: "PAPER", AccountID: accountID, Store: store, Entries: runtimeEntrySource{}, Reviews: availableReviewSource{observation: reviewObservation}, Execution: &runtimeExecutionSource{ticks: []papertrading.MarketTick{exitTick}}, ExitApprover: store, Venue: venue, CostModel: entry.CostModel, Now: func() time.Time { return current }}
	if err := runtime.RunDueReviews(ctx); err != nil {
		t.Fatalf("post-latency exit pass: %v", err)
	}
	closed, err := store.Get(ctx, entry.PositionID)
	if err != nil || closed.Outcome == nil || closed.Position.State != StateClosed || closed.Outcome.ExitOrderID != exitOrder.OrderID || closed.Outcome.ExitAt.Before(exitActivationAt) {
		t.Fatalf("durable exit outcome=%#v position=%#v err=%v", closed.Outcome, closed.Position, err)
	}
	if err := runtime.RunDueReviews(ctx); err != nil {
		t.Fatalf("exit replay after close: %v", err)
	}
	var exitOrders, exitFills, exitLedger int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_orders WHERE paper_intent_id=$1`, exitApproval.PaperIntent.IntentID).Scan(&exitOrders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_fills WHERE order_id=$1`, exitOrder.OrderID).Scan(&exitFills); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM paper_ledger_events WHERE order_id=$1`, exitOrder.OrderID).Scan(&exitLedger); err != nil {
		t.Fatal(err)
	}
	if exitOrders != 1 || exitFills != 1 || exitLedger != 1 {
		t.Fatalf("exit replay duplicated durable artifacts: orders=%d fills=%d ledger=%d", exitOrders, exitFills, exitLedger)
	}
}
