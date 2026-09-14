// Command val03r-recovery-oos is the performance-locked VAL-03R3 recovery
// runner. R3 may verify the future execution contract, but cannot execute it.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	parentManifestPath = "Docs/validation/manifests/VAL-02-ma_crossover_v1-PREREGISTRATION.json"
	parentManifestSHA  = "96a0a21ae6b4d25047da0b90e34bdc839b241e5bc4b2da5a37f89135bea9360a"
	r2ManifestPath     = "Docs/validation/manifests/VAL-03R2-ma_crossover_v1-RECOVERY-PREREGISTRATION.json"
	r2ManifestSHA      = "c19a4cfc774a7bf53466d9bcf5705bd3813f69258a9600291883db48aab490eb"
	conformancePath    = "Docs/validation/results/VAL-03R3-RECOVERY-CONTRACT-CONFORMANCE.json"
	conformanceSHA     = "d1799329cadf925fbf78f213ab15c546570c93f6942253664cf290a4a8ca3f60"
	readinessPath      = "Docs/validation/results/VAL-03R3-PRE-HOLDOUT-READINESS.json"
	readinessSHA       = "10ddf19ef8a2b6612ba65efe9c0685c1b46a015cfc41091a442a10117baf0f28"
	dataContractPath   = "Docs/validation/results/VAL-03R3-RECOVERY-DATA-CONTRACT.json"
	dataContractSHA    = "9fb35ef0be7e1378a72f04ba5c10d0cf838d94ef57115772469308ab28f14cf0"
	datasetPath        = "Docs/validation/results/VAL-03R3-RECOVERY-DATASET-READINESS.json"
	datasetSHA         = "f4bb7c1f05b009bd4d2d284533137c1151693fd9cc59e76871581c144fa8f73f"
	recoveryStart      = "2025-01-01"
	recoveryEnd        = "2026-09-11"
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
		Name         string   `json:"name"`
		Feed         string   `json:"feed"`
		Timeframe    string   `json:"timeframe"`
		AsOf         string   `json:"asof"`
		Families     []string `json:"adjustment_families"`
		Universe     []string `json:"universe"`
		Fallback     string   `json:"fallback"`
		PaidSpendUSD int      `json:"paid_spend_usd"`
		RawRoot      string   `json:"raw_persistence"`
		Request      struct {
			Start                  string   `json:"start"`
			End                    string   `json:"end"`
			ReturnedDates          []string `json:"returned_session_dates_inclusive"`
			RejectBefore           bool     `json:"reject_before_start"`
			RejectAfter            bool     `json:"reject_after_end"`
			RejectExplicit20260914 bool     `json:"reject_explicit_2026_09_14"`
		} `json:"request_identity"`
	} `json:"provider"`
	Execution struct {
		Authority     string  `json:"execution_authority"`
		CreatesFill   bool    `json:"creates_fill"`
		LiveTrading   bool    `json:"allow_live_trading"`
		BrokerAllowed bool    `json:"broker_execution_allowed"`
		Enabled       bool    `json:"execution_enabled"`
		Leverage      float64 `json:"maximum_leverage"`
	} `json:"execution"`
	Lifecycle struct {
		PerformanceExecuted bool `json:"recovery_performance_executed"`
		RunCount            int  `json:"recovery_oos_run_count"`
		PostBoundary        bool `json:"post_boundary_extension"`
	} `json:"lifecycle"`
}

type datasetArtifact struct {
	Status               string `json:"status"`
	AsOf                 string `json:"as_of"`
	Start                string `json:"date_range_start"`
	End                  string `json:"date_range_end"`
	Provider             string `json:"provider"`
	Feed                 string `json:"feed"`
	Timeframe            string `json:"timeframe"`
	SynchronizedSessions int    `json:"synchronized_session_count"`
	CompleteNo2025Guard  bool   `json:"complete_no_2025_guard"`
	PerformanceOutput    bool   `json:"performance_output_generated"`
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "--contract-audit" && os.Args[1] != "--preflight-only") {
		fatal(errors.New("performance execution is locked; allowed modes: --contract-audit|--preflight-only"))
	}
	if err := validateContract(); err != nil {
		fatal(err)
	}
	fmt.Printf("VAL03R3_RECOVERY_RUNNER=%s performance_execution_started=false performance_artifacts_written=false performance_run_count=0\n", os.Args[1][2:])
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "VAL03R3_RECOVERY_RUNNER=FAILED: %v\n", err)
	os.Exit(1)
}

func validateContract() error {
	for path, expected := range map[string]string{
		parentManifestPath: parentManifestSHA,
		r2ManifestPath:     r2ManifestSHA,
		conformancePath:    conformanceSHA,
		readinessPath:      readinessSHA,
		dataContractPath:   dataContractSHA,
		datasetPath:        datasetSHA,
	} {
		if err := verifySHA(path, expected); err != nil {
			return err
		}
	}
	var readiness readinessArtifact
	if err := decodeFile(readinessPath, &readiness); err != nil {
		return fmt.Errorf("readiness artifact: %w", err)
	}
	if readiness.DevelopmentEpisodes < readiness.DevelopmentFloor || readiness.ValidationEpisodes < readiness.ValidationFloor || readiness.DevelopmentStatus != "PASS" || readiness.ValidationStatus != "PASS" || readiness.RecoveryStatus != "PASS" {
		return errors.New("pre-holdout readiness gate is not PASS")
	}
	var contract dataContractArtifact
	if err := decodeFile(dataContractPath, &contract); err != nil {
		return fmt.Errorf("data contract: %w", err)
	}
	if contract.Status != "FROZEN_FOR_DATA_QUALITY_ACQUISITION" || contract.Candidate != "ma_crossover_v1" || contract.PerformanceAuthorized || contract.SignalsAuthorized {
		return errors.New("recovery data contract is not outcome-free")
	}
	if contract.Provider.Name != "Alpaca" || contract.Provider.Feed != "SIP" || contract.Provider.Timeframe != "1Day" || contract.Provider.AsOf != recoveryEnd || contract.Provider.Fallback != "NONE" || contract.Provider.PaidSpendUSD != 0 || contract.Provider.RawRoot != ".runtime/val03r3/raw" || !sameStrings(contract.Provider.Families, expectedAdjustmentFamilies) || !sameStrings(contract.Provider.Universe, expectedRecoveryUniverse) || contract.Provider.Request.Start != recoveryStart || contract.Provider.Request.End != recoveryEnd || !contract.Provider.Request.RejectBefore || !contract.Provider.Request.RejectAfter || !contract.Provider.Request.RejectExplicit20260914 {
		return errors.New("recovery provider or request identity mismatch")
	}
	if contract.Execution.Authority != "NONE" || contract.Execution.CreatesFill || contract.Execution.LiveTrading || contract.Execution.BrokerAllowed || contract.Execution.Enabled || contract.Execution.Leverage != 1 || contract.Lifecycle.PerformanceExecuted || contract.Lifecycle.RunCount != 0 || contract.Lifecycle.PostBoundary {
		return errors.New("recovery safety or lifecycle mismatch")
	}
	var dataset datasetArtifact
	if err := decodeFile(datasetPath, &dataset); err != nil {
		return fmt.Errorf("dataset readiness artifact: %w", err)
	}
	if dataset.Status != "PASS" || dataset.Provider != "Alpaca" || dataset.Feed != "SIP" || dataset.Timeframe != "1Day" || dataset.AsOf != recoveryEnd || dataset.SynchronizedSessions <= 0 || !dataset.CompleteNo2025Guard || dataset.PerformanceOutput {
		return errors.New("recovery dataset readiness is not a structural PASS")
	}
	return nil
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func verifySHA(path, expected string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	sum := sha256.Sum256(b)
	got := hex.EncodeToString(sum[:])
	if got != expected {
		return fmt.Errorf("%s SHA-256 mismatch: got %s want %s", path, got, expected)
	}
	return nil
}

func decodeFile(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}
