package candidates

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScoreEvidenceNoEvidenceIsMissing(t *testing.T) {
	candidate := completeStructuredCandidate()

	score := ScoreEvidenceForCandidate(candidate, nil, time.Now().UTC())

	if score.EvidenceStatus != EvidenceStatusMissing {
		t.Fatalf("evidence status = %q, want %q", score.EvidenceStatus, EvidenceStatusMissing)
	}
	if score.EvidenceReady {
		t.Fatal("candidate with no evidence must not be evidence-ready")
	}
	if score.EvidenceItemCount != 0 {
		t.Fatalf("evidence item count = %d, want 0", score.EvidenceItemCount)
	}
}

func TestScoreEvidenceHighQualityFreshSupportScoresHigherThanWeakEvidence(t *testing.T) {
	candidate := completeStructuredCandidate()
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)

	strong := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		supportingEvidence(candidate.ID, now, 0.90, 0.85, 0.95),
	}, now)
	weak := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		supportingEvidence(candidate.ID, now, 0.35, 0.30, 0.40),
	}, now)

	if strong.OverallEvidenceScore <= weak.OverallEvidenceScore {
		t.Fatalf("strong overall score %.3f must be greater than weak score %.3f", strong.OverallEvidenceScore, weak.OverallEvidenceScore)
	}
	if strong.EvidenceStatus != EvidenceStatusSufficient {
		t.Fatalf("strong evidence status = %q, want %q", strong.EvidenceStatus, EvidenceStatusSufficient)
	}
	if weak.EvidenceStatus != EvidenceStatusWeak {
		t.Fatalf("weak evidence status = %q, want %q", weak.EvidenceStatus, EvidenceStatusWeak)
	}
}

func TestScoreEvidenceContradictionReducesScore(t *testing.T) {
	candidate := completeStructuredCandidate()
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	support := supportingEvidence(candidate.ID, now, 0.85, 0.80, 0.90)

	supportOnly := ScoreEvidenceForCandidate(candidate, []EvidenceItem{support}, now)
	withContradiction := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		support,
		contradictoryEvidence(candidate.ID, now, 0.80, 0.75, 0.85),
	}, now)

	if withContradiction.OverallEvidenceScore >= supportOnly.OverallEvidenceScore {
		t.Fatalf("contradictory evidence score %.3f must be lower than support-only score %.3f", withContradiction.OverallEvidenceScore, supportOnly.OverallEvidenceScore)
	}
	if withContradiction.EvidenceStatus != EvidenceStatusMixed {
		t.Fatalf("evidence status = %q, want %q", withContradiction.EvidenceStatus, EvidenceStatusMixed)
	}
}

func TestScoreEvidenceOnlyContradictoryEvidenceIsNotGateReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)

	score := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		contradictoryEvidence(candidate.ID, now, 0.90, 0.90, 0.95),
	}, now)

	if score.EvidenceReady {
		t.Fatal("only contradictory evidence must not be evidence-ready")
	}
	if score.EvidenceGateReady {
		t.Fatal("only contradictory evidence must not be gate-ready")
	}
	if score.EvidenceStatus != EvidenceStatusBlocked {
		t.Fatalf("evidence status = %q, want %q", score.EvidenceStatus, EvidenceStatusBlocked)
	}
}

func TestScoreEvidenceStaleEvidenceIsStale(t *testing.T) {
	candidate := completeStructuredCandidate()
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	item := supportingEvidence(candidate.ID, now.Add(-49*time.Hour), 0.90, 0.90, 0.90)
	item.FreshnessStatus = FreshnessStatusStale

	score := ScoreEvidenceForCandidate(candidate, []EvidenceItem{item}, now)

	if score.EvidenceStatus != EvidenceStatusStale {
		t.Fatalf("evidence status = %q, want %q", score.EvidenceStatus, EvidenceStatusStale)
	}
	if score.StaleItemCount != 1 {
		t.Fatalf("stale item count = %d, want 1", score.StaleItemCount)
	}
}

func TestScoreEvidenceStructurallyIncompleteCandidateCannotBecomeEvidenceReady(t *testing.T) {
	candidate := completeStructuredCandidate()
	candidate.StopLoss = nil
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)

	score := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		supportingEvidence(candidate.ID, now, 0.95, 0.95, 0.95),
	}, now)

	if score.EvidenceReady {
		t.Fatal("structurally incomplete candidate must not become evidence-ready")
	}
	if score.EvidenceStatus != EvidenceStatusBlocked {
		t.Fatalf("evidence status = %q, want %q", score.EvidenceStatus, EvidenceStatusBlocked)
	}
}

func TestScoreEvidenceDoesNotApproveOrExecuteCandidate(t *testing.T) {
	candidate := completeStructuredCandidate()
	now := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)

	score := ScoreEvidenceForCandidate(candidate, []EvidenceItem{
		supportingEvidence(candidate.ID, now, 0.95, 0.95, 0.95),
	}, now)

	if score.ApprovalGranted {
		t.Fatal("evidence scoring must not approve candidates")
	}
	if score.BrokerExecutionAllowed {
		t.Fatal("evidence scoring must not allow broker execution")
	}
	if score.ExecutionInstructionCreated {
		t.Fatal("evidence scoring must not create execution instructions")
	}
	if candidate.Status == StatusApproved || candidate.Status == StatusSubmitted || candidate.ExecutionInstructionID != nil {
		t.Fatal("evidence scoring must not mutate candidate approval or execution state")
	}
}

func supportingEvidence(candidateID uuid.UUID, observedAt time.Time, confidence, impact, quality float64) EvidenceItem {
	return EvidenceItem{
		EvidenceID:        uuid.New(),
		CandidateID:       candidateID,
		SourceType:        "research",
		SourceRef:         "unit-test-support",
		ObservedAt:        observedAt,
		Summary:           "Fresh supporting evidence.",
		EvidenceKind:      "catalyst",
		SupportsCandidate: true,
		Confidence:        confidence,
		ImpactScore:       impact,
		QualityScore:      quality,
		FreshnessStatus:   FreshnessStatusFresh,
	}
}

func contradictoryEvidence(candidateID uuid.UUID, observedAt time.Time, confidence, impact, quality float64) EvidenceItem {
	item := supportingEvidence(candidateID, observedAt, confidence, impact, quality)
	item.SourceRef = "unit-test-contradiction"
	item.Summary = "Fresh contradictory evidence."
	item.SupportsCandidate = false
	item.ContradictsCandidate = true
	return item
}
