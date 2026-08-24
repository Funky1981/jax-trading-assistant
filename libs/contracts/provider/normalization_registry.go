package provider

import (
	"fmt"

	"jax-trading-assistant/libs/contracts/canonical"
)

type normalizerRoute struct {
	providerID    string
	capabilityID  CapabilityID
	rawBoundary   RawBoundary
	rawFormat     RawFormat
	schemaNS      string
	schemaVersion string
	targetKind    canonical.ContractKind
	targetVersion canonical.ContractVersion
}

type registeredNormalizer struct {
	descriptor NormalizerDescriptor
	normalizer Normalizer
}

// NormalizerRegistry performs exact, process-local routing. One active
// normalizer is allowed for each provider/capability/raw-schema/target route.
// It has no plugin discovery, dynamic loading, network, or configuration I/O.
type NormalizerRegistry struct {
	providers *Registry
	routes    map[normalizerRoute]registeredNormalizer
}

func NewNormalizerRegistry(providers *Registry) (*NormalizerRegistry, error) {
	if providers == nil || providers.ContractVersion() != RegistryContractV1 {
		return nil, &NormalizationError{Stage: NormalizationStageRouting, Code: NormalizationErrorUnsupportedProvider, Detail: "a valid provider registry is required"}
	}
	return &NormalizerRegistry{providers: providers, routes: make(map[normalizerRoute]registeredNormalizer)}, nil
}

func (registry *NormalizerRegistry) Register(normalizer Normalizer) error {
	if registry == nil || registry.providers == nil {
		return &NormalizationError{Stage: NormalizationStageRouting, Code: NormalizationErrorUnsupportedProvider, Detail: "normalizer registry is not initialized"}
	}
	if normalizer == nil {
		return &NormalizationError{Stage: NormalizationStageRouting, Code: NormalizationErrorUnknownNormalizer, Detail: "normalizer is required"}
	}
	descriptor := normalizer.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnknownNormalizer, RawPayloadRef{Provider: descriptor.Provider, CapabilityID: descriptor.CapabilityID}, "normalizer descriptor is invalid", err)
	}
	capability, err := registry.providers.Capability(descriptor.Provider, descriptor.CapabilityID)
	if err != nil {
		return mapRegistryNormalizationError(RawPayloadRef{Provider: descriptor.Provider, CapabilityID: descriptor.CapabilityID}, err)
	}
	if capability.Support != SupportSupported {
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedCapability, RawPayloadRef{Provider: descriptor.Provider, CapabilityID: descriptor.CapabilityID}, "capability is not statically supported", nil)
	}
	if !rawRepresentationsEqual(capability.Raw, descriptor.Raw) {
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedRawSchema, RawPayloadRef{Provider: descriptor.Provider, CapabilityID: descriptor.CapabilityID}, "normalizer raw expectation does not equal the provider capability declaration", nil)
	}
	route := routeFor(descriptor.Provider, descriptor.CapabilityID, descriptor.Raw, descriptor.Target)
	if _, exists := registry.routes[route]; exists {
		return normalizationError(NormalizationStageRouting, NormalizationErrorAmbiguousNormalizer, RawPayloadRef{Provider: descriptor.Provider, CapabilityID: descriptor.CapabilityID}, "the exact route already has a registered normalizer", nil)
	}
	registry.routes[route] = registeredNormalizer{descriptor: cloneNormalizerDescriptor(descriptor), normalizer: normalizer}
	return nil
}

func (registry *NormalizerRegistry) selectNormalizer(ref RawPayloadRef, target canonical.ContractSchemaRef, requested canonical.ComponentIdentity) (Normalizer, NormalizerDescriptor, error) {
	if registry == nil {
		return nil, NormalizerDescriptor{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnknownNormalizer, ref, "normalizer registry is required", nil)
	}
	route := routeFor(ref.Provider, ref.CapabilityID, ref.Raw, target)
	registration, exists := registry.routes[route]
	if !exists {
		return nil, NormalizerDescriptor{}, normalizationError(NormalizationStageRouting, NormalizationErrorUnknownNormalizer, ref, "no exact provider/capability/raw-schema/target normalizer is registered", nil)
	}
	current := registration.normalizer.Descriptor()
	if !normalizerDescriptorsEqual(current, registration.descriptor) {
		return nil, NormalizerDescriptor{}, normalizationError(NormalizationStageRouting, NormalizationErrorAmbiguousNormalizer, ref, "registered normalizer descriptor changed after registration", nil)
	}
	if !sameComponentIdentity(registration.descriptor.Component, requested) {
		return nil, NormalizerDescriptor{}, normalizationError(NormalizationStageRouting, NormalizationErrorNormalizerVersionMismatch, ref, "requested normalizer/mapping identity does not match the registered route", nil)
	}
	return registration.normalizer, cloneNormalizerDescriptor(registration.descriptor), nil
}

func routeFor(provider canonical.ProviderIdentity, capability CapabilityID, raw RawRepresentation, target canonical.ContractSchemaRef) normalizerRoute {
	return normalizerRoute{
		providerID: provider.ID, capabilityID: capability, rawBoundary: raw.Boundary, rawFormat: raw.Format,
		schemaNS: raw.Schema.Namespace, schemaVersion: raw.Schema.Value,
		targetKind: target.Kind, targetVersion: target.Version,
	}
}

func rawRepresentationsEqual(left, right RawRepresentation) bool {
	return left == right
}

func normalizerDescriptorsEqual(left, right NormalizerDescriptor) bool {
	return left.ContractVersion == right.ContractVersion && sameProviderIdentity(left.Provider, right.Provider) &&
		left.CapabilityID == right.CapabilityID && left.Raw == right.Raw && left.Target == right.Target &&
		sameComponentIdentity(left.Component, right.Component)
}

func cloneNormalizerDescriptor(descriptor NormalizerDescriptor) NormalizerDescriptor {
	copyDescriptor := descriptor
	copyDescriptor.Provider = cloneProviderIdentity(descriptor.Provider)
	copyDescriptor.Component = cloneComponentIdentity(descriptor.Component)
	return copyDescriptor
}

func mapRegistryNormalizationError(ref RawPayloadRef, err error) error {
	registryErr, ok := err.(*RegistryError)
	if !ok {
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedProvider, ref, "provider registry lookup failed", err)
	}
	switch registryErr.Code {
	case ErrorUnknownCapability:
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedCapability, ref, "capability is not declared by the provider", err)
	case ErrorProviderIdentityMismatch, ErrorInvalidProviderIdentity:
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedProvider, ref, "provider identity does not match the registry", err)
	default:
		return normalizationError(NormalizationStageRouting, NormalizationErrorUnsupportedProvider, ref, fmt.Sprintf("provider registry rejected lookup (%s)", registryErr.Code), err)
	}
}
