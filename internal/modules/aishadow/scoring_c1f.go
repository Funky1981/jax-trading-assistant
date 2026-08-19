package aishadow

import (
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

type C1FMetric struct {
	Numerator   int     `json:"numerator"`
	Denominator int     `json:"denominator"`
	Percentage  float64 `json:"percentage"`
}

type C1FMetricLens struct {
	IdentityMode            string    `json:"identity_mode"`
	FinalValidity           C1FMetric `json:"final_schema_validity"`
	WholeMapping            C1FMetric `json:"whole_mapping_correctness"`
	DirectPrecision         C1FMetric `json:"direct_precision"`
	DirectRecall            C1FMetric `json:"direct_recall"`
	ProxyPrecision          C1FMetric `json:"proxy_precision"`
	ProxyRecall             C1FMetric `json:"proxy_recall"`
	UnresolvedCorrectness   C1FMetric `json:"unresolved_correctness"`
	FalseDirect             C1FMetric `json:"false_direct_rate"`
	FalseProxy              C1FMetric `json:"false_proxy_rate"`
	WholeAttribution        C1FMetric `json:"whole_case_attribution_correctness"`
	AttributionCompleteness C1FMetric `json:"attribution_completeness"`
	Principal               C1FMetric `json:"principal_correctness"`
	EqualPrincipal          C1FMetric `json:"equal_principal_correctness"`
	SecondaryAffected       C1FMetric `json:"secondary_affected_correctness"`
	ContextOnly             C1FMetric `json:"context_only_correctness"`
	PossiblePrincipal       C1FMetric `json:"possible_principal_correctness"`
	PrincipalProxyCandidate C1FMetric `json:"principal_proxy_candidate_correctness"`
}

type C1FDualScore struct {
	Version             string        `json:"version"`
	EvidenceDisposition string        `json:"evidence_disposition"`
	Dataset             string        `json:"dataset"`
	Strict              C1FMetricLens `json:"strict_identity"`
	Semantic            C1FMetricLens `json:"semantic_identity_normalized"`
}

func ScoreC1FDataset(dataset string, labels []TypedExpectedCase, audits []DiagnosticAttemptAudit, identity IssuerSemanticIdentity) C1FDualScore {
	byCase := map[string]DiagnosticAttemptAudit{}
	for _, audit := range audits {
		if current, ok := byCase[audit.CaseID]; !ok || audit.AttemptNumber > current.AttemptNumber {
			byCase[audit.CaseID] = audit
		}
	}
	return C1FDualScore{Version: C1FScoringVersion, EvidenceDisposition: "FORENSIC / OFFLINE DEVELOPMENT REPLAY", Dataset: dataset,
		Strict: scoreC1FLens(labels, byCase, identity, false), Semantic: scoreC1FLens(labels, byCase, identity, true)}
}

func scoreC1FLens(labels []TypedExpectedCase, audits map[string]DiagnosticAttemptAudit, identity IssuerSemanticIdentity, semantic bool) C1FMetricLens {
	lens := C1FMetricLens{IdentityMode: "STRICT"}
	if semantic {
		lens.IdentityMode = IssuerSemanticIdentityVersion
	}
	total := len(labels)
	observedDirect, expectedDirect, correctDirect := 0, 0, 0
	observedProxy, expectedProxy, correctProxy := 0, 0, 0
	correctMapping, correctUnresolved, expectedUnresolved, valid := 0, 0, 0, 0
	wholeAttr, completeness := 0, 0
	roleExpected := map[CausalRole]int{}
	roleCorrect := map[CausalRole]int{}
	correctCandidates := 0
	for _, expected := range labels {
		audit := audits[expected.CaseID]
		if audit.ValidationStatus == "accepted" {
			valid++
		}
		mapping := AssetMapping{}
		if audit.EffectiveSemanticMapping != nil {
			mapping = *audit.EffectiveSemanticMapping
		}
		if expected.ExpectedMappingStatus == "DIRECT" {
			expectedDirect++
		}
		if mapping.MappingStatus == "DIRECT" {
			observedDirect++
		}
		if expected.ExpectedMappingStatus == "PROXY" {
			expectedProxy++
		}
		if mapping.MappingStatus == "PROXY" {
			observedProxy++
		}
		if expected.ExpectedMappingStatus == "UNRESOLVED" {
			expectedUnresolved++
		}
		mappingOK := c1fMappingMatches(expected, mapping, identity, semantic)
		if mappingOK {
			correctMapping++
		}
		if mappingOK && expected.ExpectedMappingStatus == "DIRECT" {
			correctDirect++
		}
		if mappingOK && expected.ExpectedMappingStatus == "PROXY" {
			correctProxy++
		}
		if mappingOK && expected.ExpectedMappingStatus == "UNRESOLVED" {
			correctUnresolved++
		}
		observedAttrs := []IssuerAttribution{}
		observedCandidates := []string{}
		if audit.TypedAttribution != nil {
			observedAttrs = audit.TypedAttribution.IssuerAttributions
			observedCandidates = audit.TypedAttribution.PrincipalProxyCandidates
		}
		attrsEqual := attributionSetMatches(expected.ExpectedIssuerAttributions, observedAttrs, identity, semantic, true)
		candidatesEqual := stringSetEqual(expected.ExpectedPrincipalProxyCandidates, observedCandidates)
		if candidatesEqual {
			correctCandidates++
		}
		if attrsEqual && candidatesEqual {
			wholeAttr++
		}
		if attributionSetMatches(expected.ExpectedIssuerAttributions, observedAttrs, identity, semantic, false) && stringSetContains(observedCandidates, expected.ExpectedPrincipalProxyCandidates) {
			completeness++
		}
		for _, role := range []CausalRole{CausalRolePrincipal, CausalRoleEqualPrincipal, CausalRoleSecondaryAffected, CausalRoleContextOnly, CausalRolePossiblePrincipal} {
			want := filterRole(expected.ExpectedIssuerAttributions, role)
			got := filterRole(observedAttrs, role)
			if len(want) == 0 && len(got) == 0 {
				continue
			}
			roleExpected[role]++
			if attributionSetMatches(want, got, identity, semantic, true) {
				roleCorrect[role]++
			}
		}
	}
	lens.FinalValidity = metric(valid, total)
	lens.WholeMapping = metric(correctMapping, total)
	lens.DirectPrecision = metric(correctDirect, observedDirect)
	lens.DirectRecall = metric(correctDirect, expectedDirect)
	lens.ProxyPrecision = metric(correctProxy, observedProxy)
	lens.ProxyRecall = metric(correctProxy, expectedProxy)
	lens.UnresolvedCorrectness = metric(correctUnresolved, expectedUnresolved)
	lens.FalseDirect = metric(observedDirect-correctDirect, total)
	lens.FalseProxy = metric(observedProxy-correctProxy, total)
	lens.WholeAttribution = metric(wholeAttr, total)
	lens.AttributionCompleteness = metric(completeness, total)
	lens.Principal = metric(roleCorrect[CausalRolePrincipal], roleExpected[CausalRolePrincipal])
	lens.EqualPrincipal = metric(roleCorrect[CausalRoleEqualPrincipal], roleExpected[CausalRoleEqualPrincipal])
	lens.SecondaryAffected = metric(roleCorrect[CausalRoleSecondaryAffected], roleExpected[CausalRoleSecondaryAffected])
	lens.ContextOnly = metric(roleCorrect[CausalRoleContextOnly], roleExpected[CausalRoleContextOnly])
	lens.PossiblePrincipal = metric(roleCorrect[CausalRolePossiblePrincipal], roleExpected[CausalRolePossiblePrincipal])
	lens.PrincipalProxyCandidate = metric(correctCandidates, total)
	return lens
}

func c1fMappingMatches(e TypedExpectedCase, observed AssetMapping, identity IssuerSemanticIdentity, semantic bool) bool {
	if e.ExpectedMappingStatus != observed.MappingStatus {
		return false
	}
	switch e.ExpectedMappingStatus {
	case "DIRECT":
		return issuerMatches(e.ExpectedDirectIssuer, observed.DirectIssuer, identity, semantic)
	case "PROXY":
		return strings.EqualFold(e.ExpectedProxyExposure, observed.ProxyExposure)
	case "UNRESOLVED":
		return true
	}
	return false
}

func issuerMatches(left, right string, identity IssuerSemanticIdentity, semantic bool) bool {
	if semantic {
		return identity.Equivalent(left, right)
	}
	return assetresolution.CanonicalizeIssuerName(left) == assetresolution.CanonicalizeIssuerName(right)
}

func attributionSetMatches(expected, observed []IssuerAttribution, identity IssuerSemanticIdentity, semantic, requireEqual bool) bool {
	if requireEqual && len(expected) != len(observed) || len(expected) > len(observed) {
		return false
	}
	used := make([]bool, len(observed))
	for _, want := range expected {
		found := -1
		for idx, got := range observed {
			if !used[idx] && want.CausalRole == got.CausalRole && issuerMatches(want.Issuer, got.Issuer, identity, semantic) {
				found = idx
				break
			}
		}
		if found < 0 {
			return false
		}
		used[found] = true
	}
	return true
}
func filterRole(values []IssuerAttribution, role CausalRole) []IssuerAttribution {
	out := []IssuerAttribution{}
	for _, v := range values {
		if v.CausalRole == role {
			out = append(out, v)
		}
	}
	return out
}
func stringSetEqual(a, b []string) bool { return len(a) == len(b) && stringSetContains(a, b) }
func stringSetContains(container, wanted []string) bool {
	a := append([]string(nil), container...)
	b := append([]string(nil), wanted...)
	for i := range a {
		a[i] = strings.ToUpper(strings.TrimSpace(a[i]))
	}
	for i := range b {
		b[i] = strings.ToUpper(strings.TrimSpace(b[i]))
	}
	sort.Strings(a)
	sort.Strings(b)
	j := 0
	for _, v := range a {
		if j < len(b) && v == b[j] {
			j++
		}
	}
	return j == len(b)
}
func metric(n, d int) C1FMetric {
	p := 0.0
	if d > 0 {
		p = float64(n) * 100 / float64(d)
	}
	return C1FMetric{n, d, p}
}
