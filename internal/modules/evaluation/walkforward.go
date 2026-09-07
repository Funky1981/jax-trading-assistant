package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const WalkForwardProtocolContractV1 = "jax.walk_forward_protocol/v1"

type WalkForwardWindow struct {
	WindowNumber          int            `json:"window_number"`
	Split                 BenchmarkSplit `json:"split"`
	BenchmarkID           string         `json:"benchmark_id"`
	CaseIDs               []string       `json:"case_ids"`
	StartAt               time.Time      `json:"start_at"`
	EndAt                 time.Time      `json:"end_at"`
	PurgeDuration         time.Duration  `json:"purge_duration"`
	EmbargoDuration       time.Duration  `json:"embargo_duration"`
	OutcomesUsedForTuning bool           `json:"outcomes_used_for_tuning"`
}

func (window WalkForwardWindow) Validate(benchmarks map[string]FrozenBenchmark) error {
	if window.WindowNumber < 1 || window.StartAt.IsZero() || window.EndAt.IsZero() || window.StartAt.Location() != time.UTC || window.EndAt.Location() != time.UTC || !window.EndAt.After(window.StartAt) || window.PurgeDuration < 0 || window.EmbargoDuration < 0 || !strictSortedUnique(window.CaseIDs) || len(window.CaseIDs) == 0 {
		return fmt.Errorf("walk-forward window requires ordered UTC bounds, cases, purge and embargo")
	}
	benchmark, ok := benchmarks[window.BenchmarkID]
	if !ok {
		return fmt.Errorf("walk-forward window references unknown benchmark %q", window.BenchmarkID)
	}
	if window.Split != benchmark.Split || window.StartAt.Before(benchmark.StartAt) || window.EndAt.After(benchmark.EndAt) {
		return fmt.Errorf("walk-forward window does not match its frozen benchmark")
	}
	knownCases := map[string]struct{}{}
	for _, caseID := range benchmark.CaseIDs {
		knownCases[caseID] = struct{}{}
	}
	for _, caseID := range window.CaseIDs {
		if _, exists := knownCases[caseID]; !exists {
			return fmt.Errorf("walk-forward window references undeclared case %q", caseID)
		}
		decisionAt := benchmark.DecisionTimes[caseID]
		if decisionAt.Before(window.StartAt) || decisionAt.After(window.EndAt) {
			return fmt.Errorf("walk-forward window does not contain decision time for %q", caseID)
		}
	}
	if (window.Split == BenchmarkOutOfSample || window.Split == BenchmarkFinalHoldout) && window.OutcomesUsedForTuning {
		return fmt.Errorf("out-of-sample and final-holdout outcomes cannot tune candidate logic")
	}
	return nil
}

type WalkForwardProtocol struct {
	ContractVersion       string              `json:"contract_version"`
	ID                    string              `json:"id"`
	Version               string              `json:"version"`
	BenchmarkIDs          []string            `json:"benchmark_ids"`
	Windows               []WalkForwardWindow `json:"windows"`
	ConfigurationFrozenAt time.Time           `json:"configuration_frozen_at"`
	CandidateLogicVersion string              `json:"candidate_logic_version"`
	CostPolicyVersion     string              `json:"cost_policy_version"`
	NoLookAheadRule       string              `json:"no_look_ahead_rule"`
	FinalHoldoutUntouched bool                `json:"final_holdout_untouched"`
}

func NewWalkForwardProtocol(benchmarks []FrozenBenchmark, windows []WalkForwardWindow, version, candidateLogicVersion, costPolicyVersion, noLookAheadRule string, configurationFrozenAt time.Time) (WalkForwardProtocol, error) {
	protocol := WalkForwardProtocol{ContractVersion: WalkForwardProtocolContractV1, Version: version, Windows: append([]WalkForwardWindow(nil), windows...), ConfigurationFrozenAt: configurationFrozenAt, CandidateLogicVersion: candidateLogicVersion, CostPolicyVersion: costPolicyVersion, NoLookAheadRule: noLookAheadRule, FinalHoldoutUntouched: true}
	benchmarkMap := make(map[string]FrozenBenchmark, len(benchmarks))
	for _, benchmark := range benchmarks {
		protocol.BenchmarkIDs = append(protocol.BenchmarkIDs, benchmark.ID)
		benchmarkMap[benchmark.ID] = benchmark
	}
	sort.Strings(protocol.BenchmarkIDs)
	protocol.ID = deriveWalkForwardProtocolID(protocol)
	if err := protocol.Validate(benchmarkMap); err != nil {
		return WalkForwardProtocol{}, err
	}
	return protocol, nil
}

func (protocol WalkForwardProtocol) Validate(benchmarks map[string]FrozenBenchmark) error {
	if protocol.ContractVersion != WalkForwardProtocolContractV1 || !validIdentity("walk_", protocol.ID) || strings.TrimSpace(protocol.Version) == "" || strings.TrimSpace(protocol.CandidateLogicVersion) == "" || strings.TrimSpace(protocol.CostPolicyVersion) == "" || strings.TrimSpace(protocol.NoLookAheadRule) == "" || protocol.ConfigurationFrozenAt.IsZero() || protocol.ConfigurationFrozenAt.Location() != time.UTC || !protocol.FinalHoldoutUntouched || len(protocol.BenchmarkIDs) == 0 || !strictSortedUnique(protocol.BenchmarkIDs) || len(protocol.Windows) < 3 {
		return fmt.Errorf("walk-forward protocol requires frozen versions, no-look-ahead rule and development/validation/OOS windows")
	}
	if len(protocol.BenchmarkIDs) != len(benchmarks) {
		return fmt.Errorf("walk-forward protocol benchmark set is incomplete")
	}
	for _, benchmarkID := range protocol.BenchmarkIDs {
		benchmark, ok := benchmarks[benchmarkID]
		if !ok {
			return fmt.Errorf("walk-forward protocol references unknown benchmark %q", benchmarkID)
		}
		if err := benchmark.Validate(); err != nil {
			return err
		}
	}
	hasDevelopment, hasValidation, hasOutOfSample := false, false, false
	var previous WalkForwardWindow
	for index, window := range protocol.Windows {
		if window.WindowNumber != index+1 {
			return fmt.Errorf("walk-forward windows must have contiguous deterministic numbers")
		}
		if err := window.Validate(benchmarks); err != nil {
			return err
		}
		if index > 0 && !window.StartAt.After(previous.EndAt.Add(previous.PurgeDuration+previous.EmbargoDuration)) {
			return fmt.Errorf("walk-forward windows overlap or violate purge/embargo separation")
		}
		switch window.Split {
		case BenchmarkDevelopment:
			hasDevelopment = true
		case BenchmarkValidation:
			hasValidation = true
		case BenchmarkOutOfSample:
			hasOutOfSample = true
			if !protocol.ConfigurationFrozenAt.Before(window.StartAt) {
				return fmt.Errorf("configuration must freeze before out-of-sample evaluation")
			}
		case BenchmarkFinalHoldout:
			if !protocol.ConfigurationFrozenAt.Before(window.StartAt) || window.OutcomesUsedForTuning {
				return fmt.Errorf("final holdout must follow configuration freeze and remain untouched")
			}
		}
		previous = window
	}
	if !hasDevelopment || !hasValidation || !hasOutOfSample || protocol.ID != deriveWalkForwardProtocolID(protocol) {
		return fmt.Errorf("walk-forward protocol lacks required chronological split evidence")
	}
	return nil
}

func deriveWalkForwardProtocolID(protocol WalkForwardProtocol) string {
	protocol.ID = ""
	seed, _ := json.Marshal(protocol)
	digest := sha256.Sum256(seed)
	return "walk_" + hex.EncodeToString(digest[:])
}
