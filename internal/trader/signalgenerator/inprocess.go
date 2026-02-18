package signalgenerator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"jax-trading-assistant/libs/contracts/converters"
	"jax-trading-assistant/libs/contracts/domain"
	"jax-trading-assistant/libs/strategies"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InProcessSignalGenerator implements the SignalGenerator interface using in-process strategy execution
type InProcessSignalGenerator struct {
	db       *pgxpool.Pool
	registry *strategies.Registry
}

// New creates a new in-process signal generator
func New(db *pgxpool.Pool, registry *strategies.Registry) *InProcessSignalGenerator {
	return &InProcessSignalGenerator{
		db:       db,
		registry: registry,
	}
}

// GenerateSignals implements services.SignalGenerator
func (g *InProcessSignalGenerator) GenerateSignals(ctx context.Context, symbols []string) ([]domain.Signal, error) {
	var signals []domain.Signal

	strategyIDs := g.registry.List()

	for _, symbol := range symbols {
		// Fetch latest market data for this symbol
		input, err := g.fetchAnalysisInput(ctx, symbol)
		if err != nil {
			log.Printf("failed to fetch data for %s: %v", symbol, err)
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
				continue
			}

			// Only return actionable signals (buy/sell) with sufficient confidence
			if signal.Type != strategies.SignalHold && signal.Confidence >= 0.6 {
				// Convert to domain signal
				domainSignal := converters.SignalToDomain(strategyID, signal)
				domainSignal.ID = uuid.New().String()

				// ADR-0012 Phase 4: Add artifact tracking
				if metadata, err := g.registry.GetMetadata(strategyID); err == nil && metadata.Extra != nil {
					if artifactID, ok := metadata.Extra["artifact_id"].(string); ok {
						domainSignal.ArtifactID = artifactID
					}
				}

				// Store signal in database
				if err := g.storeSignal(ctx, domainSignal); err != nil {
					log.Printf("failed to store signal for %s/%s: %v", symbol, strategyID, err)
					continue
				}

				signals = append(signals, domainSignal)
				log.Printf("generated %s signal for %s (strategy: %s, confidence: %.2f)",
					signal.Type, symbol, strategyID, signal.Confidence)
			}
		}
	}

	return signals, nil
}

// GetSignalHistory implements services.SignalGenerator
func (g *InProcessSignalGenerator) GetSignalHistory(ctx context.Context, symbol string, limit int) ([]domain.Signal, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT id, symbol, strategy_id, signal_type, confidence, 
		       entry_price, stop_loss, take_profit, reasoning, 
		       generated_at, status
		FROM strategy_signals
		WHERE symbol = $1
		ORDER BY generated_at DESC
		LIMIT $2
	`

	rows, err := g.db.Query(ctx, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query signal history: %w", err)
	}
	defer rows.Close()

	var signals []domain.Signal
	for rows.Next() {
		var sig domain.Signal
		var takeProfit sql.NullFloat64 // Database stores single value, not array
		var generatedAt time.Time
		var status string

		err := rows.Scan(
			&sig.ID,
			&sig.Symbol,
			&sig.StrategyID,
			&sig.Type,
			&sig.Confidence,
			&sig.EntryPrice,
			&sig.StopLoss,
			&takeProfit,
			&sig.Reason,
			&generatedAt,
			&status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal: %w", err)
		}

		sig.Timestamp = generatedAt

		// Convert single take_profit value to array (domain model expects []float64)
		if takeProfit.Valid {
			sig.TakeProfit = []float64{takeProfit.Float64}
		}

		signals = append(signals, sig)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating signal rows: %w", err)
	}

	return signals, nil
}

// Health implements services.SignalGenerator
func (g *InProcessSignalGenerator) Health(ctx context.Context) error {
	// Check database connectivity
	if g.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	if err := g.db.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check strategy registry
	if len(g.registry.List()) == 0 {
		return fmt.Errorf("no strategies registered")
	}

	return nil
}

// storeSignal persists a signal to the database
func (g *InProcessSignalGenerator) storeSignal(ctx context.Context, sig domain.Signal) error {
	// Extract first take_profit value (database schema expects single numeric value, not array)
	var takeProfit *float64
	if len(sig.TakeProfit) > 0 {
		takeProfit = &sig.TakeProfit[0]
	}

	// ADR-0012 Phase 4: Include artifact_id in signal storage
	var artifactIDPtr *uuid.UUID
	if sig.ArtifactID != "" {
		if artifactUUID, err := uuid.Parse(sig.ArtifactID); err == nil {
			artifactIDPtr = &artifactUUID
		}
	}

	query := `
		INSERT INTO strategy_signals 
		(id, symbol, strategy_id, signal_type, confidence, entry_price, 
		 stop_loss, take_profit, reasoning, generated_at, expires_at, status, artifact_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending', $12)
	`

	expiresAt := sig.Timestamp.Add(24 * time.Hour) // Signals expire after 24 hours

	_, err := g.db.Exec(ctx, query,
		sig.ID,
		sig.Symbol,
		sig.StrategyID,
		sig.Type,
		sig.Confidence,
		sig.EntryPrice,
		sig.StopLoss,
		takeProfit,
		sig.Reason,
		sig.Timestamp,
		expiresAt,
		artifactIDPtr,
	)

	if err != nil {
		return fmt.Errorf("failed to insert signal: %w", err)
	}

	return nil
}

// fetchAnalysisInput gets latest market data and calculates indicators
func (g *InProcessSignalGenerator) fetchAnalysisInput(ctx context.Context, symbol string) (strategies.AnalysisInput, error) {
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
	row := g.db.QueryRow(ctx, query, symbol)
	err := row.Scan(&input.Price, &volume)
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

func (g *InProcessSignalGenerator) fetchRecentCandles(ctx context.Context, symbol string, limit int) ([]candle, error) {
	query := `
		SELECT timestamp, open, high, low, close, volume
		FROM candles
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := g.db.Query(ctx, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query candles: %w", err)
	}
	defer rows.Close()

	var candles []candle
	for rows.Next() {
		var c candle
		if err := rows.Scan(&c.timestamp, &c.open, &c.high, &c.low, &c.close, &c.volume); err != nil {
			return nil, fmt.Errorf("failed to scan candle: %w", err)
		}
		candles = append(candles, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating candle rows: %w", err)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}

	return candles, nil
}

// Technical indicator calculations (extracted from jax-signal-generator for exact compatibility)

func calculateRSI(candles []candle, period int) float64 {
	if len(candles) < period+1 {
		return 50.0 // Neutral
	}

	var gains, losses float64
	for i := len(candles) - period; i < len(candles); i++ {
		change := candles[i].close - candles[i-1].close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	return rsi
}

func calculateMACD(candles []candle) strategies.MACD {
	if len(candles) < 26 {
		return strategies.MACD{}
	}

	ema12 := calculateEMA(candles, 12)
	ema26 := calculateEMA(candles, 26)
	macdValue := ema12 - ema26

	// Calculate signal line (9-period EMA of MACD)
	// For simplicity, we'll use the MACD value as approximation
	// A full implementation would track MACD history for signal calculation
	signal := macdValue * 0.9 // Simplified approximation

	return strategies.MACD{
		Value:     macdValue,
		Signal:    signal,
		Histogram: macdValue - signal,
	}
}

func calculateEMA(candles []candle, period int) float64 {
	if len(candles) < period {
		return 0.0
	}

	multiplier := 2.0 / float64(period+1)
	ema := candles[0].close

	for i := 1; i < len(candles); i++ {
		ema = (candles[i].close * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

func calculateSMA(candles []candle, period int) float64 {
	if len(candles) < period {
		return 0.0
	}

	var sum float64
	start := len(candles) - period
	for i := start; i < len(candles); i++ {
		sum += candles[i].close
	}

	return sum / float64(period)
}

func calculateATR(candles []candle, period int) float64 {
	if len(candles) < period+1 {
		return 0.0
	}

	var trSum float64
	start := len(candles) - period

	for i := start; i < len(candles); i++ {
		highLow := candles[i].high - candles[i].low
		highClose := absFloat(candles[i].high - candles[i-1].close)
		lowClose := absFloat(candles[i].low - candles[i-1].close)

		tr := maxFloat(highLow, maxFloat(highClose, lowClose))
		trSum += tr
	}

	return trSum / float64(period)
}

func calculateBollingerBands(candles []candle, period int, stdDevMultiplier float64) strategies.BollingerBands {
	if len(candles) < period {
		return strategies.BollingerBands{}
	}

	sma := calculateSMA(candles, period)

	// Calculate standard deviation
	var sumSquares float64
	start := len(candles) - period
	for i := start; i < len(candles); i++ {
		diff := candles[i].close - sma
		sumSquares += diff * diff
	}
	stdDev := sqrtFloat(sumSquares / float64(period))

	return strategies.BollingerBands{
		Upper:  sma + (stdDevMultiplier * stdDev),
		Middle: sma,
		Lower:  sma - (stdDevMultiplier * stdDev),
	}
}

func calculateAvgVolume(candles []candle, period int) int64 {
	if len(candles) < period {
		return 0
	}

	var sum int64
	start := len(candles) - period
	for i := start; i < len(candles); i++ {
		sum += candles[i].volume
	}

	return sum / int64(period)
}

func determineTrend(input strategies.AnalysisInput) string {
	if input.SMA20 > input.SMA50 && input.SMA50 > input.SMA200 {
		return "bullish"
	} else if input.SMA20 < input.SMA50 && input.SMA50 < input.SMA200 {
		return "bearish"
	}
	return "neutral"
}

// Helper math functions
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func sqrtFloat(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Simple Newton-Raphson method
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
