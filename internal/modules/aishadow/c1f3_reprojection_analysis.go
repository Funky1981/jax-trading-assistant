package aishadow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func analyzeC1F3Dataset(dataset *C1F3DatasetReprojection, labels []TypedExpectedCase, manifest DiagnosticManifest, audits []DiagnosticAttemptAudit, identity IssuerSemanticIdentity, resolver assetresolution.Resolver) {
	final := map[string]DiagnosticAttemptAudit{}
	inputs := map[string]EventInput{}
	for _, event := range manifest.Events {
		inputs[event.ID] = event.Input
	}
	for _, audit := range audits {
		if current, ok := final[audit.CaseID]; !ok || audit.AttemptNumber > current.AttemptNumber {
			final[audit.CaseID] = audit
		}
	}
	identityCounts := map[IssuerIdentityOutcome]int{}
	mappingSuccess, mappingEquivalent := 0, 0
	attributionSuccess, attributionEquivalent := 0, 0
	acceptedCorrect, acceptedDenominator := 0, 0
	abstainedCorrect, abstainedDenominator := 0, 0
	policyFalsePositive, expectedUnresolved := 0, 0
	policyFalseNegative, rawSelectableCorrect := 0, 0
	resolverCorrect, resolverDenominator := 0, 0

	for _, expected := range labels {
		audit := final[expected.CaseID]
		mapping := AssetMapping{}
		if audit.EffectiveSemanticMapping != nil {
			mapping = *audit.EffectiveSemanticMapping
		}
		mappingOK := c1fMappingMatches(expected, mapping, identity, true)
		mappingIdentity := "N/A"
		mappingDependsOnEquivalent := false
		if expected.ExpectedMappingStatus == "DIRECT" && mapping.MappingStatus == "DIRECT" {
			comparison := identity.Compare(expected.ExpectedDirectIssuer, mapping.DirectIssuer)
			identityCounts[comparison.Outcome]++
			mappingIdentity = string(comparison.Outcome)
			mappingDependsOnEquivalent = comparison.Outcome == IssuerIdentityEquivalent
		}
		if mappingOK {
			mappingSuccess++
			if mappingDependsOnEquivalent {
				mappingEquivalent++
			}
		}

		observedAttributions := []IssuerAttribution{}
		observedCandidates := []string{}
		if audit.TypedAttribution != nil {
			observedAttributions = audit.TypedAttribution.IssuerAttributions
			observedCandidates = audit.TypedAttribution.PrincipalProxyCandidates
		}
		wholeAttributionOK, attributionDependsOnEquivalent := c1f3AttributionMatches(expected.ExpectedIssuerAttributions, observedAttributions, identity, true, nil)
		wholeAttributionOK = wholeAttributionOK && stringSetEqual(expected.ExpectedPrincipalProxyCandidates, observedCandidates)
		complete, _ := c1f3AttributionMatches(expected.ExpectedIssuerAttributions, observedAttributions, identity, false, nil)
		complete = complete && stringSetContains(observedCandidates, expected.ExpectedPrincipalProxyCandidates)
		if wholeAttributionOK {
			attributionSuccess++
			if attributionDependsOnEquivalent {
				attributionEquivalent++
			}
		}

		// Count exactly the semantic issuer comparisons made by the frozen scorer:
		// whole attribution, completeness, and each populated role class.
		_, _ = c1f3AttributionMatches(expected.ExpectedIssuerAttributions, observedAttributions, identity, true, identityCounts)
		_, _ = c1f3AttributionMatches(expected.ExpectedIssuerAttributions, observedAttributions, identity, false, identityCounts)
		for _, role := range []CausalRole{CausalRolePrincipal, CausalRoleEqualPrincipal, CausalRoleSecondaryAffected, CausalRoleContextOnly, CausalRolePossiblePrincipal} {
			wanted := filterRole(expected.ExpectedIssuerAttributions, role)
			observed := filterRole(observedAttributions, role)
			if len(wanted) > 0 || len(observed) > 0 {
				_, _ = c1f3AttributionMatches(wanted, observed, identity, true, identityCounts)
			}
		}

		rawMapping := AssetMapping{}
		if audit.V5RawModelOutput != nil {
			rawMapping = AssetMapping{MappingStatus: audit.V5RawModelOutput.MappingStatus, DirectIssuer: audit.V5RawModelOutput.DirectIssuer, ProxyExposure: audit.V5RawModelOutput.ProxyExposure}
		}
		rawOK := c1fMappingMatches(expected, rawMapping, identity, true)
		policyEffect := c1f3PolicyEffect(rawOK, mappingOK)
		switch policyEffect {
		case "RAW correct -> policy preserved":
			dataset.Policy.RawCorrectPolicyPreserved++
		case "RAW wrong -> policy fixed":
			dataset.Policy.RawWrongPolicyFixed++
		case "RAW correct -> policy broke":
			dataset.Policy.RawCorrectPolicyBroke++
		case "RAW wrong -> policy failed to fix":
			dataset.Policy.RawWrongPolicyFailedToFix++
		}
		if expected.ExpectedMappingStatus == "UNRESOLVED" {
			expectedUnresolved++
		}
		if audit.CausalAttributionPolicy != nil {
			if audit.CausalAttributionPolicy.Accepted {
				dataset.Policy.Accepted++
				acceptedDenominator++
				if expected.ExpectedMappingStatus != "UNRESOLVED" && mappingOK {
					acceptedCorrect++
				}
			}
			if audit.CausalAttributionPolicy.Abstained {
				dataset.Policy.Abstained++
				abstainedDenominator++
				if expected.ExpectedMappingStatus == "UNRESOLVED" && mappingOK {
					abstainedCorrect++
				}
			}
		}
		if expected.ExpectedMappingStatus == "UNRESOLVED" && (mapping.MappingStatus == "DIRECT" || mapping.MappingStatus == "PROXY") {
			policyFalsePositive++
		}
		if rawOK && (rawMapping.MappingStatus == "DIRECT" || rawMapping.MappingStatus == "PROXY") {
			rawSelectableCorrect++
			if mapping.MappingStatus == "UNRESOLVED" {
				policyFalseNegative++
			}
		}

		resolverOK := false
		if mappingOK {
			if mapping.MappingStatus == "UNRESOLVED" {
				resolverOK = audit.DeterministicResolution != nil && audit.DeterministicResolution.Status == expected.ExpectedDeterministicResolutionStatus && audit.DeterministicResolution.ResolvedTicker == ""
			} else {
				resolverDenominator++
				expectedResolution, resolutionErr := c1f3ExpectedResolution(expected, inputs[expected.CaseID], resolver)
				resolverOK = resolutionErr == nil && c1f3ResolutionEqual(expectedResolution, audit.DeterministicResolution)
				if resolverOK {
					resolverCorrect++
				} else {
					dataset.Resolver.IncorrectCount++
					dataset.Resolver.IncorrectCaseIDs = append(dataset.Resolver.IncorrectCaseIDs, expected.CaseID)
				}
			}
		}

		issues := c1f3CaseIssues(expected, observedAttributions, observedCandidates, mapping, identity)
		c1f3CollectIssues(&dataset.ProxyRole, expected.CaseID, expected, mapping, issues)
		failure := C1F3FailureRecord{
			CaseID: expected.CaseID, Category: audit.Category,
			ExpectedMapping: AssetMapping{MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure},
			RawResult:       audit.V5RawModelOutput, IdentityOutcome: mappingIdentity, RoleProxyIssues: issues, PolicyEffect: policyEffect,
			EffectiveMapping: audit.EffectiveSemanticMapping, ResolverResult: audit.DeterministicResolution,
			SemanticMappingCorrect: mappingOK, WholeAttributionCorrect: wholeAttributionOK, AttributionComplete: complete, ResolverCorrect: resolverOK,
			RootCauseClassification: c1f3RootCauses(expected, mapping, mappingOK, wholeAttributionOK, complete, resolverOK, issues, mappingIdentity),
		}
		if !mappingOK {
			dataset.MappingFailures = append(dataset.MappingFailures, failure)
		}
		if !wholeAttributionOK || !complete {
			dataset.TypedFailures = append(dataset.TypedFailures, failure)
		}
		if !mappingOK || !wholeAttributionOK || !resolverOK {
			dataset.AllFailures = append(dataset.AllFailures, failure)
		}
		dataset.TickerFindings = append(dataset.TickerFindings, c1f3TickerFindings(audit, resolver)...)
	}

	dataset.Identity = C1F3IdentityAnalysis{
		Exact: identityCounts[IssuerIdentityExact], Equivalent: identityCounts[IssuerIdentityEquivalent],
		Distinct: identityCounts[IssuerIdentityDistinct], Ambiguous: identityCounts[IssuerIdentityAmbiguous],
		MappingEquivalentDependent: c1f3Rate(mappingEquivalent, mappingSuccess), AttributionEquivalentDependent: c1f3Rate(attributionEquivalent, attributionSuccess),
	}
	dataset.Identity.TotalComparisons = dataset.Identity.Exact + dataset.Identity.Equivalent + dataset.Identity.Distinct + dataset.Identity.Ambiguous
	dataset.Policy.AcceptanceCorrectness = c1f3Rate(acceptedCorrect, acceptedDenominator)
	dataset.Policy.AbstentionCorrectness = c1f3Rate(abstainedCorrect, abstainedDenominator)
	dataset.Policy.PolicyInducedFalsePositive = c1f3Rate(policyFalsePositive, expectedUnresolved)
	dataset.Policy.PolicyInducedFalseNegative = c1f3Rate(policyFalseNegative, rawSelectableCorrect)
	dataset.Resolver.Correctness = c1f3Rate(resolverCorrect, resolverDenominator)
	dataset.Resolver.IncorrectCaseIDs = uniqueSortedStrings(dataset.Resolver.IncorrectCaseIDs)
	c1f3NormalizeProxyRoleAnalysis(&dataset.ProxyRole)
	sort.Slice(dataset.TickerFindings, func(i, j int) bool {
		if dataset.TickerFindings[i].CaseID == dataset.TickerFindings[j].CaseID {
			return dataset.TickerFindings[i].Finding < dataset.TickerFindings[j].Finding
		}
		return dataset.TickerFindings[i].CaseID < dataset.TickerFindings[j].CaseID
	})
}

func c1f3AttributionMatches(expected, observed []IssuerAttribution, identity IssuerSemanticIdentity, requireEqual bool, counts map[IssuerIdentityOutcome]int) (bool, bool) {
	if requireEqual && len(expected) != len(observed) || len(expected) > len(observed) {
		return false, false
	}
	used := make([]bool, len(observed))
	equivalent := false
	for _, wanted := range expected {
		found := -1
		for index, got := range observed {
			if used[index] || wanted.CausalRole != got.CausalRole {
				continue
			}
			comparison := identity.Compare(wanted.Issuer, got.Issuer)
			if counts != nil {
				counts[comparison.Outcome]++
			}
			if comparison.Outcome == IssuerIdentityExact || comparison.Outcome == IssuerIdentityEquivalent {
				found = index
				if comparison.Outcome == IssuerIdentityEquivalent {
					equivalent = true
				}
				break
			}
		}
		if found < 0 {
			return false, equivalent
		}
		used[found] = true
	}
	return true, equivalent
}

func c1f3PolicyEffect(rawCorrect, effectiveCorrect bool) string {
	switch {
	case rawCorrect && effectiveCorrect:
		return "RAW correct -> policy preserved"
	case !rawCorrect && effectiveCorrect:
		return "RAW wrong -> policy fixed"
	case rawCorrect && !effectiveCorrect:
		return "RAW correct -> policy broke"
	default:
		return "RAW wrong -> policy failed to fix"
	}
}

func c1f3ExpectedResolution(expected TypedExpectedCase, input EventInput, resolver assetresolution.Resolver) (*PolicyResolution, error) {
	projection := V5StructuredResult{
		MappingStatus: expected.ExpectedMappingStatus, DirectIssuer: expected.ExpectedDirectIssuer, ProxyExposure: expected.ExpectedProxyExposure,
		IssuerAttributions: expected.ExpectedIssuerAttributions, PrincipalProxyCandidates: expected.ExpectedPrincipalProxyCandidates,
	}
	decision, err := ApplyCausalAttributionPolicy(projection)
	if err != nil {
		return nil, err
	}
	resolution := ResolveCausalAttributionDecision(decision, input, resolver)
	return &resolution, nil
}

func c1f3ResolutionEqual(expected, observed *PolicyResolution) bool {
	if expected == nil || observed == nil || expected.Status != observed.Status {
		return false
	}
	if expected.Status != "resolved" {
		return observed.ResolvedTicker == ""
	}
	return expected.ResolvedTicker == observed.ResolvedTicker && expected.MatchedRule == observed.MatchedRule
}

func c1f3CaseIssues(expected TypedExpectedCase, observed []IssuerAttribution, candidates []string, mapping AssetMapping, identity IssuerSemanticIdentity) []string {
	issues := []string{}
	if expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus != "PROXY" {
		issues = append(issues, "missed proxy")
	}
	if expected.ExpectedMappingStatus != "PROXY" && mapping.MappingStatus == "PROXY" {
		issues = append(issues, "false proxy")
	}
	if expected.ExpectedMappingStatus == "DIRECT" && mapping.MappingStatus == "PROXY" || expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus == "DIRECT" {
		issues = append(issues, "entity/exposure confusion")
	}
	if expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus == "PROXY" && !strings.EqualFold(expected.ExpectedProxyExposure, mapping.ProxyExposure) {
		issues = append(issues, "nearest-topic fallback / wrong proxy exposure")
	}
	for _, wanted := range expected.ExpectedIssuerAttributions {
		matched := false
		for _, got := range observed {
			if wanted.CausalRole == got.CausalRole && identity.Equivalent(wanted.Issuer, got.Issuer) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if role, ok := c1f3ObservedRole(observed, wanted.Issuer, identity); ok {
			issues = append(issues, fmt.Sprintf("role reversal %s: expected %s observed %s", wanted.Issuer, wanted.CausalRole, role))
		} else {
			issues = append(issues, fmt.Sprintf("attribution omission %s:%s", wanted.Issuer, wanted.CausalRole))
		}
	}
	for _, got := range observed {
		matched := false
		for _, wanted := range expected.ExpectedIssuerAttributions {
			if wanted.CausalRole == got.CausalRole && identity.Equivalent(wanted.Issuer, got.Issuer) {
				matched = true
				break
			}
		}
		if !matched {
			if _, ok := c1f3ObservedRole(expected.ExpectedIssuerAttributions, got.Issuer, identity); !ok {
				issues = append(issues, fmt.Sprintf("unexpected attribution %s:%s", got.Issuer, got.CausalRole))
			}
		}
	}
	if len(filterRole(expected.ExpectedIssuerAttributions, CausalRolePossiblePrincipal)) != len(filterRole(observed, CausalRolePossiblePrincipal)) {
		issues = append(issues, "POSSIBLE_PRINCIPAL misuse")
	}
	if !stringSetEqual(expected.ExpectedPrincipalProxyCandidates, candidates) {
		issues = append(issues, "principal proxy candidate mismatch")
	}
	return uniqueSortedStrings(issues)
}

func c1f3ObservedRole(values []IssuerAttribution, issuer string, identity IssuerSemanticIdentity) (CausalRole, bool) {
	for _, value := range values {
		if identity.Equivalent(issuer, value.Issuer) {
			return value.CausalRole, true
		}
	}
	return "", false
}

func c1f3RootCauses(expected TypedExpectedCase, mapping AssetMapping, mappingOK, attributionOK, complete, resolverOK bool, issues []string, identityOutcome string) []string {
	causes := []string{}
	if !mappingOK {
		switch {
		case expected.ExpectedMappingStatus == "DIRECT" && mapping.MappingStatus == "UNRESOLVED":
			causes = append(causes, "missed direct issuer / over-abstention")
		case expected.ExpectedMappingStatus == "UNRESOLVED" && mapping.MappingStatus == "DIRECT":
			causes = append(causes, "false DIRECT causal-boundary error")
		case expected.ExpectedMappingStatus == "UNRESOLVED" && mapping.MappingStatus == "PROXY":
			causes = append(causes, "false PROXY fallback")
		case expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus == "UNRESOLVED":
			causes = append(causes, "missed principal proxy")
		case expected.ExpectedMappingStatus == "DIRECT" && mapping.MappingStatus == "PROXY" || expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus == "DIRECT":
			causes = append(causes, "entity/exposure confusion")
		case expected.ExpectedMappingStatus == "DIRECT" && mapping.MappingStatus == "DIRECT":
			causes = append(causes, "issuer identity mismatch: "+identityOutcome)
		case expected.ExpectedMappingStatus == "PROXY" && mapping.MappingStatus == "PROXY":
			causes = append(causes, "wrong bounded proxy exposure")
		default:
			causes = append(causes, "mapping class mismatch")
		}
	}
	if !attributionOK {
		causes = append(causes, "typed causal-attribution mismatch")
	}
	if !complete {
		causes = append(causes, "attribution omission")
	}
	if mappingOK && !resolverOK {
		causes = append(causes, "deterministic resolver mismatch")
	}
	for _, issue := range issues {
		if strings.Contains(issue, "role reversal") {
			causes = append(causes, "causal role reversal")
		}
		if strings.Contains(issue, "POSSIBLE_PRINCIPAL") {
			causes = append(causes, "POSSIBLE_PRINCIPAL misuse")
		}
		if strings.Contains(issue, "proxy candidate") {
			causes = append(causes, "principal proxy candidate error")
		}
		if strings.Contains(issue, "unexpected attribution") {
			causes = append(causes, "unexpected attribution")
		}
	}
	return uniqueSortedStrings(causes)
}

func c1f3CollectIssues(summary *C1F3ProxyRoleAnalysis, caseID string, expected TypedExpectedCase, mapping AssetMapping, issues []string) {
	for _, issue := range issues {
		entry := caseID + ": " + issue
		switch {
		case issue == "false proxy":
			summary.FalseProxies = append(summary.FalseProxies, entry)
		case issue == "missed proxy":
			summary.MissedProxies = append(summary.MissedProxies, entry)
		case strings.Contains(issue, "nearest-topic"):
			summary.NearestTopicFallbackErrors = append(summary.NearestTopicFallbackErrors, entry)
		case strings.Contains(issue, "entity/exposure"):
			summary.EntityExposureConfusion = append(summary.EntityExposureConfusion, entry)
		case strings.Contains(issue, "role reversal"):
			summary.RoleReversals = append(summary.RoleReversals, entry)
		case strings.Contains(issue, "POSSIBLE_PRINCIPAL"):
			summary.PossiblePrincipalMisuse = append(summary.PossiblePrincipalMisuse, entry)
		case strings.Contains(issue, "omission"):
			summary.AttributionOmissions = append(summary.AttributionOmissions, entry)
		case strings.Contains(issue, "unexpected attribution"):
			summary.UnexpectedAttributions = append(summary.UnexpectedAttributions, entry)
		}
	}
	if strings.Contains(expected.DatasetIdentity, "nearest-topic") && expected.ExpectedMappingStatus != mapping.MappingStatus {
		summary.NearestTopicFallbackErrors = append(summary.NearestTopicFallbackErrors, caseID+": category-level nearest-topic failure")
	}
}

func c1f3NormalizeProxyRoleAnalysis(summary *C1F3ProxyRoleAnalysis) {
	summary.FalseProxies = uniqueSortedStrings(summary.FalseProxies)
	summary.MissedProxies = uniqueSortedStrings(summary.MissedProxies)
	summary.NearestTopicFallbackErrors = uniqueSortedStrings(summary.NearestTopicFallbackErrors)
	summary.EntityExposureConfusion = uniqueSortedStrings(summary.EntityExposureConfusion)
	summary.RoleReversals = uniqueSortedStrings(summary.RoleReversals)
	summary.PossiblePrincipalMisuse = uniqueSortedStrings(summary.PossiblePrincipalMisuse)
	summary.AttributionOmissions = uniqueSortedStrings(summary.AttributionOmissions)
	summary.UnexpectedAttributions = uniqueSortedStrings(summary.UnexpectedAttributions)
}

func c1f3TickerFindings(audit DiagnosticAttemptAudit, resolver assetresolution.Resolver) []C1F3TickerFinding {
	if audit.V5RawModelOutput == nil {
		return nil
	}
	text := strings.Join(append([]string{audit.V5RawModelOutput.Reason, audit.V5RawModelOutput.CatalystType}, audit.V5RawModelOutput.MissingEvidence...), " ")
	findings := []C1F3TickerFinding{}
	for _, rule := range resolver.Rules.Aliases {
		symbol := strings.ToUpper(strings.TrimSpace(rule.Symbol))
		if symbol == "" {
			continue
		}
		pattern := regexp.MustCompile(`(?i)(^|[^A-Za-z0-9])` + regexp.QuoteMeta(symbol) + `([^A-Za-z0-9]|$)`)
		if pattern.MatchString(text) {
			findings = append(findings, C1F3TickerFinding{CaseID: audit.CaseID, Finding: "ticker token in model free text: " + symbol})
		}
	}
	return findings
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
