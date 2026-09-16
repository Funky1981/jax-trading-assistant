package main

import (
	"errors"
	"fmt"
	"os"
)

const (
	r4a1FreezePath       = "Docs/validation/results/VAL-03R4A1-RECOVERY-OOS-EXECUTION-FREEZE.json"
	r4a1ContractID       = "jax.val-03r4a1.recovery-oos-execution-freeze/v1"
	r4a1ParentFreezeSHA  = "f359db05cd3a4bfe2a96b0dc2a3b993a710410695d15329db60cf8de8f8d0eef"
	r4a1FutureAuthPath   = "Docs/validation/results/VAL-03R4B-EXECUTION-AUTHORIZATION.json"
	r4a1FutureContractID = "jax.val-03r4b.execution-authorization/v1"
)

var r4a1RunnerPaths = map[string]string{
	"main.go":            "cmd/val03r-recovery-oos/main.go",
	"config.go":          "cmd/val03r-recovery-oos/config.go",
	"contract.go":        "cmd/val03r-recovery-oos/contract.go",
	"recovery.go":        "cmd/val03r-recovery-oos/recovery.go",
	"r3a.go":             "cmd/val03r-recovery-oos/r3a.go",
	"r3b.go":             "cmd/val03r-recovery-oos/r3b.go",
	"structural.go":      "cmd/val03r-recovery-oos/structural.go",
	"r3c.go":             "cmd/val03r-recovery-oos/r3c.go",
	"r4a.go":             "cmd/val03r-recovery-oos/r4a.go",
	"r4a1.go":            "cmd/val03r-recovery-oos/r4a1.go",
	"val03b_contract.go": "libs/marketdata/val03b_contract.go",
}

func r4a1ValidateFreeze() error {
	b, err := os.ReadFile(r4a1FreezePath)
	if err != nil {
		return fmt.Errorf("read R4A1 execution freeze: %w", err)
	}
	var root map[string]any
	if err := unmarshalStrictObject(b, &root); err != nil {
		return err
	}
	for _, key := range []string{"contract_id", "status", "parent", "r4a_evidence", "canonical_contract", "scientific_terms", "runner", "authorization", "run_state", "lifecycle", "safety"} {
		if _, ok := root[key]; !ok {
			return fmt.Errorf("R4A1 freeze missing %s", key)
		}
	}
	if s, _ := stringAt(root, "contract_id"); s != r4a1ContractID {
		return errors.New("R4A1 contract identity mismatch")
	}
	if s, _ := stringAt(root, "status"); s != "FROZEN_FOR_VAL03R4B_EXTERNAL_AUTHORIZATION" {
		return errors.New("R4A1 status mismatch")
	}
	checks := map[string]string{
		"parent.r4a_freeze_sha256":           r4a1ParentFreezeSHA,
		"r4a_evidence.forensic_sha256":       r4aForensicSHA,
		"r4a_evidence.preflight_sha256":      r4aPreflightSHA,
		"canonical_contract.factor_helper":   "marketdata.DeriveVAL03BSplitFactor",
		"canonical_contract.boundary_helper": "marketdata.IsVAL03BStructuralFactorBoundary",
		"canonical_contract.tolerance":       "5e-4",
		"canonical_contract.shared_path":     "r3aRunDataIntegrityPreflight",
		"authorization.old_r4_status":        "HISTORICAL / NOT REUSABLE",
		"authorization.future_path":          r4a1FutureAuthPath,
		"authorization.future_contract_id":   r4a1FutureContractID,
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
			return fmt.Errorf("R4A1 field %s mismatch", path)
		}
	}
	for _, path := range []string{
		"lifecycle.performance_execution_started",
		"lifecycle.performance_artifacts_written",
		"safety.allow_live_trading",
		"safety.broker_execution_allowed",
		"safety.execution_enabled",
	} {
		v, err := boolAt(root, splitDot(path)...)
		if err != nil || v {
			return fmt.Errorf("R4A1 boolean guard mismatch %s", path)
		}
	}
	for _, path := range []string{"run_state.absent", "authorization.r4b_absent"} {
		v, err := boolAt(root, splitDot(path)...)
		if err != nil || !v {
			return fmt.Errorf("R4A1 absence guard mismatch %s", path)
		}
	}
	if n, err := intAt(root, "lifecycle", "performance_run_count"); err != nil || n != 0 {
		return errors.New("R4A1 lifecycle count mismatch")
	}
	if leverage, err := numberAt(root, "safety", "maximum_leverage"); err != nil || leverage != 1 {
		return errors.New("R4A1 leverage mismatch")
	}
	if s, _ := stringAt(root, "runner", "execution_authority"); s != "NONE" {
		return errors.New("R4A1 runner authority mismatch")
	}
	if creates, err := boolAt(root, "runner", "creates_fill"); err != nil || creates {
		return errors.New("R4A1 fill authority mismatch")
	}
	if !sameStrings(mustStrings(root, "runner", "allowed_modes"), []string{"--contract-audit", "--preflight-only", "--execute"}) {
		return errors.New("R4A1 modes mismatch")
	}
	blobs, err := valueAt(root, "runner", "source_blobs_sha1")
	if err != nil {
		return err
	}
	m, ok := blobs.(map[string]any)
	if !ok || len(m) != len(r4a1RunnerPaths) {
		return errors.New("R4A1 source identity set mismatch")
	}
	for name, path := range r4a1RunnerPaths {
		want, err := stringAt(root, "runner", "source_blobs_sha1", name)
		if err != nil || want == "" {
			return fmt.Errorf("R4A1 source identity missing %s", name)
		}
		got, err := gitBlobSHA1(path)
		if err != nil || got != want {
			return fmt.Errorf("R4A1 source identity mismatch %s", name)
		}
	}
	return nil
}
