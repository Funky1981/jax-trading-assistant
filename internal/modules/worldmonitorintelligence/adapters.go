package worldmonitorintelligence

import (
	"fmt"
	"sort"
)

const AdapterEvaluationAlgorithmV1 = "jax.world_monitor_evidence_adapters/v1"

type AdapterKind string

const (
	AdapterACLED            AdapterKind = "ACLED"
	AdapterAIS              AdapterKind = "AIS"
	AdapterPredictionMarket AdapterKind = "PREDICTION_MARKET"
)

type AdapterStatus string

const (
	AdapterEvaluatedReadOnly AdapterStatus = "EVALUATED_READ_ONLY"
	AdapterDeferred          AdapterStatus = "DEFERRED_EXTERNAL_PROVISIONING"
)

type AdapterSpec struct {
	ID                 string
	Kind               AdapterKind
	SchemaVersion      string
	ReadOnly           bool
	RequiresCredential bool
	BetaOrNoSLA        bool
	CanCorroborate     bool
	Notes              []string
}

type AdapterEvaluation struct {
	Algorithm          string
	ID                 string
	Kind               AdapterKind
	Status             AdapterStatus
	ReadOnly           bool
	CanCorroborate     bool
	RequiresCredential bool
	BetaOrNoSLA        bool
	Unknowns           []string
	Notes              []string
}

type AdapterEvidence struct {
	AdapterID     string
	ExternalID    string
	SchemaVersion string
	RawPayload    []byte
}

func (evidence AdapterEvidence) Validate() error {
	if evidence.AdapterID == "" || evidence.ExternalID == "" || evidence.SchemaVersion == "" {
		return fmt.Errorf("adapter evidence requires adapter, external ID, and schema version")
	}
	if len(evidence.RawPayload) == 0 {
		return fmt.Errorf("adapter evidence %q has no exact raw payload", evidence.ExternalID)
	}
	return nil
}

// DefaultAdapterSpecs records the bounded Phase-04 evaluation without making
// network calls or requiring credentials. All adapters are evidence-only.
func DefaultAdapterSpecs() []AdapterSpec {
	return []AdapterSpec{
		{ID: "adapter_acled_v1", Kind: AdapterACLED, SchemaVersion: "acled-event/v1", ReadOnly: true, RequiresCredential: true, CanCorroborate: true, Notes: []string{"structured conflict-event evidence", "credential provisioning remains external"}},
		{ID: "adapter_ais_v1", Kind: AdapterAIS, SchemaVersion: "ais-vessel/v1", ReadOnly: true, BetaOrNoSLA: true, CanCorroborate: true, Notes: []string{"maritime corroboration only", "beta/no-SLA source; never sole truth"}},
		{ID: "adapter_prediction_market_v1", Kind: AdapterPredictionMarket, SchemaVersion: "prediction-market-orderbook/v1", ReadOnly: true, CanCorroborate: true, Notes: []string{"public market signal only", "market probability is not event truth"}},
	}
}

func EvaluateEvidenceAdapters(specs []AdapterSpec) ([]AdapterEvaluation, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one adapter specification is required")
	}
	seen := map[string]struct{}{}
	result := make([]AdapterEvaluation, 0, len(specs))
	for _, spec := range specs {
		if spec.ID == "" || spec.SchemaVersion == "" || spec.Kind == "" {
			return nil, fmt.Errorf("adapter identity, kind, and schema version are required")
		}
		if _, exists := seen[spec.ID]; exists {
			return nil, fmt.Errorf("duplicate adapter ID %q", spec.ID)
		}
		seen[spec.ID] = struct{}{}
		if !spec.ReadOnly {
			return nil, fmt.Errorf("adapter %q is not read-only", spec.ID)
		}
		status := AdapterEvaluatedReadOnly
		unknowns := []string{}
		if spec.RequiresCredential {
			status = AdapterDeferred
			unknowns = append(unknowns, "credential and licensing availability are external to this evidence-only boundary")
		}
		if spec.BetaOrNoSLA {
			unknowns = append(unknowns, "provider is beta or has no SLA; corroboration quality requires ongoing qualification")
		}
		if spec.Kind == AdapterPredictionMarket {
			unknowns = append(unknowns, "market probability is a participant signal, not independent event truth")
		}
		notes := append([]string(nil), spec.Notes...)
		sort.Strings(notes)
		sort.Strings(unknowns)
		result = append(result, AdapterEvaluation{Algorithm: AdapterEvaluationAlgorithmV1, ID: spec.ID, Kind: spec.Kind, Status: status, ReadOnly: true, CanCorroborate: spec.CanCorroborate, RequiresCredential: spec.RequiresCredential, BetaOrNoSLA: spec.BetaOrNoSLA, Unknowns: uniqueStrings(unknowns), Notes: notes})
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID < result[right].ID })
	return result, nil
}
