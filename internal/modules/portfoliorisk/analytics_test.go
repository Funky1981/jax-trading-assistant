package portfoliorisk

import (
	"testing"
	"time"
)

func TestCalculateExposureIsDeterministicAndSigned(t *testing.T) {
	snapshot := fixtureSnapshot()
	snapshot.Positions = append(snapshot.Positions, Position{InstrumentID: "NYSE:XYZ", InstrumentResolved: true, Currency: "USD", SignedQuantity: -100, Price: KnownNumber(100, "fixture"), MarketValue: KnownNumber(-10000, "fixture"), ValuationAsOf: snapshot.AsOf, PriceSource: "fixture"})
	first, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if first.AnalyticsID != second.AnalyticsID || first.GrossExposure != 30000 || first.NetExposure != 10000 || first.LongExposure != 20000 || first.ShortExposure != 10000 {
		t.Fatalf("non-deterministic or wrong exposure: %#v %#v", first, second)
	}
	if first.Lines[1].Long {
		t.Fatal("short line lost its sign")
	}
	if first.MaxConcentration != 0.2 {
		t.Fatalf("max concentration=%v want .2", first.MaxConcentration)
	}
}

func TestDerivedAnalyticsIdentityCannotBeTampered(t *testing.T) {
	snapshot := fixtureSnapshot()
	analytics, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	analytics.Lines[0].MarketValue = 1
	if err := analytics.Validate(); err == nil {
		t.Fatal("tampered analytics validated")
	}
}

func TestDerivedAnalyticsCannotBeReboundToSameSnapshot(t *testing.T) {
	snapshot := fixtureSnapshot()
	analytics, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	analytics.Lines = nil
	analytics.AnalyticsID = exposureIdentity(analytics)
	if analytics.Validate() != nil {
		t.Fatal("test analytics should be internally content-valid")
	}
	policy, err := BuildRiskPolicy(fixturePolicy())
	if err != nil {
		t.Fatal(err)
	}
	result := EvaluateRecommendation(recommendation(5000), snapshot, analytics, policy, stateNow, 10*time.Minute)
	if result.Outcome == DecisionAccept {
		t.Fatal("fabricated analytics was accepted")
	}
}

func TestExposureFailsClosedForUnknownAndStaleState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PortfolioSnapshot)
	}{
		{"unknown equity", func(s *PortfolioSnapshot) { s.Equity = UnknownNumber("omitted") }},
		{"missing price", func(s *PortfolioSnapshot) { s.Positions[0].Price = UnknownNumber("omitted") }},
		{"inconsistent signed value", func(s *PortfolioSnapshot) { s.Positions[0].MarketValue = KnownNumber(-20000, "tampered") }},
		{"stale state", func(s *PortfolioSnapshot) {
			s.AsOf = stateNow.Add(-time.Hour)
			s.CapturedAt = stateNow.Add(-time.Hour + time.Minute)
			s.Positions[0].ValuationAsOf = s.AsOf
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := fixtureSnapshot()
			tt.mutate(&snapshot)
			if _, err := CalculateExposure(snapshot, stateNow, 10*time.Minute); err == nil {
				t.Fatal("incomplete state was accepted")
			}
		})
	}
}

func TestCorrelationRejectsInsufficientLookaheadAndZeroVariance(t *testing.T) {
	asOf := stateNow.Add(-time.Minute)
	short := []ReturnObservation{{At: asOf.Add(-2 * time.Minute), AssetReturn: .1, BenchmarkReturn: .1}, {At: asOf.Add(-time.Minute), AssetReturn: .2, BenchmarkReturn: .2}}
	if result := CalculateCorrelation("ABC", "SPY", asOf, short, 10*time.Minute); result.Status != CorrelationUnknown {
		t.Fatalf("short correlation=%#v", result)
	}
	lookahead := append([]ReturnObservation(nil), short...)
	lookahead = append(lookahead, ReturnObservation{At: asOf.Add(time.Minute), AssetReturn: .3, BenchmarkReturn: .3})
	if result := CalculateCorrelation("ABC", "SPY", asOf, lookahead, 10*time.Minute); result.Status != CorrelationUnknown {
		t.Fatalf("lookahead correlation=%#v", result)
	}
	zero := []ReturnObservation{{At: asOf.Add(-3 * time.Minute), AssetReturn: .1, BenchmarkReturn: .1}, {At: asOf.Add(-2 * time.Minute), AssetReturn: .1, BenchmarkReturn: .2}, {At: asOf.Add(-time.Minute), AssetReturn: .1, BenchmarkReturn: .3}}
	if result := CalculateCorrelation("ABC", "SPY", asOf, zero, 10*time.Minute); result.Status != CorrelationUnknown {
		t.Fatalf("zero variance correlation=%#v", result)
	}
}

func TestCorrelationIsAlignedAndReproducible(t *testing.T) {
	asOf := stateNow.Add(-time.Minute)
	observations := []ReturnObservation{{At: asOf.Add(-3 * time.Minute), AssetReturn: .01, BenchmarkReturn: .02}, {At: asOf.Add(-2 * time.Minute), AssetReturn: .02, BenchmarkReturn: .01}, {At: asOf.Add(-time.Minute), AssetReturn: .03, BenchmarkReturn: .04}}
	a := CalculateCorrelation("ABC", "SPY", asOf, observations, 10*time.Minute)
	b := CalculateCorrelation("ABC", "SPY", asOf, observations, 10*time.Minute)
	if a.Status != CorrelationKnown || a.Value == nil || a.CorrelationID != b.CorrelationID {
		t.Fatalf("correlation not known/reproducible: %#v %#v", a, b)
	}
}
