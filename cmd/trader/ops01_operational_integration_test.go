package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/internal/modules/assetresolution"
	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/instruments"
	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
	"jax-trading-assistant/internal/testsupport"
	"jax-trading-assistant/libs/auth"
)

func TestOPS01OperationalReadinessProof(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("EXECUTION_ENABLED", "false")
	t.Setenv("BROKER_EXECUTION_ALLOWED", "false")
	t.Setenv("MAX_LEVERAGE", "1")
	dsn := testsupport.PostgresDSN(t, "OPS01_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	now, err := ops01TestNow(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	accountID := "ops01-account-" + suffix
	t.Setenv("PAPER_ACCOUNT_ID", accountID)
	eventID := "ops01-provider-" + suffix
	strategyInstanceID := uuid.New()
	var candidateID string
	var positionID string
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if positionID != "" {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_outcomes WHERE position_id=$1`, positionID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_checkpoints WHERE position_id=$1`, positionID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_reviews WHERE position_id=$1`, positionID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_evidence_reassessments WHERE position_id=$1`, positionID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_lifecycles WHERE position_id=$1`, positionID)
		}
		if candidateID != "" {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM exploratory_paper_entry_queue WHERE candidate_id=$1`, candidateID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_evidence_scores WHERE candidate_id=$1::uuid`, candidateID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_evidence_items WHERE candidate_id=$1::uuid`, candidateID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_events WHERE candidate_id=$1::uuid`, candidateID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id=$1::uuid`, candidateID)
		}
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE instance_id=$1`, strategyInstanceID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candles WHERE symbol='QQQ' AND source='ops01-provider-substitute'`)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM quotes WHERE symbol='QQQ'`)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_instances WHERE id=$1`, strategyInstanceID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_pull_cursors WHERE consumer_name=$1 AND source_endpoint_identity LIKE $2`, worldMonitorPullConsumer, "%/api/v1/jax/events")
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_pull_pages WHERE consumer_name=$1 AND source_endpoint_identity LIKE $2`, worldMonitorPullConsumer, "%/api/v1/jax/events")
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id=$1`, eventID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,config,config_hash)
		VALUES($1,$2,'etf_news_sector_momentum_v1','etf_news_sector_momentum_v1',TRUE,$3::jsonb,$4)
	`, strategyInstanceID, "ops01-"+suffix, `{"symbols":["QQQ"]}`, "ops01-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,exchange) VALUES('QQQ',100,99.5,100.5,1000,1000,100000,$1,'ops01-provider-substitute') ON CONFLICT (symbol) DO UPDATE SET price=EXCLUDED.price,bid=EXCLUDED.bid,ask=EXCLUDED.ask,bid_size=EXCLUDED.bid_size,ask_size=EXCLUDED.ask_size,volume=EXCLUDED.volume,timestamp=EXCLUDED.timestamp,exchange=EXCLUDED.exchange`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM candles WHERE symbol='QQQ' AND source='ops01-provider-substitute'`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		at := now.Add(-time.Duration(25-i) * time.Hour)
		closePrice := 95.0 + float64(i)*0.25
		if _, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification) VALUES('QQQ',$1,$2,$2,$3,$2,$4,$2,'1h','ops01-provider-substitute','provider_observation','substitute')`, at, closePrice, closePrice-0.5, 10000+i); err != nil {
			t.Fatal(err)
		}
	}

	articleURL := "https://ops01.example/provider/" + suffix
	event := worldMonitorPullEvent{
		EventID: eventID, PersistenceSeq: "1", SourceID: "ops01-provider", SourceName: "OPS-01 fixture provider",
		FeedURL: "https://ops01.example/feed", ArticleURL: &articleURL,
		Title:           "Federal Reserve announces policy decision for OPS-01 fixture",
		Summary:         "The Federal Reserve published a policy decision affecting technology markets.",
		PublicationTime: timePointer(now.Add(-time.Minute)), SourceTimestamp: timePointer(now.Add(-time.Minute)),
		CollectedAt: now, FirstSeenAt: now, LastSeenAt: now, ContentHash: suffix, SchemaVersion: 1,
		RawSourcePayload: map[string]any{"possible_affected_etfs": []any{"QQQ"}, "asset_themes": []any{"technology"}, "confidence": 0.8},
		Provenance:       map[string]any{"event_type": "macro_rates"},
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != worldMonitorEventsPath {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("after") == "0" {
			_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{event}, NextCursor: "1", Count: 1})
			return
		}
		_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{}, NextCursor: r.URL.Query().Get("after"), Count: 0})
	}))
	defer provider.Close()
	endpoint := provider.URL + worldMonitorEventsPath
	t.Setenv("WORLD_MONITOR_PULL_ENABLED", "true")
	t.Setenv("WORLD_MONITOR_EVENTS_URL", endpoint)
	rules, err := eventdecisions.LoadRuleset("../../config/genuine-event-decision-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		t.Fatal(err)
	}
	assetRules, err := assetresolution.LoadRuleset("../../config/event-asset-resolution-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	resolver := assetresolution.Resolver{Rules: assetRules}
	worker := &worldMonitorPullWorker{
		pool:       pool,
		config:     worldMonitorPullConfig{Enabled: true, Endpoint: endpoint, PageSize: 10, Timeout: time.Second},
		httpClient: provider.Client(),
		replayer:   eventdecisions.Replayer{Store: eventdecisions.NewStore(pool), Evaluator: eventdecisions.Evaluator{Ruleset: rules, Catalog: catalog}, Resolver: &resolver, Origin: eventdecisions.DecisionOriginLive},
		now:        func() time.Time { return now },
	}
	cycle, err := worker.cycle(ctx)
	if err != nil || cycle.Cursor != 1 || cycle.Ingested != 1 || cycle.DecisionsCreated != 1 {
		t.Fatalf("World Monitor intake cycle=%+v err=%v", cycle, err)
	}
	var rawCount, inboxCount, decisionCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM world_monitor_pull_pages WHERE source_endpoint_identity=$1`, endpoint).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM world_monitor_research_inbox WHERE source_event_id=$1`, eventID).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM genuine_event_decisions d JOIN world_monitor_research_inbox w ON w.id=d.source_inbox_event_id WHERE w.source_event_id=$1`, eventID).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if rawCount != 1 || inboxCount != 1 || decisionCount != 1 {
		t.Fatalf("intake provenance counts raw=%d inbox=%d decisions=%d", rawCount, inboxCount, decisionCount)
	}
	health := worldMonitorPullHealth{}
	health.LastFetched = cycle.Fetched
	health.LastIngested = cycle.Ingested
	health.LastDuplicates = cycle.Duplicates
	health.LastDecisionsCreated = cycle.DecisionsCreated
	health.LastDecisionsReused = cycle.DecisionsReused
	health.success(now, cycle.Cursor)
	if err := persistWorldMonitorPullHealth(ctx, worker, health, cycle, nil); err != nil {
		t.Fatal(err)
	}
	ops01WorkerStarted("world_monitor")
	ops01WorkerRun("world_monitor", now, nil)
	ops01WorkerStarted("entry_worker")
	ops01WorkerStarted("review_worker")
	readinessRequest := httptest.NewRequest(http.MethodGet, "/api/v1/ops-01/readiness", nil)
	readinessRecorder := httptest.NewRecorder()
	ops01ReadinessHandler(pool)(readinessRecorder, readinessRequest)
	if readinessRecorder.Code != http.StatusOK {
		t.Fatalf("readiness status=%d body=%s", readinessRecorder.Code, readinessRecorder.Body.String())
	}
	var readiness map[string]any
	if err := json.Unmarshal(readinessRecorder.Body.Bytes(), &readiness); err != nil {
		t.Fatal(err)
	}
	if readiness["status"] != "READY" || readiness["ready"] != true {
		t.Fatalf("OPS-01 readiness projection is not ready: %#v", readiness)
	}
	worldMonitorReadiness, ok := readiness["worldMonitor"].(map[string]any)
	if !ok || worldMonitorReadiness["lastCommittedCursor"] != float64(cycle.Cursor) || worldMonitorReadiness["status"] != "PROCESSED" {
		t.Fatalf("OPS-01 World Monitor diagnostics are incomplete: %#v", readiness["worldMonitor"])
	}

	promoter := newWorldMonitorOpportunityPromoter(pool)
	promoter.now = func() time.Time { return now }
	rows, err := promoter.loadPromotionRows(ctx, 250)
	if err != nil {
		t.Fatal(err)
	}
	var promotionRow worldMonitorInboxPromotionRow
	for _, row := range rows {
		if row.SourceEventID == eventID {
			promotionRow = row
			break
		}
	}
	if promotionRow.ID == uuid.Nil {
		t.Fatal("fixture inbox row was not eligible for the production promoter")
	}
	promoted, outcomes, err := promoter.promoteRow(ctx, promotionRow)
	if err != nil || promoted == nil || len(outcomes) == 0 {
		t.Fatalf("production opportunity promotion promoted=%#v outcomes=%#v err=%v", promoted, outcomes, err)
	}
	candidateID = promoted.CandidateID
	riskReview := outcomes[len(outcomes)-1].RiskReview
	if riskReview == nil || !riskReview.Result.RiskReady {
		t.Fatalf("candidate/risk path was not ready: %#v", outcomes)
	}
	riskDecision := ops01SyntheticRisk(t, candidateID, now)

	thesis := ops01Thesis(eventID, now)
	approval := ops01ApprovedApproval(t, riskDecision, now, "ops01-entry-operator", workflow.ConfirmationApprove, "LONG")
	calendar := ops01Calendar(now)
	ledger, err := papertrading.NewPaperLedger(accountID, "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	ledgerSnapshot := ledger.Snapshot()
	runtime, err := newExploratoryPaperRuntime(pool)
	if err != nil {
		t.Fatal(err)
	}
	entryAt := now.Add(2 * time.Second)
	entry := exploratorypaper.EntryRequest{
		CandidateID: candidateID,
		Candidate:   exploratorypaper.CandidateInput{EventID: eventID, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, EventCategory: thesis.EventCategory, EventTimestamp: thesis.EventTimestamp, GeneratedAt: thesis.CandidateGeneratedAt, SourceURL: articleURL, Direction: thesis.Direction, CausalMechanism: thesis.ExpectedMechanism, QuantContext: thesis.QuantTechnicalContext, RiskAssessment: thesis.RiskAssessment, TechnicalConfirmation: "persisted candle confirmation", CandidatePolicyVersion: "candidate-evidence-scoring-v1"},
		Evidence:    exploratorypaper.EvidenceAssessment{Provider: "candidate_evidence_scores", PolicyVersion: "candidate-evidence-scoring-v1", ReviewedAt: now, SourceBacked: true, QualityState: "sufficient", QualityScore: .9, RequiredQualityScore: .7, EvidenceReady: true, EvidenceGateReady: true, Corroborated: true, IssuerRelevant: true, InstrumentRelevant: true, IndependentSourceGroups: 2},
		Thesis:      thesis, RiskDecision: riskDecision, Approval: approval, PositionID: "ops01-position-" + suffix, Quantity: 10,
		Tick:  papertrading.MarketTick{TickID: "ops01-entry-tick-" + suffix, InstrumentID: "QQQ", Bid: 99.5, Ask: 100.5, Last: 100, AvailableQuantity: 10, Timestamp: entryAt.Add(2 * time.Second), ReceivedAt: entryAt.Add(2 * time.Second), Session: papertrading.SessionOpen, Source: "ops01-market-substitute"},
		Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: papertrading.DefaultCostModel(), Ledger: ledgerSnapshot, Calendar: calendar, Now: entryAt,
	}
	store := runtime.Store.(*exploratorypaper.PostgresStore)
	if err := store.QueueApprovedEntry(ctx, entry); err != nil {
		t.Fatal(err)
	}
	if err := store.QueueApprovedEntry(ctx, entry); err != nil {
		t.Fatalf("approved entry replay was not idempotent: %v", err)
	}
	queuedEntries, err := store.LoadApprovedEntries(ctx)
	if err != nil || len(queuedEntries) != 1 {
		t.Fatalf("canonical approved entry projection entries=%d err=%v", len(queuedEntries), err)
	}
	projection, err := exploratorypaper.GenerateCandidate(queuedEntries[0].Candidate)
	if err != nil || projection.Decision != exploratorypaper.DecisionCandidate {
		t.Fatalf("canonical candidate projection decision=%#v err=%v candidate=%#v evidence=%#v", projection, err, queuedEntries[0].Candidate, queuedEntries[0].Evidence)
	}
	runtime.Reviews = &ops01FixtureReviewSource{pool: pool, calendar: calendar, observedAt: now.Add(24 * time.Hour), price: 104}
	runtime.Now = func() time.Time { return entryAt }
	if err := runtime.RunEntryCycle(ctx); err != nil {
		t.Fatal(err)
	}
	restartedRuntime, err := newExploratoryPaperRuntime(pool)
	if err != nil {
		t.Fatal(err)
	}
	restartedStore := restartedRuntime.Store.(*exploratorypaper.PostgresStore)
	record, err := restartedStore.Get(ctx, entry.PositionID)
	if err != nil {
		t.Fatal(err)
	}
	positionID = record.Position.PositionID
	if record.CandidateID != candidateID || record.EntryOrder.OrderID == "" || record.EntryFill.FillID == "" || len(record.Reviews) != 5 {
		t.Fatalf("entry lifecycle identity incomplete: %#v", record)
	}
	if _, err := pool.Exec(ctx, `UPDATE exploratory_paper_reviews SET scheduled_at=$2,status='PENDING' WHERE position_id=$1 AND session_number=1`, positionID, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	restartedRuntime.Reviews = &ops01FixtureReviewSource{pool: pool, calendar: calendar, observedAt: now.Add(24 * time.Hour), price: 104}
	restartedRuntime.Now = func() time.Time { return now.Add(2 * time.Hour) }
	reviewRuntime := restartedRuntime
	if err := reviewRuntime.RunDueReviews(ctx); err != nil {
		t.Fatal(err)
	}
	recommended, err := restartedStore.Get(ctx, positionID)
	if err != nil {
		t.Fatal(err)
	}
	review := ops01RecommendedReview(t, recommended)
	if review.Status != "EXIT_RECOMMENDED" || review.ExitRecommendationID == "" {
		t.Fatalf("exit recommendation was not durable: %#v", review)
	}

	exitApproval := ops01ApprovalForRecommendation(t, review.ExitRecommendationID, now.Add(24*time.Hour), "ops01-exit-operator")
	exitApproval.ExitBinding = &exploratorypaper.ExitApprovalBinding{PositionID: positionID, ReviewID: review.ReviewID, RecommendationID: review.ExitRecommendationID, EntryWorkflowID: record.Binding.WorkflowID, EntryPaperIntentID: record.Binding.PaperIntentID}
	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("ops01-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.GenerateToken("ops01-exit-operator", "ops01-exit-operator", "operator")
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(exitApproval)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/exploratory-paper/positions/"+positionID+"/exit-approval", strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-User-ID", "spoofed-operator")
	rec := httptest.NewRecorder()
	manager.MiddlewareFunc(func(w http.ResponseWriter, r *http.Request) {
		handleExploratoryExitDecision(w, r, restartedStore, positionID, "exit-approval")
	})(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("exit approval API status=%d body=%s", rec.Code, rec.Body.String())
	}

	finalRuntime, err := newExploratoryPaperRuntime(pool)
	if err != nil {
		t.Fatal(err)
	}
	finalRuntime.Reviews = &ops01FixtureReviewSource{pool: pool, calendar: calendar, observedAt: now.Add(24 * time.Hour), price: 104}
	finalRuntime.Now = func() time.Time { return now.Add(3 * time.Hour) }
	if err := finalRuntime.RunDueReviews(ctx); err != nil {
		t.Fatal(err)
	}
	finalRecord, err := finalRuntime.Store.(*exploratorypaper.PostgresStore).Get(ctx, positionID)
	if err != nil {
		t.Fatal(err)
	}
	if finalRecord.Outcome == nil || finalRecord.Position.State != exploratorypaper.StateClosed || finalRecord.Outcome.EntryFillID != record.EntryFill.FillID || finalRecord.Outcome.ExitFillID == "" {
		t.Fatalf("final durable outcome incomplete: %#v", finalRecord)
	}
	var exitWorkflowID string
	for _, storedReview := range finalRecord.Reviews {
		if storedReview.SessionNumber == 1 && storedReview.ExitApproval != nil {
			exitWorkflowID = storedReview.ExitApproval.Workflow.WorkflowID
			break
		}
	}
	if exitWorkflowID == "" {
		t.Fatalf("durable exit workflow identity is missing: %#v", finalRecord.Reviews)
	}
	var orderCount, fillCount, ledgerCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM paper_orders WHERE workflow_id=$1 OR workflow_id=$2`, record.Binding.WorkflowID, exitWorkflowID).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM paper_fills WHERE workflow_id=$1 OR workflow_id=$2`, record.Binding.WorkflowID, exitWorkflowID).Scan(&fillCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM paper_ledger_events WHERE workflow_id=$1 OR workflow_id=$2`, record.Binding.WorkflowID, exitWorkflowID).Scan(&ledgerCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != 2 || fillCount != 2 || ledgerCount != 2 {
		t.Fatalf("economic action identities were duplicated or lost: orders=%d fills=%d ledger=%d", orderCount, fillCount, ledgerCount)
	}
}

type ops01FixtureReviewSource struct {
	pool       *pgxpool.Pool
	calendar   exploratorypaper.SessionCalendar
	observedAt time.Time
	price      float64
}

func (s *ops01FixtureReviewSource) LoadReviewObservation(ctx context.Context, record exploratorypaper.LifecycleRecord, review exploratorypaper.Review) (exploratorypaper.ReviewObservation, error) {
	ledger, err := loadPaperLedger(ctx, s.pool, record.EntryFill.FillID)
	if err != nil {
		return exploratorypaper.ReviewObservation{}, err
	}
	observedAt := s.observedAt.UTC()
	evidence := make([]exploratorypaper.RelevantEvidence, 0, len(record.Thesis.Thesis.Evidence))
	for _, item := range record.Thesis.Thesis.Evidence {
		evidence = append(evidence, exploratorypaper.RelevantEvidence{Reference: item, IssuerID: record.Thesis.Thesis.IssuerID, InstrumentID: record.Thesis.Thesis.InstrumentID, Signal: exploratorypaper.EvidenceInvalidates, Reason: "OPS-01 fixture review invalidation"})
	}
	tick := papertrading.MarketTick{TickID: "ops01-review-" + record.Position.PositionID + "-" + string(rune('0'+review.SessionNumber)), InstrumentID: record.Thesis.Thesis.InstrumentID, Bid: s.price - 0.5, Ask: s.price + 0.5, Last: s.price, AvailableQuantity: record.EntryFill.Quantity, Timestamp: observedAt, ReceivedAt: observedAt, Session: papertrading.SessionOpen, Source: "ops01-market-substitute"}
	return exploratorypaper.ReviewObservation{Tick: tick, Price: s.price, PriceSource: "ops01-market-substitute", Evidence: evidence, Calendar: s.calendar, Ledger: ledger, QuoteMode: "MODELED_CANDLE_CLOSE", LiquidityMode: "MODELED_POSITION_CAPACITY", ActualQuoteAvailable: false, ObservedAt: observedAt, ReceivedAt: observedAt, ThesisInvalidated: true, Path: []exploratorypaper.PriceObservation{{ObservationID: tick.TickID, At: observedAt, Price: s.price, Source: "ops01-market-substitute"}}, Excursion: exploratorypaper.ExcursionCoverage{Status: "INCOMPLETE", WindowStart: record.Position.EntryAt, WindowEnd: observedAt, ExpectedObservationCount: review.SessionNumber, Cadence: "ops01-fixture", SourceProvenance: true}}, nil
}

func ops01TestNow(ctx context.Context, pool *pgxpool.Pool) (time.Time, error) {
	fallback := time.Date(2026, 9, 23, 14, 0, 0, 0, time.UTC)
	var latest time.Time
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(filled_at), $1) FROM paper_fills`, fallback).Scan(&latest); err != nil {
		return time.Time{}, err
	}
	if latest.Before(fallback) {
		latest = fallback
	}
	return latest.UTC().Add(time.Hour), nil
}

func ops01Thesis(eventID string, now time.Time) exploratorypaper.TradeThesis {
	return exploratorypaper.TradeThesis{ThesisID: "ops01-thesis-" + strings.ReplaceAll(eventID, "ops01-provider-", ""), ContractVersion: exploratorypaper.ContractVersion, Mode: exploratorypaper.ExploratoryPaperMode, EventID: eventID, IssuerID: "ops01-federal-reserve", InstrumentID: "QQQ", Direction: exploratorypaper.DirectionLong, Exposure: "technology ETF", EventCategory: "macro_rates", EventTimestamp: now, CandidateGeneratedAt: now, CausalReason: "fixture policy decision changes technology-rate sensitivity", Evidence: []exploratorypaper.EvidenceReference{{EvidenceID: "ops01-evidence-provider-" + eventID, SourceID: "ops01-provider", SourceURL: "https://ops01.example/provider/" + eventID, Quality: "high", ObservedAt: now}, {EvidenceID: "ops01-evidence-chart-" + eventID, SourceID: "ops01-market", SourceURL: "https://ops01.example/market/" + eventID, Quality: "high", ObservedAt: now}}, ExpectedMechanism: "rate decision changes technology duration valuation", ExpectedHorizonSessions: 3, EntryRationale: "explicit OPS-01 readiness fixture", QuantTechnicalContext: "provider event plus persisted rising candle confirmation", RiskAssessment: "paper-only bounded risk", EntryPolicyVersion: "paper-02-entry-v1", ProtectiveStop: 98, Target: 104, InvalidationConditions: []string{"fixture review invalidates thesis"}, Confidence: .8, Uncertainty: "fixture market transport", PolicyVersion: "paper-02-policy-v1", CreatedAt: now}
}

func ops01Calendar(now time.Time) exploratorypaper.SessionCalendar {
	sessions := map[string]bool{}
	for i := -1; i < 8; i++ {
		sessions[now.AddDate(0, 0, i).Format("2006-01-02")] = true
	}
	return exploratorypaper.SessionCalendar{Sessions: sessions, Timezone: "UTC", OpenTime: "00:00", CloseTime: "23:59"}
}

func ops01ApprovedApproval(t *testing.T, risk portfoliorisk.RiskDecision, at time.Time, actor string, decision workflow.ConfirmationDecision, direction string) exploratorypaper.ApprovalSnapshot {
	t.Helper()
	nonce := uuid.NewString()
	store := workflow.NewStore()
	wf, err := store.Create(context.Background(), workflow.CreateRequest{RiskDecision: risk, Now: at, IdempotencyKey: "ops01-create-" + actor + "-" + nonce})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.RequestHumanConfirmation(context.Background(), workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "ops01-await-" + actor + "-" + nonce, Now: at})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := workflow.NewConfirmation(wf, "QQQ", direction, at, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision(actor, decision)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: actor, ActorRole: workflow.ActorHuman, IdempotencyKey: "ops01-confirm-" + actor + "-" + nonce, Now: at})
	if err != nil {
		t.Fatal(err)
	}
	if decision == workflow.ConfirmationApprove {
		wf, intent, err := store.CreatePaperIntent(context.Background(), workflow.PaperIntentRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "ops01-intent-" + actor + "-" + nonce, Now: at})
		if err != nil {
			t.Fatal(err)
		}
		events, err := store.Events(context.Background(), wf.WorkflowID)
		if err != nil {
			t.Fatal(err)
		}
		return exploratorypaper.ApprovalSnapshot{Workflow: wf, Events: events, PaperIntent: intent}
	}
	events, err := store.Events(context.Background(), wf.WorkflowID)
	if err != nil {
		t.Fatal(err)
	}
	return exploratorypaper.ApprovalSnapshot{Workflow: wf, Events: events}
}

func ops01ApprovalForRecommendation(t *testing.T, recommendationID string, at time.Time, actor string) exploratorypaper.ApprovalSnapshot {
	risk := ops01SyntheticRisk(t, recommendationID, at)
	return ops01ApprovedApproval(t, risk, at, actor, workflow.ConfirmationApprove, "SHORT")
}

func ops01SyntheticRisk(t *testing.T, recommendationID string, at time.Time) portfoliorisk.RiskDecision {
	t.Helper()
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: "ops01-account", AsOf: at, CapturedAt: at, Provider: "ops01-fixture", Currency: "USD", Synthetic: true,
		Cash: portfoliorisk.KnownNumber(10000, "fixture"), Equity: portfoliorisk.KnownNumber(10000, "fixture"),
		Positions:      []portfoliorisk.Position{{InstrumentID: "MSFT", InstrumentResolved: true, Currency: "USD", SignedQuantity: 1, Price: portfoliorisk.KnownNumber(100, "fixture"), MarketValue: portfoliorisk.KnownNumber(100, "fixture"), CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: at, PriceSource: "fixture", Provenance: []string{"fixture"}}},
		ValuationBasis: "frozen", Provenance: []string{"ops01-fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, at, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := portfoliorisk.BuildRiskPolicy(portfoliorisk.RiskPolicy{
		Version: "ops01-policy", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(1), MaximumPositionValue: portfoliorisk.Limit(2000), MaximumConcentration: portfoliorisk.Limit(.5), MaximumGrossExposure: portfoliorisk.Limit(5000), MaximumNetExposure: portfoliorisk.Limit(5000),
	})
	if err != nil {
		t.Fatal(err)
	}
	riskAllocation, leverage := .01, 1.0
	decision := portfoliorisk.EvaluateRecommendation(portfoliorisk.RecommendationRiskInput{
		RecommendationID: recommendationID, InstrumentID: "QQQ", Currency: "USD", SignedMarketValue: 1200,
		RiskAllocation: &riskAllocation, RequestedLeverage: &leverage, ExecutionAuthority: "NONE", QuantResultIDs: []string{"ops01-fixture"},
	}, snapshot, analytics, policy, at, time.Hour)
	if decision.Outcome != portfoliorisk.DecisionAccept {
		t.Fatalf("risk decision=%#v", decision)
	}
	return decision
}

func ops01RecommendedReview(t *testing.T, record exploratorypaper.LifecycleRecord) exploratorypaper.Review {
	t.Helper()
	for _, review := range record.Reviews {
		if review.Status == "EXIT_RECOMMENDED" {
			return review
		}
	}
	t.Fatalf("no EXIT_RECOMMENDED review in %#v", record.Reviews)
	return exploratorypaper.Review{}
}
