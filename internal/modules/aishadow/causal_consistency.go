package aishadow

import (
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	CausalConsistencyPolicyVersion = "ai-shadow-causal-consistency-c1d-v1"

	ReasonDirectRelevanceNotHigh    = "DIRECT_RELEVANCE_NOT_HIGH"
	ReasonProxyCompetingIssuerInput = "PROXY_COMPETING_TRADED_ISSUERS"
)

// AssetMapping is the mapping-only projection of the canonical model output.
// Keeping this projection separate avoids mutating or redefining the provider's
// ten-field response contract.
type AssetMapping struct {
	MappingStatus string `json:"mapping_status"`
	DirectIssuer  string `json:"direct_issuer"`
	ProxyExposure string `json:"proxy_exposure"`
}

// CausalConsistencyDecision records the raw mapping and the independently
// effective mapping consumed by deterministic asset resolution.
type CausalConsistencyDecision struct {
	PolicyVersion              string       `json:"policy_version"`
	RawMapping                 AssetMapping `json:"raw_mapping"`
	EffectiveMapping           AssetMapping `json:"effective_mapping"`
	Abstained                  bool         `json:"abstained"`
	ReasonCodes                []string     `json:"reason_codes"`
	RecognizedInputIssuerCount int          `json:"recognized_input_issuer_count"`
}

// ApplyCausalConsistencyGuard is monotonic: it either preserves the model
// mapping or replaces a DIRECT/PROXY mapping with UNRESOLVED. It never selects
// or substitutes an issuer, proxy, or ticker.
func ApplyCausalConsistencyGuard(result StructuredResult, input EventInput, resolver assetresolution.Resolver) CausalConsistencyDecision {
	raw := mappingFromStructuredResult(result)
	decision := CausalConsistencyDecision{
		PolicyVersion:    CausalConsistencyPolicyVersion,
		RawMapping:       raw,
		EffectiveMapping: raw,
		ReasonCodes:      []string{},
	}

	switch result.MappingStatus {
	case "DIRECT":
		if result.MarketRelevance != "HIGH" {
			decision.abstain(ReasonDirectRelevanceNotHigh)
		}
	case "PROXY":
		issuerCount := recognizedInputIssuerCount(input, resolver)
		decision.RecognizedInputIssuerCount = issuerCount
		if strings.EqualFold(strings.TrimSpace(input.EventCategory), "company") && issuerCount >= 2 {
			decision.abstain(ReasonProxyCompetingIssuerInput)
		}
	}
	return decision
}

func (d *CausalConsistencyDecision) abstain(reason string) {
	d.Abstained = true
	d.EffectiveMapping = AssetMapping{MappingStatus: "UNRESOLVED", ProxyExposure: NoProxyExposure}
	d.ReasonCodes = []string{reason}
}

func mappingFromStructuredResult(result StructuredResult) AssetMapping {
	return AssetMapping{
		MappingStatus: result.MappingStatus,
		DirectIssuer:  result.DirectIssuer,
		ProxyExposure: result.ProxyExposure,
	}
}

func recognizedInputIssuerCount(input EventInput, resolver assetresolution.Resolver) int {
	seen := map[string]bool{}
	for _, entity := range input.Entities {
		normalized := assetresolution.CanonicalizeIssuerName(entity)
		if normalized == "" {
			continue
		}
		for _, rule := range resolver.Rules.Aliases {
			if !activeAliasRule(rule.EffectiveFrom, rule.EffectiveTo, input.PublicationTimestamp) {
				continue
			}
			if assetresolution.CanonicalizeIssuerName(rule.CanonicalEntity) == normalized || exactAliasMatch(normalized, rule.Aliases) {
				seen[assetresolution.CanonicalizeIssuerName(rule.CanonicalEntity)] = true
			}
		}
	}
	return len(seen)
}

func exactAliasMatch(normalized string, aliases []string) bool {
	for _, alias := range aliases {
		if assetresolution.CanonicalizeIssuerName(alias) == normalized {
			return true
		}
	}
	return false
}

func activeAliasRule(fromValue, toValue string, at time.Time) bool {
	if from, err := time.Parse("2006-01-02", strings.TrimSpace(fromValue)); err == nil && at.Before(from) {
		return false
	}
	if to, err := time.Parse("2006-01-02", strings.TrimSpace(toValue)); err == nil && at.After(to.Add(24*time.Hour-time.Nanosecond)) {
		return false
	}
	return true
}
