package app

import (
	"context"
	"testing"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

func TestPortfolioRiskManager_ValidatePosition(t *testing.T) {
	constraints := domain.PortfolioConstraints{
		MaxPositionSize:       10000,
		MaxPositions:          5,
		MaxSectorExposure:     0.30,
		MaxPortfolioRisk:      0.15,
		MaxDrawdown:           0.20,
		MinAccountSize:        5000,
		MaxCorrelatedExposure: 0.40,
	}

	positionLimits := domain.PositionLimits{
		MaxRiskPerTrade: 0.02,
		MinRiskPerTrade: 0.005,
		MaxLeverage:     2.0,
		MinStopDistance: 0.01,
		MaxStopDistance: 0.10,
	}

	tests := []struct {
		name          string
		accountSize   float64
		openPositions int
		drawdown      float64
		symbol        string
		sector        string
		entry         float64
		stop          float64
		riskPercent   float64
		wantAllowed   bool
		wantViolation string
	}{
		{
			name:          "valid trade within all limits",
			accountSize:   10000,
			openPositions: 2,
			drawdown:      0.05,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.01,
			wantAllowed:   true,
		},
		{
			name:          "account size too small",
			accountSize:   3000,
			openPositions: 0,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.01,
			wantAllowed:   false,
			wantViolation: "account size",
		},
		{
			name:          "max drawdown exceeded",
			accountSize:   10000,
			openPositions: 2,
			drawdown:      0.25,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.01,
			wantAllowed:   false,
			wantViolation: "drawdown",
		},
		{
			name:          "max positions reached",
			accountSize:   10000,
			openPositions: 5,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.01,
			wantAllowed:   false,
			wantViolation: "max positions",
		},
		{
			name:          "stop too tight",
			accountSize:   10000,
			openPositions: 0,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          149.50,
			riskPercent:   0.01,
			wantAllowed:   false,
			wantViolation: "stop too tight",
		},
		{
			name:          "stop too wide",
			accountSize:   10000,
			openPositions: 0,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          130.00,
			riskPercent:   0.01,
			wantAllowed:   false,
			wantViolation: "stop too wide",
		},
		{
			name:          "risk too small",
			accountSize:   10000,
			openPositions: 0,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.001,
			wantAllowed:   false,
			wantViolation: "risk too small",
		},
		{
			name:          "risk too large",
			accountSize:   10000,
			openPositions: 0,
			drawdown:      0,
			symbol:        "AAPL",
			sector:        "Technology",
			entry:         150.00,
			stop:          145.00,
			riskPercent:   0.05,
			wantAllowed:   false,
			wantViolation: "risk too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewPortfolioRiskManager(constraints, positionLimits, nil)
			manager.SetPortfolioState(domain.PortfolioState{
				AccountSize:     tt.accountSize,
				OpenPositions:   tt.openPositions,
				CurrentDrawdown: tt.drawdown,
				SectorExposure:  make(map[string]float64),
			})

			result := manager.ValidatePosition(
				context.Background(),
				tt.symbol,
				tt.sector,
				tt.entry,
				tt.stop,
				tt.riskPercent,
			)

			if result.Allowed != tt.wantAllowed {
				t.Errorf("ValidatePosition() allowed = %v, want %v", result.Allowed, tt.wantAllowed)
				t.Logf("Violations: %v", result.Violations)
			}

			if !tt.wantAllowed && tt.wantViolation != "" {
				found := false
				for _, v := range result.Violations {
					if contains(v, tt.wantViolation) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected violation containing '%s', got: %v", tt.wantViolation, result.Violations)
				}
			}
		})
	}
}

func TestPortfolioRiskManager_CalculatePositionSize(t *testing.T) {
	constraints := domain.PortfolioConstraints{
		MaxPositionSize: 10000,
		MaxPositions:    5,
	}

	positionLimits := domain.PositionLimits{
		MaxRiskPerTrade: 0.02,
		MinRiskPerTrade: 0.005,
		MaxLeverage:     2.0,
		MinStopDistance: 0.01,
		MaxStopDistance: 0.10,
	}

	manager := NewPortfolioRiskManager(constraints, positionLimits, nil)

	tests := []struct {
		name            string
		accountSize     float64
		riskPercent     float64
		entry           float64
		stop            float64
		wantSize        int
		wantDollarRisk  float64
		wantRiskPerUnit float64
	}{
		{
			name:            "standard position",
			accountSize:     10000,
			riskPercent:     0.01,
			entry:           100.00,
			stop:            95.00,
			wantSize:        20,
			wantDollarRisk:  100,
			wantRiskPerUnit: 5.00,
		},
		{
			name:            "higher risk percentage",
			accountSize:     10000,
			riskPercent:     0.02,
			entry:           100.00,
			stop:            95.00,
			wantSize:        40,
			wantDollarRisk:  200,
			wantRiskPerUnit: 5.00,
		},
		{
			name:            "entry equals stop",
			accountSize:     10000,
			riskPercent:     0.01,
			entry:           100.00,
			stop:            100.00,
			wantSize:        0,
			wantDollarRisk:  0,
			wantRiskPerUnit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, dollarRisk, riskPerUnit := manager.calculatePositionSize(
				tt.accountSize, tt.riskPercent, tt.entry, tt.stop)

			if size != tt.wantSize {
				t.Errorf("positionSize = %d, want %d", size, tt.wantSize)
			}
			if dollarRisk != tt.wantDollarRisk {
				t.Errorf("dollarRisk = %.2f, want %.2f", dollarRisk, tt.wantDollarRisk)
			}
			if riskPerUnit != tt.wantRiskPerUnit {
				t.Errorf("riskPerUnit = %.2f, want %.2f", riskPerUnit, tt.wantRiskPerUnit)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}
