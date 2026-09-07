package portfoliorisk

import (
	"testing"
	"time"
)

func TestStressScenarioIsFrozenAndReproducible(t *testing.T) {
	snapshot := fixtureSnapshot()
	snapshot.Positions = append(snapshot.Positions, Position{InstrumentID: "NYSE:XYZ", InstrumentResolved: true, Currency: "USD", SignedQuantity: -100, Price: KnownNumber(100, "fixture"), MarketValue: KnownNumber(-10000, "fixture"), ValuationAsOf: snapshot.AsOf, PriceSource: "fixture"})
	analytics, err := CalculateExposure(snapshot, stateNow, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := BuildStressScenario(StressScenario{Version: "v1", Name: "correlated decline", DefaultShock: Limit(-.10), PriceShocks: map[string]float64{"NYSE:XYZ": -.20}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := CalculateStress(snapshot, analytics, scenario)
	if err != nil {
		t.Fatal(err)
	}
	b, err := CalculateStress(snapshot, analytics, scenario)
	if err != nil {
		t.Fatal(err)
	}
	if a.ResultID != b.ResultID || a.PnL != 0 || a.StressedEquity != 100000 {
		t.Fatalf("stress not reproducible: %#v %#v", a, b)
	}
	if a.ScenarioID == "" || a.Lines[1].Shock != -.20 {
		t.Fatalf("scenario identity/override missing: %#v", a)
	}
}

func TestStressScenarioRejectsMissingOrMutatedInputs(t *testing.T) {
	if _, err := BuildStressScenario(StressScenario{Version: "v1", Name: "bad"}); err == nil {
		t.Fatal("scenario without explicit default shock accepted")
	}
	snapshot, analytics, _ := fixtureAnalytics(t)
	scenario, err := BuildStressScenario(StressScenario{Version: "v1", Name: "shock", DefaultShock: Limit(-.1)})
	if err != nil {
		t.Fatal(err)
	}
	analytics.SnapshotID = "tampered"
	if _, err := CalculateStress(snapshot, analytics, scenario); err == nil {
		t.Fatal("mismatched analytics accepted")
	}
	changed := scenario
	changed.DefaultShock = Limit(-.2)
	if _, err := BuildStressScenario(changed); err == nil {
		t.Fatal("changed scenario reused old identity")
	}
}
