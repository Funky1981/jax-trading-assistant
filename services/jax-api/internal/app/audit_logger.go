package app

import (
	"context"
	"time"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

type AuditStore interface {
	SaveAuditEvent(ctx context.Context, event domain.AuditEvent) error
}

type AuditLogger struct {
	store AuditStore
	clock func() time.Time
	idFn  func(string) string
}

func NewAuditLogger(store AuditStore) *AuditLogger {
	return &AuditLogger{
		store: store,
		clock: func() time.Time { return time.Now().UTC() },
		idFn:  newAuditID,
	}
}

func (l *AuditLogger) Log(ctx context.Context, event domain.AuditEvent) error {
	if l == nil || l.store == nil {
		return nil
	}
	if event.CorrelationID == "" {
		event.CorrelationID = CorrelationIDFromContext(ctx)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = l.clock()
	}
	if event.ID == "" {
		event.ID = l.idFn("audit")
	}
	return l.store.SaveAuditEvent(ctx, event)
}

func (l *AuditLogger) LogDecision(ctx context.Context, action string, outcome domain.AuditOutcome, payload map[string]any, err error) error {
	event := domain.AuditEvent{
		Action:  action,
		Outcome: outcome,
		Payload: payload,
	}
	if err != nil {
		event.Error = err.Error()
	}
	return l.Log(ctx, event)
}
