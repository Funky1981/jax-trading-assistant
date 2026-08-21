package canonical

import (
	"fmt"
	"time"
)

type ObservationType string

const (
	ObservationTypePrice       ObservationType = "price"
	ObservationTypeYield       ObservationType = "yield"
	ObservationTypeVolume      ObservationType = "volume"
	ObservationTypeMacroValue  ObservationType = "macro_value"
	ObservationTypeFundamental ObservationType = "fundamental"
	ObservationTypeMeasurement ObservationType = "measurement"
	ObservationTypeState       ObservationType = "state"
)

type ObservedValueType string

const (
	ObservedValueTypeNumber  ObservedValueType = "number"
	ObservedValueTypeText    ObservedValueType = "text"
	ObservedValueTypeBoolean ObservedValueType = "boolean"
)

// ObservedValue is a closed, typed value union. Exactly one value field is set
// according to Type; numeric observations also carry an explicit unit.
type ObservedValue struct {
	Type    ObservedValueType `json:"type"`
	Number  *float64          `json:"number,omitempty"`
	Text    *string           `json:"text,omitempty"`
	Boolean *bool             `json:"boolean,omitempty"`
	Unit    string            `json:"unit,omitempty"`
}

// Observation is a measured or recorded value/state at an observation time.
// It is neither an Event nor source material, though it may cite Evidence.
type Observation struct {
	ContractVersion ContractVersion `json:"contract_version"`
	ID              ObservationID   `json:"id"`
	Type            ObservationType `json:"type"`
	Subject         ContractRef     `json:"subject"`
	Metric          string          `json:"metric"`
	Value           ObservedValue   `json:"value"`
	Source          SourceReference `json:"source"`
	EvidenceIDs     []EvidenceID    `json:"evidence_ids,omitempty"`
	ObservedAt      time.Time       `json:"observed_at"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	CollectedAt     time.Time       `json:"collected_at"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (observation Observation) Validate() error {
	const contract = "observation"
	if err := validateVersion(contract, observation.ContractVersion, ObservationContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(observation.ID), "obs_"); err != nil {
		return err
	}
	switch observation.Type {
	case ObservationTypePrice, ObservationTypeYield, ObservationTypeVolume, ObservationTypeMacroValue,
		ObservationTypeFundamental, ObservationTypeMeasurement, ObservationTypeState:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateContractRef(contract, "subject", observation.Subject); err != nil {
		return err
	}
	if observation.Subject.Kind != ContractKindInstrument && observation.Subject.Kind != ContractKindIssuer && observation.Subject.Kind != ContractKindEvent {
		return invalid(contract, "subject.kind", "must identify an instrument, issuer, or event")
	}
	if err := validateCode(contract, "metric", observation.Metric); err != nil {
		return err
	}
	if err := validateObservedValue(contract, observation.Value); err != nil {
		return err
	}
	if err := validateSource(contract, "source", observation.Source); err != nil {
		return err
	}
	seenEvidence := make(map[EvidenceID]struct{}, len(observation.EvidenceIDs))
	for i, evidenceID := range observation.EvidenceIDs {
		field := fmt.Sprintf("evidence_ids[%d]", i)
		if err := validateCanonicalID(contract, field, string(evidenceID), "evd_"); err != nil {
			return err
		}
		if _, ok := seenEvidence[evidenceID]; ok {
			return invalid(contract, field, "duplicates an earlier evidence identity")
		}
		seenEvidence[evidenceID] = struct{}{}
	}
	if err := validateRequiredUTC(contract, "observed_at", observation.ObservedAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "published_at", observation.PublishedAt); err != nil {
		return err
	}
	if observation.PublishedAt != nil && observation.PublishedAt.Before(observation.ObservedAt) {
		return invalid(contract, "published_at", "must not precede observed_at")
	}
	if err := validateRequiredUTC(contract, "collected_at", observation.CollectedAt); err != nil {
		return err
	}
	if observation.CollectedAt.Before(observation.ObservedAt) {
		return invalid(contract, "collected_at", "must not precede observed_at")
	}
	if observation.PublishedAt != nil && observation.CollectedAt.Before(*observation.PublishedAt) {
		return invalid(contract, "collected_at", "must not precede published_at")
	}
	if err := validateRequiredUTC(contract, "created_at", observation.CreatedAt); err != nil {
		return err
	}
	if observation.CreatedAt.Before(observation.CollectedAt) {
		return invalid(contract, "created_at", "must not precede collected_at")
	}
	return nil
}

func validateObservedValue(contract string, value ObservedValue) error {
	set := 0
	if value.Number != nil {
		set++
	}
	if value.Text != nil {
		set++
	}
	if value.Boolean != nil {
		set++
	}
	if set != 1 {
		return invalid(contract, "value", "must set exactly one typed value")
	}
	switch value.Type {
	case ObservedValueTypeNumber:
		if value.Number == nil {
			return invalid(contract, "value.number", "is required for number values")
		}
		if err := validateFinite(contract, "value.number", *value.Number); err != nil {
			return err
		}
		if err := validateRequiredText(contract, "value.unit", value.Unit, maxShortText); err != nil {
			return err
		}
	case ObservedValueTypeText:
		if value.Text == nil {
			return invalid(contract, "value.text", "is required for text values")
		}
		if err := validateRequiredText(contract, "value.text", *value.Text, maxDescription); err != nil {
			return err
		}
		if value.Unit != "" {
			return invalid(contract, "value.unit", "must be empty for text values")
		}
	case ObservedValueTypeBoolean:
		if value.Boolean == nil {
			return invalid(contract, "value.boolean", "is required for boolean values")
		}
		if value.Unit != "" {
			return invalid(contract, "value.unit", "must be empty for boolean values")
		}
	default:
		return invalid(contract, "value.type", "is not supported")
	}
	return nil
}
