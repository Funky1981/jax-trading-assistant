package domain

import "time"

type AuditOutcome string

const (
	AuditOutcomeStarted  AuditOutcome = "started"
	AuditOutcomeSuccess  AuditOutcome = "success"
	AuditOutcomeSkipped  AuditOutcome = "skipped"
	AuditOutcomeRejected AuditOutcome = "rejected"
	AuditOutcomeError    AuditOutcome = "error"
)

type AuditEvent struct {
	ID            string         `json:"id"`
	CorrelationID string         `json:"correlationId"`
	Action        string         `json:"action"`
	Outcome       AuditOutcome   `json:"outcome"`
	Timestamp     time.Time      `json:"timestamp"`
	Payload       map[string]any `json:"payload,omitempty"`
	Error         string         `json:"error,omitempty"`
}
