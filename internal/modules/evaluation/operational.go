package evaluation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const OperationalReplayReportContractV1 = "jax.operational_replay_report/v1"

type OperationalReplayReport struct {
	ContractVersion       string `json:"contract_version"`
	ID                    string `json:"id"`
	CaseID                string `json:"case_id"`
	BenchmarkID           string `json:"benchmark_id"`
	ReplayMode            string `json:"replay_mode"`
	ContextPlanID         string `json:"context_plan_id"`
	ResearchOutputID      string `json:"research_output_id"`
	RecommendationID      string `json:"recommendation_id"`
	OutcomeID             string `json:"outcome_id"`
	AdapterAssessmentID   string `json:"adapter_assessment_id"`
	CostPolicyID          string `json:"cost_policy_id"`
	WalkForwardProtocolID string `json:"walk_forward_protocol_id"`
	ReplayVerified        bool   `json:"replay_verified"`
	OutcomeLeakageChecked bool   `json:"outcome_leakage_checked"`
	OutOfSampleEvidence   bool   `json:"out_of_sample_evidence"`
	EvaluationStatus      string `json:"evaluation_status"`
	EdgeConclusion        string `json:"edge_conclusion"`
	ExecutionAuthority    string `json:"execution_authority"`
	InferenceUsed         bool   `json:"inference_used"`
}

func BuildOperationalReplayReport(benchmark FrozenBenchmark, caseFile HistoricalCase, replay ReplayResult, outcome RecommendationOutcome, protocol WalkForwardProtocol, benchmarks map[string]FrozenBenchmark, adapter BacktestAdapterAssessment, policy CostSlippagePolicy, cost HypotheticalTradingCost) (OperationalReplayReport, error) {
	if err := benchmark.Validate(); err != nil {
		return OperationalReplayReport{}, err
	}
	if err := caseFile.Validate(); err != nil {
		return OperationalReplayReport{}, err
	}
	if replay.ContractVersion != HistoricalReplayContractV1 || replay.Mode != ReplayModeArtifact || !replay.Reconstructed || replay.CaseID != caseFile.ID || replay.DecisionAt != caseFile.DecisionAt || replay.ContextPlanID != caseFile.ContextPlan.ID || replay.ResearchOutputID != caseFile.Research.ID || replay.RecommendationID != caseFile.Recommendation.ID {
		return OperationalReplayReport{}, fmt.Errorf("operational report requires a complete artifact replay")
	}
	if err := outcome.Validate(caseFile); err != nil {
		return OperationalReplayReport{}, err
	}
	if err := protocol.Validate(benchmarks); err != nil {
		return OperationalReplayReport{}, err
	}
	if benchmark.Split != BenchmarkOutOfSample || !protocolContainsCase(protocol, benchmark.ID, BenchmarkOutOfSample, caseFile.ID) {
		return OperationalReplayReport{}, fmt.Errorf("operational report requires explicit out-of-sample case membership")
	}
	if err := adapter.Validate(); err != nil {
		return OperationalReplayReport{}, err
	}
	if err := cost.Validate(policy); err != nil {
		return OperationalReplayReport{}, err
	}
	report := OperationalReplayReport{ContractVersion: OperationalReplayReportContractV1, CaseID: caseFile.ID, BenchmarkID: benchmark.ID, ReplayMode: replay.Mode, ContextPlanID: replay.ContextPlanID, ResearchOutputID: replay.ResearchOutputID, RecommendationID: replay.RecommendationID, OutcomeID: outcome.ID, AdapterAssessmentID: adapter.ID, CostPolicyID: policy.ID, WalkForwardProtocolID: protocol.ID, ReplayVerified: true, OutcomeLeakageChecked: true, OutOfSampleEvidence: true, EvaluationStatus: "EXPLICIT_OOS_SINGLE_CASE_INSUFFICIENT_SAMPLE", EdgeConclusion: "UNKNOWN_INSUFFICIENT_SAMPLE_NO_EDGE_CLAIM", ExecutionAuthority: "NONE", InferenceUsed: false}
	report.ID = deriveOperationalReplayReportID(report)
	if err := report.Validate(benchmark, caseFile, outcome, protocol, benchmarks, adapter, policy, cost); err != nil {
		return OperationalReplayReport{}, err
	}
	return report, nil
}

func (report OperationalReplayReport) Validate(benchmark FrozenBenchmark, caseFile HistoricalCase, outcome RecommendationOutcome, protocol WalkForwardProtocol, benchmarks map[string]FrozenBenchmark, adapter BacktestAdapterAssessment, policy CostSlippagePolicy, cost HypotheticalTradingCost) error {
	if err := benchmark.Validate(); err != nil {
		return err
	}
	if err := caseFile.Validate(); err != nil {
		return err
	}
	if err := outcome.Validate(caseFile); err != nil {
		return err
	}
	if err := protocol.Validate(benchmarks); err != nil {
		return err
	}
	if err := adapter.Validate(); err != nil {
		return err
	}
	if err := cost.Validate(policy); err != nil {
		return err
	}
	if report.ContractVersion != OperationalReplayReportContractV1 || !validIdentity("opreplay_", report.ID) || report.CaseID != caseFile.ID || report.BenchmarkID != benchmark.ID || report.ReplayMode != ReplayModeArtifact || report.ContextPlanID != caseFile.ContextPlan.ID || report.ResearchOutputID != caseFile.Research.ID || report.RecommendationID != caseFile.Recommendation.ID || report.OutcomeID != outcome.ID || report.AdapterAssessmentID != adapter.ID || report.CostPolicyID != policy.ID || report.WalkForwardProtocolID != protocol.ID || !report.ReplayVerified || !report.OutcomeLeakageChecked || !report.OutOfSampleEvidence || strings.TrimSpace(report.EvaluationStatus) == "" || strings.TrimSpace(report.EdgeConclusion) == "" || report.ExecutionAuthority != "NONE" || report.InferenceUsed {
		return fmt.Errorf("operational replay report is incomplete or crosses a safety boundary")
	}
	if report.ID != deriveOperationalReplayReportID(report) {
		return fmt.Errorf("operational replay report ID does not match its evidence bindings")
	}
	return nil
}

func protocolContainsCase(protocol WalkForwardProtocol, benchmarkID string, split BenchmarkSplit, caseID string) bool {
	for _, window := range protocol.Windows {
		if window.BenchmarkID == benchmarkID && window.Split == split {
			for _, candidate := range window.CaseIDs {
				if candidate == caseID {
					return true
				}
			}
		}
	}
	return false
}

func deriveOperationalReplayReportID(report OperationalReplayReport) string {
	report.ID = ""
	seed, _ := json.Marshal(report)
	digest := sha256.Sum256(seed)
	return "opreplay_" + hex.EncodeToString(digest[:])
}
