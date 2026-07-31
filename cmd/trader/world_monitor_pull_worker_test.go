package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/instruments"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoadWorldMonitorPullConfigFailsClosedAndBoundsValues(t *testing.T) {
	lookup := func(values map[string]string) func(string) (string, bool) {
		return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	}
	disabled, err := loadWorldMonitorPullConfig(lookup(nil))
	if err != nil || disabled.Enabled {
		t.Fatalf("disabled config = %+v, err=%v", disabled, err)
	}
	valid, err := loadWorldMonitorPullConfig(lookup(map[string]string{
		"WORLD_MONITOR_PULL_ENABLED": "true", "WORLD_MONITOR_EVENTS_URL": "http://worldmonitor-events:8082/api/v1/jax/events",
		"WORLD_MONITOR_PULL_INTERVAL_SECONDS": "5", "WORLD_MONITOR_PULL_TIMEOUT_SECONDS": "1", "WORLD_MONITOR_PULL_PAGE_SIZE": "250",
	}))
	if err != nil || !valid.Enabled || valid.PageSize != 250 || valid.Interval != 5*time.Second || valid.Timeout != time.Second {
		t.Fatalf("valid config = %+v, err=%v", valid, err)
	}
	for _, values := range []map[string]string{
		{"WORLD_MONITOR_PULL_ENABLED": "true"},
		{"WORLD_MONITOR_PULL_ENABLED": "true", "WORLD_MONITOR_EVENTS_URL": "file:///tmp/events"},
		{"WORLD_MONITOR_PULL_ENABLED": "true", "WORLD_MONITOR_EVENTS_URL": "http://wm/events", "WORLD_MONITOR_PULL_INTERVAL_SECONDS": "0"},
		{"WORLD_MONITOR_PULL_ENABLED": "true", "WORLD_MONITOR_EVENTS_URL": "http://wm/events", "WORLD_MONITOR_PULL_PAGE_SIZE": "251"},
	} {
		if _, err := loadWorldMonitorPullConfig(lookup(values)); err == nil {
			t.Fatalf("expected invalid config for %+v", values)
		}
	}
}

func TestWorldMonitorFetchPageUsesCursorAndValidatesContract(t *testing.T) {
	collected := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") != "7" || r.URL.Query().Get("limit") != "2" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{{EventID: "wm_8", PersistenceSeq: "8", Title: "Event", FeedURL: "https://example.com/feed", CollectedAt: collected}}, NextCursor: "8", Count: 1})
	}))
	defer server.Close()
	worker := worldMonitorPullWorker{config: worldMonitorPullConfig{Endpoint: server.URL, PageSize: 2}, httpClient: server.Client()}
	page, err := worker.fetchPage(context.Background(), 7)
	if err != nil || len(page.Events) != 1 || page.NextCursor != "8" {
		t.Fatalf("page = %+v, err=%v", page, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{{EventID: "wm_7", PersistenceSeq: "7"}}, NextCursor: "7", Count: 1})
	}))
	defer bad.Close()
	worker.config.Endpoint = bad.URL
	worker.httpClient = bad.Client()
	if _, err := worker.fetchPage(context.Background(), 7); err == nil {
		t.Fatal("expected non-monotonic page rejection")
	}
}

func TestWorldMonitorFetchPageTreatsUnavailableServiceAsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	worker := worldMonitorPullWorker{config: worldMonitorPullConfig{Endpoint: server.URL, PageSize: 10}, httpClient: server.Client()}
	if _, err := worker.fetchPage(context.Background(), 0); err == nil {
		t.Fatal("expected unavailable service error")
	}
}

func TestWorldMonitorPullTriggerPreservesOldPublicationAndHonestMissingPublication(t *testing.T) {
	collected := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	article := "https://example.com/a"
	base := worldMonitorPullEvent{EventID: "wm_1", PersistenceSeq: "1", SourceID: "source", SourceName: "Source", FeedURL: "https://example.com/feed", ArticleURL: &article, Title: "Old event", CollectedAt: collected, FirstSeenAt: collected, LastSeenAt: collected, SchemaVersion: 1, Provenance: map[string]any{"event_type": "unknown"}}
	base.PublicationTime = &old
	trigger, err := worldMonitorPullTrigger(base)
	if err != nil || !trigger.AllowStalePublication || !trigger.TimestampUTC.Equal(old) || len(trigger.PossibleAffectedETFs) != 0 {
		t.Fatalf("old-publication trigger = %+v, err=%v", trigger, err)
	}
	base.PublicationTime = nil
	trigger, err = worldMonitorPullTrigger(base)
	if err != nil || !trigger.TimestampUTC.Equal(collected) || trigger.RawPayload["publication_time_supplied"] != false {
		t.Fatalf("missing-publication trigger = %+v, err=%v", trigger, err)
	}
}

func TestWorldMonitorPullTriggerAllowsNewsTradeWordsOnlyOnTrustedPath(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	article := "https://example.com/markets"
	item := worldMonitorPullEvent{
		EventID: "wm_trade_words", PersistenceSeq: "33", SourceID: "source", SourceName: "Source",
		FeedURL: "https://example.com/feed", ArticleURL: &article, Title: "What investors buy and sell after the announcement",
		PublicationTime: &now, CollectedAt: now, FirstSeenAt: now, LastSeenAt: now, SchemaVersion: 1,
		Provenance: map[string]any{"event_type": "unknown"},
	}
	trigger, err := worldMonitorPullTrigger(item)
	if err != nil {
		t.Fatalf("worldMonitorPullTrigger() error = %v", err)
	}
	if !trigger.AllowNewsTradeLanguage {
		t.Fatal("trusted pull trigger did not enable news-language exemption")
	}
	if result := validateWorldMonitorResearchTrigger(trigger, now); !result.Valid {
		t.Fatalf("trusted pull trigger rejected ordinary news language: %+v", result)
	}
	trigger.AllowNewsTradeLanguage = false
	if result := validateWorldMonitorResearchTrigger(trigger, now); result.Valid || !strings.Contains(result.Reason, "trade instruction") {
		t.Fatalf("external-equivalent trigger was not rejected: %+v", result)
	}
}

func TestWorldMonitorPullWorkerAtomicCommitReplayAndRollback(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorPullSchema(t, pool)
	rules, err := eventdecisions.LoadRuleset("../../config/genuine-event-decision-v1.json")
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	goodID := "wm_pull_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	goodIDTwo := goodID + "_second"
	rolledBackID := goodID + "_rollback"
	invalidID := goodID + "_invalid"
	makeEvent := func(id, sequence, title string) worldMonitorPullEvent {
		article := "https://example.com/" + id
		return worldMonitorPullEvent{EventID: id, PersistenceSeq: sequence, SourceID: "integration-proof", SourceName: "Integration Proof", FeedURL: "https://example.com/feed.xml", ArticleURL: &article, Title: title, Summary: "Oil supply update", PublicationTime: &now, CollectedAt: now, FirstSeenAt: now, LastSeenAt: now, ContentHash: id, SchemaVersion: 1, Provenance: map[string]any{"event_type": "energy_oil"}}
	}
	goodEvent := makeEvent(goodID, "1", "Federal Reserve announces interest rate decision")
	goodEvent.Summary = "The Federal Reserve published its interest rate decision after the policy meeting."
	goodEvent.Provenance["event_type"] = "macro_rates"
	firstArticle := "https://www.federalreserve.gov/newsevents/pressreleases/" + goodID + ".htm"
	goodEvent.ArticleURL = &firstArticle
	goodEventTwo := makeEvent(goodIDTwo, "2", "Fed publishes interest rate decision after policy meeting")
	goodEventTwo.Summary = "The Federal Reserve interest rate decision followed the scheduled policy meeting."
	goodEventTwo.Provenance["event_type"] = "macro_rates"
	secondArticle := "https://www.sec.gov/newsroom/press-releases/" + goodIDTwo
	goodEventTwo.ArticleURL = &secondArticle
	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		after := r.URL.Query().Get("after")
		if after == "0" {
			_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{goodEvent, goodEventTwo}, NextCursor: "2", Count: 2})
			return
		}
		_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{}, NextCursor: after, Count: 0})
	}))
	defer goodServer.Close()

	newWorker := func(endpoint string, client *http.Client) *worldMonitorPullWorker {
		return &worldMonitorPullWorker{pool: pool, config: worldMonitorPullConfig{Enabled: true, Endpoint: endpoint, PageSize: 10, Timeout: 2 * time.Second}, httpClient: client, replayer: eventdecisions.Replayer{Store: eventdecisions.NewStore(pool), Evaluator: eventdecisions.Evaluator{Ruleset: rules, Catalog: catalog}}, now: func() time.Time { return now }}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	prohibited := []string{"candidate_approvals", "candidate_paper_tickets", "execution_instructions", "order_intents", "trades", "fills"}
	before := map[string]int{}
	for _, table := range prohibited {
		before[table] = countRows(t, pool, table)
	}

	worker := newWorker(goodServer.URL, goodServer.Client())
	first, err := worker.cycle(ctx)
	if err != nil || first.Cursor != 2 || first.Ingested != 2 || first.DecisionsCreated != 2 {
		t.Fatalf("first cycle = %+v, err=%v", first, err)
	}
	assertSubjectPersistenceCounts(t, pool, []string{goodID, goodIDTwo}, 1, 2, 2)
	if _, err := pool.Exec(ctx, `DELETE FROM world_monitor_pull_cursors WHERE consumer_name=$1 AND source_endpoint_identity=$2`, worldMonitorPullConsumer, goodServer.URL); err != nil {
		t.Fatalf("rewind cursor: %v", err)
	}
	replay, err := worker.cycle(ctx)
	if err != nil || replay.Cursor != 2 || replay.Duplicates != 2 || replay.DecisionsReused != 2 {
		t.Fatalf("replay cycle = %+v, err=%v", replay, err)
	}
	assertSubjectPersistenceCounts(t, pool, []string{goodID, goodIDTwo}, 1, 2, 2)

	rollbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(worldMonitorPullPage{Events: []worldMonitorPullEvent{makeEvent(rolledBackID, "1", "Must roll back"), makeEvent(invalidID, "2", "")}, NextCursor: "2", Count: 2})
	}))
	defer rollbackServer.Close()
	if _, err := newWorker(rollbackServer.URL, rollbackServer.Client()).cycle(ctx); err == nil {
		t.Fatal("expected mid-page failure")
	}
	for _, id := range []string{rolledBackID, invalidID} {
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM world_monitor_research_inbox WHERE source_event_id=$1`, id).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rolled-back inbox %s count=%d err=%v", id, count, err)
		}
	}
	var rollbackCursorCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM world_monitor_pull_cursors WHERE source_endpoint_identity=$1`, rollbackServer.URL).Scan(&rollbackCursorCount); err != nil || rollbackCursorCount != 0 {
		t.Fatalf("rollback cursor count=%d err=%v", rollbackCursorCount, err)
	}

	unavailable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }))
	defer unavailable.Close()
	if _, err := newWorker(unavailable.URL, unavailable.Client()).cycle(ctx); err == nil {
		t.Fatal("expected pre-transaction service failure")
	}
	for _, table := range prohibited {
		if after := countRows(t, pool, table); after != before[table] {
			t.Fatalf("%s changed from %d to %d", table, before[table], after)
		}
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_pull_cursors WHERE source_endpoint_identity IN ($1,$2,$3)`, goodServer.URL, rollbackServer.URL, unavailable.URL)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM evidence_subjects WHERE id IN (SELECT l.subject_id FROM evidence_subject_events l JOIN world_monitor_research_inbox w ON w.id=l.genuine_event_id WHERE w.source_event_id=ANY($1))`, []string{goodID, goodIDTwo, rolledBackID, invalidID})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM genuine_event_decisions WHERE source_event_identity=ANY($1)`, []string{"world-monitor:" + goodID, "world-monitor:" + goodIDTwo})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id=ANY($1)`, []string{goodID, goodIDTwo, rolledBackID, invalidID})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE raw_event_id IN (SELECT id FROM event_raw WHERE source_event_id=ANY($1))`, []string{goodID, goodIDTwo, rolledBackID, invalidID})
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_event_id=ANY($1)`, []string{goodID, goodIDTwo, rolledBackID, invalidID})
	})
}

func assertSubjectPersistenceCounts(t *testing.T, pool *pgxpool.Pool, sourceEventIDs []string, subjects, links, evaluations int) {
	t.Helper()
	var gotSubjects, gotLinks, gotEvaluations int
	err := pool.QueryRow(t.Context(), `
		SELECT COUNT(DISTINCT s.id)::int,COUNT(DISTINCT l.id)::int,COUNT(DISTINCT e.id)::int
		FROM evidence_subjects s
		JOIN evidence_subject_events l ON l.subject_id=s.id
		JOIN world_monitor_research_inbox w ON w.id=l.genuine_event_id
		LEFT JOIN evidence_subject_evaluations e ON e.subject_id=s.id
		WHERE w.source_event_id=ANY($1)
	`, sourceEventIDs).Scan(&gotSubjects, &gotLinks, &gotEvaluations)
	if err != nil || gotSubjects != subjects || gotLinks != links || gotEvaluations != evaluations {
		t.Fatalf("subject persistence counts=%d/%d/%d want=%d/%d/%d err=%v", gotSubjects, gotLinks, gotEvaluations, subjects, links, evaluations, err)
	}
}

func requireWorldMonitorPullSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, table := range []string{"world_monitor_pull_cursors", "world_monitor_research_inbox", "genuine_event_decisions", "evidence_subjects", "evidence_subject_events", "evidence_subject_evaluations"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("check %s: %v", table, err)
		}
		if !exists {
			t.Skipf("skip World Monitor pull integration: table %s is missing", table)
		}
	}
}
