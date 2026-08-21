package canonical

import (
	"encoding/json"
	"fmt"
)

// CanonicalContractBytes returns the exact deterministic bytes used when a
// Jax-owned canonical record is content-addressed. The envelope binds the
// record bytes to the contract kind, schema version, and canonicalization
// version. Contract values contain no maps, array order is significant,
// time.Time uses RFC3339Nano UTC JSON, and encoding/json supplies locale-free
// finite-number formatting after Validate has rejected NaN and infinity.
func CanonicalContractBytes(contract Contract) ([]byte, error) {
	content, err := EncodeJSON(contract)
	if err != nil {
		return nil, err
	}
	kind, version, err := canonicalContractMetadata(contract)
	if err != nil {
		return nil, err
	}
	envelope := struct {
		Canonicalization string          `json:"canonicalization"`
		ContractKind     ContractKind    `json:"contract_kind"`
		ContractVersion  ContractVersion `json:"contract_version"`
		Content          json.RawMessage `json:"content"`
	}{Canonicalization: CanonicalJSONIdentityV1, ContractKind: kind, ContractVersion: version, Content: content}
	return json.Marshal(envelope)
}

func CanonicalContractContentIdentity(contract Contract) (ContentIdentity, error) {
	raw, err := CanonicalContractBytes(contract)
	if err != nil {
		return ContentIdentity{}, err
	}
	return ContentIdentity{
		Representation:   ContentRepresentationCanonicalJSON,
		Digest:           DigestBytes(raw),
		Canonicalization: CanonicalJSONIdentityV1,
	}, nil
}

func (identity ContentIdentity) VerifyCanonicalContract(contract Contract) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	if identity.Representation != ContentRepresentationCanonicalJSON {
		return invalid("content_identity", "representation", "must be canonical_json to verify a canonical contract")
	}
	raw, err := CanonicalContractBytes(contract)
	if err != nil {
		return err
	}
	return identity.Digest.VerifyBytes(raw)
}

func canonicalContractMetadata(contract Contract) (ContractKind, ContractVersion, error) {
	switch value := contract.(type) {
	case Issuer:
		return ContractKindIssuer, value.ContractVersion, nil
	case *Issuer:
		return ContractKindIssuer, value.ContractVersion, nil
	case Instrument:
		return ContractKindInstrument, value.ContractVersion, nil
	case *Instrument:
		return ContractKindInstrument, value.ContractVersion, nil
	case Event:
		return ContractKindEvent, value.ContractVersion, nil
	case *Event:
		return ContractKindEvent, value.ContractVersion, nil
	case Evidence:
		return ContractKindEvidence, value.ContractVersion, nil
	case *Evidence:
		return ContractKindEvidence, value.ContractVersion, nil
	case Observation:
		return ContractKindObservation, value.ContractVersion, nil
	case *Observation:
		return ContractKindObservation, value.ContractVersion, nil
	case ResearchRun:
		return ContractKindResearchRun, value.ContractVersion, nil
	case *ResearchRun:
		return ContractKindResearchRun, value.ContractVersion, nil
	case QuantResult:
		return ContractKindQuantResult, value.ContractVersion, nil
	case *QuantResult:
		return ContractKindQuantResult, value.ContractVersion, nil
	case Recommendation:
		return ContractKindRecommendation, value.ContractVersion, nil
	case *Recommendation:
		return ContractKindRecommendation, value.ContractVersion, nil
	default:
		return "", "", fmt.Errorf("canonical content: unsupported contract type %T", contract)
	}
}
