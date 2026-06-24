package persistence

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"jax-trading-assistant/internal/decisioning/review"
)

type MemoryRepository struct {
	mu              sync.RWMutex
	pipelineResults map[string]PipelineResultRecord
	decisionLogs    map[string]review.DecisionLog
	reviewSchedules map[string]review.ReviewSchedule
	outcomeReviews  map[string]review.OutcomeReview
	auditRecords    []AuditRecord
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		pipelineResults: make(map[string]PipelineResultRecord),
		decisionLogs:    make(map[string]review.DecisionLog),
		reviewSchedules: make(map[string]review.ReviewSchedule),
		outcomeReviews:  make(map[string]review.OutcomeReview),
		auditRecords:    []AuditRecord{},
	}
}

func (r *MemoryRepository) SavePipelineResult(_ context.Context, record PipelineResultRecord) error {
	if record.PipelineID == "" {
		return fmt.Errorf("pipeline result pipeline_id is required")
	}
	if record.DecisionID == "" {
		return fmt.Errorf("pipeline result decision_id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pipelineResults[record.PipelineID] = clonePipelineResult(record)
	r.auditRecords = append(r.auditRecords, auditForPipelineResult(record))
	return nil
}

func (r *MemoryRepository) GetPipelineResult(_ context.Context, pipelineID string) (PipelineResultRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.pipelineResults[pipelineID]
	if !ok {
		return PipelineResultRecord{}, fmt.Errorf("pipeline result %q not found", pipelineID)
	}
	return clonePipelineResult(record), nil
}

func (r *MemoryRepository) SaveDecisionLog(_ context.Context, log review.DecisionLog) error {
	if log.DecisionID == "" {
		return fmt.Errorf("decision log decision_id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.decisionLogs[log.DecisionID] = cloneDecisionLog(log)
	r.auditRecords = append(r.auditRecords, AuditRecord{
		AuditID:          "audit_" + log.DecisionID + "_" + AuditActionDecisionLogSaved,
		DecisionID:       log.DecisionID,
		EventID:          log.EventID,
		Action:           AuditActionDecisionLogSaved,
		Actor:            "system",
		SourceModule:     SourceModulePersistence,
		AfterState:       string(log.FinalDecision),
		Reason:           "persist deterministic decision log",
		CreatedAt:        log.CreatedAt,
		ForbiddenActions: append([]string(nil), log.ForbiddenActions...),
	})
	return nil
}

func (r *MemoryRepository) GetDecisionLog(_ context.Context, decisionID string) (review.DecisionLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	log, ok := r.decisionLogs[decisionID]
	if !ok {
		return review.DecisionLog{}, fmt.Errorf("decision log %q not found", decisionID)
	}
	return cloneDecisionLog(log), nil
}

func (r *MemoryRepository) SaveReviewSchedule(_ context.Context, schedule review.ReviewSchedule) error {
	if schedule.DecisionID == "" {
		return fmt.Errorf("review schedule decision_id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reviewSchedules[schedule.DecisionID] = cloneReviewSchedule(schedule)
	r.auditRecords = append(r.auditRecords, AuditRecord{
		AuditID:      "audit_" + schedule.DecisionID + "_" + AuditActionReviewScheduleSaved,
		DecisionID:   schedule.DecisionID,
		Action:       AuditActionReviewScheduleSaved,
		Actor:        "system",
		SourceModule: SourceModulePersistence,
		AfterState:   string(schedule.ReviewStatus),
		Reason:       "persist deterministic review schedule",
		CreatedAt:    schedule.CreatedAt,
	})
	return nil
}

func (r *MemoryRepository) GetReviewSchedule(_ context.Context, decisionID string) (review.ReviewSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedule, ok := r.reviewSchedules[decisionID]
	if !ok {
		return review.ReviewSchedule{}, fmt.Errorf("review schedule %q not found", decisionID)
	}
	return cloneReviewSchedule(schedule), nil
}

func (r *MemoryRepository) SaveOutcomeReview(_ context.Context, outcome review.OutcomeReview) error {
	if outcome.ReviewID == "" {
		return fmt.Errorf("outcome review review_id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.outcomeReviews[outcome.ReviewID] = cloneOutcomeReview(outcome)
	r.auditRecords = append(r.auditRecords, AuditRecord{
		AuditID:          "audit_" + outcome.ReviewID + "_" + AuditActionOutcomeReviewSaved,
		DecisionID:       outcome.DecisionID,
		EventID:          outcome.EventID,
		Action:           AuditActionOutcomeReviewSaved,
		Actor:            "system",
		SourceModule:     SourceModulePersistence,
		AfterState:       outcome.FinalDecision,
		Reason:           "persist deterministic outcome review",
		CreatedAt:        outcome.CreatedAt,
		ForbiddenActions: append([]string(nil), outcome.ForbiddenActions...),
	})
	return nil
}

func (r *MemoryRepository) GetOutcomeReview(_ context.Context, reviewID string) (review.OutcomeReview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	outcome, ok := r.outcomeReviews[reviewID]
	if !ok {
		return review.OutcomeReview{}, fmt.Errorf("outcome review %q not found", reviewID)
	}
	return cloneOutcomeReview(outcome), nil
}

func (r *MemoryRepository) SaveAuditRecord(_ context.Context, record AuditRecord) error {
	if record.AuditID == "" {
		return fmt.Errorf("audit record audit_id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	record.ForbiddenActions = append([]string(nil), record.ForbiddenActions...)
	r.auditRecords = append(r.auditRecords, record)
	return nil
}

func (r *MemoryRepository) ListAuditRecordsForDecision(_ context.Context, decisionID string) ([]AuditRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]AuditRecord, 0)
	for _, record := range r.auditRecords {
		if record.DecisionID == decisionID {
			record.ForbiddenActions = append([]string(nil), record.ForbiddenActions...)
			records = append(records, record)
		}
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].AuditID < records[j].AuditID
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})
	return records, nil
}

func (r *MemoryRepository) ListDecisionsPendingReview(_ context.Context, asOf time.Time) ([]review.ReviewSchedule, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pending := make([]review.ReviewSchedule, 0)
	for _, schedule := range r.reviewSchedules {
		if schedule.ReviewStatus == review.ReviewStatusCompleted || schedule.ReviewStatus == review.ReviewStatusCancelled {
			continue
		}
		if schedule.NextReviewAt.Before(asOf) || schedule.NextReviewAt.Equal(asOf) {
			pending = append(pending, cloneReviewSchedule(schedule))
		}
	}
	sort.SliceStable(pending, func(i, j int) bool {
		if pending[i].NextReviewAt.Equal(pending[j].NextReviewAt) {
			return pending[i].DecisionID < pending[j].DecisionID
		}
		return pending[i].NextReviewAt.Before(pending[j].NextReviewAt)
	})
	return pending, nil
}

func clonePipelineResult(record PipelineResultRecord) PipelineResultRecord {
	record.SourceModules = append([]string(nil), record.SourceModules...)
	record.AllowedActions = append([]string(nil), record.AllowedActions...)
	record.ForbiddenActions = append([]string(nil), record.ForbiddenActions...)
	record.RiskAssessmentSummary.VetoReasons = append([]string(nil), record.RiskAssessmentSummary.VetoReasons...)
	record.RiskAssessmentSummary.DowngradeReasons = append([]string(nil), record.RiskAssessmentSummary.DowngradeReasons...)
	record.RiskAssessmentSummary.RequiredActions = append([]string(nil), record.RiskAssessmentSummary.RequiredActions...)
	record.ResearchEvidenceSummary.Warnings = append([]string(nil), record.ResearchEvidenceSummary.Warnings...)
	if record.PaperTicketSummary != nil {
		summary := *record.PaperTicketSummary
		record.PaperTicketSummary = &summary
	}
	record.ReviewSchedule = cloneReviewSchedule(record.ReviewSchedule)
	record.ValidationWarnings = append([]string(nil), record.ValidationWarnings...)
	record.ValidationErrors = append([]string(nil), record.ValidationErrors...)
	return record
}

func cloneDecisionLog(log review.DecisionLog) review.DecisionLog {
	log.SupportingReasons = append([]string(nil), log.SupportingReasons...)
	log.AllowedActions = append([]string(nil), log.AllowedActions...)
	log.ForbiddenActions = append([]string(nil), log.ForbiddenActions...)
	log.RequiredConfirmations = append([]string(nil), log.RequiredConfirmations...)
	log.InvalidationConditions = append([]string(nil), log.InvalidationConditions...)
	log.RiskAssessmentSummary.VetoReasons = append([]string(nil), log.RiskAssessmentSummary.VetoReasons...)
	log.RiskAssessmentSummary.DowngradeReasons = append([]string(nil), log.RiskAssessmentSummary.DowngradeReasons...)
	log.RiskAssessmentSummary.RequiredActions = append([]string(nil), log.RiskAssessmentSummary.RequiredActions...)
	log.ResearchEvidenceSummary.Warnings = append([]string(nil), log.ResearchEvidenceSummary.Warnings...)
	log.ReviewSchedule = cloneReviewSchedule(log.ReviewSchedule)
	log.MemoryTags = append([]string(nil), log.MemoryTags...)
	return log
}

func cloneReviewSchedule(schedule review.ReviewSchedule) review.ReviewSchedule {
	schedule.ReviewWindows = append([]string(nil), schedule.ReviewWindows...)
	return schedule
}

func cloneOutcomeReview(outcome review.OutcomeReview) review.OutcomeReview {
	outcome.MemoryTags = append([]string(nil), outcome.MemoryTags...)
	outcome.ForbiddenActions = append([]string(nil), outcome.ForbiddenActions...)
	outcome.Lesson.EvidenceRefs = append([]string(nil), outcome.Lesson.EvidenceRefs...)
	return outcome
}
