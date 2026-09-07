package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const ResearchOutputContractV1 = "jax.structured_research_output/v1"

type ResearchClaim struct {
	Statement   string   `json:"statement"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type ResearchUnknown struct {
	Statement   string   `json:"statement"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

type InvalidationCondition struct {
	Condition   string   `json:"condition"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

// InferenceProvenance retains enough metadata to reconstruct a stochastic
// inference artifact. It does not claim that re-inference will be bit-exact.
type InferenceProvenance struct {
	Provider              string    `json:"provider"`
	Model                 string    `json:"model"`
	PromptVersion         string    `json:"prompt_version"`
	SystemVersion         string    `json:"system_version"`
	OutputContractVersion string    `json:"output_contract_version"`
	RequestID             string    `json:"request_id"`
	RawResponse           string    `json:"raw_response"`
	RawResponseSHA256     string    `json:"raw_response_sha256"`
	CapturedAt            time.Time `json:"captured_at"`
	RetryCount            int       `json:"retry_count"`
	InputTokens           int       `json:"input_tokens"`
	OutputTokens          int       `json:"output_tokens"`
	ActualCostUSD         float64   `json:"actual_cost_usd"`
	UsageComplete         bool      `json:"usage_complete"`
	Paid                  bool      `json:"paid"`
}

type StructuredResearchOutput struct {
	ContractVersion        string                  `json:"contract_version"`
	ID                     string                  `json:"id"`
	ContextPlanID          string                  `json:"context_plan_id"`
	Subject                string                  `json:"subject"`
	Thesis                 string                  `json:"thesis"`
	ThesisEvidenceIDs      []string                `json:"thesis_evidence_ids"`
	BullCase               []ResearchClaim         `json:"bull_case"`
	BearCase               []ResearchClaim         `json:"bear_case"`
	Contradictions         []ResearchClaim         `json:"contradictions"`
	Unknowns               []ResearchUnknown       `json:"unknowns"`
	InvalidationConditions []InvalidationCondition `json:"invalidation_conditions"`
	Inference              InferenceProvenance     `json:"inference"`
}

func NewStructuredResearchOutput(plan ContextPlan, packet EvidencePacket, output StructuredResearchOutput) (StructuredResearchOutput, error) {
	if output.ID == "" {
		output.ID = deriveResearchOutputID(plan, output)
	}
	if err := ValidateStructuredResearchOutput(plan, packet, output); err != nil {
		return StructuredResearchOutput{}, err
	}
	return output, nil
}

func ValidateStructuredResearchOutput(plan ContextPlan, packet EvidencePacket, output StructuredResearchOutput) error {
	if err := packet.Validate(); err != nil {
		return err
	}
	if err := plan.Validate(); err != nil {
		return err
	}
	if output.ContractVersion != ResearchOutputContractV1 || !validIdentity("rso_", output.ID) {
		return fmt.Errorf("research output contract and rso_ identity are required")
	}
	if output.ContextPlanID == "" || output.ContextPlanID != plan.ID {
		return fmt.Errorf("research output must bind to the exact context plan")
	}
	if output.ID != deriveResearchOutputID(plan, output) {
		return fmt.Errorf("research output ID does not match its context, thesis, and raw artifact")
	}
	if strings.TrimSpace(output.Subject) == "" || output.Subject != packet.Subject.ID {
		return fmt.Errorf("research output subject must match the packet subject")
	}
	if strings.TrimSpace(output.Thesis) == "" || len(output.ThesisEvidenceIDs) == 0 || len(output.BullCase) == 0 || len(output.BearCase) == 0 || len(output.Unknowns) == 0 || len(output.InvalidationConditions) == 0 {
		return fmt.Errorf("research output requires thesis, bull case, bear case, unknowns, and invalidation conditions")
	}
	known := map[string]EvidenceItem{}
	for _, item := range packet.Items {
		known[item.Identity.ID] = item
	}
	if err := validateClaim("thesis", ResearchClaim{Statement: output.Thesis, EvidenceIDs: output.ThesisEvidenceIDs}, known, true); err != nil {
		return err
	}
	for index, claim := range output.BullCase {
		if err := validateClaim(fmt.Sprintf("bull_case[%d]", index), claim, known, true); err != nil {
			return err
		}
	}
	for index, claim := range output.BearCase {
		if err := validateClaim(fmt.Sprintf("bear_case[%d]", index), claim, known, true); err != nil {
			return err
		}
	}
	for index, claim := range output.Contradictions {
		if err := validateClaim(fmt.Sprintf("contradictions[%d]", index), claim, known, true); err != nil {
			return err
		}
		if len(uniqueStrings(claim.EvidenceIDs)) < 2 {
			return fmt.Errorf("contradictions[%d] must retain at least two evidence references", index)
		}
	}
	for index, unknown := range output.Unknowns {
		if strings.TrimSpace(unknown.Statement) == "" {
			return fmt.Errorf("unknowns[%d] requires a statement", index)
		}
		if err := validateEvidenceIDs(fmt.Sprintf("unknowns[%d]", index), unknown.EvidenceIDs, known, false); err != nil {
			return err
		}
	}
	for index, condition := range output.InvalidationConditions {
		if strings.TrimSpace(condition.Condition) == "" {
			return fmt.Errorf("invalidation_conditions[%d] requires a condition", index)
		}
		if err := validateEvidenceIDs(fmt.Sprintf("invalidation_conditions[%d]", index), condition.EvidenceIDs, known, false); err != nil {
			return err
		}
	}
	if err := output.Inference.Validate(); err != nil {
		return err
	}
	if output.Inference.OutputContractVersion != ResearchOutputContractV1 || output.Inference.RetryCount > plan.Budget.MaxRetries {
		return fmt.Errorf("inference provenance does not match the output contract or retry budget")
	}
	return nil
}

func validateClaim(name string, claim ResearchClaim, known map[string]EvidenceItem, required bool) error {
	if strings.TrimSpace(claim.Statement) == "" {
		return fmt.Errorf("%s requires a statement", name)
	}
	return validateEvidenceIDs(name, claim.EvidenceIDs, known, required)
}

func validateEvidenceIDs(name string, ids []string, known map[string]EvidenceItem, required bool) error {
	if required && len(ids) == 0 {
		return fmt.Errorf("%s requires evidence references", name)
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if _, exists := known[id]; !exists {
			return fmt.Errorf("%s references evidence not supplied to the task: %q", name, id)
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("%s repeats evidence reference %q", name, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func (provenance InferenceProvenance) Validate() error {
	if strings.TrimSpace(provenance.Provider) == "" || strings.TrimSpace(provenance.Model) == "" || strings.TrimSpace(provenance.PromptVersion) == "" || strings.TrimSpace(provenance.SystemVersion) == "" || strings.TrimSpace(provenance.OutputContractVersion) == "" || strings.TrimSpace(provenance.RequestID) == "" || provenance.RawResponse == "" || !validSHA256(provenance.RawResponseSHA256) || provenance.CapturedAt.IsZero() || provenance.CapturedAt.Location() != time.UTC || provenance.RetryCount < 0 || provenance.InputTokens < 0 || provenance.OutputTokens < 0 || !finite(provenance.ActualCostUSD) || provenance.ActualCostUSD < 0 || !provenance.UsageComplete {
		return fmt.Errorf("inference provenance is incomplete or invalid")
	}
	digest := sha256.Sum256([]byte(provenance.RawResponse))
	if provenance.RawResponseSHA256 != hex.EncodeToString(digest[:]) {
		return fmt.Errorf("raw inference response fingerprint does not match the retained artifact")
	}
	return nil
}

func deriveResearchOutputID(plan ContextPlan, output StructuredResearchOutput) string {
	output.ID = ""
	seed, _ := json.Marshal(struct {
		ContractVersion string                   `json:"contract_version"`
		ContextPlanID   string                   `json:"context_plan_id"`
		Output          StructuredResearchOutput `json:"output"`
	}{ResearchOutputContractV1, plan.ID, output})
	digest := sha256.Sum256(seed)
	return "rso_" + hex.EncodeToString(digest[:])
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
