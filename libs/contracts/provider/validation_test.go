package provider

import (
	"errors"
	"testing"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

func TestCapabilityVocabularyHasValidUnambiguousCanonicalMappings(t *testing.T) {
	ids := []CapabilityID{
		CapabilityInstrumentReference,
		CapabilityMarketQuote,
		CapabilityMarketBars,
		CapabilityMarketTrades,
		CapabilityCorporateEarnings,
		CapabilityCorporateFiling,
		CapabilityFundamentalObservation,
		CapabilityNewsArticle,
		CapabilityEventFeed,
		CapabilityMacroObservation,
		CapabilityEconomicCalendar,
	}
	for _, id := range ids {
		t.Run(string(id), func(t *testing.T) {
			capability := validCapability(id)
			if err := capability.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
			for _, output := range capability.CanonicalOutputs {
				switch output.Kind {
				case canonical.ContractKindResearchRun, canonical.ContractKindQuantResult, canonical.ContractKindRecommendation:
					t.Fatalf("external capability maps to derived Jax family %q", output.Kind)
				}
				if (output.Kind == canonical.ContractKindEvidence && output.Version != canonical.EvidenceContractV2) ||
					(output.Kind == canonical.ContractKindObservation && output.Version != canonical.ObservationContractV2) {
					t.Fatalf("provenance-bearing output uses unsupported phase version: %#v", output)
				}
			}
		})
	}
}

func TestCapabilityValidationFailsClosed(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Capability)
	}{
		{"unsupported_contract_version", func(c *Capability) { c.ContractVersion = "jax.provider_capability/v99" }},
		{"empty_capability", func(c *Capability) { c.ID = "" }},
		{"unknown_capability", func(c *Capability) { c.ID = "vendor.get_prices" }},
		{"wrong_category", func(c *Capability) { c.Category = DataCategoryNewsEvidence }},
		{"unknown_support", func(c *Capability) { c.Support = "MAYBE" }},
		{"canonical_boundary_masquerading_as_raw", func(c *Capability) { c.Raw.Boundary = "CANONICAL" }},
		{"missing_raw_schema", func(c *Capability) { c.Raw.Schema = canonical.VersionIdentity{} }},
		{"invalid_auth", func(c *Capability) { c.Authentication.Class = "PASSWORD" }},
		{"missing_delivery_semantics", func(c *Capability) { c.Operational.DeliveryModes = nil }},
		{"unknown_freshness_semantics", func(c *Capability) { c.Operational.FreshnessModes = []FreshnessMode{"MAGICAL"} }},
		{"no_output", func(c *Capability) { c.CanonicalOutputs = nil }},
		{"ambiguous_extra_output", func(c *Capability) {
			c.CanonicalOutputs = append(c.CanonicalOutputs, canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2})
		}},
		{"wrong_output_version", func(c *Capability) { c.CanonicalOutputs[0].Version = canonical.ObservationContractV1 }},
		{"derived_output", func(c *Capability) {
			c.CanonicalOutputs[0] = canonical.ContractSchemaRef{Kind: canonical.ContractKindRecommendation, Version: canonical.RecommendationContractV2}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			capability := validCapability(CapabilityMarketBars)
			test.edit(&capability)
			assertValidationError(t, capability.Validate())
		})
	}
}

func TestProviderDefinitionValidationRejectsInvalidIdentityVersionsAndDuplicates(t *testing.T) {
	tests := []struct {
		name string
		edit func(*ProviderDefinition)
	}{
		{"unsupported_contract_version", func(d *ProviderDefinition) { d.ContractVersion = "jax.provider_definition/v2" }},
		{"empty_provider_id", func(d *ProviderDefinition) { d.Identity.ID = "" }},
		{"invalid_provider_namespace", func(d *ProviderDefinition) { d.Identity.Namespace = "Market Data" }},
		{"missing_display_name", func(d *ProviderDefinition) { d.DisplayName = "" }},
		{"missing_adapter_version", func(d *ProviderDefinition) { d.AdapterVersion = canonical.VersionIdentity{} }},
		{"invalid_provider_api_version", func(d *ProviderDefinition) { d.ProviderAPIVersion = &canonical.VersionIdentity{} }},
		{"no_capabilities", func(d *ProviderDefinition) { d.Capabilities = nil }},
		{"duplicate_capability", func(d *ProviderDefinition) { d.Capabilities = append(d.Capabilities, d.Capabilities[0]) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := validDefinition(CapabilityMarketQuote)
			test.edit(&definition)
			assertValidationError(t, definition.Validate())
		})
	}
}

func TestRuntimeStateIsSeparateAndRejectsContradictions(t *testing.T) {
	definition := validDefinition(CapabilityMarketQuote)
	state := validRuntimeState(definition, CapabilityMarketQuote)
	if err := ValidateRuntimeState(definition, state); err != nil {
		t.Fatalf("ValidateRuntimeState() error = %v", err)
	}

	tests := []struct {
		name           string
		editDefinition func(*ProviderDefinition)
		editState      func(*CapabilityRuntimeState)
	}{
		{"provider_mismatch", nil, func(s *CapabilityRuntimeState) { s.Provider.ID = "pvd_other" }},
		{"undeclared_capability", nil, func(s *CapabilityRuntimeState) { s.CapabilityID = CapabilityNewsArticle }},
		{"healthy_but_stale", nil, func(s *CapabilityRuntimeState) { s.Freshness = FreshnessStale }},
		{"healthy_but_degraded_quality", nil, func(s *CapabilityRuntimeState) { s.Quality = QualityDegraded }},
		{"unavailable_without_reason", nil, func(s *CapabilityRuntimeState) {
			s.Status = RuntimeUnavailable
			s.Freshness = FreshnessUnknown
			s.Quality = QualityUnknown
		}},
		{"degraded_with_rejected_quality", nil, func(s *CapabilityRuntimeState) {
			s.Status = RuntimeDegraded
			s.Quality = QualityRejected
			s.ReasonCode = "invalid_rows"
		}},
		{"static_unavailable_runtime_healthy", func(d *ProviderDefinition) { d.Capabilities[0].Support = SupportUnavailable }, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidateDefinition := definition
			candidateDefinition.Capabilities = append([]Capability(nil), definition.Capabilities...)
			candidateState := state
			if test.editDefinition != nil {
				test.editDefinition(&candidateDefinition)
			}
			if test.editState != nil {
				test.editState(&candidateState)
			}
			assertValidationError(t, ValidateRuntimeState(candidateDefinition, candidateState))
		})
	}
}

func validDefinition(id CapabilityID) ProviderDefinition {
	apiVersion := canonical.VersionIdentity{Namespace: "provider.api", Value: "2026-08"}
	return ProviderDefinition{
		ContractVersion: ProviderDefinitionV1,
		Identity: canonical.ProviderIdentity{
			ID:         "pvd_market_data",
			Namespace:  "market.data",
			ExternalID: &canonical.ExternalID{Namespace: "vendor.registry", Value: "market-data"},
		},
		DisplayName:        "Market Data Provider",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "git.commit", Value: "a9bcbd12"},
		ProviderAPIVersion: &apiVersion,
		Capabilities:       []Capability{validCapability(id)},
	}
}

func validCapability(id CapabilityID) Capability {
	category, _, ok := capabilitySpecification(id)
	if !ok {
		panic("invalid test capability " + id)
	}
	outputs, _ := CanonicalOutputsFor(id)
	return Capability{
		ContractVersion: CapabilityContractV1,
		ID:              id,
		Category:        category,
		Support:         SupportSupported,
		Raw: RawRepresentation{
			Boundary:  RawBoundaryProvider,
			Format:    RawFormatJSONDocument,
			Schema:    canonical.VersionIdentity{Namespace: "provider.schema", Value: string(id) + "/v1"},
			MediaType: "application/json",
		},
		Authentication: AuthenticationRequirement{Class: AuthenticationAPIKey},
		Operational: OperationalSemantics{
			DeliveryModes:      []DeliveryMode{DeliverySnapshot},
			FreshnessModes:     []FreshnessMode{FreshnessDelayed},
			QualityRequirement: QualityCanonicalValidationRequired,
		},
		CanonicalOutputs: outputs,
	}
}

func validRuntimeState(definition ProviderDefinition, capabilityID CapabilityID) CapabilityRuntimeState {
	return CapabilityRuntimeState{
		ContractVersion: CapabilityRuntimeStateV1,
		Provider:        definition.Identity,
		CapabilityID:    capabilityID,
		Status:          RuntimeHealthy,
		Freshness:       FreshnessFresh,
		Quality:         QualityAcceptable,
		ObservedAt:      time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
	}
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Validate() accepted invalid value")
	}
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *ValidationError: %v", err, err)
	}
}
