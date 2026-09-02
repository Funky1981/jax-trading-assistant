package sec

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

const (
	ProviderID                        = "pvd_sec_edgar"
	ProviderNamespace                 = "sec.data_api"
	SubmissionsSourceID               = "src_sec_submissions"
	CompanyFactsSourceID              = "src_sec_companyfacts"
	SubmissionsRawSchema              = "sec.submissions"
	CompanyFactsRawSchema             = "sec.companyfacts"
	SubmissionsNormalizerID           = "cmp_sec_submissions_normalizer"
	CompanyFactsNormalizerID          = "cmp_sec_companyfacts_normalizer"
	SECFairAccessMaxRequestsPerSecond = 10
	maximumHistoricalFiles            = 64
)

var (
	ProviderIdentity   = canonical.ProviderIdentity{ID: ProviderID, Namespace: ProviderNamespace, ExternalID: &canonical.ExternalID{Namespace: "provider.slug", Value: "sec-edgar"}}
	SubmissionsSource  = canonical.SourceIdentity{ID: SubmissionsSourceID, Kind: canonical.SourceKindRegulator}
	CompanyFactsSource = canonical.SourceIdentity{ID: CompanyFactsSourceID, Kind: canonical.SourceKindRegulator}
	accessionPattern   = regexp.MustCompile(`^[0-9]{10}-[0-9]{2}-[0-9]{6}$`)
	cikPattern         = regexp.MustCompile(`^[0-9]{10}$`)
	datePattern        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// SECProviderDefinition declares the two existing Jax capabilities used by
// SEC. The endpoint schemas remain distinct raw representations.
func SECProviderDefinition() providercontract.ProviderDefinition {
	return providercontract.ProviderDefinition{
		ContractVersion:    providercontract.ProviderDefinitionV1,
		Identity:           ProviderIdentity,
		DisplayName:        "U.S. Securities and Exchange Commission EDGAR",
		AdapterVersion:     canonical.VersionIdentity{Namespace: "jax.sec.adapter", Value: "1.1.0"},
		ProviderAPIVersion: &canonical.VersionIdentity{Namespace: "sec.data_api.documentation", Value: "2025-04"},
		Capabilities: []providercontract.Capability{
			{ContractVersion: providercontract.CapabilityContractV1, ID: providercontract.CapabilityCorporateFiling, Category: providercontract.DataCategoryRegulatoryFiling, Support: providercontract.SupportSupported,
				Raw:            providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: SubmissionsRawSchema, Value: "documented/v1"}, MediaType: "application/json"},
				Authentication: providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone}, Operational: providercontract.OperationalSemantics{DeliveryModes: []providercontract.DeliveryMode{providercontract.DeliverySnapshot, providercontract.DeliveryHistorical}, FreshnessModes: []providercontract.FreshnessMode{providercontract.FreshnessOnDemand}, QualityRequirement: providercontract.QualityCanonicalValidationRequired},
				CanonicalOutputs: []canonical.ContractSchemaRef{{Kind: canonical.ContractKindEvidence, Version: canonical.EvidenceContractV2}}},
			{ContractVersion: providercontract.CapabilityContractV1, ID: providercontract.CapabilityFundamentalObservation, Category: providercontract.DataCategoryFundamentals, Support: providercontract.SupportSupported,
				Raw:            providercontract.RawRepresentation{Boundary: providercontract.RawBoundaryProvider, Format: providercontract.RawFormatJSONDocument, Schema: canonical.VersionIdentity{Namespace: CompanyFactsRawSchema, Value: "documented/v1"}, MediaType: "application/json"},
				Authentication: providercontract.AuthenticationRequirement{Class: providercontract.AuthenticationNone}, Operational: providercontract.OperationalSemantics{DeliveryModes: []providercontract.DeliveryMode{providercontract.DeliverySnapshot, providercontract.DeliveryHistorical}, FreshnessModes: []providercontract.FreshnessMode{providercontract.FreshnessOnDemand}, QualityRequirement: providercontract.QualityCanonicalValidationRequired},
				CanonicalOutputs: []canonical.ContractSchemaRef{{Kind: canonical.ContractKindObservation, Version: canonical.ObservationContractV2}}},
		},
	}
}

// CIKIdentity binds the authoritative SEC identifier to an existing Jax
// issuer. A ticker is deliberately absent from this contract.
type CIKIdentity struct {
	Issuer canonical.ContractRef
	CIK    string
}

func (identity CIKIdentity) Validate() error {
	if identity.Issuer.Kind != canonical.ContractKindIssuer || identity.Issuer.ID == "" || identity.Issuer.ContractVersion != canonical.IssuerContractV1 {
		return fmt.Errorf("SEC identity requires a canonical issuer reference")
	}
	if err := validateCIK(identity.CIK); err != nil {
		return err
	}
	return nil
}

type SECIdentityResolver interface {
	ResolveSECIdentity(string) (canonical.Issuer, error)
}

// StaticIdentityResolver is useful for deterministic composition and tests.
type StaticIdentityResolver map[string]canonical.Issuer

func (resolver StaticIdentityResolver) ResolveSECIdentity(cik string) (canonical.Issuer, error) {
	issuer, ok := resolver[cik]
	if !ok {
		return canonical.Issuer{}, fmt.Errorf("no canonical issuer mapping for SEC CIK %s", cik)
	}
	if err := issuer.Validate(); err != nil {
		return canonical.Issuer{}, err
	}
	for _, id := range issuer.ExternalIDs {
		if id.Namespace == "sec.cik" && id.Value == cik {
			return issuer, nil
		}
	}
	return canonical.Issuer{}, fmt.Errorf("canonical issuer mapping for SEC CIK %s lacks matching sec.cik external identity", cik)
}

// SECDate is a date-only SEC value. It intentionally is not time.Time: a
// filing/report date has no asserted time-of-day or public-availability
// meaning.
type SECDate string

func (date SECDate) Validate() error { return validateDate(string(date)) }

type FilingDateSemantics struct {
	FilingDate             SECDate
	ReportDate             *SECDate
	AcceptanceDateTime     *time.Time
	AcquiredAt             time.Time
	PublicAvailabilityTime *time.Time
}

// FilingIdentity is a provider-neutral filing identity, retaining the SEC
// accession as a source identifier rather than as a Jax canonical ID.
type FilingIdentity struct {
	CIK                        string
	CompanyName                string
	AccessionNumber            string
	Form                       string
	Dates                      FilingDateSemantics
	PrimaryDocument            string
	PrimaryDocumentDescription string
	IsXBRL                     bool
	IsInlineXBRL               bool
	Amended                    bool
	SourcePayload              providercontract.RawPayloadRef
}

type FilingEvidence struct {
	Evidence canonical.Evidence
	Filing   FilingIdentity
}

type CompletenessState string

const (
	CompletenessComplete                 CompletenessState = "COMPLETE"
	CompletenessAdditionalFilesAvailable CompletenessState = "ADDITIONAL_FILES_AVAILABLE"
)

type SubmissionsRequest struct {
	Identity           CIKIdentity
	PayloadID          providercontract.RawPayloadID
	Retention          providercontract.RawPayloadRetentionPolicy
	IncludeHistorical  bool
	MaxHistoricalFiles int
}
type SubmissionsResult struct {
	Execution    providercontract.ExecutionResult
	RawPayloads  []providercontract.RawPayloadDescriptor
	Filings      []FilingEvidence
	Completeness CompletenessState
}

func (result SubmissionsResult) IsComplete() bool { return result.Completeness == CompletenessComplete }

type PeriodKind string

const (
	PeriodInstant  PeriodKind = "INSTANT"
	PeriodDuration PeriodKind = "DURATION"
)

type XBRLPeriod struct {
	Kind  PeriodKind
	Start *SECDate
	End   SECDate
}

// XBRLFactSemantics preserves interpretation metadata not represented by the
// generic Observation contract. No concepts are mapped to a pretend universal
// accounting schema.
type XBRLFactSemantics struct {
	Taxonomy        string
	Concept         string
	Label           string
	Description     string
	Unit            string
	SourceValue     string
	Period          XBRLPeriod
	Form            string
	AccessionNumber string
	FilingDate      SECDate
	FiscalYear      *int
	FiscalPeriod    string
	Frame           string
	Amended         bool
	SourceIndex     int
	SourcePayload   providercontract.RawPayloadRef
}

type CompanyFactObservation struct {
	Observation canonical.Observation
	Evidence    canonical.Evidence
	Semantics   XBRLFactSemantics
}
type CompanyFactsRequest struct {
	Identity  CIKIdentity
	PayloadID providercontract.RawPayloadID
	Retention providercontract.RawPayloadRetentionPolicy
}
type CompanyFactsResult struct {
	Execution providercontract.ExecutionResult
	Raw       providercontract.RawPayloadDescriptor
	Facts     []CompanyFactObservation
	Coverage  CompanyFactsCoverageSemantics
}

// CompanyFactsCoverageSemantics records the documented scope of the SEC
// Company Facts aggregation. It is not a completeness claim.
type CompanyFactsCoverageSemantics struct {
	EntityWideNonCustomTaxonomies bool
	CustomTaxonomiesIncluded      bool
	AbsenceIsProofOfNonDisclosure bool
}

var SECCompanyFactsCoverage = CompanyFactsCoverageSemantics{
	EntityWideNonCustomTaxonomies: true,
	CustomTaxonomiesIncluded:      false,
	AbsenceIsProofOfNonDisclosure: false,
}

// Dependencies is the Phase 02 composition boundary. Resolver is required so
// SEC CIKs are bound to existing canonical issuers without using ticker-only
// identity.
type Dependencies struct {
	Registry *providercontract.Registry
	Executor *providercontract.OperationalExecutor
	Store    providercontract.RawPayloadStore
	Pipeline *providercontract.NormalizationPipeline
	Resolver SECIdentityResolver
}

func validateCIK(value string) error {
	if !cikPattern.MatchString(value) {
		return fmt.Errorf("SEC CIK must be exactly ten digits")
	}
	if strings.Trim(value, "0") == "" {
		return fmt.Errorf("SEC CIK must not be all zeroes")
	}
	return nil
}
func validateDate(value string) error {
	if !datePattern.MatchString(value) {
		return fmt.Errorf("SEC date %q must use YYYY-MM-DD", value)
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return fmt.Errorf("SEC date %q is invalid: %w", value, err)
	}
	if parsed.Format("2006-01-02") != value {
		return fmt.Errorf("SEC date %q is invalid", value)
	}
	return nil
}
func validateAccession(value string) error {
	if !accessionPattern.MatchString(value) {
		return fmt.Errorf("SEC accession number %q is malformed", value)
	}
	return nil
}
func metricFor(taxonomy, concept string) string {
	return "xbrl." + strings.ToLower(taxonomy) + "." + strings.ToLower(concept)
}
