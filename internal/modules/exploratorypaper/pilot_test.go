package exploratorypaper

import (
	"strings"
	"testing"
	"time"
)

func pilotTestProtocol() PilotProtocol {
	return PilotProtocol{
		ProtocolVersion: "paper-02-protocol-v1", Purpose: "prospective operational learning", ExploratoryStatus: "EXPLORATORY_ONLY", StartCondition: "external activation after readiness review", EndCondition: "50 eligible opportunities, 90 days, or abort", EligibleEventUniverse: []string{"source-backed issuer-relevant events"}, EligibleInstrumentUniverse: []string{"resolved liquid instruments"}, DataEvidenceRequirements: []string{"canonical reviewed source-backed evidence"}, CandidateGate: []string{"event catalyst", "two independent sources", "no unresolved contradiction"}, HumanApprovalRequirements: []string{"explicit human PAPER approval"}, RiskBoundaries: []string{"existing deterministic risk veto", "maximum leverage 1x"}, SimulatedExecutionAssumptions: []string{"isolated paper venue", "costs, spread, slippage, latency"}, MaximumLeverage: 1, PositionSizing: "existing configured paper-risk sizing", ConcurrentPositionPolicy: "existing configured exploratory concurrency limit", EntryRules: []string{"CANDIDATE only", "human approval required"}, ReviewCadence: "each known trading session", ExitRules: []string{"STOP", "TARGET", "THESIS_INVALIDATED", "RISK_KILL", "TIME_LIMIT", "MANUAL_OPERATOR"}, FiveSessionRule: "hard time-limit is actionable at opening of session five", MissingDataTreatment: "explicit MISSING_DATA; never fabricate", IncidentHandling: "pause admission and preserve records", OperatorResponsibilities: []string{"review evidence", "approve paper actions", "surface incidents"}, RequiredAuditFields: []string{"identity", "provenance", "timestamps", "versions"}, PilotMetrics: []string{"opportunity classification", "latency", "missing data", "descriptive outcome"}, ProhibitedInterpretations: []string{"no profitability claim", "no demonstrated edge", "not formal evidence"}, StopAbortConditions: []string{"broker path", "persistence conflict", "accounting inconsistency", "calendar failure"}, PostPilotReview: []string{"selection bias", "luck", "leakage", "operational bias", "formal design inputs"}, TargetOpportunityCount: 50, MaximumDurationDays: 90, EligibleUniverseVersion: "event-universe-v1"}
}

func pilotTestVersions() PolicyVersions {
	return PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "risk-v1", EntryPolicy: "entry-v1", ExitPolicy: "exit-v1", CostModel: "paper-cost-v1"}
}

func activePilotFixture(t *testing.T) PilotRecord {
	t.Helper()
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	record, err := NewPilotDraft("pilot-test-1", pilotTestProtocol(), now, pilotTestVersions(), "paper-cost-v1", "calendar-test-v1", "test-code-sha")
	if err != nil {
		t.Fatal(err)
	}
	record, err = MarkPilotReady(record)
	if err != nil {
		t.Fatal(err)
	}
	record, err = ActivatePilotForExternalReview(record, ExternalActivationAuthorization{Reviewer: "external-reviewer", Decision: "ACTIVATE_PAPER_02", Acknowledged: "EXPLORATORY_PAPER_ONLY_NO_LIVE_NO_FORMAL", AuthorizedAt: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func TestPilotProtocolIdentityRequiresExternalActivation(t *testing.T) {
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	protocol := pilotTestProtocol()
	record, err := NewPilotDraft("pilot-test-1", protocol, now, pilotTestVersions(), "paper-cost-v1", "calendar-test-v1", "test-code-sha")
	if err != nil || record.Status != PilotStatusDraft {
		t.Fatalf("draft = %+v, err=%v", record, err)
	}
	ready, err := MarkPilotReady(record)
	if err != nil || ready.Status != PilotStatusReadyForExternalReview {
		t.Fatalf("ready = %+v, err=%v", ready, err)
	}
	if _, err := ActivatePilotForExternalReview(ready, ExternalActivationAuthorization{Reviewer: "", Decision: "ACTIVATE_PAPER_02"}); err == nil {
		t.Fatal("incomplete external authorization activated pilot")
	}
	active := activePilotFixture(t)
	active.Protocol.ProtocolVersion = "tampered"
	if err := active.Identity.Validate(active.Protocol); err == nil {
		t.Fatal("tampered protocol identity was accepted")
	}
	if active.Identity.Mode != PilotMode || active.Status != PilotStatusActive {
		t.Fatal("fixture lost exploratory identity")
	}
}

func TestPilotOpportunityBoundaryRetainsNonTradesAndMetrics(t *testing.T) {
	pilot := activePilotFixture(t)
	start := *pilot.Identity.StartTimestamp
	intake := PilotOpportunity{PilotID: pilot.Identity.PilotID, OpportunityID: "op-retro", EventID: "event", SourceEventIdentity: "source-event", EventCategory: "issuer-event", FirstSeenAt: start.Add(-time.Minute), IngestionAt: start, ResolutionState: "RESOLVED", EvidenceState: "sufficient", FinalClassification: PilotClassificationUnresolved}
	if err := intake.ValidateForPilot(pilot); err == nil {
		t.Fatal("retrospective opportunity was admitted")
	}
	classes := []OpportunityClassification{PilotClassificationApprovedExploratoryTrade, PilotClassificationWatch, PilotClassificationNoTrade, PilotClassificationUnresolved, PilotClassificationRejectedEvidence, PilotClassificationRejectedRisk, PilotClassificationRejectedHuman}
	opportunities := make([]PilotOpportunity, 0, len(classes))
	for i, class := range classes {
		opportunity := PilotOpportunity{PilotID: pilot.Identity.PilotID, OpportunityID: "op-" + string(rune('a'+i)), EventID: "event-" + string(rune('a'+i)), SourceEventIdentity: "source-" + string(rune('a'+i)), EventCategory: "issuer-event", FirstSeenAt: start.Add(time.Duration(i+1) * time.Minute), IngestionAt: start.Add(time.Duration(i+1) * time.Minute), IssuerID: "issuer", InstrumentID: "AAPL", ResolutionState: "RESOLVED", EvidenceState: "sufficient", FinalClassification: class}
		if class == PilotClassificationUnresolved {
			opportunity.IssuerID, opportunity.InstrumentID, opportunity.ResolutionState = "", "", "UNRESOLVED"
		}
		opportunities = append(opportunities, opportunity)
	}
	metrics := ComputePilotMetrics(opportunities)
	if metrics.EligibleOpportunities != len(classes) || metrics.WatchRate == 0 || metrics.NoTradeRate == 0 || metrics.UnresolvedRate == 0 || metrics.RiskRejectionRate == 0 || metrics.HumanRejectionRate == 0 || metrics.ExploratoryEntryRate == 0 {
		t.Fatalf("non-trade classifications were not retained in metrics: %+v", metrics)
	}
	if metrics.FormalEvidenceEligible || metrics.DemonstratedEdge {
		t.Fatal("exploratory metrics crossed the formal/edge firewall")
	}
	ids := map[string]struct{}{stableOpportunityID(pilot.Identity.PilotID, "same-event"): {}}
	ids[stableOpportunityID(pilot.Identity.PilotID, "same-event")] = struct{}{}
	if len(ids) != 1 {
		t.Fatal("opportunity identity is not idempotent")
	}
}

func TestPilotEvidencePhasesAndUnknownLatency(t *testing.T) {
	pilot := activePilotFixture(t)
	entry := pilot.Identity.StartTimestamp.Add(time.Hour)
	entryEvidence := PilotOpportunity{PilotID: pilot.Identity.PilotID, OpportunityID: "op-1", EntryAt: &entry, EntryEvidenceIDs: []string{"e-entry"}, FinalClassification: PilotClassificationApprovedExploratoryTrade}
	publication := entry.Add(time.Hour)
	cases := []struct {
		name, id, relevance, signal, want string
		first                             time.Time
		pub                               *time.Time
	}{
		{"entry replay", "e-entry", "relevant", "SUPPORTS", EvidencePhaseEntry, entry.Add(2 * time.Hour), &publication},
		{"new support", "e-support", "relevant", "SUPPORTS", EvidencePhaseSupporting, entry.Add(2 * time.Hour), &publication},
		{"new contradiction", "e-contradiction", "relevant", "CONTRADICTS", EvidencePhaseContradictory, entry.Add(3 * time.Hour), &publication},
		{"new invalidation", "e-invalidation", "relevant", "INVALIDATES", EvidencePhaseInvalidating, entry.Add(4 * time.Hour), &publication},
		{"irrelevant", "e-irrelevant", "irrelevant", "SUPPORTS", EvidencePhaseNoNewRelevant, entry.Add(5 * time.Hour), &publication},
		{"unknown signal", "e-unknown", "relevant", "UNKNOWN", EvidencePhaseMissing, entry.Add(6 * time.Hour), &publication},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			phase, err := classifyPilotEvidence(entryEvidence, EvidenceInput{EvidenceID: tc.id, FirstSeenAt: tc.first, PublicationAt: tc.pub, IngestionAt: tc.first.Add(time.Minute), SourceID: "source", SourceURL: "https://example.test/evidence", Relevance: tc.relevance, Signal: tc.signal, PolicyVersion: "evidence-v1"})
			if err != nil || phase != tc.want {
				t.Fatalf("phase=%s err=%v want=%s", phase, err, tc.want)
			}
		})
	}
	unknown := BuildLatencyChain(nil, entry, entry, entry, entry, entry)
	if unknown.PublicationToIngestion.Known || !strings.Contains(unknown.PublicationToIngestion.UnknownReason, "unavailable") {
		t.Fatalf("publication latency was fabricated: %+v", unknown.PublicationToIngestion)
	}
	known := buildEvidenceLatency(&publication, publication.Add(time.Minute))
	if !known.Known || known.DurationMS == nil || *known.DurationMS != 60000 {
		t.Fatalf("known evidence latency incorrect: %+v", known)
	}
}

func TestPilotMarketQualityAndDescriptiveAccounting(t *testing.T) {
	missing := MarketObservationQuality{ObservationID: "missing", Kind: PilotObservationKindReview, CostModelVersion: "paper-cost-v1", Missing: true, MissingReason: "provider unavailable"}
	if err := missing.Validate(); err != nil {
		t.Fatal(err)
	}
	stale := missing
	stale.ObservationID, stale.Missing, stale.MissingReason, stale.Source, stale.ObservedAt = "stale", false, "", "feed", time.Now().UTC()
	stale.PriceUsed, stale.Bid, stale.Ask, stale.Spread, stale.BidAvailable, stale.AskAvailable, stale.Stale = 100, 99, 101, 2, true, true, true
	if err := stale.Validate(); err == nil {
		t.Fatal("stale market data accepted")
	}
	valid := stale
	valid.ObservationID, valid.Stale = "valid", false
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	pilot := activePilotFixture(t)
	start := *pilot.Identity.StartTimestamp
	opportunities := []PilotOpportunity{
		{PilotID: pilot.Identity.PilotID, OpportunityID: "trade", EventID: "e", SourceEventIdentity: "s", EventCategory: "event", FirstSeenAt: start, IngestionAt: start, IssuerID: "issuer", InstrumentID: "AAPL", ResolutionState: "RESOLVED", EvidenceState: "sufficient", FinalClassification: PilotClassificationApprovedExploratoryTrade, TradeLifecycleID: "lifecycle", Outcome: &PilotOutcomeSummary{OutcomeID: "outcome", GrossPnL: 10, NetPnL: 8, NetReturn: .008, ExitReason: "TARGET", CostModelVersion: "paper-cost-v1"}},
		{PilotID: pilot.Identity.PilotID, OpportunityID: "watch", EventID: "e2", SourceEventIdentity: "s2", EventCategory: "event", FirstSeenAt: start, IngestionAt: start, ResolutionState: "UNRESOLVED", EvidenceState: "missing", FinalClassification: PilotClassificationWatch},
	}
	metrics := ComputePilotMetrics(opportunities)
	if metrics.EligibleOpportunities != 2 || metrics.DescriptiveNetPnL != 8 || metrics.DescriptiveOutcomeCount != 1 || metrics.FormalEvidenceEligible || metrics.DemonstratedEdge {
		t.Fatalf("descriptive metrics or firewall incorrect: %+v", metrics)
	}
}
