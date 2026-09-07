package researchrecommendation

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const FreshnessGateContractV1 = "jax.research_freshness_sufficiency/v1"

type FreshnessRule struct {
	Kind     EvidenceKind  `json:"kind"`
	MaxAge   time.Duration `json:"max_age"`
	Required bool          `json:"required"`
}

type FreshnessPolicy struct {
	AsOf                      time.Time       `json:"as_of"`
	Rules                     []FreshnessRule `json:"rules"`
	RequiredEvidenceIDs       []string        `json:"required_evidence_ids,omitempty"`
	MinimumEvidenceItems      int             `json:"minimum_evidence_items"`
	MinimumIndependentSources int             `json:"minimum_independent_sources"`
	AllowStale                bool            `json:"allow_stale"`
	AllowUnknown              bool            `json:"allow_unknown"`
}

func (policy FreshnessPolicy) Validate() error {
	if policy.AsOf.IsZero() || policy.AsOf.Location() != time.UTC || policy.MinimumEvidenceItems < 1 || policy.MinimumIndependentSources < 1 {
		return fmt.Errorf("freshness policy requires UTC as-of and positive sufficiency thresholds")
	}
	seen := map[EvidenceKind]struct{}{}
	for _, rule := range policy.Rules {
		if rule.Kind == "" || rule.MaxAge <= 0 {
			return fmt.Errorf("freshness rules require a kind and positive max age")
		}
		if _, exists := seen[rule.Kind]; exists {
			return fmt.Errorf("freshness rule duplicates kind %q", rule.Kind)
		}
		seen[rule.Kind] = struct{}{}
	}
	return nil
}

type EvidenceFreshnessAssessment struct {
	EvidenceID  string         `json:"evidence_id"`
	Kind        EvidenceKind   `json:"kind"`
	State       FreshnessState `json:"state"`
	ReferenceAt time.Time      `json:"reference_at"`
	Age         time.Duration  `json:"age"`
	Required    bool           `json:"required"`
	Reason      string         `json:"reason"`
}

type FreshnessDecision struct {
	ContractVersion        string                        `json:"contract_version"`
	PacketID               string                        `json:"packet_id"`
	AsOf                   time.Time                     `json:"as_of"`
	Sufficient             bool                          `json:"sufficient"`
	State                  FreshnessState                `json:"state"`
	Assessments            []EvidenceFreshnessAssessment `json:"assessments"`
	StaleIDs               []string                      `json:"stale_ids,omitempty"`
	UnknownIDs             []string                      `json:"unknown_ids,omitempty"`
	MissingKinds           []EvidenceKind                `json:"missing_kinds,omitempty"`
	MissingIDs             []string                      `json:"missing_ids,omitempty"`
	IndependentSourceCount int                           `json:"independent_source_count"`
	ReasonCodes            []string                      `json:"reason_codes"`
}

func EvaluateFreshnessAndSufficiency(packet EvidencePacket, policy FreshnessPolicy) (FreshnessDecision, error) {
	if err := packet.Validate(); err != nil {
		return FreshnessDecision{}, err
	}
	if err := policy.Validate(); err != nil {
		return FreshnessDecision{}, err
	}
	rules := map[EvidenceKind]FreshnessRule{}
	for _, rule := range policy.Rules {
		rules[rule.Kind] = rule
	}
	assessments := make([]EvidenceFreshnessAssessment, 0, len(packet.Items))
	staleIDs, unknownIDs := []string{}, []string{}
	requiredKinds := map[EvidenceKind]struct{}{}
	for _, rule := range policy.Rules {
		if rule.Required {
			requiredKinds[rule.Kind] = struct{}{}
		}
	}
	for _, item := range packet.Items {
		referenceAt := item.Identity.Source.CollectedAt
		if item.Identity.Source.ObservedAt != nil {
			referenceAt = *item.Identity.Source.ObservedAt
		} else if item.Identity.Source.PublishedAt != nil {
			referenceAt = *item.Identity.Source.PublishedAt
		}
		assessment := EvidenceFreshnessAssessment{EvidenceID: item.Identity.ID, Kind: item.Identity.Kind, State: item.Identity.Freshness, ReferenceAt: referenceAt, Required: false}
		if _, required := requiredKinds[item.Identity.Kind]; required {
			assessment.Required = true
		}
		if referenceAt.After(policy.AsOf) {
			assessment.State = FreshnessUnknown
			assessment.Reason = "reference time is after policy as-of"
		} else if rule, exists := rules[item.Identity.Kind]; !exists {
			assessment.State = FreshnessUnknown
			assessment.Reason = "no freshness rule exists for evidence kind"
		} else if item.Identity.Freshness == FreshnessUnknown {
			assessment.Reason = "source freshness is unknown"
		} else {
			assessment.Age = policy.AsOf.Sub(referenceAt)
			if assessment.Age > rule.MaxAge {
				assessment.State = FreshnessStale
				assessment.Reason = "age exceeds explicit freshness rule"
			} else if assessment.State == FreshnessFresh {
				assessment.Reason = "within explicit freshness rule"
			}
		}
		if assessment.State == FreshnessStale {
			staleIDs = append(staleIDs, assessment.EvidenceID)
		}
		if assessment.State == FreshnessUnknown {
			unknownIDs = append(unknownIDs, assessment.EvidenceID)
		}
		assessments = append(assessments, assessment)
	}
	knownIDs := map[string]struct{}{}
	independence := map[string]struct{}{}
	presentKinds := map[EvidenceKind]struct{}{}
	for _, item := range packet.Items {
		knownIDs[item.Identity.ID] = struct{}{}
		independence[item.Identity.IndependenceKey] = struct{}{}
		presentKinds[item.Identity.Kind] = struct{}{}
	}
	missingKinds := []EvidenceKind{}
	for kind := range requiredKinds {
		if _, exists := presentKinds[kind]; !exists {
			missingKinds = append(missingKinds, kind)
		}
	}
	missingIDs := []string{}
	for _, id := range policy.RequiredEvidenceIDs {
		if _, exists := knownIDs[id]; !exists {
			missingIDs = append(missingIDs, id)
		}
	}
	sort.Strings(staleIDs)
	sort.Strings(unknownIDs)
	sort.Slice(missingKinds, func(left, right int) bool { return missingKinds[left] < missingKinds[right] })
	sort.Strings(missingIDs)
	reasons := []string{}
	if len(packet.Items) < policy.MinimumEvidenceItems {
		reasons = append(reasons, "evidence_count_below_minimum")
	}
	if len(independence) < policy.MinimumIndependentSources {
		reasons = append(reasons, "independent_source_count_below_minimum")
	}
	if len(missingKinds) > 0 {
		reasons = append(reasons, "required_evidence_kind_missing")
	}
	if len(missingIDs) > 0 {
		reasons = append(reasons, "required_evidence_id_missing")
	}
	if len(staleIDs) > 0 && !policy.AllowStale {
		reasons = append(reasons, "stale_evidence_present")
	}
	if len(unknownIDs) > 0 && !policy.AllowUnknown {
		reasons = append(reasons, "unknown_freshness_present")
	}
	sort.Strings(reasons)
	state := FreshnessFresh
	if len(unknownIDs) > 0 {
		state = FreshnessUnknown
	} else if len(staleIDs) > 0 {
		state = FreshnessStale
	}
	return FreshnessDecision{ContractVersion: FreshnessGateContractV1, PacketID: packet.ID, AsOf: policy.AsOf, Sufficient: len(reasons) == 0, State: state, Assessments: assessments, StaleIDs: staleIDs, UnknownIDs: unknownIDs, MissingKinds: missingKinds, MissingIDs: missingIDs, IndependentSourceCount: len(independence), ReasonCodes: reasons}, nil
}

func (decision FreshnessDecision) Validate() error {
	if decision.ContractVersion != FreshnessGateContractV1 || decision.PacketID == "" || decision.AsOf.IsZero() || decision.AsOf.Location() != time.UTC || len(decision.Assessments) == 0 {
		return fmt.Errorf("freshness decision identity and assessments are required")
	}
	if decision.State != FreshnessFresh && decision.State != FreshnessStale && decision.State != FreshnessUnknown {
		return fmt.Errorf("freshness decision has unsupported state")
	}
	return nil
}

func freshnessReason(decision FreshnessDecision) string {
	if decision.Sufficient {
		return "freshness and evidence sufficiency passed"
	}
	return strings.Join(decision.ReasonCodes, ",")
}
