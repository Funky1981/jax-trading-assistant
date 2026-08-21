package canonical

import (
	"fmt"
	"time"
)

// RecommendationDisposition intentionally preserves the active deterministic
// NO_TRADE/WATCH/CANDIDATE vocabulary. CANDIDATE is a research classification,
// not a candidates.Candidate aggregate, approval, or execution instruction.
type RecommendationDisposition string

const (
	RecommendationDispositionNoTrade   RecommendationDisposition = "NO_TRADE"
	RecommendationDispositionWatch     RecommendationDisposition = "WATCH"
	RecommendationDispositionCandidate RecommendationDisposition = "CANDIDATE"
)

type RecommendationAuthority string

const RecommendationAuthorityResearchDecisionSupport RecommendationAuthority = "RESEARCH_DECISION_SUPPORT"

type ExecutionAuthority string

const ExecutionAuthorityNone ExecutionAuthority = "NONE"

type RecommendationReason struct {
	Code    string `json:"code"`
	Summary string `json:"summary"`
}

// Recommendation is Jax research/decision-support output. Its authority fields
// fail closed: v1 accepts research decision support with no execution authority
// and carries no order, quantity, approval, or broker fields.
type Recommendation struct {
	ContractVersion    ContractVersion           `json:"contract_version"`
	ID                 RecommendationID          `json:"id"`
	Disposition        RecommendationDisposition `json:"disposition"`
	ResearchRunID      ResearchRunID             `json:"research_run_id"`
	Subjects           []ContractRef             `json:"subjects"`
	Basis              []ContractRef             `json:"basis"`
	Reasons            []RecommendationReason    `json:"reasons"`
	Confidence         *float64                  `json:"confidence,omitempty"`
	Authority          RecommendationAuthority   `json:"authority"`
	ExecutionAuthority ExecutionAuthority        `json:"execution_authority"`
	CreatedAt          time.Time                 `json:"created_at"`
	ValidUntil         *time.Time                `json:"valid_until,omitempty"`
	Provenance         *Provenance               `json:"provenance,omitempty"`
}

func (recommendation Recommendation) Validate() error {
	const contract = "recommendation"
	if err := validateVersionOneOf(contract, recommendation.ContractVersion, RecommendationContractV1, RecommendationContractV2); err != nil {
		return err
	}
	isV2 := recommendation.ContractVersion == RecommendationContractV2
	if err := validateCanonicalID(contract, "id", string(recommendation.ID), "rec_"); err != nil {
		return err
	}
	switch recommendation.Disposition {
	case RecommendationDispositionNoTrade, RecommendationDispositionWatch, RecommendationDispositionCandidate:
	default:
		return invalid(contract, "disposition", "is not supported")
	}
	if err := validateCanonicalID(contract, "research_run_id", string(recommendation.ResearchRunID), "run_"); err != nil {
		return err
	}
	if len(recommendation.Subjects) == 0 {
		return invalid(contract, "subjects", "requires at least one instrument, issuer, or event")
	}
	allowedSubjects := map[ContractKind]bool{
		ContractKindInstrument: true,
		ContractKindIssuer:     true,
		ContractKindEvent:      true,
	}
	if err := validateContractRefsForOwner(contract, "subjects", recommendation.Subjects, allowedSubjects, isV2); err != nil {
		return err
	}
	if len(recommendation.Basis) == 0 {
		return invalid(contract, "basis", "requires at least one identified research basis")
	}
	allowedBasis := map[ContractKind]bool{
		ContractKindEvent:       true,
		ContractKindEvidence:    true,
		ContractKindObservation: true,
		ContractKindResearchRun: true,
		ContractKindQuantResult: true,
	}
	if err := validateContractRefsForOwner(contract, "basis", recommendation.Basis, allowedBasis, isV2); err != nil {
		return err
	}
	if len(recommendation.Reasons) == 0 {
		return invalid(contract, "reasons", "requires at least one reason")
	}
	seenReasons := make(map[string]struct{}, len(recommendation.Reasons))
	for i, reason := range recommendation.Reasons {
		field := fmt.Sprintf("reasons[%d]", i)
		if err := validateCode(contract, field+".code", reason.Code); err != nil {
			return err
		}
		if err := validateRequiredText(contract, field+".summary", reason.Summary, maxDescription); err != nil {
			return err
		}
		if _, ok := seenReasons[reason.Code]; ok {
			return invalid(contract, field+".code", "duplicates an earlier reason code")
		}
		seenReasons[reason.Code] = struct{}{}
	}
	if recommendation.Confidence != nil {
		if err := validateFinite(contract, "confidence", *recommendation.Confidence); err != nil {
			return err
		}
		if *recommendation.Confidence < 0 || *recommendation.Confidence > 1 {
			return invalid(contract, "confidence", "must be between 0 and 1")
		}
	}
	if recommendation.Authority != RecommendationAuthorityResearchDecisionSupport {
		return invalid(contract, "authority", "must be RESEARCH_DECISION_SUPPORT")
	}
	if recommendation.ExecutionAuthority != ExecutionAuthorityNone {
		return invalid(contract, "execution_authority", "must be NONE")
	}
	if err := validateRequiredUTC(contract, "created_at", recommendation.CreatedAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "valid_until", recommendation.ValidUntil); err != nil {
		return err
	}
	if recommendation.ValidUntil != nil && !recommendation.ValidUntil.After(recommendation.CreatedAt) {
		return invalid(contract, "valid_until", "must be after created_at")
	}
	if !isV2 {
		if recommendation.Provenance != nil {
			return invalid(contract, "contract_version", "v1 cannot carry v2 provenance fields")
		}
		return nil
	}
	if recommendation.Provenance == nil {
		return invalid(contract, "provenance", "is required by recommendation v2")
	}
	self := ContractRef{Kind: ContractKindRecommendation, ID: string(recommendation.ID), ContractVersion: recommendation.ContractVersion}
	if err := validateProvenance(contract, "provenance", *recommendation.Provenance, &self); err != nil {
		return err
	}
	required := append([]ContractRef(nil), recommendation.Basis...)
	required = append(required, ContractRef{Kind: ContractKindResearchRun, ID: string(recommendation.ResearchRunID), ContractVersion: ResearchRunContractV2})
	if !provenanceCoversContractRefsFlexibleRun(*recommendation.Provenance, required, recommendation.ResearchRunID) {
		return invalid(contract, "provenance.inputs", "must immutably cover every basis reference and the producing research run")
	}
	return nil
}
