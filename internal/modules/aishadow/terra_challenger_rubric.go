package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

const C1F3TerraChallengerRubricVersion = "ai-shadow-c1f3-terra-challenger-rubric-v1"

const (
	C1F3TerraMateriallyBetter     = "MATERIALLY BETTER"
	C1F3TerraBetterButNotMaterial = "BETTER BUT NOT MATERIAL"
	C1F3TerraEquivalent           = "EQUIVALENT"
	C1F3TerraWorse                = "WORSE"
)

type C1F3TerraChallengerBaseline struct {
	Profile                    string `json:"profile"`
	RunID                      string `json:"run_id"`
	ArtifactIndexSHA256        string `json:"artifact_index_sha256"`
	SemanticMappingCorrect     int    `json:"semantic_mapping_correct"`
	SemanticMappingDenominator int    `json:"semantic_mapping_denominator"`
	ProxyRecallCorrect         int    `json:"proxy_recall_correct"`
	ProxyRecallDenominator     int    `json:"proxy_recall_denominator"`
}

type C1F3TerraQualityRetentionGates struct {
	FinalValidityMinimumPercent              float64 `json:"final_validity_minimum_percent"`
	SemanticDirectPrecisionMinimumPercent    float64 `json:"semantic_direct_precision_minimum_percent"`
	SemanticDirectRecallMinimumPercent       float64 `json:"semantic_direct_recall_minimum_percent"`
	SemanticFalseDirectMaximumPercent        float64 `json:"semantic_false_direct_maximum_percent"`
	IncorrectDeterministicResolutionsMaximum int     `json:"incorrect_deterministic_resolutions_maximum"`
	SafetyPersistenceViolationsMaximum       int     `json:"safety_persistence_violations_maximum"`
}

type C1F3TerraMaterialImprovementGates struct {
	SemanticMappingCorrectMinimum int `json:"semantic_mapping_correct_minimum"`
	SemanticMappingDenominator    int `json:"semantic_mapping_denominator"`
	ProxyRecallCorrectMinimum     int `json:"proxy_recall_correct_minimum"`
	ProxyRecallDenominator        int `json:"proxy_recall_denominator"`
}

type C1F3TerraChallengerRubric struct {
	Version                  string                            `json:"version"`
	CreatedAt                time.Time                         `json:"created_at"`
	Fingerprint              string                            `json:"fingerprint"`
	Question                 string                            `json:"question"`
	Baseline                 C1F3TerraChallengerBaseline       `json:"baseline"`
	QualityRetentionGates    C1F3TerraQualityRetentionGates    `json:"quality_retention_gates"`
	MaterialImprovementGates C1F3TerraMaterialImprovementGates `json:"material_improvement_gates"`
	Categories               []string                          `json:"categories"`
	CategoryRules            map[string]string                 `json:"category_rules"`
	CaseComparisonCategories []string                          `json:"case_comparison_categories"`
	ComparisonMetrics        []string                          `json:"comparison_metrics"`
	FrozenLunaFailureCases   []string                          `json:"frozen_luna_failure_cases"`
	NoPostResultTuning       bool                              `json:"no_post_result_tuning"`
}

func c1f3TerraChallengerRubricFingerprint(rubric C1F3TerraChallengerRubric) (string, error) {
	copy := rubric
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func LoadFrozenC1F3TerraChallengerRubric(path, expectedFileSHA, expectedFingerprint string) (C1F3TerraChallengerRubric, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return C1F3TerraChallengerRubric{}, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != expectedFileSHA {
		return C1F3TerraChallengerRubric{}, fmt.Errorf("Terra challenger rubric file hash changed: got %s want %s", got, expectedFileSHA)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var rubric C1F3TerraChallengerRubric
	if err := decoder.Decode(&rubric); err != nil {
		return C1F3TerraChallengerRubric{}, err
	}
	if err := ensureEOF(decoder); err != nil {
		return C1F3TerraChallengerRubric{}, err
	}
	computed, err := c1f3TerraChallengerRubricFingerprint(rubric)
	if err != nil || rubric.Fingerprint != computed || rubric.Fingerprint != expectedFingerprint {
		return C1F3TerraChallengerRubric{}, fmt.Errorf("Terra challenger rubric fingerprint changed: got %s computed %s want %s", rubric.Fingerprint, computed, expectedFingerprint)
	}
	wantCategories := []string{C1F3TerraMateriallyBetter, C1F3TerraBetterButNotMaterial, C1F3TerraEquivalent, C1F3TerraWorse}
	wantComparisons := []string{"both correct", "Luna correct / Terra wrong", "Luna wrong / Terra correct", "both wrong same way", "both wrong differently"}
	wantMetrics := []string{"effective_mapping_agreement", "typed_attribution_agreement", "policy_decision_agreement", "deterministic_resolution_agreement"}
	if rubric.Version != C1F3TerraChallengerRubricVersion || rubric.Baseline != frozenC1F3TerraChallengerBaseline() ||
		rubric.QualityRetentionGates != frozenC1F3TerraQualityRetentionGates() || rubric.MaterialImprovementGates != frozenC1F3TerraMaterialImprovementGates() ||
		!reflect.DeepEqual(rubric.Categories, wantCategories) || !reflect.DeepEqual(rubric.CaseComparisonCategories, wantComparisons) ||
		!reflect.DeepEqual(rubric.ComparisonMetrics, wantMetrics) || !reflect.DeepEqual(rubric.FrozenLunaFailureCases, []string{"041", "042", "043"}) ||
		len(rubric.CategoryRules) != 4 || !rubric.NoPostResultTuning {
		return C1F3TerraChallengerRubric{}, fmt.Errorf("Terra challenger rubric bindings changed")
	}
	return rubric, nil
}

func frozenC1F3TerraChallengerBaseline() C1F3TerraChallengerBaseline {
	return C1F3TerraChallengerBaseline{
		Profile: C1F3RepeatabilityR3ProfileIdentity, RunID: C1F3TerraAcceptedLunaRunID,
		ArtifactIndexSHA256:    C1F3TerraAcceptedLunaArtifactIndexSHA256,
		SemanticMappingCorrect: 45, SemanticMappingDenominator: 48, ProxyRecallCorrect: 3, ProxyRecallDenominator: 6,
	}
}

func frozenC1F3TerraQualityRetentionGates() C1F3TerraQualityRetentionGates {
	return C1F3TerraQualityRetentionGates{
		FinalValidityMinimumPercent: 98, SemanticDirectPrecisionMinimumPercent: 95,
		SemanticDirectRecallMinimumPercent: 90, SemanticFalseDirectMaximumPercent: 5,
		IncorrectDeterministicResolutionsMaximum: 0, SafetyPersistenceViolationsMaximum: 0,
	}
}

func frozenC1F3TerraMaterialImprovementGates() C1F3TerraMaterialImprovementGates {
	return C1F3TerraMaterialImprovementGates{SemanticMappingCorrectMinimum: 47, SemanticMappingDenominator: 48, ProxyRecallCorrectMinimum: 5, ProxyRecallDenominator: 6}
}

func C1F3TerraChallengerDisposition(rubric C1F3TerraChallengerRubric, score C1F3RepeatabilityScore) string {
	semantic := score.AccuracyRetention.Semantic
	g := rubric.QualityRetentionGates
	retained := semantic.FinalValidity.Percentage >= g.FinalValidityMinimumPercent &&
		semantic.DirectPrecision.Percentage >= g.SemanticDirectPrecisionMinimumPercent &&
		semantic.DirectRecall.Percentage >= g.SemanticDirectRecallMinimumPercent &&
		semantic.FalseDirect.Percentage <= g.SemanticFalseDirectMaximumPercent &&
		score.IncorrectDeterministicResolutions <= g.IncorrectDeterministicResolutionsMaximum &&
		score.SafetyPersistenceViolations <= g.SafetyPersistenceViolationsMaximum
	if !retained {
		return C1F3TerraWorse
	}
	m := rubric.MaterialImprovementGates
	if semantic.WholeMapping.Denominator == m.SemanticMappingDenominator && semantic.WholeMapping.Numerator >= m.SemanticMappingCorrectMinimum &&
		semantic.ProxyRecall.Denominator == m.ProxyRecallDenominator && semantic.ProxyRecall.Numerator >= m.ProxyRecallCorrectMinimum {
		return C1F3TerraMateriallyBetter
	}
	if semantic.WholeMapping.Numerator > rubric.Baseline.SemanticMappingCorrect || semantic.ProxyRecall.Numerator > rubric.Baseline.ProxyRecallCorrect {
		return C1F3TerraBetterButNotMaterial
	}
	if semantic.WholeMapping.Numerator < rubric.Baseline.SemanticMappingCorrect || semantic.ProxyRecall.Numerator < rubric.Baseline.ProxyRecallCorrect {
		return C1F3TerraWorse
	}
	return C1F3TerraEquivalent
}
