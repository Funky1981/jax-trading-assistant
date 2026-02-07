package executor

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// Signal represents a trading signal to execute
type Signal struct {
	ID         uuid.UUID
	Symbol     string
	SignalType string // BUY or SELL
	EntryPrice float64
	StopLoss   float64
	TakeProfit float64
	StrategyID string
	Confidence float64
}

// AccountInfo contains account balance and risk parameters
type AccountInfo struct {
	NetLiquidation float64
	BuyingPower    float64
	Currency       string
}

// RiskParameters defines risk management rules
type RiskParameters struct {
	MaxRiskPerTrade     float64 // Percentage of account to risk per trade (e.g., 0.01 = 1%)
	MinPositionSize     int     // Minimum shares to trade
	MaxPositionSize     int     // Maximum shares to trade
	MaxPositionValuePct float64 // Max position value as % of account (e.g., 0.2 = 20%)
}

// OrderRequest represents an order to be placed
type OrderRequest struct {
	Symbol     string
	Action     string // BUY or SELL
	Quantity   int
	OrderType  string // MKT, LMT, STP
	LimitPrice *float64
	StopPrice  *float64
}

// TradeResult represents the result of trade execution
type TradeResult struct {
	TradeID       uuid.UUID
	SignalID      uuid.UUID
	OrderID       string
	Symbol        string
	Direction     string
	Quantity      int
	EntryPrice    float64
	StopLoss      float64
	TakeProfit    float64
	StrategyID    string
	Status        string // pending, filled, rejected, cancelled
	RiskAmount    float64
	RiskPercent   float64
	PositionValue float64
	SubmittedAt   time.Time
	Error         string
}

// Executor handles trade execution logic
type Executor struct {
	riskParams RiskParameters
}

// NewExecutor creates a new trade executor
func NewExecutor(riskParams RiskParameters) *Executor {
	// Set defaults if not provided
	if riskParams.MaxRiskPerTrade == 0 {
		riskParams.MaxRiskPerTrade = 0.01 // 1% default
	}
	if riskParams.MinPositionSize == 0 {
		riskParams.MinPositionSize = 1
	}
	if riskParams.MaxPositionValuePct == 0 {
		riskParams.MaxPositionValuePct = 0.2 // 20% default
	}

	return &Executor{
		riskParams: riskParams,
	}
}

// CalculatePositionSize calculates the number of shares to trade based on risk parameters
func (e *Executor) CalculatePositionSize(signal Signal, account AccountInfo) (int, error) {
	// Calculate risk amount (dollars at risk)
	riskAmount := account.NetLiquidation * e.riskParams.MaxRiskPerTrade

	// Calculate stop distance (dollars per share)
	stopDistance := math.Abs(signal.EntryPrice - signal.StopLoss)
	if stopDistance == 0 {
		return 0, fmt.Errorf("invalid stop loss: stop distance is zero")
	}

	// Calculate shares based on risk
	shares := int(riskAmount / stopDistance)

	// Apply minimum position size
	if shares < e.riskParams.MinPositionSize {
		shares = e.riskParams.MinPositionSize
	}

	// Apply maximum position size
	if e.riskParams.MaxPositionSize > 0 && shares > e.riskParams.MaxPositionSize {
		shares = e.riskParams.MaxPositionSize
	}

	// Apply maximum position value constraint
	positionValue := float64(shares) * signal.EntryPrice
	maxPositionValue := account.NetLiquidation * e.riskParams.MaxPositionValuePct
	if positionValue > maxPositionValue {
		shares = int(maxPositionValue / signal.EntryPrice)
	}

	// Ensure we have at least minimum shares
	if shares < e.riskParams.MinPositionSize {
		return 0, fmt.Errorf("calculated position size (%d shares) is below minimum (%d shares)",
			shares, e.riskParams.MinPositionSize)
	}

	// Check buying power
	requiredCapital := float64(shares) * signal.EntryPrice
	if requiredCapital > account.BuyingPower {
		// Adjust to fit buying power
		shares = int(account.BuyingPower / signal.EntryPrice)
		if shares < e.riskParams.MinPositionSize {
			return 0, fmt.Errorf("insufficient buying power: need $%.2f, have $%.2f",
				requiredCapital, account.BuyingPower)
		}
	}

	return shares, nil
}

// CreateOrderRequest creates an order request from a signal and position size
func (e *Executor) CreateOrderRequest(signal Signal, quantity int, orderType string) OrderRequest {
	req := OrderRequest{
		Symbol:    signal.Symbol,
		Action:    signal.SignalType,
		Quantity:  quantity,
		OrderType: orderType,
	}

	// Set price based on order type
	if orderType == "LMT" {
		req.LimitPrice = &signal.EntryPrice
	} else if orderType == "STP" {
		req.StopPrice = &signal.EntryPrice
	}

	return req
}

// ValidateSignal validates that a signal has all required fields
func (e *Executor) ValidateSignal(signal Signal) error {
	if signal.Symbol == "" {
		return fmt.Errorf("signal missing symbol")
	}
	if signal.SignalType != "BUY" && signal.SignalType != "SELL" {
		return fmt.Errorf("invalid signal type: %s (must be BUY or SELL)", signal.SignalType)
	}
	if signal.EntryPrice <= 0 {
		return fmt.Errorf("invalid entry price: %.2f", signal.EntryPrice)
	}
	if signal.StopLoss <= 0 {
		return fmt.Errorf("invalid stop loss: %.2f", signal.StopLoss)
	}
	if signal.TakeProfit <= 0 {
		return fmt.Errorf("invalid take profit: %.2f", signal.TakeProfit)
	}

	// Validate stop loss is on correct side of entry
	if signal.SignalType == "BUY" && signal.StopLoss >= signal.EntryPrice {
		return fmt.Errorf("BUY signal stop loss (%.2f) must be below entry (%.2f)",
			signal.StopLoss, signal.EntryPrice)
	}
	if signal.SignalType == "SELL" && signal.StopLoss <= signal.EntryPrice {
		return fmt.Errorf("SELL signal stop loss (%.2f) must be above entry (%.2f)",
			signal.StopLoss, signal.EntryPrice)
	}

	return nil
}

// CalculateRiskRewardRatio calculates the risk:reward ratio
func (e *Executor) CalculateRiskRewardRatio(signal Signal) float64 {
	risk := math.Abs(signal.EntryPrice - signal.StopLoss)
	reward := math.Abs(signal.TakeProfit - signal.EntryPrice)

	if risk == 0 {
		return 0
	}

	return reward / risk
}
