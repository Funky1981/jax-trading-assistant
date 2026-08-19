package aishadow

import (
	"sort"
	"strings"

	"jax-trading-assistant/internal/modules/assetresolution"
)

const IssuerSemanticIdentityVersion = "ai-shadow-issuer-semantic-identity-v1"

type IssuerIdentityOutcome string

const (
	IssuerIdentityExact      IssuerIdentityOutcome = "EXACT"
	IssuerIdentityEquivalent IssuerIdentityOutcome = "EQUIVALENT"
	IssuerIdentityDistinct   IssuerIdentityOutcome = "DISTINCT"
	IssuerIdentityAmbiguous  IssuerIdentityOutcome = "AMBIGUOUS"
)

type IssuerIdentityComparison struct {
	Version  string                `json:"version"`
	Outcome  IssuerIdentityOutcome `json:"outcome"`
	LeftKey  string                `json:"left_key"`
	RightKey string                `json:"right_key"`
	Reason   string                `json:"reason"`
}

type IssuerSemanticIdentity struct {
	bySurface map[string][]string
	byLegal   map[string][]string
}

func NewIssuerSemanticIdentity(rules assetresolution.Ruleset) IssuerSemanticIdentity {
	identity := IssuerSemanticIdentity{bySurface: map[string][]string{}, byLegal: map[string][]string{}}
	for _, rule := range rules.Aliases {
		canonical := assetresolution.CanonicalizeIssuerName(rule.CanonicalEntity)
		identity.addSurface(canonical, canonical)
		identity.addLegal(legalSurfaceKey(canonical), canonical)
		for _, alias := range rule.Aliases {
			identity.addSurface(assetresolution.CanonicalizeIssuerName(alias), canonical)
		}
	}
	return identity
}

func (i IssuerSemanticIdentity) Compare(left, right string) IssuerIdentityComparison {
	l := assetresolution.CanonicalizeIssuerName(left)
	r := assetresolution.CanonicalizeIssuerName(right)
	result := IssuerIdentityComparison{Version: IssuerSemanticIdentityVersion, LeftKey: l, RightKey: r}
	if l == "" || r == "" {
		result.Outcome, result.Reason = IssuerIdentityDistinct, "empty issuer identities do not match"
		return result
	}
	lCandidates := i.candidates(l)
	rCandidates := i.candidates(r)
	if len(lCandidates) > 1 || len(rCandidates) > 1 {
		result.Outcome, result.Reason = IssuerIdentityAmbiguous, "a surface form maps to multiple explicit semantic issuers"
		return result
	}
	if l == r {
		result.Outcome, result.Reason = IssuerIdentityExact, "deterministically normalized surface forms are exact"
		return result
	}
	if len(lCandidates) == 1 && len(rCandidates) == 1 && lCandidates[0] == rCandidates[0] {
		result.Outcome, result.Reason = IssuerIdentityEquivalent, "explicit resolver alias metadata identifies one semantic issuer"
		return result
	}
	lLegal, rLegal := legalSurfaceKey(l), legalSurfaceKey(r)
	if lLegal != "" && lLegal == rLegal {
		known := i.byLegal[lLegal]
		if len(known) > 1 {
			result.Outcome, result.Reason = IssuerIdentityAmbiguous, "legal-format normalization collides across semantic issuers"
			return result
		}
		result.Outcome, result.Reason = IssuerIdentityEquivalent, "surface forms differ only by mechanically safe legal formatting"
		return result
	}
	result.Outcome, result.Reason = IssuerIdentityDistinct, "no explicit alias or collision-free legal-format equivalence exists"
	return result
}

func (i IssuerSemanticIdentity) Equivalent(left, right string) bool {
	o := i.Compare(left, right).Outcome
	return o == IssuerIdentityExact || o == IssuerIdentityEquivalent
}

func (i *IssuerSemanticIdentity) addSurface(surface, canonical string) {
	if surface == "" || canonical == "" {
		return
	}
	i.bySurface[surface] = appendUniqueSorted(i.bySurface[surface], canonical)
}

func (i *IssuerSemanticIdentity) addLegal(key, canonical string) {
	if key == "" || canonical == "" {
		return
	}
	i.byLegal[key] = appendUniqueSorted(i.byLegal[key], canonical)
}

func (i IssuerSemanticIdentity) candidates(surface string) []string {
	if candidates := i.bySurface[surface]; len(candidates) > 0 {
		return candidates
	}
	return i.byLegal[legalSurfaceKey(surface)]
}

func appendUniqueSorted(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	values = append(values, value)
	sort.Strings(values)
	return values
}

func legalSurfaceKey(normalized string) string {
	parts := strings.Fields(normalized)
	for len(parts) > 1 && legalSuffixes[parts[len(parts)-1]] {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, " ")
}

var legalSuffixes = map[string]bool{
	"inc": true, "incorporated": true, "corporation": true, "corp": true,
	"company": true, "co": true, "limited": true, "ltd": true, "plc": true,
	"ag": true, "nv": true, "sa": true, "as": true,
}
