package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestVersionComparisonUsesSameFrozenBenchmarkAndReportsEveryVariant(t *testing.T) {
	benchmark := benchmarkFixture(t)
	first, err := NewComparisonVariant(ComparisonVariant{Provider: "fixture", Model: "model-a", PromptVersion: "prompt-a", SystemVersion: "system-a", ResearchContractVersion: "research-v1", RecommendationContractVersion: "recommendation-v1", EligibilityRulesVersion: "eligibility-v1", QuantAlgorithmVersions: []string{"quant-v1"}, PolicyVersion: "policy-a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewComparisonVariant(ComparisonVariant{Provider: "fixture", Model: "model-b", PromptVersion: "prompt-b", SystemVersion: "system-a", ResearchContractVersion: "research-v1", RecommendationContractVersion: "recommendation-v1", EligibilityRulesVersion: "eligibility-v1", QuantAlgorithmVersions: []string{"quant-v1"}, PolicyVersion: "policy-a"})
	if err != nil {
		t.Fatal(err)
	}
	caseResult := func(variant string) VariantComparisonResult {
		return VariantComparisonResult{VariantID: variant, Cases: []VariantCaseResult{{CaseID: benchmark.CaseIDs[0], ResearchOutputID: "rso_" + strings.Repeat("b", 64), RecommendationID: "rec6_" + strings.Repeat("c", 64), Disposition: "WATCH", ThesisFingerprint: strings.Repeat("d", 64)}}}
	}
	comparison, err := NewVersionComparison(benchmark, []ComparisonVariant{first, second}, []VariantComparisonResult{caseResult(first.ID), caseResult(second.ID)}, "report all variants; no winner selected")
	if err != nil {
		t.Fatal(err)
	}
	if err := comparison.Validate(benchmark, []ComparisonVariant{first, second}); err != nil || len(comparison.ReportedVariantIDs) != 2 {
		t.Fatalf("comparison validation=%v artifact=%+v", err, comparison)
	}
}

func TestVersionComparisonRejectsMissingCasesAndHoldoutTouch(t *testing.T) {
	benchmark := benchmarkFixture(t)
	variant, err := NewComparisonVariant(ComparisonVariant{Provider: "fixture", Model: "model", PromptVersion: "prompt", SystemVersion: "system", ResearchContractVersion: "research-v1", RecommendationContractVersion: "recommendation-v1", EligibilityRulesVersion: "eligibility-v1", QuantAlgorithmVersions: []string{"quant-v1"}, PolicyVersion: "policy"})
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := NewVersionComparison(benchmark, []ComparisonVariant{variant}, []VariantComparisonResult{{VariantID: variant.ID, Cases: nil}}, "report all")
	if err == nil {
		t.Fatal("comparison accepted missing benchmark cases")
	}
	_ = comparison
	validCase := VariantCaseResult{CaseID: benchmark.CaseIDs[0], ResearchOutputID: "rso_" + strings.Repeat("b", 64), RecommendationID: "rec6_" + strings.Repeat("c", 64), Disposition: "WATCH", ThesisFingerprint: strings.Repeat("d", 64)}
	comparison, err = NewVersionComparison(benchmark, []ComparisonVariant{variant}, []VariantComparisonResult{{VariantID: variant.ID, Cases: []VariantCaseResult{validCase}}}, "report all")
	if err != nil {
		t.Fatal(err)
	}
	comparison.HoldoutTouched = true
	if err := comparison.Validate(benchmark, []ComparisonVariant{variant}); err == nil {
		t.Fatal("comparison accepted holdout-touch state")
	}
}

func TestComparisonThesisFingerprintUsesCanonicalSHA(t *testing.T) {
	digest := sha256.Sum256([]byte("thesis"))
	if !validSHA256(hex.EncodeToString(digest[:])) {
		t.Fatal("canonical SHA helper rejected known digest")
	}
}
