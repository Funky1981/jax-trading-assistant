package candidates

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPaperTicketReviewQueueReturnsPersistedTicketsAndOmitsExecutionControlFields(t *testing.T) {
	ctx := context.Background()
	pool := testCandidatePaperTicketPool(t)
	store := NewStore(pool)
	ticket := createPersistedPaperTicketForReview(t, ctx, store)

	queue, err := store.ListPaperTicketReviews(ctx, 10)
	if err != nil {
		t.Fatalf("list paper ticket reviews: %v", err)
	}
	if len(queue) == 0 {
		t.Fatal("paper ticket review queue returned no persisted tickets")
	}

	var found *PaperTicketReview
	for i := range queue {
		if queue[i].PaperTicketID == ticket.PaperTicketID {
			found = &queue[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("paper ticket %q not found in review queue: %+v", ticket.PaperTicketID, queue)
	}
	if !found.PaperOnly || found.Symbol != ticket.Symbol || found.Status != PaperTicketStatusPaperTicketCreated {
		t.Fatalf("unexpected review queue item: %+v", found)
	}

	payload, err := json.Marshal(found)
	if err != nil {
		t.Fatalf("marshal review queue item: %v", err)
	}
	for _, key := range []string{"brokerExecutionAllowed", "executionInstructionCreated", "liveTradingAllowed", "leverageAllowed", "executionReady", "autoExecutionEnabled"} {
		if jsonContainsKey(payload, key) {
			t.Fatalf("review queue item exposed forbidden key %q: %s", key, string(payload))
		}
	}
}

func TestPaperTicketReviewActionsOnlyChangeReviewStatusAndNeverCreateExecution(t *testing.T) {
	ctx := context.Background()
	pool := testCandidatePaperTicketPool(t)
	store := NewStore(pool)
	ticket := createPersistedPaperTicketForReview(t, ctx, store)

	reviewed, err := store.MarkPaperTicketReviewed(ctx, ticket.PaperTicketID, "checked by reviewer")
	if err != nil {
		t.Fatalf("mark reviewed: %v", err)
	}
	if reviewed.Status != PaperTicketStatusPaperTicketReviewed {
		t.Fatalf("reviewed status = %q, want %q", reviewed.Status, PaperTicketStatusPaperTicketReviewed)
	}
	assertPaperTicketSafetyFlags(t, ctx, pool, ticket.CandidateID)
	assertNoExecutionInstruction(t, ctx, pool, ticket.CandidateID)

	cancelledTicket := createPersistedPaperTicketForReview(t, ctx, store)
	cancelled, err := store.CancelPaperTicketReview(ctx, cancelledTicket.PaperTicketID, "paper review no longer needed")
	if err != nil {
		t.Fatalf("cancel paper ticket: %v", err)
	}
	if cancelled.Status != PaperTicketStatusPaperTicketCancelled {
		t.Fatalf("cancelled status = %q, want %q", cancelled.Status, PaperTicketStatusPaperTicketCancelled)
	}
	assertPaperTicketSafetyFlags(t, ctx, pool, cancelledTicket.CandidateID)
	assertNoExecutionInstruction(t, ctx, pool, cancelledTicket.CandidateID)

	if _, err := store.MarkPaperTicketReviewed(ctx, cancelledTicket.PaperTicketID, "should remain cancelled"); err == nil {
		t.Fatal("cancelled paper ticket must not become reviewed or execution-ready")
	}
}

func TestPaperTicketReviewAddNoteDoesNotExposeOrCreateExecutionControls(t *testing.T) {
	ctx := context.Background()
	pool := testCandidatePaperTicketPool(t)
	store := NewStore(pool)
	ticket := createPersistedPaperTicketForReview(t, ctx, store)

	review, err := store.AddPaperTicketReviewNote(ctx, ticket.PaperTicketID, "needs tomorrow morning review")
	if err != nil {
		t.Fatalf("add review note: %v", err)
	}
	if review.Status != PaperTicketStatusPaperTicketCreated {
		t.Fatalf("add note changed status = %q, want %q", review.Status, PaperTicketStatusPaperTicketCreated)
	}
	payload, err := json.Marshal(review)
	if err != nil {
		t.Fatalf("marshal review after note: %v", err)
	}
	if jsonContainsKey(payload, "reviewNotes") {
		t.Fatalf("safe review model exposed review notes unexpectedly: %s", string(payload))
	}
	assertPaperTicketSafetyFlags(t, ctx, pool, ticket.CandidateID)
	assertNoExecutionInstruction(t, ctx, pool, ticket.CandidateID)
}

func testCandidatePaperTicketPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip DB-backed paper ticket test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skip DB-backed paper ticket test: %v", err)
	}
	if _, err := pool.Exec(ctx, `ALTER TABLE candidate_paper_tickets ADD COLUMN IF NOT EXISTS review_notes TEXT`); err != nil {
		pool.Close()
		t.Skipf("skip DB-backed paper ticket test; schema not ready: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION append_review_note(existing TEXT, new_note TEXT)
		RETURNS TEXT
		LANGUAGE SQL
		IMMUTABLE
		AS $$
			SELECT CASE
				WHEN NULLIF(BTRIM(COALESCE(new_note, '')), '') IS NULL THEN existing
				WHEN NULLIF(BTRIM(COALESCE(existing, '')), '') IS NULL THEN BTRIM(new_note)
				ELSE existing || E'\n' || BTRIM(new_note)
			END
		$$
	`); err != nil {
		pool.Close()
		t.Skipf("skip DB-backed paper ticket test; note helper not ready: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func createPersistedPaperTicketForReview(t *testing.T, ctx context.Context, store *Store) PaperTicket {
	t.Helper()

	candidate := approvalReadyCandidate()
	candidate.Symbol = "SPY" + strings.ReplaceAll(uuid.NewString()[:4], "-", "")
	candidate.Status = StatusAwaitingApproval
	if _, err := store.Create(ctx, &candidate); err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.pool.Exec(context.Background(), `DELETE FROM candidate_paper_tickets WHERE candidate_id = $1`, candidate.ID)
		_, _ = store.pool.Exec(context.Background(), `DELETE FROM execution_instructions WHERE candidate_id = $1`, candidate.ID)
		_, _ = store.pool.Exec(context.Background(), `DELETE FROM candidate_trades WHERE id = $1`, candidate.ID)
	})

	eligibility := EvaluateApprovalEligibility(candidate, sufficientEvidenceScore(candidate.ID), readyRiskGate(candidate.ID), readyRiskReview(candidate), fixedApprovalReviewTime())
	result := CreatePaperTicket(PaperTicketRequest{
		Candidate:            candidate,
		Eligibility:          eligibility,
		HumanApprovalGranted: true,
		ApprovalDecisionRef:  uuid.NewString(),
		SourceApprovalID:     uuid.New(),
		CreatedAt:            fixedApprovalReviewTime(),
	})
	ticket, err := NewPersistedPaperTicket(candidate, sufficientEvidenceScore(candidate.ID), eligibility, result)
	if err != nil {
		t.Fatalf("new persisted paper ticket: %v", err)
	}
	persisted, err := store.CreatePaperTicket(ctx, ticket)
	if err != nil {
		t.Fatalf("persist paper ticket: %v", err)
	}
	return *persisted
}

func assertPaperTicketSafetyFlags(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID uuid.UUID) {
	t.Helper()
	var paperOnly, brokerAllowed, instructionCreated, liveAllowed, leverageAllowed bool
	if err := pool.QueryRow(ctx, `
		SELECT paper_only, broker_execution_allowed, execution_instruction_created, live_trading_allowed, leverage_allowed
		FROM candidate_paper_tickets
		WHERE candidate_id = $1
	`, candidateID).Scan(&paperOnly, &brokerAllowed, &instructionCreated, &liveAllowed, &leverageAllowed); err != nil {
		t.Fatalf("query paper ticket safety flags: %v", err)
	}
	if !paperOnly || brokerAllowed || instructionCreated || liveAllowed || leverageAllowed {
		t.Fatalf("unsafe paper ticket flags: paperOnly=%v broker=%v instruction=%v live=%v leverage=%v", paperOnly, brokerAllowed, instructionCreated, liveAllowed, leverageAllowed)
	}
}

func assertNoExecutionInstruction(t *testing.T, ctx context.Context, pool *pgxpool.Pool, candidateID uuid.UUID) {
	t.Helper()
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM execution_instructions WHERE candidate_id = $1`, candidateID).Scan(&count); err != nil {
		t.Fatalf("query execution instructions: %v", err)
	}
	if count != 0 {
		t.Fatalf("execution instructions = %d, want 0", count)
	}
}
