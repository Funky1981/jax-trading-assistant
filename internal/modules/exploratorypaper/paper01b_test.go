package exploratorypaper

import (
	"testing"
	"time"
)

func reviewedCandidateInput(now time.Time) CandidateInput {
	evidence := []EvidenceReference{{EvidenceID: "e-1", SourceID: "source-1", SourceURL: "https://example.test/e-1", Quality: "high", ObservedAt: now}, {EvidenceID: "e-3", SourceID: "source-3", SourceURL: "https://example.test/e-3", Quality: "high", ObservedAt: now}}
	return CandidateInput{
		EventID: "event-1", IssuerID: "issuer-1", InstrumentID: "AAPL", EventCategory: "news",
		EventTimestamp: now.Add(-time.Hour), GeneratedAt: now, SourceURL: evidence[0].SourceURL, Evidence: evidence,
		ReviewedEvidence:       &EvidenceAssessment{Provider: "candidate_evidence_scores", PolicyVersion: "candidate-v1", EvidenceSetFingerprint: EvidenceSetFingerprint(evidence), ReviewedAt: now, SourceBacked: true, QualityState: "sufficient", QualityScore: .9, RequiredQualityScore: .7, EvidenceReady: true, EvidenceGateReady: true, Corroborated: true, IssuerRelevant: true, InstrumentRelevant: true, IndependentSourceGroups: 2},
		CandidatePolicyVersion: "candidate-v1", CausalMechanism: "event changes near-term expectations", QuantContext: "bounded volatility", RiskAssessment: "one-times leverage", TechnicalConfirmation: "support confirmed", Direction: DirectionLong,
	}
}

func TestCandidateEligibilityUsesReviewedEvidenceFacts(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*CandidateInput)
		want   CandidateDecision
	}{
		{"valid canonical review", func(*CandidateInput) {}, DecisionCandidate},
		{"legacy booleans cannot fake review", func(in *CandidateInput) { in.ReviewedEvidence = nil; in.EvidenceQuality = 1; in.Corroborated = true }, DecisionWatch},
		{"unknown quality", func(in *CandidateInput) { in.ReviewedEvidence.Unknown = true }, DecisionWatch},
		{"insufficient quality", func(in *CandidateInput) { in.ReviewedEvidence.QualityState = "insufficient" }, DecisionWatch},
		{"single source group", func(in *CandidateInput) { in.ReviewedEvidence.IndependentSourceGroups = 1 }, DecisionWatch},
		{"contradiction preserved", func(in *CandidateInput) { in.ReviewedEvidence.Contradictory = true }, DecisionWatch},
		{"fingerprint mismatch", func(in *CandidateInput) { in.ReviewedEvidence.EvidenceSetFingerprint = "sha256:wrong" }, DecisionWatch},
		{"issuer relevance required", func(in *CandidateInput) { in.ReviewedEvidence.IssuerRelevant = false }, DecisionWatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := reviewedCandidateInput(now)
			test.mutate(&in)
			got, err := GenerateCandidate(in)
			if err != nil || got.Decision != test.want {
				t.Fatalf("decision=%#v err=%v", got, err)
			}
		})
	}
}

func TestFrozenThesisAndApprovalIdentityDetectTampering(t *testing.T) {
	thesis := fixtureThesis()
	binding := fixtureBinding(thesis)
	position, err := OpenApprovedPosition("position-frozen", thesis, binding, thesis.CreatedAt, 100)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		mutate func(*Position, *EntryBinding)
	}{
		{"causal reason", func(p *Position, _ *EntryBinding) { p.Thesis.CausalReason = "changed" }},
		{"evidence order", func(p *Position, _ *EntryBinding) {
			p.Thesis.Evidence[0], p.Thesis.CounterEvidence[0] = p.Thesis.CounterEvidence[0], p.Thesis.Evidence[0]
		}},
		{"stop", func(p *Position, _ *EntryBinding) { p.Thesis.ProtectiveStop = 90 }},
		{"binding workflow", func(_ *Position, b *EntryBinding) { b.WorkflowID = "other-workflow" }},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			p, b := position, binding
			test.mutate(&p, &b)
			if test.name == "binding workflow" {
				if err := p.VerifyApprovalBinding(b); err == nil {
					t.Fatal("tampered approval identity accepted")
				}
				return
			}
			if err := p.VerifyFrozenIdentity(); err == nil {
				t.Fatal("tampered frozen thesis accepted")
			}
		})
	}
}

func TestEvidenceReassessmentIsIdempotentAndConflictsFailClosed(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	thesis := fixtureThesis()
	position, err := OpenApprovedPosition("position-reassess", thesis, fixtureBinding(thesis), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	evidence := RelevantEvidence{Reference: EvidenceReference{EvidenceID: "replay-1", SourceID: "issuer-source", SourceURL: "https://example.test/replay", ObservedAt: now}, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, Signal: EvidenceInvalidates, Reason: "mechanism disproved"}
	if err := position.Reassess(evidence, now.Add(time.Hour), "policy-v1"); err != nil {
		t.Fatal(err)
	}
	transitions := len(position.Transitions)
	if err := position.Reassess(evidence, now.Add(2*time.Hour), "policy-v1"); err != nil {
		t.Fatal(err)
	}
	if len(position.Transitions) != transitions {
		t.Fatalf("duplicate evidence created %d transitions", len(position.Transitions)-transitions)
	}
	conflict := evidence
	conflict.Reason = "different content for same identity"
	if err := position.Reassess(conflict, now.Add(3*time.Hour), "policy-v1"); err == nil {
		t.Fatal("conflicting evidence identity accepted")
	}
}

func TestExitPrecedenceIsCanonicalAndDeterministic(t *testing.T) {
	cal := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-03": false, "2026-01-04": false, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true}, Timezone: "UTC", OpenTime: "09:00", CloseTime: "17:00"}
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	position, err := OpenApprovedPosition("position-exit", fixtureThesis(), fixtureBinding(fixtureThesis()), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*ExitInput)
		want   ExitReason
	}{
		{"risk beats stop", func(in *ExitInput) { in.RiskKill = true; in.Bid = 95 }, ExitRiskKill},
		{"manual beats stop", func(in *ExitInput) { in.ManualOperator = true; in.Bid = 95 }, ExitManualOperator},
		{"invalidation beats target", func(in *ExitInput) { in.ThesisInvalidated = true; in.Bid = 110 }, ExitThesisInvalidated},
		{"stop beats target", func(in *ExitInput) { in.Bid = 95 }, ExitStop},
		{"time limit after explicit triggers", func(in *ExitInput) { in.RiskKill = true; in.Bid = 110 }, ExitRiskKill},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := ExitInput{Now: now, Session: SessionOpen, Calendar: cal}
			test.mutate(&in)
			got, err := EvaluateExit(position, in)
			if err != nil || got.Reason != test.want {
				t.Fatalf("decision=%#v err=%v", got, err)
			}
		})
	}
	late, err := EvaluateExit(position, ExitInput{Now: time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC), Session: SessionOpen, Bid: 101, Calendar: cal})
	if err != nil || late.Reason != ExitTimeLimit {
		t.Fatalf("late decision=%#v err=%v", late, err)
	}
}

func TestCalendarRejectsUnknownOrCallerOverrideAndHandlesEarlyClose(t *testing.T) {
	cal := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true}, Timezone: "UTC", OpenTime: "09:00", CloseTime: "17:00"}
	thesis := fixtureThesis()
	position, err := OpenApprovedPosition("position-calendar", thesis, fixtureBinding(thesis), thesis.CreatedAt, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []ExitInput{
		{Now: time.Date(2026, 1, 2, 8, 59, 0, 0, time.UTC), Session: SessionClosed, Calendar: cal},
		{Now: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), Session: SessionClosed, Calendar: cal},
		{Now: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC), Session: SessionClosed, Calendar: cal},
	} {
		if _, err := EvaluateExit(position, in); err == nil {
			t.Fatal("invalid session state accepted")
		}
	}
	unknown := cal
	unknown.Sessions = map[string]bool{"2026-01-02": true}
	if _, err := EvaluateExit(position, ExitInput{Now: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), Session: SessionOpen, Calendar: unknown}); err == nil {
		t.Fatal("incomplete calendar accepted")
	}
}

func TestOutcomeAccountingAndExcursionsAreDirectionAware(t *testing.T) {
	versions := PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "risk-v1", EntryPolicy: "entry-v1", ExitPolicy: "exit-v1", CostModel: "paper-cost-v1"}
	base := Outcome{OutcomeID: "o", Mode: ExploratoryPaperMode, TraderModelVersion: TraderModelVersion, ThesisID: "t", ThesisHash: "sha256:t", PositionID: "p", EventID: "e", IssuerID: "i", InstrumentID: "AAPL", Direction: DirectionLong, Quantity: 2, EntryOrderID: "eo", EntryFillID: "ef", ExitOrderID: "xo", ExitFillID: "xf", EntryAt: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC), ExitAt: time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC), EntryPrice: 100, ExitPrice: 110, EntryCosts: 1, ExitCosts: 2, TotalCosts: 3, Costs: 3, GrossPnL: 20, NetPnL: 17, ReturnDenominator: 200, NetReturn: .085, SessionsHeld: 2, ExitReason: ExitTarget, ExcursionStatus: "COMPLETE", ExcursionKnown: true, PricePath: []PriceObservation{{ObservationID: "a", At: time.Date(2026, 1, 2, 11, 0, 0, 0, time.UTC), Price: 105, Source: "path"}, {ObservationID: "b", At: time.Date(2026, 1, 3, 11, 0, 0, 0, time.UTC), Price: 98, Source: "path"}}, MFE: 10, MAE: -4, PolicyVersions: versions, CostModelVersion: "paper-cost-v1"}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	short := base
	short.Direction, short.EntryPrice, short.ExitPrice, short.GrossPnL, short.NetPnL, short.NetReturn = DirectionShort, 110, 100, 20, 17, .07727272727272727
	short.ReturnDenominator = 220
	short.PricePath = []PriceObservation{{ObservationID: "c", At: base.EntryAt.Add(time.Hour), Price: 105, Source: "path"}, {ObservationID: "d", At: base.EntryAt.Add(2 * time.Hour), Price: 115, Source: "path"}}
	short.MFE, short.MAE = 10, -10
	if err := short.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := base
	bad.NetPnL = 20
	if err := bad.Validate(); err == nil {
		t.Fatal("costs were not deducted from net P&L")
	}
	incomplete := base
	incomplete.ExcursionKnown, incomplete.ExcursionStatus, incomplete.MFE, incomplete.MAE, incomplete.PricePath = false, "INCOMPLETE", 0, 0, nil
	if err := incomplete.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckpointValidationRejectsIdentityTimeAndArithmeticErrors(t *testing.T) {
	versions := PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "risk-v1", EntryPolicy: "entry-v1", ExitPolicy: "exit-v1", CostModel: "paper-cost-v1"}
	entryAt := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	exitAt := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	base := Outcome{OutcomeID: "checkpoint-outcome", Mode: ExploratoryPaperMode, TraderModelVersion: TraderModelVersion, ThesisID: "t", ThesisHash: "sha256:t", PositionID: "p", EventID: "e", IssuerID: "i", InstrumentID: "AAPL", Direction: DirectionLong, Quantity: 2, EntryOrderID: "eo", EntryFillID: "ef", ExitOrderID: "xo", ExitFillID: "xf", EntryAt: entryAt, ExitAt: exitAt, EntryPrice: 100, ExitPrice: 104, EntryCosts: 1, ExitCosts: 1, TotalCosts: 2, Costs: 2, GrossPnL: 8, NetPnL: 6, ReturnDenominator: 200, NetReturn: .03, SessionsHeld: 2, ExitReason: ExitTarget, ExcursionStatus: "INCOMPLETE", PolicyVersions: versions, CostModelVersion: "paper-cost-v1"}
	valid := Checkpoint{PositionID: "p", ThesisID: "t", Sessions: 1, At: entryAt.Add(time.Hour), Price: 102, PriceSource: "identified-path", EvidenceReviewID: "review-1", GrossPnL: 4, NetPnL: 2, NetReturn: .01, DataQuality: "complete"}
	base.Checkpoints = []Checkpoint{valid}
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*Outcome){
		func(o *Outcome) { o.Checkpoints = append(o.Checkpoints, valid) },
		func(o *Outcome) { o.Checkpoints[0].PositionID = "other" },
		func(o *Outcome) { o.Checkpoints[0].At = entryAt.Add(-time.Minute) },
		func(o *Outcome) { o.Checkpoints[0].NetPnL = 999 },
		func(o *Outcome) { o.Checkpoints[0].Sessions = 3 },
	}
	for i, mutate := range mutations {
		candidate := base
		candidate.Checkpoints = append([]Checkpoint(nil), base.Checkpoints...)
		mutate(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
	}
}

func TestCalendarTimezoneAndDSTAreExplicit(t *testing.T) {
	cal := SessionCalendar{Sessions: map[string]bool{"2026-03-06": true, "2026-03-07": false, "2026-03-08": false, "2026-03-09": true, "2026-03-10": true, "2026-03-11": true, "2026-03-12": true}, Timezone: "America/New_York", OpenTime: "09:30", CloseTime: "16:00"}
	entry := time.Date(2026, 3, 6, 15, 0, 0, 0, time.UTC)
	if state, err := cal.SessionState(entry); err != nil || state != SessionOpen {
		t.Fatalf("DST pre-switch state=%s err=%v", state, err)
	}
	hard, err := cal.HardExitDate(entry)
	if err != nil || hard.In(time.FixedZone("EDT", -4*60*60)).Format("2006-01-02") != "2026-03-12" {
		t.Fatalf("DST hard exit=%s err=%v", hard, err)
	}
	if state, err := cal.SessionState(time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)); err != nil || state != SessionClosed {
		t.Fatalf("DST timezone boundary was not enforced: state=%s err=%v", state, err)
	}
}
