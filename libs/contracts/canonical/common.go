package canonical

import "time"

// ContractVersion identifies a JSON/domain schema, not the revision of a
// particular domain record.
type ContractVersion string

// ContractKind identifies one of the eight canonical contract families.
type ContractKind string

const (
	ContractKindInstrument     ContractKind = "instrument"
	ContractKindIssuer         ContractKind = "issuer"
	ContractKindEvent          ContractKind = "event"
	ContractKindEvidence       ContractKind = "evidence"
	ContractKindObservation    ContractKind = "observation"
	ContractKindResearchRun    ContractKind = "research_run"
	ContractKindQuantResult    ContractKind = "quant_result"
	ContractKindRecommendation ContractKind = "recommendation"
)

const (
	InstrumentContractV1     ContractVersion = "jax.instrument/v1"
	IssuerContractV1         ContractVersion = "jax.issuer/v1"
	EventContractV1          ContractVersion = "jax.event/v1"
	EvidenceContractV1       ContractVersion = "jax.evidence/v1"
	EvidenceContractV2       ContractVersion = "jax.evidence/v2"
	ObservationContractV1    ContractVersion = "jax.observation/v1"
	ObservationContractV2    ContractVersion = "jax.observation/v2"
	ResearchRunContractV1    ContractVersion = "jax.research_run/v1"
	ResearchRunContractV2    ContractVersion = "jax.research_run/v2"
	QuantResultContractV1    ContractVersion = "jax.quant_result/v1"
	QuantResultContractV2    ContractVersion = "jax.quant_result/v2"
	RecommendationContractV1 ContractVersion = "jax.recommendation/v1"
	RecommendationContractV2 ContractVersion = "jax.recommendation/v2"
)

// The distinct ID types prevent accidental interchange between canonical
// entities. IDs are opaque, stable strings with a contract-specific prefix;
// their suffix may be assigned or deterministically derived by a boundary.
type (
	InstrumentID     string
	IssuerID         string
	EventID          string
	EvidenceID       string
	ObservationID    string
	ResearchRunID    string
	QuantResultID    string
	RecommendationID string
)

// ContractRef identifies a canonical record and the schema under which the
// referenced representation is interpreted. It is an identity link only, not
// the immutable provenance reference owned by WP-01.03.
type ContractRef struct {
	Kind            ContractKind    `json:"kind"`
	ID              string          `json:"id"`
	ContractVersion ContractVersion `json:"contract_version"`
}

// ExternalID preserves a provider, venue, registry, or other boundary identity
// without confusing it with Jax canonical identity.
type ExternalID struct {
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

// SourceKind classifies the source placeholder used by Evidence and
// Observation. V2 immutable references additionally separate logical source
// identity from provider/transport identity.
type SourceKind string

const (
	SourceKindPublisher SourceKind = "publisher"
	SourceKindRegulator SourceKind = "regulator"
	SourceKindExchange  SourceKind = "exchange"
	SourceKindIssuer    SourceKind = "issuer"
	SourceKindProvider  SourceKind = "provider"
	SourceKindDataset   SourceKind = "dataset"
	SourceKindModel     SourceKind = "model"
	SourceKindInternal  SourceKind = "internal"
)

// SourceReference attributes a record to a stable source identity and a
// resolvable boundary locator. V2 records bind it to a SourceIdentity and
// immutable content through provenance types in this package.
type SourceReference struct {
	ID         string      `json:"id"`
	Kind       SourceKind  `json:"kind"`
	ExternalID *ExternalID `json:"external_id,omitempty"`
	URI        string      `json:"uri,omitempty"`
}

// EffectivePeriod describes when reference data is economically applicable.
// Nil bounds mean the bound is not known, not infinite precision.
type EffectivePeriod struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// ComponentKind identifies how a research or quantitative component
// participates without importing a provider-specific or orchestration type.
type ComponentKind string

const (
	ComponentKindMethod    ComponentKind = "method"
	ComponentKindAlgorithm ComponentKind = "algorithm"
	ComponentKindModel     ComponentKind = "model"
	ComponentKindTool      ComponentKind = "tool"
	ComponentKindPolicy    ComponentKind = "policy"
)

// ComponentRef preserves the compact V1 method field. V2 provenance binds the
// same method to a namespaced ComponentIdentity and exact immutable inputs.
type ComponentRef struct {
	Kind    ComponentKind `json:"kind"`
	Name    string        `json:"name"`
	Version string        `json:"version"`
}

// Contract is implemented by every accepted canonical record.
type Contract interface {
	Validate() error
}
