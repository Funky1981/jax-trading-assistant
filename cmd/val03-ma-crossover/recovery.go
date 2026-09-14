package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	recoveryManifestPath = "Docs/validation/manifests/VAL-03R2-ma_crossover_v1-RECOVERY-PREREGISTRATION.json"
	parentManifestSHA    = "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a"
)

type recoveryManifest struct {
	ContractID      string `json:"contract_id"`
	ContractVersion string `json:"contract_version"`
	Status          string `json:"status"`
	Parent          struct {
		CandidateID    string `json:"candidate_id"`
		Manifest       string `json:"manifest"`
		ManifestSHA256 string `json:"manifest_sha256"`
		Version        string `json:"version"`
	} `json:"parent"`
	Candidate struct {
		ID                   string  `json:"candidate_id"`
		StrategyVersion      string  `json:"strategy_version"`
		PrimaryClaim         string  `json:"primary_claim"`
		ActionableConfidence float64 `json:"actionable_confidence_threshold"`
		SMAFast              int     `json:"sma_fast"`
		SMAMedium            int     `json:"sma_medium"`
		SMASlow              int     `json:"sma_slow"`
		ATRPeriod            int     `json:"atr_period"`
		AvgVolumePeriod      int     `json:"avg_volume_period"`
		TargetATR            float64 `json:"target_atr"`
		StopATR              float64 `json:"stop_atr"`
		HoldingPeriod        int     `json:"holding_period_sessions"`
		ReferenceCapitalUSD  int     `json:"reference_capital_usd"`
		QuantityConvention   string  `json:"quantity_convention"`
	} `json:"candidate"`
	InheritedPolicies []string `json:"inherited_policies"`
	SourceIdentities  map[string]struct {
		Path     string `json:"path"`
		BlobSHA1 string `json:"blob_sha1"`
	} `json:"source_identities"`
	Boundary struct {
		Start              string `json:"start"`
		End                string `json:"end"`
		ExcludedIncomplete string `json:"excluded_incomplete_session"`
		ExtensionAllowed   bool   `json:"post_result_extension_allowed"`
	} `json:"recovery_oos_boundary"`
	StructuralFeasibility struct {
		UniverseInstruments int `json:"universe_instruments"`
		RecoveryYears       int `json:"recovery_years"`
		Max2025Blocks       int `json:"maximum_theoretical_2025_blocks"`
		MaxRecoveryBlocks   int `json:"maximum_theoretical_recovery_blocks"`
		EffectiveBlockFloor int `json:"effective_block_floor"`
	} `json:"structural_feasibility"`
	Warmup struct {
		Source              string `json:"source"`
		PreRecoveryState    string `json:"pre_recovery_state"`
		PerformanceIncluded bool   `json:"pre_recovery_performance_included"`
	} `json:"warmup"`
	Provider struct {
		Name         string   `json:"provider"`
		Feed         string   `json:"feed"`
		Timeframe    string   `json:"timeframe"`
		Adjustments  []string `json:"adjustment_families"`
		Fallback     string   `json:"fallback"`
		PaidSpendUSD int      `json:"paid_spend_usd"`
	} `json:"provider_contract"`
	SampleFloors struct {
		Development int `json:"development"`
		Validation  int `json:"validation"`
		RecoveryOOS int `json:"recovery_oos"`
		Paired      int `json:"paired"`
		Blocks      int `json:"instrument_year_blocks"`
		Instruments int `json:"instruments"`
		Slices      int `json:"year_regime_cells"`
	} `json:"sample_floors"`
	Gates struct {
		ConcentrationCeiling float64 `json:"concentration_ceiling"`
	} `json:"promotion_gates"`
	Safety struct {
		ExecutionAuthority string `json:"execution_authority"`
		CreatesFill        bool   `json:"creates_fill"`
		ForwardPaper       string `json:"forward_paper"`
		Phase13            string `json:"phase_13"`
	} `json:"safety"`
	ContaminatedRun struct {
		Artifact       string `json:"artifact"`
		RunCount       int    `json:"formal_oos_run_count"`
		NoRerun        bool   `json:"no_rerun"`
		Classification string `json:"classification"`
	} `json:"contaminated_run_guard"`
	Audit struct {
		NoDataAccess          bool `json:"no_market_data_access"`
		NoOutcomeAccess       bool `json:"no_outcome_access"`
		NoExecution           bool `json:"no_execution"`
		NoPostResultExtension bool `json:"no_post_result_extension"`
	} `json:"outcome_free_audit"`
}

func runRecoveryContractAudit() error {
	parentBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read parent manifest: %w", err)
	}
	if got := sha256Hex(parentBytes); got != parentManifestSHA {
		return fmt.Errorf("parent manifest hash mismatch: %s", got)
	}
	recoveryBytes, err := os.ReadFile(recoveryManifestPath)
	if err != nil {
		return fmt.Errorf("read recovery manifest: %w", err)
	}
	var m recoveryManifest
	if err := json.Unmarshal(recoveryBytes, &m); err != nil {
		return fmt.Errorf("decode recovery manifest: %w", err)
	}
	if err := validateRecoveryManifest(m); err != nil {
		return err
	}
	guard, err := contaminatedRunGuard(integrityPath)
	if err != nil || !guard {
		return errors.New("old contaminated runner guard is not active")
	}
	if _, err := os.Stat(".runtime/val03r2"); err == nil {
		return errors.New("recovery data directory exists; outcome-free audit refuses local recovery data")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect recovery data directory: %w", err)
	}
	for _, source := range m.SourceIdentities {
		if source.Path == "" || source.BlobSHA1 == "" {
			return errors.New("recovery source identity is incomplete")
		}
	}
	for key, expected := range m.SourceIdentities {
		actual, err := gitBlobSHA1(expected.Path)
		if err != nil {
			return fmt.Errorf("hash recovery source %s: %w", key, err)
		}
		if actual != expected.BlobSHA1 {
			return fmt.Errorf("recovery source identity mismatch for %s: got %s want %s", key, actual, expected.BlobSHA1)
		}
	}
	fmt.Printf("VAL03R2_CONTRACT_AUDIT=PASS identity=%s candidate=%s boundary=%s..%s validation_floor=%d no_market_data=true no_outcomes=true no_2025=true no_2026=true\n", m.ContractID, m.Candidate.ID, m.Boundary.Start, m.Boundary.End, m.SampleFloors.Validation)
	return nil
}

func gitBlobSHA1(path string) (string, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	header := fmt.Sprintf("blob %d\x00", len(b))
	h := sha1.New()
	_, _ = h.Write([]byte(header))
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func recoveryDateAllowed(date string) bool {
	return date >= "2025-01-01" && date <= "2026-09-11"
}

func validateRecoveryManifest(m recoveryManifest) error {
	if m.ContractID != "jax.val-03r2.ma-crossover-recovery-preregistration" || m.ContractVersion != "v1" || m.Status != "FROZEN_FOR_EXTERNAL_REVIEW" {
		return errors.New("recovery identity/status mismatch")
	}
	if m.Parent.CandidateID != "ma_crossover_v1" || m.Parent.Version != "v1.5" || m.Parent.Manifest != manifestPath || m.Parent.ManifestSHA256 != parentManifestSHA {
		return errors.New("parent manifest binding mismatch")
	}
	if m.Candidate.ID != "ma_crossover_v1" || m.Candidate.StrategyVersion != "ma_crossover_v1" || m.Candidate.PrimaryClaim != "BULLISH_LONG_ONLY" || m.Candidate.ActionableConfidence != .6 || m.Candidate.SMAFast != 20 || m.Candidate.SMAMedium != 50 || m.Candidate.SMASlow != 200 || m.Candidate.ATRPeriod != 14 || m.Candidate.AvgVolumePeriod != 20 || m.Candidate.StopATR != 1 || m.Candidate.TargetATR != 3 || m.Candidate.HoldingPeriod != 20 || m.Candidate.ReferenceCapitalUSD != 10000 || m.Candidate.QuantityConvention != "WHOLE_SHARES" {
		return errors.New("candidate or strategy parameter binding mismatch")
	}
	if !sameStrings(m.InheritedPolicies, []string{"CEIL_5_PERCENT_V1", "CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1", "SECONDARY_DIRECTIONAL_PRICE_EFFECT_V1", "SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1", "MIXED_INSTRUMENT_YEAR_ONLY_V1", "FAIL_CLOSED_UNKNOWN_STATUS_V1", "SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1"}) {
		return errors.New("inherited policy identity mismatch")
	}
	if m.Boundary.Start != "2025-01-01" || m.Boundary.End != "2026-09-11" || m.Boundary.ExcludedIncomplete != "2026-09-14" || m.Boundary.ExtensionAllowed {
		return errors.New("recovery boundary mismatch")
	}
	if m.StructuralFeasibility.UniverseInstruments != 9 || m.StructuralFeasibility.RecoveryYears != 2 || m.StructuralFeasibility.Max2025Blocks != 9 || m.StructuralFeasibility.MaxRecoveryBlocks != 18 || m.StructuralFeasibility.EffectiveBlockFloor != 12 {
		return errors.New("structural feasibility mismatch")
	}
	if m.Warmup.Source == "" || m.Warmup.PreRecoveryState != "WARMUP_ONLY" || m.Warmup.PerformanceIncluded {
		return errors.New("warm-up contract mismatch")
	}
	if m.Provider.Name != "Alpaca" || m.Provider.Feed != "SIP" || m.Provider.Timeframe != "1Day" || !sameStrings(m.Provider.Adjustments, []string{"RAW", "SPLIT", "SPLIT+SPIN-OFF"}) || m.Provider.Fallback != "NONE" || m.Provider.PaidSpendUSD != 0 {
		return errors.New("provider contract mismatch")
	}
	if m.SampleFloors.Development != 90 || m.SampleFloors.Validation != 30 || m.SampleFloors.RecoveryOOS != 30 || m.SampleFloors.Paired != 30 || m.SampleFloors.Blocks != 12 || m.SampleFloors.Instruments != 6 || m.SampleFloors.Slices != 3 || m.Gates.ConcentrationCeiling != .4 {
		return errors.New("recovery sample/gate mismatch")
	}
	if m.Safety.ExecutionAuthority != "NONE" || m.Safety.CreatesFill || m.Safety.ForwardPaper != "NOT_STARTED" || m.Safety.Phase13 != "NOT_STARTED" {
		return errors.New("safety contract mismatch")
	}
	if m.ContaminatedRun.RunCount != 1 || !m.ContaminatedRun.NoRerun || m.ContaminatedRun.Classification != "CONTAMINATED_FOR_FORMAL_OOS" || m.ContaminatedRun.Artifact == "" {
		return errors.New("contaminated-run binding mismatch")
	}
	if !m.Audit.NoDataAccess || !m.Audit.NoOutcomeAccess || !m.Audit.NoExecution || !m.Audit.NoPostResultExtension {
		return errors.New("outcome-free audit contract mismatch")
	}
	if strings.Contains(m.Boundary.End, "2026-09-14") || m.Boundary.End >= "2026-09-14" {
		return errors.New("incomplete 2026-09-14 session included")
	}
	return nil
}

func recoveryManifestSHA(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
