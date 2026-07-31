package evidencequality

import (
	"net/url"
	"sort"
	"strings"
	"time"
)

func BuildPopulation(events []Event, coverageEnd time.Time, rules Ruleset) ([]Event, []Exclusion) {
	ordered := append([]Event(nil), events...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ReceiptAt.Equal(ordered[j].ReceiptAt) {
			return ordered[i].SourceEventIdentity < ordered[j].SourceEventIdentity
		}
		return ordered[i].ReceiptAt.Before(ordered[j].ReceiptAt)
	})
	included := []Event{}
	exclusions := []Exclusion{}
	seen := map[string]bool{}
	for _, event := range ordered {
		reason := exclusionReason(event, coverageEnd, rules)
		identity := duplicateIdentity(event)
		if reason == "" && identity != "" && seen[identity] {
			reason = "duplicate_event"
		}
		if reason != "" {
			exclusions = append(exclusions, Exclusion{SourceEventIdentity: event.SourceEventIdentity, Decision: event.Decision, Reason: reason})
			continue
		}
		if identity != "" {
			seen[identity] = true
		}
		included = append(included, event)
	}
	return included, exclusions
}

func exclusionReason(event Event, coverageEnd time.Time, rules Ruleset) string {
	if event.IsSynthetic {
		return "synthetic_record"
	}
	lowerSource := strings.ToLower(strings.TrimSpace(event.Source))
	for _, prefix := range rules.ControlledSourcePrefixes {
		if strings.HasPrefix(lowerSource, strings.ToLower(prefix)) {
			return "controlled_proof_record"
		}
	}
	lowerIdentity := strings.ToLower(event.SourceEventIdentity)
	for _, marker := range rules.ControlledEventIDMarkers {
		if strings.Contains(lowerIdentity, strings.ToLower(marker)) {
			return "manual_test_record"
		}
	}
	lowerHeadline := strings.ToLower(strings.TrimSpace(event.Headline))
	for _, marker := range rules.ControlledHeadlineMarkers {
		if strings.Contains(lowerHeadline, strings.ToLower(marker)) {
			return "controlled_proof_record"
		}
	}
	if parsed, err := url.Parse(strings.TrimSpace(event.SourceURL)); err == nil {
		host := strings.ToLower(parsed.Hostname())
		for _, testHost := range rules.TestHosts {
			if host == strings.ToLower(testHost) || strings.HasSuffix(host, "."+strings.ToLower(testHost)) {
				return "test_source_url"
			}
		}
	}
	if event.PublicationAt.IsZero() || event.ReceiptAt.IsZero() || event.DecisionAt.IsZero() {
		return "untrustworthy_timing"
	}
	if event.PublicationAt.After(event.ReceiptAt) || event.ReceiptAt.After(event.DecisionAt) {
		return "invalid_timestamp_order"
	}
	if event.CollectionAt != nil && (event.CollectionAt.Before(event.PublicationAt) || event.CollectionAt.After(event.ReceiptAt)) {
		return "invalid_collection_timestamp"
	}
	if !coverageEnd.IsZero() && event.ReceiptAt.After(coverageEnd) {
		return "after_market_data_coverage"
	}
	return ""
}

func duplicateIdentity(event Event) string {
	if value := strings.ToLower(strings.TrimSpace(event.ContentHash)); value != "" {
		return "hash:" + value
	}
	if parsed, err := url.Parse(strings.TrimSpace(event.SourceURL)); err == nil && parsed.Hostname() != "" {
		parsed.Fragment = ""
		query := parsed.Query()
		for key := range query {
			lower := strings.ToLower(key)
			if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
				query.Del(key)
			}
		}
		parsed.RawQuery = query.Encode()
		parsed.Host = strings.ToLower(parsed.Host)
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
		return "url:" + parsed.String()
	}
	return ""
}

func MapEvent(event Event, rules Ruleset) Mapping {
	base := Mapping{RulesetVersion: rules.Version, Reason: "no conservative deterministic asset mapping was available"}
	if symbol := normalizeSymbol(event.PrimarySymbol); symbol != "" {
		return withBenchmark(Mapping{Mapped: true, MappingType: "explicit_normalized_primary_symbol", Symbol: symbol, Confidence: "high", Reason: "persisted normalized event primary symbol", Direct: true, RulesetVersion: rules.Version}, rules)
	}
	assets := normalizedSymbols(event.AffectedAssets)
	if len(assets) == 1 {
		return withBenchmark(Mapping{Mapped: true, MappingType: "explicit_resolved_asset", Symbol: assets[0], Confidence: "high", Reason: "single explicitly persisted affected asset", Direct: true, RulesetVersion: rules.Version}, rules)
	}
	if rule, ok := rules.CategoryProxies[strings.ToLower(strings.TrimSpace(event.EventType))]; ok {
		symbol := normalizeSymbol(rule.Symbol)
		if symbol != "" {
			return withBenchmark(Mapping{Mapped: true, MappingType: "event_category_proxy", Symbol: symbol, Confidence: rule.Confidence, Reason: rule.Reason, Direct: false, RulesetVersion: rules.Version}, rules)
		}
	}
	return base
}

func withBenchmark(mapping Mapping, rules Ruleset) Mapping {
	if rule, ok := rules.Benchmarks[mapping.Symbol]; ok {
		mapping.Benchmark = normalizeSymbol(rule.Symbol)
		mapping.BenchmarkReason = rule.Reason
	}
	return mapping
}

func normalizeSymbol(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || value == "UNKNOWN" || value == "N/A" {
		return ""
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '.' && char != '-' {
			return ""
		}
	}
	return value
}

func normalizedSymbols(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if symbol := normalizeSymbol(value); symbol != "" && !seen[symbol] {
			seen[symbol] = true
			result = append(result, symbol)
		}
	}
	sort.Strings(result)
	return result
}
