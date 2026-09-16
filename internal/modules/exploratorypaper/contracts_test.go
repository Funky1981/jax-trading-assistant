package exploratorypaper

import (
	"testing"
	"time"
)

func fixtureThesis() TradeThesis {
	now := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	return TradeThesis{ThesisID: "thesis-1", ContractVersion: ContractVersion, Mode: ExploratoryPaperMode, EventID: "event-1", IssuerID: "issuer-1", InstrumentID: "AAPL", Direction: DirectionLong, Exposure: "LONG", EventCategory: "earnings", EventTimestamp: now.Add(-time.Hour), CandidateGeneratedAt: now, CausalReason: "reported demand supports a near-term repricing", Evidence: []EvidenceReference{{EvidenceID: "e-1", SourceID: "source-1", SourceURL: "https://example.test/e-1", Quality: "high", ObservedAt: now}}, ExpectedMechanism: "new demand estimate changes next-session expectations", ExpectedHorizonSessions: 3, EntryRationale: "event plus corroborated evidence and confirmation", QuantTechnicalContext: "price above confirmed short-term trend with bounded volatility", RiskAssessment: "one position, one-times leverage, fixed stop", EntryPolicyVersion: "entry-v1", ProtectiveStop: 95, Target: 110, InvalidationConditions: []string{"issuer contradicts reported demand"}, CounterEvidence: []EvidenceReference{{EvidenceID: "e-2", SourceID: "source-2", SourceURL: "https://example.test/e-2", Quality: "medium", ObservedAt: now}}, Confidence: .78, Uncertainty: "reaction and timing remain uncertain", PolicyVersion: "policy-v1", CreatedAt: now}
}

func TestThesisRequiresExploratoryIdentityAndFiveSessionBound(t *testing.T) {
	thesis := fixtureThesis()
	if err := thesis.Validate(); err != nil {
		t.Fatal(err)
	}
	thesis.ExpectedHorizonSessions = 6
	if err := thesis.Validate(); err == nil {
		t.Fatal("accepted horizon beyond five sessions")
	}
	thesis = fixtureThesis()
	thesis.Mode = FormalForwardMode
	if err := thesis.Validate(); err == nil {
		t.Fatal("formal thesis admitted to exploratory contract")
	}
}

func TestCandidateTechnicalOnlyCannotBecomeCandidate(t *testing.T) {
	now := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	got, err := GenerateCandidate(CandidateInput{EventID: "event", IssuerID: "issuer", InstrumentID: "AAPL", EventCategory: "news", EventTimestamp: now, GeneratedAt: now, SourceURL: "https://example.test", TechnicalOnly: true, TechnicalConfirmation: "SMA20>SMA50", Direction: DirectionLong})
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision != DecisionNoTrade {
		t.Fatalf("decision=%s", got.Decision)
	}
}

func TestRelevantEvidenceTransitionsAndIgnoresGenericNews(t *testing.T) {
	now := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	p, err := OpenPosition("position-1", fixtureThesis(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	generic := RelevantEvidence{Reference: EvidenceReference{EvidenceID: "generic", SourceID: "source", SourceURL: "https://example.test/generic", ObservedAt: now}, IssuerID: "other", InstrumentID: "OTHER", Signal: EvidenceContradicts, Reason: "unrelated macro headline"}
	if err := p.Reassess(generic, now.Add(time.Hour), "policy-v1"); err != nil {
		t.Fatal(err)
	}
	if len(p.Transitions) != 0 {
		t.Fatal("irrelevant evidence changed thesis")
	}
	adverse := generic
	adverse.Reference.EvidenceID = "adverse"
	adverse.IssuerID = "issuer-1"
	adverse.InstrumentID = "AAPL"
	adverse.Signal = EvidenceInvalidates
	adverse.Reason = "issuer correction contradicts the mechanism"
	if err := p.Reassess(adverse, now.Add(2*time.Hour), "policy-v1"); err != nil {
		t.Fatal(err)
	}
	if p.State != StateInvalidated || p.OperationalState != StateExitRecommended {
		t.Fatalf("state=%s operational=%s", p.State, p.OperationalState)
	}
}

func TestFiveTradingSessionHardExitSkipsWeekendAndClosedInvalidationDefers(t *testing.T) {
	cal := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-03": false, "2026-01-04": false, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true, "2026-01-09": true}}
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	p, err := OpenPosition("p", fixtureThesis(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	hard, err := cal.HardExitDate(now)
	if err != nil {
		t.Fatal(err)
	}
	if hard.Format("2006-01-02") != "2026-01-08" {
		t.Fatalf("hard exit=%s", hard)
	}
	decision, err := EvaluateExit(p, ExitInput{Now: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), Session: SessionClosed, ThesisInvalidated: true, Calendar: cal})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ExitAtNextTradableSession || decision.NextTradableSession.Format("2006-01-02") != "2026-01-05" {
		t.Fatalf("decision=%#v", decision)
	}
	decision, err = EvaluateExit(p, ExitInput{Now: time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC), Session: SessionOpen, Bid: 100, Calendar: cal})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Reason != ExitTimeLimit {
		t.Fatalf("time exit=%#v", decision)
	}
}

func TestEvidenceFirewall(t *testing.T) {
	if err := ValidateEvidenceAdmission(ExploratoryPaperMode, true, FutureOnlyIdentity{}); err == nil {
		t.Fatal("exploratory evidence admitted to formal request")
	}
	if CanPopulateFormal(ExploratoryPaperMode) {
		t.Fatal("exploratory mode can populate formal")
	}
	if err := RelabelExploratoryAsFormal(); err == nil {
		t.Fatal("relabel succeeded")
	}
	if err := ValidateEvidenceAdmission(FormalForwardMode, false, FutureOnlyIdentity{RunID: "run", StartedAt: time.Now(), FirstObservationAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
}

func TestOutcomeBindsModeHorizonExitAndPolicyVersions(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	versions := PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "risk-v1", EntryPolicy: "entry-v1", ExitPolicy: "exit-v1", CostModel: "paper-cost-v1"}
	outcome := Outcome{OutcomeID: "outcome-1", Mode: ExploratoryPaperMode, TraderModelVersion: TraderModelVersion, ThesisID: "thesis-1", EventID: "event-1", IssuerID: "issuer-1", InstrumentID: "AAPL", EntryAt: now, ExitAt: now.AddDate(0, 0, 2), EntryPrice: 100, ExitPrice: 104, Costs: 1, NetReturn: 39, SessionsHeld: 2, ExitReason: ExitThesisInvalidated, MFE: 6, MAE: -2, PolicyVersions: versions, Checkpoints: []Checkpoint{{Sessions: 1, Price: 102, NetReturn: 19}, {Sessions: 2, Price: 104, NetReturn: 39}}}
	if err := outcome.Validate(); err != nil {
		t.Fatal(err)
	}
	outcome.Mode = FormalForwardMode
	if err := outcome.Validate(); err == nil {
		t.Fatal("formal outcome accepted by exploratory contract")
	}
}

func TestEntryBindingRequiresExistingPaperWorkflowSafetyIdentity(t *testing.T) {
	versions := PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "risk-v1", EntryPolicy: "entry-v1", ExitPolicy: "exit-v1", CostModel: "paper-cost-v1"}
	binding := EntryBinding{Mode: ExploratoryPaperMode, CandidateID: "candidate-1", ThesisID: "thesis-1", TraderModelVersion: TraderModelVersion, WorkflowID: "workflow-1", PaperIntentID: "paper-intent-1", Environment: "PAPER", ExecutionAuthority: "NONE", MaximumLeverage: 1, PolicyVersions: versions, BoundAt: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)}
	if err := binding.Validate(); err != nil {
		t.Fatal(err)
	}
	binding.Environment = "LIVE"
	if err := binding.Validate(); err == nil {
		t.Fatal("live binding accepted")
	}
}
