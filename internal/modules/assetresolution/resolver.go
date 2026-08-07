package assetresolution

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.-]{0,14}$`)
var exposurePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,79}$`)
var shareClassPattern = regexp.MustCompile(`(?i)\bclass\s+[a-z0-9]+\b`)

type Resolver struct{ Rules Ruleset }

// CanonicalizeIssuerName applies the complete deterministic normalization used
// for issuer identity lookups. It intentionally does not perform fuzzy or
// substring matching.
func CanonicalizeIssuerName(value string) string {
	var normalized strings.Builder
	space := true
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			normalized.WriteRune(r)
			space = false
		case r == '&':
			if !space && normalized.Len() > 0 {
				normalized.WriteByte(' ')
			}
			normalized.WriteString("and")
			normalized.WriteByte(' ')
			space = true
		default:
			if !space && normalized.Len() > 0 {
				normalized.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(normalized.String())
}

// ResolveIssuer maps one explicit issuer identity through the existing alias
// rules. Unknown identities and collisions remain valid unresolved/ambiguous
// results; no ticker or proxy fallback is synthesized.
func (r Resolver) ResolveIssuer(input IssuerInput) Result {
	raw := strings.TrimSpace(input.IssuerName)
	normalized := CanonicalizeIssuerName(raw)
	base := Result{
		Status: StatusUnresolved, MappingType: "none", Relationship: "none", ConfidenceClass: "unresolved",
		Reason:       "issuer name did not match a bounded deterministic alias rule",
		SourceFields: []string{"direct_issuer"}, SourceValues: map[string]any{"raw_direct_issuer": raw, "normalized_issuer": normalized},
		RulesetVersion: r.Rules.Version, KnowableAtOperationalAnchor: validIssuerAnchor(input),
	}
	if !base.KnowableAtOperationalAnchor {
		base.Status = StatusRejected
		base.ConfidenceClass = "rejected"
		base.Reason = "issuer resolution requires valid receipt-time publication and receipt anchors"
		base.RejectionReason = base.Reason
		return base
	}
	if normalized == "" {
		return base
	}

	type issuerMatch struct {
		rule  AliasRule
		alias string
	}
	matches := map[string]issuerMatch{}
	for _, rule := range r.Rules.Aliases {
		if !effective(rule, input.PublicationAt) {
			continue
		}
		matchedAlias := ""
		if CanonicalizeIssuerName(rule.CanonicalEntity) == normalized {
			matchedAlias = rule.CanonicalEntity
		} else {
			for _, alias := range rule.Aliases {
				if CanonicalizeIssuerName(alias) == normalized {
					matchedAlias = alias
					break
				}
			}
		}
		if matchedAlias != "" {
			matches[strings.ToUpper(strings.TrimSpace(rule.Symbol))] = issuerMatch{rule: rule, alias: matchedAlias}
		}
	}
	if len(matches) == 0 {
		return base
	}
	if len(matches) > 1 {
		symbols := make([]string, 0, len(matches))
		for symbol := range matches {
			symbols = append(symbols, symbol)
		}
		sort.Strings(symbols)
		base.Status = StatusAmbiguous
		base.ConfidenceClass = "ambiguous"
		base.MappingType = "ambiguous_issuer_identity"
		base.Relationship = "direct"
		base.Reason = "issuer identity matched multiple listed share classes or issuers; no ticker was selected"
		base.AmbiguityReason = strings.Join(symbols, ",")
		return base
	}
	for symbol, match := range matches {
		if shareClassPattern.MatchString(match.rule.Provenance) || shareClassPattern.MatchString(match.rule.Rationale) {
			base.Status = StatusAmbiguous
			base.ConfidenceClass = "ambiguous"
			base.MappingType = "ambiguous_share_class"
			base.Relationship = "direct"
			base.Reason = "issuer identity maps to a share-class-specific listing but does not select a share class; no ticker was selected"
			base.AmbiguityReason = symbol
			base.CanonicalEntity = match.rule.CanonicalEntity
			base.MatchedAlias = match.alias
			return base
		}
		base.Status = StatusResolved
		base.Symbol = symbol
		base.MappingType = "verified_issuer_identity"
		base.Relationship = "direct"
		base.ConfidenceClass = "high_confidence_deterministic"
		base.Reason = match.rule.Rationale
		base.CanonicalEntity = match.rule.CanonicalEntity
		base.MatchedAlias = match.alias
		base.AssetClass = match.rule.AssetClass
		base.Exchange = match.rule.Exchange
		base.EffectiveFrom = parsedRuleDate(match.rule.EffectiveFrom)
		base.EffectiveTo = parsedRuleDate(match.rule.EffectiveTo)
		base.SourceValues["alias_provenance"] = match.rule.Provenance
		return base
	}
	return base
}

// ProxyExposures returns the model-facing exposure names derived from the
// versioned proxy rules. Symbols remain private to the deterministic resolver.
func (r Resolver) ProxyExposures() ([]string, error) {
	seen := map[string]bool{}
	exposures := make([]string, 0, len(r.Rules.Proxies))
	for _, rule := range r.Rules.Proxies {
		exposure := strings.ToUpper(strings.TrimSpace(rule.Key))
		if !exposurePattern.MatchString(exposure) {
			return nil, fmt.Errorf("asset resolution proxy key %q cannot be used as a bounded exposure", rule.Key)
		}
		if exposure == "NONE" {
			return nil, fmt.Errorf("asset resolution proxy key %q conflicts with the unresolved exposure sentinel", rule.Key)
		}
		symbol := strings.ToUpper(strings.TrimSpace(rule.Symbol))
		if !symbolPattern.MatchString(symbol) {
			return nil, fmt.Errorf("asset resolution proxy %q has invalid symbol %q", rule.Key, rule.Symbol)
		}
		if seen[exposure] {
			return nil, fmt.Errorf("duplicate asset resolution proxy exposure %q", exposure)
		}
		seen[exposure] = true
		exposures = append(exposures, exposure)
	}
	if len(exposures) == 0 {
		return nil, fmt.Errorf("asset resolution ruleset has no proxy exposures")
	}
	sort.Strings(exposures)
	return exposures, nil
}

// ResolveProxyExposure maps a bounded exposure through the existing versioned
// proxy policy. It does not re-run event matching or accept model-supplied
// symbols.
func (r Resolver) ResolveProxyExposure(exposure string) (Result, bool) {
	wanted := strings.ToUpper(strings.TrimSpace(exposure))
	for _, proxy := range r.Rules.Proxies {
		if strings.ToUpper(strings.TrimSpace(proxy.Key)) != wanted {
			continue
		}
		return Result{
			Status:          StatusResolved,
			Symbol:          strings.ToUpper(proxy.Symbol),
			Benchmark:       strings.ToUpper(proxy.Benchmark),
			MappingType:     proxy.MappingType,
			Relationship:    "proxy",
			ConfidenceClass: "proxy",
			Reason:          proxy.Reason,
			SourceFields:    []string{"model_proxy_exposure"},
			SourceValues:    map[string]any{"proxy_exposure": wanted},
			RulesetVersion:  r.Rules.Version,
			CanonicalEntity: proxy.Key,
			AssetClass:      proxy.AssetClass,
		}, true
	}
	return Result{}, false
}

func LoadRuleset(path string) (Ruleset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Ruleset{}, fmt.Errorf("read asset resolution ruleset: %w", err)
	}
	var rules Ruleset
	if err := json.Unmarshal(raw, &rules); err != nil {
		return Ruleset{}, fmt.Errorf("decode asset resolution ruleset: %w", err)
	}
	if strings.TrimSpace(rules.Version) == "" || len(rules.Aliases) == 0 || len(rules.Proxies) == 0 {
		return Ruleset{}, fmt.Errorf("asset resolution ruleset is incomplete")
	}
	return rules, nil
}

func (r Resolver) Resolve(input Input) Result {
	base := Result{Status: StatusUnresolved, MappingType: "none", Relationship: "none", ConfidenceClass: "unresolved",
		Reason: "no bounded deterministic asset rule matched", SourceFields: []string{"headline", "summary", "event_type", "source_name", "source_url"},
		SourceValues: sourceValues(input), RulesetVersion: r.Rules.Version, KnowableAtOperationalAnchor: validAnchorInput(input)}
	base.MaterialEvent = containsPhraseAny(strings.ToLower(input.Headline), r.Rules.MaterialEventTerms)
	explicit := normalizedSymbols(input.ExplicitSymbols)
	if len(explicit) == 1 {
		base.Status, base.Symbol, base.MappingType, base.Relationship, base.ConfidenceClass = StatusResolved, explicit[0], "trusted_structured_symbol", "direct", "exact"
		base.Reason = firstNonEmpty(input.ExplicitReason, "single trusted structured source symbol")
		base.SourceFields = append([]string{"explicit_symbols"}, input.ExplicitMethods...)
		base.MaterialEvent = base.MaterialEvent || recognizedMaterialCategory(input.EventType)
		return base
	}
	if len(explicit) > 1 {
		base.Status, base.ConfidenceClass = StatusAmbiguous, "ambiguous"
		base.Reason = "multiple structured symbols were present; no primary asset was selected"
		base.AmbiguityReason = strings.Join(explicit, ",")
		base.SourceFields = []string{"explicit_symbols"}
		return base
	}

	text := strings.ToLower(input.Headline)
	contextOK := containsPhraseAny(text, r.Rules.FinancialContextTerms)
	matches := map[string]AliasRule{}
	for _, rule := range r.Rules.Aliases {
		if !effective(rule, input.PublicationAt) || (rule.RequiresContext && !contextOK) {
			continue
		}
		for _, alias := range rule.Aliases {
			if wholePhrase(text, alias) {
				matches[rule.Symbol] = rule
				break
			}
		}
	}
	if len(matches) == 1 {
		for _, match := range matches {
			base.Status, base.Symbol, base.MappingType, base.Relationship, base.ConfidenceClass = StatusResolved, strings.ToUpper(match.Symbol), "verified_issuer_alias", "direct", "high_confidence_deterministic"
			base.Reason, base.CanonicalEntity, base.AssetClass, base.Exchange = match.Rationale, match.CanonicalEntity, match.AssetClass, match.Exchange
			base.EffectiveFrom, base.EffectiveTo = parsedRuleDate(match.EffectiveFrom), parsedRuleDate(match.EffectiveTo)
			base.SourceFields = []string{"headline"}
			base.SourceValues["alias_provenance"] = match.Provenance
			base.MaterialEvent = contextOK
			return base
		}
	}
	if len(matches) > 1 {
		symbols := make([]string, 0, len(matches))
		for symbol := range matches {
			symbols = append(symbols, symbol)
		}
		sort.Strings(symbols)
		base.Status, base.ConfidenceClass = StatusAmbiguous, "ambiguous"
		base.Reason = "multiple listed issuers matched the event text; no primary asset was selected"
		base.AmbiguityReason = strings.Join(symbols, ",")
		return base
	}

	for _, proxy := range r.Rules.Proxies {
		if proxyMatches(proxy, input) {
			base.Status, base.Symbol, base.Benchmark, base.MappingType = StatusResolved, strings.ToUpper(proxy.Symbol), strings.ToUpper(proxy.Benchmark), proxy.MappingType
			base.Relationship, base.ConfidenceClass, base.Reason, base.AssetClass = "proxy", "proxy", proxy.Reason, proxy.AssetClass
			base.CanonicalEntity = proxy.Key
			base.SourceFields = proxySourceFields(proxy)
			base.MaterialEvent = proxy.Key != "federal_reserve_official" || containsPhraseAny(text, r.Rules.MaterialEventTerms)
			return base
		}
	}
	base.MaterialEvent = base.MaterialEvent || recognizedMaterialCategory(input.EventType)
	return base
}

func sourceValues(input Input) map[string]any {
	return map[string]any{"headline": input.Headline, "summary": input.Summary, "event_type": input.EventType, "source_name": input.SourceName, "source_url": input.SourceURL}
}
func validAnchorInput(input Input) bool {
	return !input.ReceiptAt.IsZero() && !input.PublicationAt.IsZero() && !input.PublicationAt.After(input.ReceiptAt)
}
func validIssuerAnchor(input IssuerInput) bool {
	return !input.ReceiptAt.IsZero() && !input.PublicationAt.IsZero() && !input.PublicationAt.After(input.ReceiptAt)
}
func normalizedSymbols(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if symbolPattern.MatchString(value) && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
func containsAny(text string, values []string) bool {
	for _, value := range values {
		if strings.Contains(text, strings.ToLower(value)) {
			return true
		}
	}
	return false
}
func containsPhraseAny(text string, values []string) bool {
	for _, value := range values {
		if wholePhrase(text, value) {
			return true
		}
	}
	return false
}
func recognizedMaterialCategory(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "macro_rates", "inflation", "energy_oil", "cyber_outage", "supply_chain", "financial_credit", "geopolitical":
		return true
	default:
		return false
	}
}
func wholePhrase(text, phrase string) bool {
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	if phrase == "" {
		return false
	}
	pattern := regexp.MustCompile(`(^|[^a-z0-9])` + regexp.QuoteMeta(phrase) + `([^a-z0-9]|$)`)
	return pattern.MatchString(text)
}
func effective(rule AliasRule, at time.Time) bool {
	from, to := parsedRuleDate(rule.EffectiveFrom), parsedRuleDate(rule.EffectiveTo)
	if from != nil && at.Before(*from) {
		return false
	}
	if to != nil && at.After(to.Add(24*time.Hour-time.Nanosecond)) {
		return false
	}
	return true
}
func parsedRuleDate(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}
func proxyMatches(rule ProxyRule, input Input) bool {
	for _, name := range rule.SourceNames {
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(input.SourceName)) {
			return true
		}
	}
	host := ""
	if parsed, err := url.Parse(input.SourceURL); err == nil {
		host = strings.ToLower(parsed.Hostname())
	}
	for _, candidate := range rule.SourceHosts {
		candidate = strings.ToLower(candidate)
		if host == candidate || strings.HasSuffix(host, "."+candidate) {
			return true
		}
	}
	for _, kind := range rule.EventTypes {
		if strings.EqualFold(kind, input.EventType) {
			return true
		}
	}
	lower := strings.ToLower(input.Headline)
	for _, term := range rule.HeadlineTerms {
		if wholePhrase(lower, term) {
			return true
		}
	}
	return false
}
func proxySourceFields(rule ProxyRule) []string {
	fields := []string{}
	if len(rule.SourceNames) > 0 {
		fields = append(fields, "source_name")
	}
	if len(rule.SourceHosts) > 0 {
		fields = append(fields, "source_url")
	}
	if len(rule.EventTypes) > 0 {
		fields = append(fields, "event_type")
	}
	if len(rule.HeadlineTerms) > 0 {
		fields = append(fields, "headline")
	}
	return fields
}
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
