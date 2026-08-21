package aishadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

type C1F3TerraCaseComparison struct {
	CaseID                 string                     `json:"case_id"`
	ExpectedMapping        AssetMapping               `json:"expected_mapping"`
	LunaMapping            AssetMapping               `json:"luna_mapping"`
	TerraMapping           AssetMapping               `json:"terra_mapping"`
	LunaCorrect            bool                       `json:"luna_correct"`
	TerraCorrect           bool                       `json:"terra_correct"`
	Classification         string                     `json:"classification"`
	EffectiveMappingAgrees bool                       `json:"effective_mapping_agrees"`
	IdentityOutcome        string                     `json:"identity_outcome"`
	CausalRoleDifferences  []string                   `json:"causal_role_differences"`
	LunaProxyCandidates    []string                   `json:"luna_proxy_candidates"`
	TerraProxyCandidates   []string                   `json:"terra_proxy_candidates"`
	ProxyCandidatesDiffer  bool                       `json:"proxy_candidates_differ"`
	LunaPolicy             *CausalAttributionDecision `json:"luna_policy"`
	TerraPolicy            *CausalAttributionDecision `json:"terra_policy"`
	PolicyDiffers          bool                       `json:"policy_differs"`
	LunaResolver           *PolicyResolution          `json:"luna_resolver"`
	TerraResolver          *PolicyResolution          `json:"terra_resolver"`
	ResolverDiffers        bool                       `json:"resolver_differs"`
}

type C1F3TerraComparisonSummary struct {
	BothCorrect             int      `json:"both_correct"`
	LunaCorrectTerraWrong   int      `json:"luna_correct_terra_wrong"`
	LunaWrongTerraCorrect   int      `json:"luna_wrong_terra_correct"`
	BothWrongSameWay        int      `json:"both_wrong_same_way"`
	BothWrongDifferently    int      `json:"both_wrong_differently"`
	EffectiveMapping        C1F3Rate `json:"effective_mapping_agreement"`
	TypedAttribution        C1F3Rate `json:"typed_attribution_agreement"`
	PolicyDecision          C1F3Rate `json:"policy_decision_agreement"`
	DeterministicResolution C1F3Rate `json:"deterministic_resolution_agreement"`
}

type C1F3TerraChallengerScore struct {
	Version                     string                           `json:"version"`
	RubricFingerprint           string                           `json:"rubric_fingerprint"`
	AcceptedLuna                C1F3TerraLunaPreservationBinding `json:"accepted_luna"`
	TerraRunID                  string                           `json:"terra_run_id"`
	TerraArtifactIndexSHA256    string                           `json:"terra_artifact_index_sha256"`
	TerraProviderSnapshot       *HostedExperimentSnapshot        `json:"terra_provider_snapshot"`
	TerraAccuracy               C1FDualScore                     `json:"terra_accuracy"`
	TerraPolicy                 C1F3PolicyAnalysis               `json:"terra_policy"`
	TerraResolver               C1F3ResolverAnalysis             `json:"terra_resolver"`
	IncorrectResolutions        int                              `json:"incorrect_deterministic_ticker_or_rule_resolutions"`
	SafetyPersistenceViolations int                              `json:"safety_persistence_violations"`
	Comparison                  C1F3TerraComparisonSummary       `json:"comparison"`
	Cases                       []C1F3TerraCaseComparison        `json:"cases"`
	ChangedEffectiveMappings    []C1F3TerraCaseComparison        `json:"changed_effective_mappings"`
	IncorrectSemanticMappings   []C1F3FailureRecord              `json:"incorrect_semantic_mappings"`
	QualityRetentionGates       []C1F3GateResult                 `json:"quality_retention_gates"`
	MaterialImprovementGates    []C1F3GateResult                 `json:"material_improvement_gates"`
	Disposition                 string                           `json:"disposition"`
}

type c1f3ComparisonRun struct {
	Plan     DiagnosticPlan
	Report   DiagnosticRunReport
	Audits   []DiagnosticAttemptAudit
	IndexSHA string
}

func LoadC1F3ComparisonRun(runDirectory, expectedIndexSHA, expectedRunID, expectedProfile, expectedModel string, inputs map[string]EventInput, resolver assetresolution.Resolver) (c1f3ComparisonRun, error) {
	_, artifactHashes, err := verifyC1F3ArtifactIndex(runDirectory, expectedIndexSHA)
	if err != nil {
		return c1f3ComparisonRun{}, err
	}
	planRaw, err := os.ReadFile(filepath.Join(runDirectory, "plan.json"))
	if err != nil {
		return c1f3ComparisonRun{}, err
	}
	var planEnvelope struct {
		RunID string         `json:"run_id"`
		Plan  DiagnosticPlan `json:"plan"`
	}
	if err := json.Unmarshal(planRaw, &planEnvelope); err != nil {
		return c1f3ComparisonRun{}, err
	}
	reportRaw, err := os.ReadFile(filepath.Join(runDirectory, "report.json"))
	if err != nil {
		return c1f3ComparisonRun{}, err
	}
	var report DiagnosticRunReport
	if err := json.Unmarshal(reportRaw, &report); err != nil {
		return c1f3ComparisonRun{}, err
	}
	if planEnvelope.RunID != expectedRunID || report.RunID != expectedRunID || planEnvelope.Plan.EvaluationProfile != expectedProfile ||
		planEnvelope.Plan.ModelConfiguration.Provider != OpenAIDiagnosticProvider || planEnvelope.Plan.ModelConfiguration.Model != expectedModel ||
		planEnvelope.Plan.ModelConfiguration.ReasoningEffort != OpenAIDiagnosticReasoningEffort || planEnvelope.Plan.PromptVersion != V6PromptVersion ||
		planEnvelope.Plan.OutputContract != V5SchemaVersion || planEnvelope.Plan.ValidatorVersion != C1FValidatorVersion ||
		planEnvelope.Plan.CausalAttributionPolicy != CausalAttributionPolicyVersion || planEnvelope.Plan.ScoringVersion != C1FScoringVersion ||
		planEnvelope.Plan.CasesPerRepetition != 48 || planEnvelope.Plan.Repetitions != 1 || report.ModelIdentity.Name != expectedModel ||
		report.HostedExperiment == nil || report.HostedExperiment.RequestedModel != expectedModel || len(report.Repetitions) != 1 {
		return c1f3ComparisonRun{}, fmt.Errorf("comparison run %s does not match its frozen execution contract", expectedRunID)
	}
	paths, err := filepath.Glob(filepath.Join(runDirectory, "repetition-01", "*-attempt-*.json"))
	if err != nil {
		return c1f3ComparisonRun{}, err
	}
	sort.Strings(paths)
	audits := make([]DiagnosticAttemptAudit, 0, len(paths))
	finalCases := map[string]bool{}
	for _, path := range paths {
		relative, _ := filepath.Rel(runDirectory, path)
		if artifactHashes[filepath.ToSlash(relative)] == "" {
			return c1f3ComparisonRun{}, fmt.Errorf("comparison attempt is not indexed: %s", relative)
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return c1f3ComparisonRun{}, readErr
		}
		var audit DiagnosticAttemptAudit
		if err := json.Unmarshal(raw, &audit); err != nil {
			return c1f3ComparisonRun{}, err
		}
		input, ok := inputs[audit.CaseID]
		if !ok || audit.RunID != expectedRunID || audit.ConfiguredModel != expectedModel || audit.ModelReportedIdentifier != expectedModel || strings.TrimSpace(audit.RawResponseBody) == "" {
			return c1f3ComparisonRun{}, fmt.Errorf("comparison attempt identity changed for %s", audit.CaseID)
		}
		rawDigest := sha256.Sum256([]byte(audit.RawResponseBody))
		if audit.RawResponseHash != hex.EncodeToString(rawDigest[:]) {
			return c1f3ComparisonRun{}, fmt.Errorf("raw response digest changed for %s", audit.CaseID)
		}
		parsed, decision, resolution, validationErrors := ParseValidateAndApplyC1F(audit.RawResponseBody, input, resolver)
		if audit.ValidationStatus == "accepted" {
			if len(validationErrors) != 0 || parsed == nil || decision == nil || resolution == nil {
				return c1f3ComparisonRun{}, fmt.Errorf("accepted case %s does not revalidate through frozen C1F: %v", audit.CaseID, validationErrors)
			}
			attribution := TypedAttributionFromV5(*parsed)
			effective := decision.EffectiveMapping
			audit.V5RawModelOutput = parsed
			audit.TypedAttribution = &attribution
			audit.CausalAttributionPolicy = decision
			audit.EffectiveSemanticMapping = &effective
			audit.DeterministicResolution = resolution
			finalCases[audit.CaseID] = true
		} else if audit.AttemptNumber == 2 {
			return c1f3ComparisonRun{}, fmt.Errorf("final corrective attempt rejected for %s", audit.CaseID)
		}
		audits = append(audits, audit)
	}
	if len(finalCases) != 48 {
		return c1f3ComparisonRun{}, fmt.Errorf("comparison run has %d accepted final cases; want 48", len(finalCases))
	}
	return c1f3ComparisonRun{Plan: planEnvelope.Plan, Report: report, Audits: audits, IndexSHA: expectedIndexSHA}, nil
}

func BuildC1F3TerraChallengerScore(repositoryRoot, terraRunDirectory, terraArtifactIndexSHA256 string) (C1F3TerraChallengerScore, error) {
	lunaPreservation, err := ValidateC1F3TerraAcceptedLunaEvidence(repositoryRoot)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	rubric, err := LoadFrozenC1F3TerraChallengerRubric(filepath.Join(repositoryRoot, filepath.FromSlash(C1F3TerraChallengerRubricPath)), C1F3TerraChallengerRubricFileSHA256, C1F3TerraChallengerRubricFingerprint)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	profile, err := LoadC1F3EvaluationProfile(C1F3ProfileGeneralization)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	rules, err := assetresolution.LoadRuleset(filepath.Join(repositoryRoot, "config", "event-asset-resolution-v1.json"))
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	resolver := assetresolution.Resolver{Rules: rules}
	exposures, err := resolver.ProxyExposures()
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	manifest, err := LoadFrozenC1F3Manifest(profile, filepath.Join(repositoryRoot, filepath.FromSlash(profile.Dataset.ManifestPath)), exposures)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	labels, err := LoadFrozenC1F3TypedLabelSidecar(profile, filepath.Join(repositoryRoot, filepath.FromSlash(profile.TypedSidecarPath)))
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	if err := ValidateC1F3TypedLabelSidecar(profile, labels, manifest, resolver); err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	inputs := map[string]EventInput{}
	for _, event := range manifest.Events {
		inputs[event.ID] = event.Input
	}
	lunaDirectory := filepath.Join(repositoryRoot, filepath.FromSlash(C1F3TerraAcceptedLunaRelativeDirectory))
	luna, err := LoadC1F3ComparisonRun(lunaDirectory, C1F3TerraAcceptedLunaArtifactIndexSHA256, C1F3TerraAcceptedLunaRunID, C1F3RepeatabilityR3ProfileIdentity, OpenAIDiagnosticLunaModel, inputs, resolver)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	terraIndexRaw, err := os.ReadFile(filepath.Join(terraRunDirectory, "artifact-index.json"))
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	terraIndexDigest := sha256.Sum256(terraIndexRaw)
	actualTerraIndexSHA := hex.EncodeToString(terraIndexDigest[:])
	if actualTerraIndexSHA != terraArtifactIndexSHA256 {
		return C1F3TerraChallengerScore{}, fmt.Errorf("Terra artifact-index SHA-256 mismatch: got %s want %s", actualTerraIndexSHA, terraArtifactIndexSHA256)
	}
	var terraReport DiagnosticRunReport
	reportRaw, err := os.ReadFile(filepath.Join(terraRunDirectory, "report.json"))
	if err != nil || json.Unmarshal(reportRaw, &terraReport) != nil || terraReport.RunID == "" {
		return C1F3TerraChallengerScore{}, fmt.Errorf("load Terra report")
	}
	terra, err := LoadC1F3ComparisonRun(terraRunDirectory, terraArtifactIndexSHA256, terraReport.RunID, C1F3TerraChallengerProfileIdentity, OpenAIDiagnosticTerraModel, inputs, resolver)
	if err != nil {
		return C1F3TerraChallengerScore{}, err
	}
	identity := NewIssuerSemanticIdentity(rules)
	safetyViolations := c1f3SafetyViolations(terra.Plan.Safety)
	comparisonScore := ScoreC1F3Repeatability(labels.Cases, manifest, luna.Audits, terra.Audits, identity, resolver, safetyViolations)
	dataset := C1F3DatasetReprojection{Score: comparisonScore.AccuracyRetention, SafetyPersistenceViolations: safetyViolations}
	analyzeC1F3Dataset(&dataset, labels.Cases, manifest, terra.Audits, identity, resolver)

	lunaFinal, terraFinal := finalC1FAudits(luna.Audits), finalC1FAudits(terra.Audits)
	cases := make([]C1F3TerraCaseComparison, 0, 48)
	changed := []C1F3TerraCaseComparison{}
	summary := C1F3TerraComparisonSummary{
		EffectiveMapping:        comparisonScore.Metrics.SemanticEffectiveMapping,
		TypedAttribution:        comparisonScore.Metrics.WholeTypedAttribution,
		PolicyDecision:          comparisonScore.Metrics.PolicyDecision,
		DeterministicResolution: comparisonScore.Metrics.DeterministicResolution,
	}
	for _, expected := range labels.Cases {
		before, after := lunaFinal[expected.CaseID], terraFinal[expected.CaseID]
		beforeMapping, afterMapping := auditMapping(before), auditMapping(after)
		agrees, outcome := repeatabilityMappingAgreement(beforeMapping, afterMapping, identity, true)
		beforeCorrect := c1fMappingMatches(expected, beforeMapping, identity, true)
		afterCorrect := c1fMappingMatches(expected, afterMapping, identity, true)
		classification := ""
		switch {
		case beforeCorrect && afterCorrect:
			classification = "both correct"
			summary.BothCorrect++
		case beforeCorrect && !afterCorrect:
			classification = "Luna correct / Terra wrong"
			summary.LunaCorrectTerraWrong++
		case !beforeCorrect && afterCorrect:
			classification = "Luna wrong / Terra correct"
			summary.LunaWrongTerraCorrect++
		case agrees:
			classification = "both wrong same way"
			summary.BothWrongSameWay++
		default:
			classification = "both wrong differently"
			summary.BothWrongDifferently++
		}
		beforeAttrs, beforeCandidates := auditAttribution(before)
		afterAttrs, afterCandidates := auditAttribution(after)
		roleDiffs := []string{}
		for _, role := range []CausalRole{CausalRolePrincipal, CausalRoleEqualPrincipal, CausalRoleSecondaryAffected, CausalRoleContextOnly, CausalRolePossiblePrincipal} {
			roleAgrees, _ := c1f3AttributionMatches(filterRole(beforeAttrs, role), filterRole(afterAttrs, role), identity, true, nil)
			if !roleAgrees {
				roleDiffs = append(roleDiffs, string(role))
			}
		}
		entry := C1F3TerraCaseComparison{
			CaseID: expected.CaseID, ExpectedMapping: AssetMapping{MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure},
			LunaMapping: beforeMapping, TerraMapping: afterMapping, LunaCorrect: beforeCorrect, TerraCorrect: afterCorrect,
			Classification: classification, EffectiveMappingAgrees: agrees, IdentityOutcome: outcome, CausalRoleDifferences: roleDiffs,
			LunaProxyCandidates: beforeCandidates, TerraProxyCandidates: afterCandidates, ProxyCandidatesDiffer: !stringSetEqual(beforeCandidates, afterCandidates),
			LunaPolicy: before.CausalAttributionPolicy, TerraPolicy: after.CausalAttributionPolicy, PolicyDiffers: !reflect.DeepEqual(before.CausalAttributionPolicy, after.CausalAttributionPolicy),
			LunaResolver: before.DeterministicResolution, TerraResolver: after.DeterministicResolution, ResolverDiffers: !reflect.DeepEqual(before.DeterministicResolution, after.DeterministicResolution),
		}
		cases = append(cases, entry)
		if !agrees {
			changed = append(changed, entry)
		}
	}
	g := rubric.QualityRetentionGates
	semantic := comparisonScore.AccuracyRetention.Semantic
	qualityGates := []C1F3GateResult{
		{Gate: "final validity", Observed: fmt.Sprintf("%d/%d", semantic.FinalValidity.Numerator, semantic.FinalValidity.Denominator), Required: ">=98%", Passed: semantic.FinalValidity.Percentage >= g.FinalValidityMinimumPercent},
		{Gate: "semantic DIRECT precision", Observed: fmt.Sprintf("%d/%d", semantic.DirectPrecision.Numerator, semantic.DirectPrecision.Denominator), Required: ">=95%", Passed: semantic.DirectPrecision.Percentage >= g.SemanticDirectPrecisionMinimumPercent},
		{Gate: "semantic DIRECT recall", Observed: fmt.Sprintf("%d/%d", semantic.DirectRecall.Numerator, semantic.DirectRecall.Denominator), Required: ">=90%", Passed: semantic.DirectRecall.Percentage >= g.SemanticDirectRecallMinimumPercent},
		{Gate: "semantic false DIRECT", Observed: fmt.Sprintf("%d/%d", semantic.FalseDirect.Numerator, semantic.FalseDirect.Denominator), Required: "<=5%", Passed: semantic.FalseDirect.Percentage <= g.SemanticFalseDirectMaximumPercent},
		{Gate: "incorrect deterministic ticker/rule resolutions", Observed: fmt.Sprint(dataset.Resolver.IncorrectCount), Required: "=0", Passed: dataset.Resolver.IncorrectCount == g.IncorrectDeterministicResolutionsMaximum},
		{Gate: "safety/persistence violations", Observed: fmt.Sprint(safetyViolations), Required: "=0", Passed: safetyViolations == g.SafetyPersistenceViolationsMaximum},
	}
	m := rubric.MaterialImprovementGates
	materialGates := []C1F3GateResult{
		{Gate: "semantic mapping correctness", Observed: fmt.Sprintf("%d/%d", semantic.WholeMapping.Numerator, semantic.WholeMapping.Denominator), Required: ">=47/48", Passed: semantic.WholeMapping.Numerator >= m.SemanticMappingCorrectMinimum && semantic.WholeMapping.Denominator == m.SemanticMappingDenominator},
		{Gate: "PROXY recall", Observed: fmt.Sprintf("%d/%d", semantic.ProxyRecall.Numerator, semantic.ProxyRecall.Denominator), Required: ">=5/6", Passed: semantic.ProxyRecall.Numerator >= m.ProxyRecallCorrectMinimum && semantic.ProxyRecall.Denominator == m.ProxyRecallDenominator},
	}
	return C1F3TerraChallengerScore{
		Version: C1F3TerraChallengerRubricVersion, RubricFingerprint: rubric.Fingerprint, AcceptedLuna: lunaPreservation,
		TerraRunID: terra.Report.RunID, TerraArtifactIndexSHA256: terraArtifactIndexSHA256, TerraProviderSnapshot: terra.Report.HostedExperiment,
		TerraAccuracy: dataset.Score, TerraPolicy: dataset.Policy, TerraResolver: dataset.Resolver,
		IncorrectResolutions: dataset.Resolver.IncorrectCount, SafetyPersistenceViolations: safetyViolations,
		Comparison: summary, Cases: cases, ChangedEffectiveMappings: changed, IncorrectSemanticMappings: dataset.MappingFailures,
		QualityRetentionGates: qualityGates, MaterialImprovementGates: materialGates,
		Disposition: C1F3TerraChallengerDisposition(rubric, comparisonScore),
	}, nil
}
