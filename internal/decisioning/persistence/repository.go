package persistence

import (
	"context"
	"time"

	"jax-trading-assistant/internal/decisioning/review"
)

type Repository interface {
	SavePipelineResult(context.Context, PipelineResultRecord) error
	GetPipelineResult(context.Context, string) (PipelineResultRecord, error)
	SaveDecisionLog(context.Context, review.DecisionLog) error
	GetDecisionLog(context.Context, string) (review.DecisionLog, error)
	SaveReviewSchedule(context.Context, review.ReviewSchedule) error
	GetReviewSchedule(context.Context, string) (review.ReviewSchedule, error)
	SaveOutcomeReview(context.Context, review.OutcomeReview) error
	GetOutcomeReview(context.Context, string) (review.OutcomeReview, error)
	SaveAuditRecord(context.Context, AuditRecord) error
	ListAuditRecordsForDecision(context.Context, string) ([]AuditRecord, error)
	ListDecisionsPendingReview(context.Context, time.Time) ([]review.ReviewSchedule, error)
}
