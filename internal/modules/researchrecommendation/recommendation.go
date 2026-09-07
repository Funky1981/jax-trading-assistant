package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/libs/contracts/canonical"
)

const ResearchRecommendationContractV1 = "jax.research_recommendation/v1"

type RecommendationDisposition string

const (
	DispositionNoTrade   RecommendationDisposition = "NO_TRADE"
	DispositionWatch     RecommendationDisposition = "WATCH"
	DispositionCandidate RecommendationDisposition = "CANDIDATE"
)

type EligibilityInput struct {
	InstrumentResolved   bool
	ResearchValid        bool
	ContextComplete      bool
	EvidenceSufficient   bool
	FreshnessAcceptable  bool
	QuantContextComplete bool
	ModelRouteAllowed    bool
	RequestedDisposition RecommendationDisposition
}

type EligibilityDecision struct {
	Eligible    bool                      `json:"eligible"`
	Disposition RecommendationDisposition `json:"disposition"`
	ReasonCodes []string                  `json:"reason_codes"`
}

func EvaluateEligibility(input EligibilityInput) (EligibilityDecision, error) {
	if input.RequestedDisposition != DispositionNoTrade && input.RequestedDisposition != DispositionWatch && input.RequestedDisposition != DispositionCandidate {
		return EligibilityDecision{}, fmt.Errorf("unsupported requested recommendation disposition")
	}
	reasons := []string{}
	if !input.InstrumentResolved {
		reasons = append(reasons, "instrument_unresolved")
	}
	if !input.ResearchValid {
		reasons = append(reasons, "research_invalid")
	}
	if !input.ContextComplete {
		reasons = append(reasons, "context_incomplete")
	}
	if !input.EvidenceSufficient {
		reasons = append(reasons, "evidence_insufficient")
	}
	if !input.FreshnessAcceptable {
		reasons = append(reasons, "evidence_freshness_unacceptable")
	}
	if !input.QuantContextComplete {
		reasons = append(reasons, "quant_context_incomplete")
	}
	if !input.ModelRouteAllowed {
		reasons = append(reasons, "model_route_not_allowed")
	}
	sort.Strings(reasons)
	if len(reasons) != 0 {
		return EligibilityDecision{Eligible: false, Disposition: DispositionNoTrade, ReasonCodes: reasons}, nil
	}
	return EligibilityDecision{Eligible: true, Disposition: input.RequestedDisposition, ReasonCodes: []string{"deterministic_eligibility_passed"}}, nil
}

type ResearchRecommendation struct {
	ContractVersion        string                    `json:"contract_version"`
	ID                     string                    `json:"id"`
	Subject                canonical.ContractRef     `json:"subject"`
	ResearchOutputID       string                    `json:"research_output_id"`
	ContextPlanID          string                    `json:"context_plan_id"`
	Disposition            RecommendationDisposition `json:"disposition"`
	Thesis                 string                    `json:"thesis"`
	ThesisEvidenceIDs      []string                  `json:"thesis_evidence_ids"`
	BullCase               []ResearchClaim           `json:"bull_case"`
	CounterEvidence        []ResearchClaim           `json:"counter_evidence"`
	Contradictions         []ResearchClaim           `json:"contradictions"`
	Unknowns               []ResearchUnknown         `json:"unknowns"`
	InvalidationConditions []InvalidationCondition   `json:"invalidation_conditions"`
	EvidenceIDs            []string                  `json:"evidence_ids"`
	Eligibility            EligibilityDecision       `json:"eligibility"`
	Confidence             ConfidenceAssessment      `json:"confidence"`
	Authority              string                    `json:"authority"`
	ExecutionAuthority     string                    `json:"execution_authority"`
	CreatedAt              time.Time                 `json:"created_at"`
}

func BuildRecommendation(packet EvidencePacket, plan ContextPlan, research StructuredResearchOutput, eligibility EligibilityDecision, confidence ConfidenceAssessment, createdAt time.Time) (ResearchRecommendation, error) {
	if createdAt.IsZero() || createdAt.Location() != time.UTC {
		return ResearchRecommendation{}, fmt.Errorf("recommendation created_at must be a non-zero UTC timestamp")
	}
	if err := ValidateStructuredResearchOutput(plan, packet, research); err != nil {
		return ResearchRecommendation{}, err
	}
	if err := validateEligibilityDecision(eligibility); err != nil {
		return ResearchRecommendation{}, err
	}
	if err := confidence.Validate(); err != nil {
		return ResearchRecommendation{}, err
	}
	recommendation := ResearchRecommendation{ContractVersion: ResearchRecommendationContractV1, Subject: packet.Subject, ResearchOutputID: research.ID, ContextPlanID: plan.ID, Disposition: eligibility.Disposition, Thesis: research.Thesis, ThesisEvidenceIDs: append([]string(nil), research.ThesisEvidenceIDs...), BullCase: append([]ResearchClaim(nil), research.BullCase...), CounterEvidence: append([]ResearchClaim(nil), research.BearCase...), Contradictions: append([]ResearchClaim(nil), research.Contradictions...), Unknowns: append([]ResearchUnknown(nil), research.Unknowns...), InvalidationConditions: append([]InvalidationCondition(nil), research.InvalidationConditions...), Eligibility: eligibility, Confidence: confidence, Authority: "RESEARCH_DECISION_SUPPORT", ExecutionAuthority: "NONE", CreatedAt: createdAt}
	recommendation.EvidenceIDs = collectResearchEvidenceIDs(research)
	recommendation.ID = deriveRecommendationID(recommendation)
	if err := recommendation.Validate(packet, research); err != nil {
		return ResearchRecommendation{}, err
	}
	return recommendation, nil
}

func (recommendation ResearchRecommendation) Validate(packet EvidencePacket, research StructuredResearchOutput) error {
	if recommendation.ContractVersion != ResearchRecommendationContractV1 || !validIdentity("rec6_", recommendation.ID) {
		return fmt.Errorf("research recommendation contract and identity are required")
	}
	if recommendation.Subject != packet.Subject || recommendation.ResearchOutputID != research.ID || recommendation.ContextPlanID == "" {
		return fmt.Errorf("recommendation provenance does not match packet and research output")
	}
	switch recommendation.Disposition {
	case DispositionNoTrade, DispositionWatch, DispositionCandidate:
	default:
		return fmt.Errorf("unsupported recommendation disposition")
	}
	if strings.TrimSpace(recommendation.Thesis) == "" || len(recommendation.ThesisEvidenceIDs) == 0 || len(recommendation.Unknowns) == 0 || len(recommendation.InvalidationConditions) == 0 {
		return fmt.Errorf("recommendation requires thesis, evidence, unknowns, and invalidation")
	}
	if recommendation.Authority != "RESEARCH_DECISION_SUPPORT" || recommendation.ExecutionAuthority != "NONE" {
		return fmt.Errorf("recommendation authority must remain research-only")
	}
	if err := recommendation.Confidence.Validate(); err != nil {
		return err
	}
	if recommendation.CreatedAt.IsZero() || recommendation.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("recommendation created_at must be UTC")
	}
	known := map[string]struct{}{}
	for _, item := range packet.Items {
		known[item.Identity.ID] = struct{}{}
	}
	for _, id := range recommendation.EvidenceIDs {
		if _, exists := known[id]; !exists {
			return fmt.Errorf("recommendation references evidence outside the packet")
		}
	}
	if recommendation.ID != deriveRecommendationID(recommendation) {
		return fmt.Errorf("recommendation ID does not match its content")
	}
	return nil
}

func validateEligibilityDecision(decision EligibilityDecision) error {
	if decision.Disposition != DispositionNoTrade && decision.Disposition != DispositionWatch && decision.Disposition != DispositionCandidate {
		return fmt.Errorf("eligibility decision has unsupported disposition")
	}
	if len(decision.ReasonCodes) == 0 {
		return fmt.Errorf("eligibility decision requires reason codes")
	}
	return nil
}

func collectResearchEvidenceIDs(research StructuredResearchOutput) []string {
	values := append([]string{}, research.ThesisEvidenceIDs...)
	for _, claim := range append(append([]ResearchClaim{}, research.BullCase...), research.BearCase...) {
		values = append(values, claim.EvidenceIDs...)
	}
	for _, claim := range research.Contradictions {
		values = append(values, claim.EvidenceIDs...)
	}
	for _, unknown := range research.Unknowns {
		values = append(values, unknown.EvidenceIDs...)
	}
	for _, condition := range research.InvalidationConditions {
		values = append(values, condition.EvidenceIDs...)
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func deriveRecommendationID(recommendation ResearchRecommendation) string {
	seed, _ := json.Marshal(struct {
		ContractVersion   string                    `json:"contract_version"`
		Subject           canonical.ContractRef     `json:"subject"`
		ResearchOutputID  string                    `json:"research_output_id"`
		Disposition       RecommendationDisposition `json:"disposition"`
		Thesis            string                    `json:"thesis"`
		EvidenceIDs       []string                  `json:"evidence_ids"`
		ReasonCodes       []string                  `json:"reason_codes"`
		CalibrationStatus CalibrationStatus         `json:"calibration_status"`
	}{ResearchRecommendationContractV1, recommendation.Subject, recommendation.ResearchOutputID, recommendation.Disposition, recommendation.Thesis, recommendation.EvidenceIDs, recommendation.Eligibility.ReasonCodes, recommendation.Confidence.CalibrationStatus})
	digest := sha256.Sum256(seed)
	return "rec6_" + hex.EncodeToString(digest[:])
}
