package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/libs/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCanonicalPaperHandoffPostgresAcceptRestartConcurrencyAndQueueOnly(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Microsecond)
	fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "accept", .5)
	service := newCanonicalTestService(pool, fixture.accountID, now, nil)

	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("canonical-handoff-integration-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.GenerateToken("operator-02a4", "operator-02a4", "operator")
	if err != nil {
		t.Fatal(err)
	}
	prepareMux := http.NewServeMux()
	registerCanonicalPaperHandoffServiceRoutes(prepareMux, manager.MiddlewareFunc, service)
	path := "/api/v1/exploratory-paper/handoff/" + fixture.candidateID.String()
	prepared := canonicalHandoffHTTP(t, prepareMux, http.MethodPost, path+"/prepare", token, "", "")
	if prepared.Code != http.StatusOK {
		t.Fatalf("PREPARE status=%d body=%s", prepared.Code, prepared.Body.String())
	}
	var first canonicalPrepareResult
	if err := json.Unmarshal(prepared.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if first.Outcome != portfoliorisk.DecisionAccept || first.WorkflowState != "AWAITING_HUMAN_CONFIRMATION" || first.WorkflowID == "" || first.PortfolioSnapshotID == "" || first.RiskDecisionID == "" || first.SlippageAllowance != .5 {
		t.Fatalf("PREPARE did not produce accepted, provenance-bound pending review: %+v", first)
	}
	var snapshotPayload []byte
	if err := pool.QueryRow(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, first.PortfolioSnapshotID).Scan(&snapshotPayload); err != nil {
		t.Fatal(err)
	}
	var snapshot portfoliorisk.PortfolioSnapshot
	if err := json.Unmarshal(snapshotPayload, &snapshot); err != nil {
		t.Fatal(err)
	}
	if !snapshot.KnownEmpty || snapshot.Synthetic || len(snapshot.Positions) != 0 || snapshot.Cash.Value != 100_000 || snapshot.Equity.Value != 100_000 {
		t.Fatalf("canonical empty PAPER account was not explicit and provenance-bearing: %+v", snapshot)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, snapshot.AsOf, 48*time.Hour)
	if err != nil || analytics.GrossExposure != 0 || analytics.NetExposure != 0 || analytics.LongExposure != 0 || analytics.ShortExposure != 0 || analytics.CashAllocation != 1 {
		t.Fatalf("known-empty portfolio exposure=%+v err=%v", analytics, err)
	}
	if lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1`, fixture.candidateID) != 0 || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()) != 0 {
		t.Fatal("PREPARE created a human decision or entry queue")
	}
	replay := canonicalHandoffHTTP(t, prepareMux, http.MethodPost, path+"/prepare", token, "", "")
	var replayed canonicalPrepareResult
	if replay.Code != http.StatusOK || json.Unmarshal(replay.Body.Bytes(), &replayed) != nil || replayed.PortfolioSnapshotID != first.PortfolioSnapshotID || replayed.RiskDecisionID != first.RiskDecisionID || replayed.WorkflowID != first.WorkflowID {
		t.Fatalf("identical PREPARE replay changed canonical identities: status=%d result=%+v body=%s", replay.Code, replayed, replay.Body.String())
	}

	// Rebuild the handler/service to prove no required workflow state is held in memory.
	restarted := newCanonicalTestService(pool, fixture.accountID, now, nil)
	approveMux := http.NewServeMux()
	registerCanonicalPaperHandoffServiceRoutes(approveMux, manager.MiddlewareFunc, restarted)
	beforeOrders := lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM paper_orders WHERE workflow_id=$1`, first.WorkflowID)
	beforeFills := lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM paper_fills WHERE workflow_id=$1`, first.WorkflowID)
	beforeLifecycle := lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_lifecycles WHERE candidate_id=$1`, fixture.candidateID.String())
	results := make([]*httptest.ResponseRecorder, 2)
	var group sync.WaitGroup
	for index := range results {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			spoof := ""
			if index == 0 {
				spoof = "victim-user"
			}
			results[index] = canonicalHandoffHTTP(t, approveMux, http.MethodPost, path+"/approve", token, spoof, "")
		}(index)
	}
	group.Wait()
	for _, result := range results {
		if result.Code != http.StatusOK {
			t.Fatalf("concurrent authenticated APPROVE status=%d body=%s", result.Code, result.Body.String())
		}
	}
	var approvals, workflows, queues, tickets, instructions, ledgerEvents int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1 AND decision='approved'`, fixture.candidateID).Scan(&approvals); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM workflow_instances WHERE recommendation_id=$1`, fixture.candidateID.String()).Scan(&workflows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()).Scan(&queues); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1`, fixture.candidateID).Scan(&tickets); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1`, fixture.candidateID).Scan(&instructions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM paper_ledger_events WHERE account_id=$1`, fixture.accountID).Scan(&ledgerEvents); err != nil {
		t.Fatal(err)
	}
	if approvals != 1 || workflows != 1 || queues != 1 || tickets != 0 || instructions != 0 || ledgerEvents != 0 {
		t.Fatalf("queue-only boundary counts approvals=%d workflows=%d queues=%d tickets=%d instructions=%d ledger=%d", approvals, workflows, queues, tickets, instructions, ledgerEvents)
	}
	if lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM paper_orders WHERE workflow_id=$1`, first.WorkflowID) != beforeOrders || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM paper_fills WHERE workflow_id=$1`, first.WorkflowID) != beforeFills || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_lifecycles WHERE candidate_id=$1`, fixture.candidateID.String()) != beforeLifecycle {
		t.Fatal("APPROVE crossed the queue boundary and created orders, fills, or a lifecycle")
	}
	var actor, approvalDecision string
	if err := pool.QueryRow(ctx, `SELECT approved_by, decision FROM candidate_approvals WHERE candidate_id=$1 AND decision='approved'`, fixture.candidateID).Scan(&actor, &approvalDecision); err != nil {
		t.Fatal(err)
	}
	if actor != "operator-02a4" || approvalDecision != "approved" {
		t.Fatalf("caller-controlled X-User-ID overrode validated JWT actor: actor=%q decision=%q", actor, approvalDecision)
	}
	var queuePayload []byte
	if err := pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()).Scan(&queuePayload); err != nil {
		t.Fatal(err)
	}
	var queued exploratorypaper.EntryRequest
	if err := json.Unmarshal(queuePayload, &queued); err != nil {
		t.Fatal(err)
	}
	if queued.Approval.Workflow.State != "PAPER_INTENT_CREATED" || queued.Approval.Workflow.Confirmation == nil || queued.Approval.Workflow.Confirmation.Actor != "operator-02a4" || queued.RiskDecision.DecisionID != first.RiskDecisionID || queued.Approval.PaperIntent.IntentID == "" || queued.Quantity*queued.Tick.Last > queued.RiskDecision.ResultingValue+1e-6 {
		t.Fatalf("durable queue is not bound to authenticated approved risk: %+v", queued)
	}
	identicalRetry := canonicalHandoffHTTP(t, approveMux, http.MethodPost, path+"/approve", token, "attacker", "")
	if identicalRetry.Code != http.StatusOK {
		t.Fatalf("durable APPROVE replay failed: status=%d body=%s", identicalRetry.Code, identicalRetry.Body.String())
	}
}

func TestCanonicalPaperHandoffPostgresAmendRejectAndAccountIsolation(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Microsecond)
	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("canonical-handoff-branch-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	operator, err := manager.GenerateToken("operator-branch", "operator-branch", "operator")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("AMEND uses reduced durable risk value", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "amend", .5)
		policy := canonicalTestRiskPolicy(10_000)
		service := newCanonicalTestService(pool, fixture.accountID, now, &policy)
		prepared, prepErr := service.prepare(ctx, fixture.candidateID)
		if prepErr != nil {
			t.Fatal(prepErr)
		}
		if prepared.Outcome != portfoliorisk.DecisionAmend || prepared.ResultingNotional > 10_000.000001 || prepared.WorkflowID == "" {
			t.Fatalf("expected persisted AMEND under position cap, got %+v", prepared)
		}
		approved, err := service.approve(ctx, fixture.candidateID, "operator-branch")
		if err != nil {
			t.Fatal(err)
		}
		var payload []byte
		if err := pool.QueryRow(ctx, `SELECT payload FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()).Scan(&payload); err != nil {
			t.Fatal(err)
		}
		var entry exploratorypaper.EntryRequest
		if err := json.Unmarshal(payload, &entry); err != nil {
			t.Fatal(err)
		}
		if approved.RiskDecisionID != prepared.RiskDecisionID || mathAbs(entry.RiskDecision.ResultingValue) > 10_000.000001 || entry.Quantity*entry.Tick.Last > mathAbs(entry.RiskDecision.ResultingValue)+1e-6 {
			t.Fatalf("approved/queued quantity did not use AMENDED risk value: prepared=%+v entry=%+v", prepared, entry)
		}
	})

	t.Run("risk REJECT persists no workflow or approval", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "risk-reject", .5)
		policy := canonicalTestRiskPolicy(0)
		service := newCanonicalTestService(pool, fixture.accountID, now, &policy)
		prepared, prepErr := service.prepare(ctx, fixture.candidateID)
		if prepErr != nil {
			t.Fatal(prepErr)
		}
		if prepared.Outcome != portfoliorisk.DecisionReject || prepared.WorkflowID != "" || prepared.RiskDecisionID == "" {
			t.Fatalf("risk rejection did not persist without workflow: %+v", prepared)
		}
		if lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM workflow_instances WHERE recommendation_id=$1`, fixture.candidateID.String()) != 0 || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1`, fixture.candidateID) != 0 || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()) != 0 {
			t.Fatal("risk REJECT created downstream human approval or queue state")
		}
	})

	t.Run("human REJECT leaves no intent or queue", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "human-reject", .5)
		service := newCanonicalTestService(pool, fixture.accountID, now, nil)
		if got, err := service.prepare(ctx, fixture.candidateID); err != nil || got.Outcome != portfoliorisk.DecisionAccept {
			t.Fatalf("prepare before human rejection: result=%+v err=%v", got, err)
		}
		mux := http.NewServeMux()
		registerCanonicalPaperHandoffServiceRoutes(mux, manager.MiddlewareFunc, service)
		result := canonicalHandoffHTTP(t, mux, http.MethodPost, "/api/v1/exploratory-paper/handoff/"+fixture.candidateID.String()+"/reject", operator, "spoofed-actor", "")
		if result.Code != http.StatusOK || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1 AND decision='rejected' AND approved_by='operator-branch'`, fixture.candidateID) != 1 || lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()) != 0 {
			t.Fatalf("human rejection failed or actor was spoofed: status=%d body=%s", result.Code, result.Body.String())
		}
		var state string
		if err := pool.QueryRow(ctx, `SELECT state FROM workflow_instances WHERE recommendation_id=$1`, fixture.candidateID.String()).Scan(&state); err != nil || state != "HUMAN_REJECTED" {
			t.Fatalf("rejected workflow state=%q err=%v", state, err)
		}
	})

	t.Run("cross-account approval fails closed", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "account-isolation", .5)
		service := newCanonicalTestService(pool, fixture.accountID, now, nil)
		if _, err := service.prepare(ctx, fixture.candidateID); err != nil {
			t.Fatal(err)
		}
		otherAccount := fixture.accountID + "-other"
		if _, err := pool.Exec(ctx, `INSERT INTO paper_accounts(account_id,contract_version,environment,currency,initial_cash,cash,equity,realized_pnl,fees,updated_at) VALUES($1,$2,'PAPER','USD',100000,100000,100000,0,0,$3)`, otherAccount, papertrading.LedgerContractVersion, now); err != nil {
			t.Fatal(err)
		}
		other := newCanonicalTestService(pool, otherAccount, now, nil)
		if _, err := other.approve(ctx, fixture.candidateID, "operator-branch"); err == nil {
			t.Fatal("cross-account pending workflow was approved")
		}
		if lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, fixture.candidateID.String()) != 0 {
			t.Fatal("cross-account approval created an entry queue")
		}
	})
}

func TestCanonicalPaperHandoffPostgresFailsClosedAtQueueTime(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Microsecond)

	t.Run("missing PAPER account", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "missing-account", .5)
		service := newCanonicalTestService(pool, fixture.accountID+"-missing", now, nil)
		if _, err := service.prepare(ctx, fixture.candidateID); err == nil {
			t.Fatal("PREPARE accepted a missing configured PAPER account")
		}
		assertCanonicalHandoffNoDecisionOrQueue(t, ctx, pool, fixture.candidateID)
	})

	t.Run("wrong provider", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "wrong-provider", .5)
		service := newCanonicalTestService(pool, fixture.accountID, now, nil)
		if _, err := service.prepare(ctx, fixture.candidateID); err != nil {
			t.Fatal(err)
		}
		service.market.AllowedSources = []string{"unapproved-provider"}
		if _, err := service.approve(ctx, fixture.candidateID, "operator-fail-closed"); err == nil {
			t.Fatal("APPROVE accepted a quote from a disallowed provider")
		}
		assertCanonicalHandoffNoDecisionOrQueue(t, ctx, pool, fixture.candidateID)
	})

	t.Run("materially divergent fresh entry price", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "price-divergence", .5)
		service := newCanonicalTestService(pool, fixture.accountID, now, nil)
		if _, err := service.prepare(ctx, fixture.candidateID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE quotes SET price=110,bid=109.9,ask=110.1,timestamp=$1,received_at=$1 WHERE symbol='QQQ'`, now); err != nil {
			t.Fatal(err)
		}
		if _, err := service.approve(ctx, fixture.candidateID, "operator-fail-closed"); err == nil {
			t.Fatal("APPROVE accepted a fresh price outside explicit slippage allowance")
		}
		assertCanonicalHandoffNoDecisionOrQueue(t, ctx, pool, fixture.candidateID)
	})

	t.Run("unsupported short opening", func(t *testing.T) {
		fixture := createCanonicalHandoffFixture(t, ctx, pool, now, "short-opening", .5)
		if _, err := pool.Exec(ctx, `UPDATE candidate_trades SET direction='SHORT',signal_type='SELL' WHERE id=$1`, fixture.candidateID); err != nil {
			t.Fatal(err)
		}
		service := newCanonicalTestService(pool, fixture.accountID, now, nil)
		if _, err := service.prepare(ctx, fixture.candidateID); err == nil {
			t.Fatal("PREPARE enabled unsupported short PAPER exposure")
		}
		assertCanonicalHandoffNoDecisionOrQueue(t, ctx, pool, fixture.candidateID)
	})
}

func assertCanonicalHandoffNoDecisionOrQueue(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID uuid.UUID) {
	t.Helper()
	if lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1 AND decision IN ('approved','rejected')`, candidateID) != 0 ||
		lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, candidateID.String()) != 0 ||
		lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1`, candidateID) != 0 ||
		lenMustCount(t, ctx, pool, `SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1`, candidateID) != 0 {
		t.Fatal("failed canonical handoff created unintended approval, queue, instruction, or ticket state")
	}
}

type canonicalHandoffFixture struct {
	candidateID uuid.UUID
	accountID   string
}

func createCanonicalHandoffFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time, label string, leverage float64) canonicalHandoffFixture {
	t.Helper()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	fixtureSource := "canonical-fixture-" + suffix
	accountID := "canonical-account-" + label + "-" + suffix
	strategyID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,config,config_hash) VALUES($1,$2,'etf_news_sector_momentum_v1','etf_news_sector_momentum_v1',TRUE,$3::jsonb,$4)`, strategyID, "canonical-"+suffix, `{"symbols":["QQQ"]}`, "canonical-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO event_sources(id,display_name,provider_type) VALUES($1,$2,'external')`, fixtureSource, fixtureSource); err != nil {
		t.Fatal(err)
	}
	rawID, normalizedID, inboxID := uuid.New(), uuid.New(), uuid.New()
	sourceEventID := "canonical-source-event-" + suffix
	article := "https://canonical.example/event/" + suffix
	eventAt := now.Add(-2 * time.Minute)
	if _, err := pool.Exec(ctx, `INSERT INTO event_raw(id,source_id,source_event_id,event_kind,event_time,received_at,payload,content_hash,data_source_type) VALUES($1,$2,$3,'macro_rates',$4,$5,'{}'::jsonb,$6,'real')`, rawID, fixtureSource, sourceEventID, eventAt, now, suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO event_normalized(id,raw_event_id,canonical_key,event_kind,title,summary,severity,event_time,source_id,primary_symbol,confidence,attributes,data_source_type,is_synthetic) VALUES($1,$2,$3,'macro_rates',$4,$5,'medium',$6,$7,'QQQ',0.8,'{}'::jsonb,'real',FALSE)`, normalizedID, rawID, "canonical-event-"+suffix, "Federal Reserve policy decision", "The published policy decision changes technology-rate sensitivity.", eventAt, fixtureSource); err != nil {
		t.Fatal(err)
	}
	summary := "The Federal Reserve published a policy decision affecting technology markets."
	mapping := "World Monitor maps the normalized macro-rates event to QQQ technology duration exposure."
	if _, err := pool.Exec(ctx, `INSERT INTO world_monitor_research_inbox(id,source,world_monitor_event_id,source_event_id,status,event_type,headline,summary,source_urls,source_count,event_time,possible_affected_etfs,asset_themes,severity,source_tier,confidence,confidence_reasons,mapping_reason,dedupe_key,raw_payload,normalized_event_id) VALUES($1,$2,$3,$4,'new','macro_rates',$5,$6,$7::jsonb,2,$8,'["QQQ"]'::jsonb,'["technology"]'::jsonb,'medium','tier1',0.8,'["provider confidence"]'::jsonb,$9,$10,'{}'::jsonb,$11)`, inboxID, fixtureSource, sourceEventID, sourceEventID, "Federal Reserve announces policy decision", summary, fmt.Sprintf(`["%s"]`, article), eventAt, mapping, "canonical-dedupe-"+suffix, normalizedID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,exchange,provider,received_at) VALUES('QQQ',100,99.9,100.1,1000,1000,100000,$1,'fixture-exchange',$2,$1) ON CONFLICT (symbol) DO UPDATE SET price=EXCLUDED.price,bid=EXCLUDED.bid,ask=EXCLUDED.ask,bid_size=EXCLUDED.bid_size,ask_size=EXCLUDED.ask_size,volume=EXCLUDED.volume,timestamp=EXCLUDED.timestamp,exchange=EXCLUDED.exchange,provider=EXCLUDED.provider,received_at=EXCLUDED.received_at`, now, fixtureSource); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM candles WHERE symbol='QQQ' AND source=$1`, fixtureSource); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 25; index++ {
		at := now.Add(-time.Duration(25-index) * time.Hour)
		closePrice := 95 + float64(index)*.25
		if _, err := pool.Exec(ctx, `INSERT INTO candles(symbol,timestamp,open,high,low,close,volume,vwap,timeframe,source,timestamp_semantics,market_data_classification,ingested_at) VALUES('QQQ',$1,$2,$2,$3,$2,$4,$2,'1h',$5,'provider_observation','substitute',$1)`, at, closePrice, closePrice-.5, 10000+index, fixtureSource); err != nil {
			t.Fatal(err)
		}
	}
	row := worldMonitorInboxPromotionRow{ID: inboxID, SourceEventID: sourceEventID, NormalizedEventID: &normalizedID, RawEventID: &rawID,
		NormalizedSummary: summary, EventType: "macro_rates", Headline: "Federal Reserve announces policy decision", Summary: summary,
		SourceURLs: []string{article}, SourceCount: 2, PossibleAffectedETFs: []string{"QQQ"}, AssetThemes: []string{"technology"},
		Confidence: .8, ConfidenceReasons: []string{"provider confidence"}, MappingReason: mapping, EventTime: eventAt}
	market := marketDataSafetyPolicy{AllowedSources: []string{fixtureSource}, Timeframe: "1h", MaxAge: 48 * time.Hour, allowNonProductionSources: true}
	inputPolicy := &candidateEconomicPolicy{PolicyVersion: "canonical-test-policy-v1", IdentityPolicy: "canonical-test-identity-v1", IdentitySource: "disposable-integration-fixture",
		SizingPolicyID: "canonical-test-sizing-v1", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: leverage, SlippageAllowance: floatPointer(.5),
		Instruments: map[string]candidateEconomicIdentity{"QQQ": {InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq"}}}
	promoter := newWorldMonitorOpportunityPromoter(pool)
	promoter.marketPolicy, promoter.economicPolicy, promoter.now = &market, inputPolicy, func() time.Time { return now }
	promoted, _, err := promoter.promoteRow(ctx, row)
	if err != nil || promoted == nil {
		t.Fatalf("produce canonical candidate fixture: promoted=%+v err=%v", promoted, err)
	}
	candidateID, err := uuid.Parse(promoted.CandidateID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO genuine_event_decisions(source_inbox_event_id,normalized_event_id,source_event_identity,decision,decision_version,ruleset_version,processor_identity,processing_mode,decision_at,event_publication_at,event_collection_at,event_receipt_at,source,source_url,event_type,severity,evidence_score,evidence_score_source,confidence,affected_assets,unknown_assets,asset_mapping_provenance,reasons,blocking_reasons,missing_evidence,trust_gate_state,risk_review_state,candidate_id,replay_identity,input_fingerprint,replay_metadata,is_current) VALUES($1,$2,$3,'CANDIDATE',1,'canonical-test-v1','canonical-test-processor','deterministic',$4,$5,$6,$7,$8,$9,'macro_rates','medium',0.8,'fixture',0.8,ARRAY['QQQ'],FALSE,'{}'::jsonb,ARRAY['source-backed event'],ARRAY[]::text[],ARRAY[]::text[],'sufficient','ready',$10,$11,$12,'{}'::jsonb,TRUE)`, inboxID, normalizedID, sourceEventID, now, eventAt, now, now, fixtureSource, article, candidateID, "canonical-decision-"+suffix, "canonical-fingerprint-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO paper_accounts(account_id,contract_version,environment,currency,initial_cash,cash,equity,realized_pnl,fees,updated_at) VALUES($1,$2,'PAPER','USD',100000,100000,100000,0,0,$3)`, accountID, "jax.paper.ledger/v1", now); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM candles WHERE symbol='QQQ' AND source=$1`, fixtureSource)
		_, _ = pool.Exec(context.Background(), `DELETE FROM quotes WHERE symbol='QQQ' AND provider=$1`, fixtureSource)
	})
	return canonicalHandoffFixture{candidateID: candidateID, accountID: accountID}
}

func newCanonicalTestService(pool *pgxpool.Pool, accountID string, now time.Time, policy *portfoliorisk.RiskPolicy) *canonicalHandoffService {
	if policy == nil {
		built := canonicalTestRiskPolicy(100_000)
		policy = &built
	}
	economic := candidateEconomicPolicy{PolicyVersion: "canonical-test-policy-v1", IdentityPolicy: "canonical-test-identity-v1", IdentitySource: "disposable-integration-fixture",
		SizingPolicyID: "canonical-test-sizing-v1", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: .5, SlippageAllowance: floatPointer(.5),
		Instruments: map[string]candidateEconomicIdentity{"QQQ": {InstrumentID: "instrument-test-qqq", IssuerID: "issuer-test-qqq"}}}
	market := marketDataSafetyPolicy{AllowedSources: []string{"canonical-fixture"}, Timeframe: "1h", MaxAge: 48 * time.Hour, allowNonProductionSources: true}
	// Resolve the per-fixture source from the durable most recent quote.
	var source string
	_ = pool.QueryRow(context.Background(), `SELECT provider FROM quotes WHERE symbol='QQQ'`).Scan(&source)
	market.AllowedSources = []string{source}
	calendar := ops01Calendar(now)
	return &canonicalHandoffService{pool: pool, accountID: accountID, now: func() time.Time { return now }, market: market, policy: *policy, economic: economic, maxAge: 48 * time.Hour, calendar: &calendar}
}

func canonicalTestRiskPolicy(maximumPositionValue float64) portfoliorisk.RiskPolicy {
	return mustCanonicalPolicy(portfoliorisk.RiskPolicy{Version: "canonical-test-risk-v1", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(.5),
		MaximumPositionValue: portfoliorisk.Limit(maximumPositionValue), MaximumConcentration: portfoliorisk.Limit(1), MaximumGrossExposure: portfoliorisk.Limit(100_000), MaximumNetExposure: portfoliorisk.Limit(100_000), MaximumRiskAllocation: portfoliorisk.Limit(.02)})
}

func mustCanonicalPolicy(input portfoliorisk.RiskPolicy) portfoliorisk.RiskPolicy {
	policy, err := portfoliorisk.BuildRiskPolicy(input)
	if err != nil {
		panic(err)
	}
	return policy
}

func canonicalHandoffHTTP(t *testing.T, handler http.Handler, method, path, token, spoofHeader, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if spoofHeader != "" {
		request.Header.Set("X-User-ID", spoofHeader)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func lenMustCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
