package main

import "fmt"

const (
	structuralWarmupState      = "POST_SPLIT_WARMUP"
	structuralInitializedState = "INITIALIZATION_COMPLETE"
)

// structuralStateAt is the single eligibility calculation for the recovery
// runner. The boundary session is reset first and then counted as post-boundary
// valid session one, matching the frozen VAL-03C contract.
func structuralStateAt(d *instrumentData, date string, required int) (string, error) {
	if required <= 0 {
		return "", fmt.Errorf("invalid structural lookback %d", required)
	}
	idx := indexOf(d.Dates, date)
	if idx < 0 {
		return "", fmt.Errorf("unknown structural date %s", date)
	}
	count := 0
	for i := 0; i <= idx; i++ {
		current := d.Dates[i]
		if d.Boundaries[current] {
			count = 0
		}
		if !validSynchronizedSession(d, current) {
			continue
		}
		count++
		if count >= required {
			// Later sessions remain initialized until a new boundary is seen.
			return structuralInitializedState, nil
		}
	}
	return structuralWarmupState, nil
}

func structurallyEligibleAt(d *instrumentData, date string, required int) bool {
	state, err := structuralStateAt(d, date, required)
	return err == nil && state == structuralInitializedState
}

func validSynchronizedSession(d *instrumentData, date string) bool {
	raw, split, detector := d.Raw[date], d.Split[date], d.Detector[date]
	return raw.Date == date && split.Date == date && detector.Date == date &&
		raw.Open > 0 && raw.High > 0 && raw.Low > 0 && raw.Close > 0 && raw.Volume >= 0 &&
		split.Open > 0 && split.High > 0 && split.Low > 0 && split.Close > 0 && split.Volume >= 0 &&
		detector.Open > 0 && detector.High > 0 && detector.Low > 0 && detector.Close > 0 && detector.Volume >= 0
}

func structuralWarmupAbstention(r *partitionResult, d *instrumentData, date string, required int) bool {
	if required <= 0 {
		return false
	}
	if structurallyEligibleAt(d, date, required) {
		return false
	}
	r.Abstentions[structuralWarmupState]++
	return true
}
