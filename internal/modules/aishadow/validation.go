package aishadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"jax-trading-assistant/internal/modules/assetresolution"
)

var tickerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,14}$`)

var requiredResultFields = []string{
	"market_relevance", "mapping_status", "direct_ticker", "proxy_exposure", "mapping_confidence", "expected_horizon",
	"likely_direction", "catalyst_type", "reason", "missing_evidence",
}

func ParseAndValidate(raw string, input EventInput, resolver assetresolution.Resolver) (*StructuredResult, *PolicyResolution, []string) {
	errors := []string{}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, nil, []string{"invalid JSON: " + err.Error()}
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
	for _, field := range []string{"direct_ticker", "proxy_exposure"} {
		if rawValue, ok := fields[field]; ok && bytes.Equal(bytes.TrimSpace(rawValue), []byte("null")) {
			errors = append(errors, field+" must be a string, not null")
		}
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var result StructuredResult
	if err := decoder.Decode(&result); err != nil {
		errors = append(errors, "schema decode: "+err.Error())
		return nil, nil, uniqueSorted(errors)
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
	if result.DirectTicker != "" && !tickerPattern.MatchString(result.DirectTicker) {
		errors = append(errors, "direct_ticker has an invalid format")
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
	var resolution *PolicyResolution
	switch result.MappingStatus {
	case "DIRECT":
		if result.DirectTicker == "" {
			errors = append(errors, "DIRECT mapping requires a non-empty direct_ticker")
		}
		if result.ProxyExposure != NoProxyExposure {
			errors = append(errors, "DIRECT mapping requires proxy_exposure NONE")
		}
		if result.MappingConfidence == "LOW" {
			errors = append(errors, "DIRECT mapping requires HIGH or MEDIUM mapping_confidence")
		}
		resolved := resolver.Resolve(assetResolutionInput(input))
		if resolved.Status != assetresolution.StatusResolved || resolved.Relationship != "direct" || resolved.Symbol != result.DirectTicker {
			errors = append(errors, "direct_ticker was not independently verified by receipt-time Jax policy")
		} else {
			value := newPolicyResolution(resolved)
			resolution = &value
		}
	case "PROXY":
		if result.DirectTicker != "" {
			errors = append(errors, "PROXY mapping requires an empty direct_ticker")
		}
		if result.ProxyExposure == NoProxyExposure {
			errors = append(errors, "PROXY mapping requires a bounded proxy_exposure")
		} else if resolved, ok := resolver.ResolveProxyExposure(result.ProxyExposure); !ok {
			errors = append(errors, "proxy_exposure is not allowlisted by the active Jax policy")
		} else {
			value := newPolicyResolution(resolved)
			resolution = &value
		}
	case "UNRESOLVED":
		if result.DirectTicker != "" {
			errors = append(errors, "UNRESOLVED mapping requires an empty direct_ticker")
		}
		if result.ProxyExposure != NoProxyExposure {
			errors = append(errors, "UNRESOLVED mapping requires proxy_exposure NONE")
		}
		if result.MappingConfidence != "LOW" {
			errors = append(errors, "UNRESOLVED mapping requires LOW mapping_confidence")
		}
		value := PolicyResolution{Status: assetresolution.StatusUnresolved, PolicyVersion: resolver.Rules.Version, MappingType: "none", Relationship: "none", Reason: "model classified the receipt-time asset mapping as unresolved"}
		resolution = &value
	}
	if len(errors) > 0 {
		return nil, nil, uniqueSorted(errors)
	}
	return &result, resolution, nil
}

func assetResolutionInput(input EventInput) assetresolution.Input {
	return assetresolution.Input{
		Headline:      input.Title,
		Summary:       input.Summary,
		SourceName:    input.Source,
		EventType:     input.EventCategory,
		PublicationAt: input.PublicationTimestamp,
		ReceiptAt:     input.ReceiptTimestamp,
	}
}

func newPolicyResolution(result assetresolution.Result) PolicyResolution {
	return PolicyResolution{
		Status:         result.Status,
		PolicyVersion:  result.RulesetVersion,
		MatchedRule:    result.CanonicalEntity,
		ResolvedTicker: result.Symbol,
		MappingType:    result.MappingType,
		Relationship:   result.Relationship,
		Reason:         result.Reason,
	}
}

func DecodePersistedResult(schemaVersion string, raw []byte) (PersistedStructuredResult, error) {
	result := PersistedStructuredResult{SchemaVersion: schemaVersion}
	switch schemaVersion {
	case SchemaVersion:
		var current V3PersistedResult
		if err := decodePersistedJSON(raw, &current); err != nil {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: %w", schemaVersion, err)
		}
		if current.ModelOutput.MappingStatus == "" || current.DeterministicResolution.PolicyVersion == "" {
			return PersistedStructuredResult{}, fmt.Errorf("decode %s result: model output or deterministic resolution provenance is missing", schemaVersion)
		}
		result.Current = &current
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
