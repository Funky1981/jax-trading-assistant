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
	risk  RiskTool
	audit *AuditLogger
}

func NewRiskEngine(risk RiskTool, audit *AuditLogger) *RiskEngine {
	return &RiskEngine{risk: risk, audit: audit}
}

func (r *RiskEngine) Calculate(ctx context.Context, accountSize float64, riskPercent float64, entry float64, stop float64, target *float64) (domain.RiskResult, error) {
	if r.audit != nil {
		_ = r.audit.LogDecision(ctx, "risk_calculate_start", domain.AuditOutcomeStarted, redactRiskInputPayload(accountSize, riskPercent, entry, stop, target), nil)
	}
	positionSize, riskPerUnit, totalRisk, err := r.risk.PositionSize(ctx, accountSize, riskPercent, entry, stop)
	if err != nil {
		if r.audit != nil {
			_ = r.audit.LogDecision(ctx, "risk_calculate_error", domain.AuditOutcomeError, redactRiskInputPayload(accountSize, riskPercent, entry, stop, target), err)
		}
		return domain.RiskResult{}, err
	}

	var rMultiple float64
	if target != nil {
		rm, err := r.risk.RMultiple(ctx, entry, stop, *target)
		if err != nil {
			if r.audit != nil {
				_ = r.audit.LogDecision(ctx, "risk_calculate_rmultiple_error", domain.AuditOutcomeError, redactRiskInputPayload(accountSize, riskPercent, entry, stop, target), err)
			}
			return domain.RiskResult{}, err
		}
		rMultiple = rm
	}

	result := domain.RiskResult{
		PositionSize: positionSize,
		RiskPerUnit:  riskPerUnit,
		TotalRisk:    totalRisk,
		RMultiple:    rMultiple,
	}

	if r.audit != nil {
		_ = r.audit.LogDecision(ctx, "risk_calculate_success", domain.AuditOutcomeSuccess, redactRiskResultPayload(result), nil)
	}

	return result, nil
}
