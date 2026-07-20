package approvals

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestApprovalDetailPersistedStatesAndDuplicateSafety(t *testing.T) {
	ctx := context.Background()
	pool := testApprovalDetailPool(t)

	tests := []struct {
		name       string
		status     string
		decision   string
		withTicket bool
		wantState  string
		wantTicket bool
	}{
		{name: "no decision", status: "awaiting_approval", wantState: "no_decision"},
		{name: "approval only", status: "approved", decision: DecisionApproved, wantState: "approval_persisted_ticket_missing"},
		{name: "approval and ticket", status: "approved", decision: DecisionApproved, withTicket: true, wantState: "approval_and_ticket_persisted", wantTicket: true},
		{name: "rejection", status: "rejected", decision: DecisionRejected, wantState: "rejected"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidateID := insertApprovalDetailFixture(t, ctx, pool, tt.status, tt.decision, tt.withTicket)
			detail, err := NewStore(pool).GetDetailByCandidateID(ctx, candidateID)
			if err != nil {
				t.Fatalf("get approval detail: %v", err)
			}
			if detail.State != tt.wantState || (detail.PaperTicket != nil) != tt.wantTicket {
				t.Fatalf("detail state = %q ticket=%v, want state=%q ticket=%v", detail.State, detail.PaperTicket != nil, tt.wantState, tt.wantTicket)
			}

			assertApprovalFixtureHasNoExecution(t, ctx, pool, candidateID)
			if tt.decision == DecisionApproved {
				_, err := NewService(pool).Decide(ctx, ApprovalRequest{
					CandidateID: candidateID,
					Decision:    DecisionApproved,
					ApprovedBy:  "same-operator",
				})
				if !errors.Is(err, ErrNotAwaitingApproval) {
					t.Fatalf("duplicate approval error = %v, want ErrNotAwaitingApproval", err)
				}
				var approvalCount, ticketCount int
				if err := pool.QueryRow(ctx, `
					SELECT
						(SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id = $1),
						(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id = $1)
				`, candidateID).Scan(&approvalCount, &ticketCount); err != nil {
					t.Fatalf("count duplicate safety rows: %v", err)
				}
				wantTickets := 0
				if tt.withTicket {
					wantTickets = 1
				}
				if approvalCount != 1 || ticketCount != wantTickets {
					t.Fatalf("after duplicate request approvals=%d tickets=%d, want approvals=1 tickets=%d", approvalCount, ticketCount, wantTickets)
				}
			}
		})
	}
}

func testApprovalDetailPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip DB-backed approval detail test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skip DB-backed approval detail test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func insertApprovalDetailFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, status, decision string, withTicket bool) uuid.UUID {
	t.Helper()
	candidateID := uuid.New()
	instanceID := uuid.New()
	symbol := "T" + strings.ToUpper(strings.ReplaceAll(candidateID.String()[:7], "-", ""))
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_trades
			(id, strategy_instance_id, symbol, signal_type, status, session_date, data_provenance, metadata)
		VALUES ($1, $2, $3, 'BUY', $4, CURRENT_DATE, 'approval-detail-test', '{"paperOnly":true}'::jsonb)
	`, candidateID, instanceID, symbol, status); err != nil {
		t.Fatalf("insert candidate fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM candidate_paper_tickets WHERE candidate_id = $1`, candidateID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM execution_instructions WHERE candidate_id = $1`, candidateID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM candidate_approvals WHERE candidate_id = $1`, candidateID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM candidate_trades WHERE id = $1`, candidateID)
	})

	approvalID := uuid.Nil
	if decision != "" {
		approvalID = uuid.New()
		if _, err := pool.Exec(ctx, `
			INSERT INTO candidate_approvals
				(id, candidate_id, decision, approved_by, notes, reanalysis_requested, decided_at, created_at)
			VALUES ($1, $2, $3, 'same-operator', 'same reason', false, NOW(), NOW())
		`, approvalID, candidateID, decision); err != nil {
			t.Fatalf("insert approval fixture: %v", err)
		}
	}

	if withTicket {
		if _, err := pool.Exec(ctx, `
			INSERT INTO candidate_paper_tickets (
				paper_ticket_id, candidate_id, source_approval_id, approval_decision_ref,
				symbol, direction, setup_type, catalyst_summary, entry_price, stop_loss_price,
				target_price, position_size, max_normal_loss, max_slippage_adjusted_loss,
				reward_risk_ratio, evidence_status, gate_status, risk_status, approval_status,
				paper_only, broker_execution_allowed, execution_instruction_created,
				live_trading_allowed, leverage_allowed, reject_reasons, warning_reasons, review_notes
			) VALUES (
				$1, $2, $3, $3::text, $4, 'long', 'test', 'test', 100, 99, 102, 1, 1, 1,
				2, 'sufficient', 'ready_for_risk_review', 'ready_for_approval_review',
				'paper_ticket_ready', true, false, false, false, false, '{}', '{}', NULL
			)
		`, "pt_"+candidateID.String(), candidateID, approvalID, symbol); err != nil {
			t.Fatalf("insert paper ticket fixture: %v", err)
		}
	}
	return candidateID
}

func assertApprovalFixtureHasNoExecution(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID uuid.UUID) {
	t.Helper()
	var instructions, brokerOrders, trades int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id = $1),
			(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id = $1 AND broker_order_id IS NOT NULL),
			(SELECT COUNT(*) FROM trades WHERE id IN (SELECT trade_id FROM execution_instructions WHERE candidate_id = $1))
	`, candidateID).Scan(&instructions, &brokerOrders, &trades); err != nil {
		t.Fatalf("query execution safety: %v", err)
	}
	if instructions != 0 || brokerOrders != 0 || trades != 0 {
		t.Fatalf("unsafe persisted state: instructions=%d brokerOrders=%d trades=%d", instructions, brokerOrders, trades)
	}
}
