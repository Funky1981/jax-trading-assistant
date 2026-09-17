package exploratorypaper

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPaper02PilotLedgerRestartAndEvidenceIdempotency(t *testing.T) {
	databaseURL := os.Getenv("PAPER02_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("PAPER01B_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("PAPER02_DATABASE_URL/PAPER01B_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	record := activePilotFixture(t)
	record.Identity.PilotID = "pilot-restart-" + time.Now().UTC().Format("20060102150405.000000000")
	record.Status = PilotStatusActive
	identityPayload, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO exploratory_paper_pilots(pilot_id,mode,status,protocol_version,protocol_hash,created_at,payload,formal_evidence_eligible) VALUES($1,$2,$3,$4,$5,$6,$7,FALSE)`, record.Identity.PilotID, PilotMode, record.Status, record.Identity.ProtocolVersion, record.Identity.ProtocolContentHash, record.Identity.CreatedAt, identityPayload); err != nil {
		t.Fatal(err)
	}
	cleanupPool := pool
	defer func() {
		_, _ = cleanupPool.Exec(ctx, `DELETE FROM exploratory_paper_pilot_evidence WHERE pilot_id=$1`, record.Identity.PilotID)
		_, _ = cleanupPool.Exec(ctx, `DELETE FROM exploratory_paper_opportunity_events WHERE opportunity_id IN (SELECT opportunity_id FROM exploratory_paper_opportunities WHERE pilot_id=$1)`, record.Identity.PilotID)
		_, _ = cleanupPool.Exec(ctx, `DELETE FROM exploratory_paper_opportunities WHERE pilot_id=$1`, record.Identity.PilotID)
		_, _ = cleanupPool.Exec(ctx, `DELETE FROM exploratory_paper_pilots WHERE pilot_id=$1`, record.Identity.PilotID)
	}()

	store := NewPilotPostgresStore(pool)
	firstSeen := *record.Identity.StartTimestamp
	opportunity, err := store.RecordOpportunity(ctx, record.Identity.PilotID, OpportunityIntake{EventID: "event-restart", SourceEventIdentity: "source-restart", EventCategory: "issuer-event", FirstSeenAt: firstSeen.Add(time.Minute), IngestionAt: firstSeen.Add(2 * time.Minute), SourceProvenance: []EvidenceReference{{EvidenceID: "e-entry", SourceID: "source-a", SourceURL: "https://example.test/a", Quality: "high", ObservedAt: firstSeen}}, IssuerID: "issuer", InstrumentID: "AAPL", ResolutionState: "RESOLVED", EvidenceState: "sufficient"})
	if err != nil {
		t.Fatal(err)
	}
	decisionAt := firstSeen.Add(3 * time.Minute)
	if _, err := store.RecordDecision(ctx, record.Identity.PilotID, opportunity.OpportunityID, OpportunityDecision{CandidateDecision: DecisionCandidate, DecisionAt: decisionAt, Reason: "sufficient", RiskDecision: "ACCEPTED", HumanDecision: "APPROVED", HumanReviewer: "operator", HumanDecisionAt: &decisionAt, HumanRationale: "paper-only approval", PaperOnlyAcknowledged: true, CandidateID: "candidate-restart", ThesisID: "thesis-restart"}); err != nil {
		t.Fatal(err)
	}
	entryAt := firstSeen.Add(4 * time.Minute)
	if err := store.LinkLifecycle(ctx, record.Identity.PilotID, opportunity.OpportunityID, "lifecycle-restart", "candidate-restart", "thesis-restart", entryAt, []string{"e-entry"}); err != nil {
		t.Fatal(err)
	}
	entryEvidence, err := store.RecordEvidence(ctx, record.Identity.PilotID, opportunity.OpportunityID, EvidenceInput{EvidenceID: "e-entry", FirstSeenAt: firstSeen, IngestionAt: firstSeen.Add(time.Minute), SourceID: "source-a", SourceURL: "https://example.test/a", Relevance: "relevant", Signal: "SUPPORTS", PolicyVersion: "evidence-v1"})
	if err != nil || entryEvidence.Phase != EvidencePhaseEntry {
		t.Fatalf("entry evidence = %+v, err=%v", entryEvidence, err)
	}
	newEvidenceInput := EvidenceInput{EvidenceID: "e-new", FirstSeenAt: entryAt.Add(time.Hour), PublicationAt: timePointer(entryAt.Add(time.Hour)), IngestionAt: entryAt.Add(2 * time.Hour), SourceID: "source-b", SourceURL: "https://example.test/b", Relevance: "relevant", Signal: "INVALIDATES", PolicyVersion: "evidence-v1"}
	newEvidence, err := store.RecordEvidence(ctx, record.Identity.PilotID, opportunity.OpportunityID, newEvidenceInput)
	if err != nil || newEvidence.Phase != EvidencePhaseInvalidating {
		t.Fatalf("new evidence = %+v, err=%v", newEvidence, err)
	}
	duplicate, err := store.RecordEvidence(ctx, record.Identity.PilotID, opportunity.OpportunityID, newEvidenceInput)
	if err != nil || duplicate.Phase != EvidencePhaseInvalidating {
		t.Fatalf("duplicate evidence = %+v, err=%v", duplicate, err)
	}

	pool.Close()
	restartedPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer restartedPool.Close()
	cleanupPool = restartedPool
	restarted := NewPilotPostgresStore(restartedPool)
	restored, err := restarted.GetPilot(ctx, record.Identity.PilotID)
	if err != nil || restored.Status != PilotStatusActive || restored.Identity.ProtocolContentHash != record.Identity.ProtocolContentHash {
		t.Fatalf("restored pilot = %+v, err=%v", restored, err)
	}
	opportunities, err := restarted.ListOpportunities(ctx, record.Identity.PilotID)
	if err != nil || len(opportunities) != 1 || opportunities[0].NewEvidenceCount != 1 || opportunities[0].TradeLifecycleID != "lifecycle-restart" {
		t.Fatalf("restored opportunities = %+v, err=%v", opportunities, err)
	}
}
