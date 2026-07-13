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

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
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
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_paper_tickets WHERE candidate_id = $1::uuid`, response.CandidateID)
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

	candidateID, err := uuid.Parse(response.CandidateID)
	if err != nil {
		t.Fatalf("parse candidate id: %v", err)
	}
	approval, err := approvalsmod.NewService(pool).Decide(ctx, approvalsmod.ApprovalRequest{
		CandidateID: candidateID,
		Decision:    approvalsmod.DecisionApproved,
		ApprovedBy:  "test-operator",
	})
	if err != nil {
		t.Fatalf("approve candidate: %v", err)
	}
	if approval.Decision != approvalsmod.DecisionApproved {
		t.Fatalf("approval decision = %q, want approved", approval.Decision)
	}

	var approvedStatus string
	var approvalStatus string
	if err := pool.QueryRow(ctx, `
		SELECT status, approval_status
		FROM candidate_trades
		WHERE id = $1::uuid
	`, response.CandidateID).Scan(&approvedStatus, &approvalStatus); err != nil {
		t.Fatalf("query approved candidate: %v", err)
	}
	if approvedStatus != "approved" {
		t.Fatalf("candidate status after approval = %q, want approved", approvedStatus)
	}
	if approvalStatus != "paper_ticket_ready" {
		t.Fatalf("approval status after approval = %q, want paper_ticket_ready", approvalStatus)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM execution_instructions
		WHERE candidate_id = $1::uuid AND approval_id = $2
	`, response.CandidateID, approval.ID).Scan(&executionCount); err != nil {
		t.Fatalf("query execution instructions after approval: %v", err)
	}
	if executionCount != 0 {
		t.Fatalf("execution instruction count after approval = %d, want 0", executionCount)
	}

	var ticketCount int
	var paperOnly, brokerAllowed, instructionCreated, liveAllowed, leverageAllowed bool
	var ticketStatus string
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*), bool_and(paper_only), bool_or(broker_execution_allowed),
		       bool_or(execution_instruction_created), bool_or(live_trading_allowed),
		       bool_or(leverage_allowed), max(status)
		FROM candidate_paper_tickets
		WHERE candidate_id = $1::uuid AND source_approval_id = $2
	`, response.CandidateID, approval.ID).Scan(&ticketCount, &paperOnly, &brokerAllowed, &instructionCreated, &liveAllowed, &leverageAllowed, &ticketStatus); err != nil {
		t.Fatalf("query candidate paper tickets after approval: %v", err)
	}
	if ticketCount != 1 {
		t.Fatalf("candidate paper ticket count = %d, want 1", ticketCount)
	}
	if ticketStatus != "paper_ticket_created" || !paperOnly || brokerAllowed || instructionCreated || liveAllowed || leverageAllowed {
		t.Fatalf("persisted paper ticket is not review-only: status=%q paperOnly=%v broker=%v instruction=%v live=%v leverage=%v",
			ticketStatus, paperOnly, brokerAllowed, instructionCreated, liveAllowed, leverageAllowed)
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
