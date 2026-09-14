// Command val03r3-recovery-preflight performs the outcome-free VAL-03R3
// recovery contract audit and pre-holdout readiness count. It cannot execute
// recovery OOS performance.
package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	parentManifestPath     = "Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json"
	recoveryManifestPath   = "Docs/validation/manifests/VAL-03R2-ma_crossover_v1-RECOVERY-PREREGISTRATION.json"
	integrityPath          = "Docs/validation/results/VAL-03-HYP-MA-001-INTEGRITY-REVIEW.json"
	barFamilyPath          = "Docs/validation/results/VAL-03B-DATASET-READINESS.json"
	readinessOutputPath    = "Docs/validation/results/VAL-03R3-PRE-HOLDOUT-READINESS.json"
	oldRawRoot             = ".runtime/val03b/raw"
	recoveryRawRoot        = ".runtime/val03r3/raw"
	parentManifestSHA256   = "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a"
	recoveryManifestSHA256 = "c19a4cfc774a7bf53466d9bcf5705bd3813f69258a9600291883db48aab490eb"
	barFamilySHA256        = "78c1fbeec2122eebba1a8dbe21c2edfc3c569377770561afe9e128cd633037e5"
	recoveryStart          = "2025-01-01"
	recoveryEnd            = "2026-09-11"
	developmentStart       = "2016-01-01"
	developmentEnd         = "2020-12-31"
	validationStart        = "2021-01-01"
	validationEnd          = "2022-12-31"
	factorTolerance        = 5e-4
)

var expectedUniverse = []string{"SPY", "QQQ", "IWM", "DIA", "XLK", "XLF", "XLE", "TLT", "GLD"}

var expectedInheritedPolicies = []string{
	"CEIL_5_PERCENT_V1",
	"CALENDAR_YEAR_X_SPY_SMA200_REGIME_V1",
	"SECONDARY_DIRECTIONAL_PRICE_EFFECT_V1",
	"SHA256_INDEPENDENT_SIGN_ASSIGNMENT_V1",
	"MIXED_INSTRUMENT_YEAR_ONLY_V1",
	"FAIL_CLOSED_UNKNOWN_STATUS_V1",
	"SIGN_X_ABSOLUTE_MAGNITUDE_NULL_V1",
}

type recoveryManifest struct {
	ContractID      string `json:"contract_id"`
	ContractVersion string `json:"contract_version"`
	Status          string `json:"status"`
	Parent          struct {
		CandidateID    string `json:"candidate_id"`
		Manifest       string `json:"manifest"`
		Version        string `json:"version"`
		ManifestSHA256 string `json:"manifest_sha256"`
	} `json:"parent"`
	Candidate struct {
		CandidateID                   string  `json:"candidate_id"`
		StrategyVersion               string  `json:"strategy_version"`
		PrimaryClaim                  string  `json:"primary_claim"`
		ActionableConfidenceThreshold float64 `json:"actionable_confidence_threshold"`
		SMAFast                       int     `json:"sma_fast"`
		SMAMedium                     int     `json:"sma_medium"`
		SMASlow                       int     `json:"sma_slow"`
		ATRPeriod                     int     `json:"atr_period"`
		AvgVolumePeriod               int     `json:"avg_volume_period"`
		StopATR                       float64 `json:"stop_atr"`
		TargetATR                     float64 `json:"target_atr"`
		HoldingPeriodSessions         int     `json:"holding_period_sessions"`
		ReferenceCapitalUSD           int     `json:"reference_capital_usd"`
		QuantityConvention            string  `json:"quantity_convention"`
		ParameterChangePermitted      bool    `json:"parameter_change_permitted"`
	} `json:"candidate"`
	InheritedPolicies []string `json:"inherited_policies"`
	SourceIdentities  map[string]struct {
		Path     string `json:"path"`
		BlobSHA1 string `json:"blob_sha1"`
	} `json:"source_identities"`
	RecoveryBoundary struct {
		Start                      string `json:"start"`
		End                        string `json:"end"`
		ScoredSessions             string `json:"scored_sessions"`
		ExcludedIncompleteSession  string `json:"excluded_incomplete_session"`
		PostResultExtensionAllowed bool   `json:"post_result_extension_allowed"`
		ExtensionRequiresNewID     bool   `json:"extension_requires_new_preregistration"`
	} `json:"recovery_oos_boundary"`
	StructuralFeasibility struct {
		UniverseInstruments    int  `json:"universe_instruments"`
		RecoveryYears          int  `json:"recovery_years"`
		Max2025Blocks          int  `json:"maximum_theoretical_2025_blocks"`
		MaxRecoveryBlocks      int  `json:"maximum_theoretical_recovery_blocks"`
		EffectiveBlockFloor    int  `json:"effective_block_floor"`
		OutcomeCountsInspected bool `json:"outcome_counts_inspected"`
	} `json:"structural_feasibility"`
	Warmup struct {
		Source                         string `json:"source"`
		Mode                           string `json:"mode"`
		PreRecoveryState               string `json:"pre_recovery_state"`
		PreRecoveryPerformanceIncluded bool   `json:"pre_recovery_performance_included"`
		ContaminatedResultReused       bool   `json:"contaminated_2023_2024_result_reused"`
	} `json:"warmup"`
	Provider struct {
		Provider                    string   `json:"provider"`
		Feed                        string   `json:"feed"`
		Timeframe                   string   `json:"timeframe"`
		Universe                    []string `json:"universe"`
		AdjustmentFamilies          []string `json:"adjustment_families"`
		Route                       string   `json:"route"`
		Fallback                    string   `json:"fallback"`
		PaidSpendUSD                int      `json:"paid_spend_usd"`
		RawBytesBeforeNormalization bool     `json:"raw_bytes_before_normalization"`
		ProviderCallsAuthorizedInR2 bool     `json:"provider_calls_authorized_in_r2"`
		RequestDateGuard            string   `json:"request_date_guard"`
	} `json:"provider_contract"`
	SampleFloors struct {
		Development          int    `json:"development"`
		Validation           int    `json:"validation"`
		RecoveryOOS          int    `json:"recovery_oos"`
		Paired               int    `json:"paired"`
		InstrumentYearBlocks int    `json:"instrument_year_blocks"`
		Instruments          int    `json:"instruments"`
		YearRegimeCells      int    `json:"year_regime_cells"`
		ReadinessFailure     string `json:"development_validation_readiness_failure"`
	} `json:"sample_floors"`
	PromotionGates struct {
		PrimaryMeanNetLongReturn   string  `json:"primary_mean_net_long_return"`
		PrimaryBootstrapLowerBound string  `json:"primary_block_bootstrap_lower_bound"`
		PairedMeanDifference       string  `json:"paired_mean_difference"`
		PairedBootstrapLowerBound  string  `json:"paired_bootstrap_lower_bound"`
		ConcentrationCeiling       float64 `json:"concentration_ceiling"`
		TopFiveConcentration       string  `json:"top_five_concentration"`
		BlockingFalsification      string  `json:"blocking_falsification"`
		DataQuality                string  `json:"data_quality"`
		UnknownMaterialStatus      string  `json:"unknown_material_status"`
		SecondarySignPermutation   string  `json:"secondary_sign_permutation"`
	} `json:"promotion_gates"`
	NoPostResultSalvage struct {
		ParameterTuning            bool `json:"parameter_tuning"`
		ThresholdTuning            bool `json:"threshold_tuning"`
		UniverseSubstitution       bool `json:"universe_substitution"`
		DateExtension              bool `json:"date_extension"`
		InstrumentDroppingOrAdding bool `json:"instrument_dropping_or_adding"`
		CostChange                 bool `json:"cost_change"`
		PlaceboChange              bool `json:"placebo_change"`
		BootstrapChange            bool `json:"bootstrap_change"`
		FalsificationChange        bool `json:"falsification_change"`
		RegimeChange               bool `json:"regime_change"`
		TopFivePolicyChange        bool `json:"top_five_policy_change"`
		NewExperimentRequired      bool `json:"new_experiment_required_for_change"`
	} `json:"no_post_result_salvage"`
	ContaminatedRun struct {
		Artifact           string `json:"artifact"`
		FormalOOSRunCount  int    `json:"formal_oos_run_count"`
		NoRerun            bool   `json:"no_rerun"`
		Classification     string `json:"classification"`
		OldRunnerExecution string `json:"old_runner_execution"`
	} `json:"contaminated_run_guard"`
	OutcomeFreeAudit struct {
		NoMarketDataAccess    bool   `json:"no_market_data_access"`
		No2025Access          bool   `json:"no_2025_access"`
		No2026Access          bool   `json:"no_2026_access"`
		NoSignalCounts        bool   `json:"no_signal_counts"`
		NoEpisodeCounts       bool   `json:"no_episode_counts"`
		NoOutcomeAccess       bool   `json:"no_outcome_access"`
		NoExecution           bool   `json:"no_execution"`
		NoPostResultExtension bool   `json:"no_post_result_extension"`
		Command               string `json:"command"`
	} `json:"outcome_free_audit"`
	Safety struct {
		ExecutionAuthority     string  `json:"execution_authority"`
		CreatesFill            bool    `json:"creates_fill"`
		AllowLiveTrading       bool    `json:"allow_live_trading"`
		BrokerExecutionAllowed bool    `json:"broker_execution_allowed"`
		ExecutionEnabled       bool    `json:"execution_enabled"`
		MaximumLeverage        float64 `json:"maximum_leverage"`
		ForwardPaper           string  `json:"forward_paper"`
		Phase13                string  `json:"phase_13"`
	} `json:"safety"`
}

type integrityRecord struct {
	FinalTerminalClassification string `json:"final_terminal_classification"`
	NoRerun                     bool   `json:"no_rerun"`
	FormalOOSRunCount           int    `json:"formal_oos_run_count"`
}

type familyRef struct {
	Instrument       string   `json:"instrument"`
	Adjustment       string   `json:"adjustment"`
	RawPayloadSHA256 []string `json:"raw_payload_sha256"`
}

type readinessInput struct {
	Families []familyRef `json:"families"`
}

type providerBar struct {
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
	Timestamp string  `json:"t"`
}

type providerPayload struct {
	Bars map[string][]providerBar `json:"bars"`
}

type bar struct {
	Open, High, Low, Close, Volume float64
}

type instrumentData struct {
	Raw, Split, Detector map[string]bar
	Dates                []string
	Boundaries           map[string]bool
}

type signal struct {
	Confidence, Stop, Target float64
}

type readinessResult struct {
	ContractID                  string `json:"contract_id"`
	DevelopmentEligibleEpisodes int    `json:"development_eligible_primary_episodes"`
	ValidationEligibleEpisodes  int    `json:"validation_eligible_primary_episodes"`
	DevelopmentFloor            int    `json:"development_floor"`
	ValidationFloor             int    `json:"validation_floor"`
	DevelopmentReadiness        string `json:"development_readiness"`
	ValidationReadiness         string `json:"validation_readiness"`
	RecoveryReadiness           string `json:"recovery_readiness"`
}

func main() {
	if len(os.Args) != 2 {
		fatal(errors.New("usage: val03r3-recovery-preflight --contract-audit|--pre-holdout-readiness"))
	}
	var err error
	switch os.Args[1] {
	case "--contract-audit":
		err = runContractAudit()
	case "--pre-holdout-readiness":
		err = runPreHoldoutReadiness()
	default:
		err = errors.New("unsupported mode")
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "VAL03R3=FAILED: %v\n", err)
	os.Exit(1)
}

func runContractAudit() error {
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		return err
	}
	if err := validateContaminatedGuard(m); err != nil {
		return err
	}
	for key, source := range m.SourceIdentities {
		actual, err := gitBlobSHA1(source.Path)
		if err != nil {
			return fmt.Errorf("hash source %s: %w", key, err)
		}
		if actual != source.BlobSHA1 {
			return fmt.Errorf("source identity mismatch %s: got %s want %s", key, actual, source.BlobSHA1)
		}
	}
	if _, err := os.Stat(recoveryRawRoot); err == nil {
		return errors.New("recovery raw directory exists before R3 data authorization")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect recovery raw directory: %w", err)
	}
	fmt.Printf("VAL03R3_CONTRACT_AUDIT=PASS candidate=%s boundary=%s..%s threshold=%.2f no_recovery_data=true no_outcomes=true\n", m.Candidate.CandidateID, m.RecoveryBoundary.Start, m.RecoveryBoundary.End, m.Candidate.ActionableConfidenceThreshold)
	return nil
}

func loadAndValidateRecoveryManifest() (recoveryManifest, error) {
	parentBytes, err := os.ReadFile(parentManifestPath)
	if err != nil {
		return recoveryManifest{}, fmt.Errorf("read parent manifest: %w", err)
	}
	if got := sha256Hex(parentBytes); got != parentManifestSHA256 {
		return recoveryManifest{}, fmt.Errorf("parent manifest hash mismatch: %s", got)
	}
	recoveryBytes, err := os.ReadFile(recoveryManifestPath)
	if err != nil {
		return recoveryManifest{}, fmt.Errorf("read recovery manifest: %w", err)
	}
	if got := sha256Hex(recoveryBytes); got != recoveryManifestSHA256 {
		return recoveryManifest{}, fmt.Errorf("recovery manifest hash mismatch: %s", got)
	}
	var m recoveryManifest
	if err := json.Unmarshal(recoveryBytes, &m); err != nil {
		return recoveryManifest{}, fmt.Errorf("decode recovery manifest: %w", err)
	}
	if err := validateRecoveryManifest(m); err != nil {
		return recoveryManifest{}, err
	}
	return m, nil
}

func validateRecoveryManifest(m recoveryManifest) error {
	if m.ContractID != "jax.val-03r2.ma-crossover-recovery-preregistration" || m.ContractVersion != "v1" || m.Status != "FROZEN_FOR_EXTERNAL_REVIEW" {
		return errors.New("recovery identity/status mismatch")
	}
	if m.Parent.CandidateID != "ma_crossover_v1" || m.Parent.Manifest != parentManifestPath || m.Parent.Version != "v1.5" || m.Parent.ManifestSHA256 != parentManifestSHA256 {
		return errors.New("parent binding mismatch")
	}
	c := m.Candidate
	if c.CandidateID != "ma_crossover_v1" || c.StrategyVersion != "ma_crossover_v1" || c.PrimaryClaim != "BULLISH_LONG_ONLY" || c.ActionableConfidenceThreshold != 0.60 || c.SMAFast != 20 || c.SMAMedium != 50 || c.SMASlow != 200 || c.ATRPeriod != 14 || c.AvgVolumePeriod != 20 || c.StopATR != 1 || c.TargetATR != 3 || c.HoldingPeriodSessions != 20 || c.ReferenceCapitalUSD != 10000 || c.QuantityConvention != "WHOLE_SHARES" || c.ParameterChangePermitted {
		return errors.New("candidate contract mismatch")
	}
	if !equalStrings(m.InheritedPolicies, expectedInheritedPolicies) {
		return errors.New("inherited policy mismatch")
	}
	b := m.RecoveryBoundary
	if b.Start != recoveryStart || b.End != recoveryEnd || b.ScoredSessions != "valid US regular sessions only" || b.ExcludedIncompleteSession != "2026-09-14" || b.PostResultExtensionAllowed || !b.ExtensionRequiresNewID || !recoveryDateAllowed(b.Start) || !recoveryDateAllowed(b.End) || recoveryDateAllowed("2026-09-12") {
		return errors.New("recovery boundary mismatch")
	}
	s := m.StructuralFeasibility
	if s.UniverseInstruments != 9 || s.RecoveryYears != 2 || s.Max2025Blocks != 9 || s.MaxRecoveryBlocks != 18 || s.EffectiveBlockFloor != 12 || s.OutcomeCountsInspected {
		return errors.New("structural feasibility mismatch")
	}
	w := m.Warmup
	if w.Source != "accepted VAL-03B/VAL-03C hash-bound pre-recovery evidence" || w.Mode != "minimum preceding valid sessions required by frozen indicator contract" || w.PreRecoveryState != "WARMUP_ONLY" || w.PreRecoveryPerformanceIncluded || w.ContaminatedResultReused {
		return errors.New("warm-up contract mismatch")
	}
	p := m.Provider
	if p.Provider != "Alpaca" || p.Feed != "SIP" || p.Timeframe != "1Day" || !equalStrings(p.Universe, expectedUniverse) || !equalStrings(p.AdjustmentFamilies, []string{"RAW", "SPLIT", "SPLIT+SPIN-OFF"}) || p.Route != "existing hardened Alpaca historical route" || p.Fallback != "NONE" || p.PaidSpendUSD != 0 || !p.RawBytesBeforeNormalization || p.ProviderCallsAuthorizedInR2 || p.RequestDateGuard != "reject any request or returned date >= 2026-09-12; 2025/2026 acquisition is not authorized in R2" {
		return errors.New("provider contract mismatch")
	}
	f := m.SampleFloors
	if f.Development != 90 || f.Validation != 30 || f.RecoveryOOS != 30 || f.Paired != 30 || f.InstrumentYearBlocks != 12 || f.Instruments != 6 || f.YearRegimeCells != 3 || f.ReadinessFailure != "INSUFFICIENT_EVIDENCE before holdout access" {
		return errors.New("sample/readiness floor mismatch")
	}
	g := m.PromotionGates
	if g.PrimaryMeanNetLongReturn != "> 0" || g.PrimaryBootstrapLowerBound != "> 0" || g.PairedMeanDifference != "> 0" || g.PairedBootstrapLowerBound != "> 0" || g.ConcentrationCeiling != 0.4 || g.TopFiveConcentration != "must PASS" || g.BlockingFalsification != "must PASS" || g.DataQuality != "must PASS" || g.UnknownMaterialStatus != "FAIL_CLOSED" || g.SecondarySignPermutation != "NON_BLOCKING_INFORMATIONAL" {
		return errors.New("promotion gate mismatch")
	}
	n := m.NoPostResultSalvage
	if n.ParameterTuning || n.ThresholdTuning || n.UniverseSubstitution || n.DateExtension || n.InstrumentDroppingOrAdding || n.CostChange || n.PlaceboChange || n.BootstrapChange || n.FalsificationChange || n.RegimeChange || n.TopFivePolicyChange || !n.NewExperimentRequired {
		return errors.New("no-post-result-salvage contract mismatch")
	}
	cg := m.ContaminatedRun
	if cg.Artifact != integrityPath || cg.FormalOOSRunCount != 1 || !cg.NoRerun || cg.Classification != "CONTAMINATED_FOR_FORMAL_OOS" || cg.OldRunnerExecution != "BLOCKED" {
		return errors.New("contaminated-run guard mismatch")
	}
	a := m.OutcomeFreeAudit
	if !a.NoMarketDataAccess || !a.No2025Access || !a.No2026Access || !a.NoSignalCounts || !a.NoEpisodeCounts || !a.NoOutcomeAccess || !a.NoExecution || !a.NoPostResultExtension || a.Command != "go run ./cmd/val03-ma-crossover --recovery-contract-audit" {
		return errors.New("outcome-free audit mismatch")
	}
	safe := m.Safety
	if safe.ExecutionAuthority != "NONE" || safe.CreatesFill || safe.AllowLiveTrading || safe.BrokerExecutionAllowed || safe.ExecutionEnabled || safe.MaximumLeverage != 1 || safe.ForwardPaper != "NOT_STARTED" || safe.Phase13 != "NOT_STARTED" {
		return errors.New("safety contract mismatch")
	}
	return nil
}

func validateContaminatedGuard(m recoveryManifest) error {
	b, err := os.ReadFile(integrityPath)
	if err != nil {
		return fmt.Errorf("read integrity artifact: %w", err)
	}
	var r integrityRecord
	if err := json.Unmarshal(b, &r); err != nil {
		return fmt.Errorf("decode integrity artifact: %w", err)
	}
	if r.FinalTerminalClassification != "CONTAMINATED_FOR_FORMAL_OOS" || !r.NoRerun || r.FormalOOSRunCount != 1 {
		return errors.New("contaminated run is not permanently guarded")
	}
	if m.ContaminatedRun.FormalOOSRunCount != r.FormalOOSRunCount || m.ContaminatedRun.NoRerun != r.NoRerun || m.ContaminatedRun.Classification != r.FinalTerminalClassification {
		return errors.New("recovery manifest does not match integrity artifact")
	}
	return nil
}

func runPreHoldoutReadiness() error {
	m, err := loadAndValidateRecoveryManifest()
	if err != nil {
		return err
	}
	if err := validateContaminatedGuard(m); err != nil {
		return err
	}
	if _, err := os.Stat(recoveryRawRoot); err == nil {
		return errors.New("recovery data already exists before pre-holdout readiness")
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := loadPreRecoveryData()
	if err != nil {
		return err
	}
	dev, err := countEligibleEpisodes(data, developmentStart, developmentEnd, m)
	if err != nil {
		return fmt.Errorf("development readiness: %w", err)
	}
	val, err := countEligibleEpisodes(data, validationStart, validationEnd, m)
	if err != nil {
		return fmt.Errorf("validation readiness: %w", err)
	}
	result := readinessResult{
		ContractID:                  "jax.val-03r3.pre-holdout-readiness/v1",
		DevelopmentEligibleEpisodes: dev,
		ValidationEligibleEpisodes:  val,
		DevelopmentFloor:            m.SampleFloors.Development,
		ValidationFloor:             m.SampleFloors.Validation,
		DevelopmentReadiness:        passFail(dev >= m.SampleFloors.Development),
		ValidationReadiness:         passFail(val >= m.SampleFloors.Validation),
	}
	if dev >= m.SampleFloors.Development && val >= m.SampleFloors.Validation {
		result.RecoveryReadiness = "PASS"
	} else {
		result.RecoveryReadiness = "INSUFFICIENT_EVIDENCE"
	}
	if err := writeJSON(readinessOutputPath, result); err != nil {
		return err
	}
	fmt.Printf("VAL03R3_PRE_HOLDOUT_READINESS development=%d floor=%d status=%s validation=%d floor=%d status=%s recovery=%s returns=false outcomes=false\n", dev, result.DevelopmentFloor, result.DevelopmentReadiness, val, result.ValidationFloor, result.ValidationReadiness, result.RecoveryReadiness)
	if result.RecoveryReadiness != "PASS" {
		return errors.New("RECOVERY READINESS = INSUFFICIENT_EVIDENCE")
	}
	return nil
}

func loadPreRecoveryData() (map[string]*instrumentData, error) {
	b, err := os.ReadFile(barFamilyPath)
	if err != nil {
		return nil, fmt.Errorf("read bar-family artifact: %w", err)
	}
	if got := sha256Hex(b); got != barFamilySHA256 {
		return nil, fmt.Errorf("bar-family artifact hash mismatch: %s", got)
	}
	var input readinessInput
	if err := json.Unmarshal(b, &input); err != nil {
		return nil, err
	}
	if len(input.Families) == 0 {
		return nil, errors.New("bar-family artifact contains no family references")
	}
	data := make(map[string]*instrumentData, len(expectedUniverse))
	for _, symbol := range expectedUniverse {
		data[symbol] = &instrumentData{Raw: map[string]bar{}, Split: map[string]bar{}, Detector: map[string]bar{}, Boundaries: map[string]bool{}}
	}
	for _, family := range input.Families {
		d := data[family.Instrument]
		if d == nil {
			return nil, fmt.Errorf("unexpected instrument %s", family.Instrument)
		}
		if family.Adjustment != "raw" && family.Adjustment != "split" && family.Adjustment != "split_spin_off" {
			continue
		}
		for _, h := range family.RawPayloadSHA256 {
			payloadBytes, err := os.ReadFile(filepath.Join(oldRawRoot, h+".json"))
			if err != nil {
				return nil, fmt.Errorf("read hash-bound pre-recovery payload %s: %w", h, err)
			}
			if sha256Hex(payloadBytes) != h {
				return nil, fmt.Errorf("pre-recovery payload hash mismatch %s", h)
			}
			var payload providerPayload
			if err := json.Unmarshal(payloadBytes, &payload); err != nil {
				return nil, err
			}
			for _, item := range payload.Bars[family.Instrument] {
				if len(item.Timestamp) < 10 {
					return nil, errors.New("provider timestamp too short")
				}
				date := item.Timestamp[:10]
				if date < "2016-01-01" || date > validationEnd {
					continue
				}
				bb := bar{Open: item.Open, High: item.High, Low: item.Low, Close: item.Close, Volume: item.Volume}
				switch family.Adjustment {
				case "raw":
					d.Raw[date] = bb
				case "split":
					d.Split[date] = bb
				case "split_spin_off":
					d.Detector[date] = bb
				}
			}
		}
	}
	for _, symbol := range expectedUniverse {
		d := data[symbol]
		if len(d.Raw) == 0 || len(d.Split) == 0 || len(d.Detector) == 0 {
			return nil, fmt.Errorf("missing pre-recovery bar family for %s", symbol)
		}
		for date := range d.Raw {
			if _, ok := d.Split[date]; !ok {
				return nil, fmt.Errorf("RAW/SPLIT unsynchronized %s %s", symbol, date)
			}
			if _, ok := d.Detector[date]; !ok {
				return nil, fmt.Errorf("SPLIT/SPIN_OFF unsynchronized %s %s", symbol, date)
			}
			d.Dates = append(d.Dates, date)
		}
		sort.Strings(d.Dates)
		previous := 0.0
		for _, date := range d.Dates {
			raw, split, detector := d.Raw[date], d.Split[date], d.Detector[date]
			if raw.Close <= 0 || split.Close <= 0 || split.High < split.Low || raw.High < raw.Low {
				return nil, fmt.Errorf("invalid bar %s %s", symbol, date)
			}
			factor := split.Close / raw.Close
			if previous != 0 && math.Abs(factor-previous) > factorTolerance*math.Max(1, math.Max(factor, previous)) {
				d.Boundaries[date] = true
			}
			if !sameBarScale(split, detector) {
				return nil, fmt.Errorf("unsupported structural action %s %s", symbol, date)
			}
			previous = factor
		}
	}
	return data, nil
}

func countEligibleEpisodes(data map[string]*instrumentData, start, end string, m recoveryManifest) (int, error) {
	total := 0
	for _, symbol := range expectedUniverse {
		d := data[symbol]
		activeUntil := -1
		lastBoundary := -1000
		for i, date := range d.Dates {
			if d.Boundaries[date] {
				lastBoundary = i
			}
			if date < start || date > end || i < m.Candidate.SMASlow-1 || i-lastBoundary < m.Candidate.SMASlow {
				continue
			}
			sig, kind := buildSignal(d, i, m)
			if kind != "BUY" || sig.Confidence < m.Candidate.ActionableConfidenceThreshold || i <= activeUntil {
				continue
			}
			entryIndex := i + 1
			if entryIndex >= len(d.Dates) || d.Dates[entryIndex] > end {
				continue
			}
			next := d.Split[d.Dates[entryIndex]]
			if !(sig.Stop < next.Open && next.Open < sig.Target) {
				continue
			}
			exitIndex, ok := readinessExitIndex(d, sig, entryIndex, end, m.Candidate.HoldingPeriodSessions)
			if !ok {
				continue
			}
			rawEntry := d.Raw[d.Dates[entryIndex]].Open
			if rawEntry <= 0 || int(math.Floor(float64(m.Candidate.ReferenceCapitalUSD)/rawEntry)) < 1 {
				continue
			}
			activeUntil = exitIndex
			total++
		}
	}
	return total, nil
}

func buildSignal(d *instrumentData, i int, m recoveryManifest) (signal, string) {
	sf := avgClose(d, i, m.Candidate.SMAFast)
	sm := avgClose(d, i, m.Candidate.SMAMedium)
	ss := avgClose(d, i, m.Candidate.SMASlow)
	price := d.Split[d.Dates[i]].Close
	atrValue := atr(d, i, m.Candidate.ATRPeriod)
	avgVol := avgVolume(d, i, m.Candidate.AvgVolumePeriod)
	s := signal{Confidence: 0.65}
	if sf > sm && sm > ss && price > sf {
		s.Confidence += 0.12
		if d.Split[d.Dates[i]].Volume > avgVol {
			s.Confidence += 0.08
		}
		if (sf-ss)/ss > 0.05 {
			s.Confidence += 0.10
		}
		s.Confidence = math.Min(s.Confidence, 1)
		s.Stop = sm - atrValue*m.Candidate.StopATR
		s.Target = price + m.Candidate.TargetATR*atrValue
		return s, "BUY"
	}
	if sf < sm && sm < ss && price < sf {
		return s, "SELL"
	}
	return s, "HOLD"
}

func readinessExitIndex(d *instrumentData, sig signal, entryIndex int, partitionEnd string, holding int) (int, bool) {
	for offset := 0; offset < holding; offset++ {
		idx := entryIndex + offset
		if idx >= len(d.Dates) || d.Dates[idx] > partitionEnd {
			return 0, false
		}
		if d.Boundaries[d.Dates[idx]] {
			return 0, false
		}
		b := d.Split[d.Dates[idx]]
		if offset > 0 && (b.Open <= sig.Stop || b.Open >= sig.Target) {
			return idx, true
		}
		if b.Low <= sig.Stop || b.High >= sig.Target || offset == holding-1 {
			return idx, true
		}
	}
	return 0, false
}

func avgClose(d *instrumentData, i, n int) float64 {
	total := 0.0
	for j := i - n + 1; j <= i; j++ {
		total += d.Split[d.Dates[j]].Close
	}
	return total / float64(n)
}

func avgVolume(d *instrumentData, i, n int) float64 {
	total := 0.0
	for j := i - n + 1; j <= i; j++ {
		total += d.Split[d.Dates[j]].Volume
	}
	return total / float64(n)
}

func atr(d *instrumentData, i, n int) float64 {
	total := 0.0
	for j := i - n + 1; j <= i; j++ {
		b := d.Split[d.Dates[j]]
		prev := d.Split[d.Dates[j-1]].Close
		total += math.Max(b.High-b.Low, math.Max(math.Abs(b.High-prev), math.Abs(b.Low-prev)))
	}
	return total / float64(n)
}

func sameBarScale(a, b bar) bool {
	return closeEnough(a.Open, b.Open) && closeEnough(a.High, b.High) && closeEnough(a.Low, b.Low) && closeEnough(a.Close, b.Close) && closeEnough(a.Volume, b.Volume)
}

func closeEnough(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func recoveryDateAllowed(date string) bool {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	start, _ := time.Parse("2006-01-02", recoveryStart)
	end, _ := time.Parse("2006-01-02", recoveryEnd)
	return !t.Before(start) && !t.After(end)
}

func gitBlobSHA1(path string) (string, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	h := sha1.New()
	_, _ = fmt.Fprintf(h, "blob %d\x00", len(b))
	_, _ = h.Write(b)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

func equalStrings(a, b []string) bool {
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
