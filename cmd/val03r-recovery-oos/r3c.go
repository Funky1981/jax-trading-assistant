package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const r3cFreezePath = "Docs/validation/results/VAL-03R3C-RECOVERY-OOS-EXECUTION-FREEZE.json"

var r3cRunnerPaths = map[string]string{
	"main.go":       "cmd/val03r-recovery-oos/main.go",
	"config.go":     "cmd/val03r-recovery-oos/config.go",
	"contract.go":   "cmd/val03r-recovery-oos/contract.go",
	"recovery.go":   "cmd/val03r-recovery-oos/recovery.go",
	"r3a.go":        "cmd/val03r-recovery-oos/r3a.go",
	"r3b.go":        "cmd/val03r-recovery-oos/r3b.go",
	"structural.go": "cmd/val03r-recovery-oos/structural.go",
	"r3c.go":        "cmd/val03r-recovery-oos/r3c.go",
}

func r3cValidateFreeze() error {
	b, err := os.ReadFile(r3cFreezePath)
	if err != nil {
		return fmt.Errorf("read R3C execution freeze: %w", err)
	}
	var root map[string]any
	if err := unmarshalStrictObject(b, &root); err != nil {
		return err
	}
	topLevelKeys := []string{"contract_id", "status", "parent", "identities", "runner", "candidate", "boundary", "structural_contract", "costs", "placebo", "bootstrap", "sample_gates", "falsification_suite", "result_contract", "authorization", "run_state", "safety", "lifecycle"}
	if len(root) != len(topLevelKeys) {
		return errors.New("R3C freeze top-level field set mismatch")
	}
	for _, key := range topLevelKeys {
		if _, ok := root[key]; !ok {
			return fmt.Errorf("R3C freeze missing %s", key)
		}
	}
	if s, _ := stringAt(root, "contract_id"); s != "jax.val-03r3c.recovery-oos-execution-freeze/v1" {
		return errors.New("R3C contract identity mismatch")
	}
	if s, _ := stringAt(root, "status"); s != "FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION" {
		return errors.New("R3C status mismatch")
	}
	checks := map[string]string{
		"parent.manifest_sha256":                     parentManifestSHA,
		"parent.r2_manifest_sha256":                  r3aR2ManifestSHA,
		"parent.r3_conformance_sha256":               "d1799329cadf925fbf78f213ab15c546570c93f6942253664cf290a4a8ca3f60",
		"parent.r3_readiness_sha256":                 r3aReadinessSHA,
		"parent.r3_data_contract_sha256":             r3aDataContractSHA,
		"parent.r3_recovery_dataset_sha256":          r3aRecoveryDatasetSHA,
		"parent.r3a_corrected_dataset_sha256":        r3aCorrectedDatasetSHA,
		"parent.val03b_bar_family_sha256":            r3aBarFamilySHA,
		"parent.val03c_readiness_sha256":             "b7081deb3bf55be5a23c1f1fcafbe44ff90c4fcee924de47f738cdcad86861ce",
		"candidate.id":                               "ma_crossover_v1",
		"candidate.primary_population":               "ACTIONABLE LONG EPISODES",
		"candidate.threshold_source":                 "parent v1.5 manifest",
		"boundary.start":                             r3aRecoveryStart,
		"boundary.end":                               r3aRecoveryEnd,
		"boundary.excluded_incomplete_date":          "2026-09-14",
		"boundary.semantics":                         "PARTIAL CALENDAR-YEAR INSTRUMENT BLOCK",
		"structural_contract.post_split_reset":       "200 valid synchronized sessions",
		"structural_contract.boundary_session_count": "1",
		"structural_contract.split_spanning_episode": "STRUCTURAL_ACTION_INVALIDATED",
		"costs.base_model_id":                        "cost_val03_ibkr_fixed_personal_base_v1",
		"costs.stress_model_id":                      "cost_4c9e2fbffb41d05b537eadf05b142c8f8d00ed46d672d75bca5f83a4269404f8",
		"placebo.matched":                            "jax.val-02c.matched-non-signal-placebo/v1",
		"placebo.timestamp":                          "jax.val-02c.timestamp-placebo/v1",
		"bootstrap.unit":                             "non-empty instrument-year block",
		"bootstrap.interval":                         "95% percentile",
		"bootstrap.lower_bound":                      "2.5th percentile",
		"bootstrap.randomness":                       "SHA256 counter-derived",
		"bootstrap.seed_manifest":                    "parent v1.5 SHA",
		"result_contract.schema":                     "jax.val-03r4.recovery-oos-results/v1",
		"authorization.path":                         r3aAuthorizationPath,
		"authorization.required_external_value":      "GO_VAL03R4_SINGLE_RECOVERY_OOS_EXECUTION",
		"run_state.path":                             r3aRunStatePath,
		"run_state.started_once":                     "run count 1 / attempt consumed",
		"run_state.completed_once":                   "run count 1 / attempt consumed",
		"safety.execution_authority":                 "NONE",
		"safety.forward_paper":                       "NOT_STARTED",
		"safety.phase_13":                            "NOT_STARTED",
		"lifecycle.status":                           "NOT_STARTED / FROZEN_FOR_FUTURE_AUTHORIZATION",
		"identities.candidate_id":                    "ma_crossover_v1",
		"identities.strategy_parameters":             "20/50/200 SMA; ATR14; AvgVolume20; threshold 0.60",
		"identities.original_formal_oos":             "2023-01-01..2024-12-31 CONTAMINATED_FOR_FORMAL_OOS",
		"identities.recovery_boundary":               "2025-01-01..2026-09-11",
	}
	for path, want := range checks {
		parts := splitDot(path)
		got, err := stringAt(root, parts...)
		if err != nil || got != want {
			return fmt.Errorf("R3C field %s mismatch", path)
		}
	}
	threshold, err := numberAt(root, "candidate", "threshold")
	if err != nil || threshold != .6 {
		return errors.New("R3C threshold mismatch")
	}
	if fast, _ := stringsAt(root, "candidate", "sma"); !sameStrings(fast, []string{"20", "50", "200"}) {
		return errors.New("R3C SMA contract mismatch")
	}
	if n, _ := intAt(root, "candidate", "atr"); n != 14 {
		return errors.New("R3C ATR contract mismatch")
	}
	if n, _ := intAt(root, "candidate", "avg_volume"); n != 20 {
		return errors.New("R3C average-volume contract mismatch")
	}
	if n, _ := intAt(root, "candidate", "hold"); n != 20 {
		return errors.New("R3C holding-period contract mismatch")
	}
	if n, _ := intAt(root, "candidate", "reference_capital_usd"); n != 10000 {
		return errors.New("R3C reference-capital contract mismatch")
	}
	if s, _ := stringAt(root, "candidate", "quantity"); s != "whole shares" {
		return errors.New("R3C quantity contract mismatch")
	}
	if extended, err := boolAt(root, "boundary", "post_boundary_extension"); err != nil || extended {
		return errors.New("R3C boundary extension mismatch")
	}
	if s, _ := stringAt(root, "runner", "path"); s != "cmd/val03r-recovery-oos" {
		return errors.New("R3C runner path mismatch")
	}
	if locked, err := boolAt(root, "runner", "performance_lock_before_authorization"); err != nil || !locked {
		return errors.New("R3C performance lock mismatch")
	}
	if s, _ := stringAt(root, "runner", "execution_authority"); s != "NONE" {
		return errors.New("R3C runner authority mismatch")
	}
	if !sameStrings(mustStrings(root, "runner", "allowed_modes"), []string{"--contract-audit", "--preflight-only", "--execute"}) {
		return errors.New("R3C allowed modes mismatch")
	}
	for _, p := range []string{"runner.arbitrary_threshold_override", "runner.contaminated_runner_reenabled", "runner.creates_fill", "lifecycle.performance_execution_started", "lifecycle.performance_artifacts_written", "safety.allow_live_trading", "safety.broker_execution_allowed", "safety.execution_enabled"} {
		v, err := boolAt(root, splitDot(p)...)
		if err != nil || v {
			return fmt.Errorf("R3C boolean guard mismatch %s", p)
		}
	}
	if n, _ := intAt(root, "lifecycle", "performance_run_count"); n != 0 {
		return errors.New("R3C lifecycle run count mismatch")
	}
	if n, _ := intAt(root, "bootstrap", "replicates"); n != 10000 {
		return errors.New("R3C bootstrap replicate mismatch")
	}
	if n, _ := intAt(root, "sample_gates", "recovery_episode_floor"); n != 30 {
		return errors.New("R3C recovery floor mismatch")
	}
	if n, _ := intAt(root, "sample_gates", "paired_placebo_floor"); n != 30 {
		return errors.New("R3C paired floor mismatch")
	}
	if n, _ := intAt(root, "sample_gates", "effective_block_floor"); n != 12 {
		return errors.New("R3C block floor mismatch")
	}
	if f, _ := numberAt(root, "sample_gates", "instrument_concentration_ceiling"); f != .4 {
		return errors.New("R3C concentration ceiling mismatch")
	}
	wantTests := []string{"matched_non_signal_placebo", "timestamp_placebo", "secondary_sign_permutation", "top_5_percent_exclusion", "leave_one_instrument_out", "regime_slices", "theme_slices", "sma_19_49_199", "sma_21_51_201", "atr_0_90", "atr_1_10", "stress_cost", "overlap_sensitivity", "zero_and_buy_hold_baselines"}
	if tests := mustStrings(root, "falsification_suite", "tests"); !sameStrings(tests, wantTests) {
		return errors.New("R3C falsification suite mismatch")
	}
	if n, _ := intAt(root, "sample_gates", "instrument_floor"); n != 6 {
		return errors.New("R3C instrument floor mismatch")
	}
	if n, _ := intAt(root, "sample_gates", "calendar_regime_cell_floor"); n != 3 {
		return errors.New("R3C slice floor mismatch")
	}
	if authority, err := stringAt(root, "safety", "execution_authority"); err != nil || authority != "NONE" {
		return errors.New("R3C safety authority mismatch")
	}
	if leverage, err := numberAt(root, "safety", "maximum_leverage"); err != nil || leverage != 1 {
		return errors.New("R3C leverage mismatch")
	}
	for name, path := range r3cRunnerPaths {
		want, err := stringAt(root, "runner", "source_blobs_sha1", name)
		if err != nil || want == "" {
			return fmt.Errorf("R3C source identity missing %s", name)
		}
		got, err := gitBlobSHA1(path)
		if err != nil || got != want {
			return fmt.Errorf("R3C source identity mismatch %s", name)
		}
	}
	return nil
}

func splitDot(path string) []string {
	result := []string{}
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '.' {
			result = append(result, path[start:i])
			start = i + 1
		}
	}
	return result
}

func mustStrings(root map[string]any, path ...string) []string {
	v, err := stringsAt(root, path...)
	if err != nil {
		return nil
	}
	return v
}

func boolAt(root map[string]any, path ...string) (bool, error) {
	v, err := valueAt(root, path...)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, errors.New("expected boolean")
	}
	return b, nil
}

func unmarshalStrictObject(b []byte, target *map[string]any) error {
	if err := json.Unmarshal(b, target); err != nil {
		return err
	}
	if *target == nil {
		return errors.New("expected JSON object")
	}
	return nil
}
