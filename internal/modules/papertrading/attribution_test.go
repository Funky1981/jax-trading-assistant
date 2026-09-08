package papertrading

import (
	"math"
	"testing"
	"time"
)

func TestAttributionSeparatesThesisRiskExecutionAndCostEffects(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	fill := testFill("attribution", "LONG", 6, 101.07, .03, now)
	fill.Costs.SpreadCost, fill.Costs.SlippageCost = .12, .30
	fill.FillID = fillIdentity(fill)
	result, err := Attribute(AttributionInput{RecommendationID: "rec", RiskDecisionID: "risk", PaperIntentID: fill.PaperIntentID, OrderID: fill.OrderID, FillID: fill.FillID, Direction: "LONG", RecommendationEntry: 100, RiskEntry: 100, ActivationReference: 101, RecommendationQty: 10, RiskQty: 6, ExitPrice: 110, Fill: fill})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(result.GrossThesisPnL-100) > 1e-9 || math.Abs(result.RiskSizingImpact+40) > 1e-9 || math.Abs(result.LatencyImpact-6) > 1e-9 || math.Abs(result.ExecutionPriceImpact-.42) > 1e-9 || math.Abs(result.SpreadImpact+.12) > 1e-9 || math.Abs(result.SlippageImpact+.30) > 1e-9 || math.Abs(result.FeeImpact+.03) > 1e-9 || math.Abs(result.NetPaperPnL-53.55) > 1e-9 || result.Identity == "" {
		t.Fatalf("attribution = %#v", result)
	}
}

func TestAttributionRejectsMismatchedFillProvenance(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	fill := testFill("attribution-mismatch", "LONG", 1, 100, 0, now)
	if _, err := Attribute(AttributionInput{RecommendationID: "rec", RiskDecisionID: "risk", PaperIntentID: fill.PaperIntentID, OrderID: fill.OrderID, FillID: "wrong", Direction: "LONG", RecommendationEntry: 100, RiskEntry: 100, ActivationReference: 100, RecommendationQty: 1, RiskQty: 1, ExitPrice: 101, Fill: fill}); err == nil {
		t.Fatal("mismatched fill provenance accepted")
	}
}
