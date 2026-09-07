package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const FrozenBenchmarkContractV1 = "jax.frozen_benchmark/v1"

var ErrBenchmarkImmutable = errors.New("frozen benchmark identity is immutable")

type BenchmarkSplit string

const (
	BenchmarkDevelopment  BenchmarkSplit = "DEVELOPMENT"
	BenchmarkValidation   BenchmarkSplit = "VALIDATION"
	BenchmarkOutOfSample  BenchmarkSplit = "OUT_OF_SAMPLE"
	BenchmarkFinalHoldout BenchmarkSplit = "FINAL_HOLDOUT"
)

type ConfigurationFreeze struct {
	EvidenceSelectionVersion      string             `json:"evidence_selection_version"`
	ResearchContractVersion       string             `json:"research_contract_version"`
	PromptVersion                 string             `json:"prompt_version"`
	SystemVersion                 string             `json:"system_version"`
	Provider                      string             `json:"provider"`
	Model                         string             `json:"model"`
	RecommendationContractVersion string             `json:"recommendation_contract_version"`
	EligibilityRulesVersion       string             `json:"eligibility_rules_version"`
	QuantAlgorithmVersions        []string           `json:"quant_algorithm_versions"`
	Thresholds                    map[string]float64 `json:"thresholds"`
	CostPolicyVersion             string             `json:"cost_policy_version"`
	Universe                      []string           `json:"universe"`
	BenchmarkReference            string             `json:"benchmark_reference"`
}

func (freeze ConfigurationFreeze) Validate() error {
	if strings.TrimSpace(freeze.EvidenceSelectionVersion) == "" || strings.TrimSpace(freeze.ResearchContractVersion) == "" || strings.TrimSpace(freeze.PromptVersion) == "" || strings.TrimSpace(freeze.SystemVersion) == "" || strings.TrimSpace(freeze.Provider) == "" || strings.TrimSpace(freeze.Model) == "" || strings.TrimSpace(freeze.RecommendationContractVersion) == "" || strings.TrimSpace(freeze.EligibilityRulesVersion) == "" || strings.TrimSpace(freeze.CostPolicyVersion) == "" || len(freeze.Universe) == 0 || strings.TrimSpace(freeze.BenchmarkReference) == "" {
		return fmt.Errorf("configuration freeze requires evidence, research, model, recommendation, cost and universe versions")
	}
	if len(freeze.QuantAlgorithmVersions) == 0 {
		return fmt.Errorf("configuration freeze requires quant algorithm versions")
	}
	for key, value := range freeze.Thresholds {
		if strings.TrimSpace(key) == "" || !finite(value) {
			return fmt.Errorf("configuration threshold %q is invalid", key)
		}
	}
	if !strictSortedUnique(freeze.QuantAlgorithmVersions) || !strictSortedUnique(freeze.Universe) {
		return fmt.Errorf("configuration freeze lists must be sorted and unique")
	}
	return nil
}

type FrozenBenchmark struct {
	ContractVersion         string               `json:"contract_version"`
	ID                      string               `json:"id"`
	Version                 string               `json:"version"`
	DatasetContentSHA256    string               `json:"dataset_content_sha256"`
	Split                   BenchmarkSplit       `json:"split"`
	Instruments             []string             `json:"instruments"`
	StartAt                 time.Time            `json:"start_at"`
	EndAt                   time.Time            `json:"end_at"`
	CaseIDs                 []string             `json:"case_ids"`
	DecisionTimes           map[string]time.Time `json:"decision_times"`
	AllowedEvidenceVintages map[string]string    `json:"allowed_evidence_vintages"`
	ExpectedProtocol        string               `json:"expected_protocol"`
	Configuration           ConfigurationFreeze  `json:"configuration"`
	FrozenAt                time.Time            `json:"frozen_at"`
	Provenance              BenchmarkProvenance  `json:"provenance"`
	OutcomesUsedForTuning   bool                 `json:"outcomes_used_for_tuning"`
	HoldoutTouched          bool                 `json:"holdout_touched"`
	VariantsTried           int                  `json:"variants_tried"`
}

type BenchmarkProvenance struct {
	Source      string    `json:"source"`
	ManifestRef string    `json:"manifest_ref"`
	RecordedAt  time.Time `json:"recorded_at"`
}

func NewFrozenBenchmark(benchmark FrozenBenchmark) (FrozenBenchmark, error) {
	benchmark.ContractVersion = FrozenBenchmarkContractV1
	benchmark.ID = deriveFrozenBenchmarkID(benchmark)
	if err := benchmark.Validate(); err != nil {
		return FrozenBenchmark{}, err
	}
	return benchmark, nil
}

func (benchmark FrozenBenchmark) Validate() error {
	if benchmark.ContractVersion != FrozenBenchmarkContractV1 || !validIdentity("bench_", benchmark.ID) || strings.TrimSpace(benchmark.Version) == "" || !validSHA256(benchmark.DatasetContentSHA256) || benchmark.StartAt.IsZero() || benchmark.EndAt.IsZero() || benchmark.StartAt.Location() != time.UTC || benchmark.EndAt.Location() != time.UTC || !benchmark.EndAt.After(benchmark.StartAt) || benchmark.FrozenAt.IsZero() || benchmark.FrozenAt.Location() != time.UTC || strings.TrimSpace(benchmark.ExpectedProtocol) == "" {
		return fmt.Errorf("frozen benchmark identity, UTC window, dataset hash and protocol are required")
	}
	switch benchmark.Split {
	case BenchmarkDevelopment, BenchmarkValidation, BenchmarkOutOfSample, BenchmarkFinalHoldout:
	default:
		return fmt.Errorf("unsupported benchmark split %q", benchmark.Split)
	}
	if len(benchmark.Instruments) == 0 || !strictSortedUnique(benchmark.Instruments) || len(benchmark.CaseIDs) == 0 || !strictSortedUnique(benchmark.CaseIDs) {
		return fmt.Errorf("frozen benchmark requires sorted unique instruments and cases")
	}
	for _, caseID := range benchmark.CaseIDs {
		if !validIdentity("hcase_", caseID) {
			return fmt.Errorf("benchmark case ID is not a historical case identity: %q", caseID)
		}
		decisionAt, ok := benchmark.DecisionTimes[caseID]
		if !ok || decisionAt.IsZero() || decisionAt.Location() != time.UTC || decisionAt.Before(benchmark.StartAt) || decisionAt.After(benchmark.EndAt) {
			return fmt.Errorf("benchmark decision time is missing or outside its window for %q", caseID)
		}
	}
	for caseID := range benchmark.DecisionTimes {
		if !containsString(benchmark.CaseIDs, caseID) {
			return fmt.Errorf("benchmark decision time exists for an undeclared case")
		}
	}
	for evidenceID, vintage := range benchmark.AllowedEvidenceVintages {
		if strings.TrimSpace(evidenceID) == "" || strings.TrimSpace(vintage) == "" {
			return fmt.Errorf("benchmark evidence vintages require non-empty identities")
		}
	}
	if err := benchmark.Configuration.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(benchmark.Provenance.Source) == "" || strings.TrimSpace(benchmark.Provenance.ManifestRef) == "" || benchmark.Provenance.RecordedAt.IsZero() || benchmark.Provenance.RecordedAt.Location() != time.UTC {
		return fmt.Errorf("benchmark provenance is required")
	}
	if benchmark.VariantsTried < 1 || benchmark.OutcomesUsedForTuning || benchmark.HoldoutTouched && benchmark.Split == BenchmarkFinalHoldout {
		return fmt.Errorf("benchmark freeze records invalid tuning or holdout state")
	}
	if benchmark.ID != deriveFrozenBenchmarkID(benchmark) {
		return fmt.Errorf("frozen benchmark ID does not match its contents")
	}
	return nil
}

type BenchmarkRegistry struct {
	mu         sync.RWMutex
	benchmarks map[string]FrozenBenchmark
}

func NewBenchmarkRegistry() *BenchmarkRegistry {
	return &BenchmarkRegistry{benchmarks: map[string]FrozenBenchmark{}}
}

func (registry *BenchmarkRegistry) Register(benchmark FrozenBenchmark) error {
	if registry == nil {
		return fmt.Errorf("benchmark registry is required")
	}
	if err := benchmark.Validate(); err != nil {
		return err
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if existing, ok := registry.benchmarks[benchmark.ID]; ok {
		if stringMustJSON(existing) != stringMustJSON(benchmark) {
			return fmt.Errorf("%w: %s", ErrBenchmarkImmutable, benchmark.ID)
		}
		return nil
	}
	registry.benchmarks[benchmark.ID] = benchmark
	return nil
}

func (registry *BenchmarkRegistry) Get(id string) (FrozenBenchmark, error) {
	if registry == nil {
		return FrozenBenchmark{}, fmt.Errorf("benchmark registry is required")
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	benchmark, ok := registry.benchmarks[id]
	if !ok {
		return FrozenBenchmark{}, fmt.Errorf("benchmark %q is not registered", id)
	}
	return benchmark, nil
}

func (registry *BenchmarkRegistry) List() []FrozenBenchmark {
	if registry == nil {
		return nil
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	values := make([]FrozenBenchmark, 0, len(registry.benchmarks))
	for _, benchmark := range registry.benchmarks {
		values = append(values, benchmark)
	}
	sort.Slice(values, func(left, right int) bool { return values[left].ID < values[right].ID })
	return values
}

func deriveFrozenBenchmarkID(benchmark FrozenBenchmark) string {
	benchmark.ID = ""
	seed, _ := json.Marshal(benchmark)
	digest := sha256.Sum256(seed)
	return "bench_" + hex.EncodeToString(digest[:])
}

func strictSortedUnique(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] >= values[index] {
			return false
		}
	}
	return true
}

func containsString(values []string, want string) bool {
	index := sort.SearchStrings(values, want)
	return index < len(values) && values[index] == want
}

func stringMustJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
