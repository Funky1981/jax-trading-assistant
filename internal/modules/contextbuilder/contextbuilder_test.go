package contextbuilder

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/harnesscontracts"
)

var cbReferenceTime = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
var cbHash = "sha256:" + strings.Repeat("2", 64)

type staticEvidenceRetriever struct {
	result EvidenceRetrievalResult
	err    error
	calls  int
}

func (r *staticEvidenceRetriever) Retrieve(_ context.Context, _ RetrievalRequest) (EvidenceRetrievalResult, error) {
	r.calls++
	return r.result, r.err
}

type staticCounterRetriever struct {
	result ContradictionRetrievalResult
	err    error
	calls  int
}

func (r *staticCounterRetriever) RetrieveContradictions(_ context.Context, _ RetrievalRequest) (ContradictionRetrievalResult, error) {
	r.calls++
	return r.result, r.err
}

type staticMemoryRetriever struct {
	result MemoryRetrievalResult
	err    error
	calls  int
}

func (r *staticMemoryRetriever) RetrieveMemory(_ context.Context, _ RetrievalRequest) (MemoryRetrievalResult, error) {
	r.calls++
	return r.result, r.err
}

func cbPolicy() harnesscontracts.PolicyReference {
	return harnesscontracts.PolicyReference{Name: "research-policy", Version: "v1"}
}

func cbObjective() harnesscontracts.ResearchObjective {
	return harnesscontracts.ResearchObjective{
		ContractVersion:    harnesscontracts.ResearchObjectiveContractV1,
		ObjectiveID:        "objective-cb",
		Version:            "v1",
		Purpose:            "Assess the event and its causal implications.",
		Question:           "What evidence is relevant as of the reference time?",
		CreatedAt:          cbReferenceTime.Add(-time.Hour),
		Constraints:        []string{"paper-only", "no-execution"},
		RequiredOutputs:    []string{"answer", "uncertainty"},
		CompletionCriteria: []harnesscontracts.CompletionCriterion{{ID: "criterion-cb", Description: "Relevant evidence and uncertainty are represented."}},
		PolicyVersions:     []harnesscontracts.PolicyReference{cbPolicy()},
		Status:             harnesscontracts.ObjectiveStatusActive,
	}
}

func cbTaskState() harnesscontracts.TaskState {
	return harnesscontracts.TaskState{
		ContractVersion:  harnesscontracts.TaskStateContractV1,
		TaskID:           "task-cb",
		ObjectiveID:      "objective-cb",
		ObjectiveVersion: "v1",
		StateVersion:     1,
		CurrentStage:     "context-selection",
		OpenQuestions:    []string{"What remains unknown?"},
		KnownUnknowns:    []string{"Publication quality may be incomplete."},
		Constraints:      []string{"No broker action."},
		NextAction:       "Review the bounded context.",
		CreatedAt:        cbReferenceTime.Add(-time.Hour),
		UpdatedAt:        cbReferenceTime.Add(-time.Minute),
		Status:           harnesscontracts.TaskStatusRunning,
	}
}

func cbBudget() harnesscontracts.ContextBudget {
	budget := harnesscontracts.ContextBudget{
		ContractVersion:           harnesscontracts.ContextBudgetContractV1,
		SystemPolicy:              10,
		Objective:                 10,
		TaskState:                 10,
		Evidence:                  8,
		CounterEvidence:           6,
		CounterEvidenceReserve:    3,
		Memory:                    5,
		ToolOutputs:               5,
		WorkingAllowance:          5,
		StructuredOutputAllowance: 5,
		RequireCounterEvidence:    true,
	}
	budget.Total = budget.SystemPolicy + budget.Objective + budget.TaskState + budget.Evidence + budget.CounterEvidence + budget.Memory + budget.ToolOutputs + budget.WorkingAllowance + budget.StructuredOutputAllowance
	return budget
}

func cbEvidence(id, classification, source, dedup string, units int64, factors RankingFactors) EvidenceCandidate {
	observed := cbReferenceTime.Add(-20 * time.Minute)
	published := cbReferenceTime.Add(-30 * time.Minute)
	firstSeen := cbReferenceTime.Add(-40 * time.Minute)
	return EvidenceCandidate{
		Reference: harnesscontracts.EvidenceReference{
			ContractVersion:  harnesscontracts.EvidenceReferenceContractV1,
			EvidenceID:       id,
			CanonicalStore:   "jax-evidence",
			ProviderIdentity: source,
			SourceReference:  "source-" + id,
			ContentHash:      cbHash,
			ObservedAt:       &observed,
			PublishedAt:      &published,
			FirstSeenAt:      &firstSeen,
			Classification:   classification,
			QualityReference: "quality-" + id,
			PolicyReference:  cbPolicy(),
			InformationState: "observed",
		},
		Factors: factors, EstimatedUnits: units, SubjectKey: "issuer-a", EventType: "guidance", SourceGroup: source, DeduplicationKey: dedup, PolicyEligible: true,
	}
}

func cbMemory(id string, units int64, score int) MemoryCandidate {
	return MemoryCandidate{
		Reference: harnesscontracts.MemoryReference{
			ContractVersion:         harnesscontracts.MemoryReferenceContractV1,
			MemoryID:                id,
			MemoryType:              "experience",
			SourceTaskReference:     "source-task-" + id,
			SourceDecisionReference: "source-decision-" + id,
			SourceOutcomeReference:  "source-outcome-" + id,
			CreatedAt:               cbReferenceTime.Add(-24 * time.Hour),
			RegimeReference:         "regime-current",
			ReliabilityReference:    "reliability-" + id,
			CalibrationReference:    "calibration-" + id,
			ContentHash:             cbHash,
			Version:                 "v1",
		},
		Factors: RankingFactors{ObjectiveRelevance: score, SubjectRelevance: score, EventRelevance: score, CausalRelevance: score, SourceQuality: score}, EstimatedUnits: units, EventType: "guidance", SubjectKey: "issuer-a", RegimeStatus: RegimeMatch, PolicyEligible: true,
	}
}

func cbRequest(evidence []EvidenceCandidate, counter []EvidenceCandidate, memory []MemoryCandidate) (BuildRequest, *staticEvidenceRetriever, *staticCounterRetriever, *staticMemoryRetriever) {
	evidenceRetriever := &staticEvidenceRetriever{result: EvidenceRetrievalResult{RetrieverID: "evidence-fixture", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, SearchStatus: SearchPerformed, Candidates: evidence}}
	counterRetriever := &staticCounterRetriever{result: ContradictionRetrievalResult{RetrieverID: "counter-fixture", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, SearchStatus: SearchPerformed, Candidates: counter}}
	memoryRetriever := &staticMemoryRetriever{result: MemoryRetrievalResult{RetrieverID: "memory-fixture", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, SearchStatus: SearchPerformed, Candidates: memory}}
	policy := DefaultBuildPolicy()
	policy.Model = harnesscontracts.ModelReference{Provider: "future-model-provider", Model: "future-model", Version: "v1"}
	policy.PolicyVersions = []harnesscontracts.PolicyReference{cbPolicy()}
	return BuildRequest{Objective: cbObjective(), TaskState: cbTaskState(), Budget: cbBudget(), Policy: policy, ReferenceTime: cbReferenceTime, EvidenceRetriever: evidenceRetriever, ContradictionRetriever: counterRetriever, MemoryRetriever: memoryRetriever}, evidenceRetriever, counterRetriever, memoryRetriever
}

func cbBaseRequest(t *testing.T) (BuildRequest, *staticEvidenceRetriever, *staticCounterRetriever, *staticMemoryRetriever) {
	t.Helper()
	return cbRequest([]EvidenceCandidate{cbEvidence("support-1", harnesscontracts.EvidenceSupporting, "provider-a", "story-1", 2, RankingFactors{ObjectiveRelevance: 90, SubjectRelevance: 90, EventRelevance: 80, CausalRelevance: 80, SourceQuality: 90, Recency: 80, Independence: 70, Corroboration: 60})}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-b", "counter-story-1", 3, RankingFactors{ObjectiveRelevance: 90, SubjectRelevance: 80, EventRelevance: 80, CausalRelevance: 90, SourceQuality: 90, Recency: 80, Independence: 90, Corroboration: 80})}, []MemoryCandidate{cbMemory("memory-1", 2, 70)})
}

func buildCB(t *testing.T, request BuildRequest) BuildResult {
	t.Helper()
	result, err := Build(context.Background(), request)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	return result
}

func requireBuildError(t *testing.T, request BuildRequest) {
	t.Helper()
	if _, err := Build(context.Background(), request); err == nil {
		t.Fatal("expected build error")
	}
}

func TestContextBuilderCoreContracts(t *testing.T) {
	t.Run("valid bounded package", func(t *testing.T) {
		request, evidence, counter, memory := cbBaseRequest(t)
		result := buildCB(t, request)
		if len(result.Package.SelectedEvidence) != 1 || len(result.Package.CounterEvidence) != 1 || len(result.Package.SelectedMemory) != 1 {
			t.Fatalf("unexpected selections: %+v", result.Package)
		}
		if evidence.calls != 1 || counter.calls != 1 || memory.calls != 1 || result.Report.CounterEvidenceSearchPerformed != true {
			t.Fatal("retrieval seams were not called exactly once")
		}
	})
	t.Run("objective/task identity mismatch", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.TaskState.ObjectiveID = "different-objective"
		requireBuildError(t, request)
	})
	t.Run("invalid budget", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Budget.Evidence = -1
		requireBuildError(t, request)
	})
	t.Run("missing evidence retriever", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.EvidenceRetriever = nil
		requireBuildError(t, request)
	})
	t.Run("missing contradiction retriever", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.ContradictionRetriever = nil
		requireBuildError(t, request)
	})
	t.Run("policy requires explicit future exclusion", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Policy.ExcludeFutureEvidence = false
		requireBuildError(t, request)
	})
	t.Run("model identity is required for target context", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Policy.Model = harnesscontracts.ModelReference{}
		requireBuildError(t, request)
	})
	t.Run("tool and checkpoint references remain bounded", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		completed := cbReferenceTime.Add(time.Minute)
		request.ToolReferences = []harnesscontracts.ToolReference{{ContractVersion: harnesscontracts.ToolReferenceContractV1, ToolInvocationID: "tool-1", ToolID: "reader", ToolVersion: "v1", Purpose: "Read reference.", InputHash: cbHash, InputReference: "input-1", OutputHash: cbHash, OutputReference: "output-1", StartedAt: cbReferenceTime, CompletedAt: &completed, Status: harnesscontracts.ToolStatusSucceeded}}
		request.CheckpointReference = &harnesscontracts.CheckpointReference{ContractVersion: harnesscontracts.CheckpointContractV1, CheckpointID: "checkpoint-1", TaskID: "task-cb", StateVersion: 1, ContentHash: cbHash, CreatedAt: cbReferenceTime}
		result := buildCB(t, request)
		if len(result.Package.ToolReferences) != 1 || result.Package.CheckpointReference == nil {
			t.Fatal("references were not retained")
		}
	})
}

func TestCounterEvidencePolicy(t *testing.T) {
	t.Run("mandatory reserve is selected separately", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Budget.Evidence = 1
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if len(result.Package.CounterEvidence) != 1 || result.Report.BudgetUsed.CounterEvidenceUsed != 3 {
			t.Fatal("counter-evidence reserve was not protected")
		}
	})
	t.Run("counter-evidence not searched is distinct and fails mandatory policy", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		counter.result.SearchStatus = SearchNotPerformed
		counter.result.Candidates = nil
		requireBuildError(t, request)
	})
	t.Run("counter-evidence no results is distinct and fails mandatory policy", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		counter.result.SearchStatus = SearchNoResults
		counter.result.Candidates = nil
		requireBuildError(t, request)
	})
	t.Run("optional no-results report records search result", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		request.Policy.RequireCounterEvidence = false
		request.Budget.RequireCounterEvidence = false
		request.Budget.CounterEvidenceReserve = 0
		counter.result.SearchStatus = SearchNoResults
		counter.result.Candidates = nil
		result := buildCB(t, request)
		if result.Report.CounterEvidenceSearchStatus != SearchNoResults || !containsString(result.Report.Warnings, "NO_COUNTER_EVIDENCE_FOUND") {
			t.Fatal("no-results state was not reported")
		}
	})
	t.Run("unknown missing counter-evidence remains explicit", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		unknown := cbEvidence("counter-unknown", harnesscontracts.EvidenceUnknownMissing, "provider-b", "counter-unknown", 3, RankingFactors{ObjectiveRelevance: 90, CausalRelevance: 80})
		unknown.Reference.ObservedAt = nil
		unknown.Reference.PublishedAt = nil
		counter.result.Candidates = []EvidenceCandidate{unknown}
		counter.result.KnownMissing = []string{"missing-confirmation"}
		result := buildCB(t, request)
		if result.Package.CounterEvidence[0].Classification != harnesscontracts.EvidenceUnknownMissing || !containsString(result.Package.KnownUnknowns, "missing-confirmation") {
			t.Fatal("unknown/missing counter-evidence was upgraded or dropped")
		}
	})
	t.Run("supporting stream cannot smuggle contradiction", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		candidate := cbEvidence("bad-support", harnesscontracts.EvidenceContradictory, "provider-a", "bad", 1, RankingFactors{ObjectiveRelevance: 90})
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = []EvidenceCandidate{candidate}
		requireBuildError(t, request)
	})
	t.Run("insufficient counter reserve fails closed", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		request.Budget.CounterEvidenceReserve = 4
		counter.result.Candidates[0].EstimatedUnits = 3
		requireBuildError(t, request)
	})
}

func TestRankingBudgetAndOmissionBehaviour(t *testing.T) {
	t.Run("high-value evidence outranks low-value evidence", func(t *testing.T) {
		high := cbEvidence("high", harnesscontracts.EvidenceSupporting, "provider-a", "high-story", 2, RankingFactors{ObjectiveRelevance: 100, SubjectRelevance: 100, CausalRelevance: 100, SourceQuality: 100})
		low := cbEvidence("low", harnesscontracts.EvidenceSupporting, "provider-b", "low-story", 2, RankingFactors{ObjectiveRelevance: 10, SubjectRelevance: 10, CausalRelevance: 10, SourceQuality: 10})
		request, _, _, _ := cbRequest([]EvidenceCandidate{low, high}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Budget.Evidence = 2
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if result.Package.SelectedEvidence[0].EvidenceID != "high" || !containsString(result.Report.AuditLikeOmissionCodes(), ReasonOmittedBudget) {
			t.Fatal("deterministic ranking/budget omission failed")
		}
	})
	t.Run("tight medium and large budgets degrade deterministically", func(t *testing.T) {
		candidates := []EvidenceCandidate{
			cbEvidence("support-a", harnesscontracts.EvidenceSupporting, "provider-a", "a", 2, RankingFactors{ObjectiveRelevance: 100}),
			cbEvidence("support-b", harnesscontracts.EvidenceSupporting, "provider-b", "b", 2, RankingFactors{ObjectiveRelevance: 90}),
			cbEvidence("support-c", harnesscontracts.EvidenceSupporting, "provider-c", "c", 2, RankingFactors{ObjectiveRelevance: 80}),
		}
		request, _, _, _ := cbRequest(candidates, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-z", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Budget.Evidence = 2
		recalculateCBBudget(&request.Budget)
		tight := buildCB(t, request)
		request.Budget.Evidence = 4
		recalculateCBBudget(&request.Budget)
		medium := buildCB(t, request)
		request.Budget.Evidence = 8
		recalculateCBBudget(&request.Budget)
		large := buildCB(t, request)
		if len(tight.Package.SelectedEvidence) >= len(medium.Package.SelectedEvidence) || len(medium.Package.SelectedEvidence) > len(large.Package.SelectedEvidence) {
			t.Fatal("budget pressure did not degrade monotonically")
		}
	})
	t.Run("duplicate source records do not create corroboration", func(t *testing.T) {
		first := cbEvidence("copy-a", harnesscontracts.EvidenceSupporting, "provider-a", "same-story", 2, RankingFactors{ObjectiveRelevance: 90, Corroboration: 100})
		second := cbEvidence("copy-b", harnesscontracts.EvidenceSupporting, "provider-b", "same-story", 2, RankingFactors{ObjectiveRelevance: 80, Corroboration: 100})
		request, _, _, _ := cbRequest([]EvidenceCandidate{first, second}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Budget.Evidence = 8
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if len(result.Package.SelectedEvidence) != 1 || !containsString(result.Report.PackageSelectedIDs(), "copy-a") {
			t.Fatal("duplicate underlying story was treated as independent corroboration")
		}
	})
	t.Run("independent source groups are preferred", func(t *testing.T) {
		first := cbEvidence("same-source-high", harnesscontracts.EvidenceSupporting, "provider-a", "story-a", 2, RankingFactors{ObjectiveRelevance: 100})
		second := cbEvidence("same-source-next", harnesscontracts.EvidenceSupporting, "provider-a", "story-b", 2, RankingFactors{ObjectiveRelevance: 99})
		independent := cbEvidence("independent", harnesscontracts.EvidenceSupporting, "provider-b", "story-c", 2, RankingFactors{ObjectiveRelevance: 80})
		request, _, _, _ := cbRequest([]EvidenceCandidate{first, second, independent}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Budget.Evidence = 4
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if !containsString(result.Report.PackageSelectedIDs(), "independent") {
			t.Fatal("independent source was not preferred")
		}
	})
	t.Run("ranking ties use stable identity tie-breaker", func(t *testing.T) {
		first := cbEvidence("tie-b", harnesscontracts.EvidenceSupporting, "provider-b", "b", 2, RankingFactors{ObjectiveRelevance: 50})
		second := cbEvidence("tie-a", harnesscontracts.EvidenceSupporting, "provider-a", "a", 2, RankingFactors{ObjectiveRelevance: 50})
		request, _, _, _ := cbRequest([]EvidenceCandidate{first, second}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Budget.Evidence = 2
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if result.Package.SelectedEvidence[0].EvidenceID != "tie-a" {
			t.Fatal("tie-breaker was not stable")
		}
	})
	t.Run("required evidence survives ranking and budget", func(t *testing.T) {
		required := cbEvidence("required", harnesscontracts.EvidenceSupporting, "provider-a", "required", 2, RankingFactors{ObjectiveRelevance: 1})
		request, _, _, _ := cbRequest([]EvidenceCandidate{required, cbEvidence("optional", harnesscontracts.EvidenceSupporting, "provider-b", "optional", 2, RankingFactors{ObjectiveRelevance: 100})}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		request.Policy.RequiredEvidenceIDs = []string{"required"}
		request.Budget.Evidence = 2
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if result.Package.SelectedEvidence[0].EvidenceID != "required" {
			t.Fatal("required evidence was not preserved")
		}
	})
	t.Run("required evidence unavailable fails closed", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Policy.RequiredEvidenceIDs = []string{"not-present"}
		requireBuildError(t, request)
	})
	t.Run("policy-ineligible material is omitted", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		candidate := cbEvidence("ineligible", harnesscontracts.EvidenceSupporting, "provider-a", "ineligible", 1, RankingFactors{ObjectiveRelevance: 100})
		candidate.PolicyEligible = false
		candidate.PolicyReason = "policy version does not permit this source"
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, candidate)
		result := buildCB(t, request)
		if !containsString(result.Report.OmissionReferences(), "ineligible") {
			t.Fatal("policy-ineligible evidence omission not reported")
		}
	})
	t.Run("budget omission becomes deferred reference", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		candidate := cbEvidence("deferred", harnesscontracts.EvidenceSupporting, "provider-z", "deferred", 100, RankingFactors{ObjectiveRelevance: 100})
		candidate.OnDemandReference = "deferred-evidence"
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, candidate)
		result := buildCB(t, request)
		if !containsString(result.Report.DeferredReferencesIDs(), "deferred") {
			t.Fatal("on-demand reference was not recorded")
		}
	})
	t.Run("budget omission without on-demand is explicit", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		candidate := cbEvidence("omitted", harnesscontracts.EvidenceSupporting, "provider-z", "omitted", 100, RankingFactors{ObjectiveRelevance: 100})
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, candidate)
		result := buildCB(t, request)
		if !containsString(result.Report.OmissionReferences(), "omitted") {
			t.Fatal("budget omission was not recorded")
		}
	})
}

func TestTemporalMemoryAndDeterminism(t *testing.T) {
	t.Run("future evidence is excluded", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		future := cbEvidence("future", harnesscontracts.EvidenceSupporting, "provider-future", "future", 1, RankingFactors{ObjectiveRelevance: 100})
		futureTime := cbReferenceTime.Add(time.Hour)
		future.Reference.ObservedAt = &futureTime
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, future)
		result := buildCB(t, request)
		if containsString(result.Report.PackageSelectedIDs(), "future") || !containsString(result.Report.OmissionReferences(), "future") {
			t.Fatal("future evidence leaked into context or was not recorded")
		}
	})
	t.Run("stale evidence is excluded by explicit interval", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Policy.StaleAfter = time.Hour
		stale := cbEvidence("stale", harnesscontracts.EvidenceSupporting, "provider-stale", "stale", 1, RankingFactors{ObjectiveRelevance: 100})
		old := cbReferenceTime.Add(-24 * time.Hour)
		stale.Reference.ObservedAt = &old
		stale.Reference.PublishedAt = &old
		stale.Reference.FirstSeenAt = &old
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, stale)
		result := buildCB(t, request)
		if containsString(result.Report.PackageSelectedIDs(), "stale") || !containsString(result.Report.OmissionReferences(), "stale") {
			t.Fatal("stale evidence was not excluded")
		}
	})
	t.Run("unknown observation time remains unknown", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		unknown := cbEvidence("unknown-time", harnesscontracts.EvidenceSupporting, "provider-unknown", "unknown-time", 1, RankingFactors{ObjectiveRelevance: 100})
		unknown.Reference.ObservedAt = nil
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, unknown)
		result := buildCB(t, request)
		if !containsString(result.Report.PackageSelectedIDs(), "unknown-time") {
			t.Fatal("unknown timing candidate was incorrectly rejected by default")
		}
		if !recordHasTemporalStatus(result.Report, "unknown-time", TemporalObservationUnknown) {
			t.Fatal("unknown observation timing was not preserved")
		}
	})
	t.Run("known observation time can be required", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.Policy.RequireKnownObservationTime = true
		unknown := cbEvidence("unknown-required", harnesscontracts.EvidenceSupporting, "provider-unknown", "unknown-required", 1, RankingFactors{ObjectiveRelevance: 100})
		unknown.Reference.ObservedAt = nil
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = append(request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates, unknown)
		result := buildCB(t, request)
		if containsString(result.Report.PackageSelectedIDs(), "unknown-required") {
			t.Fatal("unknown observation time was selected despite explicit requirement")
		}
	})
	t.Run("input order does not change selection or hash", func(t *testing.T) {
		requestA, _, _, _ := cbRequest([]EvidenceCandidate{cbEvidence("b", harnesscontracts.EvidenceSupporting, "provider-b", "b", 2, RankingFactors{ObjectiveRelevance: 80}), cbEvidence("a", harnesscontracts.EvidenceSupporting, "provider-a", "a", 2, RankingFactors{ObjectiveRelevance: 80})}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-c", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, nil)
		requestA.Budget.Evidence = 2
		recalculateCBBudget(&requestA.Budget)
		requestB := requestA
		a := requestA.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates
		requestB.EvidenceRetriever = &staticEvidenceRetriever{result: EvidenceRetrievalResult{RetrieverID: "evidence-fixture", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, SearchStatus: SearchPerformed, Candidates: []EvidenceCandidate{a[1], a[0]}}}
		first := buildCB(t, requestA)
		second := buildCB(t, requestB)
		if first.Package.ContentHash != second.Package.ContentHash || !reflect.DeepEqual(first.Package.SelectedEvidence, second.Package.SelectedEvidence) {
			t.Fatal("input ordering changed deterministic context")
		}
	})
	t.Run("repeated build is deterministic", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		first := buildCB(t, request)
		second := buildCB(t, request)
		if first.Package.ContentHash != second.Package.ContentHash || first.Package.ContextPackageID != second.Package.ContextPackageID {
			t.Fatal("repeated build changed identity")
		}
	})
	t.Run("memory is selected separately", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		result := buildCB(t, request)
		if len(result.Package.SelectedMemory) != 1 || len(result.Package.SelectedEvidence) != 1 {
			t.Fatal("memory/evidence selection boundary was not preserved")
		}
		for _, reference := range result.Package.SelectedEvidence {
			if reference.EvidenceID == "memory-1" {
				t.Fatal("memory entered evidence selection")
			}
		}
	})
	t.Run("memory budget pressure omits lower memory", func(t *testing.T) {
		request, _, _, _ := cbRequest([]EvidenceCandidate{cbEvidence("support-1", harnesscontracts.EvidenceSupporting, "provider-a", "support", 2, RankingFactors{ObjectiveRelevance: 80})}, []EvidenceCandidate{cbEvidence("counter-1", harnesscontracts.EvidenceContradictory, "provider-b", "counter", 3, RankingFactors{ObjectiveRelevance: 80})}, []MemoryCandidate{cbMemory("memory-a", 4, 100), cbMemory("memory-b", 4, 90)})
		request.Budget.Memory = 4
		recalculateCBBudget(&request.Budget)
		result := buildCB(t, request)
		if len(result.Package.SelectedMemory) != 1 || !containsString(result.Report.OmissionReferences(), "memory-b") {
			t.Fatal("memory budget omission was not deterministic")
		}
	})
	t.Run("known regime mismatch is excluded", func(t *testing.T) {
		request, _, _, memory := cbBaseRequest(t)
		request.Policy.ExcludeRegimeMismatchMemory = true
		memory.result.Candidates[0].RegimeStatus = RegimeMismatch
		result := buildCB(t, request)
		if len(result.Package.SelectedMemory) != 0 || !containsString(result.Report.OmissionReferences(), "memory-1") {
			t.Fatal("regime-mismatched memory was selected")
		}
	})
	t.Run("future memory is omitted at the explicit reference time", func(t *testing.T) {
		request, _, _, memory := cbBaseRequest(t)
		memory.result.Candidates[0].Reference.CreatedAt = cbReferenceTime.Add(time.Minute)
		result := buildCB(t, request)
		if len(result.Package.SelectedMemory) != 0 || !containsString(result.Report.OmissionReferences(), "memory-1") {
			t.Fatalf("future memory was selected or not audited: %#v", result.Report)
		}
		foundFuture := false
		for _, record := range result.Report.RetrievalRecords {
			if record.MemoryReference != nil && record.MemoryReference.MemoryID == "memory-1" && record.ReasonCode == ReasonOmittedFuture && record.TemporalStatus == TemporalFuture {
				foundFuture = true
			}
		}
		if !foundFuture {
			t.Fatal("future memory omission did not retain temporal audit state")
		}
	})
	t.Run("future task state is rejected at the explicit reference time", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.TaskState.UpdatedAt = cbReferenceTime.Add(time.Minute)
		requireBuildError(t, request)
	})
	t.Run("memory not searched is reported", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.MemoryRetriever = nil
		result := buildCB(t, request)
		if !containsString(result.Report.Warnings, "MEMORY_NOT_SEARCHED") {
			t.Fatal("memory not-searched state was hidden")
		}
	})
	t.Run("source reference secrets are rejected", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		candidate := cbEvidence("secret", harnesscontracts.EvidenceSupporting, "provider-a", "secret", 1, RankingFactors{ObjectiveRelevance: 90})
		candidate.OnDemandReference = "api_key=secret"
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.Candidates = []EvidenceCandidate{candidate}
		requireBuildError(t, request)
	})
	t.Run("missing data remains a known unknown", func(t *testing.T) {
		request, _, counter, _ := cbBaseRequest(t)
		counter.result.KnownMissing = []string{"missing-instrument-confirmation"}
		result := buildCB(t, request)
		if !containsString(result.Package.KnownUnknowns, "missing-instrument-confirmation") {
			t.Fatal("missing information was not retained")
		}
	})
}

func TestRetrievalResultAndIsolationContracts(t *testing.T) {
	t.Run("malformed ranking factor fails", func(t *testing.T) {
		candidate := cbEvidence("bad-factor", harnesscontracts.EvidenceSupporting, "provider-a", "bad", 1, RankingFactors{ObjectiveRelevance: 101})
		if candidate.Validate() == nil {
			t.Fatal("invalid factor accepted")
		}
	})
	t.Run("zero estimate fails", func(t *testing.T) {
		candidate := cbEvidence("zero", harnesscontracts.EvidenceSupporting, "provider-a", "zero", 0, RankingFactors{})
		if candidate.Validate() == nil {
			t.Fatal("zero estimate accepted")
		}
	})
	t.Run("memory reference cannot validate as evidence record", func(t *testing.T) {
		memory := cbMemory("memory-record", 1, 50)
		record := RetrievalRecord{ContractVersion: RetrievalRecordContractV1, RecordID: "record-1", ItemKind: EvidenceItemKind, MemoryReference: &memory.Reference, RetrieverID: "memory", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, Factors: memory.Factors, Classification: "memory", Decision: RetrievalSelected, ReasonCode: ReasonSelectedHighRelevance, Reason: "selected", EstimatedUnits: 1}
		if record.Validate() == nil {
			t.Fatal("memory/evidence boundary was not enforced")
		}
	})
	t.Run("evidence record cannot validate as memory", func(t *testing.T) {
		evidence := cbEvidence("evidence-record", harnesscontracts.EvidenceSupporting, "provider-a", "record", 1, RankingFactors{ObjectiveRelevance: 50})
		record := RetrievalRecord{ContractVersion: RetrievalRecordContractV1, RecordID: "record-2", ItemKind: MemoryItemKind, EvidenceReference: &evidence.Reference, RetrieverID: "evidence", RetrieverVersion: "v1", RetrievedAt: cbReferenceTime, Factors: evidence.Factors, Classification: harnesscontracts.EvidenceSupporting, Decision: RetrievalSelected, ReasonCode: ReasonSelectedHighRelevance, Reason: "selected", EstimatedUnits: 1}
		if record.Validate() == nil {
			t.Fatal("evidence/memory boundary was not enforced")
		}
	})
	t.Run("retriever errors fail closed", func(t *testing.T) {
		request, evidence, _, _ := cbBaseRequest(t)
		evidence.err = errors.New("fixture failure")
		requireBuildError(t, request)
	})
	t.Run("retriever metadata is audited", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		request.EvidenceRetriever.(*staticEvidenceRetriever).result.RetrieverVersion = "v2"
		result := buildCB(t, request)
		found := false
		for _, record := range result.Report.RetrievalRecords {
			if record.RetrieverVersion == "v2" {
				found = true
			}
		}
		if !found {
			t.Fatal("retriever version was not recorded")
		}
	})
	t.Run("report contains input hashes", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		result := buildCB(t, request)
		if !harnesscontracts.IsCanonicalHash(result.Report.ObjectiveInputHash) || !harnesscontracts.IsCanonicalHash(result.Report.TaskStateInputHash) || !harnesscontracts.IsCanonicalHash(result.Report.BudgetInputHash) {
			t.Fatal("build input hashes missing")
		}
	})
	t.Run("report validates against package identity", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		result := buildCB(t, request)
		if err := result.Validate(); err != nil {
			t.Fatal(err)
		}
		result.Report.FinalContextPackageHash = cbHash
		if result.Validate() == nil {
			t.Fatal("report/package mismatch accepted")
		}
	})
	t.Run("BuildContext alias is equivalent", func(t *testing.T) {
		request, _, _, _ := cbBaseRequest(t)
		first := buildCB(t, request)
		second, err := BuildContext(context.Background(), request)
		if err != nil || first.Package.ContentHash != second.Package.ContentHash {
			t.Fatal("BuildContext alias diverged")
		}
	})
	t.Run("context package hash changes with selected memory", func(t *testing.T) {
		request, _, _, memory := cbBaseRequest(t)
		first := buildCB(t, request)
		memory.result.Candidates[0].Reference.MemoryID = "memory-2"
		second := buildCB(t, request)
		if first.Package.ContentHash == second.Package.ContentHash {
			t.Fatal("selected memory did not affect context identity")
		}
	})
	t.Run("retrieval is read-only", func(t *testing.T) {
		request, evidence, counter, memory := cbBaseRequest(t)
		before := request
		_ = buildCB(t, request)
		if evidence.calls != 1 || counter.calls != 1 || memory.calls != 1 || !reflect.DeepEqual(request.Objective, before.Objective) || !reflect.DeepEqual(request.TaskState, before.TaskState) || !reflect.DeepEqual(request.Budget, before.Budget) {
			t.Fatal("builder mutated caller-owned task inputs or retrieval state")
		}
	})
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func recalculateCBBudget(budget *harnesscontracts.ContextBudget) {
	budget.Total = budget.SystemPolicy + budget.Objective + budget.TaskState + budget.Evidence + budget.CounterEvidence + budget.Memory + budget.ToolOutputs + budget.WorkingAllowance + budget.StructuredOutputAllowance
}

func recordHasTemporalStatus(report BuildReport, evidenceID, status string) bool {
	for _, record := range report.RetrievalRecords {
		if record.EvidenceReference != nil && record.EvidenceReference.EvidenceID == evidenceID && record.TemporalStatus == status {
			return true
		}
	}
	return false
}

// These helpers intentionally inspect only the report's structured references;
// they do not introduce a second report schema.
func (report BuildReport) PackageSelectedIDs() []string {
	ids := []string{}
	for _, record := range report.RetrievalRecords {
		if record.Decision == RetrievalSelected && record.EvidenceReference != nil && record.ItemKind == EvidenceItemKind {
			ids = append(ids, record.EvidenceReference.EvidenceID)
		}
	}
	return ids
}

func (report BuildReport) OmissionReferences() []string {
	ids := []string{}
	for _, omission := range report.Omissions {
		ids = append(ids, omission.CanonicalReference)
	}
	return ids
}

func (report BuildReport) DeferredReferencesIDs() []string {
	ids := []string{}
	for _, deferred := range report.DeferredReferences {
		ids = append(ids, deferred.CanonicalReference)
	}
	return ids
}

func (report BuildReport) AuditLikeOmissionCodes() []string {
	codes := []string{}
	for _, omission := range report.Omissions {
		codes = append(codes, omission.ReasonCode)
	}
	return codes
}
