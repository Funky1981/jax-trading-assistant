package canonical

import (
	"encoding/json"
	"fmt"
)

const ContractCompatibilityV1 ContractVersion = "jax.contract_compatibility/v1"

// ContractSchemaRef identifies a domain schema independently from a record.
// Audit/replay schemas deliberately cannot masquerade as domain schemas here.
type ContractSchemaRef struct {
	Kind    ContractKind    `json:"kind"`
	Version ContractVersion `json:"version"`
}

func (ref ContractSchemaRef) Validate() error {
	const contract = "contract_schema_ref"
	if _, _, ok := contractIdentity(ref.Kind); !ok {
		return invalid(contract, "kind", "is not a supported domain contract kind")
	}
	if !isSupportedContractVersion(ref.Kind, ref.Version) {
		return invalid(contract, "version", "is not a supported version for the contract kind")
	}
	return nil
}

type CompatibilityClassification string

const (
	CompatibilityExact               CompatibilityClassification = "EXACT"
	CompatibilityLosslessTranslation CompatibilityClassification = "LOSSLESS_TRANSLATION"
	CompatibilityLossyTranslation    CompatibilityClassification = "LOSSY_TRANSLATION"
	CompatibilityIncompatible        CompatibilityClassification = "INCOMPATIBLE"
)

// CompatibilityLoss names information that a declared translation cannot
// preserve. Losses are explicit codes, never silently dropped fields.
type CompatibilityLoss struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// ContractCompatibility is one explicit compatibility decision. A translating
// adapter is a versioned mapping component bound to its exact specification or
// implementation bytes. This record does not execute the translation.
type ContractCompatibility struct {
	ContractVersion ContractVersion             `json:"contract_version"`
	ID              string                      `json:"id"`
	Source          ContractSchemaRef           `json:"source"`
	Target          ContractSchemaRef           `json:"target"`
	Classification  CompatibilityClassification `json:"classification"`
	Translator      *ComponentIdentity          `json:"translator,omitempty"`
	Losses          []CompatibilityLoss         `json:"losses,omitempty"`
	ReasonCode      string                      `json:"reason_code,omitempty"`
}

func (assessment ContractCompatibility) Validate() error {
	const contract = "contract_compatibility"
	if err := validateVersion(contract, assessment.ContractVersion, ContractCompatibilityV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", assessment.ID, "cpa_"); err != nil {
		return err
	}
	if err := assessment.Source.Validate(); err != nil {
		return invalid(contract, "source", err.Error())
	}
	if err := assessment.Target.Validate(); err != nil {
		return invalid(contract, "target", err.Error())
	}
	if assessment.Source.Kind != assessment.Target.Kind {
		return invalid(contract, "target.kind", "must match source.kind")
	}
	if assessment.ReasonCode != "" {
		if err := validateCode(contract, "reason_code", assessment.ReasonCode); err != nil {
			return err
		}
	}
	seenLosses := map[string]struct{}{}
	for i, loss := range assessment.Losses {
		field := fmt.Sprintf("losses[%d]", i)
		if err := validateCode(contract, field+".code", loss.Code); err != nil {
			return err
		}
		if err := validateRequiredText(contract, field+".detail", loss.Detail, maxDescription); err != nil {
			return err
		}
		if _, exists := seenLosses[loss.Code]; exists {
			return invalid(contract, field+".code", "duplicates an earlier compatibility loss")
		}
		seenLosses[loss.Code] = struct{}{}
	}

	sameSchema := assessment.Source == assessment.Target
	switch assessment.Classification {
	case CompatibilityExact:
		if !sameSchema {
			return invalid(contract, "classification", "EXACT requires identical source and target schemas")
		}
		if assessment.Translator != nil || len(assessment.Losses) != 0 || assessment.ReasonCode != "" {
			return invalid(contract, "classification", "EXACT cannot carry a translator, loss, or incompatibility reason")
		}
	case CompatibilityLosslessTranslation:
		if sameSchema {
			return invalid(contract, "classification", "LOSSLESS_TRANSLATION requires different schema versions")
		}
		if err := validateCompatibilityTranslator(contract, assessment.Translator); err != nil {
			return err
		}
		if len(assessment.Losses) != 0 {
			return invalid(contract, "losses", "must be empty for LOSSLESS_TRANSLATION")
		}
	case CompatibilityLossyTranslation:
		if sameSchema {
			return invalid(contract, "classification", "LOSSY_TRANSLATION requires different schema versions")
		}
		if err := validateCompatibilityTranslator(contract, assessment.Translator); err != nil {
			return err
		}
		if len(assessment.Losses) == 0 {
			return invalid(contract, "losses", "requires at least one explicit information loss")
		}
	case CompatibilityIncompatible:
		if sameSchema {
			return invalid(contract, "classification", "INCOMPATIBLE cannot describe an identical schema")
		}
		if assessment.Translator != nil {
			return invalid(contract, "translator", "must be absent when no translation is available")
		}
		if assessment.ReasonCode == "" {
			return invalid(contract, "reason_code", "is required for INCOMPATIBLE")
		}
	default:
		return invalid(contract, "classification", "is not supported")
	}
	return nil
}

func validateCompatibilityTranslator(contract string, translator *ComponentIdentity) error {
	if translator == nil {
		return invalid(contract, "translator", "is required for a compatibility translation")
	}
	if err := translator.Validate(); err != nil {
		return invalid(contract, "translator", err.Error())
	}
	if translator.Kind != ComponentKindMapping {
		return invalid(contract, "translator.kind", "must be mapping")
	}
	if translator.Content == nil {
		return invalid(contract, "translator.content", "is required to bind deterministic translation bytes")
	}
	return nil
}

// CompatibilityResolutionError is returned when no single explicit declared
// translation answers a version change. Unsupported future versions fail
// before declaration lookup.
type CompatibilityResolutionError struct {
	Code   string
	Source ContractSchemaRef
	Target ContractSchemaRef
	Detail string
}

func (err *CompatibilityResolutionError) Error() string {
	return fmt.Sprintf("canonical compatibility %s: %s (%s %s -> %s %s)", err.Code, err.Detail, err.Source.Kind, err.Source.Version, err.Target.Kind, err.Target.Version)
}

// ResolveContractCompatibility permits identity compatibility automatically,
// but every version change requires exactly one validated declaration.
func ResolveContractCompatibility(source, target ContractSchemaRef, declarations []ContractCompatibility) (ContractCompatibility, error) {
	if err := source.Validate(); err != nil {
		return ContractCompatibility{}, &CompatibilityResolutionError{Code: "unsupported_source_version", Source: source, Target: target, Detail: err.Error()}
	}
	if err := target.Validate(); err != nil {
		return ContractCompatibility{}, &CompatibilityResolutionError{Code: "unsupported_target_version", Source: source, Target: target, Detail: err.Error()}
	}
	if source == target {
		assessment := ContractCompatibility{
			ContractVersion: ContractCompatibilityV1,
			ID:              compatibilityIdentity(source, target),
			Source:          source,
			Target:          target,
			Classification:  CompatibilityExact,
		}
		return assessment, assessment.Validate()
	}
	var match *ContractCompatibility
	for i := range declarations {
		declaration := declarations[i]
		if declaration.Source != source || declaration.Target != target {
			continue
		}
		if match != nil {
			return ContractCompatibility{}, &CompatibilityResolutionError{Code: "ambiguous_translation", Source: source, Target: target, Detail: "multiple declarations match"}
		}
		match = &declaration
	}
	if match == nil {
		return ContractCompatibility{}, &CompatibilityResolutionError{Code: "translation_not_declared", Source: source, Target: target, Detail: "version changes fail closed without an explicit declaration"}
	}
	if err := match.Validate(); err != nil {
		return ContractCompatibility{}, &CompatibilityResolutionError{Code: "invalid_translation_declaration", Source: source, Target: target, Detail: err.Error()}
	}
	return *match, nil
}

// V1ToV2WithoutHistoricalProvenance records the accepted Jax reality: a V1
// value alone cannot be promoted to a provenance-bearing V2 value. A future
// adapter may declare a translation only when separately retained immutable
// historical evidence actually supplies every required V2 identity.
func V1ToV2WithoutHistoricalProvenance(kind ContractKind) (ContractCompatibility, error) {
	versions := map[ContractKind][2]ContractVersion{
		ContractKindEvidence:       {EvidenceContractV1, EvidenceContractV2},
		ContractKindObservation:    {ObservationContractV1, ObservationContractV2},
		ContractKindResearchRun:    {ResearchRunContractV1, ResearchRunContractV2},
		ContractKindQuantResult:    {QuantResultContractV1, QuantResultContractV2},
		ContractKindRecommendation: {RecommendationContractV1, RecommendationContractV2},
	}
	pair, ok := versions[kind]
	if !ok {
		return ContractCompatibility{}, invalid("contract_compatibility", "source.kind", "has no accepted provenance-bearing V2 family")
	}
	source := ContractSchemaRef{Kind: kind, Version: pair[0]}
	target := ContractSchemaRef{Kind: kind, Version: pair[1]}
	assessment := ContractCompatibility{
		ContractVersion: ContractCompatibilityV1,
		ID:              compatibilityIdentity(source, target),
		Source:          source,
		Target:          target,
		Classification:  CompatibilityIncompatible,
		ReasonCode:      "insufficient_historical_provenance",
	}
	return assessment, assessment.Validate()
}

func compatibilityIdentity(source, target ContractSchemaRef) string {
	raw, _ := json.Marshal(struct {
		Source ContractSchemaRef `json:"source"`
		Target ContractSchemaRef `json:"target"`
	}{source, target})
	return "cpa_" + DigestBytes(raw).Value[:24]
}
