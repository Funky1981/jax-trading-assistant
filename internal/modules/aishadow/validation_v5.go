package aishadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"jax-trading-assistant/internal/modules/assetresolution"
)

// ParseValidateAndApplyV5 validates the raw v5 output, applies only the typed
// C1E policy, and invokes the unchanged resolver only for a valid projection.
func ParseValidateAndApplyV5(raw string, input EventInput, resolver assetresolution.Resolver) (*V5StructuredResult, *CausalAttributionDecision, *PolicyResolution, []string) {
	errors := []string{}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, nil, nil, []string{"invalid JSON: " + err.Error()}
	}
	for _, field := range requiredV5ResultFields {
		if _, ok := fields[field]; !ok {
			errors = append(errors, "missing required field: "+field)
		}
	}
	for field := range fields {
		if !contains(requiredV5ResultFields, field) {
			errors = append(errors, "unknown field: "+field)
		}
	}
	for _, field := range []string{"direct_issuer", "proxy_exposure", "issuer_attributions", "principal_proxy_candidates"} {
		if rawValue, ok := fields[field]; ok && bytes.Equal(bytes.TrimSpace(rawValue), []byte("null")) {
			errors = append(errors, field+" must not be null")
		}
	}

	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var result V5StructuredResult
	if err := decoder.Decode(&result); err != nil {
		errors = append(errors, "schema decode: "+err.Error())
		return nil, nil, nil, uniqueSorted(errors)
	}
	if err := ensureEOF(decoder); err != nil {
		errors = append(errors, err.Error())
	}

	errors = append(errors, validateV5ScalarFields(result, input, resolver)...)
	errors = append(errors, validateV5Attribution(result, resolver)...)
	if len(errors) > 0 {
		return nil, nil, nil, uniqueSorted(errors)
	}
	decision, err := ApplyCausalAttributionPolicy(result)
	if err != nil {
		return nil, nil, nil, []string{err.Error()}
	}
	resolution := ResolveCausalAttributionDecision(decision, input, resolver)
	return &result, &decision, &resolution, nil
}

func validateV5ScalarFields(result V5StructuredResult, input EventInput, resolver assetresolution.Resolver) []string {
	errors := []string{}
	if !contains([]string{"HIGH", "MEDIUM", "LOW", "UNCERTAIN"}, result.MarketRelevance) {
		errors = append(errors, "market_relevance must be HIGH, MEDIUM, LOW, or UNCERTAIN")
	}
	if !contains([]string{"DIRECT", "PROXY", "UNRESOLVED"}, result.MappingStatus) {
		errors = append(errors, "mapping_status must be DIRECT, PROXY, or UNRESOLVED")
	}
	if !contains([]string{"HIGH", "MEDIUM", "LOW"}, result.MappingConfidence) {
		errors = append(errors, "mapping_confidence must be HIGH, MEDIUM, or LOW")
	}
	if !contains([]string{"INTRADAY", "ONE_DAY", "MULTI_DAY", "UNCLEAR"}, result.ExpectedHorizon) {
		errors = append(errors, "expected_horizon has an invalid value")
	}
	if !contains([]string{"POSITIVE", "NEGATIVE", "NEUTRAL", "UNCLEAR"}, result.LikelyDirection) {
		errors = append(errors, "likely_direction has an invalid value")
	}
	issuerLength := utf8.RuneCountInString(strings.TrimSpace(result.DirectIssuer))
	if issuerLength > 200 {
		errors = append(errors, "direct_issuer must contain at most 200 characters")
	}
	reasonLength := utf8.RuneCountInString(strings.TrimSpace(result.Reason))
	if reasonLength < 20 || reasonLength > 400 {
		errors = append(errors, "reason must contain 20 to 400 characters")
	}
	catalystLength := utf8.RuneCountInString(strings.TrimSpace(result.CatalystType))
	if catalystLength < 1 || catalystLength > 80 {
		errors = append(errors, "catalyst_type must contain 1 to 80 characters")
	}
	if result.MissingEvidence == nil {
		errors = append(errors, "missing_evidence must be an array")
	} else if len(result.MissingEvidence) > 10 {
		errors = append(errors, "missing_evidence must contain at most 10 items")
	}
	for index, value := range result.MissingEvidence {
		length := utf8.RuneCountInString(strings.TrimSpace(value))
		if length < 1 || length > 160 {
			errors = append(errors, fmt.Sprintf("missing_evidence[%d] must contain 1 to 160 characters", index))
		}
	}

	switch result.MappingStatus {
	case "DIRECT":
		if issuerLength == 0 {
			errors = append(errors, "DIRECT mapping requires a non-empty direct_issuer")
		}
		if result.ProxyExposure != NoProxyExposure {
			errors = append(errors, "DIRECT mapping requires proxy_exposure NONE")
		}
		if result.MappingConfidence == "LOW" {
			errors = append(errors, "DIRECT mapping requires HIGH or MEDIUM mapping_confidence")
		}
		if input.PublicationTimestamp.IsZero() || input.ReceiptTimestamp.IsZero() || input.PublicationTimestamp.After(input.ReceiptTimestamp) {
			errors = append(errors, "issuer resolution requires valid receipt-time publication and receipt anchors")
		}
	case "PROXY":
		if result.DirectIssuer != "" {
			errors = append(errors, "PROXY mapping requires an empty direct_issuer")
		}
		if result.ProxyExposure == NoProxyExposure {
			errors = append(errors, "PROXY mapping requires a bounded proxy_exposure")
		} else if !isAllowedProxyExposure(result.ProxyExposure, resolver) {
			errors = append(errors, "proxy_exposure is not allowlisted by the active Jax policy")
		}
	case "UNRESOLVED":
		if result.DirectIssuer != "" {
			errors = append(errors, "UNRESOLVED mapping requires an empty direct_issuer")
		}
		if result.ProxyExposure != NoProxyExposure {
			errors = append(errors, "UNRESOLVED mapping requires proxy_exposure NONE")
		}
		if result.MappingConfidence != "LOW" {
			errors = append(errors, "UNRESOLVED mapping requires LOW mapping_confidence")
		}
	}
	return errors
}

func validateV5Attribution(result V5StructuredResult, resolver assetresolution.Resolver) []string {
	errors := []string{}
	if result.IssuerAttributions == nil {
		errors = append(errors, "issuer_attributions must be an array")
	} else if len(result.IssuerAttributions) > 10 {
		errors = append(errors, "issuer_attributions must contain at most 10 items")
	}
	seenIssuers := map[string]bool{}
	for index, attribution := range result.IssuerAttributions {
		issuer := strings.TrimSpace(attribution.Issuer)
		length := utf8.RuneCountInString(issuer)
		if length < 1 || length > 200 {
			errors = append(errors, fmt.Sprintf("issuer_attributions[%d].issuer must contain 1 to 200 characters", index))
		}
		normalized := assetresolution.CanonicalizeIssuerName(issuer)
		if normalized != "" && seenIssuers[normalized] {
			errors = append(errors, "issuer identities must be unique after deterministic normalization")
		}
		seenIssuers[normalized] = true
		if !contains([]string{
			string(CausalRolePrincipal), string(CausalRoleEqualPrincipal), string(CausalRoleSecondaryAffected),
			string(CausalRoleContextOnly), string(CausalRolePossiblePrincipal),
		}, string(attribution.CausalRole)) {
			errors = append(errors, fmt.Sprintf("issuer_attributions[%d].causal_role has an invalid value", index))
		}
	}
	if result.PrincipalProxyCandidates == nil {
		errors = append(errors, "principal_proxy_candidates must be an array")
	}
	seenProxies := map[string]bool{}
	for index, candidate := range result.PrincipalProxyCandidates {
		normalized := strings.ToUpper(strings.TrimSpace(candidate))
		if normalized == NoProxyExposure || !isAllowedProxyExposure(candidate, resolver) {
			errors = append(errors, fmt.Sprintf("principal_proxy_candidates[%d] is not an active bounded exposure", index))
		}
		if seenProxies[normalized] {
			errors = append(errors, "principal_proxy_candidates must be unique")
		}
		seenProxies[normalized] = true
	}
	if len(result.PrincipalProxyCandidates) > len(resolver.Rules.Proxies) {
		errors = append(errors, "principal_proxy_candidates exceeds the active bounded exposure count")
	}

	principalCount, equalCount, possibleCount := causalRoleCounts(result.IssuerAttributions)
	if principalCount > 1 {
		errors = append(errors, "at most one PRINCIPAL attribution is permitted")
	}
	if equalCount == 1 {
		errors = append(errors, "EQUAL_PRINCIPAL requires at least two issuer attributions")
	}
	if principalCount > 0 && (equalCount > 0 || possibleCount > 0) || equalCount > 0 && possibleCount > 0 {
		errors = append(errors, "PRINCIPAL, EQUAL_PRINCIPAL, and POSSIBLE_PRINCIPAL are mutually exclusive role classes")
	}
	if (principalCount > 0 || equalCount > 0 || possibleCount > 0) && len(result.PrincipalProxyCandidates) > 0 {
		errors = append(errors, "principal issuer roles require empty principal_proxy_candidates")
	}

	switch result.MappingStatus {
	case "DIRECT":
		if principalCount != 1 || equalCount != 0 || possibleCount != 0 || len(result.PrincipalProxyCandidates) != 0 {
			errors = append(errors, "DIRECT requires exactly one PRINCIPAL and no equal, possible, or proxy candidates")
		}
		if principalCount == 1 && strings.TrimSpace(result.DirectIssuer) != principalIssuer(result.IssuerAttributions) {
			errors = append(errors, "direct_issuer must exactly match the trimmed PRINCIPAL issuer identity")
		}
	case "PROXY":
		if principalCount != 0 || equalCount != 0 || possibleCount != 0 || len(result.PrincipalProxyCandidates) != 1 {
			errors = append(errors, "PROXY requires no principal issuer roles and exactly one principal proxy candidate")
		}
		if len(result.PrincipalProxyCandidates) == 1 && result.ProxyExposure != result.PrincipalProxyCandidates[0] {
			errors = append(errors, "proxy_exposure must exactly match the principal proxy candidate")
		}
	case "UNRESOLVED":
		validState := principalCount == 0 && ((equalCount >= 2 && possibleCount == 0 && len(result.PrincipalProxyCandidates) == 0) ||
			(equalCount == 0 && possibleCount > 0 && len(result.PrincipalProxyCandidates) == 0) ||
			(equalCount == 0 && possibleCount == 0 && len(result.PrincipalProxyCandidates) != 1))
		if !validState {
			errors = append(errors, "UNRESOLVED requires an equal/possible issuer ambiguity, no candidate, or multiple proxy candidates")
		}
	}
	return errors
}
