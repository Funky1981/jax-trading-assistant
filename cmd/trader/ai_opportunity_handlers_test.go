package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAISuggestionPromoteCreatesApprovalCandidate(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	instanceID := uuid.New()
	instanceName := "ai-suggestion-test-" + uuid.NewString()
	if _, err := pool.Exec(ctx, `INSERT INTO strategy_instances(id,name,strategy_type_id,strategy_id,enabled,session_timezone,flatten_by_close_time,config,config_hash) VALUES($1,$2,'etf_news_sector_momentum_v1','etf_news_sector_momentum_v1',TRUE,'America/New_York','15:55','{"symbols":["SOXX"]}'::jsonb,$2)`, instanceID, instanceName); err != nil {
		t.Fatalf("insert disposable strategy instance: %v", err)
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, last_trade_timestamp, exchange, provider, received_at, updated_at)
		VALUES ('SOXX', 240.00, 239.95, 240.05, 100, 100, 100000, NOW(), NOW(), 'TEST', 'core-readiness-fixture', NOW(), NOW())
		ON CONFLICT (symbol) DO UPDATE
		SET price = EXCLUDED.price,
		    bid = EXCLUDED.bid,
		    ask = EXCLUDED.ask,
		    bid_size = EXCLUDED.bid_size,
		    ask_size = EXCLUDED.ask_size,
		    timestamp = EXCLUDED.timestamp,
		    last_trade_timestamp = EXCLUDED.last_trade_timestamp,
		    provider = EXCLUDED.provider,
		    received_at = EXCLUDED.received_at,
		    updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		t.Fatalf("insert quote: %v", err)
	}

	payload := aiSuggestionPromoteRequest{
		Symbol:     "SOXX",
		Action:     "BUY",
		Confidence: 0.64,
		Reasoning:  "Jax review sees semiconductor ETF momentum with manageable paper risk.",
		Risk:       "medium",
		Source:     "jax_manual_review",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/suggestions/promote", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	promoter := newWorldMonitorOpportunityPromoter(pool)
	promoter.marketPolicy = &marketDataSafetyPolicy{AllowedSources: []string{"core-readiness-fixture"}, Timeframe: "1h", QuoteMaxAge: time.Minute, LatestCompletedCandleMaxAge: 90 * time.Minute, CandleHistoryLookback: 7 * 24 * time.Hour, allowNonProductionSources: true}
	aiSuggestionPromoteHandlerWithPromoter(pool, promoter)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response aiSuggestionPromoteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Route != "candidate_repair_required" || response.Status != "blocked" {
		t.Fatalf("unexpected response: %+v", response)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_paper_tickets WHERE candidate_id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE candidate_id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE candidate_id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id = $1::uuid`, response.SignalID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_instances WHERE id = $1`, instanceID)
	})

	var status string
	var signalID string
	if err := pool.QueryRow(ctx, `
		SELECT status, signal_id::text
		FROM candidate_trades
		WHERE id = $1::uuid
	`, response.CandidateID).Scan(&status, &signalID); err != nil {
		t.Fatalf("query candidate: %v", err)
	}
	if status != "blocked" || signalID != response.SignalID {
		t.Fatalf("candidate status/signal = %q/%q, want blocked/%q", status, signalID, response.SignalID)
	}

	var executionCount int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM execution_instructions
		WHERE candidate_id = $1::uuid
	`, response.CandidateID).Scan(&executionCount); err != nil {
		t.Fatalf("query execution instructions: %v", err)
	}
	if executionCount != 0 {
		t.Fatalf("execution instruction count = %d, want 0 before approval", executionCount)
	}

	var approvalCount, ticketCount int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1::uuid),
		(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1::uuid)
	`, response.CandidateID).Scan(&approvalCount, &ticketCount); err != nil {
		t.Fatal(err)
	}
	if approvalCount != 0 || ticketCount != 0 {
		t.Fatalf("structurally incomplete manual suggestion crossed approval boundary: approvals=%d tickets=%d", approvalCount, ticketCount)
	}
}

func TestAISuggestionPromoteRejectsWatchOnlySuggestion(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/suggestions/promote", bytes.NewBufferString(`{
		"symbol":"SPY",
		"action":"WATCH",
		"confidence":0.7,
		"reasoning":"watch only",
		"risk":"low",
		"source":"jax_manual_review"
	}`))
	rec := httptest.NewRecorder()

	aiSuggestionPromoteHandler(nil)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
