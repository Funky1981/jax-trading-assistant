package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOpportunityScannerPromotesEligibleWorldMonitorTrigger(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("ALLOW_LIVE_TRADING", "false")

	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceEventID = "wm-scanner-" + uuid.NewString()
	trigger.TimestampUTC = now.Add(-5 * time.Minute)
	trigger.PossibleAffectedETFs = []string{"SOXX"}
	trigger.Confidence = 0.78

	_, err := pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ('SOXX', 500.00, 499.95, 500.05, 100, 100, 100000, NOW(), 'TEST', NOW())
		ON CONFLICT (symbol) DO UPDATE
		SET price = EXCLUDED.price,
		    bid = EXCLUDED.bid,
		    ask = EXCLUDED.ask,
		    bid_size = EXCLUDED.bid_size,
		    ask_size = EXCLUDED.ask_size,
		    timestamp = EXCLUDED.timestamp,
		    updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		t.Fatalf("insert quote: %v", err)
	}
	insertWorldMonitorChartCandles(t, ctx, pool, "SOXX", now, []float64{
		460, 462, 464, 466, 468, 470, 472, 474, 476, 478,
		480, 482, 484, 486, 488, 490, 492, 494, 496, 500,
	})

	state := defaultAIScannerState()
	state.Symbols = []string{"SOXX"}
	state.MinimumConfidence = 0.7
	state.IntervalSeconds = 60
	if err := saveAIScannerState(ctx, pool, state); err != nil {
		t.Fatalf("save scanner state: %v", err)
	}

	receipt, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("ingest trigger: %v", err)
	}

	result, err := newOpportunityScanner(pool).ScanOnce(ctx)
	if err != nil {
		t.Fatalf("scan once: %v", err)
	}
	if result.Promoted != 1 {
		t.Fatalf("promoted = %d, want 1; result=%+v", result.Promoted, result)
	}

	var candidateID string
	var signalID string
	if err := pool.QueryRow(ctx, `
		SELECT w.candidate_id::text, c.signal_id::text
		FROM world_monitor_research_inbox w
		JOIN candidate_trades c ON c.id = w.candidate_id
		WHERE w.source_event_id = $1
	`, trigger.SourceEventID).Scan(&candidateID, &signalID); err != nil {
		t.Fatalf("query inbox candidate: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE candidate_id = $1::uuid`, candidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE candidate_id = $1::uuid`, candidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, candidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id = $1::uuid`, signalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
	})

	var candidateStatus string
	if err := pool.QueryRow(ctx, `
		SELECT status
		FROM candidate_trades
		WHERE id = $1::uuid
	`, candidateID).Scan(&candidateStatus); err != nil {
		t.Fatalf("query candidate status: %v", err)
	}
	if candidateStatus != "awaiting_approval" {
		t.Fatalf("candidate status = %q, want awaiting_approval", candidateStatus)
	}

	again, err := newOpportunityScanner(pool).ScanOnce(ctx)
	if err != nil {
		t.Fatalf("scan again: %v", err)
	}
	if again.Promoted != 0 {
		t.Fatalf("second scan promoted = %d, want 0", again.Promoted)
	}

	var executionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM execution_instructions
		WHERE candidate_id = $1::uuid
	`, candidateID).Scan(&executionCount); err != nil {
		t.Fatalf("query execution instructions: %v", err)
	}
	if executionCount != 0 {
		t.Fatalf("execution instructions = %d, want 0", executionCount)
	}
}

func TestOpportunityScannerSkipsWhenRuntimeIsNotPaper(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "dev")
	t.Setenv("ALLOW_LIVE_TRADING", "false")

	result, err := newOpportunityScanner(nil).ScanOnce(context.Background())
	if err != nil {
		t.Fatalf("scan once: %v", err)
	}
	if !result.Disabled {
		t.Fatal("expected scanner to be disabled without a pool/runtime")
	}
}
