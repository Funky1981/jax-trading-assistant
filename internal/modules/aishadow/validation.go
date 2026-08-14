package aishadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"jax-trading-assistant/internal/modules/assetresolution"
)

var requiredResultFields = []string{
	"market_relevance", "mapping_status", "direct_issuer", "proxy_exposure", "mapping_confidence", "expected_horizon",
	"likely_direction", "catalyst_type", "reason", "missing_evidence",
}

func ParseAndValidate(raw string, input EventInput, resolver assetresolution.Resolver) (*StructuredResult, *PolicyResolution, []string) {
	result, _, resolution, errors := ParseValidateAndGuard(raw, input, resolver)
	return result, resolution, errors
}

// ParseValidateAndGuard validates the canonical provider output, applies the
// causal-consistency policy, and only then invokes deterministic resolution.
func ParseValidateAndGuard(raw string, input EventInput, resolver assetresolution.Resolver) (*StructuredResult, *CausalConsistencyDecision, *PolicyResolution, []string) {
	errors := []string{}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, nil, nil, []string{"invalid JSON: " + err.Error()}
	}
	for _, field := range requiredResultFields {
		if _, ok := fields[field]; !ok {
			errors = append(errors, "missing required field: "+field)
		}
	}
	for field := range fields {
		if !contains(requiredResultFields, field) {
			errors = append(errors, "unknown field: "+field)
		}
	}
	for _, field := range []string{"direct_issuer", "proxy_exposure"} {
		if rawValue, ok := fields[field]; ok && bytes.Equal(bytes.TrimSpace(rawValue), []byte("null")) {
			errors = append(errors, field+" must be a string, not null")
		}
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var result StructuredResult
	if err := decoder.Decode(&result); err != nil {
		errors = append(errors, "schema decode: "+err.Error())
		return nil, nil, nil, uniqueSorted(errors)
	}
	if err := ensureEOF(decoder); err != nil {
		errors = append(errors, err.Error())
	}
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
	if len(errors) > 0 {
		return nil, nil, nil, uniqueSorted(errors)
	}
	guard := ApplyCausalConsistencyGuard(result, input, resolver)
	resolution := resolveEffectiveMapping(guard, input, resolver)
	return &result, &guard, &resolution, nil
}

func isAllowedProxyExposure(exposure string, resolver assetresolution.Resolver) bool {
	for _, rule := range resolver.Rules.Proxies {
		if strings.EqualFold(strings.TrimSpace(rule.Key), strings.TrimSpace(exposure)) {
			return true
		}
	}
	return false
}

func resolveEffectiveMapping(guard CausalConsistencyDecision, input EventInput, resolver assetresolution.Resolver) PolicyResolution {
	effective := guard.EffectiveMapping
	switch effective.MappingStatus {
	case "DIRECT":
		resolved := resolver.ResolveIssuer(assetresolution.IssuerInput{
			IssuerName: effective.DirectIssuer, PublicationAt: input.PublicationTimestamp, ReceiptAt: input.ReceiptTimestamp,
		})
		return newPolicyResolution(resolved, effective.DirectIssuer)
	case "PROXY":
		resolved, _ := resolver.ResolveProxyExposure(effective.ProxyExposure)
		return newPolicyResolution(resolved, "")
	default:
		reason := "model classified the receipt-time asset mapping as unresolved"
		if guard.Abstained {
			reason = "causal-consistency guard abstained before deterministic asset resolution: " + strings.Join(guard.ReasonCodes, ",")
		}
		return PolicyResolution{Status: assetresolution.StatusUnresolved, PolicyVersion: resolver.Rules.Version, MappingType: "none", Relationship: "none", Reason: reason}
	}
}

func newPolicyResolution(result assetresolution.Result, rawDirectIssuer string) PolicyResolution {
	return PolicyResolution{
		Status: result.Status, PolicyVersion: result.RulesetVersion,
		RawDirectIssuer: rawDirectIssuer, NormalizedIssuer: assetresolution.CanonicalizeIssuerName(rawDirectIssuer),
		CanonicalIssuer: result.CanonicalEntity, MatchedAlias: result.MatchedAlias, MatchedRule: result.CanonicalEntity,
		ResolvedTicker: result.Symbol, MappingType: result.MappingType, Relationship: result.Relationship, Reason: result.Reason,
	}
}

func DecodePersistedResult(schemaVersion string, raw []byte) (PersistedStructuredResult, error) {
	result := PersistedStructuredResult{SchemaVersion: schemaVersion}
	switch schemaVersion {
	case SchemaVersion:
		var current V4PersistedResult
		if err := decodePersistedJSON(raw, &current); err != nil {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: %w", schemaVersion, err)
		}
		if current.ModelOutput.MappingStatus == "" || current.DeterministicResolution.PolicyVersion == "" {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: model output or deterministic resolution provenance is missing", schemaVersion)
		}
		result.Current = &current
	case V3SchemaVersion:
		var current V3PersistedResult
		if err := decodePersistedJSON(raw, &current); err != nil {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: %w", schemaVersion, err)
		}
		if current.ModelOutput.MappingStatus == "" || current.DeterministicResolution.PolicyVersion == "" {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: model output or deterministic resolution provenance is missing", schemaVersion)
		}
		result.V3 = &current
	case V2SchemaVersion:
		var v2 V2StructuredResult
		if err := decodePersistedJSON(raw, &v2); err != nil {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: %w", schemaVersion, err)
		}
		if v2.MappingStatus == "" {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: required v2 fields are missing", schemaVersion)
		}
		result.V2 = &v2
	case LegacySchemaVersion:
		var legacy LegacyStructuredResult
		if err := decodePersistedJSON(raw, &legacy); err != nil {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: %w", schemaVersion, err)
		}
		if legacy.AssetMappingType == "" {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: required v1 fields are missing", schemaVersion)
		}
		result.Legacy = &legacy
	default:
		return PersistedStructuredResult{}, fmt.Errorf("unsupported AI shadow schema version %q", schemaVersion)
	}
	return result, nil
}

func decodePersistedJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("JSON response contains trailing data")
	}
	return fmt.Errorf("JSON trailing data: %v", err)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
