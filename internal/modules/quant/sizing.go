package quant

import (
	"fmt"
	"math"
)

const PositionSizingAlgorithmV1 = "jax.quant.position_sizing/v1"

// CalculatePositionSizing calculates a bounded, descriptive quantity from an
// explicit account risk budget. It does not create an order or trade
// candidate. Fractional quantities are retained because the contract does not
// assume a venue's lot-size rules.
func CalculatePositionSizing(dataset FrozenDataset, entryPrice, stopPrice, accountEquity, riskFraction, maxQuantity float64) (Response, error) {
	if !finitePositive(entryPrice) || !finitePositive(stopPrice) || entryPrice == stopPrice || !finitePositive(accountEquity) || !finitePositive(riskFraction) || riskFraction > 1 || !finitePositive(maxQuantity) {
		return Response{}, fmt.Errorf("position sizing requires distinct positive prices, positive equity, risk fraction in (0,1], and a positive quantity cap")
	}
	request, err := newCalculationRequest(dataset, nil, "position_sizing", PositionSizingAlgorithmV1, []Parameter{
		{Name: "account_equity", Value: formatFloat(accountEquity)},
		{Name: "entry_price", Value: formatFloat(entryPrice)},
		{Name: "max_quantity", Value: formatFloat(maxQuantity)},
		{Name: "risk_fraction", Value: formatFloat(riskFraction)},
		{Name: "stop_price", Value: formatFloat(stopPrice)},
		{Name: "quantity_rounding", Value: "none_fractional_allowed"},
		{Name: "risk_budget_definition", Value: "account_equity_times_risk_fraction"},
		{Name: "unit_risk_definition", Value: "absolute_entry_minus_stop"},
	})
	if err != nil {
		return Response{}, err
	}
	riskBudget := accountEquity * riskFraction
	unitRisk := math.Abs(entryPrice - stopPrice)
	rawQuantity := riskBudget / unitRisk
	quantity := math.Min(rawQuantity, maxQuantity)
	if !finitePositive(riskBudget) || !finitePositive(unitRisk) || !finite(rawQuantity) || !finite(quantity) {
		return Response{}, fmt.Errorf("position sizing produced a non-finite result")
	}
	return newCalculationResponse(request, []Value{
		{Metric: "risk_budget", Value: riskBudget, Unit: "account_currency"},
		{Metric: "unit_risk", Value: unitRisk, Unit: "account_currency_per_unit"},
		{Metric: "raw_quantity", Value: rawQuantity, Unit: "units"},
		{Metric: "quantity", Value: quantity, Unit: "units"},
	}, nil)
}
