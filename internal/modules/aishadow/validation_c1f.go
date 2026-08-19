package aishadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const C1FValidatorVersion = "ai-shadow-causal-attribution-validator-c1f-v1"

// ParseValidateAndApplyC1F preserves every v5 causal invariant while allowing
// a valid UNRESOLVED classification to carry HIGH, MEDIUM, or LOW confidence.
func ParseValidateAndApplyC1F(raw string, input EventInput, resolver assetresolution.Resolver) (*V5StructuredResult, *CausalAttributionDecision, *PolicyResolution, []string) {
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
	errors = append(errors, validateC1FScalarFields(result, input, resolver)...)
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

func validateC1FScalarFields(result V5StructuredResult, input EventInput, resolver assetresolution.Resolver) []string {
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
	}
	return errors
}
