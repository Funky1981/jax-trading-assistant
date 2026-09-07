package worldmonitorintelligence

import (
	"fmt"
	"sort"
	"time"
)

const VelocityAlgorithmV1 = "jax.world_monitor_velocity_baseline/v1"

type VelocityState string

const (
	VelocityNormal   VelocityState = "NORMAL"
	VelocityElevated VelocityState = "ELEVATED"
	VelocityAbnormal VelocityState = "ABNORMAL"
	VelocityUnknown  VelocityState = "UNKNOWN"
)

type VelocitySignal struct {
	Algorithm           string
	Window              time.Duration
	BaselineWindow      time.Duration
	WindowEventCount    int
	BaselineEventCount  int
	CurrentRatePerHour  float64
	BaselineRatePerHour float64
	RateRatio           float64
	State               VelocityState
	Unknowns            []string
}

// AssessVelocity compares the cluster's recent event count with a prior
// equal-rate baseline. It reports zero-baseline explicitly and never emits an
// action or candidate.
func AssessVelocity(cluster EventCluster, history []time.Time, now time.Time, window, baselineWindow time.Duration) (VelocitySignal, error) {
	if window <= 0 || baselineWindow <= 0 {
		return VelocitySignal{}, fmt.Errorf("velocity windows must be positive")
	}
	if now.IsZero() {
		return VelocitySignal{}, fmt.Errorf("velocity reference time is required")
	}
	currentStart := now.Add(-window)
	baselineStart := currentStart.Add(-baselineWindow)
	current := 0
	unknowns := []string{}
	for _, member := range cluster.Members {
		if member.CollectedAt.IsZero() {
			unknowns = append(unknowns, fmt.Sprintf("collection time is missing for source event %q", member.ID))
			continue
		}
		if !member.CollectedAt.Before(currentStart) && !member.CollectedAt.After(now) {
			current++
		}
	}
	baseline := 0
	for _, observedAt := range history {
		if observedAt.IsZero() {
			unknowns = append(unknowns, "baseline contains an observation with no timestamp")
			continue
		}
		if !observedAt.Before(baselineStart) && observedAt.Before(currentStart) {
			baseline++
		}
	}
	currentRate := float64(current) / window.Hours()
	baselineRate := float64(baseline) / baselineWindow.Hours()
	ratio := 0.0
	state := VelocityNormal
	if baseline == 0 {
		if current == 0 {
			state = VelocityUnknown
			unknowns = append(unknowns, "baseline and current windows contain no events")
		} else {
			state = VelocityAbnormal
			unknowns = append(unknowns, "baseline contains zero events; ratio is capped and requires interpretation")
			ratio = 999.0
		}
	} else {
		ratio = currentRate / baselineRate
		switch {
		case ratio >= 3:
			state = VelocityAbnormal
		case ratio >= 1.5:
			state = VelocityElevated
		}
	}
	sort.Strings(unknowns)
	return VelocitySignal{Algorithm: VelocityAlgorithmV1, Window: window, BaselineWindow: baselineWindow, WindowEventCount: current, BaselineEventCount: baseline, CurrentRatePerHour: currentRate, BaselineRatePerHour: baselineRate, RateRatio: ratio, State: state, Unknowns: uniqueStrings(unknowns)}, nil
}
