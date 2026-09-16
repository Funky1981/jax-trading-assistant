// Package validation contains reusable, outcome-independent validation
// semantics for future candidates. It deliberately does not import the
// historical VAL-03 runner or its candidate-specific types.
package validation

import (
	"fmt"
	"math"
	"strings"
)

// GateStatus is the fail-closed status of one promotion gate.
type GateStatus string

const (
	GatePass         GateStatus = "PASS"
	GateFail         GateStatus = "FAIL"
	GateInsufficient GateStatus = "INSUFFICIENT"
)

// TerminalStatus is the only classification a future validation package may
// emit. Unknown or unsupported evidence never becomes a pass.
type TerminalStatus string

const (
	ForwardPaperEligible TerminalStatus = "FORWARD_PAPER_ELIGIBLE"
	FailedValidation     TerminalStatus = "FAILED_VALIDATION"
	InsufficientEvidence TerminalStatus = "INSUFFICIENT_EVIDENCE"
)

// FalsificationDisposition is intentionally string-shaped so callers can
// adapt persisted or independently versioned diagnostic contracts without
// importing a candidate-specific runner package.
type FalsificationDisposition struct {
	Name     string
	Status   string
	Blocking bool
}

// PromotionInput contains every independent promotion gate. Floors and
// observed values are kept together so a report can expose the complete gate
// vector rather than collapsing it into one opaque boolean.
type PromotionInput struct {
	EpisodeFloor         int
	MatchedPairFloor     int
	EffectiveBlockFloor  int
	InstrumentFloor      int
	CalendarRegimeFloor  int
	EpisodeCount         int
	MatchedPairCount     int
	EffectiveBlockCount  int
	InstrumentCount      int
	CalendarRegimeSlices int

	InstrumentConcentrationCeiling float64
	MaximumInstrumentContribution  float64

	PrimaryMeanNet        float64
	PrimaryBootstrapLower float64
	MatchedMeanDifference float64
	PairedBootstrapLower  float64

	RegisteredFalsifications []FalsificationDisposition
}

// GateResult is a named, independently inspectable gate outcome.
type GateResult struct {
	Name   string     `json:"name"`
	Status GateStatus `json:"status"`
	Reason string     `json:"reason,omitempty"`
}

// PromotionDecision is deterministic and contains the full gate vector.
type PromotionDecision struct {
	Gates          []GateResult   `json:"gates"`
	Classification TerminalStatus `json:"classification"`
}

// EvaluatePromotion evaluates future-candidate gates in a fixed order.
// Missing, malformed, or unsupported gate inputs are INSUFFICIENT rather than
// silently passing. A genuine FAIL takes precedence over INSUFFICIENT when
// both are present, preserving a terminal failed-validation decision.
func EvaluatePromotion(input PromotionInput) PromotionDecision {
	gates := []GateResult{
		minimumGate("sample_episode_floor", input.EpisodeCount, input.EpisodeFloor),
		minimumGate("matched_pair_floor", input.MatchedPairCount, input.MatchedPairFloor),
		minimumGate("effective_block_floor", input.EffectiveBlockCount, input.EffectiveBlockFloor),
		minimumGate("instrument_floor", input.InstrumentCount, input.InstrumentFloor),
		minimumGate("calendar_regime_floor", input.CalendarRegimeSlices, input.CalendarRegimeFloor),
		concentrationGate(input.MaximumInstrumentContribution, input.InstrumentConcentrationCeiling),
		positiveGate("primary_mean_net_positive", input.PrimaryMeanNet, "primary mean net return must be positive"),
		positiveGate("primary_bootstrap_lower_positive", input.PrimaryBootstrapLower, "primary bootstrap lower bound must be positive"),
		comparisonGate("matched_mean_difference_positive", input.MatchedPairCount, input.MatchedPairFloor, input.MatchedMeanDifference, "matched actual-minus-placebo mean must be positive"),
		comparisonGate("paired_bootstrap_lower_positive", input.MatchedPairCount, input.MatchedPairFloor, input.PairedBootstrapLower, "paired bootstrap lower bound must be positive"),
		falsificationGate(input.RegisteredFalsifications),
	}

	classification := ForwardPaperEligible
	for _, gate := range gates {
		if gate.Status == GateFail {
			classification = FailedValidation
			break
		}
		if gate.Status == GateInsufficient {
			classification = InsufficientEvidence
		}
	}
	return PromotionDecision{Gates: gates, Classification: classification}
}

func minimumGate(name string, observed, floor int) GateResult {
	if floor <= 0 {
		return GateResult{Name: name, Status: GateInsufficient, Reason: "unsupported non-positive registered floor"}
	}
	if observed < 0 {
		return GateResult{Name: name, Status: GateInsufficient, Reason: "unsupported negative observed count"}
	}
	if observed < floor {
		return GateResult{Name: name, Status: GateInsufficient, Reason: fmt.Sprintf("observed %d is below floor %d", observed, floor)}
	}
	return GateResult{Name: name, Status: GatePass}
}

func positiveGate(name string, value float64, reason string) GateResult {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return GateResult{Name: name, Status: GateInsufficient, Reason: "unsupported non-finite value"}
	}
	if value <= 0 {
		return GateResult{Name: name, Status: GateFail, Reason: reason}
	}
	return GateResult{Name: name, Status: GatePass}
}

func comparisonGate(name string, pairs, floor int, value float64, reason string) GateResult {
	if floor <= 0 || pairs < floor {
		return GateResult{Name: name, Status: GateInsufficient, Reason: "matched placebo evidence is below the registered pair floor"}
	}
	return positiveGate(name, value, reason)
}

func concentrationGate(maximum, ceiling float64) GateResult {
	if math.IsNaN(maximum) || math.IsInf(maximum, 0) || math.IsNaN(ceiling) || math.IsInf(ceiling, 0) || maximum < 0 || maximum > 1 || ceiling <= 0 || ceiling > 1 {
		return GateResult{Name: "instrument_concentration_ceiling", Status: GateInsufficient, Reason: "unsupported concentration value or ceiling"}
	}
	if maximum > ceiling {
		return GateResult{Name: "instrument_concentration_ceiling", Status: GateFail, Reason: fmt.Sprintf("maximum contribution %.6f exceeds ceiling %.6f", maximum, ceiling)}
	}
	return GateResult{Name: "instrument_concentration_ceiling", Status: GatePass}
}

func falsificationGate(dispositions []FalsificationDisposition) GateResult {
	if len(dispositions) == 0 {
		return GateResult{Name: "registered_blocking_falsifications", Status: GateInsufficient, Reason: "registered blocking falsification dispositions are missing"}
	}
	unknown := []string{}
	failures := []string{}
	insufficient := []string{}
	for _, disposition := range dispositions {
		if !disposition.Blocking {
			continue
		}
		name := disposition.Name
		if name == "" {
			name = "<unnamed>"
		}
		switch strings.ToUpper(disposition.Status) {
		case "PASS":
		case "FAIL":
			failures = append(failures, name)
		case "INSUFFICIENT":
			insufficient = append(insufficient, name)
		default:
			unknown = append(unknown, name)
		}
	}
	if len(failures) > 0 {
		return GateResult{Name: "registered_blocking_falsifications", Status: GateFail, Reason: "blocking falsifications failed: " + strings.Join(failures, ", ")}
	}
	if len(unknown) > 0 || len(insufficient) > 0 {
		reason := append([]string{}, insufficient...)
		reason = append(reason, unknown...)
		return GateResult{Name: "registered_blocking_falsifications", Status: GateInsufficient, Reason: "blocking falsification evidence is insufficient or unsupported: " + strings.Join(reason, ", ")}
	}
	return GateResult{Name: "registered_blocking_falsifications", Status: GatePass}
}

// TopFiveExclusionInput is the already-calculated, future-report summary.
// It intentionally carries both primary and paired effects before and after
// exclusion so wording cannot imply a causal sign change that was not shown.
type TopFiveExclusionInput struct {
	PrimaryMeanBefore float64 `json:"primary_mean_before"`
	PrimaryMeanAfter  float64 `json:"primary_mean_after"`
	PairedMeanBefore  float64 `json:"paired_mean_before"`
	PairedMeanAfter   float64 `json:"paired_mean_after"`
	RemovedCount      int     `json:"removed_count"`
	RemainingPairs    int     `json:"remaining_pairs"`
	PairedPairFloor   int     `json:"paired_pair_floor"`
}

// TopFiveExclusionResult is neutral about causality: it reports the observed
// remaining effect and never claims removal caused an effect to disappear
// unless a caller separately establishes a sign change.
type TopFiveExclusionResult struct {
	PrimaryMeanBefore float64    `json:"primary_mean_before"`
	PrimaryMeanAfter  float64    `json:"primary_mean_after"`
	PairedMeanBefore  float64    `json:"paired_mean_before"`
	PairedMeanAfter   float64    `json:"paired_mean_after"`
	RemovedCount      int        `json:"removed_count"`
	RemainingPairs    int        `json:"remaining_pairs"`
	Disposition       GateStatus `json:"disposition"`
	Reason            string     `json:"reason"`
}

// EvaluateTopFiveExclusion applies the future, non-causal wording contract.
func EvaluateTopFiveExclusion(input TopFiveExclusionInput) TopFiveExclusionResult {
	result := TopFiveExclusionResult{
		PrimaryMeanBefore: input.PrimaryMeanBefore,
		PrimaryMeanAfter:  input.PrimaryMeanAfter,
		PairedMeanBefore:  input.PairedMeanBefore,
		PairedMeanAfter:   input.PairedMeanAfter,
		RemovedCount:      input.RemovedCount,
		RemainingPairs:    input.RemainingPairs,
		Disposition:       GatePass,
		Reason:            "remaining primary and paired effects are positive",
	}
	if input.PairedPairFloor <= 0 || input.RemainingPairs < input.PairedPairFloor {
		result.Disposition = GateInsufficient
		result.Reason = "remaining matched evidence is below the registered pair floor"
		return result
	}
	if input.PrimaryMeanAfter <= 0 || input.PairedMeanAfter <= 0 {
		result.Disposition = GateFail
		result.Reason = "remaining primary or paired effect is non-positive"
	}
	return result
}
