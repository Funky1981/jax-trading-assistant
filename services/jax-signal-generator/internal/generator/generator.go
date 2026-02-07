package generator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"jax-trading-assistant/libs/strategies"
	"jax-trading-assistant/services/jax-signal-generator/internal/orchestrator"

	"github.com/google/uuid"
)

// Generator generates trading signals by running strategies against market data
type Generator struct {
	db                  *sql.DB
	registry            *strategies.Registry
	symbols             []string
	metricsCallback     func(generated int, failed int, duration time.Duration)
	orchestratorClient  *orchestrator.Client
	confidenceThreshold float64
}

// New creates a new signal generator
func New(db *sql.DB, registry *strategies.Registry, symbols []string, callback func(int, int, time.Duration)) *Generator {
	return &Generator{
		db:                  db,
		registry:            registry,
		symbols:             symbols,
		metricsCallback:     callback,
		confidenceThreshold: 0.75, // Default threshold for auto-orchestration
	}
}

// WithOrchestrator adds orchestrator integration
func (g *Generator) WithOrchestrator(client *orchestrator.Client) *Generator {
	g.orchestratorClient = client
	return g
}

// WithConfidenceThreshold sets the confidence threshold for auto-orchestration
func (g *Generator) WithConfidenceThreshold(threshold float64) *Generator {
	g.confidenceThreshold = threshold
	return g
}

// Generate runs all strategies against all symbols and stores signals
func (g *Generator) Generate(ctx context.Context) error {
	startTime := time.Now()
	generated := 0
	failed := 0

	// Cleanup expired signals first
	if err := g.cleanupExpiredSignals(ctx); err != nil {
		log.Printf("warning: failed to cleanup expired signals: %v", err)
	}

	strategyIDs := g.registry.List()

	for _, symbol := range g.symbols {
		// Fetch latest market data for this symbol
		input, err := g.fetchAnalysisInput(ctx, symbol)
		if err != nil {
			log.Printf("failed to fetch data for %s: %v", symbol, err)
			failed++
			continue
		}

		// Run each strategy
		for _, strategyID := range strategyIDs {
			strategy, err := g.registry.Get(strategyID)
			if err != nil {
				log.Printf("failed to get strategy %s: %v", strategyID, err)
				continue
			}

			signal, err := strategy.Analyze(ctx, input)
			if err != nil {
				log.Printf("strategy %s failed for %s: %v", strategyID, symbol, err)
				failed++
				continue
			}

			// Only store actionable signals (buy/sell) with sufficient confidence
			if signal.Type != strategies.SignalHold && signal.Confidence >= 0.6 {
				signalID, err := g.storeSignal(ctx, strategyID, signal)
				if err != nil {
					log.Printf("failed to store signal for %s/%s: %v", symbol, strategyID, err)
					failed++
					continue
				}
				generated++
				log.Printf("generated %s signal for %s (strategy: %s, confidence: %.2f)",
					signal.Type, symbol, strategyID, signal.Confidence)

				// Auto-trigger orchestration for high-confidence signals
				if g.orchestratorClient != nil && signal.Confidence >= g.confidenceThreshold {
					context := g.buildOrchestrationContext(strategyID, signal)
					runID, err := g.orchestratorClient.TriggerOrchestration(ctx, signalID, symbol, context)
					if err != nil {
						log.Printf("failed to trigger orchestration for signal %s: %v", signalID, err)
					} else {
						// Update signal with orchestration run ID
						if err := g.linkOrchestration(ctx, signalID, runID); err != nil {
							log.Printf("failed to link orchestration run %s to signal %s: %v", runID, signalID, err)
						} else {
							log.Printf("triggered orchestration run %s for signal %s (confidence: %.2f)", runID, signalID, signal.Confidence)
						}
					}
				}
			}
		}
	}

	duration := time.Since(startTime)
	if g.metricsCallback != nil {
		g.metricsCallback(generated, failed, duration)
	}

	return nil
}

// fetchAnalysisInput gets latest market data and calculates indicators
func (g *Generator) fetchAnalysisInput(ctx context.Context, symbol string) (strategies.AnalysisInput, error) {
	input := strategies.AnalysisInput{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}

	// Fetch latest quote for current price
	query := `
		SELECT price, volume 
		FROM quotes 
		WHERE symbol = $1 
		ORDER BY timestamp DESC 
		LIMIT 1
	`
	var volume sql.NullInt64
	err := g.db.QueryRowContext(ctx, query, symbol).Scan(&input.Price, &volume)
	if err != nil {
		return input, fmt.Errorf("failed to fetch latest quote: %w", err)
	}
	if volume.Valid {
		input.Volume = volume.Int64
	}

	// Fetch recent candles for indicator calculation (need at least 200 for SMA200)
	candles, err := g.fetchRecentCandles(ctx, symbol, 250)
	if err != nil {
		return input, fmt.Errorf("failed to fetch candles: %w", err)
	}

	if len(candles) < 50 {
		return input, fmt.Errorf("insufficient data: only %d candles available", len(candles))
	}

	// Calculate technical indicators
	input.RSI = calculateRSI(candles, 14)
	input.MACD = calculateMACD(candles)
	input.SMA20 = calculateSMA(candles, 20)
	input.SMA50 = calculateSMA(candles, 50)
	if len(candles) >= 200 {
		input.SMA200 = calculateSMA(candles, 200)
	}
	input.ATR = calculateATR(candles, 14)
	input.BollingerBands = calculateBollingerBands(candles, 20, 2.0)

	// Calculate average volume
	input.AvgVolume20 = calculateAvgVolume(candles, 20)

	// Determine market trend
	input.MarketTrend = determineTrend(input)

	return input, nil
}

type candle struct {
	timestamp time.Time
	open      float64
	high      float64
	low       float64
	close     float64
	volume    int64
}

func (g *Generator) fetchRecentCandles(ctx context.Context, symbol string, limit int) ([]candle, error) {
	query := `
		SELECT timestamp, open, high, low, close, volume
		FROM candles
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := g.db.QueryContext(ctx, query, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []candle
	for rows.Next() {
		var c candle
		var vol sql.NullInt64
		if err := rows.Scan(&c.timestamp, &c.open, &c.high, &c.low, &c.close, &vol); err != nil {
			return nil, err
		}
		c.volume = vol.Int64
		candles = append(candles, c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return candles, nil
}

// storeSignal stores a new signal in the database and returns the signal ID
func (g *Generator) storeSignal(ctx context.Context, strategyID string, signal strategies.Signal) (uuid.UUID, error) {
	query := `
		INSERT INTO strategy_signals 
		(symbol, strategy_id, signal_type, confidence, entry_price, stop_loss, take_profit, reasoning, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', NOW())
		RETURNING id
	`

	// Use the first take profit target (most strategies have primary target)
	var takeProfit float64
	if len(signal.TakeProfit) > 0 {
		takeProfit = signal.TakeProfit[0]
	}

	// Convert signal type to uppercase to match database constraint
	signalTypeUpper := string(signal.Type)
	if signalTypeUpper == "buy" {
		signalTypeUpper = "BUY"
	} else if signalTypeUpper == "sell" {
		signalTypeUpper = "SELL"
	} else if signalTypeUpper == "hold" {
		signalTypeUpper = "HOLD"
	}

	var signalID uuid.UUID
	err := g.db.QueryRowContext(ctx, query,
		signal.Symbol,
		strategyID,
		signalTypeUpper,
		signal.Confidence,
		signal.EntryPrice,
		signal.StopLoss,
		takeProfit,
		signal.Reason,
	).Scan(&signalID)

	if err != nil {
		return uuid.Nil, err
	}

	return signalID, nil
}

// linkOrchestration updates a signal with its orchestration run ID
func (g *Generator) linkOrchestration(ctx context.Context, signalID, runID uuid.UUID) error {
	query := `
		UPDATE strategy_signals 
		SET orchestration_run_id = $1
		WHERE id = $2
	`
	_, err := g.db.ExecContext(ctx, query, runID, signalID)
	return err
}

// buildOrchestrationContext creates a detailed context string for Agent0
func (g *Generator) buildOrchestrationContext(strategyID string, signal strategies.Signal) string {
	return fmt.Sprintf(`New trading signal detected:
Symbol: %s
Strategy: %s
Signal Type: %s
Entry Price: $%.2f
Stop Loss: $%.2f
Take Profit: $%.2f
Confidence: %.2f%%
Reasoning: %s

Please analyze this signal and provide:
1. Your recommendation (BUY/SELL/PASS)
2. Confidence level (0-1)
3. Detailed reasoning
4. Risk assessment
5. Suggested position size`,
		signal.Symbol,
		strategyID,
		signal.Type,
		signal.EntryPrice,
		signal.StopLoss,
		signal.TakeProfit[0],
		signal.Confidence*100,
		signal.Reason,
	)
}

// cleanupExpiredSignals removes old pending signals (>24 hours)
func (g *Generator) cleanupExpiredSignals(ctx context.Context) error {
	query := `
		UPDATE strategy_signals 
		SET status = 'expired'
		WHERE status = 'pending' 
		AND created_at < NOW() - INTERVAL '24 hours'
	`
	result, err := g.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		log.Printf("expired %d old signals", rows)
	}

	return nil
}

func determineTrend(input strategies.AnalysisInput) string {
	if input.SMA200 > 0 {
		if input.Price > input.SMA50 && input.SMA50 > input.SMA200 {
			return "bullish"
		} else if input.Price < input.SMA50 && input.SMA50 < input.SMA200 {
			return "bearish"
		}
	}
	return "neutral"
}
