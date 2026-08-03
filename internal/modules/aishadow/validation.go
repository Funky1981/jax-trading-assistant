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
)

var tickerPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,9}$`)

var requiredResultFields = []string{
	"market_relevance", "resolved_asset", "asset_mapping_type", "expected_horizon",
	"likely_direction", "confidence", "catalyst_type", "reason", "missing_evidence",
}

func ParseAndValidate(raw string) (*StructuredResult, []string) {
	errors := []string{}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, []string{"invalid JSON: " + err.Error()}
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
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	var result StructuredResult
	if err := decoder.Decode(&result); err != nil {
		errors = append(errors, "schema decode: "+err.Error())
		return nil, uniqueSorted(errors)
	}
	if err := ensureEOF(decoder); err != nil {
		errors = append(errors, err.Error())
	}
	if !contains([]string{"HIGH", "MEDIUM", "LOW", "UNCERTAIN"}, result.MarketRelevance) {
		errors = append(errors, "market_relevance must be HIGH, MEDIUM, LOW, or UNCERTAIN")
	}
	if !contains([]string{"direct", "proxy", "unresolved"}, result.AssetMappingType) {
		errors = append(errors, "asset_mapping_type must be direct, proxy, or unresolved")
	}
	if !contains([]string{"intraday", "1d", "multi-day", "unclear"}, result.ExpectedHorizon) {
		errors = append(errors, "expected_horizon has an invalid value")
	}
	if !contains([]string{"positive", "negative", "neutral", "unclear"}, result.LikelyDirection) {
		errors = append(errors, "likely_direction has an invalid value")
	}
	if result.Confidence < 0 || result.Confidence > 100 {
		errors = append(errors, "confidence must be between 0 and 100")
	}
	asset := ""
	if result.ResolvedAsset != nil {
		asset = *result.ResolvedAsset
	}
	if result.AssetMappingType == "unresolved" {
		if result.ResolvedAsset != nil {
			errors = append(errors, "unresolved mapping requires resolved_asset null")
		}
		if result.Confidence > 49 {
			errors = append(errors, "unresolved mapping confidence must be 49 or lower")
		}
	} else if result.AssetMappingType == "direct" || result.AssetMappingType == "proxy" {
		if result.ResolvedAsset == nil || strings.TrimSpace(asset) == "" {
			errors = append(errors, "direct and proxy mappings require a non-empty ticker")
		} else if !tickerPattern.MatchString(asset) {
			errors = append(errors, "resolved_asset has an invalid ticker format")
		}
	}
	if result.AssetMappingType == "direct" && result.Confidence < 70 {
		errors = append(errors, "direct mapping confidence must be at least 70")
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
	if len(errors) > 0 {
		return nil, uniqueSorted(errors)
	}
	return &result, nil
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
