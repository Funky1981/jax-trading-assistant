package provider

import (
	"fmt"
	"sort"

	"jax-trading-assistant/libs/contracts/canonical"
)

type ErrorCode string

const (
	ErrorUnsupportedRegistryVersion ErrorCode = "unsupported_registry_version"
	ErrorInvalidDefinition          ErrorCode = "invalid_definition"
	ErrorDuplicateProvider          ErrorCode = "duplicate_provider"
	ErrorAmbiguousProvider          ErrorCode = "ambiguous_provider"
	ErrorUnknownProvider            ErrorCode = "unknown_provider"
	ErrorInvalidProviderIdentity    ErrorCode = "invalid_provider_identity"
	ErrorProviderIdentityMismatch   ErrorCode = "provider_identity_mismatch"
	ErrorUnknownCapability          ErrorCode = "unknown_capability"
)

type RegistryError struct {
	Code         ErrorCode
	ProviderID   string
	CapabilityID CapabilityID
	Detail       string
}

func (e *RegistryError) Error() string {
	return fmt.Sprintf("provider registry %s: provider=%q capability=%q %s", e.Code, e.ProviderID, e.CapabilityID, e.Detail)
}

// Registry is a deterministic in-memory catalogue. It owns no discovery,
// runtime configuration, health collection, network client, or persistence.
type Registry struct {
	contractVersion canonical.ContractVersion
	providers       map[string]ProviderDefinition
	namespaces      map[string]string
}

func NewRegistry(version canonical.ContractVersion) (*Registry, error) {
	if version != RegistryContractV1 {
		return nil, &RegistryError{Code: ErrorUnsupportedRegistryVersion, Detail: fmt.Sprintf("version must be %q", RegistryContractV1)}
	}
	return &Registry{
		contractVersion: version,
		providers:       make(map[string]ProviderDefinition),
		namespaces:      make(map[string]string),
	}, nil
}

func (r *Registry) ContractVersion() canonical.ContractVersion {
	if r == nil {
		return ""
	}
	return r.contractVersion
}

func (r *Registry) Register(definition ProviderDefinition) error {
	if r == nil || r.contractVersion != RegistryContractV1 {
		return &RegistryError{Code: ErrorUnsupportedRegistryVersion, ProviderID: definition.Identity.ID, Detail: "registry is nil or invalid"}
	}
	if err := definition.Validate(); err != nil {
		return &RegistryError{Code: ErrorInvalidDefinition, ProviderID: definition.Identity.ID, Detail: err.Error()}
	}
	id := definition.Identity.ID
	if _, exists := r.providers[id]; exists {
		return &RegistryError{Code: ErrorDuplicateProvider, ProviderID: id, Detail: "provider ID is already registered"}
	}
	if existingID, exists := r.namespaces[definition.Identity.Namespace]; exists {
		return &RegistryError{Code: ErrorAmbiguousProvider, ProviderID: id, Detail: fmt.Sprintf("namespace %q is already owned by %q", definition.Identity.Namespace, existingID)}
	}
	r.providers[id] = cloneDefinition(definition)
	r.namespaces[definition.Identity.Namespace] = id
	return nil
}

// Lookup requires the complete typed canonical identity. Reusing an ID with a
// different namespace/external identity fails visibly rather than aliasing it.
func (r *Registry) Lookup(identity canonical.ProviderIdentity) (ProviderDefinition, error) {
	if r == nil {
		return ProviderDefinition{}, &RegistryError{Code: ErrorUnknownProvider, ProviderID: identity.ID, Detail: "registry is nil"}
	}
	if err := identity.Validate(); err != nil {
		return ProviderDefinition{}, &RegistryError{Code: ErrorInvalidProviderIdentity, ProviderID: identity.ID, Detail: err.Error()}
	}
	definition, exists := r.providers[identity.ID]
	if !exists {
		return ProviderDefinition{}, &RegistryError{Code: ErrorUnknownProvider, ProviderID: identity.ID, Detail: "provider is not registered"}
	}
	if !sameProviderIdentity(definition.Identity, identity) {
		return ProviderDefinition{}, &RegistryError{Code: ErrorProviderIdentityMismatch, ProviderID: identity.ID, Detail: "registered provider identity does not match lookup identity"}
	}
	return cloneDefinition(definition), nil
}

func (r *Registry) Capability(identity canonical.ProviderIdentity, id CapabilityID) (Capability, error) {
	definition, err := r.Lookup(identity)
	if err != nil {
		return Capability{}, err
	}
	for _, capability := range definition.Capabilities {
		if capability.ID == id {
			return cloneCapability(capability), nil
		}
	}
	return Capability{}, &RegistryError{Code: ErrorUnknownCapability, ProviderID: identity.ID, CapabilityID: id, Detail: "capability is not declared"}
}

func (r *Registry) List() []ProviderDefinition {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.providers))
	for id := range r.providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	definitions := make([]ProviderDefinition, 0, len(ids))
	for _, id := range ids {
		definitions = append(definitions, cloneDefinition(r.providers[id]))
	}
	return definitions
}

func cloneDefinition(definition ProviderDefinition) ProviderDefinition {
	copyDefinition := definition
	if definition.Identity.ExternalID != nil {
		external := *definition.Identity.ExternalID
		copyDefinition.Identity.ExternalID = &external
	}
	if definition.ProviderAPIVersion != nil {
		version := *definition.ProviderAPIVersion
		copyDefinition.ProviderAPIVersion = &version
	}
	copyDefinition.Capabilities = make([]Capability, len(definition.Capabilities))
	for i, capability := range definition.Capabilities {
		copyDefinition.Capabilities[i] = cloneCapability(capability)
	}
	return copyDefinition
}

func cloneCapability(capability Capability) Capability {
	copyCapability := capability
	copyCapability.Operational.DeliveryModes = append([]DeliveryMode(nil), capability.Operational.DeliveryModes...)
	copyCapability.Operational.FreshnessModes = append([]FreshnessMode(nil), capability.Operational.FreshnessModes...)
	copyCapability.CanonicalOutputs = append([]canonical.ContractSchemaRef(nil), capability.CanonicalOutputs...)
	return copyCapability
}
