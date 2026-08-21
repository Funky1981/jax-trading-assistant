package canonical

import (
	"fmt"
	"time"
)

type EventType string

const (
	EventTypeCorporateAction EventType = "corporate_action"
	EventTypeEarnings        EventType = "earnings"
	EventTypeMacroRelease    EventType = "macro_release"
	EventTypeRegulatory      EventType = "regulatory"
	EventTypeGeopolitical    EventType = "geopolitical"
	EventTypeMarketStructure EventType = "market_structure"
	EventTypeNews            EventType = "news"
	EventTypeOther           EventType = "other"
)

// EventAssertion makes clear whether a domain occurrence is merely asserted,
// confirmed, disputed, or retracted. Evidence remains a separate contract.
type EventAssertion string

const (
	EventAssertionAsserted  EventAssertion = "asserted"
	EventAssertionConfirmed EventAssertion = "confirmed"
	EventAssertionDisputed  EventAssertion = "disputed"
	EventAssertionRetracted EventAssertion = "retracted"
)

// Event is a domain occurrence or asserted occurrence. OccurredAt and
// EffectiveAt are separate because an announcement, release, or policy action
// may become economically effective at a different time.
type Event struct {
	ContractVersion ContractVersion `json:"contract_version"`
	ID              EventID         `json:"id"`
	Type            EventType       `json:"type"`
	Assertion       EventAssertion  `json:"assertion"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary,omitempty"`
	Subjects        []ContractRef   `json:"subjects,omitempty"`
	ExternalIDs     []ExternalID    `json:"external_ids,omitempty"`
	RelatedEventIDs []EventID       `json:"related_event_ids,omitempty"`
	OccurredAt      *time.Time      `json:"occurred_at,omitempty"`
	EffectiveAt     *time.Time      `json:"effective_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (event Event) Validate() error {
	const contract = "event"
	if err := validateVersion(contract, event.ContractVersion, EventContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(event.ID), "evt_"); err != nil {
		return err
	}
	switch event.Type {
	case EventTypeCorporateAction, EventTypeEarnings, EventTypeMacroRelease, EventTypeRegulatory,
		EventTypeGeopolitical, EventTypeMarketStructure, EventTypeNews, EventTypeOther:
	default:
		return invalid(contract, "type", "is not supported")
	}
	switch event.Assertion {
	case EventAssertionAsserted, EventAssertionConfirmed, EventAssertionDisputed, EventAssertionRetracted:
	default:
		return invalid(contract, "assertion", "is not supported")
	}
	if err := validateRequiredText(contract, "title", event.Title, maxShortText); err != nil {
		return err
	}
	if event.Summary != "" {
		if err := validateRequiredText(contract, "summary", event.Summary, maxDescription); err != nil {
			return err
		}
	}
	allowedSubjects := map[ContractKind]bool{
		ContractKindInstrument: true,
		ContractKindIssuer:     true,
	}
	if err := validateContractRefs(contract, "subjects", event.Subjects, allowedSubjects); err != nil {
		return err
	}
	if err := validateExternalIDs(contract, "external_ids", event.ExternalIDs); err != nil {
		return err
	}
	seenEvents := make(map[EventID]struct{}, len(event.RelatedEventIDs))
	for i, relatedID := range event.RelatedEventIDs {
		field := fmt.Sprintf("related_event_ids[%d]", i)
		if err := validateCanonicalID(contract, field, string(relatedID), "evt_"); err != nil {
			return err
		}
		if relatedID == event.ID {
			return invalid(contract, field, "must not refer to itself")
		}
		if _, ok := seenEvents[relatedID]; ok {
			return invalid(contract, field, "duplicates an earlier related event")
		}
		seenEvents[relatedID] = struct{}{}
	}
	if event.OccurredAt == nil && event.EffectiveAt == nil {
		return invalid(contract, "occurred_at", "or effective_at is required")
	}
	if err := validateOptionalUTC(contract, "occurred_at", event.OccurredAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "effective_at", event.EffectiveAt); err != nil {
		return err
	}
	return validateRequiredUTC(contract, "created_at", event.CreatedAt)
}
