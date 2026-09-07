package workflow

import (
	"context"
	"testing"
	"time"
)

func TestOperatorHealthReportsUnknownSafetyDependencies(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	report := NewStore().Health(context.Background(), now, HealthDependencies{PersistenceKnown: false, AuditKnown: true, AuditOK: true, SafetyKnown: true})
	if report.Status != HealthUnknown || report.FailureReason == "" || len(report.Alerts) == 0 {
		t.Fatalf("unknown dependency health = %#v", report)
	}
	if report.Safety.AllowLiveTrading || report.Safety.BrokerExecutionAllowed || report.Safety.ExecutionWorkerEnabled || report.Safety.MaximumLeverage != "1x" || report.Safety.BrokerExecutionAuthority != BrokerExecutionAuthorityNone {
		t.Fatalf("unsafe health safety flags = %#v", report.Safety)
	}
}

func TestOperatorHealthSurfacesReconciliationAndBreaker(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	store := NewStore()
	workflow, err := store.Create(ctx, CreateRequest{RiskDecision: acceptedRiskDecision(t, now), Now: now, IdempotencyKey: "create-health"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReconciliationRequired(ctx, TransitionRequest{WorkflowID: workflow.WorkflowID, Actor: "recovery", ActorRole: ActorRecovery, IdempotencyKey: "reconcile-health", Now: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetBreaker(ctx, BreakerRequest{Name: "operator-kill", Tripped: true, Reason: "operator stop", Actor: "operator-1", ActorRole: ActorOperator, IdempotencyKey: "breaker-health", Now: now}); err != nil {
		t.Fatal(err)
	}
	report := store.Health(ctx, now, HealthDependencies{PersistenceKnown: true, PersistenceOK: true, AuditKnown: true, AuditOK: true, SafetyKnown: true})
	if report.Status != HealthUnhealthy || report.ReconciliationCount != 1 || report.ActiveBreakerCount != 1 || !containsAlert(report.Alerts, "RECONCILIATION_REQUIRED") || !containsAlert(report.Alerts, "WORKFLOW_BREAKER_ACTIVE") {
		t.Fatalf("operator health = %#v", report)
	}
}

func containsAlert(alerts []string, want string) bool {
	for _, alert := range alerts {
		if alert == want {
			return true
		}
	}
	return false
}
