package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type r3bFreezeArtifact struct {
	ContractID  string `json:"contract_id"`
	Status      string `json:"status"`
	CandidateID string `json:"candidate_id"`
	Parent      struct {
		ManifestSHA256 string `json:"manifest_sha256"`
		R2SHA256       string `json:"r2_manifest_sha256"`
		R3SHA256       string `json:"r3_conformance_sha256"`
	} `json:"parent"`
	Runner struct {
		Path                      string            `json:"path"`
		SourceBlobs               map[string]string `json:"source_blobs_sha1"`
		AllowedModes              []string          `json:"allowed_modes"`
		ArbitraryThreshold        bool              `json:"arbitrary_threshold_override"`
		PerformanceLockBeforeAuth bool              `json:"performance_lock_before_authorization"`
		ContaminatedRunner        bool              `json:"contaminated_runner_reenabled"`
		ExecutionAuthority        string            `json:"execution_authority"`
		CreatesFill               bool              `json:"creates_fill"`
	} `json:"runner"`
	Execution struct {
		Threshold              float64 `json:"threshold"`
		ThresholdSource        string  `json:"threshold_source"`
		StressFormula          string  `json:"stress_formula"`
		PayloadVerification    string  `json:"payload_verification"`
		StructuralRevalidation string  `json:"structural_revalidation"`
		StartMarker            string  `json:"start_marker"`
		CompletionTransition   string  `json:"completion_transition"`
		ResultSchema           string  `json:"result_schema"`
	} `json:"execution"`
	Randomness struct {
		SeedManifest string `json:"seed_manifest"`
		Bootstrap    string `json:"bootstrap"`
		Timestamp    string `json:"timestamp_placebo"`
	} `json:"randomness"`
	Lifecycle struct {
		Started bool   `json:"performance_execution_started"`
		Written bool   `json:"performance_artifacts_written"`
		Runs    int    `json:"performance_run_count"`
		Status  string `json:"recovery_oos_status"`
		Auth    string `json:"authorization_status"`
	} `json:"lifecycle"`
}

func r3bValidateFreeze() error {
	b, err := os.ReadFile(r3bFreezePath)
	if err != nil {
		return fmt.Errorf("read R3B execution freeze: %w", err)
	}
	var v r3bFreezeArtifact
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.ContractID != "jax.val-03r3b.recovery-oos-execution-freeze/v1" || v.Status != "FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION" || v.CandidateID != "ma_crossover_v1" || v.Parent.ManifestSHA256 != parentManifestSHA || v.Parent.R2SHA256 != r3aR2ManifestSHA || v.Parent.R3SHA256 == "" {
		return errors.New("R3B freeze identity mismatch")
	}
	if v.Runner.Path != "cmd/val03r-recovery-oos" || !sameStrings(v.Runner.AllowedModes, []string{"--contract-audit", "--preflight-only", "--execute"}) || v.Runner.ArbitraryThreshold || !v.Runner.PerformanceLockBeforeAuth || v.Runner.ContaminatedRunner || v.Runner.ExecutionAuthority != "NONE" || v.Runner.CreatesFill {
		return errors.New("R3B runner lock mismatch")
	}
	if v.Execution.Threshold != .6 || v.Execution.ThresholdSource != "parent_manifest" || v.Execution.StressFormula != "fixed USD 0.50 per order plus 10 bps per leg, both legs" || v.Execution.PayloadVerification != "REQUIRED_IMMEDIATELY_BEFORE_DATA_LOAD" || v.Execution.StructuralRevalidation != "REQUIRED_BEFORE_START_MARKER" || v.Execution.StartMarker != "EXCLUSIVE_STARTED_ONCE_BEFORE_OUTCOME_CALCULATION" || v.Execution.CompletionTransition != "ATOMIC_COMPLETED_ONCE_AFTER_ALL_RESULT_ARTIFACTS" || v.Execution.ResultSchema != "jax.val-03r4.recovery-oos-results/v1" {
		return errors.New("R3B execution semantics mismatch")
	}
	if v.Randomness.SeedManifest != "PARENT_V1.5_SHA256" || v.Randomness.Bootstrap != "SHA256_COUNTER_DERIVED_FIRST_8_BIG_ENDIAN_BYTES" || v.Randomness.Timestamp != "FULL_SHA256_SEED_AND_SCORE" {
		return errors.New("R3B randomness contract mismatch")
	}
	if v.Lifecycle.Started || v.Lifecycle.Written || v.Lifecycle.Runs != 0 || v.Lifecycle.Status != "NOT_STARTED / FROZEN_FOR_FUTURE_AUTHORIZATION" || v.Lifecycle.Auth != "ABSENT_DURING_R3B" {
		return errors.New("R3B lifecycle mismatch")
	}
	paths := map[string]string{"main.go": "cmd/val03r-recovery-oos/main.go", "config.go": "cmd/val03r-recovery-oos/config.go", "contract.go": "cmd/val03r-recovery-oos/contract.go", "recovery.go": "cmd/val03r-recovery-oos/recovery.go", "r3a.go": "cmd/val03r-recovery-oos/r3a.go", "structural.go": "cmd/val03r-recovery-oos/structural.go", "r3b.go": "cmd/val03r-recovery-oos/r3b.go"}
	if len(v.Runner.SourceBlobs) != len(paths) {
		return errors.New("R3B runner identity set mismatch")
	}
	for name, path := range paths {
		want := v.Runner.SourceBlobs[name]
		got, err := gitBlobSHA1(path)
		if err != nil || want == "" || got != want {
			return fmt.Errorf("R3B runner identity mismatch %s", name)
		}
	}
	return nil
}
