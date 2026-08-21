package canonical

import "time"

type ResearchRunType string

const (
	ResearchRunTypeAnalysis   ResearchRunType = "analysis"
	ResearchRunTypeBacktest   ResearchRunType = "backtest"
	ResearchRunTypeEvaluation ResearchRunType = "evaluation"
	ResearchRunTypeScreening  ResearchRunType = "screening"
	ResearchRunTypeSynthesis  ResearchRunType = "synthesis"
	ResearchRunTypeOther      ResearchRunType = "other"
)

type ResearchRunStatus string

const (
	ResearchRunStatusPending   ResearchRunStatus = "pending"
	ResearchRunStatusRunning   ResearchRunStatus = "running"
	ResearchRunStatusSucceeded ResearchRunStatus = "succeeded"
	ResearchRunStatusFailed    ResearchRunStatus = "failed"
	ResearchRunStatusCancelled ResearchRunStatus = "cancelled"
)

// RunFailure makes a material research failure explicit without exposing a
// runtime-specific error object or stack trace as canonical state.
type RunFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ResearchRun represents one bounded research computation or process. It
// identifies inputs, a versioned method/model/tool, and outputs, but defines no
// agent or orchestration framework.
type ResearchRun struct {
	ContractVersion ContractVersion   `json:"contract_version"`
	ID              ResearchRunID     `json:"id"`
	Type            ResearchRunType   `json:"type"`
	Status          ResearchRunStatus `json:"status"`
	Method          ComponentRef      `json:"method"`
	InputRefs       []ContractRef     `json:"input_refs"`
	OutputRefs      []ContractRef     `json:"output_refs,omitempty"`
	ParentRunID     *ResearchRunID    `json:"parent_run_id,omitempty"`
	Failure         *RunFailure       `json:"failure,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
}

func (run ResearchRun) Validate() error {
	const contract = "research_run"
	if err := validateVersion(contract, run.ContractVersion, ResearchRunContractV1); err != nil {
		return err
	}
	if err := validateCanonicalID(contract, "id", string(run.ID), "run_"); err != nil {
		return err
	}
	switch run.Type {
	case ResearchRunTypeAnalysis, ResearchRunTypeBacktest, ResearchRunTypeEvaluation,
		ResearchRunTypeScreening, ResearchRunTypeSynthesis, ResearchRunTypeOther:
	default:
		return invalid(contract, "type", "is not supported")
	}
	if err := validateComponent(contract, "method", run.Method); err != nil {
		return err
	}
	if len(run.InputRefs) == 0 {
		return invalid(contract, "input_refs", "requires at least one identified input")
	}
	if err := validateContractRefs(contract, "input_refs", run.InputRefs, nil); err != nil {
		return err
	}
	if err := validateContractRefs(contract, "output_refs", run.OutputRefs, nil); err != nil {
		return err
	}
	for _, ref := range run.InputRefs {
		if ref.Kind == ContractKindResearchRun && ref.ID == string(run.ID) {
			return invalid(contract, "input_refs", "must not contain the run itself")
		}
	}
	for _, ref := range run.OutputRefs {
		if ref.Kind == ContractKindResearchRun && ref.ID == string(run.ID) {
			return invalid(contract, "output_refs", "must not contain the run itself")
		}
	}
	if run.ParentRunID != nil {
		if err := validateCanonicalID(contract, "parent_run_id", string(*run.ParentRunID), "run_"); err != nil {
			return err
		}
		if *run.ParentRunID == run.ID {
			return invalid(contract, "parent_run_id", "must not refer to itself")
		}
	}
	if err := validateRequiredUTC(contract, "created_at", run.CreatedAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "started_at", run.StartedAt); err != nil {
		return err
	}
	if err := validateOptionalUTC(contract, "completed_at", run.CompletedAt); err != nil {
		return err
	}
	if run.StartedAt != nil && run.StartedAt.Before(run.CreatedAt) {
		return invalid(contract, "started_at", "must not precede created_at")
	}
	if run.CompletedAt != nil {
		anchor := run.CreatedAt
		if run.StartedAt != nil {
			anchor = *run.StartedAt
		}
		if run.CompletedAt.Before(anchor) {
			return invalid(contract, "completed_at", "must not precede the run start anchor")
		}
	}

	switch run.Status {
	case ResearchRunStatusPending:
		if run.StartedAt != nil || run.CompletedAt != nil || run.Failure != nil || len(run.OutputRefs) != 0 {
			return invalid(contract, "status", "pending runs cannot have start, completion, failure, or outputs")
		}
	case ResearchRunStatusRunning:
		if run.StartedAt == nil || run.CompletedAt != nil || run.Failure != nil {
			return invalid(contract, "status", "running runs require started_at and cannot have completion or failure")
		}
	case ResearchRunStatusSucceeded:
		if run.StartedAt == nil || run.CompletedAt == nil || run.Failure != nil || len(run.OutputRefs) == 0 {
			return invalid(contract, "status", "succeeded runs require start, completion, and outputs without failure")
		}
	case ResearchRunStatusFailed:
		if run.StartedAt == nil || run.CompletedAt == nil || run.Failure == nil {
			return invalid(contract, "status", "failed runs require start, completion, and failure details")
		}
		if err := validateCode(contract, "failure.code", run.Failure.Code); err != nil {
			return err
		}
		if err := validateRequiredText(contract, "failure.message", run.Failure.Message, maxDescription); err != nil {
			return err
		}
	case ResearchRunStatusCancelled:
		if run.CompletedAt == nil || run.Failure != nil {
			return invalid(contract, "status", "cancelled runs require completed_at and cannot carry failure details")
		}
	default:
		return invalid(contract, "status", "is not supported")
	}
	return nil
}
