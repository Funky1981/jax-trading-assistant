package aishadow

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const C1F3RepeatabilityScoringVersion = "ai-shadow-c1f3-repeatability-scoring-v1"

type C1F3RepeatabilityBaseline struct {
	Profile             string `json:"profile"`
	RunID               string `json:"run_id"`
	ArtifactIndexSHA256 string `json:"artifact_index_sha256"`
}

type C1F3RepeatabilityImplementationBinding struct {
	Identity   string `json:"identity"`
	SHA256     string `json:"sha256"`
	SourcePath string `json:"source_path"`
}

type C1F3RepeatabilityScoringFreeze struct {
	Version                          string                                 `json:"version"`
	CreatedAt                        time.Time                              `json:"created_at"`
	Fingerprint                      string                                 `json:"fingerprint"`
	Baseline                         C1F3RepeatabilityBaseline              `json:"baseline"`
	UnderlyingAccuracyScorer         C1F3RepeatabilityImplementationBinding `json:"underlying_accuracy_scorer"`
	ComparisonImplementation         C1F3RepeatabilityImplementationBinding `json:"comparison_implementation"`
	SemanticIdentity                 string                                 `json:"semantic_identity"`
	Denominator                      int                                    `json:"denominator"`
	PrimaryMappingAgreementThreshold float64                                `json:"primary_mapping_agreement_threshold"`
	AccuracyRetentionGates           C1F3QualityGates                       `json:"accuracy_retention_gates"`
	SemanticEffectiveMappingRule     string                                 `json:"semantic_effective_mapping_rule"`
	StrictMappingRule                string                                 `json:"strict_mapping_rule"`
	TypedAttributionRule             string                                 `json:"typed_attribution_rule"`
	AttributionCompletenessRule      string                                 `json:"attribution_completeness_rule"`
	PolicyDecisionRule               string                                 `json:"policy_decision_rule"`
	DeterministicResolutionRule      string                                 `json:"deterministic_resolution_rule"`
	AccuracyContextRule              string                                 `json:"accuracy_context_rule"`
	ChangedCaseRule                  string                                 `json:"changed_case_rule"`
	OriginalFailureEvidence          []string                               `json:"original_failure_evidence"`
	Metrics                          []FrozenMetricDefinition               `json:"metrics"`
}

func c1f3RepeatabilityScoringFingerprint(freeze C1F3RepeatabilityScoringFreeze) (string, error) {
	copy := freeze
	copy.Fingerprint = ""
	return fingerprint(copy)
}

func LoadFrozenC1F3RepeatabilityScoring(path, expectedFileSHA, expectedFingerprint string) (C1F3RepeatabilityScoringFreeze, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return C1F3RepeatabilityScoringFreeze{}, err
	}
	digest := sha256.Sum256(raw)
	if got := hex.EncodeToString(digest[:]); got != expectedFileSHA {
		return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("C1F3 repeatability scoring file hash changed: got %s want %s", got, expectedFileSHA)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var freeze C1F3RepeatabilityScoringFreeze
	if err := decoder.Decode(&freeze); err != nil {
		return C1F3RepeatabilityScoringFreeze{}, err
	}
	if err := ensureEOF(decoder); err != nil {
		return C1F3RepeatabilityScoringFreeze{}, err
	}
	computed, err := c1f3RepeatabilityScoringFingerprint(freeze)
	if err != nil || freeze.Fingerprint != computed || freeze.Fingerprint != expectedFingerprint {
		return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("C1F3 repeatability scoring fingerprint changed: got %s computed %s want %s", freeze.Fingerprint, computed, expectedFingerprint)
	}
	if freeze.Version != C1F3RepeatabilityScoringVersion || freeze.Baseline != frozenC1F3RepeatabilityBaseline() ||
		freeze.UnderlyingAccuracyScorer.Identity != C1FScoringVersion || freeze.UnderlyingAccuracyScorer.SHA256 != frozenC1FScoringSourceSHA256 ||
		freeze.ComparisonImplementation.Identity != C1F3RepeatabilityScoringVersion || freeze.ComparisonImplementation.SHA256 != C1F3RepeatabilityComparisonSourceSHA256 ||
		freeze.SemanticIdentity != IssuerSemanticIdentityVersion || freeze.Denominator != 48 || freeze.PrimaryMappingAgreementThreshold != 90 ||
		freeze.AccuracyRetentionGates != FrozenC1F3QualityGates() || !reflect.DeepEqual(freeze.OriginalFailureEvidence, []string{"041", "043", "048"}) {
		return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("C1F3 repeatability scoring bindings changed")
	}
	repositoryRoot := filepath.Dir(filepath.Dir(path))
	if actual, err := hashOpaqueFile(filepath.Join(repositoryRoot, filepath.FromSlash(freeze.ComparisonImplementation.SourcePath))); err != nil || actual != freeze.ComparisonImplementation.SHA256 {
		return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("C1F3 repeatability comparison implementation hash changed")
	}
	required := []string{"semantic_effective_mapping_agreement", "strict_mapping_agreement", "whole_typed_attribution_agreement", "principal_agreement", "equal_principal_agreement", "secondary_affected_agreement", "context_only_agreement", "possible_principal_agreement", "principal_proxy_candidate_agreement", "attribution_completeness_agreement", "policy_decision_agreement", "deterministic_resolution_agreement"}
	seen := map[string]bool{}
	for _, metric := range freeze.Metrics {
		if seen[metric.Identity] || strings.TrimSpace(metric.Numerator) == "" || strings.TrimSpace(metric.Denominator) == "" {
			return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("duplicate or incomplete repeatability metric %q", metric.Identity)
		}
		seen[metric.Identity] = true
	}
	for _, metric := range required {
		if !seen[metric] {
			return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("missing repeatability metric %s", metric)
		}
	}
	if len(seen) != len(required) {
		return C1F3RepeatabilityScoringFreeze{}, fmt.Errorf("unexpected repeatability metric definition")
	}
	return freeze, nil
}

type C1F3RepeatabilityMetrics struct {
	SemanticEffectiveMapping C1F3Rate `json:"semantic_effective_mapping_agreement"`
	StrictMapping            C1F3Rate `json:"strict_mapping_agreement"`
	WholeTypedAttribution    C1F3Rate `json:"whole_typed_attribution_agreement"`
	Principal                C1F3Rate `json:"principal_agreement"`
	EqualPrincipal           C1F3Rate `json:"equal_principal_agreement"`
	SecondaryAffected        C1F3Rate `json:"secondary_affected_agreement"`
	ContextOnly              C1F3Rate `json:"context_only_agreement"`
	PossiblePrincipal        C1F3Rate `json:"possible_principal_agreement"`
	PrincipalProxyCandidate  C1F3Rate `json:"principal_proxy_candidate_agreement"`
	AttributionCompleteness  C1F3Rate `json:"attribution_completeness_agreement"`
	PolicyDecision           C1F3Rate `json:"policy_decision_agreement"`
	DeterministicResolution  C1F3Rate `json:"deterministic_resolution_agreement"`
}

type C1F3RepeatabilityChangedCase struct {
	CaseID               string                     `json:"case_id"`
	OriginalMapping      AssetMapping               `json:"original_mapping"`
	RepeatMapping        AssetMapping               `json:"repeat_mapping"`
	OriginalIssuerProxy  string                     `json:"original_issuer_or_proxy"`
	RepeatIssuerProxy    string                     `json:"repeat_issuer_or_proxy"`
	IdentityOutcome      string                     `json:"identity_outcome"`
	TypedRoleDifferences []string                   `json:"typed_role_differences"`
	OriginalPolicy       *CausalAttributionDecision `json:"original_policy"`
	RepeatPolicy         *CausalAttributionDecision `json:"repeat_policy"`
	OriginalResolver     *PolicyResolution          `json:"original_resolver"`
	RepeatResolver       *PolicyResolution          `json:"repeat_resolver"`
	PolicyDiffers        bool                       `json:"policy_differs"`
	ResolverDiffers      bool                       `json:"resolver_differs"`
	AccuracyContext      string                     `json:"accuracy_context"`
}

type C1F3RepeatabilityScore struct {
	Version                           string                         `json:"version"`
	Baseline                          C1F3RepeatabilityBaseline      `json:"baseline"`
	Denominator                       int                            `json:"denominator"`
	Metrics                           C1F3RepeatabilityMetrics       `json:"metrics"`
	ChangedCases                      []C1F3RepeatabilityChangedCase `json:"changed_cases"`
	AccuracyContextCounts             map[string]int                 `json:"accuracy_context_counts"`
	AccuracyRetention                 C1FDualScore                   `json:"accuracy_retention"`
	IncorrectDeterministicResolutions int                            `json:"incorrect_deterministic_ticker_resolutions"`
	SafetyPersistenceViolations       int                            `json:"safety_persistence_violations"`
	Gates                             []C1F3GateResult               `json:"gates"`
	Passed                            bool                           `json:"passed"`
}

func ScoreC1F3Repeatability(labels []TypedExpectedCase, manifest DiagnosticManifest, original, repeat []DiagnosticAttemptAudit, identity IssuerSemanticIdentity, resolver assetresolution.Resolver, safetyViolations int) C1F3RepeatabilityScore {
	denominator := len(labels)
	originalByCase := finalC1FAudits(original)
	repeatByCase := finalC1FAudits(repeat)
	inputs := map[string]EventInput{}
	for _, event := range manifest.Events {
		inputs[event.ID] = event.Input
	}
	counts := map[string]int{}
	accuracyContexts := map[string]int{
		"original wrong / repeat correct": 0,
		"original correct / repeat wrong": 0,
		"both wrong same way":             0,
		"both wrong differently":          0,
		"both correct":                    0,
	}
	changes := []C1F3RepeatabilityChangedCase{}
	incorrectResolutions := 0
	for _, expected := range labels {
		before := originalByCase[expected.CaseID]
		after := repeatByCase[expected.CaseID]
		beforeMapping := auditMapping(before)
		afterMapping := auditMapping(after)
		semanticAgreement, outcome := repeatabilityMappingAgreement(beforeMapping, afterMapping, identity, true)
		if semanticAgreement {
			counts["semantic"]++
		}
		if ok, _ := repeatabilityMappingAgreement(beforeMapping, afterMapping, identity, false); ok {
			counts["strict"]++
		}
		beforeAttrs, beforeCandidates := auditAttribution(before)
		afterAttrs, afterCandidates := auditAttribution(after)
		wholeAttrs, _ := c1f3AttributionMatches(beforeAttrs, afterAttrs, identity, true, nil)
		candidateAgreement := stringSetEqual(beforeCandidates, afterCandidates)
		if wholeAttrs && candidateAgreement {
			counts["whole"]++
		}
		if candidateAgreement {
			counts["candidate"]++
		}
		roleDifferences := []string{}
		for _, role := range []CausalRole{CausalRolePrincipal, CausalRoleEqualPrincipal, CausalRoleSecondaryAffected, CausalRoleContextOnly, CausalRolePossiblePrincipal} {
			agree, _ := c1f3AttributionMatches(filterRole(beforeAttrs, role), filterRole(afterAttrs, role), identity, true, nil)
			key := string(role)
			if agree {
				counts[key]++
			} else {
				roleDifferences = append(roleDifferences, key)
			}
		}
		beforeComplete := attributionSetMatches(expected.ExpectedIssuerAttributions, beforeAttrs, identity, true, false) && stringSetContains(beforeCandidates, expected.ExpectedPrincipalProxyCandidates)
		afterComplete := attributionSetMatches(expected.ExpectedIssuerAttributions, afterAttrs, identity, true, false) && stringSetContains(afterCandidates, expected.ExpectedPrincipalProxyCandidates)
		if beforeComplete == afterComplete {
			counts["completeness"]++
		}
		policyEqual := reflect.DeepEqual(before.CausalAttributionPolicy, after.CausalAttributionPolicy)
		resolverEqual := reflect.DeepEqual(before.DeterministicResolution, after.DeterministicResolution)
		if policyEqual {
			counts["policy"]++
		}
		if resolverEqual {
			counts["resolver"]++
		}
		beforeCorrect := c1fMappingMatches(expected, beforeMapping, identity, true)
		afterCorrect := c1fMappingMatches(expected, afterMapping, identity, true)
		accuracyContext := repeatabilityAccuracyContext(beforeCorrect, afterCorrect, semanticAgreement)
		accuracyContexts[accuracyContext]++
		if !semanticAgreement {
			changes = append(changes, C1F3RepeatabilityChangedCase{
				CaseID: expected.CaseID, OriginalMapping: beforeMapping, RepeatMapping: afterMapping,
				OriginalIssuerProxy: mappingIssuerOrProxy(beforeMapping), RepeatIssuerProxy: mappingIssuerOrProxy(afterMapping), IdentityOutcome: outcome,
				TypedRoleDifferences: roleDifferences, OriginalPolicy: before.CausalAttributionPolicy, RepeatPolicy: after.CausalAttributionPolicy,
				OriginalResolver: before.DeterministicResolution, RepeatResolver: after.DeterministicResolution,
				PolicyDiffers: !policyEqual, ResolverDiffers: !resolverEqual, AccuracyContext: accuracyContext,
			})
		}
		if afterCorrect && afterMapping.MappingStatus != "UNRESOLVED" {
			want, err := c1f3ExpectedResolution(expected, inputs[expected.CaseID], resolver)
			if err != nil || !c1f3ResolutionEqual(want, after.DeterministicResolution) {
				incorrectResolutions++
			}
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].CaseID < changes[j].CaseID })
	metrics := C1F3RepeatabilityMetrics{
		SemanticEffectiveMapping: c1f3Rate(counts["semantic"], denominator), StrictMapping: c1f3Rate(counts["strict"], denominator),
		WholeTypedAttribution: c1f3Rate(counts["whole"], denominator), Principal: c1f3Rate(counts[string(CausalRolePrincipal)], denominator),
		EqualPrincipal: c1f3Rate(counts[string(CausalRoleEqualPrincipal)], denominator), SecondaryAffected: c1f3Rate(counts[string(CausalRoleSecondaryAffected)], denominator),
		ContextOnly: c1f3Rate(counts[string(CausalRoleContextOnly)], denominator), PossiblePrincipal: c1f3Rate(counts[string(CausalRolePossiblePrincipal)], denominator),
		PrincipalProxyCandidate: c1f3Rate(counts["candidate"], denominator), AttributionCompleteness: c1f3Rate(counts["completeness"], denominator),
		PolicyDecision: c1f3Rate(counts["policy"], denominator), DeterministicResolution: c1f3Rate(counts["resolver"], denominator),
	}
	accuracy := ScoreC1FDataset(C1F3ProfileGeneralization, labels, repeat, identity)
	gates := FrozenC1F3QualityGates()
	results := []C1F3GateResult{
		{Gate: "semantic effective-mapping agreement", Observed: fmt.Sprintf("%d / %d / %.2f%%", metrics.SemanticEffectiveMapping.Numerator, denominator, metrics.SemanticEffectiveMapping.Percentage), Required: ">= 90%", Passed: denominator == 48 && metrics.SemanticEffectiveMapping.Percentage >= 90},
		{Gate: "final validity", Observed: fmt.Sprintf("%.2f%%", accuracy.Semantic.FinalValidity.Percentage), Required: ">= 98%", Passed: accuracy.Semantic.FinalValidity.Percentage >= gates.FinalValidity},
		{Gate: "semantic DIRECT precision", Observed: fmt.Sprintf("%.2f%%", accuracy.Semantic.DirectPrecision.Percentage), Required: ">= 95%", Passed: accuracy.Semantic.DirectPrecision.Percentage >= gates.DirectPrecision},
		{Gate: "semantic DIRECT recall", Observed: fmt.Sprintf("%.2f%%", accuracy.Semantic.DirectRecall.Percentage), Required: ">= 90%", Passed: accuracy.Semantic.DirectRecall.Percentage >= gates.DirectRecall},
		{Gate: "semantic false-DIRECT", Observed: fmt.Sprintf("%d / %d / %.2f%%", accuracy.Semantic.FalseDirect.Numerator, denominator, accuracy.Semantic.FalseDirect.Percentage), Required: "<= 5% of 48", Passed: denominator == 48 && accuracy.Semantic.FalseDirect.Percentage <= gates.SemanticFalseDirect},
		{Gate: "incorrect deterministic ticker resolutions", Observed: fmt.Sprintf("%d", incorrectResolutions), Required: "0", Passed: incorrectResolutions == gates.MaximumIncorrectTickerResolutions},
		{Gate: "safety/persistence violations", Observed: fmt.Sprintf("%d", safetyViolations), Required: "0", Passed: safetyViolations == gates.MaximumSafetyViolations},
	}
	passed := true
	for _, gate := range results {
		passed = passed && gate.Passed
	}
	return C1F3RepeatabilityScore{Version: C1F3RepeatabilityScoringVersion, Baseline: frozenC1F3RepeatabilityBaseline(), Denominator: denominator, Metrics: metrics, ChangedCases: changes, AccuracyContextCounts: accuracyContexts, AccuracyRetention: accuracy, IncorrectDeterministicResolutions: incorrectResolutions, SafetyPersistenceViolations: safetyViolations, Gates: results, Passed: passed}
}

func finalC1FAudits(audits []DiagnosticAttemptAudit) map[string]DiagnosticAttemptAudit {
	result := map[string]DiagnosticAttemptAudit{}
	for _, audit := range audits {
		if current, ok := result[audit.CaseID]; !ok || audit.AttemptNumber > current.AttemptNumber {
			result[audit.CaseID] = audit
		}
	}
	return result
}

func auditMapping(audit DiagnosticAttemptAudit) AssetMapping {
	if audit.EffectiveSemanticMapping == nil {
		return AssetMapping{}
	}
	return *audit.EffectiveSemanticMapping
}

func auditAttribution(audit DiagnosticAttemptAudit) ([]IssuerAttribution, []string) {
	if audit.TypedAttribution == nil {
		return []IssuerAttribution{}, []string{}
	}
	return audit.TypedAttribution.IssuerAttributions, audit.TypedAttribution.PrincipalProxyCandidates
}

func repeatabilityMappingAgreement(left, right AssetMapping, identity IssuerSemanticIdentity, semantic bool) (bool, string) {
	if left.MappingStatus != right.MappingStatus {
		return false, "N/A"
	}
	switch left.MappingStatus {
	case "DIRECT":
		if semantic {
			comparison := identity.Compare(left.DirectIssuer, right.DirectIssuer)
			return comparison.Outcome == IssuerIdentityExact || comparison.Outcome == IssuerIdentityEquivalent, string(comparison.Outcome)
		}
		return assetresolution.CanonicalizeIssuerName(left.DirectIssuer) == assetresolution.CanonicalizeIssuerName(right.DirectIssuer), "STRICT"
	case "PROXY":
		return strings.EqualFold(strings.TrimSpace(left.ProxyExposure), strings.TrimSpace(right.ProxyExposure)), "N/A"
	case "UNRESOLVED":
		return true, "N/A"
	default:
		return false, "N/A"
	}
}

func mappingIssuerOrProxy(mapping AssetMapping) string {
	if mapping.MappingStatus == "DIRECT" {
		return mapping.DirectIssuer
	}
	if mapping.MappingStatus == "PROXY" {
		return mapping.ProxyExposure
	}
	return ""
}

func repeatabilityAccuracyContext(originalCorrect, repeatCorrect, sameWay bool) string {
	switch {
	case originalCorrect && repeatCorrect:
		return "both correct"
	case !originalCorrect && repeatCorrect:
		return "original wrong / repeat correct"
	case originalCorrect && !repeatCorrect:
		return "original correct / repeat wrong"
	case sameWay:
		return "both wrong same way"
	default:
		return "both wrong differently"
	}
}
