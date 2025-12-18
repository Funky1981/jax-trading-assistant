package app

import (
	"context"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type RiskTool interface {
	PositionSize(ctx context.Context, accountSize float64, riskPercent float64, entry float64, stop float64) (positionSize int, riskPerUnit float64, totalRisk float64, err error)
	RMultiple(ctx context.Context, entry float64, stop float64, target float64) (float64, error)
}

type RiskEngine struct {
	risk RiskTool
}

func NewRiskEngine(risk RiskTool) *RiskEngine {
	return &RiskEngine{risk: risk}
}

func (r *RiskEngine) Calculate(ctx context.Context, accountSize float64, riskPercent float64, entry float64, stop float64, target *float64) (domain.RiskResult, error) {
	positionSize, riskPerUnit, totalRisk, err := r.risk.PositionSize(ctx, accountSize, riskPercent, entry, stop)
	if err != nil {
		return domain.RiskResult{}, err
	}

	var rMultiple float64
	if target != nil {
		rm, err := r.risk.RMultiple(ctx, entry, stop, *target)
		if err != nil {
			return domain.RiskResult{}, err
		}
		rMultiple = rm
	}

	return domain.RiskResult{
		PositionSize: positionSize,
		RiskPerUnit:  riskPerUnit,
		TotalRisk:    totalRisk,
		RMultiple:    rMultiple,
	}, nil
}
