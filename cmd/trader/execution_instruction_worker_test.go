package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/internal/modules/execution"
)

func insertCandidateForExecutionGuardTest(t *testing.T, pool *pgxpool.Pool, status string) uuid.UUID {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	candidateID := uuid.New()
	instanceID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO candidate_trades (
			id, strategy_instance_id, symbol, signal_type, status, session_date, detected_at, data_provenance, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, CURRENT_DATE, NOW(), $6, NOW(), NOW()
		)
	`, candidateID, instanceID, "AAPL", "BUY", status, "test:execution-guard")
	if err != nil {
		t.Fatalf("insert candidate: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id = $1`, candidateID)
	})

	return candidateID
}

func TestRequireInternalPaperExecuteAllowsOnlyApprovedStatuses(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	allowed := []string{"approved", "submitted", "filled"}
	for _, status := range allowed {
		t.Run("allows_"+status, func(t *testing.T) {
			candidateID := insertCandidateForExecutionGuardTest(t, pool, status)
			if err := requireInternalPaperExecute(ctx, pool, candidateID); err != nil {
				t.Fatalf("requireInternalPaperExecute(%q) error = %v, want nil", status, err)
			}
		})
	}

	blockedID := insertCandidateForExecutionGuardTest(t, pool, "detected")
	err := requireInternalPaperExecute(ctx, pool, blockedID)
	if err == nil {
		t.Fatal("expected non-approved candidate to be rejected")
	}
	if !strings.Contains(err.Error(), "not approved for execution") {
		t.Fatalf("error = %v, want approval message", err)
	}
}

func TestNormalizeExecutionStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"filled":    "filled",
		" Filled ":  "filled",
		"cancelled": "cancelled",
		"canceled":  "cancelled",
		"rejected":  "rejected",
		"submitted": "submitted",
		"OPEN":      "open",
		"":          "",
	}

	for input, want := range cases {
		if got := normalizeExecutionStatus(input); got != want {
			t.Fatalf("normalizeExecutionStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestExecutionInstructionWorkerSafetyEnabled(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("JAX_TRADER_RUNTIME_MODE", "")
	t.Setenv("IB_PAPER_TRADING", "true")
	t.Setenv("ALLOW_LIVE_TRADING", "false")
	if !executionInstructionWorkerSafetyEnabled() {
		t.Fatal("expected paper runtime safety settings to enable worker")
	}

	t.Setenv("ALLOW_LIVE_TRADING", "true")
	if executionInstructionWorkerSafetyEnabled() {
		t.Fatal("expected live trading flag to disable worker")
	}

	t.Setenv("ALLOW_LIVE_TRADING", "false")
	t.Setenv("IB_PAPER_TRADING", "false")
	if executionInstructionWorkerSafetyEnabled() {
		t.Fatal("expected non-paper IB flag to disable worker")
	}

	t.Setenv("IB_PAPER_TRADING", "true")
	t.Setenv("JAX_RUNTIME_MODE", "dev")
	if executionInstructionWorkerSafetyEnabled() {
		t.Fatal("expected non-paper runtime to disable worker")
	}
}

func TestExecutionInstructionWorkerSafetyFailsClosed(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "missing all env",
			env:  map[string]string{},
		},
		{
			name: "paper mode missing IB paper flag",
			env: map[string]string{
				"JAX_RUNTIME_MODE": "paper",
			},
		},
		{
			name: "IB paper without runtime mode",
			env: map[string]string{
				"IB_PAPER_TRADING": "true",
			},
		},
		{
			name: "live flag defaults unsafe even in paper",
			env: map[string]string{
				"JAX_RUNTIME_MODE":        "paper",
				"IB_PAPER_TRADING":        "true",
				"ALLOW_LIVE_TRADING":      "true",
				"JAX_TRADER_RUNTIME_MODE": "",
			},
		},
		{
			name: "research mode is closed",
			env: map[string]string{
				"JAX_RUNTIME_MODE":   "research",
				"IB_PAPER_TRADING":   "true",
				"ALLOW_LIVE_TRADING": "false",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JAX_RUNTIME_MODE", "")
			t.Setenv("JAX_TRADER_RUNTIME_MODE", "")
			t.Setenv("IB_PAPER_TRADING", "")
			t.Setenv("ALLOW_LIVE_TRADING", "")
			for key, value := range tt.env {
				t.Setenv(key, value)
			}
			if executionInstructionWorkerSafetyEnabled() {
				t.Fatalf("worker enabled for unsafe env %#v", tt.env)
			}
		})
	}
}

func TestHistoricalPendingInstructionIsQuarantinedBeforePickup(t *testing.T) {
	// Regression proof for historical pending instruction
	// 0f98a45b-5c64-4964-8cf7-b6e6f2aaaf68. Use an isolated test UUID so
	// the test can never mutate the operator's persisted instruction.
	pool := testFrontendAPIPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	candidateID := insertCandidateForExecutionGuardTest(t, pool, "approved")
	approvalID := uuid.New()
	instructionID := uuid.New()
	historicalCreatedAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	workerStartedAt := historicalCreatedAt.Add(time.Hour)

	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_approvals
			(id, candidate_id, decision, approved_by, decided_at, created_at)
		VALUES ($1, $2, 'approved', 'test:historical-instruction-guard', $3, $3)
	`, approvalID, candidateID, historicalCreatedAt); err != nil {
		t.Fatalf("insert approval: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO execution_instructions
			(id, approval_id, candidate_id, symbol, signal_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, 'AAPL', 'BUY', 'pending', $4, $4)
	`, instructionID, approvalID, candidateID, historicalCreatedAt); err != nil {
		t.Fatalf("insert historical instruction: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM execution_instructions WHERE id = $1`, instructionID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_approvals WHERE id = $1`, approvalID)
	})

	builder := execution.NewInstructionBuilder(pool)
	quarantined, err := builder.QuarantinePendingThrough(ctx, workerStartedAt, "test: historical pending instruction")
	if err != nil {
		t.Fatalf("quarantine historical instructions: %v", err)
	}
	if quarantined < 1 {
		t.Fatalf("quarantined rows = %d, want at least 1", quarantined)
	}

	var status string
	var errorMessage *string
	var brokerOrderID *string
	var tradeID *string
	if err := pool.QueryRow(ctx, `
		SELECT status, error_message, broker_order_id, trade_id
		FROM execution_instructions
		WHERE id = $1
	`, instructionID).Scan(&status, &errorMessage, &brokerOrderID, &tradeID); err != nil {
		t.Fatalf("read quarantined instruction: %v", err)
	}
	if status != "cancelled" {
		t.Fatalf("instruction status = %q, want cancelled", status)
	}
	if errorMessage == nil || !strings.Contains(*errorMessage, "historical pending instruction") {
		t.Fatalf("instruction error message = %v, want quarantine reason", errorMessage)
	}
	if brokerOrderID != nil || tradeID != nil {
		t.Fatalf("quarantined instruction gained execution linkage: brokerOrderID=%v tradeID=%v", brokerOrderID, tradeID)
	}
}
