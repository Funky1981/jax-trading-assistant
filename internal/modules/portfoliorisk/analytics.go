package portfoliorisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	ExposureAlgorithmV1       = "jax.portfolio.exposure_concentration/v1"
	CorrelationAlgorithmV1    = "jax.portfolio.correlation/v1"
	MinimumCorrelationSamples = 3
)

type ExposureLine struct {
	InstrumentID    string  `json:"instrument_id"`
	MarketValue     float64 `json:"market_value"`
	AbsoluteValue   float64 `json:"absolute_value"`
	PortfolioWeight float64 `json:"portfolio_weight"`
	Long            bool    `json:"long"`
}

type ExposureAnalytics struct {
	AnalyticsID      string         `json:"analytics_id"`
	Algorithm        string         `json:"algorithm"`
	SnapshotID       string         `json:"snapshot_id"`
	EvaluatedAt      time.Time      `json:"evaluated_at"`
	ValuationBasis   string         `json:"valuation_basis"`
	Equity           float64        `json:"equity"`
	CashAllocation   float64        `json:"cash_allocation"`
	GrossExposure    float64        `json:"gross_exposure"`
	NetExposure      float64        `json:"net_exposure"`
	LongExposure     float64        `json:"long_exposure"`
	ShortExposure    float64        `json:"short_exposure"`
	MaxConcentration float64        `json:"max_concentration"`
	Lines            []ExposureLine `json:"lines"`
	InputProvenance  []string       `json:"input_provenance"`
}

// CalculateExposure reuses Phase-05's signed-quantity definition and applies
// it to canonical, provenance-bearing observations. It refuses unknown or
// stale material facts instead of treating them as zero.
func CalculateExposure(snapshot PortfolioSnapshot, evaluatedAt time.Time, maxAge time.Duration) (ExposureAnalytics, error) {
	canonical, err := BuildSnapshot(snapshot)
	if err != nil {
		return ExposureAnalytics{}, err
	}
	if evaluatedAt.IsZero() || maxAge <= 0 {
		return ExposureAnalytics{}, fmt.Errorf("exposure: evaluation time and freshness window are required")
	}
	freshness := canonical.AssessFreshness(evaluatedAt, maxAge)
	if freshness.Status != FreshnessFresh {
		return ExposureAnalytics{}, fmt.Errorf("exposure: %s: %s", freshness.Status, freshness.Reason)
	}
	if !canonical.Equity.Known || canonical.Equity.Value <= 0 {
		return ExposureAnalytics{}, fmt.Errorf("%w: equity is required for exposure denominators", ErrUnknownMaterialState)
	}
	byInstrument := make(map[string]ExposureLine, len(canonical.Positions))
	for index, position := range canonical.Positions {
		if !position.MarketValue.Known || !position.Price.Known {
			return ExposureAnalytics{}, fmt.Errorf("%w: position %d valuation is unknown", ErrUnknownMaterialState, index)
		}
		expected := position.SignedQuantity * position.Price.Value
		if !closeEnough(expected, position.MarketValue.Value) {
			return ExposureAnalytics{}, fmt.Errorf("%w: position %d market value disagrees with signed quantity and price", ErrUnknownMaterialState, index)
		}
		line := byInstrument[position.InstrumentID]
		line.InstrumentID = position.InstrumentID
		line.MarketValue += position.MarketValue.Value
		byInstrument[position.InstrumentID] = line
	}
	lines := make([]ExposureLine, 0, len(byInstrument))
	for _, line := range byInstrument {
		line.AbsoluteValue = math.Abs(line.MarketValue)
		line.PortfolioWeight = line.AbsoluteValue / canonical.Equity.Value
		line.Long = line.MarketValue >= 0
		lines = append(lines, line)
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].InstrumentID < lines[j].InstrumentID })
	result := ExposureAnalytics{Algorithm: ExposureAlgorithmV1, SnapshotID: canonical.SnapshotID, EvaluatedAt: evaluatedAt.UTC(), ValuationBasis: canonical.ValuationBasis, Equity: canonical.Equity.Value, Lines: lines, InputProvenance: append([]string(nil), canonical.Provenance...)}
	for _, line := range lines {
		result.GrossExposure += line.AbsoluteValue
		result.NetExposure += line.MarketValue
		if line.MarketValue >= 0 {
			result.LongExposure += line.MarketValue
		} else {
			result.ShortExposure -= line.MarketValue
		}
		if line.PortfolioWeight > result.MaxConcentration {
			result.MaxConcentration = line.PortfolioWeight
		}
	}
	result.CashAllocation = canonical.Cash.Value / canonical.Equity.Value
	result.AnalyticsID = exposureIdentity(result)
	return result, nil
}

type ReturnObservation struct {
	At              time.Time `json:"at"`
	AssetReturn     float64   `json:"asset_return"`
	BenchmarkReturn float64   `json:"benchmark_return"`
}

type CorrelationStatus string

const (
	CorrelationKnown   CorrelationStatus = "KNOWN"
	CorrelationUnknown CorrelationStatus = "UNKNOWN / INSUFFICIENT_DATA"
)

type CorrelationResult struct {
	CorrelationID string            `json:"correlation_id"`
	Algorithm     string            `json:"algorithm"`
	InstrumentID  string            `json:"instrument_id"`
	BenchmarkID   string            `json:"benchmark_id"`
	AsOf          time.Time         `json:"as_of"`
	Window        int               `json:"window"`
	Samples       int               `json:"samples"`
	Status        CorrelationStatus `json:"status"`
	Value         *float64          `json:"value,omitempty"`
	Reason        string            `json:"reason"`
}

// CalculateCorrelation accepts already aligned returns. It never fills a
// missing timestamp with zero and rejects observations after the portfolio
// valuation anchor, preventing look-ahead.
func CalculateCorrelation(instrumentID, benchmarkID string, asOf time.Time, observations []ReturnObservation, maxAge time.Duration) CorrelationResult {
	result := CorrelationResult{Algorithm: CorrelationAlgorithmV1, InstrumentID: instrumentID, BenchmarkID: benchmarkID, AsOf: asOf.UTC(), Window: len(observations), Samples: len(observations), Status: CorrelationUnknown}
	if instrumentID == "" || benchmarkID == "" || instrumentID == benchmarkID || asOf.IsZero() || maxAge <= 0 {
		result.Reason = "correlation identity or freshness inputs are incomplete"
		return result
	}
	if len(observations) < MinimumCorrelationSamples {
		result.Reason = fmt.Sprintf("minimum %d aligned observations required", MinimumCorrelationSamples)
		return result
	}
	last := time.Time{}
	assetMean, benchmarkMean := 0.0, 0.0
	for i, observation := range observations {
		if observation.At.IsZero() || !observation.At.UTC().Equal(observation.At) || !observation.At.After(last) {
			result.Reason = "timestamps must be strictly increasing and UTC"
			return result
		}
		if observation.At.After(asOf) {
			result.Reason = "observation is after portfolio valuation anchor"
			return result
		}
		if asOf.Sub(observation.At) > maxAge {
			result.Reason = "correlation history is stale"
			return result
		}
		if !finite(observation.AssetReturn) || !finite(observation.BenchmarkReturn) {
			result.Reason = fmt.Sprintf("observation %d is non-finite", i)
			return result
		}
		assetMean += observation.AssetReturn
		benchmarkMean += observation.BenchmarkReturn
		last = observation.At
	}
	assetMean /= float64(len(observations))
	benchmarkMean /= float64(len(observations))
	covariance, assetVariance, benchmarkVariance := 0.0, 0.0, 0.0
	for _, observation := range observations {
		ad := observation.AssetReturn - assetMean
		bd := observation.BenchmarkReturn - benchmarkMean
		covariance += ad * bd
		assetVariance += ad * ad
		benchmarkVariance += bd * bd
	}
	if assetVariance <= 1e-24 || benchmarkVariance <= 1e-24 {
		result.Reason = "correlation is undefined for zero variance"
		return result
	}
	value := covariance / math.Sqrt(assetVariance*benchmarkVariance)
	if !finite(value) {
		result.Reason = "correlation is non-finite"
		return result
	}
	result.Status, result.Value, result.Reason = CorrelationKnown, &value, "aligned sample correlation"
	result.CorrelationID = correlationIdentity(result)
	return result
}

func closeEnough(left, right float64) bool {
	return finite(left) && finite(right) && math.Abs(left-right) <= 1e-9*math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
}

func exposureIdentity(result ExposureAnalytics) string {
	copyResult := result
	copyResult.AnalyticsID = ""
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	return "exp_" + hex.EncodeToString(digest[:])
}
func correlationIdentity(result CorrelationResult) string {
	copyResult := result
	copyResult.CorrelationID = ""
	b, _ := json.Marshal(copyResult)
	digest := sha256.Sum256(b)
	return "cor_" + hex.EncodeToString(digest[:])
}
