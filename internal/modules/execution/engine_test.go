package execution

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestCalculatePositionSize(t *testing.T) {
	tests := []struct {
		name          string
		signal        Signal
		account       AccountInfo
		riskParams    RiskParameters
		expectedQty   int
		expectedError bool
	}{
		{
			name: "normal long position",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   145.0,
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    50000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   133, // ($100k * 1%) / ($150 - $145) = 200, but capped by 20% max position value ($20k / $150 = 133)
			expectedError: false,
		},
		{
			name: "normal short position",
			signal: Signal{
				Symbol:     "TSLA",
				SignalType: "SELL",
				EntryPrice: 200.0,
				StopLoss:   210.0,
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    50000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   100, // ($100k * 1%) / ($210 - $200) = 100
			expectedError: false,
		},
		{
			name: "position capped by max position value",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   149.5, // tiny stop distance
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    50000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   133, // capped by 20% of $100k = $20k / $150 = 133
			expectedError: false,
		},
		{
			name: "position capped by buying power",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   145.0,
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    15000, // limited buying power
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   100, // capped by $15k buying power / $150 = 100
			expectedError: false,
		},
		{
			name: "position below minimum",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   120.0, // large stop distance
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    50000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     50,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   50, // floored to min position size
			expectedError: false,
		},
		{
			name: "position exceeds maximum",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   149.9, // very tiny stop
			},
			account: AccountInfo{
				NetLiquidation: 1000000, // large account
				BuyingPower:    500000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     500,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   500, // capped to max position size
			expectedError: false,
		},
		{
			name: "zero stop distance",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   150.0, // no stop distance
			},
			account: AccountInfo{
				NetLiquidation: 100000,
				BuyingPower:    50000,
			},
			riskParams: RiskParameters{
				MaxRiskPerTrade:     0.01,
				MinPositionSize:     1,
				MaxPositionSize:     1000,
				MaxPositionValuePct: 0.20,
			},
			expectedQty:   0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.riskParams)
			qty, err := engine.CalculatePositionSize(tt.signal, tt.account)

			if tt.expectedError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectedError && qty != tt.expectedQty {
				t.Errorf("expected quantity %d, got %d", tt.expectedQty, qty)
			}
		})
	}
}

func TestValidateSignal(t *testing.T) {
	engine := NewEngine(RiskParameters{})

	tests := []struct {
		name          string
		signal        Signal
		expectedError bool
	}{
		{
			name: "valid BUY signal",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   145.0,
				TakeProfit: 160.0,
			},
			expectedError: false,
		},
		{
			name: "valid SELL signal",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "TSLA",
				SignalType: "SELL",
				EntryPrice: 200.0,
				StopLoss:   210.0,
				TakeProfit: 180.0,
			},
			expectedError: false,
		},
		{
			name: "missing symbol",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   145.0,
			},
			expectedError: true,
		},
		{
			name: "invalid signal type",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "AAPL",
				SignalType: "HOLD",
				EntryPrice: 150.0,
				StopLoss:   145.0,
			},
			expectedError: true,
		},
		{
			name: "zero entry price",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 0.0,
				StopLoss:   145.0,
			},
			expectedError: true,
		},
		{
			name: "BUY with stop >= entry",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
				StopLoss:   155.0,
			},
			expectedError: true,
		},
		{
			name: "SELL with stop <= entry",
			signal: Signal{
				ID:         uuid.New(),
				Symbol:     "TSLA",
				SignalType: "SELL",
				EntryPrice: 200.0,
				StopLoss:   195.0,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateSignal(tt.signal)
			if tt.expectedError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateOrderRequest(t *testing.T) {
	engine := NewEngine(RiskParameters{})

	tests := []struct {
		name              string
		signal            Signal
		quantity          int
		orderType         string
		expectedAction    string
		expectedOrderType string
		expectLimitPrice  bool
		expectStopPrice   bool
	}{
		{
			name: "BUY limit order",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
			},
			quantity:          100,
			orderType:         "LMT",
			expectedAction:    "BUY",
			expectedOrderType: "LMT",
			expectLimitPrice:  true,
			expectStopPrice:   false,
		},
		{
			name: "SELL limit order",
			signal: Signal{
				Symbol:     "TSLA",
				SignalType: "SELL",
				EntryPrice: 200.0,
			},
			quantity:          50,
			orderType:         "LMT",
			expectedAction:    "SELL",
			expectedOrderType: "LMT",
			expectLimitPrice:  true,
			expectStopPrice:   false,
		},
		{
			name: "BUY market order",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
			},
			quantity:          100,
			orderType:         "MKT",
			expectedAction:    "BUY",
			expectedOrderType: "MKT",
			expectLimitPrice:  false,
			expectStopPrice:   false,
		},
		{
			name: "BUY stop order",
			signal: Signal{
				Symbol:     "AAPL",
				SignalType: "BUY",
				EntryPrice: 150.0,
			},
			quantity:          100,
			orderType:         "STP",
			expectedAction:    "BUY",
			expectedOrderType: "STP",
			expectLimitPrice:  false,
			expectStopPrice:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := engine.CreateOrderRequest(tt.signal, tt.quantity, tt.orderType)

			if order.Symbol != tt.signal.Symbol {
				t.Errorf("expected symbol %s, got %s", tt.signal.Symbol, order.Symbol)
			}
			if order.Action != tt.expectedAction {
				t.Errorf("expected action %s, got %s", tt.expectedAction, order.Action)
			}
			if order.Quantity != tt.quantity {
				t.Errorf("expected quantity %d, got %d", tt.quantity, order.Quantity)
			}
			if order.OrderType != tt.expectedOrderType {
				t.Errorf("expected order type %s, got %s", tt.expectedOrderType, order.OrderType)
			}

			if tt.expectLimitPrice && order.LimitPrice == nil {
				t.Errorf("expected limit price, got nil")
			}
			if !tt.expectLimitPrice && order.LimitPrice != nil {
				t.Errorf("expected no limit price, got %v", *order.LimitPrice)
			}
			if tt.expectStopPrice && order.StopPrice == nil {
				t.Errorf("expected stop price, got nil")
			}
			if !tt.expectStopPrice && order.StopPrice != nil {
				t.Errorf("expected no stop price, got %v", *order.StopPrice)
			}
		})
	}
}

func TestCalculateRiskRewardRatio(t *testing.T) {
	engine := NewEngine(RiskParameters{})

	tests := []struct {
		name     string
		signal   Signal
		expected float64
	}{
		{
			name: "BUY signal with 2:1 R:R",
			signal: Signal{
				EntryPrice: 100.0,
				StopLoss:   95.0,
				TakeProfit: 110.0,
			},
			expected: 2.0, // reward $10 / risk $5 = 2.0
		},
		{
			name: "SELL signal with 3:1 R:R",
			signal: Signal{
				EntryPrice: 200.0,
				StopLoss:   210.0,
				TakeProfit: 170.0,
			},
			expected: 3.0, // reward $30 / risk $10 = 3.0
		},
		{
			name: "BUY signal with 1:1 R:R",
			signal: Signal{
				EntryPrice: 50.0,
				StopLoss:   45.0,
				TakeProfit: 55.0,
			},
			expected: 1.0, // reward $5 / risk $5 = 1.0
		},
		{
			name: "zero risk distance",
			signal: Signal{
				EntryPrice: 100.0,
				StopLoss:   100.0,
				TakeProfit: 110.0,
			},
			expected: 0.0, // risk distance is zero, returns 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := engine.CalculateRiskRewardRatio(tt.signal)
			if !math.IsInf(tt.expected, 0) && math.Abs(ratio-tt.expected) > 0.01 {
				t.Errorf("expected R:R %.2f, got %.2f", tt.expected, ratio)
			}
			if math.IsInf(tt.expected, 1) && !math.IsInf(ratio, 1) {
				t.Errorf("expected R:R +Inf, got %.2f", ratio)
			}
		})
	}
}
