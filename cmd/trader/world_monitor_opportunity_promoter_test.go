package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
)

func TestWorldMonitorOpportunityPromoterCreatesApprovalCandidate(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	instanceID := uuid.New()
	instanceName := "wm-promoter-test-" + uuid.NewString()
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceEventID = "wm-promote-" + uuid.NewString()
	trigger.TimestampUTC = now.Add(-5 * time.Minute)
	trigger.PossibleAffectedETFs = []string{"QQQ"}

	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_instances (
			id, name, strategy_type_id, strategy_id, enabled,
			session_timezone, flatten_by_close_time, config, config_hash
		)
		VALUES (
			$1, $2, 'etf_news_sector_momentum_v1', 'etf_news_sector_momentum_v1', true,
			'America/New_York', '15:55', '{"symbols":["QQQ"]}'::jsonb, $2
		)
	`, instanceID, instanceName)
	if err != nil {
		t.Fatalf("insert strategy instance: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ('QQQ', 500.00, 499.95, 500.05, 100, 100, 100000, NOW(), 'TEST', NOW())
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
	insertWorldMonitorChartCandles(t, ctx, pool, "QQQ", now, []float64{
		480, 482, 484, 486, 488, 490, 492, 494, 496, 498,
		500, 502, 504, 506, 508, 510, 512, 514, 516, 520,
	})

	receipt, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("ingest world monitor trigger: %v", err)
	}

	promoter := newWorldMonitorOpportunityPromoter(pool)
	row, err := loadWorldMonitorPromotionRowForTest(ctx, t, promoter, trigger.SourceEventID)
	if err != nil {
		t.Fatalf("load promotion row: %v", err)
	}
	promoted, err := promoter.promoteRow(ctx, row)
	if err != nil {
		t.Fatalf("promote world monitor trigger: %v", err)
	}
	if promoted.Symbol != "QQQ" || promoted.SignalID == "" || promoted.CandidateID == "" {
		t.Fatalf("unexpected promotion result: %+v", promoted)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE candidate_id = $1::uuid`, promoted.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE candidate_id = $1::uuid`, promoted.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, promoted.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id = $1::uuid`, promoted.SignalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_instances WHERE id = $1`, instanceID)
	})

	var candidateStatus string
	var signalID string
	var metadata []byte
	var entryPrice, stopLoss, takeProfit float64
	if err := pool.QueryRow(ctx, `
		SELECT status, signal_id::text, COALESCE(metadata, '{}'::jsonb), entry_price::float8, stop_loss::float8, take_profit::float8
		FROM candidate_trades
		WHERE id = $1::uuid
	`, promoted.CandidateID).Scan(&candidateStatus, &signalID, &metadata, &entryPrice, &stopLoss, &takeProfit); err != nil {
		t.Fatalf("query promoted candidate: %v", err)
	}
	if candidateStatus != "awaiting_approval" {
		t.Fatalf("candidate status = %q, want awaiting_approval", candidateStatus)
	}
	if signalID != promoted.SignalID {
		t.Fatalf("candidate signal_id = %q, want %q", signalID, promoted.SignalID)
	}
	if entryPrice != 500 || stopLoss != 490 || takeProfit != 520 {
		t.Fatalf("unexpected prices entry=%f stop=%f target=%f", entryPrice, stopLoss, takeProfit)
	}
	if !json.Valid(metadata) || !containsJSONKey(metadata, "worldMonitor") || !containsJSONKey(metadata, "sizing") {
		t.Fatalf("metadata should include source URLs and sizing evidence, got %s", string(metadata))
	}
	if !strings.Contains(string(metadata), `"sourceURLs"`) || !strings.Contains(string(metadata), `"shares": 10`) || !strings.Contains(string(metadata), trigger.SourceURLs[0]) {
		t.Fatalf("metadata should include calculated 10-share paper size and monitor URL, got %s", string(metadata))
	}

	queue, err := approvalsmod.NewService(pool).GetQueue(ctx, 25)
	if err != nil {
		t.Fatalf("get approval queue: %v", err)
	}
	if !queueContainsCandidate(queue, promoted.CandidateID) {
		t.Fatalf("approval queue missing promoted candidate %s: %+v", promoted.CandidateID, queue)
	}

	var inboxStatus string
	var inboxCandidateID string
	if err := pool.QueryRow(ctx, `
		SELECT status, candidate_id::text
		FROM world_monitor_research_inbox
		WHERE source_event_id = $1
	`, trigger.SourceEventID).Scan(&inboxStatus, &inboxCandidateID); err != nil {
		t.Fatalf("query inbox status: %v", err)
	}
	if inboxStatus != worldMonitorInboxStatusCandidateCreated || inboxCandidateID != promoted.CandidateID {
		t.Fatalf("inbox status/candidate = %q/%q, want %q/%q", inboxStatus, inboxCandidateID, worldMonitorInboxStatusCandidateCreated, promoted.CandidateID)
	}

	var executionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM execution_instructions
		WHERE candidate_id = $1::uuid
	`, promoted.CandidateID).Scan(&executionCount); err != nil {
		t.Fatalf("query execution instructions: %v", err)
	}
	if executionCount != 0 {
		t.Fatalf("execution instruction count = %d, want 0 before approval", executionCount)
	}
}

func TestWorldMonitorOpportunityPromoterBlocksWhenChartConfirmationMissing(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	instanceID := uuid.New()
	instanceName := "wm-chart-block-test-" + uuid.NewString()
	trigger := validWorldMonitorResearchTrigger(now)
	trigger.SourceEventID = "wm-chart-block-" + uuid.NewString()
	trigger.TimestampUTC = now.Add(-5 * time.Minute)
	trigger.PossibleAffectedETFs = []string{"WMZZ"}

	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_instances (
			id, name, strategy_type_id, strategy_id, enabled,
			session_timezone, flatten_by_close_time, config, config_hash
		)
		VALUES (
			$1, $2, 'etf_news_sector_momentum_v1', 'etf_news_sector_momentum_v1', true,
			'America/New_York', '15:55', '{"symbols":["WMZZ"]}'::jsonb, $2
		)
	`, instanceID, instanceName)
	if err != nil {
		t.Fatalf("insert strategy instance: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ('WMZZ', 500.00, 499.95, 500.05, 100, 100, 100000, NOW(), 'TEST', NOW())
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

	receipt, err := newWorldMonitorResearchInboxService(pool).Ingest(ctx, trigger)
	if err != nil {
		t.Fatalf("ingest world monitor trigger: %v", err)
	}

	promoter := newWorldMonitorOpportunityPromoter(pool)
	row, err := loadWorldMonitorPromotionRowForTest(ctx, t, promoter, trigger.SourceEventID)
	if err != nil {
		t.Fatalf("load promotion row: %v", err)
	}
	promoted, err := promoter.promoteRow(ctx, row)
	if err != nil {
		t.Fatalf("promote world monitor trigger: %v", err)
	}
	if promoted.Route != "blocked" {
		t.Fatalf("route = %q, want blocked", promoted.Route)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM world_monitor_research_inbox WHERE source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, promoted.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_normalized WHERE id = $1::uuid`, receipt.EventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM event_raw WHERE source_id = 'world-monitor' AND source_event_id = $1`, trigger.SourceEventID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_instances WHERE id = $1`, instanceID)
	})

	var candidateStatus, blockReason, reasonCode string
	var metadata []byte
	if err := pool.QueryRow(ctx, `
		SELECT status, COALESCE(block_reason, ''), COALESCE(blocked_reason_code, ''), COALESCE(metadata, '{}'::jsonb)
		FROM candidate_trades
		WHERE id = $1::uuid
	`, promoted.CandidateID).Scan(&candidateStatus, &blockReason, &reasonCode, &metadata); err != nil {
		t.Fatalf("query blocked candidate: %v", err)
	}
	if candidateStatus != "blocked" {
		t.Fatalf("candidate status = %q, want blocked", candidateStatus)
	}
	if reasonCode != "no_chart_confirmation" {
		t.Fatalf("blocked reason code = %q, want no_chart_confirmation", reasonCode)
	}
	if blockReason == "" {
		t.Fatalf("block reason should explain missing chart confirmation")
	}
	if !json.Valid(metadata) || !containsJSONKey(metadata, "chartConfirmation") {
		t.Fatalf("metadata should include chartConfirmation evidence, got %s", string(metadata))
	}

	queue, err := approvalsmod.NewService(pool).GetQueue(ctx, 25)
	if err != nil {
		t.Fatalf("get approval queue: %v", err)
	}
	if queueContainsCandidate(queue, promoted.CandidateID) {
		t.Fatalf("blocked candidate %s must not appear in approval queue", promoted.CandidateID)
	}
}

func queueContainsCandidate(queue []map[string]any, candidateID string) bool {
	for _, item := range queue {
		if item["id"] == candidateID {
			return true
		}
	}
	return false
}

func loadWorldMonitorPromotionRowForTest(ctx context.Context, t *testing.T, promoter *worldMonitorOpportunityPromoter, sourceEventID string) (worldMonitorInboxPromotionRow, error) {
	t.Helper()
	rows, err := promoter.loadPromotionRows(ctx, worldMonitorPromoterMaxLimit)
	if err == nil {
		for _, row := range rows {
			if row.SourceEventID == sourceEventID {
				return row, nil
			}
		}
	}
	var row worldMonitorInboxPromotionRow
	var etfsRaw, themesRaw, confidenceReasonsRaw []byte
	var normalizedEventID uuid.NullUUID
	err = promoter.pool.QueryRow(ctx, `
		SELECT
			id,
			source_event_id,
			normalized_event_id,
			event_type,
			headline,
			COALESCE(summary, ''),
			possible_affected_etfs,
			asset_themes,
			confidence,
			confidence_reasons,
			mapping_reason,
			event_time
		FROM world_monitor_research_inbox
		WHERE source_event_id = $1
	`, sourceEventID).Scan(
		&row.ID,
		&row.SourceEventID,
		&normalizedEventID,
		&row.EventType,
		&row.Headline,
		&row.Summary,
		&etfsRaw,
		&themesRaw,
		&row.Confidence,
		&confidenceReasonsRaw,
		&row.MappingReason,
		&row.EventTime,
	)
	if err != nil {
		return worldMonitorInboxPromotionRow{}, err
	}
	if normalizedEventID.Valid {
		v := normalizedEventID.UUID
		row.NormalizedEventID = &v
	}
	_ = json.Unmarshal(etfsRaw, &row.PossibleAffectedETFs)
	_ = json.Unmarshal(themesRaw, &row.AssetThemes)
	_ = json.Unmarshal(confidenceReasonsRaw, &row.ConfidenceReasons)
	return row, nil
}

func insertWorldMonitorChartCandles(t *testing.T, ctx context.Context, pool *pgxpool.Pool, symbol string, now time.Time, closes []float64) {
	t.Helper()
	for i, close := range closes {
		ts := now.Add(time.Duration(i-len(closes)) * time.Minute)
		_, err := pool.Exec(ctx, `
			INSERT INTO candles (symbol, timestamp, open, high, low, close, volume, vwap)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $6)
			ON CONFLICT DO NOTHING
		`, symbol, ts, close-0.2, close+0.5, close-0.5, close, 1000+i)
		if err != nil {
			t.Fatalf("insert candle %d: %v", i, err)
		}
	}
}

func containsJSONKey(raw []byte, key string) bool {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	_, ok := payload[key]
	return ok
}
