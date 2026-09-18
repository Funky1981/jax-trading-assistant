package harnesscontracts

import (
	"strings"
	"testing"
	"time"
)

var contractTime = time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
var contractHash = "sha256:" + strings.Repeat("1", 64)
var contractSHA = strings.Repeat("a", 40)

func policyFixture() PolicyReference {
	return PolicyReference{Name: "research-policy", Version: "v1"}
}

func objectiveFixture() ResearchObjective {
	return ResearchObjective{
		ContractVersion:    ResearchObjectiveContractV1,
		ObjectiveID:        "objective-1",
		Version:            "v1",
		Purpose:            "Assess a research question with bounded evidence.",
		Question:           "What is the current answer to the stated question?",
		CreatedAt:          contractTime,
		Constraints:        []string{"paper-only", "no-execution"},
		RequiredOutputs:    []string{"answer", "uncertainty"},
		CompletionCriteria: []CompletionCriterion{{ID: "criterion-1", Description: "Answer and uncertainty are recorded."}},
		PolicyVersions:     []PolicyReference{policyFixture()},
		Status:             ObjectiveStatusActive,
	}
}

func evidenceFixture(classification string) EvidenceReference {
	return EvidenceReference{
		ContractVersion:  EvidenceReferenceContractV1,
		EvidenceID:       "evidence-1",
		CanonicalStore:   "jax-evidence",
		ProviderIdentity: "provider-1",
		SourceReference:  "source-1",
		ContentHash:      contractHash,
		Classification:   classification,
		QualityReference: "quality-1",
		PolicyReference:  policyFixture(),
		InformationState: "observed",
	}
}

func memoryFixture() MemoryReference {
	return MemoryReference{
		ContractVersion:         MemoryReferenceContractV1,
		MemoryID:                "memory-1",
		MemoryType:              "experience",
		SourceTaskReference:     "task-1",
		SourceDecisionReference: "decision-1",
		SourceOutcomeReference:  "outcome-1",
		CreatedAt:               contractTime,
		RegimeReference:         "regime-1",
		ReliabilityReference:    "reliability-1",
		CalibrationReference:    "calibration-1",
		ContentHash:             contractHash,
		Version:                 "v1",
	}
}

func toolFixture() ToolReference {
	completed := contractTime.Add(time.Minute)
	return ToolReference{
		ContractVersion:  ToolReferenceContractV1,
		ToolInvocationID: "tool-invocation-1",
		ToolID:           "source-reader",
		ToolVersion:      "v1",
		Purpose:          "Read a canonical source reference.",
		InputHash:        contractHash,
		InputReference:   "input-1",
		OutputHash:       contractHash,
		OutputReference:  "output-1",
		StartedAt:        contractTime,
		CompletedAt:      &completed,
		Status:           ToolStatusSucceeded,
		Redacted:         false,
	}
}

func taskStateFixture() TaskState {
	return TaskState{
		ContractVersion:    TaskStateContractV1,
		TaskID:             "task-1",
		ObjectiveID:        "objective-1",
		ObjectiveVersion:   "v1",
		StateVersion:       1,
		CurrentStage:       "research",
		PendingWork:        []WorkItem{{ID: "work-1", Description: "Review selected evidence."}},
		OpenQuestions:      []string{"Is the evidence sufficient?"},
		KnownUnknowns:      []string{"A missing source may change the conclusion."},
		Constraints:        []string{"No execution."},
		Decisions:          []DecisionRecord{{DecisionID: "decision-1", Summary: "Retain counter-evidence.", Reason: "It can invalidate the thesis.", DecidedAt: contractTime}},
		EvidenceReferences: []EvidenceReference{evidenceFixture(EvidenceSupporting)},
		CounterEvidenceReferences: []EvidenceReference{{
			ContractVersion:  EvidenceReferenceContractV1,
			EvidenceID:       "counter-1",
			CanonicalStore:   "jax-evidence",
			ProviderIdentity: "provider-1",
			SourceReference:  "source-counter-1",
			ContentHash:      contractHash,
			Classification:   EvidenceContradictory,
			QualityReference: "quality-1",
			PolicyReference:  policyFixture(),
			InformationState: "observed",
		}},
		MemoryReferences: []MemoryReference{memoryFixture()},
		NextAction:       "Review selected evidence.",
		CreatedAt:        contractTime,
		UpdatedAt:        contractTime.Add(time.Minute),
		Status:           TaskStatusRunning,
	}
}

func budgetFixture() ContextBudget {
	budget := ContextBudget{
		ContractVersion:           ContextBudgetContractV1,
		SystemPolicy:              100,
		Objective:                 100,
		TaskState:                 100,
		Evidence:                  300,
		CounterEvidence:           200,
		CounterEvidenceReserve:    100,
		Memory:                    100,
		ToolOutputs:               100,
		WorkingAllowance:          100,
		StructuredOutputAllowance: 100,
		RequireCounterEvidence:    true,
	}
	budget.Total = budget.SystemPolicy + budget.Objective + budget.TaskState + budget.Evidence + budget.CounterEvidence + budget.Memory + budget.ToolOutputs + budget.WorkingAllowance + budget.StructuredOutputAllowance
	return budget
}

func provenanceFixture() ContextProvenance {
	return ContextProvenance{
		ContractVersion:          ContextProvenanceContractV1,
		Objective:                ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"},
		TaskState:                TaskStateReference{TaskID: "task-1", StateVersion: 1},
		ContextPackageID:         "context-1",
		SelectedEvidenceIDs:      []string{"evidence-1"},
		CounterEvidenceIDs:       []string{"counter-1"},
		SelectedMemoryIDs:        []string{"memory-1"},
		ToolInvocationIDs:        []string{"tool-invocation-1"},
		Model:                    ModelReference{Provider: "model-provider", Model: "model", Version: "v1"},
		PromptVersion:            "prompt-v1",
		PolicyVersions:           []PolicyReference{policyFixture()},
		AssemblyAlgorithmVersion: "assembly-v1",
		AssembledAt:              contractTime,
	}
}

func contextPackageFixture(t *testing.T) ContextPackage {
	t.Helper()
	pkg, err := NewContextPackage(ContextPackage{
		ContextPackageID: "context-1",
		TaskID:           "task-1",
		TaskStateVersion: 1,
		ObjectiveID:      "objective-1",
		ObjectiveVersion: "v1",
		CreatedAt:        contractTime,
		ModelPurpose:     "Answer the research objective.",
		TokenBudget:      budgetFixture(),
		Constraints:      []string{"no-execution", "paper-only"},
		PolicyVersions:   []PolicyReference{policyFixture()},
		SelectedEvidence: []EvidenceReference{evidenceFixture(EvidenceSupporting)},
		CounterEvidence:  []EvidenceReference{evidenceFixture(EvidenceContradictory)},
		SelectedMemory:   []MemoryReference{memoryFixture()},
		CurrentState:     taskStateFixture(),
		OpenQuestions:    []string{"Is the evidence sufficient?"},
		KnownUnknowns:    []string{"A missing source may change the conclusion."},
		Assumptions:      []string{"The canonical source is available."},
		ToolReferences:   []ToolReference{toolFixture()},
		Provenance:       provenanceFixture(),
		Audit: ContextPackageAudit{
			BuilderVersion:     "builder-v1",
			OmittedEvidenceIDs: []string{"evidence-omitted"},
			SelectionReasons:   []string{"Selected for direct relevance."},
			AssembledAt:        contractTime,
		},
	})
	if err != nil {
		t.Fatalf("context fixture: %v", err)
	}
	return pkg
}

func expectInvalid(t *testing.T, name string, validate func() error) {
	t.Helper()
	if err := validate(); err == nil {
		t.Fatalf("%s: expected validation error", name)
	}
}

func TestResearchObjectiveContracts(t *testing.T) {
	if err := objectiveFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*ResearchObjective)
	}{
		{"empty identity", func(v *ResearchObjective) { v.ObjectiveID = "" }},
		{"empty objective", func(v *ResearchObjective) { v.Question = "" }},
		{"invalid version", func(v *ResearchObjective) { v.Version = "1" }},
		{"invalid required output", func(v *ResearchObjective) { v.RequiredOutputs = []string{""} }},
		{"missing completion criteria", func(v *ResearchObjective) { v.CompletionCriteria = nil }},
		{"duplicate completion criteria", func(v *ResearchObjective) {
			v.CompletionCriteria = append(v.CompletionCriteria, v.CompletionCriteria[0])
		}},
		{"malformed policy reference", func(v *ResearchObjective) { v.PolicyVersions[0].Version = "latest" }},
		{"unsupported status", func(v *ResearchObjective) { v.Status = "RUNNING" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := objectiveFixture()
			tc.edit(&value)
			expectInvalid(t, tc.name, value.Validate)
		})
	}
}

func TestTaskStateIsFreshSessionSufficientAndVersioned(t *testing.T) {
	state := taskStateFixture()
	if err := state.Validate(); err != nil {
		t.Logf("contract=%q task=%q objective=%q version=%q stage=%q created=%v updated=%v status=%q next=%q failure=%+v", state.ContractVersion, state.TaskID, state.ObjectiveID, state.ObjectiveVersion, state.CurrentStage, state.CreatedAt, state.UpdatedAt, state.Status, state.NextAction, state.Failure)
		t.Fatal(err)
	}
	if state.TaskID == "" || state.ObjectiveID == "" || state.NextAction == "" || len(state.PendingWork) == 0 || len(state.EvidenceReferences) == 0 {
		t.Fatal("fixture does not contain fresh-session continuation state")
	}
	expectInvalid(t, "zero state version", func() error { state.StateVersion = 0; return state.Validate() })
	expectInvalid(t, "duplicate work item", func() error {
		state = taskStateFixture()
		state.PendingWork = append(state.PendingWork, state.PendingWork[0])
		return state.Validate()
	})
	expectInvalid(t, "completed with pending work", func() error { state = taskStateFixture(); state.Status = TaskStatusCompleted; return state.Validate() })
	expectInvalid(t, "blocked without failure reason", func() error { state = taskStateFixture(); state.Status = TaskStatusBlocked; return state.Validate() })
	expectInvalid(t, "failed without failure reason", func() error { state = taskStateFixture(); state.Status = TaskStatusFailed; return state.Validate() })
	expectInvalid(t, "nonterminal without next action", func() error { state = taskStateFixture(); state.NextAction = ""; return state.Validate() })
	expectInvalid(t, "updated before created", func() error {
		state = taskStateFixture()
		state.UpdatedAt = state.CreatedAt.Add(-time.Minute)
		return state.Validate()
	})
	state.Status = TaskStatusBlocked
	state = taskStateFixture()
	state.Status = TaskStatusBlocked
	state.Failure = FailureState{Attempt: 1, Retryable: true, Classification: "tool_failure", Reason: "Source temporarily unavailable."}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestContextPackageCanonicalHashAndBoundary(t *testing.T) {
	pkg := contextPackageFixture(t)
	if err := pkg.Validate(); err != nil {
		t.Fatal(err)
	}
	initialHash, _ := pkg.ComputeContentHash()
	if initialHash != pkg.ContentHash {
		t.Fatalf("fixture hash is not stable: got=%s want=%s", initialHash, pkg.ContentHash)
	}
	reordered := pkg
	reordered.Constraints = []string{"paper-only", "no-execution"}
	reordered.PolicyVersions = []PolicyReference{policyFixture()}
	reordered.SelectedEvidence = append([]EvidenceReference(nil), pkg.SelectedEvidence...)
	reordered.CounterEvidence = append([]EvidenceReference(nil), pkg.CounterEvidence...)
	hash, err := reordered.ComputeContentHash()
	if err != nil || hash != pkg.ContentHash {
		t.Fatalf("reordering non-semantic collections changed hash: %v %s %s", err, hash, pkg.ContentHash)
	}
	expectInvalid(t, "changed evidence hash", func() error {
		changed := pkg
		changed.SelectedEvidence = append([]EvidenceReference(nil), pkg.SelectedEvidence...)
		changed.SelectedEvidence[0].SourceReference = "different-source"
		return changed.Validate()
	})
	expectInvalid(t, "changed objective constraint hash", func() error {
		changed := pkg
		changed.Constraints = append(changed.Constraints, "new-constraint")
		return changed.Validate()
	})
	auditChanged := pkg
	auditChanged.Audit.AssembledAt = contractTime.Add(24 * time.Hour)
	if hash, err := auditChanged.ComputeContentHash(); err != nil || hash != pkg.ContentHash {
		t.Fatalf("audit-only timestamp changed content hash: err=%v got=%s want=%s", err, hash, pkg.ContentHash)
	}
	modelVisibleChanged := pkg
	modelVisibleChanged.ModelPurpose = "A different purpose."
	if hash, err := modelVisibleChanged.ComputeContentHash(); err != nil || hash == pkg.ContentHash {
		t.Fatal("model-visible content did not change hash")
	}
	badHash := pkg
	badHash.ContentHash = contractHash
	expectInvalid(t, "hash mismatch", badHash.Validate)
	if !IsCanonicalHash(pkg.ContentHash) {
		t.Fatal("valid context hash was not recognized")
	}
}

func TestContextBudgetInvariants(t *testing.T) {
	if err := budgetFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(*ContextBudget)
	}{
		{"negative", func(v *ContextBudget) { v.Evidence = -1 }},
		{"inconsistent total", func(v *ContextBudget) { v.Total++ }},
		{"reserve exceeds class", func(v *ContextBudget) { v.CounterEvidenceReserve = v.CounterEvidence + 1 }},
		{"required reserve missing", func(v *ContextBudget) { v.CounterEvidenceReserve = 0 }},
		{"required class missing", func(v *ContextBudget) { v.CounterEvidence = 0; v.CounterEvidenceReserve = 0 }},
		{"integer overflow", func(v *ContextBudget) { v.SystemPolicy = int64(^uint64(0) >> 1); v.Total = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := budgetFixture()
			tc.edit(&value)
			expectInvalid(t, tc.name, value.Validate)
		})
	}
}

func TestEvidenceAndMemoryBoundaries(t *testing.T) {
	if err := evidenceFixture(EvidenceSupporting).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := evidenceFixture(EvidenceContradictory).ValidateCounterEvidence(); err != nil {
		t.Fatal(err)
	}
	expectInvalid(t, "supporting evidence cannot be counter-evidence", func() error { return evidenceFixture(EvidenceSupporting).ValidateCounterEvidence() })
	badEvidence := evidenceFixture(EvidenceSupporting)
	badEvidence.Classification = "made-up"
	expectInvalid(t, "unknown evidence classification", badEvidence.Validate)
	badEvidence = evidenceFixture(EvidenceSupporting)
	badEvidence.FirstSeenAt = func() *time.Time { v := contractTime.In(time.FixedZone("offset", 3600)); return &v }()
	expectInvalid(t, "non-UTC evidence time", badEvidence.Validate)
	if err := memoryFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	badMemory := memoryFixture()
	badMemory.ContentHash = "not-a-hash"
	expectInvalid(t, "invalid memory hash", badMemory.Validate)
	badMemory = memoryFixture()
	badMemory.MemoryType = ""
	expectInvalid(t, "missing memory type", badMemory.Validate)
}

func TestCheckpointToolOutputEvaluatorAndProvenanceContracts(t *testing.T) {
	provenance := provenanceFixture()
	if err := provenance.Validate(); err != nil {
		t.Fatal(err)
	}
	checkpoint := Checkpoint{
		ContractVersion: CheckpointContractV1, CheckpointID: "checkpoint-1",
		Objective:    ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"},
		TaskState:    TaskStateReference{TaskID: "task-1", StateVersion: 1},
		CurrentStage: "research", OpenQuestions: []string{"Open question."}, KnownUnknowns: []string{"Unknown."},
		NextAction: "Continue research.", Failure: FailureState{}, Provenance: provenance, ContentHash: contractHash, CreatedAt: contractTime,
	}
	if err := checkpoint.Validate(); err != nil {
		t.Fatal(err)
	}
	badCheckpoint := checkpoint
	badCheckpoint.NextAction = ""
	expectInvalid(t, "checkpoint missing next action", badCheckpoint.Validate)
	if err := toolFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	failedTool := toolFixture()
	failedTool.Status = ToolStatusFailed
	failedTool.OutputHash = ""
	failedTool.OutputReference = ""
	failedTool.FailureClassification = "timeout"
	if err := failedTool.Validate(); err != nil {
		t.Fatal(err)
	}
	secretTool := toolFixture()
	secretTool.InputReference = "api_key=secret"
	expectInvalid(t, "secret-bearing tool reference", secretTool.Validate)
	checks := EvaluatorChecks{ObjectiveSatisfied: true, EvidenceSufficient: true, EvidenceQualityAcceptable: true, ContradictionsAddressed: true, CausalChainSupported: true, ConfidenceSupported: true, UncertaintyRepresented: true, InvalidationDefined: true, PolicyCompliant: true}
	pass := EvaluatorResult{ContractVersion: EvaluatorResultContractV1, ResultID: "evaluation-1", TaskID: "task-1", ContextPackageID: "context-1", Decision: EvaluatorPass, Checks: checks, EvaluatorVersion: "evaluator-v1", EvaluatedAt: contractTime, ExecutionAuthority: "NONE"}
	if err := pass.Validate(); err != nil {
		t.Fatal(err)
	}
	revise := pass
	revise.Decision = EvaluatorRevise
	revise.Reasons = []EvaluatorReason{{Code: "missing-evidence", Detail: "A required source is missing.", Correction: "Retrieve the source."}}
	if err := revise.Validate(); err != nil {
		t.Fatal(err)
	}
	abstain := revise
	abstain.Decision = EvaluatorAbstain
	if err := abstain.Validate(); err != nil {
		t.Fatal(err)
	}
	badPass := pass
	badPass.Checks.UnsupportedClaims = []string{"unsupported claim"}
	expectInvalid(t, "PASS with unsupported claim", badPass.Validate)
	badPass = pass
	badPass.ExecutionAuthority = "PAPER"
	expectInvalid(t, "PASS cannot authorize execution", badPass.Validate)
	badRevise := pass
	badRevise.Decision = EvaluatorRevise
	expectInvalid(t, "REVISE requires reason", badRevise.Validate)
}

func TestEvaluatorInputAndCompactionRecordContracts(t *testing.T) {
	pkg := contextPackageFixture(t)
	input := EvaluatorInput{
		ContractVersion:    EvaluatorInputContractV1,
		InputID:            "evaluator-input-1",
		Objective:          ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"},
		TaskState:          TaskStateReference{TaskID: "task-1", StateVersion: 1},
		ContextPackage:     ArtifactReference{ReferenceID: "context-1", Contract: ContextPackageContractV1, ContentHash: pkg.ContentHash},
		StructuredOutput:   ArtifactReference{ReferenceID: "output-1", Contract: StructuredResearchOutputContractV1, ContentHash: contractHash},
		ContextProvenance:  pkg.Provenance,
		CreatedAt:          contractTime,
		ExecutionAuthority: "NONE",
	}
	if err := input.Validate(); err != nil {
		t.Fatal(err)
	}
	badInput := input
	badInput.ExecutionAuthority = "PAPER"
	expectInvalid(t, "evaluator input cannot authorize execution", badInput.Validate)
	badInput = input
	badInput.Objective.ObjectiveID = "other-objective"
	expectInvalid(t, "evaluator input references must agree", badInput.Validate)
	provenance := provenanceFixture()
	provenance.TaskState = TaskStateReference{TaskID: "task-1", StateVersion: 2}
	compaction := CompactionRecord{
		ContractVersion: CompactionRecordContractV1, CompactionID: "compaction-1",
		TaskStateBefore: TaskStateReference{TaskID: "task-1", StateVersion: 1}, TaskStateAfter: provenance.TaskState,
		RetainedEvidenceIDs: []string{"evidence-1"}, RetainedContradictionIDs: []string{"counter-1"}, RetainedOpenQuestions: []string{"Open question."}, RetainedKnownUnknowns: []string{"Unknown."}, RetainedDecisionIDs: []string{"decision-1"}, NextAction: "Continue research.", Provenance: provenance, AlgorithmVersion: "compaction-v1", CreatedAt: contractTime, ContentHash: contractHash,
	}
	if err := compaction.Validate(); err != nil {
		t.Fatal(err)
	}
	badCompaction := compaction
	badCompaction.CanonicalStorage = true
	expectInvalid(t, "compaction cannot become canonical storage", badCompaction.Validate)
	badCompaction = compaction
	badCompaction.TaskStateAfter = TaskStateReference{TaskID: "other-task", StateVersion: 2}
	expectInvalid(t, "compaction task identity", badCompaction.Validate)
}

func TestHarnessRunCorrelationAndIsolation(t *testing.T) {
	started := contractTime
	ended := contractTime.Add(time.Minute)
	run := HarnessRun{ContractVersion: HarnessRunContractV1, HarnessRunID: "run-1", TaskID: "task-1", Objective: ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"}, ContextPackage: ArtifactReference{ReferenceID: "context-1", Contract: ContextPackageContractV1, ContentHash: contractHash}, Model: ModelReference{Provider: "provider", Model: "model", Version: "v1"}, StartedAt: started, EndedAt: &ended, ToolReferences: []ToolReference{toolFixture()}, Status: HarnessRunSucceeded, TokenUsage: TokenUsage{InputTokens: 10, OutputTokens: 5}, LatencyMillis: 100, ExecutionAuthority: "NONE"}
	if err := run.Validate(); err != nil {
		t.Fatal(err)
	}
	badRun := run
	badRun.ExecutionAuthority = "BROKER"
	expectInvalid(t, "execution authority", badRun.Validate)
	badRun = run
	badRun.EndedAt = func() *time.Time { v := started.Add(-time.Minute); return &v }()
	expectInvalid(t, "end before start", badRun.Validate)
	badRun = run
	badRun.Status = HarnessRunFailed
	expectInvalid(t, "failed without failure state", badRun.Validate)
	if _, err := CanonicalHash(map[string]string{"b": "2", "a": "1"}); err != nil {
		t.Fatal(err)
	}
}

func repositoryFixture() RepositoryState {
	return RepositoryState{ContractVersion: RepositoryStateContractV1, RepositoryHeadSHA: contractSHA, StartingSHA: contractSHA, FrozenExperimentalCodeSHA: strings.Repeat("b", 40), Branch: "capability-reset", WorktreeStatus: RepositoryWorktreeClean, CurrentPackageReference: "HARNESS-01", CapturedAt: contractTime}
}

func engineeringObjectiveFixture() EngineeringObjective {
	return EngineeringObjective{ContractVersion: EngineeringObjectiveContractV1, ObjectiveID: "engineering-1", Version: "v1", Purpose: "Implement foundation contracts.", Request: "Create deterministic contracts and tests.", Scope: []string{"internal/modules/harnesscontracts"}, Exclusions: []string{"cmd/trader", "PAPER-02"}, AcceptanceCriteria: []string{"All contracts validate."}, CreatedAt: contractTime, Status: ObjectiveStatusActive}
}

func documentationMapFixture() DocumentationMap {
	return DocumentationMap{ContractVersion: DocumentationMapContractV1, AuthorityChain: []DocumentReference{{Path: "Docs/HARNESS/ARCHITECTURE.md", Role: "architecture"}}, SelectedDocuments: []DocumentReference{{Path: "Docs/HARNESS/ROADMAP.md", Role: "roadmap"}}, CurrentPackageReference: "HARNESS-01", GeneratedAt: contractTime}
}

func TestEngineeringHarnessContracts(t *testing.T) {
	if err := engineeringObjectiveFixture().Validate(); err != nil {
		t.Fatal(err)
	}
	repository := repositoryFixture()
	if err := repository.Validate(); err != nil {
		t.Fatal(err)
	}
	if repository.RepositoryHeadSHA == repository.FrozenExperimentalCodeSHA {
		t.Fatal("fixture must distinguish repository HEAD from frozen experimental SHA")
	}
	badRepository := repository
	badRepository.RepositoryHeadSHA = "not-a-sha"
	expectInvalid(t, "invalid repository SHA", badRepository.Validate)
	badRepository = repository
	badRepository.WorktreeStatus = "UNKNOWN_VALUE"
	expectInvalid(t, "invalid worktree status", badRepository.Validate)
	docs := documentationMapFixture()
	if err := docs.Validate(); err != nil {
		for _, ref := range append(append([]DocumentReference{}, docs.AuthorityChain...), docs.SelectedDocuments...) {
			t.Logf("document %q role %q valid=%v", ref.Path, ref.Role, ref.Validate())
		}
		t.Fatal(err)
	}
	badDocs := docs
	badDocs.SelectedDocuments = append([]DocumentReference(nil), docs.SelectedDocuments...)
	badDocs.SelectedDocuments[0].Path = "../../secret"
	expectInvalid(t, "document traversal", badDocs.Validate)
	plan := DefaultVerificationRegistry()
	if err := plan.Validate(); err != nil {
		for _, command := range plan.Commands {
			t.Logf("verification %q valid=%v", command.VerificationID, command.Validate())
		}
		t.Fatal(err)
	}
	badPlan := plan
	badPlan.Commands = append([]VerificationCommand(nil), plan.Commands...)
	badPlan.Commands[0].TrustedSource = "untrusted-input"
	expectInvalid(t, "untrusted verification command", badPlan.Validate)
	checkpoint := EngineeringCheckpoint{ContractVersion: EngineeringCheckpointContractV1, CheckpointID: "engineering-checkpoint-1", Objective: engineeringObjectiveFixture(), Repository: repository, Documentation: docs, Verification: plan, CompletedWork: []string{"Contracts drafted."}, PendingWork: []string{"External review."}, KnownFailures: []string{"Known unrelated test may fail."}, NextAction: "Run the external review.", CheckpointVersion: 1, CreatedAt: contractTime, UpdatedAt: contractTime.Add(time.Minute)}
	if err := checkpoint.Validate(); err != nil {
		t.Logf("objective=%v repository=%v docs=%v plan=%v completed=%v pending=%v failures=%v next=%q version=%d times=%v/%v", checkpoint.Objective.Validate(), checkpoint.Repository.Validate(), checkpoint.Documentation.Validate(), checkpoint.Verification.Validate(), validStringList(checkpoint.CompletedWork, 256, 4096), validStringList(checkpoint.PendingWork, 256, 4096), validStringList(checkpoint.KnownFailures, 128, 8192), checkpoint.NextAction, checkpoint.CheckpointVersion, checkpoint.CreatedAt, checkpoint.UpdatedAt)
		t.Fatal(err)
	}
	if err := validCleanExitReport().Validate(); err != nil {
		t.Fatal(err)
	}
	incomplete := validCleanExitReport()
	incomplete.ExactSHACI = nil
	expectInvalid(t, "clean exit missing exact SHA CI", incomplete.Validate)
	incomplete = validCleanExitReport()
	incomplete.HandoverComplete = false
	expectInvalid(t, "clean exit incomplete handover", incomplete.Validate)
	incomplete = validCleanExitReport()
	incomplete.DivergenceAhead = 1
	expectInvalid(t, "clean exit divergence", incomplete.Validate)
	blocked := validCleanExitReport()
	blocked.Result = CleanExitBlocked
	blocked.OpenBlockers = []string{"External review pending."}
	if err := blocked.Validate(); err != nil {
		t.Fatal(err)
	}
	ended := contractTime.Add(time.Minute)
	engineeringRun := EngineeringHarnessRun{ContractVersion: EngineeringHarnessRunContractV1, RunID: "engineering-run-1", ObjectiveID: "engineering-1", Repository: repository, CheckpointID: "engineering-checkpoint-1", VerificationPlanID: plan.PlanID, StartedAt: contractTime, EndedAt: &ended, Status: HarnessRunSucceeded}
	if err := engineeringRun.Validate(); err != nil {
		t.Fatal(err)
	}
	badRun := engineeringRun
	badRun.Status = HarnessRunFailed
	expectInvalid(t, "engineering failed without failure state", badRun.Validate)
}

func validCleanExitReport() CleanExitReport {
	return CleanExitReport{
		ContractVersion: CleanExitReportContractV1, ReportID: "clean-exit-1", StartingSHA: contractSHA, EndingSHA: contractSHA, Branch: "capability-reset",
		ExpectedDiffScope: []string{"internal/modules/harnesscontracts", "Docs/HARNESS"}, ActualDiffScope: []string{"internal/modules/harnesscontracts", "Docs/HARNESS"}, RuntimeIsolationStatus: VerificationPass,
		Checks: []CheckRecord{{CheckID: "go-test-all", Status: VerificationPass, EvidenceReference: "evidence-go-test"}}, Commit: contractSHA, PushStatus: "PUSHED", OriginEquality: true, ExactSHACI: []CIResult{
			{Workflow: "CI", RunID: "100", SHA: contractSHA, Result: VerificationPass},
			{Workflow: "Golden Tests", RunID: "101", SHA: contractSHA, Result: VerificationPass},
			{Workflow: "Import Boundary Enforcement", RunID: "102", SHA: contractSHA, Result: VerificationPass},
		}, HandoverComplete: true, Result: CleanExitPass,
	}
}

func TestSecretsAreRejectedFromReferenceFields(t *testing.T) {
	for _, value := range []string{"api_key=secret", "password=secret", "authorization=bearer", "token=secret", "database_url=secret"} {
		if validReference(value) {
			t.Fatalf("secret marker accepted: %s", value)
		}
	}
	if validReference("ordinary-reference") == false {
		t.Fatal("ordinary reference rejected")
	}
}

func TestAdditionalContractEdgeCases(t *testing.T) {
	cases := []struct {
		name  string
		check func() error
	}{
		{"counter evidence cannot be candidate", func() error {
			state := taskStateFixture()
			state.CounterEvidenceReferences[0].Classification = EvidenceCandidate
			return state.Validate()
		}},
		{"task checkpoint must belong to task", func() error {
			state := taskStateFixture()
			state.Checkpoint = &CheckpointReference{ContractVersion: CheckpointContractV1, CheckpointID: "checkpoint-1", TaskID: "other-task", StateVersion: 1, ContentHash: contractHash, CreatedAt: contractTime}
			return state.Validate()
		}},
		{"task failure classification requires reason", func() error {
			state := taskStateFixture()
			state.Status = TaskStatusBlocked
			state.Failure = FailureState{Classification: "timeout"}
			return state.Validate()
		}},
		{"context provenance package identity", func() error {
			pkg := contextPackageFixture(t)
			pkg.Provenance.ContextPackageID = "other-context"
			return pkg.Validate()
		}},
		{"context provenance package hash", func() error {
			pkg := contextPackageFixture(t)
			pkg.Provenance.ContextPackageHash = contractHash
			return pkg.Validate()
		}},
		{"selected evidence cannot be contradictory", func() error {
			pkg := contextPackageFixture(t)
			pkg.SelectedEvidence[0].Classification = EvidenceContradictory
			return pkg.Validate()
		}},
		{"context tool reference must validate", func() error {
			pkg := contextPackageFixture(t)
			pkg.ToolReferences[0].InputReference = "token=secret"
			return pkg.Validate()
		}},
		{"context budget contract version", func() error {
			budget := budgetFixture()
			budget.ContractVersion = "jax.harness.context_budget/v2"
			return budget.Validate()
		}},
		{"evidence requires provider identity", func() error {
			evidence := evidenceFixture(EvidenceSupporting)
			evidence.ProviderIdentity = ""
			return evidence.Validate()
		}},
		{"evidence content hash format", func() error {
			evidence := evidenceFixture(EvidenceSupporting)
			evidence.ContentHash = "sha1:bad"
			return evidence.Validate()
		}},
		{"memory source outcome reference", func() error {
			memory := memoryFixture()
			memory.SourceOutcomeReference = ""
			return memory.Validate()
		}},
		{"memory contract cannot be evidence contract", func() error {
			memory := memoryFixture()
			memory.ContractVersion = EvidenceReferenceContractV1
			return memory.Validate()
		}},
		{"requested tool cannot be completed", func() error {
			tool := toolFixture()
			tool.Status = ToolStatusRequested
			return tool.Validate()
		}},
		{"successful tool needs output hash", func() error {
			tool := toolFixture()
			tool.OutputHash = ""
			return tool.Validate()
		}},
		{"failed tool needs classification", func() error {
			tool := toolFixture()
			tool.Status = ToolStatusFailed
			tool.OutputHash = ""
			tool.OutputReference = ""
			return tool.Validate()
		}},
		{"checkpoint objective version", func() error {
			checkpoint := Checkpoint{ContractVersion: CheckpointContractV1, CheckpointID: "checkpoint-1", Objective: ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "bad"}, TaskState: TaskStateReference{TaskID: "task-1", StateVersion: 1}, CurrentStage: "research", NextAction: "Continue.", Provenance: provenanceFixture(), ContentHash: contractHash, CreatedAt: contractTime}
			return checkpoint.Validate()
		}},
		{"provenance model version", func() error {
			provenance := provenanceFixture()
			provenance.Model.Version = ""
			return provenance.Validate()
		}},
		{"structured output confidence bounds", func() error {
			confidence := 2.0
			output := StructuredResearchOutput{ContractVersion: StructuredResearchOutputContractV1, OutputID: "output-1", TaskID: "task-1", ContextPackageID: "context-1", Answer: "Answer.", Uncertainty: "Some uncertainty.", RecommendedNextAction: "Review.", CompletionStatus: OutputStatusComplete, Confidence: &confidence}
			return output.Validate()
		}},
		{"structured output counter classification", func() error {
			output := StructuredResearchOutput{ContractVersion: StructuredResearchOutputContractV1, OutputID: "output-1", TaskID: "task-1", ContextPackageID: "context-1", Answer: "Answer.", Uncertainty: "Some uncertainty.", RecommendedNextAction: "Review.", CompletionStatus: OutputStatusComplete, CounterEvidenceReferences: []EvidenceReference{evidenceFixture(EvidenceSupporting)}}
			return output.Validate()
		}},
		{"evaluator pass cannot be premature", func() error {
			checks := EvaluatorChecks{ObjectiveSatisfied: true, EvidenceSufficient: true, EvidenceQualityAcceptable: true, ContradictionsAddressed: true, CausalChainSupported: true, ConfidenceSupported: true, UncertaintyRepresented: true, InvalidationDefined: true, PolicyCompliant: true, PrematureCompletionDetected: true}
			result := EvaluatorResult{ContractVersion: EvaluatorResultContractV1, ResultID: "evaluation-1", TaskID: "task-1", ContextPackageID: "context-1", Decision: EvaluatorPass, Checks: checks, EvaluatorVersion: "evaluator-v1", EvaluatedAt: contractTime, ExecutionAuthority: "NONE"}
			return result.Validate()
		}},
		{"abstain requires reason", func() error {
			result := EvaluatorResult{ContractVersion: EvaluatorResultContractV1, ResultID: "evaluation-1", TaskID: "task-1", ContextPackageID: "context-1", Decision: EvaluatorAbstain, EvaluatorVersion: "evaluator-v1", EvaluatedAt: contractTime, ExecutionAuthority: "NONE"}
			return result.Validate()
		}},
		{"harness run status", func() error {
			ended := contractTime.Add(time.Minute)
			run := HarnessRun{ContractVersion: HarnessRunContractV1, HarnessRunID: "run-1", TaskID: "task-1", Objective: ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"}, ContextPackage: ArtifactReference{ReferenceID: "context-1", Contract: ContextPackageContractV1, ContentHash: contractHash}, Model: ModelReference{Provider: "provider", Model: "model", Version: "v1"}, StartedAt: contractTime, EndedAt: &ended, Status: "INVALID", ExecutionAuthority: "NONE"}
			return run.Validate()
		}},
		{"harness run token usage", func() error {
			ended := contractTime.Add(time.Minute)
			run := HarnessRun{ContractVersion: HarnessRunContractV1, HarnessRunID: "run-1", TaskID: "task-1", Objective: ObjectiveReference{ObjectiveID: "objective-1", ObjectiveVersion: "v1"}, ContextPackage: ArtifactReference{ReferenceID: "context-1", Contract: ContextPackageContractV1, ContentHash: contractHash}, Model: ModelReference{Provider: "provider", Model: "model", Version: "v1"}, StartedAt: contractTime, EndedAt: &ended, Status: HarnessRunSucceeded, TokenUsage: TokenUsage{InputTokens: -1}, ExecutionAuthority: "NONE"}
			return run.Validate()
		}},
		{"engineering objective acceptance criteria", func() error {
			objective := engineeringObjectiveFixture()
			objective.AcceptanceCriteria = nil
			return objective.Validate()
		}},
		{"documentation map needs authority", func() error {
			mapping := documentationMapFixture()
			mapping.AuthorityChain = nil
			return mapping.Validate()
		}},
		{"verification command rejects newline", func() error {
			command := DefaultVerificationRegistry().Commands[0]
			command.Command = "git diff --check\nWrite-Anything"
			return command.Validate()
		}},
		{"CI evidence must match ending SHA", func() error {
			report := validCleanExitReport()
			report.ExactSHACI[0].SHA = strings.Repeat("c", 40)
			return report.Validate()
		}},
		{"clean exit cannot contain failed check", func() error {
			report := validCleanExitReport()
			report.Checks[0].Status = VerificationFail
			return report.Validate()
		}},
		{"engineering run end ordering", func() error {
			ended := contractTime.Add(-time.Minute)
			run := EngineeringHarnessRun{ContractVersion: EngineeringHarnessRunContractV1, RunID: "engineering-run-1", ObjectiveID: "engineering-1", Repository: repositoryFixture(), CheckpointID: "checkpoint-1", VerificationPlanID: "plan-1", StartedAt: contractTime, EndedAt: &ended, Status: HarnessRunRunning}
			return run.Validate()
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { expectInvalid(t, tc.name, tc.check) })
	}
}
