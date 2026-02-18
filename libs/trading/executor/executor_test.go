package executor

import (
	"testing"

	"github.com/google/uuid"
)

func TestCalculatePositionSize(t *testing.T) {
	executor := NewExecutor(RiskParameters{
		MaxRiskPerTrade:     0.01, // 1%
		MinPositionSize:     1,
		MaxPositionSize:     1000,
		MaxPositionValuePct: 0.2, // 20%
	})

	account := AccountInfo{
		NetLiquidation: 100000, // $100k account
		BuyingPower:    50000,
		Currency:       "USD",
	}

	tests := []struct {
		name           string
		signal         Signal
		expectedShares int
		expectError    bool
	}{
		{
			name: "normal BUY signal",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.00,
				StopLoss:   145.00, // $5 risk per share
				TakeProfit: 160.00,
			},
			// Risk-based: $1000 / $5 = 200 shares, position = $30k.
			// Capped by MaxPositionValuePct (20% of $100k = $20k): 20000/150 = 133.
			expectedShares: 133,
			expectError:    false,
		},
		{
			name: "zero stop distance",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "TSLA",
				SignalType: "BUY",
				EntryPrice: 200.00,
				StopLoss:   200.00, // Same as entry
				TakeProfit: 210.00,
			},
			expectedShares: 0,
			expectError:    true,
		},
		{
			name: "exceeds max position value",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "EXPENSIVE",
				SignalType: "BUY",
				EntryPrice: 50000.00,
				StopLoss:   49000.00, // $1000 risk per share
				TakeProfit: 55000.00,
			},
			// Risk-based: $1000 / $1000 = 1 share, position = $50k.
			// Capped by MaxPositionValuePct (20% of $100k = $20k): 20000/50000 = 0 shares.
			// 0 < MinPositionSize(1) → error.
			expectedShares: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares, err := executor.CalculatePositionSize(tt.signal, account)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if shares != tt.expectedShares {
				t.Errorf("expected %d shares, got %d", tt.expectedShares, shares)
			}
		})
	}
}

func TestValidateSignal(t *testing.T) {
	executor := NewExecutor(RiskParameters{})

	tests := []struct {
		name        string
		signal      Signal
		expectError bool
	}{
		{
			name: "valid BUY signal",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.00,
				StopLoss:   145.00,
				TakeProfit: 160.00,
			},
			expectError: false,
		},
		{
			name: "invalid BUY signal - stop above entry",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.00,
				StopLoss:   155.00, // Stop above entry
				TakeProfit: 160.00,
			},
			expectError: true,
		},
		{
			name: "missing symbol",
			signal: Signal{
				Symbol:     "",
				SignalType: "BUY",
				EntryPrice: 150.00,
				StopLoss:   145.00,
				TakeProfit: 160.00,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateSignal(tt.signal)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCalculateRiskRewardRatio(t *testing.T) {
	executor := NewExecutor(RiskParameters{})

	signal := Signal{
		EntryPrice: 100.00,
		StopLoss:   95.00,  // $5 risk
		TakeProfit: 110.00, // $10 reward
	}

	ratio := executor.CalculateRiskRewardRatio(signal)
	expected := 2.0 // 2:1 R:R

	if ratio != expected {
		t.Errorf("expected R:R ratio of %.2f, got %.2f", expected, ratio)
	}
}
