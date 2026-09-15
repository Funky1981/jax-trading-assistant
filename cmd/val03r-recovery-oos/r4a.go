package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	r4aFreezePath        = "Docs/validation/results/VAL-03R4A-RECOVERY-OOS-EXECUTION-FREEZE.json"
	r4bAuthorizationPath = "Docs/validation/results/VAL-03R4B-EXECUTION-AUTHORIZATION.json"
	r4aAttemptSHA        = "8dc3874be5d4593c82fc77c3b1090a6fa55b4787b72e0a09a23bbd8066e05d96"
	r4aForensicSHA       = "70941d082cd1d430b684017129341370b25e7fe845730169dfd5a3a847d85624"
	r4aPreflightSHA      = "2d5f7fc479b6777854987fe1954114a74a528015d0952ed086666aad635b777f"
)

var r4aRunnerPaths = map[string]string{
	"main.go":            "cmd/val03r-recovery-oos/main.go",
	"config.go":          "cmd/val03r-recovery-oos/config.go",
	"contract.go":        "cmd/val03r-recovery-oos/contract.go",
	"recovery.go":        "cmd/val03r-recovery-oos/recovery.go",
	"r3a.go":             "cmd/val03r-recovery-oos/r3a.go",
	"r3b.go":             "cmd/val03r-recovery-oos/r3b.go",
	"structural.go":      "cmd/val03r-recovery-oos/structural.go",
	"r3c.go":             "cmd/val03r-recovery-oos/r3c.go",
	"r4a.go":             "cmd/val03r-recovery-oos/r4a.go",
	"val03b_contract.go": "libs/marketdata/val03b_contract.go",
}

func r4aValidateFreeze() error {
	b, err := os.ReadFile(r4aFreezePath)
	if err != nil {
		return fmt.Errorf("read R4A execution freeze: %w", err)
	}
	var root map[string]any
	if err := unmarshalStrictObject(b, &root); err != nil {
		return err
	}
	for _, key := range []string{"contract_id", "status", "parent", "attempt_1", "forensic", "preflight", "canonical_contract", "scientific_terms", "runner", "authorization", "run_state", "lifecycle", "safety"} {
		if _, ok := root[key]; !ok {
			return fmt.Errorf("R4A freeze missing %s", key)
		}
	}
	if s, _ := stringAt(root, "contract_id"); s != "jax.val-03r4a.recovery-oos-execution-freeze/v1" {
		return errors.New("R4A contract identity mismatch")
	}
	if s, _ := stringAt(root, "status"); s != "FROZEN_FOR_VAL03R4B_EXTERNAL_AUTHORIZATION" {
		return errors.New("R4A status mismatch")
	}
	checks := map[string]string{
		"parent.r3c_execution_freeze_sha256": "e42d6fb9aff86d51951fc5a0dc251059da65e73264968d972e666905fbe560d3",
		"attempt_1.integrity_sha256":         r4aAttemptSHA,
		"forensic.sha256":                    r4aForensicSHA,
		"preflight.sha256":                   r4aPreflightSHA,
		"canonical_contract.factor_helper":   "marketdata.DeriveVAL03BSplitFactor",
		"canonical_contract.boundary_helper": "marketdata.IsVAL03BStructuralFactorBoundary",
		"canonical_contract.tolerance":       "5e-4",
		"authorization.old_r4_status":        "HISTORICAL / NOT REUSABLE",
		"authorization.future_path":          r4bAuthorizationPath,
		"authorization.future_contract_id":   "jax.val-03r4b.execution-authorization/v1",
		"authorization.future_value":         "GO_VAL03R4B_SINGLE_RECOVERY_OOS_EXECUTION",
		"run_state.path":                     r3aRunStatePath,
		"run_state.status":                   "ABSENT",
		"scientific_terms.candidate":         "ma_crossover_v1",
		"scientific_terms.recovery_boundary": "2025-01-01..2026-09-11",
		"scientific_terms.threshold":         "0.60",
	}
	for path, want := range checks {
		got, err := stringAt(root, splitDot(path)...)
		if err != nil || got != want {
			return fmt.Errorf("R4A field %s mismatch", path)
		}
	}
	for _, path := range []string{"attempt_1.started_once", "attempt_1.outcomes_exposed", "attempt_1.signals_calculated", "attempt_1.episodes_calculated", "attempt_1.returns_calculated", "preflight.outcome_calculations", "lifecycle.performance_execution_started", "lifecycle.performance_artifacts_written", "safety.allow_live_trading", "safety.broker_execution_allowed", "safety.execution_enabled"} {
		v, err := boolAt(root, splitDot(path)...)
		if err != nil || v {
			return fmt.Errorf("R4A boolean guard mismatch %s", path)
		}
	}
	for _, path := range []string{"attempt_1.result_artifacts_absent", "attempt_1.formal_run_state_absent"} {
		v, err := boolAt(root, splitDot(path)...)
		if err != nil || !v {
			return fmt.Errorf("R4A required absence flag mismatch %s", path)
		}
	}
	if n, err := intAt(root, "attempt_1", "formal_recovery_run_count"); err != nil || n != 0 {
		return errors.New("R4A attempt count mismatch")
	}
	if n, err := intAt(root, "lifecycle", "performance_run_count"); err != nil || n != 0 {
		return errors.New("R4A lifecycle count mismatch")
	}
	if leverage, err := numberAt(root, "safety", "maximum_leverage"); err != nil || leverage != 1 {
		return errors.New("R4A leverage mismatch")
	}
	if s, _ := stringAt(root, "runner", "execution_authority"); s != "NONE" {
		return errors.New("R4A runner authority mismatch")
	}
	if !sameStrings(mustStrings(root, "runner", "allowed_modes"), []string{"--contract-audit", "--preflight-only", "--execute"}) {
		return errors.New("R4A modes mismatch")
	}
	if blobs, err := valueAt(root, "runner", "source_blobs_sha1"); err != nil {
		return err
	} else if m, ok := blobs.(map[string]any); !ok || len(m) != len(r4aRunnerPaths) {
		return errors.New("R4A source identity set mismatch")
	}
	for name, path := range r4aRunnerPaths {
		want, err := stringAt(root, "runner", "source_blobs_sha1", name)
		if err != nil || want == "" {
			return fmt.Errorf("R4A source identity missing %s", name)
		}
		got, err := gitBlobSHA1(path)
		if err != nil || got != want {
			return fmt.Errorf("R4A source identity mismatch %s", name)
		}
	}
	return nil
}

func r4aValidateAuthorizationBytes(b []byte, freezeSHA string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	allowed := map[string]bool{"contract_id": true, "candidate": true, "recovery_boundary": true, "execution_freeze_sha256": true, "performance_run_count_before": true, "execute_once": true, "external_authorization": true}
	if len(raw) != len(allowed) {
		return errors.New("incomplete R4B authorization")
	}
	for key := range raw {
		if !allowed[key] || raw[key] == nil {
			return errors.New("invalid R4B authorization fields")
		}
	}
	var a struct {
		ContractID                string `json:"contract_id"`
		Candidate                 string `json:"candidate"`
		RecoveryBoundary          string `json:"recovery_boundary"`
		ExecutionFreezeSHA256     string `json:"execution_freeze_sha256"`
		PerformanceRunCountBefore int    `json:"performance_run_count_before"`
		ExecuteOnce               bool   `json:"execute_once"`
		ExternalAuthorization     string `json:"external_authorization"`
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	if a.ContractID != "jax.val-03r4b.execution-authorization/v1" || a.Candidate != "ma_crossover_v1" || a.RecoveryBoundary != "2025-01-01..2026-09-11" || a.ExecutionFreezeSHA256 != freezeSHA || a.PerformanceRunCountBefore != 0 || !a.ExecuteOnce || a.ExternalAuthorization != "GO_VAL03R4B_SINGLE_RECOVERY_OOS_EXECUTION" {
		return errors.New("R4B authorization contract mismatch")
	}
	return nil
}
