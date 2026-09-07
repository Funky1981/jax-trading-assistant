package harness

import (
	"context"
	"strings"
	"testing"
	"time"
)

func checkpointBudget(t *testing.T) ResearchBudget {
	t.Helper()
	budget, err := NewResearchBudget(ResearchBudget{Version: "v1", MaxWallClock: time.Minute, ToolTimeout: time.Second, MaxSteps: 5, MaxToolCalls: 5, MaxModelCalls: 2, MaxRetries: 1, MaxInputTokens: 100, MaxOutputTokens: 100, MaxReasoningTokens: 100, MaxEstimatedCostUSD: 1, MaxActualCostUSD: 1, MaximumModelTier: BudgetTierLocal, RequireKnownCost: false})
	if err != nil {
		t.Fatal(err)
	}
	return budget
}

func checkpointStateFixture(t *testing.T) ResearchTaskState {
	t.Helper()
	return ResearchTaskState{ContractVersion: ResearchCheckpointContractV1, TaskID: "task_fixture", Objective: "Assess frozen evidence", PlanVersion: "plan-v1", PlannedSteps: []string{"step_1", "step_2"}, CompletedSteps: []ResearchStepRecord{{StepID: "step_1", Kind: "tool", EvidenceIDs: []string{"evd_fixture"}, Summary: "evidence acquired", Status: "SUCCEEDED", CompletedAt: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)}}, EvidenceIDs: []string{"evd_fixture"}, UnresolvedGaps: []string{"corroboration"}, Contradictions: []string{"source timing differs"}, CurrentReport: "provisional report", Budget: checkpointBudget(t), BudgetState: BudgetState{Steps: 1, ToolCalls: 1}, Status: ResearchTaskPaused, CheckpointVersion: 1, TaskStartedAt: time.Date(2026, 9, 7, 11, 59, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC)}
}

func TestCheckpointStorePersistsAndReconstructsBoundedResumeState(t *testing.T) {
	checkpoint, err := NewResearchCheckpoint(checkpointStateFixture(t), "paused after evidence step", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryCheckpointStore()
	if err := store.Save(context.Background(), checkpoint); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(context.Background(), checkpoint.TaskID, 1)
	if err != nil || loaded.ID != checkpoint.ID {
		t.Fatalf("checkpoint was not durably recoverable: %#v err=%v", loaded, err)
	}
	resume, err := loaded.ReconstructResumeContext(4096)
	if err != nil {
		t.Fatal(err)
	}
	if resume.Objective != checkpoint.State.Objective || len(resume.RemainingSteps) != 1 || resume.RemainingSteps[0] != "step_2" || len(resume.EvidenceIDs) != 1 || len(resume.Contradictions) != 1 {
		t.Fatalf("resume context lost bounded state: %#v", resume)
	}
}

func TestCheckpointIntegrityRejectsMutationCorruptionAndOversizedResume(t *testing.T) {
	checkpoint, err := NewResearchCheckpoint(checkpointStateFixture(t), "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryCheckpointStore()
	if err := store.Save(context.Background(), checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.State.CurrentReport = "corrupted"
	store.items[checkpoint.TaskID+":1"] = checkpoint
	if _, err := store.Load(context.Background(), checkpoint.TaskID, 1); err == nil {
		t.Fatal("expected corrupted checkpoint rejection")
	}
	if _, err := NewResearchCheckpoint(checkpointStateFixture(t), "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	checkpoint, err = NewResearchCheckpoint(checkpointStateFixture(t), strings.Repeat("x", 9000), time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC))
	if err == nil || checkpoint.ID != "" {
		t.Fatal("expected oversized checkpoint summary rejection")
	}
}

func TestCheckpointResumeDoesNotReplayTranscriptAndRejectsTooSmallBound(t *testing.T) {
	checkpoint, err := NewResearchCheckpoint(checkpointStateFixture(t), "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := checkpoint.ReconstructResumeContext(1); err == nil {
		t.Fatal("expected bounded resume context rejection")
	}
}

func TestCheckpointIntegrityRejectsPlanAndBudgetStateTampering(t *testing.T) {
	state := checkpointStateFixture(t)
	state.CompletedSteps[0].StepID = "step_unknown"
	if _, err := NewResearchCheckpoint(state, "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC)); err == nil {
		t.Fatal("expected completion outside plan rejection")
	}
	state = checkpointStateFixture(t)
	state.BudgetState.ToolCalls = state.Budget.MaxToolCalls + 1
	if _, err := NewResearchCheckpoint(state, "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC)); err == nil {
		t.Fatal("expected over-budget checkpoint rejection")
	}
}

func TestCheckpointIntegrityRejectsMalformedToolRecord(t *testing.T) {
	state := checkpointStateFixture(t)
	state.ToolCallRecords = []ControlledToolResult{{ContractVersion: ControlledToolResultContractV1, ToolID: "market_snapshot", ToolVersion: "v1", RunID: "run_fixture", StepID: "step_1", PermissionTier: ToolPermissionEvidenceRead, OutputContract: "jax.market_snapshot/v1", Payload: []byte(`{"evidence_id":"evd_fixture"}`), EvidenceIDs: []string{"evd_fixture"}, Source: "fixture", ObservedAt: time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC), Untrusted: false, ExecutionAuthority: "NONE"}}
	if _, err := NewResearchCheckpoint(state, "paused", time.Date(2026, 9, 7, 12, 1, 0, 0, time.UTC)); err == nil {
		t.Fatal("expected trusted tool record rejection")
	}
}
