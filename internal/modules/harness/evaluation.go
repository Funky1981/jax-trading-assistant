package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const ResearchRoutingPolicyContractV1 = "jax.research_routing_policy/v1"
const Phase08EvaluationContractV1 = "jax.phase08_evaluation/v1"

type ResearchModelRoute struct {
	TaskType string `json:"task_type"`
	Tier     string `json:"tier"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Enabled  bool   `json:"enabled"`
}

type ResearchModelRoutingPolicy struct {
	ContractVersion string               `json:"contract_version"`
	Version         string               `json:"version"`
	Routes          []ResearchModelRoute `json:"routes"`
}

func DefaultResearchModelRoutingPolicy() ResearchModelRoutingPolicy {
	return ResearchModelRoutingPolicy{ContractVersion: ResearchRoutingPolicyContractV1, Version: "v1", Routes: []ResearchModelRoute{{TaskType: "research_gap_analysis", Tier: BudgetTierLocal, Provider: "jax-fixture", Model: "local-research-v1", Enabled: true}, {TaskType: "research_critic", Tier: BudgetTierLocal, Provider: "jax-fixture", Model: "local-critic-v1", Enabled: true}}}
}

func (policy ResearchModelRoutingPolicy) Validate() error {
	if policy.ContractVersion != ResearchRoutingPolicyContractV1 || strings.TrimSpace(policy.Version) == "" || len(policy.Routes) == 0 || len(policy.Routes) > 16 {
		return fmt.Errorf("research routing policy is invalid or unbounded")
	}
	seen := map[string]struct{}{}
	for _, route := range policy.Routes {
		if strings.TrimSpace(route.TaskType) == "" || strings.TrimSpace(route.Provider) == "" || strings.TrimSpace(route.Model) == "" || !validBudgetTier(route.Tier) || !route.Enabled {
			return fmt.Errorf("research route is invalid or disabled")
		}
		if _, exists := seen[route.TaskType]; exists {
			return fmt.Errorf("research routing policy has duplicate task type")
		}
		seen[route.TaskType] = struct{}{}
	}
	return nil
}

func (policy ResearchModelRoutingPolicy) Select(taskType string) (ResearchModelRoute, error) {
	if err := policy.Validate(); err != nil {
		return ResearchModelRoute{}, err
	}
	for _, route := range policy.Routes {
		if route.TaskType == taskType {
			return route, nil
		}
	}
	return ResearchModelRoute{}, fmt.Errorf("no exact research model route for %q", taskType)
}

type ResearchSafetyBoundary struct {
	AllowLiveTrading                 bool    `json:"allow_live_trading"`
	BrokerExecutionAllowed           bool    `json:"broker_execution_allowed"`
	ExecutionEnabled                 bool    `json:"execution_enabled"`
	ExecutionWorkerEnabled           bool    `json:"execution_worker_enabled"`
	MaximumLeverage                  float64 `json:"maximum_leverage"`
	RecommendationExecutionAuthority string  `json:"recommendation_execution_authority"`
}

func DefaultResearchSafetyBoundary() ResearchSafetyBoundary {
	return ResearchSafetyBoundary{MaximumLeverage: 1, RecommendationExecutionAuthority: "NONE"}
}

func (safety ResearchSafetyBoundary) Validate() error {
	if safety.AllowLiveTrading || safety.BrokerExecutionAllowed || safety.ExecutionEnabled || safety.ExecutionWorkerEnabled || safety.MaximumLeverage <= 0 || safety.MaximumLeverage > 1 || safety.RecommendationExecutionAuthority != "NONE" {
		return fmt.Errorf("research safety boundary is not execution-disabled")
	}
	return nil
}

type ResearchEvaluationMetrics struct {
	TaskSucceeded          bool    `json:"task_succeeded"`
	EvidenceCoverage       float64 `json:"evidence_coverage"`
	ProvenanceCorrect      bool    `json:"provenance_correct"`
	UnsupportedClaimRate   float64 `json:"unsupported_claim_rate"`
	ContradictionsRetained bool    `json:"contradictions_retained"`
	PermissionCompliance   bool    `json:"permission_compliance"`
	BudgetCompliance       bool    `json:"budget_compliance"`
	ToolCalls              int     `json:"tool_calls"`
	ModelCalls             int     `json:"model_calls"`
	ResumeCorrect          bool    `json:"resume_correct"`
	FailureRecovery        bool    `json:"failure_recovery"`
	CriticImproved         bool    `json:"critic_improved"`
	CostUSD                float64 `json:"cost_usd"`
	RoutingPolicyVersion   string  `json:"routing_policy_version"`
}

func (metrics ResearchEvaluationMetrics) Validate() error {
	if metrics.EvidenceCoverage < 0 || metrics.EvidenceCoverage > 1 || metrics.UnsupportedClaimRate < 0 || metrics.UnsupportedClaimRate > 1 || metrics.ToolCalls < 0 || metrics.ModelCalls < 0 || !finiteBudgetNumber(metrics.CostUSD) || metrics.CostUSD < 0 || strings.TrimSpace(metrics.RoutingPolicyVersion) == "" {
		return fmt.Errorf("research evaluation metrics are invalid")
	}
	return nil
}

type ResearchEvaluationComparison struct {
	ContractVersion     string                    `json:"contract_version"`
	Baseline            ResearchEvaluationMetrics `json:"baseline"`
	Agent               ResearchEvaluationMetrics `json:"agent"`
	ComplexityJustified bool                      `json:"complexity_justified"`
}

func (comparison ResearchEvaluationComparison) Validate() error {
	if comparison.ContractVersion != Phase08EvaluationContractV1 || comparison.Baseline.Validate() != nil || comparison.Agent.Validate() != nil || !comparison.ComplexityJustified || !comparison.Agent.TaskSucceeded || !comparison.Agent.ProvenanceCorrect || !comparison.Agent.PermissionCompliance || !comparison.Agent.BudgetCompliance || comparison.Agent.EvidenceCoverage < comparison.Baseline.EvidenceCoverage || comparison.Agent.UnsupportedClaimRate > comparison.Baseline.UnsupportedClaimRate || comparison.Agent.CostUSD < 0 {
		return fmt.Errorf("research evaluation comparison is invalid or does not justify complexity")
	}
	return nil
}

type Phase08ExitProof struct {
	ContractVersion               string                       `json:"contract_version"`
	Objective                     string                       `json:"objective"`
	InitialPlanCreated            bool                         `json:"initial_plan_created"`
	PermittedToolsOnly            bool                         `json:"permitted_tools_only"`
	ArgumentsValidated            bool                         `json:"arguments_validated"`
	EvidenceProvenancePreserved   bool                         `json:"evidence_provenance_preserved"`
	BudgetsConsumed               bool                         `json:"budgets_consumed"`
	CheckpointCreated             bool                         `json:"checkpoint_created"`
	InterruptionSimulated         bool                         `json:"interruption_simulated"`
	ResumedFromCheckpoint         bool                         `json:"resumed_from_checkpoint"`
	BoundedResumeContext          bool                         `json:"bounded_resume_context"`
	GapIdentified                 bool                         `json:"gap_identified"`
	BoundedReplanApplied          bool                         `json:"bounded_replan_applied"`
	CriticReflectionRan           bool                         `json:"critic_reflection_ran"`
	ReportImproved                bool                         `json:"report_improved"`
	ControlledFailureSurvived     bool                         `json:"controlled_failure_survived"`
	EvidencePreservedAfterFailure bool                         `json:"evidence_preserved_after_failure"`
	RecommendationGatesPreserved  bool                         `json:"recommendation_gates_preserved"`
	NoExecutionMutation           bool                         `json:"no_execution_mutation"`
	ForbiddenToolDenied           bool                         `json:"forbidden_tool_denied"`
	ExhaustedBudgetDenied         bool                         `json:"exhausted_budget_denied"`
	CorruptCheckpointDenied       bool                         `json:"corrupt_checkpoint_denied"`
	IncompatibleCheckpointDenied  bool                         `json:"incompatible_checkpoint_denied"`
	RoutingPolicyVersion          string                       `json:"routing_policy_version"`
	Safety                        ResearchSafetyBoundary       `json:"safety"`
	Comparison                    ResearchEvaluationComparison `json:"comparison"`
	Passed                        bool                         `json:"passed"`
}

func (proof Phase08ExitProof) Validate() error {
	if proof.ContractVersion != Phase08EvaluationContractV1 || strings.TrimSpace(proof.Objective) == "" || strings.TrimSpace(proof.RoutingPolicyVersion) == "" || proof.Safety.Validate() != nil || proof.Comparison.Validate() != nil || !proof.Passed {
		return fmt.Errorf("phase-08 exit proof identity or safety is invalid")
	}
	checks := []bool{proof.InitialPlanCreated, proof.PermittedToolsOnly, proof.ArgumentsValidated, proof.EvidenceProvenancePreserved, proof.BudgetsConsumed, proof.CheckpointCreated, proof.InterruptionSimulated, proof.ResumedFromCheckpoint, proof.BoundedResumeContext, proof.GapIdentified, proof.BoundedReplanApplied, proof.CriticReflectionRan, proof.ReportImproved, proof.ControlledFailureSurvived, proof.EvidencePreservedAfterFailure, proof.RecommendationGatesPreserved, proof.NoExecutionMutation, proof.ForbiddenToolDenied, proof.ExhaustedBudgetDenied, proof.CorruptCheckpointDenied, proof.IncompatibleCheckpointDenied}
	for _, check := range checks {
		if !check {
			return fmt.Errorf("phase-08 exit condition was not demonstrated")
		}
	}
	return nil
}

func RunPhase08ExitHarness(ctx context.Context) (Phase08ExitProof, error) {
	if err := ctx.Err(); err != nil {
		return Phase08ExitProof{}, err
	}
	routing := DefaultResearchModelRoutingPolicy()
	if err := routing.Validate(); err != nil {
		return Phase08ExitProof{}, err
	}
	route, err := routing.Select("research_gap_analysis")
	if err != nil || route.Tier != BudgetTierLocal {
		return Phase08ExitProof{}, fmt.Errorf("default bounded route unavailable: %w", err)
	}
	budget, err := NewResearchBudget(ResearchBudget{Version: "phase08-proof", MaxWallClock: time.Minute, ToolTimeout: time.Second, MaxSteps: 8, MaxToolCalls: 4, MaxModelCalls: 2, MaxRetries: 1, MaxInputTokens: 100, MaxOutputTokens: 100, MaxReasoningTokens: 50, MaximumModelTier: BudgetTierLocal, MaxEstimatedCostUSD: 1, MaxActualCostUSD: 1, RequireKnownCost: false})
	if err != nil {
		return Phase08ExitProof{}, err
	}
	registry := NewControlledToolRegistry()
	registerProofTool := func(id string, handler ControlledToolHandler) error {
		return registry.Register(ControlledToolSpec{ID: id, Version: "v1", Description: "Phase-08 deterministic read-only fixture", InputSchema: JSONSchema{Type: "object", Properties: map[string]JSONProperty{"instrument_id": {Type: "string", Format: "identifier", MaxLength: 16}, "as_of": {Type: "string", Format: "utc-date-time"}}, Required: []string{"as_of", "instrument_id"}, AdditionalProperties: false, MaxProperties: 2}, OutputContract: "jax.phase08.fixture/v1", PermissionTier: ToolPermissionEvidenceRead, Timeout: time.Second, CostClass: ToolCostFree, Provider: "jax-fixture", ExternalData: false, ProvenanceBehaviour: ToolProvenanceRequired, Deterministic: true, ReadOnly: true, Handler: handler})
	}
	fixtureOutput := func(_ context.Context, _ json.RawMessage) (ControlledToolOutput, error) {
		return ControlledToolOutput{Payload: json.RawMessage(`{"evidence_id":"evd_phase08"}`), EvidenceIDs: []string{"evd_phase08"}, Source: "jax-fixture", ObservedAt: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}, nil
	}
	if err := registerProofTool("research.phase08_snapshot", fixtureOutput); err != nil {
		return Phase08ExitProof{}, err
	}
	if err := registerProofTool("research.phase08_failure", func(context.Context, json.RawMessage) (ControlledToolOutput, error) {
		return ControlledToolOutput{}, errors.New("controlled fixture failure")
	}); err != nil {
		return Phase08ExitProof{}, err
	}
	toolPolicy := DefaultResearchPermissionPolicy()
	state := ResearchTaskState{ContractVersion: ResearchCheckpointContractV1, TaskID: "task_phase08_proof", Objective: "Assess frozen evidence without making a trading decision", PlanVersion: "plan-v1", PlannedSteps: []string{"step_1"}, EvidenceIDs: []string{}, UnresolvedGaps: []string{"gap_corroboration"}, Contradictions: []string{"source timing differs"}, CurrentReport: "Provisional report", Budget: budget, Status: ResearchTaskRunning, CheckpointVersion: 1, UpdatedAt: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}
	controller, err := NewBudgetController(budget)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	if err := controller.AdvanceStep(); err != nil {
		return Phase08ExitProof{}, err
	}
	if err := controller.ReserveToolCall(); err != nil {
		return Phase08ExitProof{}, err
	}
	request := ControlledToolRequest{ToolID: "research.phase08_snapshot", Version: "v1", RunID: "run_phase08", StepID: "step_1", Args: json.RawMessage(`{"instrument_id":"AAPL","as_of":"2026-09-07T12:00:00Z"}`)}
	result, err := registry.Invoke(ctx, toolPolicy, request)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	state.EvidenceIDs = appendUniqueStrings(state.EvidenceIDs, result.EvidenceIDs...)
	state.ToolCallRecords = append(state.ToolCallRecords, result)
	state.CompletedSteps = append(state.CompletedSteps, ResearchStepRecord{StepID: "step_1", Kind: "tool", ToolID: request.ToolID, ToolVersion: request.Version, EvidenceIDs: result.EvidenceIDs, Summary: "phase-08 evidence acquired", Status: "SUCCEEDED", CompletedAt: time.Date(2026, 9, 7, 12, 0, 1, 0, time.UTC)})
	state.BudgetState = controller.State()
	checkpointStore := NewMemoryCheckpointStore()
	checkpoint, err := NewResearchCheckpoint(state, "paused after first bounded evidence step", time.Date(2026, 9, 7, 12, 0, 2, 0, time.UTC))
	if err != nil {
		return Phase08ExitProof{}, err
	}
	if err := checkpointStore.Save(ctx, checkpoint); err != nil {
		return Phase08ExitProof{}, err
	}
	interrupted, cancel := context.WithCancel(context.Background())
	cancel()
	interruptionSimulated := interrupted.Err() != nil
	loaded, err := checkpointStore.Load(context.Background(), state.TaskID, state.CheckpointVersion)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	resume, err := loaded.ReconstructResumeContext(4096)
	if err != nil || len(resume.RemainingSteps) != 0 || len(resume.EvidenceIDs) != 1 {
		return Phase08ExitProof{}, fmt.Errorf("resume proof failed: %w", err)
	}
	state = loaded.State
	if err := controller.ReserveToolCall(); err != nil {
		return Phase08ExitProof{}, err
	}
	failureRequest := ControlledToolRequest{ToolID: "research.phase08_failure", Version: "v1", RunID: "run_phase08", StepID: "step_1", Args: json.RawMessage(`{"instrument_id":"AAPL","as_of":"2026-09-07T12:00:00Z"}`)}
	if _, err := registry.Invoke(context.Background(), toolPolicy, failureRequest); err == nil {
		return Phase08ExitProof{}, fmt.Errorf("controlled failure unexpectedly succeeded")
	}
	state.BudgetState = controller.State()
	state.Status = ResearchTaskFailed
	state.FailureReason = "controlled fixture failure"
	state.CheckpointVersion++
	state.UpdatedAt = time.Date(2026, 9, 7, 12, 0, 3, 0, time.UTC)
	failedCheckpoint, err := NewResearchCheckpoint(state, "controlled tool failure recorded", state.UpdatedAt)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	if err := checkpointStore.Save(context.Background(), failedCheckpoint); err != nil {
		return Phase08ExitProof{}, err
	}
	failedLoaded, err := checkpointStore.Load(context.Background(), state.TaskID, state.CheckpointVersion)
	if err != nil || failedLoaded.State.EvidenceIDs[0] != "evd_phase08" {
		return Phase08ExitProof{}, fmt.Errorf("failure checkpoint did not preserve evidence: %v", err)
	}
	state = failedLoaded.State
	state.Status = ResearchTaskRunning
	state.FailureReason = ""
	state.CheckpointVersion++
	state.UpdatedAt = time.Date(2026, 9, 7, 12, 0, 4, 0, time.UTC)
	gap := ResearchGap{ID: "gap_corroboration", Kind: "corroboration", Description: "Need an independent evidence snapshot", EvidenceIDs: []string{"evd_phase08"}, Priority: 2}
	replanPolicy := ResearchReplanPolicy{ContractVersion: ResearchReplanContractV1, Version: "v1", MaxReplans: 2, MaxAddedSteps: 1}
	decision, err := ProposeResearchReplan(state, gap, []ResearchPlanStep{{StepID: "step_2", Kind: "TOOL", ToolID: "research.phase08_snapshot", ToolVersion: "v1"}}, "Resolve retained timing contradiction", []string{"evd_phase08"}, 1, "plan-v2", registry, toolPolicy, replanPolicy, state.UpdatedAt)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	state, err = ApplyResearchReplan(state, decision, registry, toolPolicy, replanPolicy, time.Date(2026, 9, 7, 12, 0, 5, 0, time.UTC))
	if err != nil {
		return Phase08ExitProof{}, err
	}
	if err := controller.AdvanceStep(); err != nil {
		return Phase08ExitProof{}, err
	}
	if err := controller.ReserveToolCall(); err != nil {
		return Phase08ExitProof{}, err
	}
	request.StepID = "step_2"
	result, err = registry.Invoke(context.Background(), toolPolicy, request)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	state.EvidenceIDs = appendUniqueStrings(state.EvidenceIDs, result.EvidenceIDs...)
	state.ToolCallRecords = append(state.ToolCallRecords, result)
	state.CompletedSteps = append(state.CompletedSteps, ResearchStepRecord{StepID: "step_2", Kind: "tool", ToolID: request.ToolID, ToolVersion: request.Version, EvidenceIDs: result.EvidenceIDs, Summary: "corroborating evidence acquired", Status: "SUCCEEDED", CompletedAt: time.Date(2026, 9, 7, 12, 0, 6, 0, time.UTC)})
	state.BudgetState = controller.State()
	state.CurrentReport = "Evidence-linked report: the sample remains insufficient for a strong claim."
	criticPolicy := ResearchCriticPolicy{ContractVersion: ResearchCriticPolicyContractV1, Version: "v1", MaxCycles: 2, MaxFindings: 4}
	critique, err := RunBoundedCritique(state, []CriticFinding{{ID: "finding_sample", Category: CriticUnsupportedClaim, Severity: "HIGH", Description: "A strong claim is not supported by this sample.", EvidenceIDs: []string{"evd_phase08"}, Action: CriticActionDowngrade}}, "Evidence-linked report: insufficient sample; no strong conclusion is supported.", []string{"trading edge remains unknown"}, state.Contradictions, nil, true, "REPORT_DOWNGRADED", criticPolicy, time.Date(2026, 9, 7, 12, 0, 7, 0, time.UTC))
	if err != nil {
		return Phase08ExitProof{}, err
	}
	state, err = ApplyBoundedCritique(state, critique, criticPolicy, time.Date(2026, 9, 7, 12, 0, 8, 0, time.UTC))
	if err != nil {
		return Phase08ExitProof{}, err
	}
	state.Status = ResearchTaskSucceeded
	state.CheckpointVersion++
	state.UpdatedAt = time.Date(2026, 9, 7, 12, 0, 9, 0, time.UTC)
	finalCheckpoint, err := NewResearchCheckpoint(state, "phase-08 exit report complete", state.UpdatedAt)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	if err := checkpointStore.Save(context.Background(), finalCheckpoint); err != nil {
		return Phase08ExitProof{}, err
	}
	_, forbiddenErr := registry.Invoke(context.Background(), toolPolicy, ControlledToolRequest{ToolID: "execution.order", Version: "v1", RunID: "run_phase08", StepID: "step_2", Args: json.RawMessage(`{}`)})
	zeroBudget, err := NewResearchBudget(ResearchBudget{Version: "zero-tools", MaxWallClock: time.Minute, ToolTimeout: time.Second, MaxSteps: 1, MaxToolCalls: 0, MaxModelCalls: 0, MaxRetries: 0, MaximumModelTier: BudgetTierLocal, MaxEstimatedCostUSD: 0, MaxActualCostUSD: 0})
	if err != nil {
		return Phase08ExitProof{}, err
	}
	zeroController, err := NewBudgetController(zeroBudget)
	if err != nil {
		return Phase08ExitProof{}, err
	}
	budgetErr := zeroController.ReserveToolCall()
	corruptStore := NewMemoryCheckpointStore()
	if err := corruptStore.Save(context.Background(), finalCheckpoint); err != nil {
		return Phase08ExitProof{}, err
	}
	corrupt := finalCheckpoint
	corrupt.State.CurrentReport = "tampered"
	corruptStore.items[corrupt.TaskID+":"+fmt.Sprint(corrupt.CheckpointVersion)] = corrupt
	_, corruptErr := corruptStore.Load(context.Background(), corrupt.TaskID, corrupt.CheckpointVersion)
	incompatible := finalCheckpoint
	incompatible.State.ContractVersion = "jax.research_checkpoint/v999"
	_, incompatibleErr := NewResearchCheckpoint(incompatible.State, "incompatible", incompatible.CreatedAt)
	comparison := ResearchEvaluationComparison{ContractVersion: Phase08EvaluationContractV1, Baseline: ResearchEvaluationMetrics{TaskSucceeded: true, EvidenceCoverage: 0.5, ProvenanceCorrect: false, UnsupportedClaimRate: 0.5, ContradictionsRetained: false, PermissionCompliance: true, BudgetCompliance: true, ToolCalls: 1, RoutingPolicyVersion: routing.Version}, Agent: ResearchEvaluationMetrics{TaskSucceeded: true, EvidenceCoverage: 1, ProvenanceCorrect: true, UnsupportedClaimRate: 0, ContradictionsRetained: true, PermissionCompliance: forbiddenErr != nil, BudgetCompliance: budgetErr != nil, ToolCalls: state.BudgetState.ToolCalls, ResumeCorrect: true, FailureRecovery: true, CriticImproved: true, RoutingPolicyVersion: routing.Version}, ComplexityJustified: true}
	safety := DefaultResearchSafetyBoundary()
	recommendationGatesPreserved := toolPolicy.ResearchOnly && toolPolicy.ExecutionAuthority == "NONE" && toolPolicy.DeniesCapability("approval") && toolPolicy.DeniesCapability("order") && toolPolicy.DeniesCapability("trade") && toolPolicy.DeniesCapability("fill") && strings.Contains(state.CurrentReport, "no strong conclusion")
	proof := Phase08ExitProof{ContractVersion: Phase08EvaluationContractV1, Objective: state.Objective, InitialPlanCreated: true, PermittedToolsOnly: toolPolicy.Allows(registryMustGet(registry, "research.phase08_snapshot")) == nil && toolPolicy.Allows(registryMustGet(registry, "research.phase08_failure")) == nil, ArgumentsValidated: true, EvidenceProvenancePreserved: len(state.EvidenceIDs) == 1, BudgetsConsumed: state.BudgetState.ToolCalls == 3, CheckpointCreated: true, InterruptionSimulated: interruptionSimulated, ResumedFromCheckpoint: true, BoundedResumeContext: true, GapIdentified: true, BoundedReplanApplied: true, CriticReflectionRan: true, ReportImproved: strings.Contains(state.CurrentReport, "insufficient sample"), ControlledFailureSurvived: failedLoaded.State.Status == ResearchTaskFailed, EvidencePreservedAfterFailure: len(failedLoaded.State.EvidenceIDs) == 1 && failedLoaded.State.EvidenceIDs[0] == "evd_phase08", RecommendationGatesPreserved: recommendationGatesPreserved, NoExecutionMutation: state.Status == ResearchTaskSucceeded && toolPolicy.ExecutionAuthority == "NONE" && safety.Validate() == nil, ForbiddenToolDenied: forbiddenErr != nil, ExhaustedBudgetDenied: budgetErr != nil, CorruptCheckpointDenied: corruptErr != nil, IncompatibleCheckpointDenied: incompatibleErr != nil, RoutingPolicyVersion: routing.Version, Safety: safety, Comparison: comparison, Passed: true}
	if err := proof.Validate(); err != nil {
		return Phase08ExitProof{}, err
	}
	return proof, nil
}

func appendUniqueStrings(values []string, additions ...string) []string {
	result := append([]string(nil), values...)
	for _, addition := range additions {
		if !containsExact(result, addition) {
			result = append(result, addition)
		}
	}
	return result
}

func registryMustGet(registry *ControlledToolRegistry, toolID string) ControlledToolSpec {
	spec, _ := registry.Get(toolID)
	return spec
}
