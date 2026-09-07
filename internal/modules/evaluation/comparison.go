package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const VersionComparisonContractV1 = "jax.version_comparison/v1"

type ComparisonVariant struct {
	ID                            string   `json:"id"`
	Provider                      string   `json:"provider"`
	Model                         string   `json:"model"`
	PromptVersion                 string   `json:"prompt_version"`
	SystemVersion                 string   `json:"system_version"`
	ResearchContractVersion       string   `json:"research_contract_version"`
	RecommendationContractVersion string   `json:"recommendation_contract_version"`
	EligibilityRulesVersion       string   `json:"eligibility_rules_version"`
	QuantAlgorithmVersions        []string `json:"quant_algorithm_versions"`
	PolicyVersion                 string   `json:"policy_version"`
}

func NewComparisonVariant(variant ComparisonVariant) (ComparisonVariant, error) {
	variant.ID = deriveComparisonVariantID(variant)
	if err := variant.Validate(); err != nil {
		return ComparisonVariant{}, err
	}
	return variant, nil
}

func (variant ComparisonVariant) Validate() error {
	if !validIdentity("variant_", variant.ID) || strings.TrimSpace(variant.Provider) == "" || strings.TrimSpace(variant.Model) == "" || strings.TrimSpace(variant.PromptVersion) == "" || strings.TrimSpace(variant.SystemVersion) == "" || strings.TrimSpace(variant.ResearchContractVersion) == "" || strings.TrimSpace(variant.RecommendationContractVersion) == "" || strings.TrimSpace(variant.EligibilityRulesVersion) == "" || strings.TrimSpace(variant.PolicyVersion) == "" || len(variant.QuantAlgorithmVersions) == 0 || !strictSortedUnique(variant.QuantAlgorithmVersions) {
		return fmt.Errorf("comparison variant requires complete sorted version identities")
	}
	if variant.ID != deriveComparisonVariantID(variant) {
		return fmt.Errorf("comparison variant ID does not match its configuration")
	}
	return nil
}

type VariantCaseResult struct {
	CaseID            string `json:"case_id"`
	ResearchOutputID  string `json:"research_output_id"`
	RecommendationID  string `json:"recommendation_id"`
	Disposition       string `json:"disposition"`
	ThesisFingerprint string `json:"thesis_fingerprint"`
}

func (result VariantCaseResult) Validate() error {
	if !validIdentity("hcase_", result.CaseID) || !validIdentity("rso_", result.ResearchOutputID) || !validIdentity("rec6_", result.RecommendationID) || (result.Disposition != "NO_TRADE" && result.Disposition != "WATCH" && result.Disposition != "CANDIDATE") || !validSHA256(result.ThesisFingerprint) {
		return fmt.Errorf("variant case result requires bound research identities and thesis fingerprint")
	}
	return nil
}

type VariantComparisonResult struct {
	VariantID string              `json:"variant_id"`
	Cases     []VariantCaseResult `json:"cases"`
}

type VersionComparison struct {
	ContractVersion    string                    `json:"contract_version"`
	ID                 string                    `json:"id"`
	BenchmarkID        string                    `json:"benchmark_id"`
	VariantIDs         []string                  `json:"variant_ids"`
	VariantsTried      int                       `json:"variants_tried"`
	Results            []VariantComparisonResult `json:"results"`
	SelectionCriteria  string                    `json:"selection_criteria"`
	ReportedVariantIDs []string                  `json:"reported_variant_ids"`
	HoldoutTouched     bool                      `json:"holdout_touched"`
}

func NewVersionComparison(benchmark FrozenBenchmark, variants []ComparisonVariant, results []VariantComparisonResult, selectionCriteria string) (VersionComparison, error) {
	comparison := VersionComparison{ContractVersion: VersionComparisonContractV1, BenchmarkID: benchmark.ID, VariantsTried: len(variants), SelectionCriteria: selectionCriteria, HoldoutTouched: false}
	comparison.VariantIDs = make([]string, 0, len(variants))
	for _, variant := range variants {
		comparison.VariantIDs = append(comparison.VariantIDs, variant.ID)
	}
	sort.Strings(comparison.VariantIDs)
	comparison.ReportedVariantIDs = append([]string(nil), comparison.VariantIDs...)
	comparison.Results = append([]VariantComparisonResult(nil), results...)
	comparison.ID = deriveVersionComparisonID(comparison)
	if err := comparison.Validate(benchmark, variants); err != nil {
		return VersionComparison{}, err
	}
	return comparison, nil
}

func (comparison VersionComparison) Validate(benchmark FrozenBenchmark, variants []ComparisonVariant) error {
	if err := benchmark.Validate(); err != nil {
		return err
	}
	if comparison.ContractVersion != VersionComparisonContractV1 || !validIdentity("compare_", comparison.ID) || comparison.BenchmarkID != benchmark.ID || strings.TrimSpace(comparison.SelectionCriteria) == "" || comparison.HoldoutTouched || comparison.VariantsTried != len(variants) || len(variants) < 1 || len(comparison.Results) != len(variants) {
		return fmt.Errorf("comparison requires one complete frozen benchmark and all declared variants")
	}
	variantSet := map[string]struct{}{}
	for _, variant := range variants {
		if err := variant.Validate(); err != nil {
			return err
		}
		if _, exists := variantSet[variant.ID]; exists {
			return fmt.Errorf("comparison repeats variant %q", variant.ID)
		}
		variantSet[variant.ID] = struct{}{}
	}
	if !sameStrings(comparison.VariantIDs, comparison.ReportedVariantIDs) || !sameStrings(comparison.VariantIDs, keys(variantSet)) {
		return fmt.Errorf("comparison must report every variant tried")
	}
	caseSet := map[string]struct{}{}
	for _, caseID := range benchmark.CaseIDs {
		caseSet[caseID] = struct{}{}
	}
	resultVariants := map[string]struct{}{}
	for _, result := range comparison.Results {
		if _, ok := variantSet[result.VariantID]; !ok {
			return fmt.Errorf("comparison result references undeclared variant")
		}
		if _, exists := resultVariants[result.VariantID]; exists {
			return fmt.Errorf("comparison repeats result for variant %q", result.VariantID)
		}
		resultVariants[result.VariantID] = struct{}{}
		seenCases := map[string]struct{}{}
		for _, caseResult := range result.Cases {
			if err := caseResult.Validate(); err != nil {
				return err
			}
			if _, ok := caseSet[caseResult.CaseID]; !ok {
				return fmt.Errorf("comparison result references undeclared case")
			}
			if _, exists := seenCases[caseResult.CaseID]; exists {
				return fmt.Errorf("comparison repeats case %q", caseResult.CaseID)
			}
			seenCases[caseResult.CaseID] = struct{}{}
		}
		if len(seenCases) != len(caseSet) {
			return fmt.Errorf("comparison variant does not report every benchmark case")
		}
	}
	if !sameStrings(comparison.VariantIDs, keys(resultVariants)) || comparison.ID != deriveVersionComparisonID(comparison) {
		return fmt.Errorf("comparison identity or variant coverage is invalid")
	}
	return nil
}

func deriveComparisonVariantID(variant ComparisonVariant) string {
	variant.ID = ""
	seed, _ := json.Marshal(variant)
	digest := sha256.Sum256(seed)
	return "variant_" + hex.EncodeToString(digest[:])
}

func deriveVersionComparisonID(comparison VersionComparison) string {
	comparison.ID = ""
	seed, _ := json.Marshal(comparison)
	digest := sha256.Sum256(seed)
	return "compare_" + hex.EncodeToString(digest[:])
}

func keys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
