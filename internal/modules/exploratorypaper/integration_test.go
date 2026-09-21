package exploratorypaper

import (
	"context"
	"math"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
)

func TestEventToApprovedPaperEntryAdverseEvidenceAndPaperExit(t *testing.T) {
	now := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	thesis := fixtureThesis()
	candidate, err := GenerateCandidate(CandidateInput{
		EventID: thesis.EventID, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID,
		EventCategory: thesis.EventCategory, EventTimestamp: thesis.EventTimestamp, GeneratedAt: thesis.CandidateGeneratedAt,
		SourceURL: thesis.Evidence[0].SourceURL, Evidence: thesis.Evidence,
		ReviewedEvidence: &EvidenceAssessment{Provider: "candidate_evidence_scores", PolicyVersion: "candidate-v1", EvidenceSetFingerprint: EvidenceSetFingerprint(thesis.Evidence), ReviewedAt: now, SourceBacked: true, QualityState: "sufficient", QualityScore: .9, RequiredQualityScore: .7, EvidenceReady: true, EvidenceGateReady: true, Corroborated: true, IssuerRelevant: true, InstrumentRelevant: true, IndependentSourceGroups: 2}, CandidatePolicyVersion: "candidate-v1",
		CausalMechanism: thesis.ExpectedMechanism, QuantContext: thesis.QuantTechnicalContext,
		RiskAssessment: thesis.RiskAssessment, TechnicalConfirmation: "confirmed short-term price support", Direction: DirectionLong,
	})
	if err != nil || candidate.Decision != DecisionCandidate {
		t.Fatalf("candidate=%#v err=%v", candidate, err)
	}

	wf, intent := approvedIntent(t, "entry", "LONG", now)
	costModel := papertrading.DefaultCostModel()
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), costModel)
	if err != nil {
		t.Fatal(err)
	}
	binding := EntryBinding{
		Mode: ExploratoryPaperMode, CandidateID: "candidate-1", ThesisID: thesis.ThesisID,
		TraderModelVersion: TraderModelVersion, WorkflowID: wf.WorkflowID, PaperIntentID: intent.IntentID,
		Environment: "PAPER", ExecutionAuthority: "NONE", MaximumLeverage: 1,
		PolicyVersions:    PolicyVersions{TraderModel: TraderModelVersion, ThesisContract: thesis.ContractVersion, CandidatePolicy: "candidate-v1", RiskPolicy: "paper-01-policy", EntryPolicy: thesis.EntryPolicyVersion, ExitPolicy: "exit-v1", CostModel: costModel.ModelID},
		ThesisContentHash: ThesisContentHash(thesis), EvidenceSetHash: EvidenceSetFingerprint(append(append([]EvidenceReference{}, thesis.Evidence...), thesis.CounterEvidence...)),
		BoundAt: now,
	}
	if err := binding.Validate(); err != nil {
		t.Fatal(err)
	}
	order, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: papertrading.DefaultCostModel(), InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 100, OrderType: papertrading.OrderMarket, CreatedAt: now, IdempotencyKey: "entry-order"})
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := papertrading.NewPaperLedger("paper-01-fixture", "USD", 10000)
	if err != nil {
		t.Fatal(err)
	}
	fills, err := venue.ProcessTick(papertrading.MarketTick{TickID: "entry-tick", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: now.Add(2 * time.Second), ReceivedAt: now.Add(2 * time.Second), Session: papertrading.SessionOpen, Source: "paper-01-fixture"})
	if err != nil || len(fills) != 1 || fills[0].OrderID != order.OrderID {
		t.Fatalf("entry fills=%#v err=%v", fills, err)
	}
	if _, err = ledger.ApplyFill(fills[0]); err != nil {
		t.Fatal(err)
	}

	position, err := OpenApprovedPosition("position-1", thesis, binding, now.Add(2*time.Second), fills[0].Price)
	if err != nil {
		t.Fatal(err)
	}
	if err := position.Reassess(RelevantEvidence{Reference: EvidenceReference{EvidenceID: "adverse-1", SourceID: "issuer-source", SourceURL: "https://example.test/adverse", Quality: "high", ObservedAt: now.AddDate(0, 0, 1)}, IssuerID: thesis.IssuerID, InstrumentID: thesis.InstrumentID, Signal: EvidenceInvalidates, Reason: "issuer correction removes the expected mechanism"}, now.AddDate(0, 0, 1), "policy-v1"); err != nil {
		t.Fatal(err)
	}
	if position.State != StateInvalidated || position.OperationalState != StateExitRecommended {
		t.Fatalf("position state=%#v", position)
	}

	exitAt := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
	calendar := SessionCalendar{Sessions: map[string]bool{"2026-01-02": true, "2026-01-03": false, "2026-01-04": false, "2026-01-05": true, "2026-01-06": true, "2026-01-07": true, "2026-01-08": true}, Timezone: "UTC", OpenTime: "09:00", CloseTime: "17:00"}
	exitDecision, err := EvaluateExit(position, ExitInput{Now: exitAt, Session: SessionOpen, Bid: 104, ThesisInvalidated: true, Calendar: calendar})
	if err != nil || exitDecision.Action != ExitNow || exitDecision.Reason != ExitThesisInvalidated {
		t.Fatalf("exit decision=%#v err=%v", exitDecision, err)
	}

	exitWF, exitIntent := approvedIntent(t, "exit", "SHORT", exitAt)
	exitOrder, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: exitWF, PaperIntent: exitIntent, Venue: papertrading.DefaultPaperCapabilityContract(), CostModel: costModel, InstrumentID: "AAPL", Quantity: 10, ReferencePrice: 104, OrderType: papertrading.OrderMarket, CreatedAt: exitAt, IdempotencyKey: "exit-order"})
	if err != nil {
		t.Fatal(err)
	}
	exitFills, err := venue.ProcessTick(papertrading.MarketTick{TickID: "exit-tick", InstrumentID: "AAPL", Bid: 103, Ask: 104, Last: 104, AvailableQuantity: 10, Timestamp: exitAt.Add(2 * time.Second), ReceivedAt: exitAt.Add(2 * time.Second), Session: papertrading.SessionOpen, Source: "paper-01-fixture"})
	if err != nil || len(exitFills) != 1 || exitFills[0].OrderID != exitOrder.OrderID {
		t.Fatalf("exit fills=%#v err=%v", exitFills, err)
	}
	account, err := ledger.ApplyFill(exitFills[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(account.Positions) != 0 {
		t.Fatalf("position remained open: %#v", account.Positions)
	}
	if err := position.Close(exitAt.Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if position.State != StateClosed || len(position.Transitions) != 1 {
		t.Fatalf("closed position=%#v", position)
	}
	path := []PriceObservation{{ObservationID: "entry-path", At: fills[0].FilledAt, Price: fills[0].Price, Source: "paper-tick"}, {ObservationID: "adverse-path", At: now.AddDate(0, 0, 1), Price: 99, Source: "paper-tick"}, {ObservationID: "exit-path", At: exitFills[0].FilledAt, Price: exitFills[0].Price, Source: "paper-tick"}}
	outcome, err := BuildOutcomeFromFills(position, binding, fills[0], exitFills[0], exitDecision.Reason, 2, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.Validate(); err != nil {
		t.Fatal(err)
	}
	if outcome.AccountingVersion != EconomicAccountingVersion || outcome.EntryCosts != fills[0].Costs.Commission || outcome.ExitCosts != exitFills[0].Costs.Commission {
		t.Fatalf("economic accounting attributed embedded execution costs twice: %#v", outcome)
	}
	if got, want := account.Cash-10000, outcome.NetPnL; math.Abs(got-want) > 1e-9 {
		t.Fatalf("ledger cash delta=%v, outcome net P&L=%v", got, want)
	}
	corrected, err := BuildOutcomeFromFillsWithCoverage(position, binding, fills[0], exitFills[0], exitDecision.Reason, 2, path, nil, ExcursionCoverage{Status: "COMPLETE", WindowStart: fills[0].FilledAt, WindowEnd: exitFills[0].FilledAt, ExpectedObservationCount: len(path), Cadence: "every-fill", SourceProvenance: true})
	if err != nil {
		t.Fatal(err)
	}
	if !corrected.ExcursionKnown || corrected.ExcursionStatus != "COMPLETE" {
		t.Fatalf("complete coverage did not produce known excursions: %#v", corrected)
	}
	checkpointGross := (101.0 - fills[0].Price) * fills[0].Quantity
	corrected.Checkpoints = []Checkpoint{{PositionID: position.PositionID, ThesisID: position.Thesis.ThesisID, Sessions: 1, At: fills[0].FilledAt.Add(time.Hour), Price: 101, PriceSource: "paper-tick", EvidenceReviewID: "checkpoint-1", GrossPnL: checkpointGross, NetPnL: checkpointGross - fills[0].Costs.Commission, NetReturn: (checkpointGross - fills[0].Costs.Commission) / (fills[0].Price * fills[0].Quantity), DataQuality: "COMPLETE", CommissionCost: fills[0].Costs.Commission, SpreadCost: fills[0].Costs.SpreadCost, SlippageCost: fills[0].Costs.SlippageCost, AccountingVersion: EconomicAccountingVersion}}
	if err := corrected.Validate(); err != nil {
		t.Fatalf("checkpoint accounting did not validate independently of future exit costs: %v", err)
	}
}

func approvedIntent(t *testing.T, suffix, direction string, now time.Time) (workflow.Workflow, workflow.PaperIntent) {
	t.Helper()
	value := 1000.0
	if suffix == "exit" {
		value = 1200
	}
	risk := acceptedRisk(t, "recommendation-"+suffix, now, value)
	store := workflow.NewStore()
	wf, err := store.Create(context.Background(), workflow.CreateRequest{RiskDecision: risk, Now: now, IdempotencyKey: suffix + "-create"})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.RequestHumanConfirmation(context.Background(), workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: suffix + "-await", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := workflow.NewConfirmation(wf, "AAPL", direction, now, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("operator-1", workflow.ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: "operator-1", ActorRole: workflow.ActorHuman, IdempotencyKey: suffix + "-approve", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	wf, intent, err := store.CreatePaperIntent(context.Background(), workflow.PaperIntentRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: suffix + "-intent", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	return wf, intent
}

func acceptedRisk(t *testing.T, recommendationID string, now time.Time, value float64) portfoliorisk.RiskDecision {
	t.Helper()
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{AccountID: "paper-01-fixture", AsOf: now, CapturedAt: now, Provider: "paper-01-fixture", Currency: "USD", Synthetic: true, Cash: portfoliorisk.KnownNumber(10000, "fixture"), Equity: portfoliorisk.KnownNumber(10000, "fixture"), Positions: []portfoliorisk.Position{{InstrumentID: "MSFT", InstrumentResolved: true, Currency: "USD", SignedQuantity: 1, Price: portfoliorisk.KnownNumber(100, "fixture"), MarketValue: portfoliorisk.KnownNumber(100, "fixture"), CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: now, PriceSource: "fixture", Provenance: []string{"fixture"}}}, ValuationBasis: "frozen", Provenance: []string{"paper-01-fixture"}})
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := portfoliorisk.BuildRiskPolicy(portfoliorisk.RiskPolicy{Version: "paper-01-policy", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(1), MaximumPositionValue: portfoliorisk.Limit(2000), MaximumConcentration: portfoliorisk.Limit(.5), MaximumGrossExposure: portfoliorisk.Limit(5000), MaximumNetExposure: portfoliorisk.Limit(5000)})
	if err != nil {
		t.Fatal(err)
	}
	riskAllocation, leverage := .01, 1.0
	decision := portfoliorisk.EvaluateRecommendation(portfoliorisk.RecommendationRiskInput{RecommendationID: recommendationID, InstrumentID: "AAPL", Currency: "USD", SignedMarketValue: value, RiskAllocation: &riskAllocation, RequestedLeverage: &leverage, ExecutionAuthority: "NONE", QuantResultIDs: []string{"quant:paper-01"}}, snapshot, analytics, policy, now, time.Hour)
	if decision.Outcome != portfoliorisk.DecisionAccept {
		t.Fatalf("risk decision=%#v", decision)
	}
	return decision
}
