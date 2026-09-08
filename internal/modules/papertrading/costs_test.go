package papertrading

import (
	"math"
	"testing"
	"time"
)

func TestCostModelIsExplicitAndDirectional(t *testing.T) {
	model := DefaultCostModel()
	tick := MarketTick{TickID: "cost", InstrumentID: "AAPL", Bid: 99, Ask: 101, Last: 100, AvailableQuantity: 10, Timestamp: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), ReceivedAt: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), Session: SessionOpen, Source: "fixture"}
	buy, err := model.Price(tick, "LONG", 10)
	if err != nil {
		t.Fatal(err)
	}
	sell, err := model.Price(tick, "SHORT", 10)
	if err != nil {
		t.Fatal(err)
	}
	if buy.ExecutedPrice <= buy.MidPrice || sell.ExecutedPrice >= sell.MidPrice || buy.Commission <= 0 || buy.SpreadCost <= 0 || buy.SlippageCost <= 0 {
		t.Fatalf("cost model did not apply directional costs: buy=%#v sell=%#v", buy, sell)
	}
	if math.Abs(buy.ExecutedPrice-100.07) > 1e-9 || math.Abs(buy.SpreadCost-0.2) > 1e-9 || math.Abs(buy.SlippageCost-0.5) > 1e-9 || math.Abs(buy.Commission-0.05) > 1e-9 {
		t.Fatalf("unexpected independently calculated buy costs: %#v", buy)
	}
	if buy.ModelID != model.ModelID || sell.ModelID != model.ModelID {
		t.Fatal("cost model provenance missing")
	}
}

func TestCostModelAllowsExplicitZeroCostDiagnosticOnly(t *testing.T) {
	model := CostModel{ModelID: "zero-cost-diagnostic", ContractVersion: CostModelContractVersion}
	if err := model.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := model
	bad.SlippageBPS = -1
	if bad.Validate() == nil {
		t.Fatal("negative slippage accepted")
	}
}
