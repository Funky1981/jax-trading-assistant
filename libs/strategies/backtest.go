package strategies

import (
	"context"
	"fmt"
	"time"
)

// BacktestResult represents the results of a single backtest run
type BacktestResult struct {
	StrategyID     string
	Symbol         string
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital float64
	FinalCapital   float64

	// Performance metrics
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        float64
	TotalReturn    float64
	TotalReturnPct float64
	MaxDrawdown    float64
	SharpeRatio    float64
	ProfitFactor   float64

	// Trade statistics
	AvgWin      float64
	AvgLoss     float64
	AvgR        float64
	LargestWin  float64
	LargestLoss float64

	// Trades executed
	Trades []BacktestTrade
}

// BacktestTrade represents a single trade in a backtest
type BacktestTrade struct {
	Symbol     string
	EntryDate  time.Time
	ExitDate   time.Time
	EntryPrice float64
	ExitPrice  float64
	StopLoss   float64
	TakeProfit []float64
	Quantity   int
	Direction  SignalType // Buy or Sell
	PnL        float64
	PnLPct     float64
	RMultiple  float64
	ExitReason string
	Confidence float64
}

// Backtester runs historical backtests on strategies
type Backtester struct {
	registry       *Registry
	initialCapital float64
	riskPerTrade   float64
	maxPositions   int
}

// NewBacktester creates a new backtester
func NewBacktester(registry *Registry) *Backtester {
	return &Backtester{
		registry:       registry,
		initialCapital: 100000.0, // $100k default
		riskPerTrade:   0.01,     // 1% risk per trade
		maxPositions:   5,        // Max 5 concurrent positions
	}
}

// WithCapital sets the initial capital
func (b *Backtester) WithCapital(capital float64) *Backtester {
	b.initialCapital = capital
	return b
}

// WithRiskPerTrade sets the risk per trade as a percentage
func (b *Backtester) WithRiskPerTrade(risk float64) *Backtester {
	b.riskPerTrade = risk
	return b
}

// WithMaxPositions sets the maximum concurrent positions
func (b *Backtester) WithMaxPositions(max int) *Backtester {
	b.maxPositions = max
	return b
}

// BacktestConfig holds configuration for a backtest run
type BacktestConfig struct {
	StrategyID string
	Symbols    []string
	StartDate  time.Time
	EndDate    time.Time
	DataSource HistoricalDataSource
}

// HistoricalDataSource provides historical market data for backtesting
type HistoricalDataSource interface {
	GetCandles(ctx context.Context, symbol string, start, end time.Time) ([]Candle, error)
	GetIndicators(ctx context.Context, symbol string, timestamp time.Time) (AnalysisInput, error)
}

// Candle represents a single price bar
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
}

// Run executes a backtest
func (b *Backtester) Run(ctx context.Context, config BacktestConfig) (*BacktestResult, error) {
	strategy, err := b.registry.Get(config.StrategyID)
	if err != nil {
		return nil, fmt.Errorf("strategy not found: %w", err)
	}

	result := &BacktestResult{
		StrategyID:     config.StrategyID,
		StartDate:      config.StartDate,
		EndDate:        config.EndDate,
		InitialCapital: b.initialCapital,
		FinalCapital:   b.initialCapital,
		Trades:         make([]BacktestTrade, 0),
	}

	capital := b.initialCapital
	activePositions := make(map[string]*BacktestTrade)

	// Run backtest for each symbol
	for _, symbol := range config.Symbols {
		candles, err := config.DataSource.GetCandles(ctx, symbol, config.StartDate, config.EndDate)
		if err != nil {
			return nil, fmt.Errorf("failed to get candles for %s: %w", symbol, err)
		}

		// Process each candle
		for _, candle := range candles {
			// Check for exits first
			b.checkExits(activePositions, candle, &capital, result)

			// Skip if max positions reached
			if len(activePositions) >= b.maxPositions {
				continue
			}

			// Get analysis input with indicators
			input, err := config.DataSource.GetIndicators(ctx, symbol, candle.Timestamp)
			if err != nil {
				continue // Skip if indicators not available
			}

			// Get signal from strategy
			signal, err := strategy.Analyze(ctx, input)
			if err != nil {
				continue
			}

			// Only enter if signal is buy or sell (not hold)
			if signal.Type == SignalHold {
				continue
			}

			// Calculate position size based on risk
			riskAmount := capital * b.riskPerTrade
			stopDistance := signal.EntryPrice - signal.StopLoss
			if signal.Type == SignalSell {
				stopDistance = signal.StopLoss - signal.EntryPrice
			}

			if stopDistance <= 0 {
				continue // Invalid stop distance
			}

			quantity := int(riskAmount / stopDistance)
			if quantity <= 0 {
				continue
			}

			// Enter trade
			trade := BacktestTrade{
				Symbol:     symbol,
				EntryDate:  candle.Timestamp,
				EntryPrice: signal.EntryPrice,
				StopLoss:   signal.StopLoss,
				TakeProfit: signal.TakeProfit,
				Quantity:   quantity,
				Direction:  signal.Type,
				Confidence: signal.Confidence,
			}

			activePositions[symbol] = &trade
		}
	}

	// Close any remaining positions at end date
	b.closeRemainingPositions(activePositions, config.EndDate, &capital, result)

	// Calculate final metrics
	result.FinalCapital = capital
	b.calculateMetrics(result)

	return result, nil
}

func (b *Backtester) checkExits(positions map[string]*BacktestTrade, candle Candle, capital *float64, result *BacktestResult) {
	trade, exists := positions[candle.Symbol]
	if !exists {
		return
	}

	exitPrice := 0.0
	exitReason := ""

	if trade.Direction == SignalBuy {
		// Check stop loss
		if candle.Low <= trade.StopLoss {
			exitPrice = trade.StopLoss
			exitReason = "Stop loss hit"
		}
		// Check take profit
		for _, tp := range trade.TakeProfit {
			if candle.High >= tp {
				exitPrice = tp
				exitReason = "Take profit hit"
				break
			}
		}
	} else if trade.Direction == SignalSell {
		// Check stop loss
		if candle.High >= trade.StopLoss {
			exitPrice = trade.StopLoss
			exitReason = "Stop loss hit"
		}
		// Check take profit
		for _, tp := range trade.TakeProfit {
			if candle.Low <= tp {
				exitPrice = tp
				exitReason = "Take profit hit"
				break
			}
		}
	}

	if exitPrice > 0 {
		trade.ExitDate = candle.Timestamp
		trade.ExitPrice = exitPrice
		trade.ExitReason = exitReason

		// Calculate PnL
		if trade.Direction == SignalBuy {
			trade.PnL = (exitPrice - trade.EntryPrice) * float64(trade.Quantity)
		} else {
			trade.PnL = (trade.EntryPrice - exitPrice) * float64(trade.Quantity)
		}

		trade.PnLPct = trade.PnL / (trade.EntryPrice * float64(trade.Quantity))

		// Calculate R-multiple
		riskAmount := trade.EntryPrice - trade.StopLoss
		if trade.Direction == SignalSell {
			riskAmount = trade.StopLoss - trade.EntryPrice
		}
		if riskAmount > 0 {
			trade.RMultiple = (exitPrice - trade.EntryPrice) / riskAmount
			if trade.Direction == SignalSell {
				trade.RMultiple = (trade.EntryPrice - exitPrice) / riskAmount
			}
		}

		*capital += trade.PnL
		result.Trades = append(result.Trades, *trade)
		delete(positions, candle.Symbol)
	}
}

func (b *Backtester) closeRemainingPositions(positions map[string]*BacktestTrade, endDate time.Time, capital *float64, result *BacktestResult) {
	for _, trade := range positions {
		trade.ExitDate = endDate
		trade.ExitPrice = trade.EntryPrice // Close at entry (no profit/loss)
		trade.ExitReason = "End of backtest period"
		trade.PnL = 0
		trade.PnLPct = 0
		trade.RMultiple = 0

		result.Trades = append(result.Trades, *trade)
	}
}

func (b *Backtester) calculateMetrics(result *BacktestResult) {
	result.TotalTrades = len(result.Trades)
	if result.TotalTrades == 0 {
		return
	}

	totalWin := 0.0
	totalLoss := 0.0
	totalRMultiple := 0.0
	maxDrawdown := 0.0
	peak := result.InitialCapital
	runningCapital := result.InitialCapital

	for _, trade := range result.Trades {
		if trade.PnL > 0 {
			result.WinningTrades++
			totalWin += trade.PnL
			if trade.PnL > result.LargestWin {
				result.LargestWin = trade.PnL
			}
		} else if trade.PnL < 0 {
			result.LosingTrades++
			totalLoss += -trade.PnL
			if trade.PnL < result.LargestLoss {
				result.LargestLoss = trade.PnL
			}
		}

		totalRMultiple += trade.RMultiple

		// Track drawdown
		runningCapital += trade.PnL
		if runningCapital > peak {
			peak = runningCapital
		}
		drawdown := (peak - runningCapital) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades)
	result.TotalReturn = result.FinalCapital - result.InitialCapital
	result.TotalReturnPct = result.TotalReturn / result.InitialCapital
	result.MaxDrawdown = maxDrawdown

	if result.WinningTrades > 0 {
		result.AvgWin = totalWin / float64(result.WinningTrades)
	}
	if result.LosingTrades > 0 {
		result.AvgLoss = totalLoss / float64(result.LosingTrades)
	}
	result.AvgR = totalRMultiple / float64(result.TotalTrades)

	// Profit factor
	if totalLoss > 0 {
		result.ProfitFactor = totalWin / totalLoss
	}

	// Sharpe ratio (simplified - assumes daily returns)
	if result.TotalTrades > 1 {
		returns := make([]float64, len(result.Trades))
		for i, trade := range result.Trades {
			returns[i] = trade.PnLPct
		}
		mean, stdDev := calculateMeanStdDev(returns)
		if stdDev > 0 {
			result.SharpeRatio = (mean / stdDev) * 15.87 // Annualized (sqrt(252))
		}
	}
}

func calculateMeanStdDev(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))
	stdDev = 0.0
	if variance > 0 {
		stdDev = 1.0
		for i := 0; i < 10; i++ { // Simple square root approximation
			stdDev = (stdDev + variance/stdDev) / 2
		}
	}

	return mean, stdDev
}
