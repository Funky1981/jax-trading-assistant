package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorldMonitorResearch_NoTradeCreated(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceEventID = "wm-smoke-" + uuid.NewString()
	trigger.TimestampUTC = now.Add(-5 * time.Minute)

	beforeCandidates := countRows(t, pool, "candidate_trades")
	beforeApprovals := countRows(t, pool, "candidate_approvals")
	beforeExecutions := countRows(t, pool, "execution_instructions")

	receipt, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("ingest world monitor trigger: %v", err)
	}
	if receipt.Status != worldMonitorInboxStatusNew {
		t.Fatalf("receipt status = %q, want %q", receipt.Status, worldMonitorInboxStatusNew)
	}
	if receipt.EventID == "" {
		t.Fatal("expected normalized event id in receipt")
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
	})

	var inboxCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM world_monitor_research_inbox
		WHERE source_event_id = $1 AND status = 'new' AND normalized_event_id = $2::uuid AND candidate_id IS NULL
	`, trigger.SourceEventID, receipt.EventID).Scan(&inboxCount); err != nil {
		t.Fatalf("query inbox row: %v", err)
	}
	if inboxCount != 1 {
		t.Fatalf("inbox rows = %d, want 1", inboxCount)
	}

	var normalizedCount int
	var rawSynthetic, normalizedSynthetic bool
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)::int, BOOL_OR(r.is_synthetic), BOOL_OR(n.is_synthetic)
		FROM event_normalized n
		JOIN event_raw r ON r.id = n.raw_event_id
		WHERE n.id = $1::uuid AND n.event_kind = 'research_trigger' AND n.source_id = 'world-monitor'
	`, receipt.EventID).Scan(&normalizedCount, &rawSynthetic, &normalizedSynthetic); err != nil {
		t.Fatalf("query normalized event: %v", err)
	}
	if normalizedCount != 1 {
		t.Fatalf("normalized rows = %d, want 1", normalizedCount)
	}
	if rawSynthetic || normalizedSynthetic {
		t.Fatalf("ordinary live input classified synthetic: raw=%t normalized=%t", rawSynthetic, normalizedSynthetic)
	}

	for _, symbol := range []string{"QQQ", "SPY", "TLT"} {
		var mapped bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM event_symbol_map
				WHERE normalized_event_id = $1::uuid AND symbol = $2
			)
		`, receipt.EventID, symbol).Scan(&mapped); err != nil {
			t.Fatalf("query symbol map %s: %v", symbol, err)
		}
		if !mapped {
			t.Fatalf("expected %s to be mapped to event %s", symbol, receipt.EventID)
		}
	}

	if got := countRows(t, pool, "candidate_trades"); got != beforeCandidates {
		t.Fatalf("candidate_trades count changed from %d to %d", beforeCandidates, got)
	}
	if got := countRows(t, pool, "candidate_approvals"); got != beforeApprovals {
		t.Fatalf("candidate_approvals count changed from %d to %d", beforeApprovals, got)
	}
	if got := countRows(t, pool, "execution_instructions"); got != beforeExecutions {
		t.Fatalf("execution_instructions count changed from %d to %d", beforeExecutions, got)
	}
}

func TestWorldMonitorResearch_SyntheticProvenancePersistsAndDuplicateIsStable(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	trigger := validWorldMonitorResearchTrigger(time.Now().UTC())
	trigger.Source = "world-monitor-local-proof"
	trigger.SourceEventID = "wm-synthetic-" + uuid.NewString()
	trigger.IsSynthetic = boolPointer(true)

	receipt, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("ingest explicit synthetic trigger: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
	})

	assertSynthetic := func() {
		var rawSynthetic, normalizedSynthetic bool
		var rawType, normalizedType, rawReason, normalizedReason string
		if err := pool.QueryRow(ctx, `
			SELECT r.is_synthetic, n.is_synthetic, r.data_source_type, n.data_source_type,
			       COALESCE(r.synthetic_reason, ''), COALESCE(n.synthetic_reason, '')
			FROM event_normalized n JOIN event_raw r ON r.id=n.raw_event_id
			WHERE n.id=$1::uuid
		`, receipt.EventID).Scan(&rawSynthetic, &normalizedSynthetic, &rawType, &normalizedType, &rawReason, &normalizedReason); err != nil {
			t.Fatalf("query synthetic provenance: %v", err)
		}
		if !rawSynthetic || !normalizedSynthetic || rawType != "synthetic" || normalizedType != "synthetic" || rawReason == "" || normalizedReason == "" {
			t.Fatalf("synthetic provenance not retained: raw=%t/%s/%q normalized=%t/%s/%q", rawSynthetic, rawType, rawReason, normalizedSynthetic, normalizedType, normalizedReason)
		}
	}
	assertSynthetic()

	duplicate, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("repost synthetic trigger: %v", err)
	}
	if !duplicate.Duplicate || duplicate.EventID != receipt.EventID {
		t.Fatalf("duplicate receipt = %+v, want same event %s", duplicate, receipt.EventID)
	}
	assertSynthetic()
}

func requireWorldMonitorSmokeSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, table := range []string{"world_monitor_research_inbox", "event_normalized", "event_symbol_map", "candidate_trades", "candidate_approvals", "execution_instructions"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, table).Scan(&exists); err != nil {
			t.Fatalf("check schema table %s: %v", table, err)
		}
		if !exists {
			t.Skipf("skip world monitor smoke test: table %s missing; apply migrations through 000030", table)
		}
	}
}

func countRows(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	if strings.ContainsAny(table, `"; `) {
		t.Fatalf("unsafe table name %q", table)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}
