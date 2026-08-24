package provider

import (
	"fmt"
	"mime"
	"strings"
	"unicode"
	"unicode/utf8"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	maxShortText = 256
	maxMediaType = 128
)

type ValidationError struct {
	Contract string
	Field    string
	Rule     string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("provider contract %s: %s %s", e.Contract, e.Field, e.Rule)
}

func invalid(contract, field, rule string) error {
	return &ValidationError{Contract: contract, Field: field, Rule: rule}
}

func (definition ProviderDefinition) Validate() error {
	const contract = "provider_definition"
	if definition.ContractVersion != ProviderDefinitionV1 {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", ProviderDefinitionV1))
	}
	if err := definition.Identity.Validate(); err != nil {
		return invalid(contract, "identity", err.Error())
	}
	if err := validateText(contract, "display_name", definition.DisplayName, maxShortText); err != nil {
		return err
	}
	if err := definition.AdapterVersion.Validate(); err != nil {
		return invalid(contract, "adapter_version", err.Error())
	}
	if definition.ProviderAPIVersion != nil {
		if err := definition.ProviderAPIVersion.Validate(); err != nil {
			return invalid(contract, "provider_api_version", err.Error())
		}
	}
	if len(definition.Capabilities) == 0 {
		return invalid(contract, "capabilities", "requires at least one declared capability")
	}
	seen := make(map[CapabilityID]struct{}, len(definition.Capabilities))
	for i, capability := range definition.Capabilities {
		field := fmt.Sprintf("capabilities[%d]", i)
		if err := capability.Validate(); err != nil {
			return invalid(contract, field, err.Error())
		}
		if _, exists := seen[capability.ID]; exists {
			return invalid(contract, field+".id", "duplicates an earlier capability declaration")
		}
		seen[capability.ID] = struct{}{}
	}
	return nil
}

func (capability Capability) Validate() error {
	const contract = "capability"
	if capability.ContractVersion != CapabilityContractV1 {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", CapabilityContractV1))
	}
	wantCategory, wantOutputs, ok := capabilitySpecification(capability.ID)
	if !ok {
		return invalid(contract, "id", "is not a supported Jax capability")
	}
	if capability.Category != wantCategory {
		return invalid(contract, "category", fmt.Sprintf("must be %q for capability %q", wantCategory, capability.ID))
	}
	switch capability.Support {
	case SupportSupported, SupportUnavailable, SupportDisabled:
	default:
		return invalid(contract, "support", "is not supported")
	}
	if err := capability.Raw.Validate(); err != nil {
		return invalid(contract, "raw", err.Error())
	}
	if err := capability.Authentication.Validate(); err != nil {
		return invalid(contract, "authentication", err.Error())
	}
	if err := capability.Operational.Validate(); err != nil {
		return invalid(contract, "operational", err.Error())
	}
	if len(capability.CanonicalOutputs) == 0 {
		return invalid(contract, "canonical_outputs", "requires at least one meaningful canonical output")
	}
	if len(capability.CanonicalOutputs) != len(wantOutputs) {
		return invalid(contract, "canonical_outputs", "must match the unambiguous Jax output mapping")
	}
	for i, output := range capability.CanonicalOutputs {
		if err := output.Validate(); err != nil {
			return invalid(contract, fmt.Sprintf("canonical_outputs[%d]", i), err.Error())
		}
		if output != wantOutputs[i] {
			return invalid(contract, fmt.Sprintf("canonical_outputs[%d]", i), "does not match the capability's Jax output mapping")
		}
		switch output.Kind {
		case canonical.ContractKindInstrument, canonical.ContractKindIssuer, canonical.ContractKindEvent,
			canonical.ContractKindEvidence, canonical.ContractKindObservation:
		default:
			return invalid(contract, fmt.Sprintf("canonical_outputs[%d].kind", i), "must be an external information family, not a derived Jax product")
		}
	}
	return nil
}

func (raw RawRepresentation) Validate() error {
	const contract = "raw_representation"
	if raw.Boundary != RawBoundaryProvider {
		return invalid(contract, "boundary", "must be PROVIDER_RAW; provider representations are not canonical")
	}
	switch raw.Format {
	case RawFormatJSONDocument, RawFormatStructuredMessage, RawFormatTabular, RawFormatBinary, RawFormatStreamMessage:
	default:
		return invalid(contract, "format", "is not supported")
	}
	if err := raw.Schema.Validate(); err != nil {
		return invalid(contract, "schema", err.Error())
	}
	if raw.MediaType != "" {
		if len(raw.MediaType) > maxMediaType || strings.TrimSpace(raw.MediaType) != raw.MediaType {
			return invalid(contract, "media_type", "must be a trimmed media type of bounded length")
		}
		if _, _, err := mime.ParseMediaType(raw.MediaType); err != nil {
			return invalid(contract, "media_type", "must be a valid media type")
		}
	}
	return nil
}

func (requirement AuthenticationRequirement) Validate() error {
	const contract = "authentication_requirement"
	switch requirement.Class {
	case AuthenticationNone, AuthenticationAPIKey, AuthenticationAPIKeyPair, AuthenticationAuthenticatedSession:
		return nil
	default:
		return invalid(contract, "class", "is not supported")
	}
}

func (semantics OperationalSemantics) Validate() error {
	const contract = "operational_semantics"
	if len(semantics.DeliveryModes) == 0 {
		return invalid(contract, "delivery_modes", "requires at least one delivery mode")
	}
	seenDelivery := make(map[DeliveryMode]struct{}, len(semantics.DeliveryModes))
	for i, mode := range semantics.DeliveryModes {
		switch mode {
		case DeliverySnapshot, DeliveryHistorical, DeliveryStream, DeliveryEvent:
		default:
			return invalid(contract, fmt.Sprintf("delivery_modes[%d]", i), "is not supported")
		}
		if _, exists := seenDelivery[mode]; exists {
			return invalid(contract, fmt.Sprintf("delivery_modes[%d]", i), "duplicates an earlier delivery mode")
		}
		seenDelivery[mode] = struct{}{}
	}
	if len(semantics.FreshnessModes) == 0 {
		return invalid(contract, "freshness_modes", "requires at least one freshness mode")
	}
	seenFreshness := make(map[FreshnessMode]struct{}, len(semantics.FreshnessModes))
	for i, mode := range semantics.FreshnessModes {
		switch mode {
		case FreshnessRealTime, FreshnessDelayed, FreshnessEndOfDay, FreshnessEventDriven, FreshnessPeriodic, FreshnessOnDemand:
		default:
			return invalid(contract, fmt.Sprintf("freshness_modes[%d]", i), "is not supported")
		}
		if _, exists := seenFreshness[mode]; exists {
			return invalid(contract, fmt.Sprintf("freshness_modes[%d]", i), "duplicates an earlier freshness mode")
		}
		seenFreshness[mode] = struct{}{}
	}
	if semantics.QualityRequirement != QualityCanonicalValidationRequired {
		return invalid(contract, "quality_requirement", "must require canonical validation")
	}
	return nil
}

func (state CapabilityRuntimeState) Validate() error {
	const contract = "capability_runtime_state"
	if state.ContractVersion != CapabilityRuntimeStateV1 {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", CapabilityRuntimeStateV1))
	}
	if err := state.Provider.Validate(); err != nil {
		return invalid(contract, "provider", err.Error())
	}
	if _, _, ok := capabilitySpecification(state.CapabilityID); !ok {
		return invalid(contract, "capability_id", "is not a supported Jax capability")
	}
	switch state.Status {
	case RuntimeUnknown, RuntimeHealthy, RuntimeDegraded, RuntimeUnavailable, RuntimeDisabled:
	default:
		return invalid(contract, "status", "is not supported")
	}
	switch state.Freshness {
	case FreshnessUnknown, FreshnessFresh, FreshnessStale:
	default:
		return invalid(contract, "freshness", "is not supported")
	}
	switch state.Quality {
	case QualityUnknown, QualityAcceptable, QualityDegraded, QualityRejected:
	default:
		return invalid(contract, "quality", "is not supported")
	}
	_, offset := state.ObservedAt.Zone()
	if state.ObservedAt.IsZero() || offset != 0 || state.ObservedAt.Year() < 0 || state.ObservedAt.Year() > 9999 {
		return invalid(contract, "observed_at", "is required and must use UTC")
	}
	if state.ReasonCode != "" {
		if err := validateCode(contract, "reason_code", state.ReasonCode); err != nil {
			return err
		}
	}
	switch state.Status {
	case RuntimeHealthy:
		if state.Freshness != FreshnessFresh || state.Quality != QualityAcceptable || state.ReasonCode != "" {
			return invalid(contract, "status", "HEALTHY requires FRESH, ACCEPTABLE, and no reason code")
		}
	case RuntimeUnknown:
		if state.Freshness != FreshnessUnknown || state.Quality != QualityUnknown {
			return invalid(contract, "status", "UNKNOWN requires unknown freshness and quality")
		}
	case RuntimeDisabled:
		if state.Freshness != FreshnessUnknown || state.Quality != QualityUnknown || state.ReasonCode == "" {
			return invalid(contract, "status", "DISABLED requires unknown freshness/quality and a reason code")
		}
	case RuntimeUnavailable:
		if state.ReasonCode == "" {
			return invalid(contract, "reason_code", "is required when unavailable")
		}
		if state.Freshness == FreshnessFresh && state.Quality == QualityAcceptable {
			return invalid(contract, "status", "UNAVAILABLE contradicts fresh acceptable data")
		}
	case RuntimeDegraded:
		if state.ReasonCode == "" {
			return invalid(contract, "reason_code", "is required when degraded")
		}
		if state.Quality == QualityRejected {
			return invalid(contract, "quality", "REJECTED data must not be represented as a usable degraded capability")
		}
	}
	return nil
}

// ValidateRuntimeState binds a separately supplied operational snapshot to a
// registered static definition without storing it in the registry.
func ValidateRuntimeState(definition ProviderDefinition, state CapabilityRuntimeState) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	if err := state.Validate(); err != nil {
		return err
	}
	if !sameProviderIdentity(definition.Identity, state.Provider) {
		return invalid("capability_runtime_state", "provider", "does not match the provider definition")
	}
	var declared *Capability
	for i := range definition.Capabilities {
		if definition.Capabilities[i].ID == state.CapabilityID {
			declared = &definition.Capabilities[i]
			break
		}
	}
	if declared == nil {
		return invalid("capability_runtime_state", "capability_id", "is not declared by the provider")
	}
	switch declared.Support {
	case SupportUnavailable:
		if state.Status != RuntimeUnavailable && state.Status != RuntimeUnknown {
			return invalid("capability_runtime_state", "status", "contradicts static UNAVAILABLE support")
		}
	case SupportDisabled:
		if state.Status != RuntimeDisabled && state.Status != RuntimeUnknown {
			return invalid("capability_runtime_state", "status", "contradicts static DISABLED support")
		}
	}
	return nil
}

func capabilitySpecification(id CapabilityID) (DataCategory, []canonical.ContractSchemaRef, bool) {
	outputs := func(values ...canonical.ContractSchemaRef) []canonical.ContractSchemaRef { return values }
	switch id {
	case CapabilityInstrumentReference:
		return DataCategoryReferenceData, outputs(
			canonical.ContractSchemaRef{Kind: canonical.ContractKindInstrument, Version: canonical.InstrumentContractV1},
			canonical.ContractSchemaRef{Kind: canonical.ContractKindIssuer, Version: canonical.IssuerContractV1},
		), true
	case CapabilityMarketQuote, CapabilityMarketBars, CapabilityMarketTrades:
		return DataCategoryMarketData, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}), true
	case CapabilityCorporateEarnings:
		return DataCategoryCorporateData, outputs(
			canonical.ContractSchemaRef{Kind: canonical.ContractKindEvent, Version: canonical.EventContractV1},
			canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2},
		), true
	case CapabilityCorporateFiling:
		return DataCategoryRegulatoryFiling, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2}), true
	case CapabilityFundamentalObservation:
		return DataCategoryFundamentals, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}), true
	case CapabilityNewsArticle:
		return DataCategoryNewsEvidence, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2}), true
	case CapabilityEventFeed:
		return DataCategoryEventEvidence, outputs(
			canonical.ContractSchemaRef{Kind: canonical.ContractKindEvent, Version: canonical.EventContractV1},
			canonical.ContractSchemaRef{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2},
		), true
	case CapabilityMacroObservation:
		return DataCategoryMacroeconomicData, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}), true
	case CapabilityEconomicCalendar:
		return DataCategoryEconomicCalendar, outputs(canonical.ContractSchemaRef{Kind: canonical.ContractKindEvent, Version: canonical.EventContractV1}), true
	default:
		return "", nil, false
	}
}

// CanonicalOutputsFor returns a copy of the normative output mapping for a
// supported Jax capability.
func CanonicalOutputsFor(id CapabilityID) ([]canonical.ContractSchemaRef, bool) {
	_, values, ok := capabilitySpecification(id)
	return append([]canonical.ContractSchemaRef(nil), values...), ok
}

func validateText(contract, field, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return invalid(contract, field, "is required")
	}
	if strings.TrimSpace(value) != value {
		return invalid(contract, field, "must not have surrounding whitespace")
	}
	if len(value) > maximum || !utf8.ValidString(value) {
		return invalid(contract, field, "must be bounded valid UTF-8")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return invalid(contract, field, "contains a control character")
		}
	}
	return nil
}

func validateCode(contract, field, value string) error {
	if err := validateText(contract, field, value, maxShortText); err != nil {
		return err
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (i > 0 && (r == '.' || r == '_' || r == '-')) {
			continue
		}
		return invalid(contract, field, "must use lower-case letters, digits, '.', '_' or '-'")
	}
	return nil
}

func sameProviderIdentity(left, right canonical.ProviderIdentity) bool {
	if left.ID != right.ID || left.Namespace != right.Namespace {
		return false
	}
	if left.ExternalID == nil || right.ExternalID == nil {
		return left.ExternalID == nil && right.ExternalID == nil
	}
	return *left.ExternalID == *right.ExternalID
}
