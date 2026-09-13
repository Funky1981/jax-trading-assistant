package marketdata

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// VAL03BHoldoutStart is the first date that must never enter the bounded
// pre-performance VAL-03B bar-family acquisition.
var VAL03BHoldoutStart = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// Alpaca returns daily OHLC values at quote precision. A 5e-4 relative
// tolerance accepts the resulting bounded ratio rounding while still
// rejecting a materially non-common OHLC transform.
const VAL03BFactorTolerance = 5e-4

// VAL03BPriceFrame is a small provider-neutral OHLCV frame used only for the
// pre-performance adjustment-family equivalence checks.
type VAL03BPriceFrame struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// DeriveVAL03BSplitFactor verifies that the split-adjusted OHLCV frame is a
// common price-scale transform of the raw frame. Volume is expected to move by
// the inverse share-scale factor, as documented for Alpaca split adjustment.
func DeriveVAL03BSplitFactor(raw, split VAL03BPriceFrame) (float64, error) {
	prices := [][2]float64{{raw.Open, split.Open}, {raw.High, split.High}, {raw.Low, split.Low}, {raw.Close, split.Close}}
	var factor float64
	for _, pair := range prices {
		if !finitePositive(pair[0]) || !finitePositive(pair[1]) {
			return 0, errors.New("raw and split prices must be finite and positive")
		}
		candidate := pair[1] / pair[0]
		if factor == 0 {
			factor = candidate
		} else if !closeEnough(factor, candidate, VAL03BFactorTolerance) {
			return 0, errors.New("split adjustment factor is inconsistent across OHLC")
		}
	}
	if raw.Volume < 0 || split.Volume < 0 || !finite(raw.Volume) || !finite(split.Volume) {
		return 0, errors.New("raw and split volumes must be finite and non-negative")
	}
	if raw.Volume == 0 || split.Volume == 0 {
		if raw.Volume != split.Volume {
			return 0, errors.New("zero-volume split relation is inconsistent")
		}
		return factor, nil
	}
	if !closeEnough(raw.Volume/split.Volume, factor, VAL03BFactorTolerance) {
		return 0, errors.New("split adjustment factor is inconsistent for volume")
	}
	return factor, nil
}

// IsVAL03BStructuralFactorBoundary identifies a deterministic scale change
// between adjacent synchronized sessions. It does not infer an action type.
func IsVAL03BStructuralFactorBoundary(previous, current float64) bool {
	return previous > 0 && current > 0 && !closeEnough(previous, current, VAL03BFactorTolerance)
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func finitePositive(value float64) bool { return finite(value) && value > 0 }

func closeEnough(left, right, tolerance float64) bool {
	return math.Abs(left-right) <= tolerance*math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
}

// ValidateVAL03BDateRange is the fail-closed date boundary for VAL-03B. It is
// intentionally separate from the general market-data client so ordinary
// live/data operations do not inherit the research holdout policy.
func ValidateVAL03BDateRange(start, end time.Time) error {
	if !isUTCDate(start) || !isUTCDate(end) || end.Before(start) {
		return errors.New("VAL-03B range must be ordered UTC calendar dates")
	}
	if !end.Before(VAL03BHoldoutStart) {
		return fmt.Errorf("VAL-03B range reaches sealed holdout at %s", VAL03BHoldoutStart.Format("2006-01-02"))
	}
	return nil
}

// ValidateVAL03BProviderDate rejects an unexpected provider row at or beyond
// the sealed holdout boundary before it can be used as a VAL-03B observation.
func ValidateVAL03BProviderDate(providerDate string) error {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(providerDate))
	if err != nil {
		return fmt.Errorf("VAL-03B provider date is invalid: %w", err)
	}
	if !date.Before(VAL03BHoldoutStart) {
		return fmt.Errorf("VAL-03B provider date reaches sealed holdout: %s", providerDate)
	}
	return nil
}
