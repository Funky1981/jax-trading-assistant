package adapters

import (
	"context"

	"jax-trading-assistant/libs/utcp"
)

type UTCPRiskAdapter struct {
	risk *utcp.RiskService
}

func NewUTCPRiskAdapter(risk *utcp.RiskService) *UTCPRiskAdapter {
	return &UTCPRiskAdapter{risk: risk}
}

func (a *UTCPRiskAdapter) PositionSize(ctx context.Context, accountSize float64, riskPercent float64, entry float64, stop float64) (int, float64, float64, error) {
	out, err := a.risk.PositionSize(ctx, utcp.PositionSizeInput{
		AccountSize: accountSize,
		RiskPercent: riskPercent,
		Entry:       entry,
		Stop:        stop,
	})
	if err != nil {
		return 0, 0, 0, err
	}
	return out.PositionSize, out.RiskPerUnit, out.TotalRisk, nil
}

func (a *UTCPRiskAdapter) RMultiple(ctx context.Context, entry float64, stop float64, target float64) (float64, error) {
	out, err := a.risk.RMultiple(ctx, utcp.RMultipleInput{
		Entry:  entry,
		Stop:   stop,
		Target: target,
	})
	if err != nil {
		return 0, err
	}
	return out.RMultiple, nil
}
