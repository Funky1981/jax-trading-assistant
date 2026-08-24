package provider

import (
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const (
	RegistryContractV1       canonical.ContractVersion = "jax.provider_registry/v1"
	ProviderDefinitionV1     canonical.ContractVersion = "jax.provider_definition/v1"
	CapabilityContractV1     canonical.ContractVersion = "jax.provider_capability/v1"
	CapabilityRuntimeStateV1 canonical.ContractVersion = "jax.provider_capability_state/v1"
)

// CapabilityID names what information a provider can supply. It is a stable
// Jax concept, not a provider method or endpoint name.
type CapabilityID string

const (
	CapabilityInstrumentReference    CapabilityID = "reference.instrument"
	CapabilityMarketQuote            CapabilityID = "market.quote"
	CapabilityMarketBars             CapabilityID = "market.bars"
	CapabilityMarketTrades           CapabilityID = "market.trades"
	CapabilityCorporateEarnings      CapabilityID = "corporate.earnings"
	CapabilityCorporateFiling        CapabilityID = "corporate.filing"
	CapabilityFundamentalObservation CapabilityID = "fundamentals.observation"
	CapabilityNewsArticle            CapabilityID = "news.article"
	CapabilityEventFeed              CapabilityID = "events.feed"
	CapabilityMacroObservation       CapabilityID = "macro.observation"
	CapabilityEconomicCalendar       CapabilityID = "macro.release_calendar"
)

// DataCategory is the bounded information domain to which a capability
// belongs. It is deliberately smaller than a vendor connector catalogue.
type DataCategory string

const (
	DataCategoryReferenceData     DataCategory = "reference_data"
	DataCategoryMarketData        DataCategory = "market_data"
	DataCategoryCorporateData     DataCategory = "corporate_data"
	DataCategoryRegulatoryFiling  DataCategory = "regulatory_filing"
	DataCategoryFundamentals      DataCategory = "fundamentals"
	DataCategoryNewsEvidence      DataCategory = "news_evidence"
	DataCategoryEventEvidence     DataCategory = "event_evidence"
	DataCategoryMacroeconomicData DataCategory = "macroeconomic_data"
	DataCategoryEconomicCalendar  DataCategory = "economic_calendar"
)

// SupportStatus is a static registry declaration. UNAVAILABLE means the known
// adapter contract cannot currently supply the capability; DISABLED means Jax
// policy disables it independently from deployment configuration. Degraded is
// intentionally absent: degradation is runtime state, not a support claim.
type SupportStatus string

const (
	SupportSupported   SupportStatus = "SUPPORTED"
	SupportUnavailable SupportStatus = "UNAVAILABLE"
	SupportDisabled    SupportStatus = "DISABLED"
)

// AuthenticationClass describes credential requirements without carrying a
// credential, secret location, endpoint, or configuration instance.
type AuthenticationClass string

const (
	AuthenticationNone                 AuthenticationClass = "NONE"
	AuthenticationAPIKey               AuthenticationClass = "API_KEY"
	AuthenticationAPIKeyPair           AuthenticationClass = "API_KEY_PAIR"
	AuthenticationAuthenticatedSession AuthenticationClass = "AUTHENTICATED_SESSION"
)

type AuthenticationRequirement struct {
	Class AuthenticationClass `json:"class"`
}

// RawBoundary makes it impossible to label a provider DTO or payload as a
// canonical Jax record inside a valid capability declaration.
type RawBoundary string

const RawBoundaryProvider RawBoundary = "PROVIDER_RAW"

type RawFormat string

const (
	RawFormatJSONDocument      RawFormat = "JSON_DOCUMENT"
	RawFormatStructuredMessage RawFormat = "STRUCTURED_MESSAGE"
	RawFormatTabular           RawFormat = "TABULAR"
	RawFormatBinary            RawFormat = "BINARY"
	RawFormatStreamMessage     RawFormat = "STREAM_MESSAGE"
)

// RawRepresentation describes the provider-specific representation at the
// acquisition boundary. Schema is the provider DTO/payload schema version,
// never a canonical contract version. Payload bytes are not stored here.
type RawRepresentation struct {
	Boundary  RawBoundary               `json:"boundary"`
	Format    RawFormat                 `json:"format"`
	Schema    canonical.VersionIdentity `json:"schema"`
	MediaType string                    `json:"media_type,omitempty"`
}

type DeliveryMode string

const (
	DeliverySnapshot   DeliveryMode = "SNAPSHOT"
	DeliveryHistorical DeliveryMode = "HISTORICAL"
	DeliveryStream     DeliveryMode = "STREAM"
	DeliveryEvent      DeliveryMode = "EVENT"
)

type FreshnessMode string

const (
	FreshnessRealTime    FreshnessMode = "REAL_TIME"
	FreshnessDelayed     FreshnessMode = "DELAYED"
	FreshnessEndOfDay    FreshnessMode = "END_OF_DAY"
	FreshnessEventDriven FreshnessMode = "EVENT_DRIVEN"
	FreshnessPeriodic    FreshnessMode = "PERIODIC"
	FreshnessOnDemand    FreshnessMode = "ON_DEMAND"
)

type QualityRequirement string

const QualityCanonicalValidationRequired QualityRequirement = "CANONICAL_VALIDATION_REQUIRED"

// OperationalSemantics declares which delivery/freshness classes a capability
// can have. It does not define TTL, last-known-good, polling, retry, or scoring
// policy.
type OperationalSemantics struct {
	DeliveryModes      []DeliveryMode     `json:"delivery_modes"`
	FreshnessModes     []FreshnessMode    `json:"freshness_modes"`
	QualityRequirement QualityRequirement `json:"quality_requirement"`
}

// Capability declares a provider's static support and the exact canonical
// families a future normalizer may emit. CanonicalOutputs never embeds a
// provider DTO and is restricted to external information families.
type Capability struct {
	ContractVersion  canonical.ContractVersion     `json:"contract_version"`
	ID               CapabilityID                  `json:"id"`
	Category         DataCategory                  `json:"category"`
	Support          SupportStatus                 `json:"support"`
	Raw              RawRepresentation             `json:"raw"`
	Authentication   AuthenticationRequirement     `json:"authentication"`
	Operational      OperationalSemantics          `json:"operational"`
	CanonicalOutputs []canonical.ContractSchemaRef `json:"canonical_outputs"`
}

// ProviderDefinition is environment-independent registry metadata. Adapter
// version and provider API version have distinct meanings and neither is the
// registry, capability, or canonical output contract version.
type ProviderDefinition struct {
	ContractVersion    canonical.ContractVersion  `json:"contract_version"`
	Identity           canonical.ProviderIdentity `json:"identity"`
	DisplayName        string                     `json:"display_name"`
	AdapterVersion     canonical.VersionIdentity  `json:"adapter_version"`
	ProviderAPIVersion *canonical.VersionIdentity `json:"provider_api_version,omitempty"`
	Capabilities       []Capability               `json:"capabilities"`
}

type RuntimeStatus string

const (
	RuntimeUnknown     RuntimeStatus = "UNKNOWN"
	RuntimeHealthy     RuntimeStatus = "HEALTHY"
	RuntimeDegraded    RuntimeStatus = "DEGRADED"
	RuntimeUnavailable RuntimeStatus = "UNAVAILABLE"
	RuntimeDisabled    RuntimeStatus = "DISABLED"
)

type FreshnessState string

const (
	FreshnessUnknown FreshnessState = "UNKNOWN"
	FreshnessFresh   FreshnessState = "FRESH"
	FreshnessStale   FreshnessState = "STALE"
)

type QualityState string

const (
	QualityUnknown    QualityState = "UNKNOWN"
	QualityAcceptable QualityState = "ACCEPTABLE"
	QualityDegraded   QualityState = "DEGRADED"
	QualityRejected   QualityState = "REJECTED"
)

// CapabilityRuntimeState is a type boundary for later health/freshness work.
// The registry does not collect, retain, or poll these snapshots.
type CapabilityRuntimeState struct {
	ContractVersion canonical.ContractVersion  `json:"contract_version"`
	Provider        canonical.ProviderIdentity `json:"provider"`
	CapabilityID    CapabilityID               `json:"capability_id"`
	Status          RuntimeStatus              `json:"status"`
	Freshness       FreshnessState             `json:"freshness"`
	Quality         QualityState               `json:"quality"`
	ReasonCode      string                     `json:"reason_code,omitempty"`
	ObservedAt      time.Time                  `json:"observed_at"`
}
