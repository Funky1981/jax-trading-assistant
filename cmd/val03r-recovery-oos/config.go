package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// FrozenExperimentConfig is the single source of truth for outcome-bearing
// VAL-03 semantics. It is loaded from the hash-bound preregistration rather
// than reconstructed from call-site literals.
type FrozenExperimentConfig struct {
	CandidateID, StrategyVersion, BaseCostID, StressCostID                     string
	ActionableConfidenceThreshold                                              float64
	SMAFast, SMAMedium, SMASlow, ATRPeriod                                     int
	AvgVolumePeriod                                                            int
	StopATR, TargetATR                                                         float64
	HoldingPeriod, InitializationRequired, BootstrapN                          int
	OverlapWindow                                                              int
	BaseSpreadBPS, BaseSlippageBPS, BaseImpactBPS                              float64
	StressSpreadBPS, StressSlippageBPS, StressImpactBPS                        float64
	StressCommissionPerOrder, StressCommissionBPS                              float64
	Universe                                                                   []string
	DevelopmentStart, DevelopmentEnd                                           string
	ValidationStart, ValidationEnd                                             string
	OOSStart, OOSEnd, HoldoutStart                                             string
	DevelopmentSampleFloor, ValidationSampleFloor, OOSSampleFloor, PairedFloor int
	EffectiveBlockFloor, InstrumentFloor                                       int
	MinimumSliceFloor                                                          int
	ConcentrationCeiling                                                       float64
	SeedDomains                                                                []string
	ExecutionAuthority                                                         string
}

var expectedUniverse = []string{"SPY", "QQQ", "IWM", "DIA", "XLK", "XLF", "XLE", "TLT", "GLD"}

func loadFrozenExperimentConfig(manifestBytes []byte, manifestHash string) (FrozenExperimentConfig, error) {
	if manifestHash != "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a" {
		return FrozenExperimentConfig{}, errors.New("manifest hash mismatch")
	}
	var root map[string]any
	if err := json.Unmarshal(manifestBytes, &root); err != nil {
		return FrozenExperimentConfig{}, fmt.Errorf("decode manifest: %w", err)
	}
	if got, err := stringAt(root, "contract_version"); err != nil || got != "v1.5" {
		return FrozenExperimentConfig{}, errors.New("unexpected manifest contract version")
	}
	c := FrozenExperimentConfig{}
	var err error
	if c.CandidateID, err = stringAt(root, "selected_candidate", "candidate_id"); err != nil || c.CandidateID != "ma_crossover_v1" {
		return FrozenExperimentConfig{}, errors.New("candidate identity mismatch")
	}
	if c.StrategyVersion, err = stringAt(root, "selected_candidate", "strategy_version"); err != nil || c.StrategyVersion != "ma_crossover_v1" {
		return FrozenExperimentConfig{}, errors.New("strategy version mismatch")
	}
	if c.ActionableConfidenceThreshold, err = numberAt(root, "confidence_contract", "actionable_threshold"); err != nil || c.ActionableConfidenceThreshold != 0.60 {
		return FrozenExperimentConfig{}, errors.New("actionable confidence threshold mismatch")
	}
	if c.SMAFast, err = intAt(root, "parameters", "sma_fast_sessions"); err != nil || c.SMAFast != 20 {
		return FrozenExperimentConfig{}, errors.New("SMA fast period mismatch")
	}
	if c.SMAMedium, err = intAt(root, "parameters", "sma_medium_sessions"); err != nil || c.SMAMedium != 50 {
		return FrozenExperimentConfig{}, errors.New("SMA medium period mismatch")
	}
	if c.SMASlow, err = intAt(root, "parameters", "sma_slow_sessions"); err != nil || c.SMASlow != 200 {
		return FrozenExperimentConfig{}, errors.New("SMA slow period mismatch")
	}
	if c.ATRPeriod, err = intAt(root, "parameters", "atr_period_sessions"); err != nil || c.ATRPeriod != 14 {
		return FrozenExperimentConfig{}, errors.New("ATR period mismatch")
	}
	c.AvgVolumePeriod = 20
	if c.StopATR, err = numberAt(root, "parameters", "crossover_stop_atr"); err != nil || c.StopATR != 1 {
		return FrozenExperimentConfig{}, errors.New("stop parameter mismatch")
	}
	if c.TargetATR, err = numberAt(root, "parameters", "crossover_target_atr"); err != nil || c.TargetATR != 3 {
		return FrozenExperimentConfig{}, errors.New("target parameter mismatch")
	}
	if c.HoldingPeriod, err = intAt(root, "parameters", "holding_period_sessions"); err != nil || c.HoldingPeriod != 20 {
		return FrozenExperimentConfig{}, errors.New("holding-period mismatch")
	}
	if c.Universe, err = stringsAt(root, "universe", "symbols"); err != nil || !sameStrings(c.Universe, expectedUniverse) {
		return FrozenExperimentConfig{}, errors.New("universe mismatch")
	}
	if c.DevelopmentStart, c.DevelopmentEnd, err = rangeAt(root, "partitions", "development"); err != nil || c.DevelopmentStart != "2016-01-01" || c.DevelopmentEnd != "2020-12-31" {
		return FrozenExperimentConfig{}, errors.New("development partition mismatch")
	}
	if c.ValidationStart, c.ValidationEnd, err = rangeAt(root, "partitions", "validation"); err != nil || c.ValidationStart != "2021-01-01" || c.ValidationEnd != "2022-12-31" {
		return FrozenExperimentConfig{}, errors.New("validation partition mismatch")
	}
	if c.OOSStart, c.OOSEnd, err = rangeAt(root, "partitions", "formal_oos"); err != nil || c.OOSStart != "2023-01-01" || c.OOSEnd != "2024-12-31" {
		return FrozenExperimentConfig{}, errors.New("formal OOS partition mismatch")
	}
	holdout, err := stringsAt(root, "partitions", "final_holdout")
	if err != nil || len(holdout) < 2 || holdout[0] != "2025-01-01" {
		return FrozenExperimentConfig{}, errors.New("holdout mismatch")
	}
	c.HoldoutStart = holdout[0]
	if c.InitializationRequired, err = intAt(root, "val03c_warmup_contract", "initialization_required_sessions"); err != nil || c.InitializationRequired != 200 {
		return FrozenExperimentConfig{}, errors.New("initialization requirement mismatch")
	}
	if c.BaseCostID, err = stringAt(root, "cost_models", "primary_base", "id"); err != nil || c.BaseCostID != "cost_val03_ibkr_fixed_personal_base_v1" {
		return FrozenExperimentConfig{}, errors.New("base cost policy mismatch")
	}
	if c.StressCostID, err = stringAt(root, "cost_models", "stress", "id"); err != nil || c.StressCostID != "cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8" {
		return FrozenExperimentConfig{}, errors.New("stress cost policy mismatch")
	}
	if c.DevelopmentSampleFloor, err = intAt(root, "sample_and_dependence", "episode_floors", "development"); err != nil || c.DevelopmentSampleFloor != 90 {
		return FrozenExperimentConfig{}, errors.New("development sample floor mismatch")
	}
	if c.ValidationSampleFloor, err = intAt(root, "sample_and_dependence", "episode_floors", "validation"); err != nil || c.ValidationSampleFloor != 30 {
		return FrozenExperimentConfig{}, errors.New("validation sample floor mismatch")
	}
	if c.OOSSampleFloor, err = intAt(root, "sample_and_dependence", "episode_floors", "formal_oos"); err != nil || c.OOSSampleFloor != 30 {
		return FrozenExperimentConfig{}, errors.New("OOS sample floor mismatch")
	}
	if c.PairedFloor, err = intAt(root, "sample_and_dependence", "minimum_paired_placebos"); err != nil || c.PairedFloor != 30 {
		return FrozenExperimentConfig{}, errors.New("paired-placebo floor mismatch")
	}
	if c.EffectiveBlockFloor, err = intAt(root, "sample_and_dependence", "effective_block_floor", "formal_oos_nonempty_instrument_year_blocks"); err != nil || c.EffectiveBlockFloor != 12 {
		return FrozenExperimentConfig{}, errors.New("effective-block floor mismatch")
	}
	if c.InstrumentFloor, err = intAt(root, "sample_and_dependence", "effective_block_floor", "formal_oos_instruments"); err != nil || c.InstrumentFloor != 6 {
		return FrozenExperimentConfig{}, errors.New("instrument floor mismatch")
	}
	if c.MinimumSliceFloor, err = intAt(root, "sample_and_dependence", "minimum_regime_or_calendar_slices"); err != nil || c.MinimumSliceFloor != 3 {
		return FrozenExperimentConfig{}, errors.New("minimum slice floor mismatch")
	}
	if c.ConcentrationCeiling, err = numberAt(root, "sample_and_dependence", "instrument_contribution_max"); err != nil || c.ConcentrationCeiling != 0.4 {
		return FrozenExperimentConfig{}, errors.New("concentration ceiling mismatch")
	}
	if c.BootstrapN, err = intAt(root, "bootstrap", "replicates"); err != nil || c.BootstrapN != bootstrapN {
		return FrozenExperimentConfig{}, errors.New("bootstrap replicate mismatch")
	}
	if c.BaseSpreadBPS, err = numberAt(root, "cost_models", "primary_base", "spread_bps_per_leg"); err != nil || c.BaseSpreadBPS != 4 {
		return FrozenExperimentConfig{}, errors.New("base spread mismatch")
	}
	if c.BaseSlippageBPS, err = numberAt(root, "cost_models", "primary_base", "slippage_bps_per_leg"); err != nil || c.BaseSlippageBPS != 5 {
		return FrozenExperimentConfig{}, errors.New("base slippage mismatch")
	}
	if c.BaseImpactBPS, err = numberAt(root, "cost_models", "primary_base", "market_impact_bps_per_leg"); err != nil || c.BaseImpactBPS != 0 {
		return FrozenExperimentConfig{}, errors.New("base impact mismatch")
	}
	if c.StressSpreadBPS, err = numberAt(root, "cost_models", "stress", "spread_bps_per_leg"); err != nil || c.StressSpreadBPS != 5 {
		return FrozenExperimentConfig{}, errors.New("stress spread mismatch")
	}
	if c.StressSlippageBPS, err = numberAt(root, "cost_models", "stress", "slippage_bps_per_leg"); err != nil || c.StressSlippageBPS != 10 {
		return FrozenExperimentConfig{}, errors.New("stress slippage mismatch")
	}
	if c.StressImpactBPS, err = numberAt(root, "cost_models", "stress", "market_impact_bps_per_leg"); err != nil || c.StressImpactBPS != 15 {
		return FrozenExperimentConfig{}, errors.New("stress impact mismatch")
	}
	if c.StressCommissionPerOrder, err = numberAt(root, "cost_models", "stress", "commission_per_order_usd"); err != nil || c.StressCommissionPerOrder != 0.50 {
		return FrozenExperimentConfig{}, errors.New("stress fixed commission mismatch")
	}
	if c.StressCommissionBPS, err = numberAt(root, "cost_models", "stress", "commission_bps_per_leg"); err != nil || c.StressCommissionBPS != 10 {
		return FrozenExperimentConfig{}, errors.New("stress commission bps mismatch")
	}
	if c.ExecutionAuthority, err = stringAt(root, "execution_authority"); err != nil || c.ExecutionAuthority != "NONE" {
		return FrozenExperimentConfig{}, errors.New("execution authority mismatch")
	}
	c.SeedDomains = []string{"|instrument-year-bootstrap-v2|", "|timestamp-placebo-v1|", "|secondary-sign-permutation-v1|"}
	c.OverlapWindow = 20
	return c, nil
}

func stringAt(root map[string]any, path ...string) (string, error) {
	v, err := valueAt(root, path...)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", errors.New("expected string")
	}
	return s, nil
}

func numberAt(root map[string]any, path ...string) (float64, error) {
	v, err := valueAt(root, path...)
	if err != nil {
		return 0, err
	}
	n, ok := v.(float64)
	if !ok {
		return 0, errors.New("expected number")
	}
	return n, nil
}

func intAt(root map[string]any, path ...string) (int, error) {
	n, err := numberAt(root, path...)
	return int(n), err
}

func stringsAt(root map[string]any, path ...string) ([]string, error) {
	v, err := valueAt(root, path...)
	if err != nil {
		return nil, err
	}
	values, ok := v.([]any)
	if !ok {
		return nil, errors.New("expected string array")
	}
	out := make([]string, len(values))
	for i, item := range values {
		out[i], ok = item.(string)
		if !ok {
			return nil, errors.New("expected string array item")
		}
	}
	return out, nil
}

func rangeAt(root map[string]any, path ...string) (string, string, error) {
	values, err := stringsAt(root, path...)
	if err != nil || len(values) != 2 {
		return "", "", errors.New("expected date range")
	}
	return values[0], values[1], nil
}

func valueAt(root map[string]any, path ...string) (any, error) {
	var current any = root
	for _, key := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, errors.New("missing manifest object")
		}
		current, ok = object[key]
		if !ok {
			return nil, fmt.Errorf("missing manifest field %s", key)
		}
	}
	return current, nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contaminatedRunGuard(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var record struct {
		FinalTerminalClassification string `json:"final_terminal_classification"`
		NoRerun                     bool   `json:"no_rerun"`
		FormalOOSRunCount           int    `json:"formal_oos_run_count"`
	}
	if err := json.Unmarshal(b, &record); err != nil {
		return false, err
	}
	return record.NoRerun && record.FormalOOSRunCount == 1 && record.FinalTerminalClassification == "CONTAMINATED_FOR_FORMAL_OOS", nil
}

//nolint:unused // retained as a compatibility helper for the future R4 CLI.
func preflightOnly() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--preflight-only" {
			return true
		}
	}
	return false
}

//nolint:unused // retained as a compatibility helper for the future R4 CLI.
func contractAuditOnly() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--contract-audit" {
			return true
		}
	}
	return false
}

//nolint:unused // retained as a compatibility helper for the future R4 CLI.
func recoveryContractAuditOnly() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--recovery-contract-audit" {
			return true
		}
	}
	return false
}

//nolint:unused // retained for the future R4 contract-audit surface.
func auditManifestContract(manifestBytes []byte, cfg FrozenExperimentConfig) error {
	var root map[string]any
	if err := json.Unmarshal(manifestBytes, &root); err != nil {
		return fmt.Errorf("decode contract: %w", err)
	}
	if cfg.MinimumSliceFloor != 3 || cfg.OverlapWindow != 20 || cfg.DevelopmentSampleFloor != 90 || cfg.ValidationSampleFloor != 30 || cfg.OOSSampleFloor != 30 || cfg.PairedFloor != 30 || cfg.EffectiveBlockFloor != 12 || cfg.InstrumentFloor != 6 || cfg.ConcentrationCeiling != 0.4 {
		return errors.New("promotion-critical floors are not fully bound")
	}
	tests, err := stringsAt(root, "falsification_suite", "tests")
	expectedTests := []string{"matched non-signal placebo", "constrained timestamp placebo", "secondary mixed-direction sign permutation where applicable", "top-five-percent contribution exclusion", "leave-one-instrument-out", "SPY SMA200 regime split", "predefined theme groups", "SMA 19/49/199", "SMA 21/51/201", "0.90*ATR14", "1.10*ATR14", "legacy Phase-07 stress cost", "20-session overlap sensitivity", "zero and buy-hold baselines"}
	if err != nil || !sameStrings(tests, expectedTests) {
		return errors.New("falsification registration is incomplete")
	}
	if _, err := stringAt(root, "secondary_benchmarks", "primary_comparator"); err != nil {
		return errors.New("primary comparator is not registered")
	}
	if _, err := stringAt(root, "placebo_contract", "timestamp_placebo", "selection"); err != nil {
		return errors.New("timestamp placebo is not registered")
	}
	if _, err := stringAt(root, "placebo_contract", "matched_non_signal", "selection"); err != nil {
		return errors.New("matched placebo is not registered")
	}
	overlap, err := stringAt(root, "robustness_variants", "overlap_sensitivity")
	if err != nil || overlap != "retain earliest signal per instrument in each 20-trading-session entry-date window; compare descriptively with baseline actual-exit non-overlap" {
		return errors.New("overlap sensitivity is not registered")
	}
	if admission, err := stringAt(root, "forward_paper_admission"); err != nil || admission != "not authorized by this manifest" {
		return errors.New("promotion admission is not fail-closed")
	}
	if cfg.ExecutionAuthority != "NONE" {
		return errors.New("execution authority is not NONE")
	}
	if topFiveRuleVersion != "CEIL_5_PERCENT_V1" || slicePolicyVersion != "CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1" || signPermutationAlgorithmVer != "SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1" || secondaryDiagnosticPolicyVer != "SECONDARY_DIRECTIONAL_PRICE_EFFECT_V1" || mixedStratumPolicyVer != "MIXED_INSTRUMENT_YEAR_ONLY_V1" || unknownDispositionPolicyVer != "FAIL_CLOSED_UNKNOWN_STATUS_V1" || nullStatisticPolicyVer != "SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1" {
		return errors.New("R1B/R1C/R1D contract identity mismatch")
	}
	source, err := os.ReadFile("cmd/val03-ma-crossover/main.go")
	if err != nil {
		return fmt.Errorf("read falsification wiring: %w", err)
	}
	if strings.Contains(string(source), "signPermutation(nil") || !strings.Contains(string(source), "secondarySignPermutation(secondary") {
		return errors.New("secondary sign permutation lacks explicit observation wiring")
	}
	contractSource, err := os.ReadFile("cmd/val03-ma-crossover/contract.go")
	if err != nil {
		return fmt.Errorf("read R1B contract implementation: %w", err)
	}
	contractText := string(contractSource)
	if !strings.Contains(contractText, "topFiveFalsification") || !strings.Contains(contractText, "Blocking && disposition.Status") || !strings.Contains(contractText, "calendarRegimeCells") || !strings.Contains(contractText, "known[disposition.Status]") || !strings.Contains(contractText, "MIXED_INSTRUMENT_YEAR_ONLY_V1") || !strings.Contains(contractText, "SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1") || !strings.Contains(contractText, "observedDirectionalStatistic") || !strings.Contains(contractText, "nullDirectionalStatistic") || !strings.Contains(contractText, "nullAssignmentSign") || !strings.Contains(string(source), "simulateDirectional") {
		return errors.New("R1B/R1C/R1D blocking, breadth, or production diagnostic semantics are not active")
	}
	blocked, err := contaminatedRunGuard(integrityPath)
	if err != nil || !blocked {
		return errors.New("old OOS rerun guard is not active")
	}
	return nil
}

func actionableByConfig(confidence float64, cfg FrozenExperimentConfig) bool {
	return confidence >= cfg.ActionableConfidenceThreshold
}
