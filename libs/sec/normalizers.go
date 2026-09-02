package sec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

type staticRequestResolver struct{ identity CIKIdentity }

func (resolver staticRequestResolver) ResolveSECIdentity(cik string) (canonical.Issuer, error) {
	return canonical.Issuer{}, fmt.Errorf("descriptor-only SEC resolver cannot resolve CIK %s", cik)
}

func normalizerDescriptor(capability providercontract.CapabilityID, raw providercontract.RawRepresentation, id, name string, kind canonical.ContractKind, version canonical.ContractVersion) providercontract.NormalizerDescriptor {
	provider := ProviderIdentity
	return providercontract.NormalizerDescriptor{ContractVersion: providercontract.NormalizerDescriptorV1, Provider: provider, CapabilityID: capability, Raw: raw, Component: canonical.ComponentIdentity{ID: id, Kind: canonical.ComponentKindNormalizer, Name: name, Version: canonical.VersionIdentity{Namespace: "jax.sec.normalizer", Value: "1.0.0"}, Provider: &provider}, Target: canonical.ContractSchemaRef{Kind: kind, Version: version}}
}

type submissionColumns struct {
	AccessionNumber            []string `json:"accessionNumber"`
	FilingDate                 []string `json:"filingDate"`
	ReportDate                 []string `json:"reportDate"`
	Form                       []string `json:"form"`
	PrimaryDocument            []string `json:"primaryDocument"`
	PrimaryDocumentDescription []string `json:"primaryDocDescription"`
	IsXBRL                     []int    `json:"isXBRL"`
	IsInlineXBRL               []int    `json:"isInlineXBRL"`
}
type historicalFile struct {
	Name       string `json:"name"`
	FilingFrom string `json:"filingFrom"`
	FilingTo   string `json:"filingTo"`
}
type submissionEnvelope struct {
	Name    string          `json:"name"`
	CIK     json.RawMessage `json:"cik"`
	Filings struct {
		Recent *submissionColumns `json:"recent"`
		Files  []historicalFile   `json:"files"`
	} `json:"filings"`
}
type parsedSubmissions struct {
	CIK             string
	Identities      []FilingIdentity
	AdditionalFiles []historicalFile
}

func decodeOneDocument(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}

func decodeCIK(raw json.RawMessage) (string, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" {
		return "", fmt.Errorf("SEC response is missing CIK")
	}
	if strings.HasPrefix(value, `"`) {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", err
		}
		if err := validateCIK(text); err != nil {
			return "", err
		}
		return text, nil
	}
	if strings.ContainsAny(value, ".eE+-") || len(value) > 10 {
		return "", fmt.Errorf("SEC CIK must be an unsigned integer of at most ten digits")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("SEC CIK must contain digits only")
		}
	}
	return strings.Repeat("0", 10-len(value)) + value, nil
}

func parseSubmissions(data []byte, ref providercontract.RawPayloadRef) (parsedSubmissions, error) {
	var envelope submissionEnvelope
	if err := decodeOneDocument(data, &envelope); err != nil {
		return parsedSubmissions{}, fmt.Errorf("SEC submissions JSON: %w", err)
	}
	cik, err := decodeCIK(envelope.CIK)
	if err != nil {
		return parsedSubmissions{}, err
	}
	if strings.TrimSpace(envelope.Name) == "" {
		return parsedSubmissions{}, fmt.Errorf("SEC submissions company name is required")
	}
	if envelope.Filings.Recent == nil {
		return parsedSubmissions{}, fmt.Errorf("SEC submissions recent filing metadata is required")
	}
	columns := envelope.Filings.Recent
	count := len(columns.AccessionNumber)
	if count == 0 {
		return parsedSubmissions{}, fmt.Errorf("SEC submissions recent accession column is empty")
	}
	for name, length := range map[string]int{"filingDate": len(columns.FilingDate), "form": len(columns.Form), "reportDate": len(columns.ReportDate), "primaryDocument": len(columns.PrimaryDocument), "primaryDocDescription": len(columns.PrimaryDocumentDescription), "isXBRL": len(columns.IsXBRL), "isInlineXBRL": len(columns.IsInlineXBRL)} {
		if length != count {
			return parsedSubmissions{}, fmt.Errorf("SEC submissions %s column length %d does not match accession count %d", name, length, count)
		}
	}
	identities := make([]FilingIdentity, 0, count)
	for index, accession := range columns.AccessionNumber {
		if err := validateAccession(accession); err != nil {
			return parsedSubmissions{}, err
		}
		if err := validateDate(columns.FilingDate[index]); err != nil {
			return parsedSubmissions{}, err
		}
		form := strings.TrimSpace(columns.Form[index])
		if err := validateForm(form); err != nil {
			return parsedSubmissions{}, err
		}
		filingDate, _ := time.Parse("2006-01-02", columns.FilingDate[index])
		var reportDate *time.Time
		if columns.ReportDate[index] != "" {
			if err := validateDate(columns.ReportDate[index]); err != nil {
				return parsedSubmissions{}, err
			}
			parsed, _ := time.Parse("2006-01-02", columns.ReportDate[index])
			reportDate = &parsed
		}
		identities = append(identities, FilingIdentity{CIK: cik, CompanyName: envelope.Name, AccessionNumber: accession, Form: form, Dates: FilingDateSemantics{FilingDate: filingDate.UTC(), ReportDate: reportDate}, PrimaryDocument: columns.PrimaryDocument[index], PrimaryDocumentDescription: columns.PrimaryDocumentDescription[index], IsXBRL: columns.IsXBRL[index] != 0, IsInlineXBRL: columns.IsInlineXBRL[index] != 0, Amended: strings.HasSuffix(form, "/A"), SourcePayload: ref})
	}
	for _, file := range envelope.Filings.Files {
		if strings.TrimSpace(file.Name) == "" || strings.ContainsAny(file.Name, `/\`) || strings.Contains(file.Name, "..") {
			return parsedSubmissions{}, fmt.Errorf("SEC historical file name is unsafe")
		}
		if err := validateDate(file.FilingFrom); err != nil {
			return parsedSubmissions{}, err
		}
		if err := validateDate(file.FilingTo); err != nil {
			return parsedSubmissions{}, err
		}
	}
	return parsedSubmissions{CIK: cik, Identities: identities, AdditionalFiles: append([]historicalFile(nil), envelope.Filings.Files...)}, nil
}

func validateForm(value string) error {
	if value == "" || len(value) > 32 {
		return fmt.Errorf("SEC form is missing or too long")
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '/' || r == ' ' {
			continue
		}
		return fmt.Errorf("SEC form contains unsupported characters")
	}
	return nil
}
func filingURI(cik, accession, document string) string {
	return "https://www.sec.gov/Archives/edgar/data/" + strings.TrimLeft(cik, "0") + "/" + strings.ReplaceAll(accession, "-", "") + "/" + document
}

func filingsForIdentities(identities []FilingIdentity, ref providercontract.RawPayloadRef, resolver SECIdentityResolver) ([]FilingEvidence, error) {
	if resolver == nil {
		return nil, fmt.Errorf("SEC canonical issuer resolver is required")
	}
	result := make([]FilingEvidence, 0, len(identities))
	for _, identity := range identities {
		issuer, err := resolver.ResolveSECIdentity(identity.CIK)
		if err != nil {
			return nil, err
		}
		identity.SourcePayload = ref
		filing, err := buildFilingEvidence(identity, ref, issuer)
		if err != nil {
			return nil, err
		}
		result = append(result, filing)
	}
	return result, nil
}

func buildFilingEvidence(identity FilingIdentity, ref providercontract.RawPayloadRef, issuer canonical.Issuer) (FilingEvidence, error) {
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("sec-filing\x00" + identity.CIK + "\x00" + identity.AccessionNumber)).Value[:24])
	issuerRef := canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}
	evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		return FilingEvidence{}, err
	}
	evidenceRef.PublishedAt = &identity.Dates.FilingDate
	evidence := canonical.Evidence{ContractVersion: canonical.EvidenceContractV2, ID: evidenceID, Type: canonical.EvidenceTypeFiling, Title: identity.CompanyName + " " + identity.Form + " " + identity.AccessionNumber, Summary: "SEC filing metadata", Source: canonical.SourceReference{ID: SubmissionsSourceID, Kind: canonical.SourceKindRegulator, ExternalID: &canonical.ExternalID{Namespace: "sec.accession", Value: identity.AccessionNumber}, URI: filingURI(identity.CIK, identity.AccessionNumber, identity.PrimaryDocument)}, Links: []canonical.EvidenceLink{{Target: issuerRef, Relationship: canonical.EvidenceRelationshipDescribes}}, PublishedAt: &identity.Dates.FilingDate, CollectedAt: ref.ReceivedAt, CreatedAt: ref.ReceivedAt, ImmutableRef: &evidenceRef}
	if err := evidence.Validate(); err != nil {
		return FilingEvidence{}, err
	}
	identity.SourcePayload = ref
	return FilingEvidence{Evidence: evidence, Filing: identity}, nil
}

type submissionsNormalizer struct {
	descriptor providercontract.NormalizerDescriptor
	resolver   SECIdentityResolver
}

func secNormalizationFailure(stage providercontract.NormalizationStage, code providercontract.NormalizationErrorCode, err error) error {
	return &providercontract.NormalizationError{Stage: stage, Code: code, Cause: err, Detail: "SEC deterministic normalizer rejected the provider representation"}
}

func newSubmissionsNormalizer(resolver SECIdentityResolver) *submissionsNormalizer {
	return &submissionsNormalizer{normalizerDescriptor(providercontract.CapabilityCorporateFiling, submissionRaw(), SubmissionsNormalizerID, "SEC submissions deterministic normalizer", canonical.ContractKindEvidence, canonical.EvidenceContractV2), resolver}
}
func (normalizer *submissionsNormalizer) Descriptor() providercontract.NormalizerDescriptor {
	return normalizer.descriptor
}
func (normalizer *submissionsNormalizer) Normalize(context.Context, providercontract.NormalizationInput) (providercontract.NormalizationCandidate, error) {
	return providercontract.NormalizationCandidate{}, fmt.Errorf("SEC submissions response contains a filing batch; single-record normalization is ambiguous")
}
func (normalizer *submissionsNormalizer) NormalizeBatch(_ context.Context, input providercontract.NormalizationInput) ([]providercontract.NormalizationCandidate, error) {
	parsed, err := parseSubmissions(input.Bytes, input.RawRef)
	if err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, err)
	}
	issuer, err := normalizer.resolver.ResolveSECIdentity(parsed.CIK)
	if err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, err)
	}
	result := make([]providercontract.NormalizationCandidate, 0, len(parsed.Identities))
	for _, identity := range parsed.Identities {
		filing, err := buildFilingEvidence(identity, input.RawRef, issuer)
		if err != nil {
			return nil, secNormalizationFailure(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorCanonicalConstruction, err)
		}
		result = append(result, filingCandidate(filing))
	}
	return result, nil
}
func filingCandidate(filing FilingEvidence) providercontract.NormalizationCandidate {
	return providercontract.NormalizationCandidate{Record: filing.Evidence, Revision: canonical.RevisionIdentity{Namespace: "jax.normalized.sec.filing", Value: string(filing.Evidence.ID)}, Dispositions: []providercontract.FieldDisposition{{ProviderField: "cik", Status: providercontract.FieldDispositionRepresented, CanonicalField: "filing.cik"}, {ProviderField: "accessionNumber", Status: providercontract.FieldDispositionRepresented, CanonicalField: "filing.accession_number"}, {ProviderField: "form", Status: providercontract.FieldDispositionRepresented, CanonicalField: "filing.form"}, {ProviderField: "filingDate", Status: providercontract.FieldDispositionRepresented, CanonicalField: "evidence.published_at"}, {ProviderField: "reportDate", Status: providercontract.FieldDispositionRepresented, CanonicalField: "filing.report_date"}, {ProviderField: "primaryDocument", Status: providercontract.FieldDispositionRepresented, CanonicalField: "filing.primary_document"}}}
}

type factValue struct {
	Val             json.RawMessage `json:"val"`
	AccessionNumber string          `json:"accn"`
	Form            string          `json:"form"`
	Filed           string          `json:"filed"`
	Start           string          `json:"start"`
	End             string          `json:"end"`
	FiscalYear      *int            `json:"fy"`
	FiscalPeriod    string          `json:"fp"`
	Frame           string          `json:"frame"`
}
type factConcept struct {
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Units       map[string][]factValue `json:"units"`
}
type factsEnvelope struct {
	CIK        json.RawMessage                   `json:"cik"`
	EntityName string                            `json:"entityName"`
	Facts      map[string]map[string]factConcept `json:"facts"`
}
type factItem struct {
	taxonomy, concept, unit string
	conceptData             factConcept
	value                   factValue
	index                   int
}

func parseCompanyFacts(data []byte, ref providercontract.RawPayloadRef, identity CIKIdentity, resolver SECIdentityResolver) ([]CompanyFactObservation, error) {
	var envelope factsEnvelope
	if err := decodeOneDocument(data, &envelope); err != nil {
		return nil, fmt.Errorf("SEC company facts JSON: %w", err)
	}
	cik, err := decodeCIK(envelope.CIK)
	if err != nil {
		return nil, err
	}
	if cik != identity.CIK {
		return nil, fmt.Errorf("SEC company facts CIK does not match requested identity")
	}
	if strings.TrimSpace(envelope.EntityName) == "" || len(envelope.Facts) == 0 {
		return nil, fmt.Errorf("SEC company facts identity or facts object is missing")
	}
	issuer, err := resolver.ResolveSECIdentity(cik)
	if err != nil {
		return nil, err
	}
	return parseFactItems(envelope, cik, ref, issuer)
}
func parseFactItems(envelope factsEnvelope, cik string, ref providercontract.RawPayloadRef, issuer canonical.Issuer) ([]CompanyFactObservation, error) {
	items := make([]factItem, 0)
	for taxonomy, concepts := range envelope.Facts {
		if strings.TrimSpace(taxonomy) == "" {
			return nil, fmt.Errorf("SEC taxonomy is empty")
		}
		for concept, conceptData := range concepts {
			if strings.TrimSpace(concept) == "" || len(conceptData.Units) == 0 {
				return nil, fmt.Errorf("SEC concept has no units")
			}
			for unit, values := range conceptData.Units {
				if strings.TrimSpace(unit) == "" {
					return nil, fmt.Errorf("SEC XBRL unit is empty")
				}
				for index, value := range values {
					items = append(items, factItem{taxonomy, concept, unit, conceptData, value, index})
				}
			}
		}
	}
	if len(items) == 0 || len(items) > 10000 {
		return nil, fmt.Errorf("SEC company facts count is outside bounded normalization policy")
	}
	for i := range items {
		value, err := validateFactValue(items[i].value)
		if err != nil {
			return nil, err
		}
		items[i].value = value
	}
	sort.SliceStable(items, func(left, right int) bool { return factItemKey(items[left]) < factItemKey(items[right]) })
	result := make([]CompanyFactObservation, 0, len(items))
	for _, item := range items {
		fact, err := makeFact(item.taxonomy, item.concept, item.unit, item.conceptData, item.value, item.index, ref, issuer, cik)
		if err != nil {
			return nil, err
		}
		result = append(result, fact)
	}
	return result, nil
}
func factItemKey(item factItem) string {
	return strings.Join([]string{item.taxonomy, item.concept, item.unit, item.value.End, item.value.Start, item.value.AccessionNumber, item.value.Form, item.value.Filed, item.value.Frame, strconv.Itoa(item.index)}, "\x00")
}
func validateFactValue(value factValue) (factValue, error) {
	lexical := strings.TrimSpace(string(value.Val))
	if lexical == "" || lexical == "null" || strings.HasPrefix(lexical, `"`) {
		return factValue{}, fmt.Errorf("SEC XBRL fact value must be a JSON number")
	}
	number, err := strconv.ParseFloat(lexical, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
		return factValue{}, fmt.Errorf("SEC XBRL fact value is not finite")
	}
	if err := validateAccession(value.AccessionNumber); err != nil {
		return factValue{}, err
	}
	if err := validateForm(strings.TrimSpace(value.Form)); err != nil {
		return factValue{}, err
	}
	if err := validateDate(value.Filed); err != nil {
		return factValue{}, err
	}
	if err := validateDate(value.End); err != nil {
		return factValue{}, err
	}
	if value.Start != "" {
		if err := validateDate(value.Start); err != nil {
			return factValue{}, err
		}
		if value.Start >= value.End {
			return factValue{}, fmt.Errorf("SEC XBRL duration start must precede end")
		}
	}
	if len(value.FiscalPeriod) > 32 || len(value.Frame) > 64 {
		return factValue{}, fmt.Errorf("SEC XBRL fiscal context is too long")
	}
	return value, nil
}

func makeFact(taxonomy, concept, unit string, conceptData factConcept, value factValue, index int, ref providercontract.RawPayloadRef, issuer canonical.Issuer, cik string) (CompanyFactObservation, error) {
	number, _ := strconv.ParseFloat(strings.TrimSpace(string(value.Val)), 64)
	end, _ := time.Parse("2006-01-02", value.End)
	filed, _ := time.Parse("2006-01-02", value.Filed)
	periodKind := PeriodInstant
	var start *time.Time
	if value.Start != "" {
		parsed, _ := time.Parse("2006-01-02", value.Start)
		start = &parsed
		periodKind = PeriodDuration
	}
	observationID := canonical.ObservationID("obs_" + canonical.DigestBytes([]byte(strings.Join([]string{"sec-fact", cik, taxonomy, concept, unit, value.AccessionNumber, value.Form, value.Filed, value.Start, value.End, value.Frame, string(value.Val), strconv.Itoa(index)}, "\x00"))).Value[:24])
	evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("sec-fact-evidence\x00" + string(observationID))).Value[:24])
	issuerRef := canonical.ContractRef{Kind: canonical.ContractKindIssuer, ID: string(issuer.ID), ContractVersion: issuer.ContractVersion}
	evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{Kind: canonical.ContractKindEvidence, ID: string(evidenceID), ContractVersion: canonical.EvidenceContractV2})
	if err != nil {
		return CompanyFactObservation{}, err
	}
	evidenceRef.PublishedAt = &filed
	observation := canonical.Observation{ContractVersion: canonical.ObservationContractV2, ID: observationID, Type: canonical.ObservationTypeFundamental, Subject: issuerRef, Metric: metricFor(taxonomy, concept), Value: canonical.ObservedValue{Type: canonical.ObservedValueTypeNumber, Number: &number, Unit: unit}, Source: canonical.SourceReference{ID: CompanyFactsSourceID, Kind: canonical.SourceKindRegulator, ExternalID: &canonical.ExternalID{Namespace: "sec.accession", Value: value.AccessionNumber}, URI: "https://data.sec.gov/api/xbrl/companyfacts/CIK" + cik + ".json"}, EvidenceIDs: []canonical.EvidenceID{evidenceID}, ObservedAt: end.UTC(), PublishedAt: &filed, CollectedAt: ref.ReceivedAt, CreatedAt: ref.ReceivedAt}
	lineage := canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef}
	fingerprint, err := canonical.ComputeInputFingerprint([]canonical.LineageInput{lineage})
	if err != nil {
		return CompanyFactObservation{}, err
	}
	observation.Provenance = &canonical.Provenance{ContractVersion: canonical.ProvenanceContractV1, ID: "pvn_" + canonical.DigestBytes([]byte(string(observationID))).Value[:24], Inputs: []canonical.LineageInput{lineage}, InputFingerprint: fingerprint, Producer: canonical.ComponentIdentity{ID: CompanyFactsNormalizerID, Kind: canonical.ComponentKindNormalizer, Name: "SEC Company Facts deterministic normalizer", Version: canonical.VersionIdentity{Namespace: "jax.sec.normalizer", Value: "1.0.0"}, Provider: &ProviderIdentity}}
	evidence := canonical.Evidence{ContractVersion: canonical.EvidenceContractV2, ID: evidenceID, Type: canonical.EvidenceTypeDocument, Title: "SEC XBRL " + taxonomy + ":" + concept + " " + value.AccessionNumber, Summary: "SEC Company Facts source context", Source: observation.Source, Links: []canonical.EvidenceLink{{Target: canonical.ContractRef{Kind: canonical.ContractKindObservation, ID: string(observationID), ContractVersion: canonical.ObservationContractV2}, Relationship: canonical.EvidenceRelationshipDescribes}}, PublishedAt: &filed, CollectedAt: ref.ReceivedAt, CreatedAt: ref.ReceivedAt, ImmutableRef: &evidenceRef}
	if err := observation.Validate(); err != nil {
		return CompanyFactObservation{}, err
	}
	if err := evidence.Validate(); err != nil {
		return CompanyFactObservation{}, err
	}
	return CompanyFactObservation{Observation: observation, Evidence: evidence, Semantics: XBRLFactSemantics{Taxonomy: taxonomy, Concept: concept, Label: conceptData.Label, Description: conceptData.Description, Unit: unit, SourceValue: strings.TrimSpace(string(value.Val)), Period: XBRLPeriod{Kind: periodKind, Start: start, End: end.UTC()}, Form: value.Form, AccessionNumber: value.AccessionNumber, FilingDate: filed.UTC(), FiscalYear: value.FiscalYear, FiscalPeriod: value.FiscalPeriod, Frame: value.Frame, Amended: strings.HasSuffix(value.Form, "/A"), SourceIndex: index, SourcePayload: ref}}, nil
}

type companyFactsNormalizer struct {
	descriptor providercontract.NormalizerDescriptor
	resolver   SECIdentityResolver
}

func newCompanyFactsNormalizer(resolver SECIdentityResolver) *companyFactsNormalizer {
	return &companyFactsNormalizer{normalizerDescriptor(providercontract.CapabilityFundamentalObservation, factsRaw(), CompanyFactsNormalizerID, "SEC Company Facts deterministic normalizer", canonical.ContractKindObservation, canonical.ObservationContractV2), resolver}
}
func (normalizer *companyFactsNormalizer) Descriptor() providercontract.NormalizerDescriptor {
	return normalizer.descriptor
}
func (normalizer *companyFactsNormalizer) Normalize(context.Context, providercontract.NormalizationInput) (providercontract.NormalizationCandidate, error) {
	return providercontract.NormalizationCandidate{}, fmt.Errorf("SEC company facts response contains a fact batch; single-record normalization is ambiguous")
}
func (normalizer *companyFactsNormalizer) NormalizeBatch(_ context.Context, input providercontract.NormalizationInput) ([]providercontract.NormalizationCandidate, error) {
	var envelope factsEnvelope
	if err := decodeOneDocument(input.Bytes, &envelope); err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorParserFailure, err)
	}
	cik, err := decodeCIK(envelope.CIK)
	if err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageParsing, providercontract.NormalizationErrorRequiredFieldMissing, err)
	}
	issuer, err := normalizer.resolver.ResolveSECIdentity(cik)
	if err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorIdentityResolution, err)
	}
	facts, err := parseFactItems(envelope, cik, input.RawRef, issuer)
	if err != nil {
		return nil, secNormalizationFailure(providercontract.NormalizationStageMapping, providercontract.NormalizationErrorInvalidProviderValue, err)
	}
	result := make([]providercontract.NormalizationCandidate, 0, len(facts))
	for _, fact := range facts {
		result = append(result, factCandidate(fact))
	}
	return result, nil
}
func factCandidate(fact CompanyFactObservation) providercontract.NormalizationCandidate {
	return providercontract.NormalizationCandidate{Record: fact.Observation, Revision: canonical.RevisionIdentity{Namespace: "jax.normalized.sec.company_fact", Value: string(fact.Observation.ID)}, Dispositions: []providercontract.FieldDisposition{{ProviderField: "taxonomy", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.taxonomy"}, {ProviderField: "concept", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.concept"}, {ProviderField: "unit", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.unit"}, {ProviderField: "start/end", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.period"}, {ProviderField: "form/accn/filed", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.filing"}, {ProviderField: "frame", Status: providercontract.FieldDispositionRepresented, CanonicalField: "fact.frame"}}}
}
