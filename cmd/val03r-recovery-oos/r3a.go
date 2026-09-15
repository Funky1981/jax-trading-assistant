package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

const (
	r3aCorrectedDatasetPath = "Docs/validation/results/VAL-03R3A-RECOVERY-DATASET-READINESS.json"
	r3aCorrectedDatasetSHA  = "dd4c49313016bcb45912142620bf8dcd965ec80eaf8b5be7376b3f36d98651d0"
	r3aFreezePath           = "Docs/validation/results/VAL-03R3A-RECOVERY-OOS-EXECUTION-FREEZE.json"
	r3bFreezePath           = "Docs/validation/results/VAL-03R3B-RECOVERY-OOS-EXECUTION-FREEZE.json"
	r3aAuthorizationPath    = "Docs/validation/results/VAL-03R4-EXECUTION-AUTHORIZATION.json"
	r3aRunStatePath         = "Docs/validation/results/VAL-03R4-RUN-STATE.json"
	r3aRecoveryDatasetPath  = "Docs/validation/results/VAL-03R3-RECOVERY-DATASET-READINESS.json"
	r3aRecoveryDatasetSHA   = "f4bb7c1f05b009bd4d2d284533137c1151693fd9cc59e76871581c144fa8f73f"
	r3aRecoveryRawRoot      = ".runtime/val03r3/raw"
	r3aPreRawRoot           = ".runtime/val03b/raw"
	r3aRecoveryStart        = "2025-01-01"
	r3aRecoveryEnd          = "2026-09-11"
	r3aPreStart             = "2016-01-01"
	r3aPreEnd               = "2024-12-31"
	r3aTolerance            = 5e-4
	r3aR2ManifestSHA        = "c19a4cfc774a7bf53466d9bcf5705bd3813f69258a9600291883db48aab490eb"
	r3aBarFamilySHA         = "78c1fbeec2122eebba1a8dbe21c2edfc3c569377770561afe9e128cd633037e5"
	r3aReadinessPath        = "Docs/validation/results/VAL-03R3-PRE-HOLDOUT-READINESS.json"
	r3aReadinessSHA         = "10ddf19ef8a2b6612ba65efe9c0685c1b46a015cfc41091a442a10117baf0f28"
	r3aDataContractPath     = "Docs/validation/results/VAL-03R3-RECOVERY-DATA-CONTRACT.json"
	r3aDataContractSHA      = "9fb35ef0be7e1378a72f04ba5c10d0cf838d94ef57115772469308ab28f14cf0"
)

var expectedRecoveryUniverse = []string{"SPY", "QQQ", "IWM", "DIA", "XLK", "XLF", "XLE", "TLT", "GLD"}
var expectedAdjustmentFamilies = []string{"RAW", "SPLIT", "SPLIT+SPIN-OFF"}

type readinessArtifact struct {
	DevelopmentEpisodes int    `json:"development_eligible_primary_episodes"`
	ValidationEpisodes  int    `json:"validation_eligible_primary_episodes"`
	DevelopmentFloor    int    `json:"development_floor"`
	ValidationFloor     int    `json:"validation_floor"`
	DevelopmentStatus   string `json:"development_readiness"`
	ValidationStatus    string `json:"validation_readiness"`
	RecoveryStatus      string `json:"recovery_readiness"`
}

type dataContractArtifact struct {
	Status                string `json:"status"`
	Candidate             string `json:"candidate"`
	PerformanceAuthorized bool   `json:"performance_execution_authorized"`
	SignalsAuthorized     bool   `json:"strategy_signals_authorized"`
	Provider              struct {
		Name      string   `json:"name"`
		Feed      string   `json:"feed"`
		Timeframe string   `json:"timeframe"`
		AsOf      string   `json:"asof"`
		Families  []string `json:"adjustment_families"`
		Universe  []string `json:"universe"`
		Fallback  string   `json:"fallback"`
		PaidSpend int      `json:"paid_spend_usd"`
		RawRoot   string   `json:"raw_persistence"`
		Request   struct {
			Start        string `json:"start"`
			End          string `json:"end"`
			RejectBefore bool   `json:"reject_before_start"`
			RejectAfter  bool   `json:"reject_after_end"`
		} `json:"request_identity"`
	} `json:"provider"`
	Execution struct {
		Authority string  `json:"execution_authority"`
		Creates   bool    `json:"creates_fill"`
		Live      bool    `json:"allow_live_trading"`
		Broker    bool    `json:"broker_execution_allowed"`
		Enabled   bool    `json:"execution_enabled"`
		Leverage  float64 `json:"maximum_leverage"`
	} `json:"execution"`
	Lifecycle struct {
		Executed bool `json:"recovery_performance_executed"`
		Runs     int  `json:"recovery_oos_run_count"`
		Extended bool `json:"post_boundary_extension"`
	} `json:"lifecycle"`
}

type r3aFrozenContract struct {
	Config               FrozenExperimentConfig
	Candidate            string
	PrimaryClaim         string
	ThresholdSource      string
	SignalCutoff         string
	EntryRule            string
	PreEntryRule         string
	StopFormula          string
	TargetFormula        string
	TargetSelection      string
	OneActiveEpisode     bool
	SignalFamily         string
	ExecutionFamily      string
	DetectorFamily       string
	CashCredit           string
	SplitReset           int
	SplitEpisodePolicy   string
	MatchedPlaceboID     string
	TimestampPlaceboID   string
	BootstrapUnit        string
	BootstrapReplicates  int
	BootstrapInterval    string
	BootstrapLowerBound  string
	DevelopmentFloor     int
	ValidationFloor      int
	RecoveryFloor        int
	PairedFloor          int
	BlockFloor           int
	InstrumentFloor      int
	SliceFloor           int
	ConcentrationCeiling float64
	FalsificationTests   []string
}

type r3aDataset struct {
	Status               string   `json:"status"`
	Provider             string   `json:"provider"`
	Feed                 string   `json:"feed"`
	Timeframe            string   `json:"timeframe"`
	AsOf                 string   `json:"asof"`
	Start                string   `json:"recovery_range_start"`
	End                  string   `json:"recovery_range_end"`
	RangeGuardStatus     string   `json:"range_guard_status"`
	OutOfRangeRows       int      `json:"out_of_range_rows"`
	ContainsExpected2025 bool     `json:"contains_expected_2025_data"`
	PostBoundaryRows     int      `json:"post_boundary_rows"`
	Universe             []string `json:"universe"`
	Families             []string `json:"adjustment_families"`
	SynchronizedSessions int      `json:"synchronized_sessions"`
	Quality              struct {
		DatasetQuality string `json:"dataset_quality"`
	} `json:"quality"`
	PerformanceOutput bool `json:"performance_output_generated"`
	SignalsGenerated  bool `json:"signals_generated"`
	EpisodesGenerated bool `json:"episodes_generated"`
	ReturnsGenerated  bool `json:"returns_generated"`
	PNLGenerated      bool `json:"pnl_generated"`
}

type r3aAuthorization struct {
	ContractID            string `json:"contract_id"`
	Candidate             string `json:"candidate"`
	RecoveryBoundary      string `json:"recovery_boundary"`
	ExecutionFreezeSHA256 string `json:"execution_freeze_sha256"`
	PerformanceRunCount   int    `json:"performance_run_count_before"`
	ExecuteOnce           bool   `json:"execute_once"`
	ExternalAuthorization string `json:"external_authorization"`
}

type r3aRunState struct {
	ContractID            string `json:"contract_id"`
	PerformanceRunCount   int    `json:"performance_run_count"`
	Status                string `json:"status"`
	Candidate             string `json:"candidate"`
	RecoveryBoundary      string `json:"recovery_boundary"`
	ExecutionFreezeSHA256 string `json:"execution_freeze_sha256"`
}

type r3aResultEnvelope struct {
	RunOutput       runOutput      `json:"run_output"`
	Lifecycle       map[string]any `json:"lifecycle"`
	RunnerIdentity  string         `json:"runner_identity"`
	ManifestSHA256  string         `json:"manifest_sha256"`
	DatasetSHA256   string         `json:"dataset_sha256"`
	ExecutionFreeze string         `json:"execution_freeze_identity"`
}

type r3aFreezeArtifact struct {
	ContractID  string `json:"contract_id"`
	Status      string `json:"status"`
	CandidateID string `json:"candidate_id"`
	R3ADataset  struct {
		SHA256            string `json:"sha256"`
		Status            string `json:"status"`
		PerformanceOutput bool   `json:"performance_output_generated"`
		SignalsGenerated  bool   `json:"signals_generated"`
		EpisodesGenerated bool   `json:"episodes_generated"`
		ReturnsGenerated  bool   `json:"returns_generated"`
		PNLGenerated      bool   `json:"pnl_generated"`
	} `json:"r3a_dataset_readiness"`
	Runner struct {
		Path                      string            `json:"path"`
		SourceBlobs               map[string]string `json:"source_blobs_sha1"`
		AllowedModes              []string          `json:"allowed_modes"`
		ArbitraryThreshold        bool              `json:"arbitrary_threshold_override"`
		PerformanceLockBeforeAuth bool              `json:"performance_lock_before_authorization"`
		ContaminatedRunner        bool              `json:"contaminated_runner_reenabled"`
	} `json:"runner"`
	Authorization struct {
		PresentDuringR3A bool `json:"present_during_r3a"`
	} `json:"authorization"`
	Lifecycle struct {
		Started bool   `json:"performance_execution_started"`
		Written bool   `json:"performance_artifacts_written"`
		Runs    int    `json:"performance_run_count"`
		Status  string `json:"recovery_oos_status"`
	} `json:"lifecycle"`
}

func runR3A(mode string) error {
	contract, err := r3aLoadAndValidateContracts()
	if err != nil {
		return err
	}
	if mode == "--execute" {
		return r3aExecute(contract)
	}
	if mode == "--preflight-only" {
		if err := r3aValidateRawPayloadHashes(); err != nil {
			return err
		}
	}
	fmt.Printf("VAL03R3A=%s candidate=%s threshold=%.2f recovery=%s..%s performance_execution_started=false performance_run_count=0\n", mode[2:], contract.Candidate, contract.Config.ActionableConfidenceThreshold, r3aRecoveryStart, r3aRecoveryEnd)
	return nil
}

func r3aLoadAndValidateContracts() (r3aFrozenContract, error) {
	parentBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return r3aFrozenContract{}, fmt.Errorf("read parent manifest: %w", err)
	}
	parentSHA := sha256Hex(parentBytes)
	if parentSHA != parentManifestSHA {
		return r3aFrozenContract{}, errors.New("parent manifest hash mismatch")
	}
	cfg, err := loadFrozenExperimentConfig(parentBytes, parentSHA)
	if err != nil {
		return r3aFrozenContract{}, fmt.Errorf("typed parent contract: %w", err)
	}
	var root map[string]any
	if err := json.Unmarshal(parentBytes, &root); err != nil {
		return r3aFrozenContract{}, err
	}
	getS := func(path ...string) (string, error) { return stringAt(root, path...) }
	getI := func(path ...string) (int, error) { return intAt(root, path...) }
	getN := func(path ...string) (float64, error) { return numberAt(root, path...) }
	contract := r3aFrozenContract{Config: cfg}
	if contract.Candidate, err = getS("selected_candidate", "candidate_id"); err != nil || contract.Candidate != "ma_crossover_v1" {
		return contract, errors.New("candidate contract mismatch")
	}
	if contract.PrimaryClaim, err = getS("selected_candidate", "execution_semantics"); err != nil {
		return contract, errors.New("missing execution semantics")
	}
	if contract.ThresholdSource, err = getS("confidence_contract", "formula"); err != nil {
		return contract, errors.New("missing threshold formula")
	}
	if contract.SignalCutoff, err = getS("protocol", "signal_cutoff"); err != nil || contract.SignalCutoff != "regular-session close" {
		return contract, errors.New("signal cutoff mismatch")
	}
	if contract.EntryRule, err = getS("protocol", "entry"); err != nil || contract.EntryRule != "next regular US equity session open strictly after signal close" {
		return contract, errors.New("entry rule mismatch")
	}
	if contract.PreEntryRule, err = getS("protocol", "pre_entry_gap_rule"); err != nil || contract.PreEntryRule != "long valid only when stop < next_open < target; otherwise PRE_ENTRY_INVALIDATED, abstention and no episode" {
		return contract, errors.New("pre-entry rule mismatch")
	}
	if contract.StopFormula, err = getS("exit_contract", "bullish_stop"); err != nil || contract.StopFormula != "signal_SMA50 - signal_ATR14" {
		return contract, errors.New("stop formula mismatch")
	}
	if contract.TargetFormula, err = getS("exit_contract", "bullish_target"); err != nil || contract.TargetFormula != "signal_close + 3 * signal_ATR14" {
		return contract, errors.New("target formula mismatch")
	}
	if contract.TargetSelection, err = getS("exit_contract", "target_selection"); err != nil || contract.TargetSelection != "first TakeProfit only; second source target ignored because scheduler persists TakeProfit[0]" {
		return contract, errors.New("target selection mismatch")
	}
	if v, e := valueAt(root, "protocol", "overlap_baseline"); e != nil || v.(string) != "one non-overlapping episode per instrument; later signals during an active episode are suppressed/no-trade observations" {
		return contract, errors.New("overlap contract mismatch")
	} else {
		contract.OneActiveEpisode = true
	}
	if contract.SignalFamily, err = getS("price_and_corporate_action_contract", "signal_bar_adjustment"); err != nil || contract.SignalFamily != "SPLIT" {
		return contract, errors.New("signal family mismatch")
	}
	if contract.ExecutionFamily, err = getS("price_and_corporate_action_contract", "execution_reference_adjustment"); err != nil || contract.ExecutionFamily != "RAW" {
		return contract, errors.New("execution family mismatch")
	}
	if contract.DetectorFamily, err = getS("price_and_corporate_action_contract", "structural_detector_adjustment"); err != nil || contract.DetectorFamily != "SPLIT_SPIN_OFF" {
		return contract, errors.New("detector family mismatch")
	}
	if contract.CashCredit, err = getS("price_and_corporate_action_contract", "cash_distribution_primary_credit"); err != nil || contract.CashCredit != "NONE" {
		return contract, errors.New("cash distribution contract mismatch")
	}
	if contract.SplitReset, err = getI("price_and_corporate_action_contract", "post_split_reset_sessions"); err != nil || contract.SplitReset != 200 {
		return contract, errors.New("split reset mismatch")
	}
	if contract.SplitEpisodePolicy, err = getS("price_and_corporate_action_contract", "split_spanning_episode"); err != nil || contract.SplitEpisodePolicy != "STRUCTURAL_ACTION_INVALIDATED" {
		return contract, errors.New("split episode policy mismatch")
	}
	if contract.MatchedPlaceboID, err = getS("placebo_contract", "matched_non_signal", "version"); err != nil || contract.MatchedPlaceboID != "jax.val-02c.matched-non-signal-placebo/v1" {
		return contract, errors.New("matched placebo identity mismatch")
	}
	if contract.TimestampPlaceboID, err = getS("placebo_contract", "timestamp_placebo", "version"); err != nil || contract.TimestampPlaceboID != "jax.val-02c.timestamp-placebo/v1" {
		return contract, errors.New("timestamp placebo identity mismatch")
	}
	if contract.BootstrapUnit, err = getS("bootstrap", "resampling_unit"); err != nil || contract.BootstrapUnit != "non-empty instrument-year block" {
		return contract, errors.New("bootstrap unit mismatch")
	}
	if contract.BootstrapReplicates, err = getI("bootstrap", "replicates"); err != nil || contract.BootstrapReplicates != 10000 {
		return contract, errors.New("bootstrap replicate mismatch")
	}
	if contract.BootstrapInterval, err = getS("bootstrap", "interval"); err != nil || contract.BootstrapInterval != "percentile 95 percent interval; lower bound is the 2.5th percentile" {
		return contract, errors.New("bootstrap interval mismatch")
	}
	contract.BootstrapLowerBound = "2.5th percentile"
	contract.DevelopmentFloor, _ = getI("sample_and_dependence", "episode_floors", "development")
	contract.ValidationFloor, _ = getI("sample_and_dependence", "episode_floors", "validation")
	contract.RecoveryFloor, _ = getI("sample_and_dependence", "episode_floors", "formal_oos")
	contract.PairedFloor, _ = getI("sample_and_dependence", "minimum_paired_placebos")
	contract.BlockFloor, _ = getI("sample_and_dependence", "effective_block_floor", "formal_oos_nonempty_instrument_year_blocks")
	contract.InstrumentFloor, _ = getI("sample_and_dependence", "effective_block_floor", "formal_oos_instruments")
	contract.SliceFloor, _ = getI("sample_and_dependence", "minimum_regime_or_calendar_slices")
	contract.ConcentrationCeiling, _ = getN("sample_and_dependence", "instrument_contribution_max")
	if contract.DevelopmentFloor != 90 || contract.ValidationFloor != 30 || contract.RecoveryFloor != 30 || contract.PairedFloor != 30 || contract.BlockFloor != 12 || contract.InstrumentFloor != 6 || contract.SliceFloor != 3 || contract.ConcentrationCeiling != .4 {
		return contract, errors.New("sample or concentration contract mismatch")
	}
	contract.FalsificationTests, err = stringsAt(root, "falsification_suite", "tests")
	if err != nil || len(contract.FalsificationTests) != 14 {
		return contract, errors.New("falsification suite incomplete")
	}
	if err := r3aValidateR2(); err != nil {
		return contract, err
	}
	if err := r3aValidateReadiness(); err != nil {
		return contract, err
	}
	if err := r3aValidateDataContract(); err != nil {
		return contract, err
	}
	if err := r3aValidateCorrectedDataset(); err != nil {
		return contract, err
	}
	if err := r3bValidateFreeze(); err != nil {
		return contract, err
	}
	return contract, nil
}

func r3aValidateFreeze() error {
	b, err := os.ReadFile(r3aFreezePath)
	if err != nil {
		return fmt.Errorf("read R3A execution freeze: %w", err)
	}
	var v r3aFreezeArtifact
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.ContractID != "jax.val-03r3a.recovery-oos-execution-freeze/v1" || v.Status != "FROZEN_FOR_VAL03R4_EXTERNAL_AUTHORIZATION" || v.CandidateID != "ma_crossover_v1" || v.R3ADataset.SHA256 != r3aCorrectedDatasetSHA || v.R3ADataset.Status != "PASS" || v.R3ADataset.PerformanceOutput || v.R3ADataset.SignalsGenerated || v.R3ADataset.EpisodesGenerated || v.R3ADataset.ReturnsGenerated || v.R3ADataset.PNLGenerated || v.Runner.Path != "cmd/val03r-recovery-oos" || !sameStrings(v.Runner.AllowedModes, []string{"--contract-audit", "--preflight-only", "--execute"}) || v.Runner.ArbitraryThreshold || !v.Runner.PerformanceLockBeforeAuth || v.Runner.ContaminatedRunner || v.Authorization.PresentDuringR3A || v.Lifecycle.Started || v.Lifecycle.Written || v.Lifecycle.Runs != 0 || v.Lifecycle.Status != "NOT_STARTED / FROZEN_FOR_FUTURE_AUTHORIZATION" {
		return errors.New("R3A execution freeze contract mismatch")
	}
	for name, expected := range map[string]string{"main.go": "cmd/val03r-recovery-oos/main.go", "config.go": "cmd/val03r-recovery-oos/config.go", "contract.go": "cmd/val03r-recovery-oos/contract.go", "recovery.go": "cmd/val03r-recovery-oos/recovery.go", "r3a.go": "cmd/val03r-recovery-oos/r3a.go"} {
		want, ok := v.Runner.SourceBlobs[name]
		if !ok || want == "" {
			return fmt.Errorf("missing R3A runner identity %s", name)
		}
		got, err := gitBlobSHA1(expected)
		if err != nil || got != want {
			return fmt.Errorf("R3A runner identity mismatch %s", name)
		}
	}
	return nil
}

func r3aValidateR2() error {
	b, err := os.ReadFile(recoveryManifestPath)
	if err != nil {
		return err
	}
	if sha256Hex(b) != r3aR2ManifestSHA {
		return errors.New("R2 manifest hash mismatch")
	}
	var m recoveryManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	return validateRecoveryManifest(m)
}

func r3aValidateReadiness() error {
	b, err := os.ReadFile(r3aReadinessPath)
	if err != nil {
		return err
	}
	if sha256Hex(b) != r3aReadinessSHA {
		return errors.New("R3 readiness hash mismatch")
	}
	var v readinessArtifact
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.DevelopmentEpisodes < v.DevelopmentFloor || v.ValidationEpisodes < v.ValidationFloor || v.DevelopmentStatus != "PASS" || v.ValidationStatus != "PASS" || v.RecoveryStatus != "PASS" {
		return errors.New("R3 readiness is not PASS")
	}
	return nil
}

func r3aValidateDataContract() error {
	b, err := os.ReadFile(r3aDataContractPath)
	if err != nil {
		return err
	}
	if sha256Hex(b) != r3aDataContractSHA {
		return errors.New("R3 data contract hash mismatch")
	}
	var v dataContractArtifact
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.Status != "FROZEN_FOR_DATA_QUALITY_ACQUISITION" || v.Candidate != "ma_crossover_v1" || v.PerformanceAuthorized || v.SignalsAuthorized || v.Provider.Name != "Alpaca" || v.Provider.Feed != "SIP" || v.Provider.Timeframe != "1Day" || v.Provider.AsOf != r3aRecoveryEnd || v.Provider.Fallback != "NONE" || v.Provider.PaidSpend != 0 || v.Provider.RawRoot != r3aRecoveryRawRoot || !sameStrings(v.Provider.Families, expectedAdjustmentFamilies) || !sameStrings(v.Provider.Universe, expectedRecoveryUniverse) || v.Provider.Request.Start != r3aRecoveryStart || v.Provider.Request.End != r3aRecoveryEnd || !v.Provider.Request.RejectBefore || !v.Provider.Request.RejectAfter || v.Execution.Authority != "NONE" || v.Execution.Creates || v.Execution.Live || v.Execution.Broker || v.Execution.Enabled || v.Execution.Leverage != 1 || v.Lifecycle.Executed || v.Lifecycle.Runs != 0 || v.Lifecycle.Extended {
		return errors.New("R3 data contract mismatch")
	}
	return nil
}

func r3aValidateCorrectedDataset() error {
	b, err := os.ReadFile(r3aCorrectedDatasetPath)
	if err != nil {
		return err
	}
	if sha256Hex(b) != r3aCorrectedDatasetSHA {
		return errors.New("R3A corrected dataset hash mismatch")
	}
	var v r3aDataset
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.Status != "PASS" || v.Provider != "Alpaca" || v.Feed != "SIP" || v.Timeframe != "1Day" || v.AsOf != r3aRecoveryEnd || v.Start != r3aRecoveryStart || v.End != r3aRecoveryEnd || v.RangeGuardStatus != "PASS" || v.OutOfRangeRows != 0 || !v.ContainsExpected2025 || v.PostBoundaryRows != 0 || v.SynchronizedSessions != 424 || v.Quality.DatasetQuality != "PASS" || v.PerformanceOutput || v.SignalsGenerated || v.EpisodesGenerated || v.ReturnsGenerated || v.PNLGenerated || !sameStrings(v.Universe, expectedRecoveryUniverse) || !sameStrings(v.Families, expectedAdjustmentFamilies) {
		return errors.New("R3A corrected dataset contract mismatch")
	}
	return nil
}

func r3aValidateRawPayloadHashes() error {
	if err := r3aValidatePayloadManifest(r3aRecoveryDatasetPath, r3aRecoveryDatasetSHA, r3aRecoveryRawRoot); err != nil {
		return err
	}
	return r3aValidatePayloadManifest(barFamilyPath, r3aBarFamilySHA, r3aPreRawRoot)
}

func r3aValidatePayloadManifest(metadataPath, expectedMetadataSHA, root string) error {
	b, err := os.ReadFile(metadataPath)
	if err != nil {
		return err
	}
	if sha256Hex(b) != expectedMetadataSHA {
		return fmt.Errorf("dataset metadata hash mismatch %s", metadataPath)
	}
	var v struct {
		Families []struct {
			RawPayloadSHA256 []string `json:"raw_payload_sha256"`
		} `json:"families"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range v.Families {
		for _, h := range f.RawPayloadSHA256 {
			if h == "" {
				return fmt.Errorf("empty payload hash in %s", metadataPath)
			}
			seen[h] = true
		}
	}
	for h := range seen {
		path := filepath.Join(root, h+".json")
		payload, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("payload %s: %w", h, err)
		}
		if sha256Hex(payload) != h {
			return fmt.Errorf("payload hash mismatch %s", h)
		}
	}
	return nil
}

func r3aValidateAuthorizationBytes(b []byte, freezeSHA string) (r3aAuthorization, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return r3aAuthorization{}, err
	}
	allowed := map[string]bool{"contract_id": true, "candidate": true, "recovery_boundary": true, "execution_freeze_sha256": true, "performance_run_count_before": true, "execute_once": true, "external_authorization": true}
	for k := range raw {
		if !allowed[k] {
			return r3aAuthorization{}, fmt.Errorf("unknown authorization field %s", k)
		}
	}
	var a r3aAuthorization
	if err := json.Unmarshal(b, &a); err != nil {
		return a, err
	}
	if a.ContractID != "jax.val-03r4.execution-authorization/v1" || a.Candidate != "ma_crossover_v1" || a.RecoveryBoundary != "2025-01-01..2026-09-11" || a.ExecutionFreezeSHA256 != freezeSHA || a.PerformanceRunCount != 0 || !a.ExecuteOnce || a.ExternalAuthorization != "GO_VAL03R4_SINGLE_RECOVERY_OOS_EXECUTION" {
		return a, errors.New("R4 authorization contract mismatch")
	}
	return a, nil
}

func r3aValidateRunStateBytes(b []byte) (r3aRunState, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return r3aRunState{}, err
	}
	allowed := map[string]bool{"contract_id": true, "performance_run_count": true, "status": true, "candidate": true, "recovery_boundary": true, "execution_freeze_sha256": true}
	if len(raw) != len(allowed) {
		return r3aRunState{}, errors.New("incomplete recovery run state")
	}
	for key := range raw {
		if !allowed[key] || raw[key] == nil {
			return r3aRunState{}, errors.New("invalid recovery run state fields")
		}
	}
	var state r3aRunState
	if err := json.Unmarshal(b, &state); err != nil {
		return state, err
	}
	if state.ContractID != "jax.val-03r4.recovery-run-state/v2" || state.Candidate != "ma_crossover_v1" || state.RecoveryBoundary != "2025-01-01..2026-09-11" || state.ExecutionFreezeSHA256 == "" {
		return state, errors.New("recovery run state identity mismatch")
	}
	if (state.PerformanceRunCount == 1 && state.Status != "COMPLETED_ONCE") || (state.PerformanceRunCount != 1 && state.Status != "STARTED_ONCE") {
		return state, errors.New("invalid recovery run state")
	}
	return state, errors.New("recovery run already consumed")
}

func r3aExecute(contract r3aFrozenContract) error {
	authBytes, err := os.ReadFile(r3aAuthorizationPath)
	if err != nil {
		return errors.New("R4 authorization required; performance locked before data load")
	}
	freezeBytes, err := os.ReadFile(r3bFreezePath)
	if err != nil {
		return fmt.Errorf("read execution freeze: %w", err)
	}
	freezeSHA := sha256Hex(freezeBytes)
	if _, err := r3aValidateAuthorizationBytes(authBytes, freezeSHA); err != nil {
		return fmt.Errorf("authorization: %w", err)
	}
	if stateBytes, err := os.ReadFile(r3aRunStatePath); err == nil {
		_, _ = r3aValidateRunStateBytes(stateBytes)
		return errors.New("recovery run already consumed")
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, p := range []string{"Docs/validation/results/VAL-03R4-RUN-MANIFEST.json", "Docs/validation/results/VAL-03R4-PRIMARY.json", "Docs/validation/results/VAL-03R4-FALSIFICATION.json"} {
		if _, err := os.Stat(p); err == nil {
			return errors.New("completed recovery artifacts already exist; rerun prohibited")
		}
	}
	if err := r3aValidateRawPayloadHashes(); err != nil {
		return fmt.Errorf("execution-time payload verification: %w", err)
	}
	data, err := r3aLoadCombinedData()
	if err != nil {
		return err
	}
	if err := r3aStartRunState(freezeSHA); err != nil {
		return err
	}
	results, err := r3aEvaluate(data, contract)
	if err != nil {
		return err
	}
	oos := results["recovery_oos"]
	executionConfig := r3aExecutionConfig(contract.Config)
	falsification := buildFalsification(data, oos, parentManifestSHA, executionConfig, oos.SecondaryDiagnostics)
	classification := classifyPromotion(oos, falsificationDispositions(falsification), executionConfig)
	runnerBlobs, err := r3aCurrentRunnerSourceIdentities()
	if err != nil {
		return err
	}
	out := runOutput{ContractVersion: "jax.val-03r4.recovery-oos-results/v1", ManifestSHA256: parentManifestSHA, DatasetReadinessSHA256: r3aCorrectedDatasetSHA, R2RecoveryManifestSHA256: r3aR2ManifestSHA, R3ReadinessSHA256: r3aReadinessSHA, R3DataContractSHA256: r3aDataContractSHA, R3DatasetReadinessSHA256: r3aRecoveryDatasetSHA, R3ADatasetReadinessSHA256: r3aCorrectedDatasetSHA, ExecutionFreezeSHA256: freezeSHA, SeedManifestSHA256: parentManifestSHA, CandidateID: contract.Candidate, RecoveryRangeStart: r3aRecoveryStart, RecoveryRangeEnd: r3aRecoveryEnd, Former2025FinalHoldoutReclassified: true, FinalHistoricalHoldoutRemaining: "NONE_AFTER_RECOVERY_RECLASSIFICATION", PostBoundaryRows: 0, RunnerSourceIdentities: runnerBlobs, Runner: "cmd/val03r-recovery-oos", ExecutionAuthority: "NONE", CreatesFill: false, Provider: "Alpaca", Feed: "SIP", Timeframe: "1Day", Universe: append([]string(nil), expectedRecoveryUniverse...), DataStart: r3aRecoveryStart, DataEnd: r3aRecoveryEnd, HoldoutAccessed: false, PerformanceRunCount: 1, Partitions: results, Falsification: falsification, AbstentionTotals: totalAbstentions(results), DataQuality: qualitySummaryForRange(data, true, r3aRecoveryStart, r3aRecoveryEnd), TerminalClassification: classification}
	envelope := r3aResultEnvelope{RunOutput: out, RunnerIdentity: "cmd/val03r-recovery-oos", ManifestSHA256: parentManifestSHA, DatasetSHA256: r3aCorrectedDatasetSHA, ExecutionFreeze: freezeSHA, Lifecycle: map[string]any{"performance_execution_started": true, "performance_artifacts_written": true, "performance_run_count": 1, "recovery_oos_status": "EXECUTED_ONCE"}}
	if err := writeJSON("Docs/validation/results/VAL-03R4-RUN-MANIFEST.json", envelope); err != nil {
		return err
	}
	if err := writeJSON("Docs/validation/results/VAL-03R4-PRIMARY.json", results["recovery_oos"]); err != nil {
		return err
	}
	if err := writeJSON("Docs/validation/results/VAL-03R4-FALSIFICATION.json", falsification); err != nil {
		return err
	}
	return r3aCompleteRunState(freezeSHA)
}

func r3aStartRunState(freezeSHA string) error {
	return r3aStartRunStateAt(r3aRunStatePath, freezeSHA)
}

func r3aStartRunStateAt(path, freezeSHA string) error {
	state := r3aRunState{ContractID: "jax.val-03r4.recovery-run-state/v2", PerformanceRunCount: 0, Status: "STARTED_ONCE", Candidate: "ma_crossover_v1", RecoveryBoundary: "2025-01-01..2026-09-11", ExecutionFreezeSHA256: freezeSHA}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return errors.New("recovery run already consumed")
		}
		return err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return err
	}
	return f.Sync()
}

func r3aCompleteRunState(freezeSHA string) error {
	return r3aCompleteRunStateAt(r3aRunStatePath, freezeSHA)
}

func r3aCompleteRunStateAt(path, freezeSHA string) error {
	state := r3aRunState{ContractID: "jax.val-03r4.recovery-run-state/v2", PerformanceRunCount: 1, Status: "COMPLETED_ONCE", Candidate: "ma_crossover_v1", RecoveryBoundary: "2025-01-01..2026-09-11", ExecutionFreezeSHA256: freezeSHA}
	return writeAtomicJSON(path, state)
}

func writeAtomicJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func r3aCurrentRunnerSourceIdentities() (map[string]string, error) {
	paths := map[string]string{"main.go": "cmd/val03r-recovery-oos/main.go", "config.go": "cmd/val03r-recovery-oos/config.go", "contract.go": "cmd/val03r-recovery-oos/contract.go", "recovery.go": "cmd/val03r-recovery-oos/recovery.go", "r3a.go": "cmd/val03r-recovery-oos/r3a.go", "structural.go": "cmd/val03r-recovery-oos/structural.go"}
	out := make(map[string]string, len(paths))
	for name, path := range paths {
		h, err := gitBlobSHA1(path)
		if err != nil {
			return nil, err
		}
		out[name] = h
	}
	return out, nil
}

func r3aExecutionConfig(cfg FrozenExperimentConfig) FrozenExperimentConfig {
	cfg.OOSStart, cfg.OOSEnd = r3aRecoveryStart, r3aRecoveryEnd
	return cfg
}

func r3aEvaluate(data map[string]*instrumentData, contract r3aFrozenContract) (map[string]partitionResult, error) {
	// The parent config retains the immutable original OOS dates. The recovery
	// runner supplies its separately frozen recovery boundary only to the future
	// execution calls, so variants and diagnostics cannot silently fall back to
	// 2023-2024.
	executionConfig := r3aExecutionConfig(contract.Config)
	r, err := evaluatePartition(data, "recovery_oos", r3aRecoveryStart, r3aRecoveryEnd, executionConfig)
	if err != nil {
		return nil, err
	}
	if err := attachPlacebos(data, &r, executionConfig, parentManifestSHA); err != nil {
		return nil, err
	}
	applyBootstrap(&r, parentManifestSHA, executionConfig)
	return map[string]partitionResult{"recovery_oos": r}, nil
}

func r3aLoadCombinedData() (map[string]*instrumentData, error) {
	data := map[string]*instrumentData{}
	for _, s := range expectedRecoveryUniverse {
		data[s] = &instrumentData{Raw: map[string]bar{}, Split: map[string]bar{}, Detector: map[string]bar{}, Factor: map[string]float64{}, Boundaries: map[string]bool{}}
	}
	for _, source := range []struct{ path, hash, root string }{{barFamilyPath, r3aBarFamilySHA, r3aPreRawRoot}, {r3aRecoveryDatasetPath, r3aRecoveryDatasetSHA, r3aRecoveryRawRoot}} {
		b, err := os.ReadFile(source.path)
		if err != nil {
			return nil, err
		}
		if sha256Hex(b) != source.hash {
			return nil, fmt.Errorf("dataset metadata hash mismatch %s", source.path)
		}
		var v struct {
			Families []familyRef `json:"families"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		for _, f := range v.Families {
			if _, ok := data[f.Instrument]; !ok {
				continue
			}
			for _, h := range f.RawPayloadSHA256 {
				payload, err := os.ReadFile(filepath.Join(source.root, h+".json"))
				if err != nil {
					return nil, err
				}
				var p providerPayload
				if err := json.Unmarshal(payload, &p); err != nil {
					return nil, err
				}
				for _, item := range p.Bars[f.Instrument] {
					if len(item.Timestamp) < 10 {
						return nil, errors.New("invalid provider timestamp")
					}
					date := item.Timestamp[:10]
					if source.root == r3aRecoveryRawRoot && (date < r3aRecoveryStart || date > r3aRecoveryEnd) {
						return nil, errors.New("recovery date outside boundary")
					}
					if source.root == r3aPreRawRoot && (date < r3aPreStart || date > r3aPreEnd) {
						return nil, errors.New("warmup date outside boundary")
					}
					b := bar{Date: date, Open: item.Open, High: item.High, Low: item.Low, Close: item.Close, Volume: item.Volume}
					switch f.Adjustment {
					case "raw":
						data[f.Instrument].Raw[date] = b
					case "split":
						data[f.Instrument].Split[date] = b
					case "split_spin_off":
						data[f.Instrument].Detector[date] = b
					}
				}
			}
		}
	}
	for _, s := range expectedRecoveryUniverse {
		d := data[s]
		if len(d.Raw) != len(d.Split) || len(d.Split) != len(d.Detector) {
			return nil, fmt.Errorf("synchronized family cardinality mismatch %s", s)
		}
		for date := range d.Raw {
			if _, ok := d.Split[date]; !ok {
				return nil, fmt.Errorf("raw/split mismatch %s %s", s, date)
			}
			if _, ok := d.Detector[date]; !ok {
				return nil, fmt.Errorf("split/spin-off mismatch %s %s", s, date)
			}
			d.Dates = append(d.Dates, date)
		}
		for date := range d.Split {
			if _, ok := d.Raw[date]; !ok {
				return nil, fmt.Errorf("split has no raw session %s %s", s, date)
			}
		}
		for date := range d.Detector {
			if _, ok := d.Split[date]; !ok {
				return nil, fmt.Errorf("detector has no split session %s %s", s, date)
			}
		}
		sort.Strings(d.Dates)
		if len(d.Dates) == 0 {
			return nil, fmt.Errorf("no data for %s", s)
		}
		prev := 0.0
		for _, date := range d.Dates {
			raw, split, det := d.Raw[date], d.Split[date], d.Detector[date]
			if !validBar(raw, date) || !validBar(split, date) || !validBar(det, date) {
				return nil, fmt.Errorf("invalid synchronized bar %s %s", s, date)
			}
			factor := split.Close / raw.Close
			if !barScaleConsistent(raw, split, factor) {
				return nil, fmt.Errorf("raw/split factor inconsistency %s %s", s, date)
			}
			if !sameBarScale(split, det) {
				return nil, fmt.Errorf("split/spin-off structural difference %s %s", s, date)
			}
			d.Factor[date] = factor
			if prev != 0 && r3aAbsFloat(factor-prev) > r3aTolerance*r3aMaxFloat(1, r3aMaxFloat(factor, prev)) {
				d.Boundaries[date] = true
			}
			prev = factor
			if det.Close <= 0 {
				return nil, errors.New("invalid detector price")
			}
		}
	}
	return data, nil
}

func validBar(v bar, date string) bool {
	return v.Date == date && v.Open > 0 && v.High > 0 && v.Low > 0 && v.Close > 0 && v.High >= v.Low && v.High >= v.Open && v.High >= v.Close && v.Low <= v.Open && v.Low <= v.Close && v.Volume >= 0 && !math.IsNaN(v.Open) && !math.IsNaN(v.High) && !math.IsNaN(v.Low) && !math.IsNaN(v.Close) && !math.IsNaN(v.Volume)
}

func barScaleConsistent(raw, split bar, factor float64) bool {
	if factor <= 0 || !closeEnoughWithTolerance(split.Open, raw.Open*factor) || !closeEnoughWithTolerance(split.High, raw.High*factor) || !closeEnoughWithTolerance(split.Low, raw.Low*factor) || !closeEnoughWithTolerance(split.Close, raw.Close*factor) {
		return false
	}
	if raw.Volume > 0 && split.Volume > 0 {
		inverse := raw.Volume / factor
		if math.Abs(split.Volume-inverse) > 0.01*math.Max(1, math.Abs(inverse)) {
			return false
		}
	}
	return true
}

func closeEnoughWithTolerance(a, b float64) bool {
	return math.Abs(a-b) <= r3aTolerance*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func r3aAbsFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
func r3aMaxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
