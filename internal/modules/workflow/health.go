package workflow

import (
	"context"
	"time"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "HEALTHY"
	HealthDegraded  HealthStatus = "DEGRADED"
	HealthUnhealthy HealthStatus = "UNHEALTHY"
	HealthUnknown   HealthStatus = "UNKNOWN"
)

type HealthDependencies struct {
	PersistenceKnown bool `json:"persistence_known"`
	PersistenceOK    bool `json:"persistence_ok"`
	AuditKnown       bool `json:"audit_known"`
	AuditOK          bool `json:"audit_ok"`
	SafetyKnown      bool `json:"safety_known"`
}

type SafetyStatus struct {
	AllowLiveTrading         bool   `json:"allow_live_trading"`
	BrokerExecutionAllowed   bool   `json:"broker_execution_allowed"`
	ExecutionWorkerEnabled   bool   `json:"execution_worker_enabled"`
	MaximumLeverage          string `json:"maximum_leverage"`
	BrokerExecutionAuthority string `json:"broker_execution_authority"`
}

type OperatorHealth struct {
	Status                    HealthStatus       `json:"status"`
	AsOf                      time.Time          `json:"as_of"`
	WorkflowCount             int                `json:"workflow_count"`
	AwaitingConfirmationCount int                `json:"awaiting_confirmation_count"`
	BlockedCount              int                `json:"blocked_count"`
	ReconciliationCount       int                `json:"reconciliation_count"`
	ActiveBreakerCount        int                `json:"active_breaker_count"`
	LastSuccessfulProcessing  *time.Time         `json:"last_successful_processing,omitempty"`
	FailureReason             string             `json:"failure_reason,omitempty"`
	Alerts                    []string           `json:"alerts"`
	Dependencies              HealthDependencies `json:"dependencies"`
	Safety                    SafetyStatus       `json:"safety"`
}

func (s *Store) Health(_ context.Context, now time.Time, dependencies HealthDependencies) OperatorHealth {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	report := OperatorHealth{Status: HealthHealthy, AsOf: now.UTC(), Dependencies: dependencies, Safety: SafetyStatus{AllowLiveTrading: false, BrokerExecutionAllowed: false, ExecutionWorkerEnabled: false, MaximumLeverage: "1x", BrokerExecutionAuthority: BrokerExecutionAuthorityNone}, Alerts: []string{}}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, workflow := range s.workflows {
		report.WorkflowCount++
		switch workflow.State {
		case StateAwaitingHumanConfirmation:
			report.AwaitingConfirmationCount++
		case StateBlocked:
			report.BlockedCount++
		case StateReconciliationRequired:
			report.ReconciliationCount++
		}
		if report.LastSuccessfulProcessing == nil || workflow.UpdatedAt.After(*report.LastSuccessfulProcessing) {
			updated := workflow.UpdatedAt
			report.LastSuccessfulProcessing = &updated
		}
	}
	for _, breaker := range s.breakers {
		if breaker.Tripped {
			report.ActiveBreakerCount++
		}
	}
	if !dependencies.PersistenceKnown || !dependencies.AuditKnown || !dependencies.SafetyKnown {
		report.Status = HealthUnknown
		report.FailureReason = "safety-critical dependency status is unknown"
		report.Alerts = append(report.Alerts, "SAFETY_CRITICAL_DEPENDENCY_UNKNOWN")
	} else if !dependencies.PersistenceOK || !dependencies.AuditOK {
		report.Status = HealthUnhealthy
		report.FailureReason = "safety-critical persistence or audit dependency is unavailable"
		report.Alerts = append(report.Alerts, "SAFETY_CRITICAL_DEPENDENCY_UNAVAILABLE")
	}
	if report.ActiveBreakerCount > 0 {
		report.Status = HealthUnhealthy
		report.FailureReason = "a workflow safety breaker is active"
		report.Alerts = append(report.Alerts, "WORKFLOW_BREAKER_ACTIVE")
	}
	if report.ReconciliationCount > 0 {
		if report.Status == HealthHealthy {
			report.Status = HealthDegraded
		}
		report.Alerts = append(report.Alerts, "RECONCILIATION_REQUIRED")
	}
	if report.BlockedCount > 0 {
		report.Alerts = append(report.Alerts, "WORKFLOW_BLOCKED")
	}
	return report
}
