package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAISuggestionPromoteCreatesApprovalCandidate(t *testing.T) {
	pool := testFrontendAPIPool(t)
	requireWorldMonitorSmokeSchema(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := pool.Exec(ctx, `
		INSERT INTO quotes (symbol, price, bid, ask, bid_size, ask_size, volume, timestamp, exchange, updated_at)
		VALUES ('SOXX', 240.00, 239.95, 240.05, 100, 100, 100000, NOW(), 'TEST', NOW())
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

	payload := aiSuggestionPromoteRequest{
		Symbol:     "SOXX",
		Action:     "BUY",
		Confidence: 0.64,
		Reasoning:  "Agent0 sees semiconductor ETF momentum with manageable paper risk.",
		Risk:       "medium",
		Source:     "agent0_manual_review",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/suggestions/promote", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	aiSuggestionPromoteHandler(pool)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response aiSuggestionPromoteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Route != "approval_required" || response.Status != "awaiting_approval" {
		t.Fatalf("unexpected response: %+v", response)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE candidate_id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE candidate_id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1::uuid`, response.CandidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id = $1::uuid`, response.SignalID)
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
	if status != "awaiting_approval" || signalID != response.SignalID {
		t.Fatalf("candidate status/signal = %q/%q, want awaiting_approval/%q", status, signalID, response.SignalID)
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
}

func TestAISuggestionPromoteRejectsWatchOnlySuggestion(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/suggestions/promote", bytes.NewBufferString(`{
		"symbol":"SPY",
		"action":"WATCH",
		"confidence":0.7,
		"reasoning":"watch only",
		"risk":"low",
		"source":"agent0_manual_review"
	}`))
	rec := httptest.NewRecorder()

	aiSuggestionPromoteHandler(nil)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
