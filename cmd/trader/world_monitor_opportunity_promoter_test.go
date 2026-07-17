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

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
	candidatesmod "jax-trading-assistant/internal/modules/candidates"
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
	trigger.PossibleAffectedETFs = []string{"XLE", "QQQ"}

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
	unsupportedEarlierRow := worldMonitorInboxPromotionRow{ID: uuid.New(), SourceEventID: "unsupported-earlier-row"}
	result, err := promoter.promoteRows(ctx, []worldMonitorInboxPromotionRow{unsupportedEarlierRow, row})
	if err != nil {
		t.Fatalf("promote world monitor batch: %v", err)
	}
	if result.PromotedCount != 1 || len(result.Promoted) != 1 {
		t.Fatalf("unsupported earlier row blocked later QQQ promotion: %+v", result)
	}
	outcomes := result.Outcomes
	if len(outcomes) != 3 || outcomes[0].ReasonCode != "no_symbols" || outcomes[1].Symbol != "XLE" || outcomes[1].Status == "promoted" || outcomes[2].Symbol != "QQQ" || outcomes[2].CandidateID == "" {
		t.Fatalf("unexpected promotion outcomes: %+v", outcomes)
	}
	promoted := &result.Promoted[0]
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
	var setupType, catalystSummary, invalidationReason, rawSourceRef, sourcePayloadRef, decisionLogRef string
	var entryPrice, stopLoss, takeProfit float64
	if err := pool.QueryRow(ctx, `
		SELECT status, signal_id::text, COALESCE(metadata, '{}'::jsonb), entry_price::float8, stop_loss::float8, take_profit::float8,
		       setup_type, catalyst_summary, invalidation_reason,
		       COALESCE(raw_source_ref, ''), COALESCE(source_payload_ref, ''), COALESCE(decision_log_ref, '')
		FROM candidate_trades
		WHERE id = $1::uuid
	`, promoted.CandidateID).Scan(&candidateStatus, &signalID, &metadata, &entryPrice, &stopLoss, &takeProfit,
		&setupType, &catalystSummary, &invalidationReason, &rawSourceRef, &sourcePayloadRef, &decisionLogRef); err != nil {
		t.Fatalf("query promoted candidate: %v", err)
	}
	if candidateStatus != "awaiting_approval" {
		t.Fatalf("candidate status = %q, want awaiting_approval after structural validation", candidateStatus)
	}
	if signalID != promoted.SignalID {
		t.Fatalf("candidate signal_id = %q, want %q", signalID, promoted.SignalID)
	}
	if entryPrice != 500 || stopLoss != 490 || takeProfit != 520 {
		t.Fatalf("unexpected prices entry=%f stop=%f target=%f", entryPrice, stopLoss, takeProfit)
	}
	if setupType != "sector_news_momentum" {
		t.Fatalf("setup_type = %q, want sector_news_momentum", setupType)
	}
	if catalystSummary != trigger.Summary {
		t.Fatalf("catalyst_summary = %q, want normalized World Monitor summary %q", catalystSummary, trigger.Summary)
	}
	if invalidationReason != "QQQ trades at or below the candidate stop level 490.00, invalidating the confirmed sector-news momentum setup." {
		t.Fatalf("unexpected invalidation reason: %q", invalidationReason)
	}
	if !strings.HasPrefix(rawSourceRef, "event_raw:") || !strings.Contains(sourcePayloadRef, row.ID.String()) || !strings.Contains(decisionLogRef, receipt.EventID) {
		t.Fatalf("provenance refs not retained: raw=%q payload=%q normalized=%q", rawSourceRef, sourcePayloadRef, decisionLogRef)
	}
	if !json.Valid(metadata) || !containsJSONKey(metadata, "worldMonitor") || !containsJSONKey(metadata, "sizing") {
		t.Fatalf("metadata should include source URLs and sizing evidence, got %s", string(metadata))
	}
	if !strings.Contains(string(metadata), `"sourceURLs"`) || !strings.Contains(string(metadata), `"shares": 10`) || !strings.Contains(string(metadata), trigger.SourceURLs[0]) {
		t.Fatalf("metadata should include calculated 10-share paper size and monitor URL, got %s", string(metadata))
	}
	assertSwingPaperMetadata(t, metadata)

	var evidenceItemCount int
	var evidenceStatus, gateStatus string
	var evidenceReady, evidenceGateReady bool
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM candidate_evidence_items WHERE candidate_id = $1::uuid),
			es.evidence_status, es.evidence_ready, es.evidence_gate_ready, ct.gate_status
		FROM candidate_trades ct
		JOIN LATERAL (
			SELECT evidence_status, evidence_ready, evidence_gate_ready
			FROM candidate_evidence_scores
			WHERE candidate_id = ct.id
			ORDER BY scored_at DESC
			LIMIT 1
		) es ON TRUE
		WHERE ct.id = $1::uuid
	`, promoted.CandidateID).Scan(&evidenceItemCount, &evidenceStatus, &evidenceReady, &evidenceGateReady, &gateStatus); err != nil {
		t.Fatalf("query persisted evidence evaluation: %v", err)
	}
	if evidenceItemCount != 2 || evidenceStatus != "sufficient" || !evidenceReady || !evidenceGateReady || gateStatus != "ready_for_risk_review" {
		t.Fatalf("unexpected evidence/gate result: items=%d evidence=%s ready=%v gateReady=%v gate=%s", evidenceItemCount, evidenceStatus, evidenceReady, evidenceGateReady, gateStatus)
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

	t.Run("missing evidence remains approval ineligible at every boundary", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `DELETE FROM candidate_evidence_scores WHERE candidate_id = $1::uuid`, promoted.CandidateID); err != nil {
			t.Fatalf("remove sufficient evidence score: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM candidate_evidence_items WHERE candidate_id = $1::uuid`, promoted.CandidateID); err != nil {
			t.Fatalf("remove evidence items: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO candidate_evidence_scores (
				candidate_id, support_score, contradiction_score, quality_score, freshness_score,
				overall_evidence_score, evidence_item_count, supporting_item_count,
				contradictory_item_count, stale_item_count, evidence_status, evidence_ready,
				evidence_gate_ready, broker_execution_allowed, execution_instruction_created,
				approval_granted
			) VALUES ($1::uuid, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'missing', false, false, false, false, false)
		`, promoted.CandidateID); err != nil {
			t.Fatalf("persist missing evidence score: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			UPDATE candidate_trades
			SET status = 'awaiting_approval', gate_status = 'evidence_missing',
				risk_status = 'ready_for_approval_review', approval_status = 'approval_review_ready'
			WHERE id = $1::uuid
		`, promoted.CandidateID); err != nil {
			t.Fatalf("arrange missing evidence state: %v", err)
		}

		queue, err := approvalsmod.NewService(pool).GetQueue(ctx, 25)
		if err != nil {
			t.Fatalf("get approval queue: %v", err)
		}
		if queueContainsCandidate(queue, promoted.CandidateID) {
			t.Fatalf("missing-evidence candidate %s appeared in approval queue", promoted.CandidateID)
		}

		candidateID := uuid.MustParse(promoted.CandidateID)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+promoted.CandidateID+"/approve", nil)
		req.Header.Set("X-User-ID", "safety-regression")
		res := httptest.NewRecorder()
		handleApprovalDecision(res, req, approvalsmod.NewService(pool), candidateID, approvalsmod.DecisionApproved)
		if res.Code >= 200 && res.Code < 300 {
			t.Fatalf("approval endpoint accepted missing evidence: status=%d body=%s", res.Code, res.Body.String())
		}

		var approvalsCount, paperTicketCount, instructionCount int
		if err := pool.QueryRow(ctx, `
			SELECT
				(SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id = $1::uuid),
				(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id = $1::uuid),
				(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id = $1::uuid)
		`, promoted.CandidateID).Scan(&approvalsCount, &paperTicketCount, &instructionCount); err != nil {
			t.Fatalf("query approval safety boundaries: %v", err)
		}
		if approvalsCount != 0 || paperTicketCount != 0 || instructionCount != 0 {
			t.Fatalf("missing evidence crossed a safety boundary: approvals=%d tickets=%d instructions=%d", approvalsCount, paperTicketCount, instructionCount)
		}
	})

	if _, err := pool.Exec(ctx, `UPDATE candidate_trades SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1::uuid`, promoted.CandidateID); err != nil {
		t.Fatalf("expire candidate for dedup regression: %v", err)
	}
	open, err := candidatesmod.NewStore(pool).HasOpenForInstanceSymbol(ctx, instanceID, "QQQ", now.Format("2006-01-02"))
	if err != nil {
		t.Fatalf("check open candidate dedup: %v", err)
	}
	if open {
		t.Fatal("expired awaiting_approval candidate must not block a fresh proof candidate")
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
		520, 519, 518, 517, 516, 515, 514, 513, 512, 511,
		510, 509, 508, 507, 506, 505, 504, 503, 502, 500,
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
	promoted, outcomes, err := promoter.promoteRow(ctx, row)
	if err != nil {
		t.Fatalf("promote world monitor trigger: %v", err)
	}
	if promoted == nil {
		t.Fatal("expected blocked candidate")
	}
	if len(outcomes) != 1 || outcomes[0].ReasonCode != "chart_confirmation_failed" {
		t.Fatalf("unexpected blocked outcomes: %+v", outcomes)
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

func TestWorldMonitorStructuredCandidateFieldsRequiresLegitimateInputs(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	row := worldMonitorInboxPromotionRow{
		ID:                uuid.New(),
		NormalizedEventID: uuidPointer(uuid.New()),
		RawEventID:        uuidPointer(uuid.New()),
		EventType:         "macro_rates",
		Summary:           "Raw inbox summary.",
		NormalizedSummary: "Normalized evidence summary.",
		SourceCount:       2,
		Confidence:        0.74,
		EventTime:         now,
	}
	chart := worldMonitorChartConfirmation{Confirmed: true, CandleCount: 30, LastClose: 500, SMA20: 495}
	fields := worldMonitorStructuredCandidateFields(row, "QQQ", "etf_news_sector_momentum_v1", 490, chart)
	if fields.CatalystSummary != row.NormalizedSummary {
		t.Fatalf("catalyst summary = %q, want normalized summary", fields.CatalystSummary)
	}
	if fields.SetupType != "sector_news_momentum" {
		t.Fatalf("setup type = %q", fields.SetupType)
	}
	if !strings.Contains(fields.InvalidationReason, "490.00") {
		t.Fatalf("invalidation reason lacks measured stop: %q", fields.InvalidationReason)
	}
	if fields.RawSourceRef == nil || fields.SourcePayloadRef == nil || fields.DecisionLogRef == nil {
		t.Fatalf("missing provenance refs: %+v", fields)
	}

	missing := row
	missing.NormalizedSummary = ""
	missing.Summary = ""
	missingFields := worldMonitorStructuredCandidateFields(missing, "QQQ", "unsupported_strategy_v1", 490, chart)
	if missingFields.CatalystSummary != "" || missingFields.SetupType != "" || missingFields.InvalidationReason != "" {
		t.Fatalf("unsupported or missing evidence received invented structured fields: %+v", missingFields)
	}
	candidate := candidatesmod.Candidate{
		Symbol:             "QQQ",
		SignalType:         "BUY",
		Direction:          "long",
		EntryPrice:         floatPointer(500),
		StopLoss:           floatPointer(490),
		SetupType:          missingFields.SetupType,
		CatalystSummary:    missingFields.CatalystSummary,
		InvalidationReason: missingFields.InvalidationReason,
	}
	validation := candidatesmod.ValidateStructuralCompleteness(candidate)
	if validation.GateReady || !containsTestString(validation.MissingFields, "setup_type") || !containsTestString(validation.MissingFields, "catalyst_summary") || !containsTestString(validation.MissingFields, "invalidation_reason") {
		t.Fatalf("structural validation was relaxed: %+v", validation)
	}
}

func uuidPointer(value uuid.UUID) *uuid.UUID { return &value }

func floatPointer(value float64) *float64 { return &value }

func containsTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
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
	var normalizedEventID, rawEventID uuid.NullUUID
	err = promoter.pool.QueryRow(ctx, `
		SELECT
			w.id,
			w.source_event_id,
			w.normalized_event_id,
			e.raw_event_id,
			COALESCE(e.summary, ''),
			w.event_type,
			w.headline,
			COALESCE(w.summary, ''),
			w.possible_affected_etfs,
			w.asset_themes,
			w.confidence,
			w.confidence_reasons,
			w.mapping_reason,
			w.event_time
		FROM world_monitor_research_inbox w
		JOIN event_normalized e ON e.id = w.normalized_event_id
		WHERE w.source_event_id = $1
	`, sourceEventID).Scan(
		&row.ID,
		&row.SourceEventID,
		&normalizedEventID,
		&rawEventID,
		&row.NormalizedSummary,
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
	if rawEventID.Valid {
		v := rawEventID.UUID
		row.RawEventID = &v
	}
	_ = json.Unmarshal(etfsRaw, &row.PossibleAffectedETFs)
	_ = json.Unmarshal(themesRaw, &row.AssetThemes)
	_ = json.Unmarshal(confidenceReasonsRaw, &row.ConfidenceReasons)
	return row, nil
}

func insertWorldMonitorChartCandles(t *testing.T, ctx context.Context, pool *pgxpool.Pool, symbol string, now time.Time, closes []float64) {
	t.Helper()
	timestamps := make([]time.Time, 0, len(closes))
	for i, close := range closes {
		ts := now.Add(time.Duration(i-len(closes)) * time.Minute)
		timestamps = append(timestamps, ts)
		_, err := pool.Exec(ctx, `
			INSERT INTO candles (symbol, timestamp, open, high, low, close, volume, vwap)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $6)
			ON CONFLICT DO NOTHING
		`, symbol, ts, close-0.2, close+0.5, close-0.5, close, 1000+i)
		if err != nil {
			t.Fatalf("insert candle %d: %v", i, err)
		}
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for i, ts := range timestamps {
			_, _ = pool.Exec(cleanupCtx, `DELETE FROM candles WHERE symbol = $1 AND timestamp = $2 AND volume = $3`, symbol, ts, 1000+i)
		}
	})
}

func containsJSONKey(raw []byte, key string) bool {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	_, ok := payload[key]
	return ok
}

func assertSwingPaperMetadata(t *testing.T, raw []byte) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("metadata is invalid JSON: %v", err)
	}
	horizon, ok := payload["horizonPolicy"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing horizonPolicy: %s", string(raw))
	}
	if horizon["horizon"] != "swing" {
		t.Fatalf("horizon = %#v, want swing", horizon["horizon"])
	}
	if maxHold, ok := horizon["maxHoldDays"].(float64); !ok || maxHold > 10 {
		t.Fatalf("maxHoldDays = %#v, want <= 10", horizon["maxHoldDays"])
	}
	if horizon["requiresDailyReview"] != true {
		t.Fatalf("requiresDailyReview = %#v, want true", horizon["requiresDailyReview"])
	}
	if payload["paperOnly"] != true {
		t.Fatalf("paperOnly = %#v, want true", payload["paperOnly"])
	}
	if payload["approvalRequired"] != true {
		t.Fatalf("approvalRequired = %#v, want true", payload["approvalRequired"])
	}
}
