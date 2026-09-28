package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
	"jax-trading-assistant/libs/auth"
)

func TestWorldMonitorApprovalHandlerFailsClosedAndKeepsRejection(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx := context.Background()
	approvalCandidate, approvalSignal := insertWorldMonitorApprovalHTTPFixture(t, ctx, pool)
	rejectCandidate, rejectSignal := insertWorldMonitorApprovalHTTPFixture(t, ctx, pool)

	manager, err := auth.NewJWTManager(auth.Config{Secret: []byte("world-monitor-approval-boundary-test-secret"), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	token, err := manager.GenerateToken("operator-alice", "alice", "operator")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	registerApprovalRoutes(mux, manager.MiddlewareFunc, pool)

	beforeApprovalCounts := readWorldMonitorHTTPBoundaryCounts(t, ctx, pool, approvalCandidate, approvalSignal)
	approveReq := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+approvalCandidate.String()+"/approve", strings.NewReader(`{}`))
	approveReq.Header.Set("Authorization", "Bearer "+token)
	approveReq.Header.Set("X-User-ID", "operator-spoof")
	approveRec := httptest.NewRecorder()
	mux.ServeHTTP(approveRec, approveReq)
	if approveRec.Code != http.StatusConflict || !strings.Contains(approveRec.Body.String(), "canonical PAPER portfolio-risk handoff is not yet available") {
		t.Fatalf("World Monitor approve response=%d %s, want 409 canonical-handoff conflict", approveRec.Code, approveRec.Body.String())
	}
	afterApprovalCounts := readWorldMonitorHTTPBoundaryCounts(t, ctx, pool, approvalCandidate, approvalSignal)
	if beforeApprovalCounts != afterApprovalCounts {
		t.Fatalf("blocked HTTP approval mutated artifacts: before=%+v after=%+v", beforeApprovalCounts, afterApprovalCounts)
	}
	var unchangedStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, approvalCandidate).Scan(&unchangedStatus); err != nil {
		t.Fatal(err)
	}
	if unchangedStatus != "awaiting_approval" {
		t.Fatalf("blocked candidate status=%q, want awaiting_approval", unchangedStatus)
	}

	rejectReq := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/"+rejectCandidate.String()+"/reject", strings.NewReader(`{}`))
	rejectReq.Header.Set("Authorization", "Bearer "+token)
	rejectReq.Header.Set("X-User-ID", "operator-spoof")
	rejectRec := httptest.NewRecorder()
	mux.ServeHTTP(rejectRec, rejectReq)
	if rejectRec.Code != http.StatusOK {
		t.Fatalf("World Monitor rejection response=%d %s, want 200", rejectRec.Code, rejectRec.Body.String())
	}
	var decision, actor, signalStatus string
	var approved bool
	if err := pool.QueryRow(ctx, `SELECT decision,approved_by FROM candidate_approvals WHERE candidate_id=$1`, rejectCandidate).Scan(&decision, &actor); err != nil {
		t.Fatalf("load durable rejection actor: %v", err)
	}
	if decision != approvalsmod.DecisionRejected || actor != "operator-alice" {
		t.Fatalf("durable rejection=(decision=%q actor=%q), want rejected by JWT actor operator-alice", decision, actor)
	}
	if err := pool.QueryRow(ctx, `SELECT approved,status FROM trade_approvals JOIN strategy_signals ON strategy_signals.id=trade_approvals.signal_id WHERE signal_id=$1`, rejectSignal).Scan(&approved, &signalStatus); err != nil {
		t.Fatalf("load durable signal rejection: %v", err)
	}
	if approved || signalStatus != "rejected" {
		t.Fatalf("signal rejection=(approved=%v status=%q), want false/rejected", approved, signalStatus)
	}
	var candidateStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, rejectCandidate).Scan(&candidateStatus); err != nil {
		t.Fatal(err)
	}
	if candidateStatus != "rejected" {
		t.Fatalf("candidate status=%q, want rejected", candidateStatus)
	}
	var ticketCount, instructionCount, queueCount, lifecycleCount, orderCount int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text),
			(SELECT COUNT(*) FROM exploratory_paper_lifecycles WHERE candidate_id=$1::text),
			(SELECT COUNT(*) FROM paper_orders WHERE paper_intent_id IN (SELECT paper_intent_id FROM exploratory_paper_lifecycles WHERE candidate_id=$1::text))
	`, rejectCandidate).Scan(&ticketCount, &instructionCount, &queueCount, &lifecycleCount, &orderCount); err != nil {
		t.Fatalf("check rejection did not cross paper boundaries: %v", err)
	}
	if ticketCount != 0 || instructionCount != 0 || queueCount != 0 || lifecycleCount != 0 || orderCount != 0 {
		t.Fatalf("rejection created economic artifacts: tickets=%d instructions=%d queue=%d lifecycles=%d orders=%d", ticketCount, instructionCount, queueCount, lifecycleCount, orderCount)
	}
}

type worldMonitorHTTPBoundaryCounts struct {
	Approvals             int
	PaperTickets          int
	ExecutionInstructions int
	TradeApprovals        int
}

func readWorldMonitorHTTPBoundaryCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID, signalID uuid.UUID) worldMonitorHTTPBoundaryCounts {
	t.Helper()
	var counts worldMonitorHTTPBoundaryCounts
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM trade_approvals WHERE signal_id=$2)
	`, candidateID, signalID).Scan(&counts.Approvals, &counts.PaperTickets, &counts.ExecutionInstructions, &counts.TradeApprovals)
	if err != nil {
		t.Fatalf("read World Monitor HTTP boundary counts: %v", err)
	}
	return counts
}

func insertWorldMonitorApprovalHTTPFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	candidateID, signalID := uuid.New(), uuid.New()
	instanceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := pool.Exec(ctx, `
		INSERT INTO strategy_signals (id,symbol,strategy_id,signal_type,confidence,generated_at,expires_at,status)
		VALUES ($1,'QQQ',$2,'BUY',0.90,$3,$4,'pending')
	`, signalID, "wmhttp-"+uuid.NewString(), now, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("insert HTTP boundary signal fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_trades (
			id,strategy_instance_id,signal_id,symbol,signal_type,status,entry_price,stop_loss,take_profit,
			confidence,reasoning,session_date,source,setup_type,direction,catalyst_summary,
			invalidation_reason,expected_reward_risk_ratio,slippage_allowance,max_normal_loss,
			max_slippage_adjusted_loss,position_size,risk_status,human_approval_required,
			approval_status,gate_status,data_provenance
		) VALUES (
			$1,$2,$3,'QQQ','BUY','awaiting_approval',100,95,110,0.90,'disposable approval HTTP fixture',CURRENT_DATE,
			'world-monitor','event-driven','long','identified event catalyst',
			'contradicting evidence invalidates thesis',2,0.01,1,1.1,10,
			'ready_for_approval_review',true,'approval_review_ready','ready_for_risk_review','world-monitor'
		)
	`, candidateID, instanceID, signalID); err != nil {
		t.Fatalf("insert historically risk-ready HTTP candidate: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_evidence_scores (
			candidate_id,support_score,contradiction_score,quality_score,freshness_score,overall_evidence_score,
			evidence_item_count,supporting_item_count,contradictory_item_count,stale_item_count,evidence_status,
			evidence_ready,evidence_gate_ready,broker_execution_allowed,execution_instruction_created,approval_granted
		) VALUES ($1,0.9,0,0.9,1,0.9,2,2,0,0,'sufficient',true,true,false,false,false)
	`, candidateID); err != nil {
		t.Fatalf("insert sufficient HTTP evidence fixture: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM candidate_trades WHERE id=$1`, candidateID)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id=$1`, signalID)
	})
	return candidateID, signalID
}
