package canonical

import (
	"fmt"
	"strings"
	"time"
)

// IssuerType distinguishes legal/economic entity roles without assuming every
// instrument is issued by a conventional corporation.
type IssuerType string

const (
	IssuerTypeCorporate        IssuerType = "corporate"
	IssuerTypeSovereign        IssuerType = "sovereign"
	IssuerTypeGovernmentAgency IssuerType = "government_agency"
	IssuerTypeSupranational    IssuerType = "supranational"
	IssuerTypeFund             IssuerType = "fund"
	IssuerTypeFundSponsor      IssuerType = "fund_sponsor"
	IssuerTypeOther            IssuerType = "other"
)

// Issuer is the legal or economic entity responsible for securities or
// business activity. It is deliberately separate from a tradeable Instrument.
type Issuer struct {
	ContractVersion ContractVersion `json:"contract_version"`
	ID              IssuerID        `json:"id"`
	Type            IssuerType      `json:"type"`
	Name            string          `json:"name"`
	Jurisdiction    string          `json:"jurisdiction,omitempty"`
	ExternalIDs     []ExternalID    `json:"external_ids,omitempty"`
	ParentIssuerID  *IssuerID       `json:"parent_issuer_id,omitempty"`
	Effective       EffectivePeriod `json:"effective"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (issuer Issuer) Validate() error {
	const contract = "issuer"
	if err := validateVersion(contract, issuer.ContractVersion, IssuerContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(issuer.ID), "iss_"); err != nil {
		return err
	}
	switch issuer.Type {
	case IssuerTypeCorporate, IssuerTypeSovereign, IssuerTypeGovernmentAgency,
		IssuerTypeSupranational, IssuerTypeFund, IssuerTypeFundSponsor, IssuerTypeOther:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateRequiredText(contract, "name", issuer.Name, maxShortText); err != nil {
		return err
	}
	if issuer.Jurisdiction != "" {
		if len(issuer.Jurisdiction) != 2 || issuer.Jurisdiction != strings.ToUpper(issuer.Jurisdiction) {
			return invalid(contract, "jurisdiction", "must be an ISO 3166-1 alpha-2 upper-case code")
		}
		for _, r := range issuer.Jurisdiction {
			if r < 'A' || r > 'Z' {
				return invalid(contract, "jurisdiction", "must be an ISO 3166-1 alpha-2 upper-case code")
			}
		}
	}
	if err := validateExternalIDs(contract, "external_ids", issuer.ExternalIDs); err != nil {
		return err
	}
	if issuer.ParentIssuerID != nil {
		if err := validateCanonicalID(contract, "parent_issuer_id", string(*issuer.ParentIssuerID), "iss_"); err != nil {
			return err
		}
		if *issuer.ParentIssuerID == issuer.ID {
			return invalid(contract, "parent_issuer_id", "must not refer to itself")
		}
	}
	if err := validatePeriod(contract, "effective", issuer.Effective); err != nil {
		return err
	}
	return validateRequiredUTC(contract, "created_at", issuer.CreatedAt)
}

// InstrumentType describes a tradeable or measurable financial instrument.
// Economic series and proxies allow non-security market subjects without
// pretending they have a conventional corporate issuer.
type InstrumentType string

const (
	InstrumentTypeEquity         InstrumentType = "equity"
	InstrumentTypeETF            InstrumentType = "etf"
	InstrumentTypeETN            InstrumentType = "etn"
	InstrumentTypeFund           InstrumentType = "fund"
	InstrumentTypeIndex          InstrumentType = "index"
	InstrumentTypeRate           InstrumentType = "rate"
	InstrumentTypeCommodity      InstrumentType = "commodity"
	InstrumentTypeFXPair         InstrumentType = "fx_pair"
	InstrumentTypeDigitalAsset   InstrumentType = "digital_asset"
	InstrumentTypeDerivative     InstrumentType = "derivative"
	InstrumentTypeEconomicSeries InstrumentType = "economic_series"
	InstrumentTypeProxy          InstrumentType = "proxy"
)

// InstrumentIssuerRole explains why an issuer is related to an instrument.
type InstrumentIssuerRole string

const (
	InstrumentIssuerRoleIssuer          InstrumentIssuerRole = "issuer"
	InstrumentIssuerRoleSponsor         InstrumentIssuerRole = "sponsor"
	InstrumentIssuerRoleAdministrator   InstrumentIssuerRole = "administrator"
	InstrumentIssuerRoleSovereign       InstrumentIssuerRole = "sovereign"
	InstrumentIssuerRoleReferenceEntity InstrumentIssuerRole = "reference_entity"
)

type InstrumentIssuer struct {
	IssuerID IssuerID             `json:"issuer_id"`
	Role     InstrumentIssuerRole `json:"role"`
}

// InstrumentRelationship represents an economic/reference relationship to
// another canonical instrument. It does not encode eligibility or resolution
// policy, which remain owned by the Phase 00 modules.
type InstrumentRelationshipType string

const (
	InstrumentRelationshipUnderlying InstrumentRelationshipType = "underlying"
	InstrumentRelationshipTracks     InstrumentRelationshipType = "tracks"
	InstrumentRelationshipProxyFor   InstrumentRelationshipType = "proxy_for"
)

type InstrumentRelationship struct {
	InstrumentID InstrumentID               `json:"instrument_id"`
	Type         InstrumentRelationshipType `json:"type"`
}

// Instrument is a tradeable or measurable financial instrument. Symbols and
// registry codes are external identities; the Jax ID remains stable when a
// symbol, venue, or provider representation changes.
type Instrument struct {
	ContractVersion ContractVersion          `json:"contract_version"`
	ID              InstrumentID             `json:"id"`
	Type            InstrumentType           `json:"type"`
	Name            string                   `json:"name"`
	Currency        string                   `json:"currency,omitempty"`
	MeasurementUnit string                   `json:"measurement_unit,omitempty"`
	ExternalIDs     []ExternalID             `json:"external_ids"`
	Issuers         []InstrumentIssuer       `json:"issuers,omitempty"`
	Relationships   []InstrumentRelationship `json:"relationships,omitempty"`
	Effective       EffectivePeriod          `json:"effective"`
	CreatedAt       time.Time                `json:"created_at"`
}

func (instrument Instrument) Validate() error {
	const contract = "instrument"
	if err := validateVersion(contract, instrument.ContractVersion, InstrumentContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(instrument.ID), "ins_"); err != nil {
		return err
	}
	switch instrument.Type {
	case InstrumentTypeEquity, InstrumentTypeETF, InstrumentTypeETN, InstrumentTypeFund,
		InstrumentTypeIndex, InstrumentTypeRate, InstrumentTypeCommodity, InstrumentTypeFXPair,
		InstrumentTypeDigitalAsset, InstrumentTypeDerivative, InstrumentTypeEconomicSeries,
		InstrumentTypeProxy:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateRequiredText(contract, "name", instrument.Name, maxShortText); err != nil {
		return err
	}
	if instrument.Currency != "" {
		if len(instrument.Currency) != 3 || instrument.Currency != strings.ToUpper(instrument.Currency) {
			return invalid(contract, "currency", "must be a three-letter upper-case currency code")
		}
		for _, r := range instrument.Currency {
			if r < 'A' || r > 'Z' {
				return invalid(contract, "currency", "must be a three-letter upper-case currency code")
			}
		}
	}
	if instrument.MeasurementUnit != "" {
		if err := validateRequiredText(contract, "measurement_unit", instrument.MeasurementUnit, maxShortText); err != nil {
			return err
		}
	}
	if len(instrument.ExternalIDs) == 0 {
		return invalid(contract, "external_ids", "requires at least one boundary identity")
	}
	if err := validateExternalIDs(contract, "external_ids", instrument.ExternalIDs); err != nil {
		return err
	}
	issuerLinks := make(map[string]struct{}, len(instrument.Issuers))
	for i, relationship := range instrument.Issuers {
		field := fmt.Sprintf("issuers[%d]", i)
		if err := validateCanonicalID(contract, field+".issuer_id", string(relationship.IssuerID), "iss_"); err != nil {
			return err
		}
		switch relationship.Role {
		case InstrumentIssuerRoleIssuer, InstrumentIssuerRoleSponsor, InstrumentIssuerRoleAdministrator,
			InstrumentIssuerRoleSovereign, InstrumentIssuerRoleReferenceEntity:
		default:
			return invalid(contract, field+".role", "is not supported")
		}
		key := string(relationship.IssuerID) + "\x00" + string(relationship.Role)
		if _, ok := issuerLinks[key]; ok {
			return invalid(contract, field, "duplicates an earlier issuer relationship")
		}
		issuerLinks[key] = struct{}{}
	}
	relations := make(map[string]struct{}, len(instrument.Relationships))
	for i, relationship := range instrument.Relationships {
		field := fmt.Sprintf("relationships[%d]", i)
		if err := validateCanonicalID(contract, field+".instrument_id", string(relationship.InstrumentID), "ins_"); err != nil {
			return err
		}
		if relationship.InstrumentID == instrument.ID {
			return invalid(contract, field+".instrument_id", "must not refer to itself")
		}
		switch relationship.Type {
		case InstrumentRelationshipUnderlying, InstrumentRelationshipTracks, InstrumentRelationshipProxyFor:
		default:
			return invalid(contract, field+".type", "is not supported")
		}
		key := string(relationship.InstrumentID) + "\x00" + string(relationship.Type)
		if _, ok := relations[key]; ok {
			return invalid(contract, field, "duplicates an earlier instrument relationship")
		}
		relations[key] = struct{}{}
	}
	if err := validatePeriod(contract, "effective", instrument.Effective); err != nil {
		return err
	}
	return validateRequiredUTC(contract, "created_at", instrument.CreatedAt)
}
