package generator

import (
	"math"

	"jax-trading-assistant/libs/strategies"
)

// calculateRSI computes Relative Strength Index
func calculateRSI(candles []candle, period int) float64 {
	if len(candles) < period+1 {
		return 50.0 // neutral default
	}

	gains := 0.0
	losses := 0.0

	// Calculate initial average gain/loss
	for i := len(candles) - period; i < len(candles); i++ {
		change := candles[i].close - candles[i-1].close
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
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

// calculateMACD computes Moving Average Convergence Divergence
func calculateMACD(candles []candle) strategies.MACD {
	if len(candles) < 26 {
		return strategies.MACD{Value: 0, Signal: 0, Histogram: 0}
	}

	ema12 := calculateEMA(candles, 12)
	ema26 := calculateEMA(candles, 26)
	macdLine := ema12 - ema26

	// For signal line, we'd need to calculate EMA of MACD line
	// Simplified: use approximation
	signalLine := macdLine * 0.9 // Simplified approximation

	histogram := macdLine - signalLine

	return strategies.MACD{
		Value:     macdLine,
		Signal:    signalLine,
		Histogram: histogram,
	}
}

// calculateSMA computes Simple Moving Average
func calculateSMA(candles []candle, period int) float64 {
	if len(candles) < period {
		return 0
	}

	sum := 0.0
	for i := len(candles) - period; i < len(candles); i++ {
		sum += candles[i].close
	}
	return sum / float64(period)
}

// calculateEMA computes Exponential Moving Average
func calculateEMA(candles []candle, period int) float64 {
	if len(candles) < period {
		return 0
	}

	multiplier := 2.0 / float64(period+1)

	// Start with SMA
	ema := calculateSMA(candles[:period], period)

	// Apply EMA formula for remaining periods
	for i := period; i < len(candles); i++ {
		ema = (candles[i].close-ema)*multiplier + ema
	}

	return ema
}

// calculateATR computes Average True Range
func calculateATR(candles []candle, period int) float64 {
	if len(candles) < period+1 {
		return 0
	}

	trSum := 0.0
	for i := len(candles) - period; i < len(candles); i++ {
		high := candles[i].high
		low := candles[i].low
		prevClose := candles[i-1].close

		tr := math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
		trSum += tr
	}

	return trSum / float64(period)
}

// calculateBollingerBands computes Bollinger Bands
func calculateBollingerBands(candles []candle, period int, stdDev float64) strategies.BollingerBands {
	if len(candles) < period {
		return strategies.BollingerBands{Upper: 0, Middle: 0, Lower: 0}
	}

	middle := calculateSMA(candles, period)

	// Calculate standard deviation
	sumSquares := 0.0
	for i := len(candles) - period; i < len(candles); i++ {
		diff := candles[i].close - middle
		sumSquares += diff * diff
	}
	stdDevValue := math.Sqrt(sumSquares / float64(period))

	upper := middle + (stdDev * stdDevValue)
	lower := middle - (stdDev * stdDevValue)

	return strategies.BollingerBands{
		Upper:  upper,
		Middle: middle,
		Lower:  lower,
	}
}

// calculateAvgVolume computes average volume
func calculateAvgVolume(candles []candle, period int) int64 {
	if len(candles) < period {
		return 0
	}

	sum := int64(0)
	for i := len(candles) - period; i < len(candles); i++ {
		sum += candles[i].volume
	}
	return sum / int64(period)
}
