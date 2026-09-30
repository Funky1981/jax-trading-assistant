package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
)

func TestProductionPaperExecutionObservationResolvesCanonicalInstrumentSymbol(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ctx := context.Background()
	const instrumentID = "jax.instrument.us.etf.qqq"
	const symbol = "QQQ"
	const provider = "identity-regression-provider"
	policyPath, err := filepath.Abs(filepath.Join("..", "..", "config", "core-readiness-02b-candidate-economic-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, policyPath)

	base := time.Now().UTC().Truncate(time.Microsecond)
	quoteAt := base.Add(time.Second)
	tradeAt := base.Add(1100 * time.Millisecond)
	receivedAt := base.Add(1200 * time.Millisecond)
	asOf := base.Add(3 * time.Second)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM quotes WHERE provider=$1 AND symbol IN ($2,$3,$4)`, provider, symbol, instrumentID, "SPY")
	})
	insertQuote := func(quoteSymbol string) error {
		_, err := pool.Exec(ctx, `
			INSERT INTO quotes(symbol,price,bid,ask,bid_size,ask_size,volume,timestamp,last_trade_timestamp,exchange,provider,received_at)
			VALUES($1,100,99.9,100.1,20,20,1000,$2,$3,'identity-test-exchange',$4,$5)
			ON CONFLICT(symbol) DO UPDATE SET price=EXCLUDED.price,bid=EXCLUDED.bid,ask=EXCLUDED.ask,
				bid_size=EXCLUDED.bid_size,ask_size=EXCLUDED.ask_size,volume=EXCLUDED.volume,
				timestamp=EXCLUDED.timestamp,last_trade_timestamp=EXCLUDED.last_trade_timestamp,
				exchange=EXCLUDED.exchange,provider=EXCLUDED.provider,received_at=EXCLUDED.received_at
		`, quoteSymbol, quoteAt, tradeAt, provider, receivedAt)
		return err
	}
	if err := insertQuote(instrumentID); err != nil {
		t.Fatal(err)
	}
	marketPolicy := marketDataSafetyPolicy{AllowedSources: []string{provider}, QuoteMaxAge: time.Minute, allowNonProductionSources: true}
	calendar := ops01Calendar(asOf)
	source := &postgresPaperExecutionObservationSource{pool: pool, marketPolicy: &marketPolicy, calendar: &calendar}
	if tick, available, err := source.LoadExecutionObservation(ctx, instrumentID, symbol, "LONG", asOf); err != nil || available {
		t.Fatalf("opaque instrument ID quote row must not be used as a symbol fallback: tick=%#v available=%t err=%v", tick, available, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM quotes WHERE symbol=$1 AND provider=$2`, instrumentID, provider); err != nil {
		t.Fatal(err)
	}
	if err := insertQuote(symbol); err != nil {
		t.Fatal(err)
	}
	tick, available, err := source.LoadExecutionObservation(ctx, instrumentID, symbol, "LONG", asOf)
	if err != nil || !available {
		t.Fatalf("production execution adapter did not load mapped QQQ quote: available=%t err=%v", available, err)
	}
	if tick.InstrumentID != instrumentID || tick.MarketSymbol != symbol || tick.Source != provider {
		t.Fatalf("canonical execution identity was not retained: %#v", tick)
	}
	provenance, err := tick.Provenance()
	if err != nil {
		t.Fatal(err)
	}
	wantAvailableAt := receivedAt
	if !provenance.QuoteProviderAt.Equal(quoteAt) || !provenance.TradeProviderAt.Equal(tradeAt) || !provenance.ReceivedAt.Equal(receivedAt) || !provenance.AsOf.Equal(asOf) || !provenance.AvailableAt.Equal(wantAvailableAt) || provenance.MarketSymbol != symbol || !provenance.StrictTemporal {
		t.Fatalf("execution quote provenance mismatch: %#v", provenance)
	}

	venueContract, costModel := papertrading.DefaultPaperCapabilityContract(), papertrading.DefaultCostModel()
	venue, err := papertrading.NewPaperVenue(venueContract, costModel)
	if err != nil {
		t.Fatal(err)
	}
	for _, direction := range []string{"LONG", "SHORT"} { // pending entry and approved exit sides
		workflowSnapshot, intent := canonicalIdentityTestApproval(t, base, instrumentID, direction)
		order, err := venue.Submit(papertrading.CreateOrderRequest{
			Workflow: workflowSnapshot, PaperIntent: intent, Venue: venueContract, CostModel: costModel,
			InstrumentID: instrumentID, Quantity: 1, ReferencePrice: tick.Last,
			OrderType: papertrading.OrderMarket, CreatedAt: base, IdempotencyKey: intent.IntentID,
		})
		if err != nil {
			t.Fatalf("create %s pending PAPER order: %v", direction, err)
		}
		if order.Status != papertrading.OrderNew || !order.ActivatesAt.After(base) {
			t.Fatalf("%s order was not pending until latency: %#v", direction, order)
		}
		fills, err := venue.ProcessOrderTickWithSafety(tick, order.OrderID, false)
		if err != nil || len(fills) != 1 {
			t.Fatalf("%s pending order could not consume mapped post-activation quote: fills=%#v err=%v", direction, fills, err)
		}
		if fills[0].InstrumentID != instrumentID || fills[0].MarketProvenance == nil || fills[0].MarketProvenance.MarketSymbol != symbol || fills[0].FilledAt.Before(order.ActivatesAt) {
			t.Fatalf("%s fill lost canonical identity/provenance or activation ordering: %#v", direction, fills[0])
		}
	}

	if tick, available, err := source.LoadExecutionObservation(ctx, "jax.instrument.unknown", symbol, "LONG", asOf); err == nil || available {
		t.Fatalf("unknown canonical instrument must fail closed, not become pending: tick=%#v available=%t err=%v", tick, available, err)
	}
	missingPolicy := filepath.Join(t.TempDir(), "missing-economic-policy.json")
	t.Setenv(candidateEconomicPolicyEnv, missingPolicy)
	if _, available, err := source.LoadExecutionObservation(ctx, instrumentID, symbol, "LONG", asOf); err == nil || available {
		t.Fatalf("missing identity policy must fail closed: available=%t err=%v", available, err)
	}

	ambiguous := candidateEconomicPolicy{
		PolicyVersion: "integration-v1", IdentityPolicy: "integration-identity-v1", IdentitySource: "disposable-integration-map",
		SizingPolicyID: "integration-sizing", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: 1,
		SlippageAllowance: floatPointer(.25),
		Instruments: map[string]candidateEconomicIdentity{
			"QQQ":    {InstrumentID: instrumentID, IssuerID: "issuer-qqq"},
			"QQQALT": {InstrumentID: instrumentID, IssuerID: "issuer-qqq-alt"},
		},
	}
	policyBytes, err := json.Marshal(ambiguous)
	if err != nil {
		t.Fatal(err)
	}
	ambiguousPath := filepath.Join(t.TempDir(), "ambiguous-economic-policy.json")
	if err := os.WriteFile(ambiguousPath, policyBytes, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, ambiguousPath)
	if _, available, err := source.LoadExecutionObservation(ctx, instrumentID, symbol, "LONG", asOf); err == nil || available {
		t.Fatalf("ambiguous canonical instrument mapping must fail closed: available=%t err=%v", available, err)
	}

	// A changed policy mapping cannot redirect an approved order away from its
	// frozen QQQ market identity.
	drifted := candidateEconomicPolicy{
		PolicyVersion: "integration-v2", IdentityPolicy: "integration-identity-v2", IdentitySource: "disposable-drift-map",
		SizingPolicyID: "integration-sizing", SizingPolicyVersion: "v1", RiskAllocation: .01, RequestedLeverage: 1,
		SlippageAllowance: floatPointer(.25),
		Instruments:       map[string]candidateEconomicIdentity{"OTHER": {InstrumentID: instrumentID, IssuerID: "issuer-qqq"}},
	}
	driftBytes, err := json.Marshal(drifted)
	if err != nil {
		t.Fatal(err)
	}
	driftPath := filepath.Join(t.TempDir(), "drifted-economic-policy.json")
	if err := os.WriteFile(driftPath, driftBytes, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(candidateEconomicPolicyEnv, driftPath)
	if _, available, err := source.LoadExecutionObservation(ctx, instrumentID, symbol, "LONG", asOf); err == nil || available {
		t.Fatalf("economic identity policy drift must fail closed before quote lookup: available=%t err=%v", available, err)
	}

	// Distinct canonical instruments with identical provider timestamps must
	// receive distinct venue tick identities and coexist without idempotency
	// collisions.
	t.Setenv(candidateEconomicPolicyEnv, policyPath)
	const spyID, spySymbol = "jax.instrument.us.etf.spy", "SPY"
	if err := insertQuote(spySymbol); err != nil {
		t.Fatal(err)
	}
	spyTick, available, err := source.LoadExecutionObservation(ctx, spyID, spySymbol, "LONG", asOf)
	if err != nil || !available {
		t.Fatalf("mapped SPY observation unavailable: available=%t err=%v", available, err)
	}
	if spyTick.TickID == tick.TickID || spyTick.Timestamp != tick.Timestamp || spyTick.LastProviderAt != tick.LastProviderAt || spyTick.ReceivedAt != tick.ReceivedAt {
		t.Fatalf("canonical execution tick identity omitted instrument while timestamps match: QQQ=%#v SPY=%#v", tick, spyTick)
	}
	venue, err = papertrading.NewPaperVenue(venueContract, costModel)
	if err != nil {
		t.Fatal(err)
	}
	for index, marketTick := range []papertrading.MarketTick{tick, spyTick} {
		wf, intent := canonicalIdentityTestApproval(t, base, marketTick.InstrumentID, "LONG")
		order, err := papertrading.CreatePaperOrder(papertrading.CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: venueContract, CostModel: costModel, InstrumentID: marketTick.InstrumentID, Quantity: 1, ReferencePrice: marketTick.Last, OrderType: papertrading.OrderMarket, CreatedAt: base, IdempotencyKey: intent.IntentID})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := venue.Submit(papertrading.CreateOrderRequest{Workflow: wf, PaperIntent: intent, Venue: venueContract, CostModel: costModel, InstrumentID: marketTick.InstrumentID, Quantity: 1, ReferencePrice: marketTick.Last, OrderType: papertrading.OrderMarket, CreatedAt: base, IdempotencyKey: intent.IntentID}); err != nil {
			t.Fatal(err)
		}
		fills, err := venue.ProcessOrderTickWithSafety(marketTick, order.OrderID, false)
		if err != nil || len(fills) != 1 {
			t.Fatalf("instrument %s tick collided with another instrument: fills=%d err=%v", marketTick.InstrumentID, len(fills), err)
		}
		replayed, err := venue.ProcessOrderTickWithSafety(marketTick, order.OrderID, false)
		if err != nil || len(replayed) != 0 || len(venue.Snapshot().Fills) != index+1 {
			t.Fatalf("same-instrument observation replay was not exactly once: replay=%d totalFills=%d err=%v", len(replayed), len(venue.Snapshot().Fills), err)
		}
	}
}

func canonicalIdentityTestApproval(t *testing.T, at time.Time, instrumentID, direction string) (workflow.Workflow, workflow.PaperIntent) {
	t.Helper()
	accountID := "identity-test-account-" + direction
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: accountID, AsOf: at, CapturedAt: at, Provider: "disposable-fixture", Currency: "USD", Synthetic: true,
		Cash: portfoliorisk.KnownNumber(10000, "fixture"), Equity: portfoliorisk.KnownNumber(10000, "fixture"),
		Positions: []portfoliorisk.Position{{InstrumentID: "MSFT", InstrumentResolved: true, Currency: "USD", SignedQuantity: 1,
			Price: portfoliorisk.KnownNumber(100, "fixture"), MarketValue: portfoliorisk.KnownNumber(100, "fixture"),
			CostBasis: portfoliorisk.UnknownNumber("not supplied"), ValuationAsOf: at, PriceSource: "fixture", Provenance: []string{"fixture"}}},
		ValuationBasis: "frozen", Provenance: []string{"disposable-fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, at, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := portfoliorisk.BuildRiskPolicy(portfoliorisk.RiskPolicy{
		Version: "identity-test-risk-v1", Currency: "USD", MaximumLeverage: portfoliorisk.Limit(1),
		MaximumPositionValue: portfoliorisk.Limit(2000), MaximumConcentration: portfoliorisk.Limit(.5),
		MaximumGrossExposure: portfoliorisk.Limit(5000), MaximumNetExposure: portfoliorisk.Limit(5000),
	})
	if err != nil {
		t.Fatal(err)
	}
	riskAllocation, leverage := .01, 1.0
	recommendationID := "identity-test-recommendation-" + direction
	risk := portfoliorisk.EvaluateRecommendation(portfoliorisk.RecommendationRiskInput{
		RecommendationID: recommendationID, InstrumentID: instrumentID, Currency: "USD", SignedMarketValue: 1000,
		RiskAllocation: &riskAllocation, RequestedLeverage: &leverage, ExecutionAuthority: "NONE", QuantResultIDs: []string{"fixture"},
	}, snapshot, analytics, policy, at, time.Hour)
	if risk.Outcome != portfoliorisk.DecisionAccept {
		t.Fatalf("disposable PAPER test risk decision was not accepted: %s %v", risk.Outcome, risk.ReasonCodes)
	}
	store := workflow.NewStore()
	wf, err := store.Create(context.Background(), workflow.CreateRequest{RiskDecision: risk, Now: at, IdempotencyKey: "identity-test-create-" + direction})
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.RequestHumanConfirmation(context.Background(), workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "identity-test-request-" + direction, Now: at})
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err := workflow.NewConfirmation(wf, instrumentID, direction, at, at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	confirmation, err = confirmation.WithDecision("disposable-human", workflow.ConfirmationApprove)
	if err != nil {
		t.Fatal(err)
	}
	wf, err = store.Confirm(context.Background(), workflow.HumanConfirmationRequest{WorkflowID: wf.WorkflowID, Confirmation: confirmation, Actor: "disposable-human", ActorRole: workflow.ActorHuman, IdempotencyKey: "identity-test-approve-" + direction, Now: at})
	if err != nil {
		t.Fatal(err)
	}
	wf, intent, err := store.CreatePaperIntent(context.Background(), workflow.PaperIntentRequest{WorkflowID: wf.WorkflowID, Actor: "system", ActorRole: workflow.ActorSystem, IdempotencyKey: "identity-test-intent-" + direction, Now: at})
	if err != nil {
		t.Fatal(err)
	}
	return wf, intent
}
