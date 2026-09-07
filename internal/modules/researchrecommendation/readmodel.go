package researchrecommendation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const RecommendationReadModelContractV1 = "jax.recommendation_read_model/v1"

type EvidenceView struct {
	ID                 string         `json:"id"`
	Kind               EvidenceKind   `json:"kind"`
	SourceID           string         `json:"source_id"`
	Provider           string         `json:"provider"`
	RawReference       string         `json:"raw_reference"`
	RawContentSHA256   string         `json:"raw_content_sha256"`
	Freshness          FreshnessState `json:"freshness"`
	Title              string         `json:"title"`
	Summary            string         `json:"summary"`
	Excerpt            string         `json:"untrusted_excerpt,omitempty"`
	SelectedInContext  bool           `json:"selected_in_context"`
	DuplicateInContext bool           `json:"duplicate_in_context"`
	NumericContext     []NumericValue `json:"numeric_context,omitempty"`
}

type InferenceView struct {
	Provider              string  `json:"provider"`
	Model                 string  `json:"model"`
	PromptVersion         string  `json:"prompt_version"`
	SystemVersion         string  `json:"system_version"`
	OutputContractVersion string  `json:"output_contract_version"`
	RequestID             string  `json:"request_id"`
	RawResponseSHA256     string  `json:"raw_response_sha256"`
	RetryCount            int     `json:"retry_count"`
	InputTokens           int     `json:"input_tokens"`
	OutputTokens          int     `json:"output_tokens"`
	ActualCostUSD         float64 `json:"actual_cost_usd"`
	UsageComplete         bool    `json:"usage_complete"`
	Paid                  bool    `json:"paid"`
}

type RecommendationReadModel struct {
	ContractVersion          string                    `json:"contract_version"`
	ID                       string                    `json:"id"`
	RecommendationID         string                    `json:"recommendation_id"`
	Subject                  string                    `json:"subject"`
	Disposition              RecommendationDisposition `json:"disposition"`
	Thesis                   string                    `json:"thesis"`
	BullCase                 []ResearchClaim           `json:"bull_case"`
	CounterEvidence          []ResearchClaim           `json:"counter_evidence"`
	Contradictions           []ResearchClaim           `json:"contradictions"`
	Unknowns                 []ResearchUnknown         `json:"unknowns"`
	InvalidationConditions   []InvalidationCondition   `json:"invalidation_conditions"`
	Evidence                 []EvidenceView            `json:"evidence"`
	Freshness                FreshnessDecision         `json:"freshness"`
	QuantContext             []NumericValue            `json:"quant_context,omitempty"`
	Confidence               ConfidenceAssessment      `json:"confidence"`
	ContextPlanID            string                    `json:"context_plan_id"`
	PacketContentFingerprint string                    `json:"packet_content_fingerprint"`
	ContextSelectedIDs       []string                  `json:"context_selected_ids"`
	ContextOmittedIDs        []string                  `json:"context_omitted_ids,omitempty"`
	ContextDuplicateIDs      []string                  `json:"context_duplicate_ids,omitempty"`
	EstimatedInputTokens     int                       `json:"estimated_input_tokens"`
	MaxInputTokens           int                       `json:"max_input_tokens"`
	MaxOutputTokens          int                       `json:"max_output_tokens"`
	MaxReasoningTokens       int                       `json:"max_reasoning_tokens"`
	Inference                InferenceView             `json:"inference"`
	Authority                string                    `json:"authority"`
	ExecutionAuthority       string                    `json:"execution_authority"`
}

func BuildRecommendationReadModel(packet EvidencePacket, plan ContextPlan, research StructuredResearchOutput, recommendation ResearchRecommendation, freshness FreshnessDecision) (RecommendationReadModel, error) {
	if err := packet.Validate(); err != nil {
		return RecommendationReadModel{}, err
	}
	packetFingerprint, err := packet.ContentFingerprint()
	if err != nil {
		return RecommendationReadModel{}, err
	}
	if err := plan.Validate(); err != nil {
		return RecommendationReadModel{}, err
	}
	if plan.Oversize || !validIdentity("ctx_", plan.ID) || plan.PacketID != packet.ID {
		return RecommendationReadModel{}, fmt.Errorf("read model requires a complete context plan bound to the packet")
	}
	if strings.TrimSpace(plan.PacketContentFingerprint) == "" || plan.PacketContentFingerprint != packetFingerprint {
		return RecommendationReadModel{}, fmt.Errorf("read model requires the context packet fingerprint")
	}
	if err := ValidateStructuredResearchOutput(plan, packet, research); err != nil {
		return RecommendationReadModel{}, err
	}
	if freshness.PacketID != packet.ID {
		return RecommendationReadModel{}, fmt.Errorf("freshness decision does not match packet")
	}
	if err := freshness.Validate(); err != nil {
		return RecommendationReadModel{}, err
	}
	if err := recommendation.Validate(packet, research); err != nil {
		return RecommendationReadModel{}, err
	}
	selected := toSet(plan.SelectedIDs)
	duplicates := toSet(plan.DuplicateIDs)
	evidence := make([]EvidenceView, 0, len(packet.Items))
	quant := []NumericValue{}
	for _, item := range packet.Items {
		view := EvidenceView{ID: item.Identity.ID, Kind: item.Identity.Kind, SourceID: item.Identity.Source.SourceID, Provider: item.Identity.Source.Provider, RawReference: item.Identity.Source.RawReference, RawContentSHA256: item.Identity.Source.RawContentSHA256, Freshness: item.Identity.Freshness, Title: item.Rendering.Title, Summary: item.Rendering.Summary, Excerpt: item.Rendering.Excerpt}
		_, view.SelectedInContext = selected[item.Identity.ID]
		_, view.DuplicateInContext = duplicates[item.Identity.ID]
		view.NumericContext = append([]NumericValue(nil), item.Rendering.NumericContext...)
		if item.Identity.Kind == EvidenceKindQuant {
			quant = append(quant, item.Rendering.NumericContext...)
		}
		evidence = append(evidence, view)
	}
	sortedViewEvidence(evidence)
	sort.Slice(quant, func(left, right int) bool {
		if quant[left].Metric != quant[right].Metric {
			return quant[left].Metric < quant[right].Metric
		}
		return quant[left].Value < quant[right].Value
	})
	model := RecommendationReadModel{ContractVersion: RecommendationReadModelContractV1, RecommendationID: recommendation.ID, Subject: packet.Subject.ID, Disposition: recommendation.Disposition, Thesis: recommendation.Thesis, BullCase: append([]ResearchClaim(nil), recommendation.BullCase...), CounterEvidence: append([]ResearchClaim(nil), recommendation.CounterEvidence...), Contradictions: append([]ResearchClaim(nil), recommendation.Contradictions...), Unknowns: append([]ResearchUnknown(nil), recommendation.Unknowns...), InvalidationConditions: append([]InvalidationCondition(nil), recommendation.InvalidationConditions...), Evidence: evidence, Freshness: freshness, QuantContext: quant, Confidence: recommendation.Confidence, ContextPlanID: plan.ID, PacketContentFingerprint: plan.PacketContentFingerprint, ContextSelectedIDs: append([]string(nil), plan.SelectedIDs...), ContextOmittedIDs: append([]string(nil), plan.OmittedIDs...), ContextDuplicateIDs: append([]string(nil), plan.DuplicateIDs...), EstimatedInputTokens: plan.EstimatedInputTokens, MaxInputTokens: plan.MaxInputTokens, MaxOutputTokens: plan.MaxOutputTokens, MaxReasoningTokens: plan.MaxReasoningTokens, Inference: InferenceView{Provider: research.Inference.Provider, Model: research.Inference.Model, PromptVersion: research.Inference.PromptVersion, SystemVersion: research.Inference.SystemVersion, OutputContractVersion: research.Inference.OutputContractVersion, RequestID: research.Inference.RequestID, RawResponseSHA256: research.Inference.RawResponseSHA256, RetryCount: research.Inference.RetryCount, InputTokens: research.Inference.InputTokens, OutputTokens: research.Inference.OutputTokens, ActualCostUSD: research.Inference.ActualCostUSD, UsageComplete: research.Inference.UsageComplete, Paid: research.Inference.Paid}, Authority: recommendation.Authority, ExecutionAuthority: recommendation.ExecutionAuthority}
	model.ID = deriveReadModelID(model)
	return model, nil
}

func deriveReadModelID(model RecommendationReadModel) string {
	model.ID = ""
	seed, _ := json.Marshal(model)
	digest := sha256.Sum256(seed)
	return "rm6_" + hex.EncodeToString(digest[:])
}

func (model RecommendationReadModel) Validate() error {
	if model.ContractVersion != RecommendationReadModelContractV1 || !validIdentity("rm6_", model.ID) || !validIdentity("rec6_", model.RecommendationID) || strings.TrimSpace(model.Subject) == "" || model.Authority != "RESEARCH_DECISION_SUPPORT" || model.ExecutionAuthority != "NONE" || strings.TrimSpace(model.PacketContentFingerprint) == "" {
		return fmt.Errorf("read model identity and research-only authority are required")
	}
	if model.Disposition != DispositionNoTrade && model.Disposition != DispositionWatch && model.Disposition != DispositionCandidate {
		return fmt.Errorf("read model disposition is unsupported")
	}
	if len(model.Evidence) == 0 || len(model.ContextSelectedIDs) == 0 {
		return fmt.Errorf("read model requires evidence and context identity")
	}
	if err := model.Freshness.Validate(); err != nil {
		return err
	}
	if err := model.Confidence.Validate(); err != nil {
		return err
	}
	if model.ID != deriveReadModelID(model) {
		return fmt.Errorf("read model ID does not match its content identity")
	}
	return nil
}

func toSet(values []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func sortedViewEvidence(values []EvidenceView) {
	sort.Slice(values, func(left, right int) bool { return values[left].ID < values[right].ID })
}
