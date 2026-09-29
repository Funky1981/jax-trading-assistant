package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOpportunityScannerDoesNotPromoteWithoutReadyCandidateEconomics(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("ALLOW_LIVE_TRADING", "false")

	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	_, _ = pool.Exec(ctx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id LIKE 'wm-scanner-%' AND status = 'new' AND candidate_id IS NULL`)
	instanceID := uuid.New()
	instanceName := "wm-scanner-test-" + uuid.NewString()
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceEventID = "wm-scanner-" + uuid.NewString()
	trigger.TimestampUTC = now.Add(-5 * time.Minute)
	trigger.PossibleAffectedETFs = []string{"SOXX"}
	trigger.Confidence = 0.78

	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_instances (
			id, name, strategy_type_id, strategy_id, enabled,
			session_timezone, flatten_by_close_time, config, config_hash
		)
		VALUES (
			$1, $2, 'etf_news_sector_momentum_v1', 'etf_news_sector_momentum_v1', true,
			'America/New_York', '15:55', '{"symbols":["SOXX"]}'::jsonb, $2
		)
	`, instanceID, instanceName)
	if err != nil {
		t.Fatalf("insert strategy instance: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, received_at, updated_at)
		VALUES ('SOXX', 500.00, 499.95, 500.05, 100, 100, 100000, NOW(), 'TEST', NOW(), NOW())
		ON CONFLICT (symbol) DO UPDATE
		SET price = EXCLUDED.price,
		    bid = EXCLUDED.bid,
		    ask = EXCLUDED.ask,
		    bid_size = EXCLUDED.bid_size,
		    ask_size = EXCLUDED.ask_size,
		    timestamp = EXCLUDED.timestamp,
		    received_at = EXCLUDED.received_at,
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

	scanner := newOpportunityScanner(pool)
	scanner.marketPolicy = &marketDataSafetyPolicy{AllowedSources: []string{"core-readiness-fixture"}, Timeframe: "1h", QuoteMaxAge: time.Minute, LatestCompletedCandleMaxAge: 90 * time.Minute, CandleHistoryLookback: 7 * 24 * time.Hour, allowNonProductionSources: true}
	result, err := scanner.ScanOnce(ctx)
	if err != nil {
		t.Fatalf("scan once: %v", err)
	}
	if result.Promoted != 0 || result.Skipped == 0 {
		t.Fatalf("scanner promotion result=%+v, want skipped/non-promoted while candidate readiness inputs are unavailable", result)
	}

	var candidateID, signalID string
	_ = pool.QueryRow(ctx, `SELECT COALESCE(candidate_id::text,''), COALESCE((SELECT signal_id::text FROM candidate_trades WHERE id=w.candidate_id),'') FROM world_monitor_research_inbox w WHERE source_event_id=$1`, trigger.SourceEventID).Scan(&candidateID, &signalID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if candidateID != "" {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE candidate_id = $1::uuid`, candidateID)
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE candidate_id = $1::uuid`, candidateID)
		}
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		if candidateID != "" {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, candidateID)
		}
		if signalID != "" {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id = $1::uuid`, signalID)
		}
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_instances WHERE id = $1`, instanceID)
	})

	if candidateID != "" {
		var status, gateStatus, riskStatus, approvalStatus string
		if err := pool.QueryRow(ctx, `SELECT status,gate_status,risk_status,approval_status FROM candidate_trades WHERE id=$1::uuid`, candidateID).Scan(&status, &gateStatus, &riskStatus, &approvalStatus); err != nil {
			t.Fatalf("query candidate state: %v", err)
		}
		if status != "awaiting_approval" || gateStatus != "risk_pending" || riskStatus != "pending" || approvalStatus != "risk_not_ready" {
			t.Fatalf("candidate crossed economic readiness boundary: status=%s gate=%s risk=%s approval=%s", status, gateStatus, riskStatus, approvalStatus)
		}
	}

	again, err := scanner.ScanOnce(ctx)
	if err != nil {
		t.Fatalf("scan again: %v", err)
	}
	if again.Promoted != 0 {
		t.Fatalf("second scan promoted = %d, want 0", again.Promoted)
	}

	if candidateID != "" {
		var executionCount int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1::uuid`, candidateID).Scan(&executionCount); err != nil {
			t.Fatalf("query execution instructions: %v", err)
		}
		if executionCount != 0 {
			t.Fatalf("execution instructions = %d, want 0", executionCount)
		}
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
