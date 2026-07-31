package eventdecisions

import (
	"encoding/json"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/instruments"

	"github.com/google/uuid"
)

func TestDecisionContractOutcomesAndBoundaries(t *testing.T) {
	evaluator := testEvaluator(t)

	t.Run("genuine eligible event can produce no trade", func(t *testing.T) {
		event := testEvent()
		event.Severity = "low"
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionNoTrade, false)
	})

	t.Run("genuine eligible event with unknown assets can produce watch", func(t *testing.T) {
		event := testEvent()
		event.AffectedAssets = nil
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionWatch, false)
		if !result.UnknownAssets || !contains(result.MissingEvidence, "truthful_asset_mapping") {
			t.Fatalf("unknown asset state not explicit: %+v", result)
		}
	})

	t.Run("weak unknown asset event produces no trade", func(t *testing.T) {
		event := testEvent()
		event.AffectedAssets = nil
		event.Confidence = 0.49
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionNoTrade, false)
	})

	t.Run("complete existing candidate produces candidate at evidence threshold", func(t *testing.T) {
		event := candidateEvent(0.60)
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionCandidate, true)
		if result.TrustGateState != candidates.GateStatusReadyForRiskReview || result.RiskReviewState != string(candidates.RiskStatusReadyForApprovalReview) {
			t.Fatalf("gate states not persisted in result: %+v", result)
		}
	})

	t.Run("below candidate threshold remains watch", func(t *testing.T) {
		event := candidateEvent(0.599)
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionWatch, false)
	})

	t.Run("missing candidate field prevents candidate", func(t *testing.T) {
		event := candidateEvent(0.80)
		event.Candidate.StopLoss = nil
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionWatch, false)
		if !contains(result.MissingEvidence, "stop_loss_price") {
			t.Fatalf("missing stop was not explained: %+v", result)
		}
	})

	t.Run("failed trust gate prevents candidate", func(t *testing.T) {
		event := candidateEvent(0.80)
		event.Candidate.GateStatus = candidates.GateStatusEvidenceWeak
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionWatch, false)
	})

	t.Run("failed risk review prevents candidate", func(t *testing.T) {
		event := candidateEvent(0.80)
		event.Candidate.RiskStatus = string(candidates.RiskStatusRiskTooHigh)
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionWatch, false)
	})

	t.Run("unsafe product fails closed", func(t *testing.T) {
		event := candidateEvent(0.80)
		event.Candidate.Symbol = "TQQQ"
		event.Candidate.InstrumentType = "etf"
		event.AffectedAssets = []string{"TQQQ"}
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionNoTrade, false)
	})

	t.Run("leverage above one fails closed", func(t *testing.T) {
		event := candidateEvent(0.80)
		metadata := json.RawMessage(`{"requested_leverage":1.5}`)
		event.Candidate.Metadata = &metadata
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionNoTrade, false)
		if !contains(result.BlockingReasons, "leverage_above_1x") {
			t.Fatalf("leverage block not explained: %+v", result)
		}
	})

	t.Run("expired legacy candidate is no trade and is not linked", func(t *testing.T) {
		event := candidateEvent(0.80)
		event.Candidate.Status = candidates.StatusExpired
		result := mustEvaluate(t, evaluator, event)
		assertDecision(t, result, DecisionNoTrade, false)
	})
}

func TestEligibilityExcludesSyntheticRejectedAndMissingProvenance(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Event)
		want   string
	}{
		{name: "synthetic", mutate: func(e *Event) { e.IsSynthetic = true }, want: "synthetic_or_fixture_event"},
		{name: "rejected", mutate: func(e *Event) { e.Status = "rejected" }, want: "rejected_event"},
		{name: "missing provenance", mutate: func(e *Event) { e.ProvenanceAvailable = false }, want: "missing_provenance"},
		{name: "missing normalization", mutate: func(e *Event) { e.NormalizedEventID = nil }, want: "missing_normalized_event"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event := testEvent()
			tc.mutate(&event)
			ok, reason := Eligible(event)
			if ok || reason != tc.want {
				t.Fatalf("Eligible()=(%v,%q), want false,%q", ok, reason, tc.want)
			}
		})
	}
}

func TestDryRunReturnsDecisionsAndRequiresNoStore(t *testing.T) {
	events := []Event{testEvent(), testEvent()}
	events[0].InboxID = uuid.New()
	events[0].Severity = "low"
	events[1].InboxID = uuid.New()
	events[1].AffectedAssets = nil
	replayer := Replayer{Evaluator: testEvaluator(t), Now: func() time.Time { return testNow() }}
	summary, err := replayer.Run(t.Context(), events, true)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !summary.DryRun || summary.NoTrade != 1 || summary.Watch != 1 || summary.DecisionsCreated != 0 || summary.CandidatesCreated != 0 {
		t.Fatalf("unexpected dry-run summary: %+v", summary)
	}
}

func TestReplayIdentityIsStableAndRulesetSensitive(t *testing.T) {
	event := testEvent()
	fingerprint1, replay1, err := replayIdentity(event, "v1")
	if err != nil {
		t.Fatal(err)
	}
	fingerprint2, replay2, _ := replayIdentity(event, "v1")
	if fingerprint1 != fingerprint2 || replay1 != replay2 {
		t.Fatal("identical replay input was not stable")
	}
	_, replay3, _ := replayIdentity(event, "v2")
	if replay3 == replay1 {
		t.Fatal("ruleset version did not change replay identity")
	}
}

func TestSafetyStateFailsClosed(t *testing.T) {
	safe := map[string]string{
		"JAX_RUNTIME_MODE":                     "paper",
		"ALLOW_LIVE_TRADING":                   "false",
		"EXECUTION_ENABLED":                    "false",
		"EXECUTION_INSTRUCTION_WORKER_ENABLED": "false",
		"BROKER_EXECUTION_ALLOWED":             "false",
		"MAX_LEVERAGE":                         "1",
	}
	lookup := func(key string) (string, bool) { value, ok := safe[key]; return value, ok }
	if _, err := ReadSafetyState(lookup); err != nil {
		t.Fatalf("safe state rejected: %v", err)
	}
	delete(safe, "MAX_LEVERAGE")
	if _, err := ReadSafetyState(lookup); err == nil {
		t.Fatal("missing safety state was accepted")
	}
	safe["MAX_LEVERAGE"] = "1.01"
	if _, err := ReadSafetyState(lookup); err == nil {
		t.Fatal("leverage above 1x was accepted")
	}
}

func testEvaluator(t *testing.T) Evaluator {
	t.Helper()
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		t.Fatalf("load instrument catalog: %v", err)
	}
	return Evaluator{Ruleset: Ruleset{
		Version: "genuine-event-decision-v1", ProcessorIdentity: "test-processor",
		WatchConfidenceMinimum: 0.5, CandidateEvidenceMinimum: 0.6,
		SubjectRulesetVersion: "genuine-watch-evidence-v1", SubjectCandidateIndependentMin: 2, SubjectFreshnessHours: 24,
		AllowedCandidateInstrumentType: "etf", MaximumLeverage: 1,
		MaterialSeverities: []string{"medium", "high", "critical"},
	}, Catalog: catalog}
}

func testEvent() Event {
	normalizedID := uuid.New()
	return Event{
		InboxID: uuid.New(), NormalizedEventID: &normalizedID, Source: "world-monitor", SourceEventID: "genuine-event-1",
		Status: "new", EventType: "energy_oil", Headline: "Material event", Summary: "Persisted summary",
		SourceURLs: []string{"https://example.com/event"}, SourceCount: 1, PublicationAt: testNow().Add(-time.Hour),
		ReceiptAt: testNow(), Severity: "medium", SourceTier: "tier1", Confidence: 0.5,
		AffectedAssets: []string{"QQQ"}, MappingReason: "provider supplied", MappingMethods: []string{"provider_symbol"},
		ProvenanceAvailable: true, IsSynthetic: false, DataSourceType: "real", SourceProvider: "world-monitor",
	}
}

func candidateEvent(score float64) Event {
	event := testEvent()
	entry, stop, target := 500.0, 490.0, 520.0
	event.Candidate = &candidates.Candidate{
		ID: uuid.New(), Symbol: "QQQ", SignalType: "BUY", Status: candidates.StatusAwaitingApproval,
		Source: "world-monitor", InstrumentType: "etf", SetupType: "sector_news_momentum", Direction: "long",
		CatalystSummary: "Persisted material catalyst", EntryPrice: &entry, StopLoss: &stop, TakeProfit: &target,
		InvalidationReason: "QQQ at or below the stop invalidates the setup", RiskStatus: string(candidates.RiskStatusReadyForApprovalReview),
		GateStatus: candidates.GateStatusReadyForRiskReview, HumanApprovalRequired: true, DataProvenance: "world-monitor",
	}
	event.CandidateEvidenceScore = &candidates.EvidenceScoreSummary{
		CandidateID: event.Candidate.ID, OverallEvidenceScore: score, QualityScore: 0.8, FreshnessScore: 1,
		EvidenceItemCount: 1, SupportingItemCount: 1, EvidenceStatus: candidates.EvidenceStatusSufficient,
		EvidenceReady: score >= 0.6, EvidenceGateReady: score >= 0.6,
	}
	return event
}

func mustEvaluate(t *testing.T, evaluator Evaluator, event Event) Result {
	t.Helper()
	result, err := evaluator.Evaluate(event)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	return result
}

func assertDecision(t *testing.T, result Result, want Decision, candidateID bool) {
	t.Helper()
	if result.Decision != want {
		t.Fatalf("decision=%s, want %s: %+v", result.Decision, want, result)
	}
	if (result.CandidateID != nil) != candidateID {
		t.Fatalf("candidate link presence=%v, want %v: %+v", result.CandidateID != nil, candidateID, result)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func testNow() time.Time {
	return time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
}
