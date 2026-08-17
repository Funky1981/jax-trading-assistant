package aishadow

import (
	"fmt"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const (
	CausalAttributionPolicyVersion = "ai-shadow-causal-attribution-c1e-v1"

	ReasonUniquePrincipalIssuerAccepted = "UNIQUE_PRINCIPAL_ISSUER_ACCEPTED"
	ReasonUniquePrincipalProxyAccepted  = "UNIQUE_PRINCIPAL_PROXY_ACCEPTED"
	ReasonMultipleEqualPrincipals       = "MULTIPLE_EQUAL_PRINCIPAL_ISSUERS"
	ReasonPossiblePrincipal             = "POSSIBLE_PRINCIPAL_ISSUER"
	ReasonNoPrincipalMapping            = "NO_PRINCIPAL_MAPPING"
	ReasonMultiplePrincipalProxies      = "MULTIPLE_PRINCIPAL_PROXY_CANDIDATES"
	ReasonAttributionProjectionMismatch = "CAUSAL_ATTRIBUTION_PROJECTION_MISMATCH"
)

// TypedCausalAttribution is the policy input retained independently of both
// the raw model object and the effective mapping.
type TypedCausalAttribution struct {
	IssuerAttributions       []IssuerAttribution `json:"issuer_attributions"`
	PrincipalProxyCandidates []string            `json:"principal_proxy_candidates"`
}

type CausalAttributionDecision struct {
	PolicyVersion    string       `json:"policy_version"`
	RawMapping       AssetMapping `json:"raw_mapping"`
	EffectiveMapping AssetMapping `json:"effective_mapping"`
	Accepted         bool         `json:"accepted"`
	Abstained        bool         `json:"abstained"`
	ReasonCode       string       `json:"reason_code"`
}

func TypedAttributionFromV5(result V5StructuredResult) TypedCausalAttribution {
	return TypedCausalAttribution{
		IssuerAttributions:       append([]IssuerAttribution(nil), result.IssuerAttributions...),
		PrincipalProxyCandidates: append([]string(nil), result.PrincipalProxyCandidates...),
	}
}

// ApplyCausalAttributionPolicy derives the only selectable mapping from typed
// v5 fields. It never reads event prose, relevance, reason, missing evidence,
// entities, or resolver coverage.
func ApplyCausalAttributionPolicy(result V5StructuredResult) (CausalAttributionDecision, error) {
	raw := AssetMapping{MappingStatus: result.MappingStatus, DirectIssuer: result.DirectIssuer, ProxyExposure: result.ProxyExposure}
	decision := CausalAttributionDecision{
		PolicyVersion: CausalAttributionPolicyVersion,
		RawMapping:    raw,
	}

	principalCount, equalCount, possibleCount := causalRoleCounts(result.IssuerAttributions)
	switch {
	case principalCount == 1 && equalCount == 0 && possibleCount == 0 && len(result.PrincipalProxyCandidates) == 0:
		principal := principalIssuer(result.IssuerAttributions)
		decision.EffectiveMapping = AssetMapping{MappingStatus: "DIRECT", DirectIssuer: principal, ProxyExposure: NoProxyExposure}
		decision.Accepted = true
		decision.ReasonCode = ReasonUniquePrincipalIssuerAccepted
	case principalCount == 0 && equalCount == 0 && possibleCount == 0 && len(result.PrincipalProxyCandidates) == 1:
		decision.EffectiveMapping = AssetMapping{MappingStatus: "PROXY", ProxyExposure: result.PrincipalProxyCandidates[0]}
		decision.Accepted = true
		decision.ReasonCode = ReasonUniquePrincipalProxyAccepted
	case principalCount == 0 && equalCount >= 2 && possibleCount == 0:
		decision.abstain(ReasonMultipleEqualPrincipals)
	case principalCount == 0 && equalCount == 0 && possibleCount > 0:
		decision.abstain(ReasonPossiblePrincipal)
	case principalCount == 0 && equalCount == 0 && possibleCount == 0 && len(result.PrincipalProxyCandidates) > 1:
		decision.abstain(ReasonMultiplePrincipalProxies)
	case principalCount == 0 && equalCount == 0 && possibleCount == 0 && len(result.PrincipalProxyCandidates) == 0:
		decision.abstain(ReasonNoPrincipalMapping)
	default:
		return CausalAttributionDecision{}, fmt.Errorf("%s", ReasonAttributionProjectionMismatch)
	}

	if decision.RawMapping != decision.EffectiveMapping {
		return CausalAttributionDecision{}, fmt.Errorf("%s", ReasonAttributionProjectionMismatch)
	}
	return decision, nil
}

func (d *CausalAttributionDecision) abstain(reason string) {
	d.EffectiveMapping = AssetMapping{MappingStatus: "UNRESOLVED", ProxyExposure: NoProxyExposure}
	d.Abstained = true
	d.ReasonCode = reason
}

func causalRoleCounts(attributions []IssuerAttribution) (principal, equal, possible int) {
	for _, attribution := range attributions {
		switch attribution.CausalRole {
		case CausalRolePrincipal:
			principal++
		case CausalRoleEqualPrincipal:
			equal++
		case CausalRolePossiblePrincipal:
			possible++
		}
	}
	return principal, equal, possible
}

func principalIssuer(attributions []IssuerAttribution) string {
	for _, attribution := range attributions {
		if attribution.CausalRole == CausalRolePrincipal {
			return strings.TrimSpace(attribution.Issuer)
		}
	}
	return ""
}

// ResolveCausalAttributionDecision invokes the existing resolver only after a
// valid typed policy decision has been produced. Resolver coverage does not
// change the effective semantic mapping.
func ResolveCausalAttributionDecision(decision CausalAttributionDecision, input EventInput, resolver assetresolution.Resolver) PolicyResolution {
	switch decision.EffectiveMapping.MappingStatus {
	case "DIRECT":
		issuer := decision.EffectiveMapping.DirectIssuer
		resolved := resolver.ResolveIssuer(assetresolution.IssuerInput{
			IssuerName: issuer, PublicationAt: input.PublicationTimestamp, ReceiptAt: input.ReceiptTimestamp,
		})
		return newPolicyResolution(resolved, issuer)
	case "PROXY":
		resolved, _ := resolver.ResolveProxyExposure(decision.EffectiveMapping.ProxyExposure)
		return newPolicyResolution(resolved, "")
	default:
		return PolicyResolution{
			Status: assetresolution.StatusUnresolved, PolicyVersion: resolver.Rules.Version,
			MappingType: "none", Relationship: "none",
			Reason: "typed causal-attribution policy abstained before deterministic asset resolution: " + decision.ReasonCode,
		}
	}
}
