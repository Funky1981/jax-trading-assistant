package provider

import (
	"errors"
	"reflect"
	"testing"

	"jax-trading-assistant/libs/contracts/canonical"
)

func TestRegistryRegisterLookupAndListAreDeterministic(t *testing.T) {
	registry := mustRegistry(t)
	second := validDefinition(CapabilityNewsArticle)
	second.Identity = canonical.ProviderIdentity{ID: "pvd_news_wire", Namespace: "news.wire"}
	second.DisplayName = "News Wire"
	if err := registry.Register(second); err != nil {
		t.Fatalf("Register(second) error = %v", err)
	}
	first := validDefinition(CapabilityMarketBars)
	if err := registry.Register(first); err != nil {
		t.Fatalf("Register(first) error = %v", err)
	}

	listed := registry.List()
	if len(listed) != 2 || listed[0].Identity.ID != "pvd_market_data" || listed[1].Identity.ID != "pvd_news_wire" {
		t.Fatalf("List() order = %#v", listed)
	}
	got, err := registry.Lookup(first.Identity)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("Lookup() = %#v, want %#v", got, first)
	}
	got.Capabilities[0].CanonicalOutputs[0].Version = canonical.ObservationContractV1
	again, err := registry.Lookup(first.Identity)
	if err != nil {
		t.Fatalf("second Lookup() error = %v", err)
	}
	if again.Capabilities[0].CanonicalOutputs[0].Version != canonical.ObservationContractV2 {
		t.Fatal("lookup result mutated the registered definition")
	}
	if registry.ContractVersion() != RegistryContractV1 {
		t.Fatalf("ContractVersion() = %q", registry.ContractVersion())
	}
}

func TestRegistryDistinguishesUnknownUnavailableAndIdentityMismatch(t *testing.T) {
	registry := mustRegistry(t)
	definition := validDefinition(CapabilityMarketTrades)
	definition.Capabilities[0].Support = SupportUnavailable
	if err := registry.Register(definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	capability, err := registry.Capability(definition.Identity, CapabilityMarketTrades)
	if err != nil {
		t.Fatalf("Capability() error = %v", err)
	}
	if capability.Support != SupportUnavailable {
		t.Fatalf("support = %q, want %q", capability.Support, SupportUnavailable)
	}

	unknown := definition.Identity
	unknown.ID = "pvd_unknown"
	_, err = registry.Lookup(unknown)
	assertErrorCode(t, err, ErrorUnknownProvider)

	invalid := definition.Identity
	invalid.ID = "not-a-provider"
	_, err = registry.Lookup(invalid)
	assertErrorCode(t, err, ErrorInvalidProviderIdentity)

	mismatch := definition.Identity
	mismatch.Namespace = "different.namespace"
	_, err = registry.Lookup(mismatch)
	assertErrorCode(t, err, ErrorProviderIdentityMismatch)

	externalMismatch := definition.Identity
	external := *externalMismatch.ExternalID
	external.Value = "different-provider"
	externalMismatch.ExternalID = &external
	_, err = registry.Lookup(externalMismatch)
	assertErrorCode(t, err, ErrorProviderIdentityMismatch)

	_, err = registry.Capability(definition.Identity, CapabilityNewsArticle)
	assertErrorCode(t, err, ErrorUnknownCapability)
}

func TestRegistryRejectsDuplicateAndAmbiguousRegistration(t *testing.T) {
	registry := mustRegistry(t)
	definition := validDefinition(CapabilityMarketQuote)
	if err := registry.Register(definition); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	assertErrorCode(t, registry.Register(definition), ErrorDuplicateProvider)

	ambiguous := definition
	ambiguous.Identity.ID = "pvd_market_data_alias"
	assertErrorCode(t, registry.Register(ambiguous), ErrorAmbiguousProvider)
}

func TestRegistryRejectsInvalidDefinitionWithoutPartialRegistration(t *testing.T) {
	registry := mustRegistry(t)
	definition := validDefinition(CapabilityMarketBars)
	definition.Capabilities = append(definition.Capabilities, definition.Capabilities[0])
	assertErrorCode(t, registry.Register(definition), ErrorInvalidDefinition)
	if len(registry.List()) != 0 {
		t.Fatal("invalid registration was partially retained")
	}
}

func TestUnsupportedRegistryVersionFailsClosed(t *testing.T) {
	registry, err := NewRegistry("jax.provider_registry/v99")
	if err == nil || registry != nil {
		t.Fatalf("NewRegistry() = %#v, %v; want nil error result", registry, err)
	}
	assertErrorCode(t, err, ErrorUnsupportedRegistryVersion)
}

func mustRegistry(t *testing.T) *Registry {
	t.Helper()
	registry, err := NewRegistry(RegistryContractV1)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}

func assertErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want registry code %q", want)
	}
	var registryErr *RegistryError
	if !errors.As(err, &registryErr) {
		t.Fatalf("error type = %T, want *RegistryError", err)
	}
	if registryErr.Code != want {
		t.Fatalf("error code = %q, want %q", registryErr.Code, want)
	}
}
