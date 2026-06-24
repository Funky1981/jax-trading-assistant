package persistence

import "time"

const (
	SourceModulePersistence        = "decisioning.persistence"
	AuditActionPipelineResultSaved = "pipeline_result_saved"
	AuditActionDecisionLogSaved    = "decision_log_saved"
	AuditActionReviewScheduleSaved = "review_schedule_saved"
	AuditActionOutcomeReviewSaved  = "outcome_review_saved"
)

type AuditRecord struct {
	AuditID          string    `json:"audit_id"`
	TraceID          string    `json:"trace_id"`
	PipelineID       string    `json:"pipeline_id"`
	DecisionID       string    `json:"decision_id"`
	EventID          string    `json:"event_id"`
	Action           string    `json:"action"`
	Actor            string    `json:"actor"`
	SourceModule     string    `json:"source_module"`
	BeforeState      string    `json:"before_state"`
	AfterState       string    `json:"after_state"`
	Reason           string    `json:"reason"`
	CreatedAt        time.Time `json:"created_at"`
	ForbiddenActions []string  `json:"forbidden_actions"`
}

func auditForPipelineResult(record PipelineResultRecord) AuditRecord {
	return AuditRecord{
		AuditID:          "audit_" + record.PipelineID + "_" + AuditActionPipelineResultSaved,
		TraceID:          record.TraceID,
		PipelineID:       record.PipelineID,
		DecisionID:       record.DecisionID,
		EventID:          record.EventID,
		Action:           AuditActionPipelineResultSaved,
		Actor:            "system",
		SourceModule:     SourceModulePersistence,
		BeforeState:      "",
		AfterState:       record.FinalStatus,
		Reason:           "persist deterministic decision pipeline result",
		CreatedAt:        record.CreatedAt,
		ForbiddenActions: append([]string(nil), record.ForbiddenActions...),
	}
}
