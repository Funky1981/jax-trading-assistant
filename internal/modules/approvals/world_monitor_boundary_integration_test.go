package approvals

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
)

func TestWorldMonitorApprovalRequiresCanonicalHandoffPostgres(t *testing.T) {
	ctx := context.Background()
	pool := testApprovalDetailPool(t)

	for _, withEconomicInput := range []bool{false, true} {
		name := "historical candidate without economic input"
		if withEconomicInput {
			name = "candidate economic input is not portfolio approval"
		}
		t.Run(name, func(t *testing.T) {
			candidateID, signalID := insertWorldMonitorApprovalBoundaryFixture(t, ctx, pool, withEconomicInput)
			service := NewService(pool)
			eligibility, err := service.ensureApprovalReviewEligible(ctx, candidateID)
			if err != nil || !eligibility.ApprovalEligible {
				t.Fatalf("fixture does not independently satisfy legacy approval eligibility: eligible=%v err=%v reasons=%v", eligibility.ApprovalEligible, err, eligibility.RejectReasons)
			}
			if withEconomicInput {
				if _, err := candidatesmod.NewStore(pool).GetCandidateEconomicInput(ctx, candidateID); err != nil {
					t.Fatalf("load canonical request-level economic input: %v", err)
				}
			}

			before := worldMonitorApprovalBoundaryCounts(t, ctx, pool, candidateID, signalID)
			var statusBefore string
			if err := pool.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, candidateID).Scan(&statusBefore); err != nil {
				t.Fatal(err)
			}
			_, err = service.Decide(ctx, ApprovalRequest{CandidateID: candidateID, Decision: DecisionApproved, ApprovedBy: "boundary-test-operator"})
			if !errors.Is(err, ErrCanonicalPaperHandoffRequired) {
				t.Fatalf("World Monitor approval error=%v, want ErrCanonicalPaperHandoffRequired", err)
			}
			after := worldMonitorApprovalBoundaryCounts(t, ctx, pool, candidateID, signalID)
			if before != after {
				t.Fatalf("approval artifacts changed before=%+v after=%+v", before, after)
			}
			var statusAfter string
			if err := pool.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, candidateID).Scan(&statusAfter); err != nil {
				t.Fatal(err)
			}
			if statusAfter != statusBefore || statusAfter != candidatesmod.StatusAwaitingApproval {
				t.Fatalf("candidate status changed: before=%q after=%q", statusBefore, statusAfter)
			}
		})
	}
}

func TestWorldMonitorNonApprovalDecisionsRemainAvailablePostgres(t *testing.T) {
	ctx := context.Background()
	pool := testApprovalDetailPool(t)
	for _, decision := range []string{DecisionSnoozed, DecisionReanalysisRequested} {
		t.Run(decision, func(t *testing.T) {
			candidateID, signalID := insertWorldMonitorApprovalBoundaryFixture(t, ctx, pool, false)
			approval, err := NewService(pool).Decide(ctx, ApprovalRequest{
				CandidateID: candidateID,
				Decision:    decision,
				ApprovedBy:  "non-economic-operator",
				SnoozeHours: 2,
			})
			if err != nil {
				t.Fatalf("World Monitor non-approval decision %q was blocked: %v", decision, err)
			}
			if approval.Decision != decision || approval.ApprovedBy != "non-economic-operator" {
				t.Fatalf("persisted non-approval=%+v", approval)
			}
			counts := worldMonitorApprovalBoundaryCounts(t, ctx, pool, candidateID, signalID)
			if counts.Approvals != 1 || counts.PaperTickets != 0 || counts.ExecutionInstructions != 0 || counts.TradeApprovals != 0 {
				t.Fatalf("non-approval decision crossed an economic boundary: %+v", counts)
			}
			var candidateStatus string
			if err := pool.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, candidateID).Scan(&candidateStatus); err != nil {
				t.Fatal(err)
			}
			if candidateStatus != candidatesmod.StatusAwaitingApproval {
				t.Fatalf("non-approval candidate status=%q, want awaiting_approval", candidateStatus)
			}
		})
	}
}

type worldMonitorApprovalArtifactCounts struct {
	Approvals             int
	PaperTickets          int
	ExecutionInstructions int
	TradeApprovals        int
}

func worldMonitorApprovalBoundaryCounts(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID, signalID uuid.UUID) worldMonitorApprovalArtifactCounts {
	t.Helper()
	var counts worldMonitorApprovalArtifactCounts
	err := pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1),
			(SELECT COUNT(*) FROM trade_approvals WHERE signal_id=$2)
	`, candidateID, signalID).Scan(&counts.Approvals, &counts.PaperTickets, &counts.ExecutionInstructions, &counts.TradeApprovals)
	if err != nil {
		t.Fatalf("read World Monitor approval-boundary artifact counts: %v", err)
	}
	return counts
}

func insertWorldMonitorApprovalBoundaryFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, withEconomicInput bool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	candidateID, signalID := uuid.New(), uuid.New()
	instanceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := pool.Exec(ctx, `
		INSERT INTO strategy_signals (id,symbol,strategy_id,signal_type,confidence,generated_at,expires_at,status)
		VALUES ($1,'QQQ',$2,'BUY',0.90,$3,$4,'pending')
	`, signalID, "wmab-"+uuid.NewString(), now, now.Add(24*time.Hour)); err != nil {
		t.Fatalf("insert signal fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_trades (
			id,strategy_instance_id,signal_id,symbol,signal_type,status,entry_price,stop_loss,take_profit,
			confidence,reasoning,session_date,source,setup_type,direction,catalyst_summary,
			invalidation_reason,expected_reward_risk_ratio,slippage_allowance,max_normal_loss,
			max_slippage_adjusted_loss,position_size,risk_status,human_approval_required,
			approval_status,gate_status,data_provenance
		) VALUES (
			$1,$2,$3,'QQQ','BUY','awaiting_approval',100,95,110,0.90,'disposable approval fixture',CURRENT_DATE,
			'world-monitor','event-driven','long','identified event catalyst',
			'contradicting evidence invalidates thesis',2,0.01,1,1.1,10,
			'ready_for_approval_review',true,'approval_review_ready','ready_for_risk_review','world-monitor'
		)
	`, candidateID, instanceID, signalID); err != nil {
		t.Fatalf("insert historically risk-ready candidate fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO candidate_evidence_scores (
			candidate_id,support_score,contradiction_score,quality_score,freshness_score,overall_evidence_score,
			evidence_item_count,supporting_item_count,contradictory_item_count,stale_item_count,evidence_status,
			evidence_ready,evidence_gate_ready,broker_execution_allowed,execution_instruction_created,approval_granted
		) VALUES ($1,0.9,0,0.9,1,0.9,2,2,0,0,'sufficient',true,true,false,false,false)
	`, candidateID); err != nil {
		t.Fatalf("insert sufficient legacy evidence fixture: %v", err)
	}
	if withEconomicInput {
		input, err := candidatesmod.BuildCandidateEconomicInput(candidatesmod.CandidateEconomicInput{
			CandidateID: candidateID, InstrumentID: "QQQ", IssuerID: "issuer-disposable-fixture",
			IdentitySource: "disposable-fixture", IdentityPolicyVersion: "test-v1",
			RiskAllocation: 0.01, RequestedLeverage: 1,
			SizingPolicyID: "test-request-policy", SizingPolicyVersion: "test-v1",
		}, now)
		if err != nil {
			t.Fatalf("build explicit request-level economic input: %v", err)
		}
		if err := candidatesmod.NewStore(pool).PersistCandidateEconomicInput(ctx, input); err != nil {
			t.Fatalf("persist explicit request-level economic input: %v", err)
		}
	}
	return candidateID, signalID
}
