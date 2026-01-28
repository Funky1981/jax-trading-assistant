package app

import (
	"context"
	"fmt"
	"math"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

// PortfolioRiskManager enforces portfolio-level risk constraints
type PortfolioRiskManager struct {
	constraints      domain.PortfolioConstraints
	positionLimits   domain.PositionLimits
	portfolioState   domain.PortfolioState
	sizingModel      domain.RiskSizingModel
	audit            *AuditLogger
}

// NewPortfolioRiskManager creates a new portfolio risk manager
func NewPortfolioRiskManager(
	constraints domain.PortfolioConstraints,
	positionLimits domain.PositionLimits,
	audit *AuditLogger,
) *PortfolioRiskManager {
	return &PortfolioRiskManager{
		constraints:    constraints,
		positionLimits: positionLimits,
		sizingModel:    domain.FixedFractional, // Default
		audit:          audit,
		portfolioState: domain.PortfolioState{
			SectorExposure: make(map[string]float64),
		},
	}
}

// SetPortfolioState updates the current portfolio state
func (p *PortfolioRiskManager) SetPortfolioState(state domain.PortfolioState) {
	p.portfolioState = state
}

// SetSizingModel sets the position sizing model
func (p *PortfolioRiskManager) SetSizingModel(model domain.RiskSizingModel) {
	p.sizingModel = model
}

// ValidatePosition checks if a proposed position passes all risk constraints
func (p *PortfolioRiskManager) ValidatePosition(
	ctx context.Context,
	symbol string,
	sector string,
	entry float64,
	stop float64,
	riskPercent float64,
) domain.RiskCheckResult {
	violations := []string{}
	metrics := domain.RiskMetrics{}

	// Check portfolio-level constraints first
	if p.portfolioState.AccountSize < p.constraints.MinAccountSize {
		return domain.RiskCheckResult{
			Allowed: false,
			Reason:  "account size below minimum threshold",
			Violations: []string{
				fmt.Sprintf("account size $%.2f < minimum $%.2f", 
					p.portfolioState.AccountSize, p.constraints.MinAccountSize),
			},
		}
	}

	if p.portfolioState.CurrentDrawdown >= p.constraints.MaxDrawdown {
		return domain.RiskCheckResult{
			Allowed: false,
			Reason:  "maximum drawdown exceeded",
			Violations: []string{
				fmt.Sprintf("current drawdown %.2f%% >= max %.2f%%", 
					p.portfolioState.CurrentDrawdown*100, p.constraints.MaxDrawdown*100),
			},
		}
	}

	if p.portfolioState.OpenPositions >= p.constraints.MaxPositions {
		violations = append(violations, 
			fmt.Sprintf("max positions reached (%d/%d)", 
				p.portfolioState.OpenPositions, p.constraints.MaxPositions))
	}

	// Calculate position metrics
	stopDistance := math.Abs((entry - stop) / entry)
	metrics.StopDistance = stopDistance

	// Validate stop distance
	if stopDistance < p.positionLimits.MinStopDistance {
		violations = append(violations, 
			fmt.Sprintf("stop too tight: %.2f%% < minimum %.2f%%", 
				stopDistance*100, p.positionLimits.MinStopDistance*100))
	}

	if stopDistance > p.positionLimits.MaxStopDistance {
		violations = append(violations, 
			fmt.Sprintf("stop too wide: %.2f%% > maximum %.2f%%", 
				stopDistance*100, p.positionLimits.MaxStopDistance*100))
	}

	// Validate risk percentage
	if riskPercent < p.positionLimits.MinRiskPerTrade {
		violations = append(violations, 
			fmt.Sprintf("risk too small: %.2f%% < minimum %.2f%%", 
				riskPercent*100, p.positionLimits.MinRiskPerTrade*100))
	}

	if riskPercent > p.positionLimits.MaxRiskPerTrade {
		violations = append(violations, 
			fmt.Sprintf("risk too large: %.2f%% > maximum %.2f%%", 
				riskPercent*100, p.positionLimits.MaxRiskPerTrade*100))
	}

	// Calculate position size using selected model
	positionSize, dollarRisk, riskPerUnit := p.calculatePositionSize(
		p.portfolioState.AccountSize, riskPercent, entry, stop)

	metrics.PositionSize = positionSize
	metrics.DollarRisk = dollarRisk
	metrics.RiskPerUnit = riskPerUnit
	metrics.PositionRisk = riskPercent

	// Check max position size
	positionValue := float64(positionSize) * entry
	if positionValue > p.constraints.MaxPositionSize {
		violations = append(violations, 
			fmt.Sprintf("position too large: $%.2f > maximum $%.2f", 
				positionValue, p.constraints.MaxPositionSize))
	}

	// Calculate leverage
	if p.portfolioState.AccountSize > 0 {
		metrics.Leverage = positionValue / p.portfolioState.AccountSize
		if metrics.Leverage > p.positionLimits.MaxLeverage {
			violations = append(violations, 
				fmt.Sprintf("leverage too high: %.2fx > maximum %.2fx", 
					metrics.Leverage, p.positionLimits.MaxLeverage))
		}
	}

	// Check portfolio risk
	newPortfolioRisk := p.portfolioState.TotalRisk + dollarRisk
	metrics.PortfolioRisk = newPortfolioRisk / p.portfolioState.AccountSize

	if metrics.PortfolioRisk > p.constraints.MaxPortfolioRisk {
		violations = append(violations, 
			fmt.Sprintf("portfolio risk too high: %.2f%% > maximum %.2f%%", 
				metrics.PortfolioRisk*100, p.constraints.MaxPortfolioRisk*100))
	}

	// Check sector exposure
	if sector != "" {
		currentSectorExposure := p.portfolioState.SectorExposure[sector]
		newSectorExposure := (currentSectorExposure + positionValue) / p.portfolioState.AccountSize
		metrics.SectorExposure = newSectorExposure

		if newSectorExposure > p.constraints.MaxSectorExposure {
			violations = append(violations, 
				fmt.Sprintf("sector exposure too high for %s: %.2f%% > maximum %.2f%%", 
					sector, newSectorExposure*100, p.constraints.MaxSectorExposure*100))
		}
	}

	// Determine if position is allowed
	allowed := len(violations) == 0
	reason := "all risk constraints satisfied"
	if !allowed {
		reason = "risk constraint violations detected"
	}

	result := domain.RiskCheckResult{
		Allowed:     allowed,
		Reason:      reason,
		Violations:  violations,
		RiskMetrics: metrics,
	}

	// Audit the risk check
	if p.audit != nil {
		outcome := domain.AuditOutcomeSuccess
		if !allowed {
			outcome = domain.AuditOutcomeRejected
		}
		_ = p.audit.LogDecision(ctx, "portfolio_risk_check", outcome, 
			map[string]interface{}{
				"symbol":     symbol,
				"sector":     sector,
				"allowed":    allowed,
				"violations": len(violations),
			}, nil)
	}

	return result
}

// calculatePositionSize computes position size using the selected sizing model
func (p *PortfolioRiskManager) calculatePositionSize(
	accountSize float64,
	riskPercent float64,
	entry float64,
	stop float64,
) (positionSize int, dollarRisk float64, riskPerUnit float64) {
	switch p.sizingModel {
	case domain.FixedFractional:
		return p.fixedFractionalSize(accountSize, riskPercent, entry, stop)
	case domain.VolatilityAdjusted:
		// For now, fallback to fixed fractional (volatility adjustment would require ATR data)
		return p.fixedFractionalSize(accountSize, riskPercent, entry, stop)
	default:
		return p.fixedFractionalSize(accountSize, riskPercent, entry, stop)
	}
}

// fixedFractionalSize calculates position size using fixed fractional risk
func (p *PortfolioRiskManager) fixedFractionalSize(
	accountSize float64,
	riskPercent float64,
	entry float64,
	stop float64,
) (positionSize int, dollarRisk float64, riskPerUnit float64) {
	if entry == 0 || entry == stop {
		return 0, 0, 0
	}

	riskPerUnit = math.Abs(entry - stop)
	dollarRisk = accountSize * riskPercent
	positionSize = int(dollarRisk / riskPerUnit)

	return positionSize, dollarRisk, riskPerUnit
}

// UpdatePortfolioStateAfterTrade updates portfolio state after a trade is executed
func (p *PortfolioRiskManager) UpdatePortfolioStateAfterTrade(
	symbol string,
	sector string,
	positionValue float64,
	dollarRisk float64,
) {
	p.portfolioState.OpenPositions++
	p.portfolioState.TotalExposure += positionValue
	p.portfolioState.TotalRisk += dollarRisk

	if sector != "" {
		p.portfolioState.SectorExposure[sector] += positionValue
	}
}

// GetPortfolioState returns the current portfolio state
func (p *PortfolioRiskManager) GetPortfolioState() domain.PortfolioState {
	return p.portfolioState
}
