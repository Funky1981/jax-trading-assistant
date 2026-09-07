package harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ResearchCheckpointContractV1 = "jax.research_checkpoint/v1"

const (
	ResearchTaskPaused          = "PAUSED"
	ResearchTaskRunning         = "RUNNING"
	ResearchTaskSucceeded       = "SUCCEEDED"
	ResearchTaskFailed          = "FAILED"
	ResearchTaskBlocked         = "BLOCKED"
	ResearchTaskBudgetExhausted = "BUDGET_EXHAUSTED"
	ResearchTaskCancelled       = "CANCELLED"
)

type ResearchStepRecord struct {
	StepID      string    `json:"step_id"`
	Kind        string    `json:"kind"`
	ToolID      string    `json:"tool_id,omitempty"`
	ToolVersion string    `json:"tool_version,omitempty"`
	EvidenceIDs []string  `json:"evidence_ids,omitempty"`
	Summary     string    `json:"summary"`
	Status      string    `json:"status"`
	CompletedAt time.Time `json:"completed_at"`
}

type ModelCallRecord struct {
	CallID         string `json:"call_id"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	PromptVersion  string `json:"prompt_version"`
	OutputContract string `json:"output_contract"`
	UsageRawSHA256 string `json:"usage_raw_sha256"`
	CostStatus     string `json:"cost_status"`
}

type ResearchTaskState struct {
	ContractVersion   string                 `json:"contract_version"`
	TaskID            string                 `json:"task_id"`
	Objective         string                 `json:"objective"`
	PlanVersion       string                 `json:"plan_version"`
	PlannedSteps      []string               `json:"planned_steps"`
	CompletedSteps    []ResearchStepRecord   `json:"completed_steps"`
	EvidenceIDs       []string               `json:"evidence_ids"`
	UnresolvedGaps    []string               `json:"unresolved_gaps"`
	Contradictions    []string               `json:"contradictions"`
	ReplanCount       int                    `json:"replan_count"`
	CurrentReport     string                 `json:"current_report"`
	ToolCallRecords   []ControlledToolResult `json:"tool_call_records"`
	ModelCallRecords  []ModelCallRecord      `json:"model_call_records"`
	Budget            ResearchBudget         `json:"budget"`
	BudgetState       BudgetState            `json:"budget_state"`
	Status            string                 `json:"status"`
	FailureReason     string                 `json:"failure_reason,omitempty"`
	CheckpointVersion int                    `json:"checkpoint_version"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

func (state ResearchTaskState) Validate() error {
	if state.ContractVersion != ResearchCheckpointContractV1 || !validControlledID(state.TaskID) || strings.TrimSpace(state.Objective) == "" || len(state.Objective) > 4096 || strings.TrimSpace(state.PlanVersion) == "" || !schemaStringList(state.PlannedSteps) || len(state.PlannedSteps) > 32 || state.ReplanCount < 0 || state.ReplanCount > 8 || len(state.CompletedSteps) > len(state.PlannedSteps) || !schemaStringList(state.EvidenceIDs) || len(state.EvidenceIDs) > 512 || !schemaStringList(state.UnresolvedGaps) || len(state.UnresolvedGaps) > 64 || !schemaStringList(state.Contradictions) || len(state.Contradictions) > 64 || len(state.CurrentReport) > 128*1024 || len(state.ToolCallRecords) > 128 || len(state.ModelCallRecords) > 64 || state.CheckpointVersion < 1 || state.UpdatedAt.IsZero() || state.UpdatedAt.Location() != time.UTC {
		return fmt.Errorf("research task state is incomplete or unbounded")
	}
	if err := state.Budget.Validate(); err != nil {
		return err
	}
	if err := validateCheckpointBudgetState(state.Budget, state.BudgetState); err != nil {
		return err
	}
	switch state.Status {
	case ResearchTaskPaused, ResearchTaskRunning, ResearchTaskSucceeded, ResearchTaskFailed, ResearchTaskBlocked, ResearchTaskBudgetExhausted, ResearchTaskCancelled:
	default:
		return fmt.Errorf("unsupported research task status %q", state.Status)
	}
	if (state.Status == ResearchTaskFailed || state.Status == ResearchTaskBlocked || state.Status == ResearchTaskBudgetExhausted || state.Status == ResearchTaskCancelled) != (strings.TrimSpace(state.FailureReason) != "") {
		return fmt.Errorf("terminal failure state and failure reason must agree")
	}
	planned := make(map[string]struct{}, len(state.PlannedSteps))
	for _, stepID := range state.PlannedSteps {
		if !validControlledID(stepID) {
			return fmt.Errorf("research plan contains invalid step identity")
		}
		if _, exists := planned[stepID]; exists {
			return fmt.Errorf("research plan contains duplicate step %q", stepID)
		}
		planned[stepID] = struct{}{}
	}
	completed := make(map[string]struct{}, len(state.CompletedSteps))
	for _, step := range state.CompletedSteps {
		if _, exists := planned[step.StepID]; !exists {
			return fmt.Errorf("completed research step is not in the plan")
		}
		if _, exists := completed[step.StepID]; exists {
			return fmt.Errorf("completed research step is duplicated")
		}
		completed[step.StepID] = struct{}{}
		if !validControlledID(step.StepID) || strings.TrimSpace(step.Kind) == "" || strings.TrimSpace(step.Summary) == "" || step.CompletedAt.IsZero() || step.CompletedAt.Location() != time.UTC || step.Status != "SUCCEEDED" || !schemaStringList(step.EvidenceIDs) && len(step.EvidenceIDs) > 0 {
			return fmt.Errorf("completed research step is invalid")
		}
	}
	for _, record := range state.ToolCallRecords {
		if record.ContractVersion != ControlledToolResultContractV1 || !validControlledToolID(record.ToolID) || strings.TrimSpace(record.ToolVersion) == "" || !validControlledID(record.RunID) || !validControlledID(record.StepID) || strings.TrimSpace(record.PermissionTier) == "" || strings.TrimSpace(record.OutputContract) == "" || len(record.Payload) == 0 || len(record.Payload) > 1024*1024 || !json.Valid(record.Payload) || !schemaStringList(record.EvidenceIDs) || len(record.EvidenceIDs) == 0 || strings.TrimSpace(record.Source) == "" || record.ObservedAt.IsZero() || record.ObservedAt.Location() != time.UTC || !record.Untrusted || record.ExecutionAuthority != "NONE" {
			return fmt.Errorf("checkpoint contains invalid or trusted tool record")
		}
		for _, evidenceID := range record.EvidenceIDs {
			if !validArgumentIdentifier(evidenceID) {
				return fmt.Errorf("checkpoint contains invalid tool evidence ID")
			}
		}
	}
	for _, record := range state.ModelCallRecords {
		if !validControlledID(record.CallID) || strings.TrimSpace(record.Provider) == "" || strings.TrimSpace(record.Model) == "" || strings.TrimSpace(record.PromptVersion) == "" || strings.TrimSpace(record.OutputContract) == "" || !validSHA256Usage(record.UsageRawSHA256) || record.CostStatus == "" {
			return fmt.Errorf("checkpoint contains invalid model provenance")
		}
	}
	return nil
}

func validateCheckpointBudgetState(budget ResearchBudget, state BudgetState) error {
	if state.Steps < 0 || state.ToolCalls < 0 || state.ModelCalls < 0 || state.Retries < 0 || state.EstimatedInput < 0 || state.EstimatedOutput < 0 || state.EstimatedReasoning < 0 || state.ActualInput < 0 || state.ActualOutput < 0 || state.ActualReasoning < 0 || !finiteBudgetNumber(state.EstimatedCostUSD) || state.EstimatedCostUSD < 0 || !finiteBudgetNumber(state.ActualCostUSD) || state.ActualCostUSD < 0 || state.Steps > budget.MaxSteps || state.ToolCalls > budget.MaxToolCalls || state.ModelCalls > budget.MaxModelCalls || state.Retries > budget.MaxRetries || state.EstimatedInput > budget.MaxInputTokens || state.EstimatedOutput > budget.MaxOutputTokens || state.EstimatedReasoning > budget.MaxReasoningTokens || state.ActualInput > budget.MaxInputTokens || state.ActualOutput > budget.MaxOutputTokens || state.ActualReasoning > budget.MaxReasoningTokens || state.EstimatedCostUSD > budget.MaxEstimatedCostUSD || state.ActualCostUSD > budget.MaxActualCostUSD {
		return fmt.Errorf("checkpoint budget state is invalid or exceeds its immutable budget")
	}
	return nil
}

type ResearchCheckpoint struct {
	ContractVersion   string            `json:"contract_version"`
	ID                string            `json:"id"`
	TaskID            string            `json:"task_id"`
	CheckpointVersion int               `json:"checkpoint_version"`
	State             ResearchTaskState `json:"state"`
	CheckpointSummary string            `json:"checkpoint_summary"`
	CreatedAt         time.Time         `json:"created_at"`
}

func NewResearchCheckpoint(state ResearchTaskState, summary string, createdAt time.Time) (ResearchCheckpoint, error) {
	checkpoint := ResearchCheckpoint{ContractVersion: ResearchCheckpointContractV1, TaskID: state.TaskID, CheckpointVersion: state.CheckpointVersion, State: state, CheckpointSummary: summary, CreatedAt: createdAt}
	checkpoint.ID = deriveCheckpointID(checkpoint)
	if err := checkpoint.Validate(); err != nil {
		return ResearchCheckpoint{}, err
	}
	return checkpoint, nil
}

func (checkpoint ResearchCheckpoint) Validate() error {
	if checkpoint.ContractVersion != ResearchCheckpointContractV1 || !validIdentityLike("checkpoint_", checkpoint.ID) || checkpoint.TaskID != checkpoint.State.TaskID || checkpoint.CheckpointVersion != checkpoint.State.CheckpointVersion || strings.TrimSpace(checkpoint.CheckpointSummary) == "" || len(checkpoint.CheckpointSummary) > 8192 || checkpoint.CreatedAt.IsZero() || checkpoint.CreatedAt.Location() != time.UTC {
		return fmt.Errorf("research checkpoint identity or bounded summary is invalid")
	}
	if err := checkpoint.State.Validate(); err != nil {
		return err
	}
	if checkpoint.ID != deriveCheckpointID(checkpoint) {
		return fmt.Errorf("research checkpoint ID does not match immutable state")
	}
	return nil
}

type ResumeContext struct {
	Objective           string   `json:"objective"`
	PlanVersion         string   `json:"plan_version"`
	CompletedStepIDs    []string `json:"completed_step_ids"`
	EvidenceIDs         []string `json:"evidence_ids"`
	UnresolvedGaps      []string `json:"unresolved_gaps"`
	Contradictions      []string `json:"contradictions"`
	CurrentReport       string   `json:"current_report"`
	RemainingSteps      []string `json:"remaining_steps"`
	RemainingToolCalls  int      `json:"remaining_tool_calls"`
	RemainingModelCalls int      `json:"remaining_model_calls"`
}

func (checkpoint ResearchCheckpoint) ReconstructResumeContext(maxBytes int) (ResumeContext, error) {
	if err := checkpoint.Validate(); err != nil {
		return ResumeContext{}, err
	}
	if maxBytes <= 0 {
		return ResumeContext{}, fmt.Errorf("resume context bound is required")
	}
	completed := make([]string, 0, len(checkpoint.State.CompletedSteps))
	completedSet := map[string]struct{}{}
	for _, step := range checkpoint.State.CompletedSteps {
		completed = append(completed, step.StepID)
		completedSet[step.StepID] = struct{}{}
	}
	remaining := make([]string, 0, len(checkpoint.State.PlannedSteps))
	for _, stepID := range checkpoint.State.PlannedSteps {
		if _, ok := completedSet[stepID]; !ok {
			remaining = append(remaining, stepID)
		}
	}
	context := ResumeContext{Objective: checkpoint.State.Objective, PlanVersion: checkpoint.State.PlanVersion, CompletedStepIDs: completed, EvidenceIDs: append([]string(nil), checkpoint.State.EvidenceIDs...), UnresolvedGaps: append([]string(nil), checkpoint.State.UnresolvedGaps...), Contradictions: append([]string(nil), checkpoint.State.Contradictions...), CurrentReport: checkpoint.State.CurrentReport, RemainingSteps: remaining, RemainingToolCalls: checkpoint.State.Budget.MaxToolCalls - checkpoint.State.BudgetState.ToolCalls, RemainingModelCalls: checkpoint.State.Budget.MaxModelCalls - checkpoint.State.BudgetState.ModelCalls}
	encoded, err := json.Marshal(context)
	if err != nil || len(encoded) > maxBytes {
		return ResumeContext{}, fmt.Errorf("bounded resume context exceeds limit")
	}
	return context, nil
}

type CheckpointStore interface {
	Save(context.Context, ResearchCheckpoint) error
	Load(context.Context, string, int) (ResearchCheckpoint, error)
}

type MemoryCheckpointStore struct {
	mu    sync.Mutex
	items map[string]ResearchCheckpoint
}

func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{items: map[string]ResearchCheckpoint{}}
}

func (store *MemoryCheckpointStore) Save(_ context.Context, checkpoint ResearchCheckpoint) error {
	if err := checkpoint.Validate(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	key := checkpoint.TaskID + ":" + fmt.Sprint(checkpoint.CheckpointVersion)
	if existing, ok := store.items[key]; ok && existing.ID != checkpoint.ID {
		return fmt.Errorf("checkpoint version is immutable")
	}
	store.items[key] = checkpoint
	return nil
}

func (store *MemoryCheckpointStore) Load(_ context.Context, taskID string, version int) (ResearchCheckpoint, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	checkpoint, ok := store.items[taskID+":"+fmt.Sprint(version)]
	if !ok {
		return ResearchCheckpoint{}, fmt.Errorf("checkpoint not found")
	}
	if err := checkpoint.Validate(); err != nil {
		return ResearchCheckpoint{}, fmt.Errorf("checkpoint integrity failure: %w", err)
	}
	return checkpoint, nil
}

type PostgresCheckpointStore struct {
	pool *pgxpool.Pool
}

func NewPostgresCheckpointStore(pool *pgxpool.Pool) *PostgresCheckpointStore {
	if pool == nil {
		return nil
	}
	return &PostgresCheckpointStore{pool: pool}
}

func (store *PostgresCheckpointStore) Save(ctx context.Context, checkpoint ResearchCheckpoint) error {
	if store == nil || store.pool == nil {
		return fmt.Errorf("checkpoint store unavailable")
	}
	if err := checkpoint.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	_, err = store.pool.Exec(ctx, `INSERT INTO research_task_checkpoints (checkpoint_id, task_id, checkpoint_version, status, payload, created_at) VALUES ($1,$2,$3,$4,$5::jsonb,$6) ON CONFLICT (checkpoint_id) DO NOTHING`, checkpoint.ID, checkpoint.TaskID, checkpoint.CheckpointVersion, checkpoint.State.Status, payload, checkpoint.CreatedAt)
	if err != nil {
		return fmt.Errorf("save research checkpoint: %w", err)
	}
	return nil
}

func (store *PostgresCheckpointStore) Load(ctx context.Context, taskID string, version int) (ResearchCheckpoint, error) {
	if store == nil || store.pool == nil {
		return ResearchCheckpoint{}, fmt.Errorf("checkpoint store unavailable")
	}
	var payload []byte
	if err := store.pool.QueryRow(ctx, `SELECT payload FROM research_task_checkpoints WHERE task_id = $1 AND checkpoint_version = $2`, taskID, version).Scan(&payload); err != nil {
		return ResearchCheckpoint{}, err
	}
	var checkpoint ResearchCheckpoint
	if err := json.Unmarshal(payload, &checkpoint); err != nil {
		return ResearchCheckpoint{}, fmt.Errorf("decode research checkpoint: %w", err)
	}
	if err := checkpoint.Validate(); err != nil {
		return ResearchCheckpoint{}, fmt.Errorf("checkpoint integrity failure: %w", err)
	}
	return checkpoint, nil
}

var _ CheckpointStore = (*MemoryCheckpointStore)(nil)
var _ CheckpointStore = (*PostgresCheckpointStore)(nil)

func deriveCheckpointID(checkpoint ResearchCheckpoint) string {
	checkpoint.ID = ""
	seed, _ := json.Marshal(checkpoint)
	digest := sha256.Sum256(seed)
	return "checkpoint_" + hex.EncodeToString(digest[:])
}

func validIdentityLike(prefix, value string) bool {
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	return validSHA256Usage(value[len(prefix):])
}
