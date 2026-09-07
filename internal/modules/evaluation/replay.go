package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/researchrecommendation"
)

const HistoricalReplayContractV1 = "jax.historical_replay_case/v1"

const ReplayModeArtifact = "ARTIFACT_REPLAY"

var ErrFutureEvidence = errors.New("historical replay requires evidence unavailable at the decision time")

type EvidenceKnowledge struct {
	EvidenceID  string    `json:"evidence_id"`
	AvailableAt time.Time `json:"available_at"`
	Basis       string    `json:"basis"`
}

func (knowledge EvidenceKnowledge) Validate() error {
	if strings.TrimSpace(knowledge.EvidenceID) == "" || knowledge.AvailableAt.IsZero() || knowledge.AvailableAt.Location() != time.UTC || strings.TrimSpace(knowledge.Basis) == "" {
		return fmt.Errorf("evidence knowledge requires an ID, UTC availability time and basis")
	}
	return nil
}

type HistoricalCase struct {
	ContractVersion string                                          `json:"contract_version"`
	ID              string                                          `json:"id"`
	CaseVersion     string                                          `json:"case_version"`
	DecisionAt      time.Time                                       `json:"decision_at"`
	Task            researchrecommendation.ResearchTask             `json:"task"`
	Packet          researchrecommendation.EvidencePacket           `json:"packet"`
	Knowledge       []EvidenceKnowledge                             `json:"knowledge"`
	ContextPlan     researchrecommendation.ContextPlan              `json:"context_plan"`
	Research        researchrecommendation.StructuredResearchOutput `json:"research"`
	Recommendation  researchrecommendation.ResearchRecommendation   `json:"recommendation"`
}

func NewHistoricalCase(caseVersion string, decisionAt time.Time, task researchrecommendation.ResearchTask, packet researchrecommendation.EvidencePacket, knowledge []EvidenceKnowledge, plan researchrecommendation.ContextPlan, research researchrecommendation.StructuredResearchOutput, recommendation researchrecommendation.ResearchRecommendation) (HistoricalCase, error) {
	caseFile := HistoricalCase{ContractVersion: HistoricalReplayContractV1, CaseVersion: caseVersion, DecisionAt: decisionAt, Task: task, Packet: packet, Knowledge: append([]EvidenceKnowledge(nil), knowledge...), ContextPlan: plan, Research: research, Recommendation: recommendation}
	caseFile.ID = deriveHistoricalCaseID(caseFile)
	if err := caseFile.Validate(); err != nil {
		return HistoricalCase{}, err
	}
	return caseFile, nil
}

func (caseFile HistoricalCase) Validate() error {
	if caseFile.ContractVersion != HistoricalReplayContractV1 || !validIdentity("hcase_", caseFile.ID) || strings.TrimSpace(caseFile.CaseVersion) == "" || caseFile.DecisionAt.IsZero() || caseFile.DecisionAt.Location() != time.UTC {
		return fmt.Errorf("historical case identity, version and UTC decision time are required")
	}
	if err := caseFile.Packet.Validate(); err != nil {
		return fmt.Errorf("historical packet: %w", err)
	}
	if strings.TrimSpace(caseFile.Task.Subject) == "" || caseFile.Task.Subject != caseFile.Packet.Subject.ID {
		return fmt.Errorf("historical task subject must match the packet subject")
	}
	if err := caseFile.ContextPlan.Validate(); err != nil {
		return fmt.Errorf("historical context plan: %w", err)
	}
	if err := researchrecommendation.ValidateStructuredResearchOutput(caseFile.ContextPlan, caseFile.Packet, caseFile.Research); err != nil {
		return fmt.Errorf("historical research artifact: %w", err)
	}
	if err := caseFile.Recommendation.Validate(caseFile.Packet, caseFile.Research); err != nil {
		return fmt.Errorf("historical recommendation artifact: %w", err)
	}
	if err := validateKnowledge(caseFile); err != nil {
		return err
	}
	if caseFile.ID != deriveHistoricalCaseID(caseFile) {
		return fmt.Errorf("historical case ID does not match its frozen contents")
	}
	return nil
}

func validateKnowledge(caseFile HistoricalCase) error {
	known := make(map[string]struct{}, len(caseFile.Packet.Items))
	for _, item := range caseFile.Packet.Items {
		known[item.Identity.ID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(caseFile.Knowledge))
	for _, knowledge := range caseFile.Knowledge {
		if err := knowledge.Validate(); err != nil {
			return err
		}
		if _, ok := known[knowledge.EvidenceID]; !ok {
			return fmt.Errorf("knowledge references evidence outside the packet: %q", knowledge.EvidenceID)
		}
		if _, ok := seen[knowledge.EvidenceID]; ok {
			return fmt.Errorf("knowledge repeats evidence %q", knowledge.EvidenceID)
		}
		seen[knowledge.EvidenceID] = struct{}{}
	}
	if len(seen) != len(known) {
		return fmt.Errorf("historical case must declare knowability for every packet item")
	}
	return nil
}

type ReplayResult struct {
	ContractVersion  string    `json:"contract_version"`
	Mode             string    `json:"mode"`
	CaseID           string    `json:"case_id"`
	DecisionAt       time.Time `json:"decision_at"`
	ContextPlanID    string    `json:"context_plan_id"`
	ResearchOutputID string    `json:"research_output_id"`
	RecommendationID string    `json:"recommendation_id"`
	AvailableIDs     []string  `json:"available_ids"`
	Reconstructed    bool      `json:"reconstructed"`
}

func ReplayHistoricalCase(caseFile HistoricalCase) (ReplayResult, error) {
	if err := caseFile.Validate(); err != nil {
		return ReplayResult{}, err
	}
	if err := validateDecisionTimeAvailability(caseFile); err != nil {
		return ReplayResult{}, err
	}
	reconstructed, err := researchrecommendation.BuildContext(caseFile.Task, caseFile.Packet)
	if err != nil {
		return ReplayResult{}, fmt.Errorf("reconstruct historical context: %w", err)
	}
	if reconstructed.ID != caseFile.ContextPlan.ID || reconstructed.Context != caseFile.ContextPlan.Context || reconstructed.PacketContentFingerprint != caseFile.ContextPlan.PacketContentFingerprint {
		return ReplayResult{}, fmt.Errorf("reconstructed context does not match the frozen artifact")
	}
	available := make([]string, 0, len(caseFile.Knowledge))
	for _, knowledge := range caseFile.Knowledge {
		available = append(available, knowledge.EvidenceID)
	}
	sort.Strings(available)
	return ReplayResult{ContractVersion: HistoricalReplayContractV1, Mode: ReplayModeArtifact, CaseID: caseFile.ID, DecisionAt: caseFile.DecisionAt, ContextPlanID: caseFile.ContextPlan.ID, ResearchOutputID: caseFile.Research.ID, RecommendationID: caseFile.Recommendation.ID, AvailableIDs: available, Reconstructed: true}, nil
}

func validateDecisionTimeAvailability(caseFile HistoricalCase) error {
	availability := make(map[string]time.Time, len(caseFile.Knowledge))
	for _, knowledge := range caseFile.Knowledge {
		availability[knowledge.EvidenceID] = knowledge.AvailableAt
		if knowledge.AvailableAt.After(caseFile.DecisionAt) {
			return fmt.Errorf("%w: evidence=%s available_at=%s decision_at=%s", ErrFutureEvidence, knowledge.EvidenceID, knowledge.AvailableAt.Format(time.RFC3339), caseFile.DecisionAt.Format(time.RFC3339))
		}
	}
	for _, item := range caseFile.Packet.Items {
		source := item.Identity.Source
		for label, value := range map[string]*time.Time{"published_at": source.PublishedAt, "observed_at": source.ObservedAt} {
			if value != nil && value.After(caseFile.DecisionAt) {
				return fmt.Errorf("%w: evidence=%s %s=%s decision_at=%s", ErrFutureEvidence, item.Identity.ID, label, value.Format(time.RFC3339), caseFile.DecisionAt.Format(time.RFC3339))
			}
		}
		if source.CollectedAt.After(caseFile.DecisionAt) {
			return fmt.Errorf("%w: evidence=%s collected_at=%s decision_at=%s", ErrFutureEvidence, item.Identity.ID, source.CollectedAt.Format(time.RFC3339), caseFile.DecisionAt.Format(time.RFC3339))
		}
		if availability[item.Identity.ID].After(caseFile.DecisionAt) {
			return fmt.Errorf("%w: evidence=%s", ErrFutureEvidence, item.Identity.ID)
		}
	}
	return nil
}

func deriveHistoricalCaseID(caseFile HistoricalCase) string {
	caseFile.ID = ""
	seed, _ := json.Marshal(caseFile)
	digest := sha256.Sum256(seed)
	return "hcase_" + hex.EncodeToString(digest[:])
}

func validIdentity(prefix, value string) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	for _, character := range value[len(prefix):] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}
