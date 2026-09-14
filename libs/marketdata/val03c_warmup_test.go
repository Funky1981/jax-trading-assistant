package marketdata

import (
	"testing"
	"time"
)

func val03cSessions(startDay, count int) []VAL03CSession {
	sessions := make([]VAL03CSession, count)
	start := time.Date(2016, 1, startDay, 0, 0, 0, 0, time.UTC)
	for i := range sessions {
		sessions[i] = VAL03CSession{ProviderDate: start.AddDate(0, 0, i).Format("2006-01-02"), Valid: true}
	}
	return sessions
}

func TestDeriveVAL03CInitializationRequiresFullLookback(t *testing.T) {
	result, err := DeriveVAL03CInitialization(val03cSessions(1, VAL03CInitializationSessions-1))
	if err != nil {
		t.Fatalf("DeriveVAL03CInitialization() error = %v", err)
	}
	if result.State != VAL03CWarmupOnly || result.InitializationComplete != "" {
		t.Fatalf("result = %+v, want warm-up only", result)
	}
}

func TestDeriveVAL03CInitializationCompletesOnTwoHundredthValidSession(t *testing.T) {
	result, err := DeriveVAL03CInitialization(val03cSessions(1, VAL03CInitializationSessions))
	if err != nil {
		t.Fatalf("DeriveVAL03CInitialization() error = %v", err)
	}
	if result.State != VAL03CInitializationComplete || result.InitializationComplete == "" {
		t.Fatalf("result = %+v, want initialization complete", result)
	}
	if result.WarmupOnlyCount != VAL03CInitializationSessions-1 {
		t.Fatalf("WarmupOnlyCount = %d, want %d", result.WarmupOnlyCount, VAL03CInitializationSessions-1)
	}
}

func TestDeriveVAL03CInitializationSkipsInvalidSessions(t *testing.T) {
	sessions := val03cSessions(1, VAL03CInitializationSessions-1)
	sessions = append(sessions, VAL03CSession{ProviderDate: "2017-01-01", Valid: false})
	sessions = append(sessions, VAL03CSession{ProviderDate: "2017-01-02", Valid: true})
	result, err := DeriveVAL03CInitialization(sessions)
	if err != nil {
		t.Fatalf("DeriveVAL03CInitialization() error = %v", err)
	}
	if result.ValidSessionCount != VAL03CInitializationSessions || result.State != VAL03CInitializationComplete {
		t.Fatalf("result = %+v, want invalid session excluded from count", result)
	}
}

func TestDeriveVAL03CInitializationResetsAtStructuralBoundary(t *testing.T) {
	sessions := val03cSessions(1, VAL03CInitializationSessions)
	firstAfterBoundary := time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)
	sessions = append(sessions, VAL03CSession{ProviderDate: firstAfterBoundary.Format("2006-01-02"), Valid: true, StructuralBoundary: true})
	for i := 0; i < VAL03CInitializationSessions-2; i++ {
		sessions = append(sessions, VAL03CSession{ProviderDate: firstAfterBoundary.AddDate(0, 0, i+1).Format("2006-01-02"), Valid: true})
	}
	result, err := DeriveVAL03CInitialization(sessions)
	if err != nil {
		t.Fatalf("DeriveVAL03CInitialization() error = %v", err)
	}
	if result.InitializationComplete != sessions[VAL03CInitializationSessions-1].ProviderDate {
		t.Fatalf("InitializationComplete = %s, want pre-boundary completion", result.InitializationComplete)
	}
	if result.State != VAL03CWarmupOnly {
		t.Fatalf("State = %s, want post-boundary warm-up", result.State)
	}
	if result.WarmupOnlyCount != VAL03CInitializationSessions-1+VAL03CInitializationSessions-1 {
		t.Fatalf("WarmupOnlyCount = %d, want %d", result.WarmupOnlyCount, VAL03CInitializationSessions-1+VAL03CInitializationSessions-1)
	}
}

func TestDeriveVAL03CInitializationRejectsHoldoutDate(t *testing.T) {
	_, err := DeriveVAL03CInitialization([]VAL03CSession{{ProviderDate: "2025-01-02", Valid: true}})
	if err == nil {
		t.Fatal("DeriveVAL03CInitialization() accepted sealed holdout date")
	}
}
