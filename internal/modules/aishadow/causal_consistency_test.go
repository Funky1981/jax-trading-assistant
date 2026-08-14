package aishadow

import (
	"reflect"
	"strings"
	"testing"

	"jax-trading-assistant/internal/modules/assetresolution"
)

func TestCausalConsistencyGuardSyntheticBehavior(t *testing.T) {
	resolver := testAssetResolver(t)

	t.Run("allowed DIRECT remains DIRECT", func(t *testing.T) {
		raw, guard, resolution, errs := ParseValidateAndGuard(directJSON("Apple"), testInput("Issuer raises guidance", "company"), resolver)
		if len(errs) != 0 || raw == nil || guard == nil || resolution == nil {
			t.Fatalf("unexpected result: raw=%+v guard=%+v resolution=%+v errors=%v", raw, guard, resolution, errs)
		}
		if guard.Abstained || guard.EffectiveMapping.MappingStatus != "DIRECT" || resolution.ResolvedTicker != "AAPL" {
			t.Fatalf("safe DIRECT changed: guard=%+v resolution=%+v", guard, resolution)
		}
	})

	t.Run("unsafe DIRECT abstains", func(t *testing.T) {
		payload := strings.Replace(directJSON("Apple"), `"market_relevance":"HIGH"`, `"market_relevance":"MEDIUM"`, 1)
		raw, guard, resolution, errs := ParseValidateAndGuard(payload, testInput("Issuer event with bounded relevance", "company"), resolver)
		if len(errs) != 0 || raw == nil || guard == nil || resolution == nil {
			t.Fatalf("unexpected result: raw=%+v guard=%+v resolution=%+v errors=%v", raw, guard, resolution, errs)
		}
		if raw.MappingStatus != "DIRECT" || raw.DirectIssuer != "Apple" {
			t.Fatalf("raw output mutated: %+v", raw)
		}
		if !guard.Abstained || guard.EffectiveMapping != (AssetMapping{MappingStatus: "UNRESOLVED", ProxyExposure: NoProxyExposure}) || !reflect.DeepEqual(guard.ReasonCodes, []string{ReasonDirectRelevanceNotHigh}) {
			t.Fatalf("unsafe DIRECT was not deterministically abstained: %+v", guard)
		}
		if resolution.Status != assetresolution.StatusUnresolved || resolution.ResolvedTicker != "" {
			t.Fatalf("resolver ran as DIRECT after abstention: %+v", resolution)
		}
	})

	t.Run("allowed PROXY remains PROXY", func(t *testing.T) {
		input := testInput("Broad semiconductor restrictions", "technology")
		input.Entities = []string{"NVIDIA", "AMD"}
		_, guard, resolution, errs := ParseValidateAndGuard(proxyJSON("SEMICONDUCTOR_GROUP"), input, resolver)
		if len(errs) != 0 || guard == nil || resolution == nil {
			t.Fatalf("unexpected result: guard=%+v resolution=%+v errors=%v", guard, resolution, errs)
		}
		if guard.Abstained || guard.EffectiveMapping.MappingStatus != "PROXY" || resolution.ResolvedTicker != "SOXX" {
			t.Fatalf("non-company PROXY changed: guard=%+v resolution=%+v", guard, resolution)
		}
	})

	t.Run("unsafe PROXY abstains", func(t *testing.T) {
		input := testInput("Two issuers announce competing products", "company")
		input.Entities = []string{"NVIDIA", "AMD"}
		raw, guard, resolution, errs := ParseValidateAndGuard(proxyJSON("SEMICONDUCTOR_GROUP"), input, resolver)
		if len(errs) != 0 || raw == nil || guard == nil || resolution == nil {
			t.Fatalf("unexpected result: raw=%+v guard=%+v resolution=%+v errors=%v", raw, guard, resolution, errs)
		}
		if raw.MappingStatus != "PROXY" || raw.ProxyExposure != "SEMICONDUCTOR_GROUP" {
			t.Fatalf("raw output mutated: %+v", raw)
		}
		if !guard.Abstained || guard.EffectiveMapping.MappingStatus != "UNRESOLVED" || !reflect.DeepEqual(guard.ReasonCodes, []string{ReasonProxyCompetingIssuerInput}) || guard.RecognizedInputIssuerCount != 2 {
			t.Fatalf("unsafe PROXY was not deterministically abstained: %+v", guard)
		}
		if resolution.Status != assetresolution.StatusUnresolved || resolution.ResolvedTicker != "" {
			t.Fatalf("resolver ran as PROXY after abstention: %+v", resolution)
		}
	})

	t.Run("UNRESOLVED remains UNRESOLVED", func(t *testing.T) {
		raw, guard, resolution, errs := ParseValidateAndGuard(unresolvedJSON(), testInput("No bounded mapping", "company"), resolver)
		if len(errs) != 0 || raw == nil || guard == nil || resolution == nil {
			t.Fatalf("unexpected result: raw=%+v guard=%+v resolution=%+v errors=%v", raw, guard, resolution, errs)
		}
		if guard.Abstained || guard.RawMapping != guard.EffectiveMapping || guard.EffectiveMapping.MappingStatus != "UNRESOLVED" || resolution.Status != assetresolution.StatusUnresolved {
			t.Fatalf("UNRESOLVED changed: guard=%+v resolution=%+v", guard, resolution)
		}
	})
}

func TestCausalConsistencyGuardCannotPromoteMappings(t *testing.T) {
	resolver := testAssetResolver(t)
	input := testInput("Synthetic event", "company")
	input.Entities = []string{"NVIDIA", "AMD"}
	cases := []StructuredResult{
		{MarketRelevance: "HIGH", MappingStatus: "DIRECT", DirectIssuer: "Apple", ProxyExposure: NoProxyExposure},
		{MarketRelevance: "LOW", MappingStatus: "DIRECT", DirectIssuer: "Apple", ProxyExposure: NoProxyExposure},
		{MarketRelevance: "HIGH", MappingStatus: "PROXY", ProxyExposure: "SEMICONDUCTOR_GROUP"},
		{MarketRelevance: "LOW", MappingStatus: "UNRESOLVED", ProxyExposure: NoProxyExposure},
	}
	for _, raw := range cases {
		decision := ApplyCausalConsistencyGuard(raw, input, resolver)
		allowed := decision.EffectiveMapping == decision.RawMapping || decision.EffectiveMapping == (AssetMapping{MappingStatus: "UNRESOLVED", ProxyExposure: NoProxyExposure})
		if !allowed {
			t.Fatalf("guard made a non-monotonic transition: raw=%+v effective=%+v", decision.RawMapping, decision.EffectiveMapping)
		}
	}
}

func TestCausalConsistencyGuardPreservesRawAndIsDeterministic(t *testing.T) {
	resolver := testAssetResolver(t)
	input := testInput("Two issuers announce competing products", "company")
	input.Entities = []string{"AMD", "NVIDIA", "NVIDIA"}
	raw := StructuredResult{
		MarketRelevance: "HIGH", MappingStatus: "PROXY", ProxyExposure: "SEMICONDUCTOR_GROUP",
		MappingConfidence: "HIGH", ExpectedHorizon: "ONE_DAY", LikelyDirection: "NEGATIVE",
		CatalystType: "product competition", Reason: "Two named issuers have equally prominent product announcements.",
		MissingEvidence: []string{"Principal issuer"},
	}
	before := raw
	before.MissingEvidence = append([]string(nil), raw.MissingEvidence...)
	want := ApplyCausalConsistencyGuard(raw, input, resolver)
	for i := 0; i < 50; i++ {
		got := ApplyCausalConsistencyGuard(raw, input, resolver)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("execution %d was non-deterministic: got=%+v want=%+v", i, got, want)
		}
	}
	if !reflect.DeepEqual(raw, before) {
		t.Fatalf("raw model output mutated: got=%+v want=%+v", raw, before)
	}
	if want.RawMapping == want.EffectiveMapping || want.RawMapping.MappingStatus != "PROXY" || want.EffectiveMapping.MappingStatus != "UNRESOLVED" {
		t.Fatalf("raw and effective mappings were not independently represented: %+v", want)
	}
	if want.RecognizedInputIssuerCount != 2 {
		t.Fatalf("recognized issuer identities are not unique: %+v", want)
	}
}
