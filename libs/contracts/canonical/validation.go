package canonical

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxIDLength    = 200
	maxShortText   = 256
	maxDescription = 4096
)

// ValidationError is returned when a canonical contract violates a specific,
// deterministic domain rule.
type ValidationError struct {
	Contract string
	Field    string
	Rule     string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("canonical %s: %s %s", e.Contract, e.Field, e.Rule)
}

func invalid(contract, field, rule string) error {
	return &ValidationError{Contract: contract, Field: field, Rule: rule}
}

func validateVersion(contract string, got, want ContractVersion) error {
	if got != want {
		return invalid(contract, "contract_version", fmt.Sprintf("must be %q", want))
	}
	return nil
}

func validateVersionOneOf(contract string, got ContractVersion, wants ...ContractVersion) error {
	for _, want := range wants {
		if got == want {
			return nil
		}
	}
	quoted := make([]string, len(wants))
	for i, want := range wants {
		quoted[i] = fmt.Sprintf("%q", want)
	}
	return invalid(contract, "contract_version", "must be one of "+strings.Join(quoted, ", "))
}

func validateCanonicalID(contract, field, value, prefix string) error {
	if len(value) <= len(prefix) || len(value) > maxIDLength || !strings.HasPrefix(value, prefix) {
		return invalid(contract, field, fmt.Sprintf("must be a %q-prefixed canonical ID", prefix))
	}
	suffix := value[len(prefix):]
	first := suffix[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || (first >= '0' && first <= '9')) {
		return invalid(contract, field, "must start its suffix with an ASCII letter or digit")
	}
	for _, r := range suffix {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.') {
			return invalid(contract, field, "contains an invalid canonical ID character")
		}
	}
	return nil
}

func validateRequiredText(contract, field, value string, max int) error {
	if strings.TrimSpace(value) == "" {
		return invalid(contract, field, "is required")
	}
	if strings.TrimSpace(value) != value {
		return invalid(contract, field, "must not have surrounding whitespace")
	}
	if len(value) > max {
		return invalid(contract, field, fmt.Sprintf("must not exceed %d bytes", max))
	}
	if !utf8.ValidString(value) {
		return invalid(contract, field, "must be valid UTF-8")
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return invalid(contract, field, "contains a control character")
		}
	}
	return nil
}

func validateCode(contract, field, value string) error {
	if err := validateRequiredText(contract, field, value, maxShortText); err != nil {
		return err
	}
	first := value[0]
	if !((first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')) {
		return invalid(contract, field, "must start with a lower-case ASCII letter or digit")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return invalid(contract, field, "must use lower-case letters, digits, '.', '_' or '-'")
	}
	return nil
}

func validateRequiredUTC(contract, field string, value time.Time) error {
	if value.IsZero() {
		return invalid(contract, field, "is required")
	}
	_, offset := value.Zone()
	if offset != 0 {
		return invalid(contract, field, "must use UTC (offset +00:00)")
	}
	if value.Year() < 0 || value.Year() > 9999 {
		return invalid(contract, field, "must use an RFC 3339 four-digit year")
	}
	return nil
}

func validateOptionalUTC(contract, field string, value *time.Time) error {
	if value == nil {
		return nil
	}
	return validateRequiredUTC(contract, field, *value)
}

func validatePeriod(contract, field string, period EffectivePeriod) error {
	if err := validateOptionalUTC(contract, field+".from", period.From); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, field+".to", period.To); err != nil {
		return err
	}
	if period.From != nil && period.To != nil && period.To.Before(*period.From) {
		return invalid(contract, field+".to", "must not precede from")
	}
	return nil
}

func validateExternalID(contract, field string, value ExternalID) error {
	if err := validateCode(contract, field+".namespace", value.Namespace); err != nil {
		return err
	}
	return validateRequiredText(contract, field+".value", value.Value, 512)
}

func validateExternalIDs(contract, field string, values []ExternalID) error {
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		itemField := fmt.Sprintf("%s[%d]", field, i)
		if err := validateExternalID(contract, itemField, value); err != nil {
			return err
		}
		key := value.Namespace + "\x00" + value.Value
		if _, ok := seen[key]; ok {
			return invalid(contract, itemField, "duplicates an earlier external identity")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateSource(contract, field string, source SourceReference) error {
	if err := validateCanonicalID(contract, field+".id", source.ID, "src_"); err != nil {
		return err
	}
	switch source.Kind {
	case SourceKindPublisher, SourceKindRegulator, SourceKindExchange, SourceKindIssuer,
		SourceKindProvider, SourceKindDataset, SourceKindModel, SourceKindInternal:
	default:
		return invalid(contract, field+".kind", "is not supported")
	}
	if source.ExternalID == nil && strings.TrimSpace(source.URI) == "" {
		return invalid(contract, field, "requires external_id or uri")
	}
	if source.ExternalID != nil {
		if err := validateExternalID(contract, field+".external_id", *source.ExternalID); err != nil {
			return err
		}
	}
	if source.URI != "" {
		parsed, err := url.Parse(source.URI)
		if err != nil || parsed.Scheme == "" {
			return invalid(contract, field+".uri", "must be an absolute URI")
		}
	}
	return nil
}

func validateComponent(contract, field string, component ComponentRef) error {
	switch component.Kind {
	case ComponentKindMethod, ComponentKindAlgorithm, ComponentKindModel, ComponentKindTool, ComponentKindPolicy:
	default:
		return invalid(contract, field+".kind", "is not supported")
	}
	if err := validateRequiredText(contract, field+".name", component.Name, maxShortText); err != nil {
		return err
	}
	return validateRequiredText(contract, field+".version", component.Version, maxShortText)
}

func validateContractRef(contract, field string, ref ContractRef) error {
	wantVersion, prefix, ok := contractIdentity(ref.Kind)
	if !ok {
		return invalid(contract, field+".kind", "is not supported")
	}
	if err := validateCanonicalID(contract, field+".id", ref.ID, prefix); err != nil {
		return err
	}
	if ref.ContractVersion != wantVersion {
		return invalid(contract, field+".contract_version", fmt.Sprintf("must be %q", wantVersion))
	}
	return nil
}

func validateSupportedContractRef(contract, field string, ref ContractRef) error {
	_, prefix, ok := contractIdentity(ref.Kind)
	if !ok {
		return invalid(contract, field+".kind", "is not supported")
	}
	if err := validateCanonicalID(contract, field+".id", ref.ID, prefix); err != nil {
		return err
	}
	if !isSupportedContractVersion(ref.Kind, ref.ContractVersion) {
		return invalid(contract, field+".contract_version", "is not supported for the referenced contract kind")
	}
	return nil
}

func validateContractRefForOwner(contract, field string, ref ContractRef, ownerV2 bool) error {
	if !ownerV2 {
		return validateContractRef(contract, field, ref)
	}
	return validateSupportedContractRef(contract, field, ref)
}

func validateContractRefs(contract, field string, refs []ContractRef, allowed map[ContractKind]bool) error {
	return validateContractRefsForOwner(contract, field, refs, allowed, false)
}

func validateContractRefsForOwner(contract, field string, refs []ContractRef, allowed map[ContractKind]bool, ownerV2 bool) error {
	seen := make(map[string]struct{}, len(refs))
	for i, ref := range refs {
		itemField := fmt.Sprintf("%s[%d]", field, i)
		if err := validateContractRefForOwner(contract, itemField, ref, ownerV2); err != nil {
			return err
		}
		if allowed != nil && !allowed[ref.Kind] {
			return invalid(contract, itemField+".kind", "is not allowed in this relationship")
		}
		key := string(ref.Kind) + "\x00" + ref.ID + "\x00" + string(ref.ContractVersion)
		if _, ok := seen[key]; ok {
			return invalid(contract, itemField, "duplicates an earlier reference")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func isSupportedContractVersion(kind ContractKind, version ContractVersion) bool {
	switch kind {
	case ContractKindInstrument:
		return version == InstrumentContractV1
	case ContractKindIssuer:
		return version == IssuerContractV1
	case ContractKindEvent:
		return version == EventContractV1
	case ContractKindEvidence:
		return version == EvidenceContractV1 || version == EvidenceContractV2
	case ContractKindObservation:
		return version == ObservationContractV1 || version == ObservationContractV2
	case ContractKindResearchRun:
		return version == ResearchRunContractV1 || version == ResearchRunContractV2
	case ContractKindQuantResult:
		return version == QuantResultContractV1 || version == QuantResultContractV2
	case ContractKindRecommendation:
		return version == RecommendationContractV1 || version == RecommendationContractV2
	default:
		return false
	}
}

func contractIdentity(kind ContractKind) (ContractVersion, string, bool) {
	switch kind {
	case ContractKindInstrument:
		return InstrumentContractV1, "ins_", true
	case ContractKindIssuer:
		return IssuerContractV1, "iss_", true
	case ContractKindEvent:
		return EventContractV1, "evt_", true
	case ContractKindEvidence:
		return EvidenceContractV1, "evd_", true
	case ContractKindObservation:
		return ObservationContractV1, "obs_", true
	case ContractKindResearchRun:
		return ResearchRunContractV1, "run_", true
	case ContractKindQuantResult:
		return QuantResultContractV1, "qnt_", true
	case ContractKindRecommendation:
		return RecommendationContractV1, "rec_", true
	default:
		return "", "", false
	}
}

func validateFinite(contract, field string, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return invalid(contract, field, "must be finite")
	}
	if value == 0 && math.Signbit(value) {
		return invalid(contract, field, "must not be negative zero")
	}
	return nil
}
