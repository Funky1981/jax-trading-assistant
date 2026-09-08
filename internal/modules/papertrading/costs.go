package papertrading

import (
	"fmt"
	"time"
)

type CostModel struct {
	ModelID           string        `json:"model_id"`
	ContractVersion   string        `json:"contract_version"`
	SpreadBPS         float64       `json:"spread_bps"`
	SlippageBPS       float64       `json:"slippage_bps"`
	CommissionPerUnit float64       `json:"commission_per_unit"`
	FixedCommission   float64       `json:"fixed_commission"`
	Latency           time.Duration `json:"latency"`
}

func DefaultCostModel() CostModel {
	return CostModel{ModelID: "diagnostic-cost-model-v1", ContractVersion: CostModelContractVersion, SpreadBPS: 4, SlippageBPS: 5, CommissionPerUnit: 0.005, Latency: time.Second}
}

func (model CostModel) Validate() error {
	if model.ModelID == "" || model.ContractVersion != CostModelContractVersion || !finiteNonNegative(model.SpreadBPS) || !finiteNonNegative(model.SlippageBPS) || !finiteNonNegative(model.CommissionPerUnit) || !finiteNonNegative(model.FixedCommission) || model.Latency < 0 {
		return fmt.Errorf("%w: invalid cost model", ErrInvalidContract)
	}
	return nil
}

type CostBreakdown struct {
	ModelID       string  `json:"model_id"`
	MidPrice      float64 `json:"mid_price"`
	SpreadCost    float64 `json:"spread_cost"`
	SlippageCost  float64 `json:"slippage_cost"`
	Commission    float64 `json:"commission"`
	ExecutedPrice float64 `json:"executed_price"`
}

func (model CostModel) Price(tick MarketTick, direction string, quantity float64) (CostBreakdown, error) {
	if err := model.Validate(); err != nil {
		return CostBreakdown{}, err
	}
	if direction != "LONG" && direction != "SHORT" || !finitePositive(quantity) {
		return CostBreakdown{}, ErrInvalidArtifact
	}
	mid := (tick.Bid + tick.Ask) / 2
	halfSpread := mid * model.SpreadBPS / 20000
	slip := mid * model.SlippageBPS / 10000
	spreadPrice := mid + halfSpread
	price := spreadPrice + slip
	if direction == "SHORT" {
		spreadPrice = mid - halfSpread
		price = spreadPrice - slip
	}
	commission := quantity*model.CommissionPerUnit + model.FixedCommission
	return CostBreakdown{ModelID: model.ModelID, MidPrice: mid, SpreadCost: halfSpread * quantity, SlippageCost: slip * quantity, Commission: commission, ExecutedPrice: price}, nil
}
