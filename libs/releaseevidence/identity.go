package releaseevidence

import (
	"fmt"
	"strings"

	"jax-trading-assistant/libs/contracts/canonical"
	providercontract "jax-trading-assistant/libs/contracts/provider"
)

// DeterministicID derives an identity from authoritative source fields. The
// revision argument must be empty only when the source supplied no revision
// identity; titles are never used to reconcile records across providers.
func DeterministicID(providerID, sourceID, sourceReleaseID, scheduledDate, revision string) string {
	key := strings.Join([]string{providerID, sourceID, sourceReleaseID, scheduledDate, revision}, "\x00")
	return "erl_" + canonical.DigestBytes([]byte(key)).Value[:24]
}

// MakeProvenance bridges one or more exact raw acquisitions into the accepted
// immutable evidence vocabulary. It does not create a canonical Evidence
// record and exposes no provider DTOs.
func MakeProvenance(refs []providercontract.RawPayloadRef, producer canonical.ComponentIdentity, salt string) (canonical.Provenance, error) {
	if len(refs) == 0 {
		return canonical.Provenance{}, fmt.Errorf("release provenance requires at least one raw payload")
	}
	inputs := make([]canonical.LineageInput, 0, len(refs))
	seen := make(map[providercontract.RawPayloadID]struct{}, len(refs))
	for _, ref := range refs {
		if _, ok := seen[ref.ID]; ok {
			continue
		}
		seen[ref.ID] = struct{}{}
		evidenceID := canonical.EvidenceID("evd_" + canonical.DigestBytes([]byte("release-evidence\x00" + string(ref.ID))).Value[:24])
		evidenceRef, err := ref.AsEvidenceRef(canonical.ContractRef{
			Kind:            canonical.ContractKindEvidence,
			ID:              string(evidenceID),
			ContractVersion: canonical.EvidenceContractV2,
		})
		if err != nil {
			return canonical.Provenance{}, err
		}
		inputs = append(inputs, canonical.LineageInput{Kind: canonical.LineageInputKindEvidence, Evidence: &evidenceRef})
	}
	fingerprint, err := canonical.ComputeInputFingerprint(inputs)
	if err != nil {
		return canonical.Provenance{}, err
	}
	return canonical.Provenance{
		ContractVersion:  canonical.ProvenanceContractV1,
		ID:               "pvn_" + canonical.DigestBytes([]byte("release-provenance\x00" + salt)).Value[:24],
		Inputs:           inputs,
		InputFingerprint: fingerprint,
		Producer:         producer,
	}, nil
}
