package main

import "testing"

func TestPaper02RuntimeReadinessUsesRealManifestsAndBlocksDisabledIntake(t *testing.T) {
	t.Setenv("JAX_RUNTIME_MODE", "paper")
	t.Setenv("WORLD_MONITOR_PULL_ENABLED", "false")
	t.Setenv("BROKER_EXECUTION_ALLOWED", "false")
	t.Setenv("MAX_LEVERAGE", "1")
	readiness := loadPaper02RuntimeReadiness()
	if readiness.CalendarReady != true || readiness.CalendarVersion == "" {
		t.Fatalf("calendar manifest was not resolved: %+v", readiness)
	}
	if readiness.EligibleUniverseReady != true || readiness.EligibleUniverseVersion == "" || readiness.EligibleUniverseHash == "" {
		t.Fatalf("eligible universe manifest was not resolved: %+v", readiness)
	}
	if readiness.RiskPolicyReady != true || readiness.RiskPolicyVersion == "" || readiness.RiskPolicyHash == "" {
		t.Fatalf("risk policy identity was not resolved: %+v", readiness)
	}
	if readiness.EntryPolicyReady != true || readiness.EntryPolicyHash == "" {
		t.Fatalf("entry policy identity was not resolved: %+v", readiness)
	}
	if readiness.ProspectiveEventIntakeReady || readiness.OverallReady {
		t.Fatal("disabled genuine intake incorrectly made PAPER-02 ready")
	}
	if readiness.BrokerExecutionAllowed || readiness.ExecutionAuthority != "NONE" || readiness.RuntimeMode != "PAPER" {
		t.Fatalf("unsafe runtime projection: %+v", readiness)
	}
	t.Logf("calendar=%s calendarHash=%s universe=%s universeHash=%s risk=%s riskHash=%s entry=%s entryHash=%s", readiness.CalendarVersion, readiness.CalendarHash, readiness.EligibleUniverseVersion, readiness.EligibleUniverseHash, readiness.RiskPolicyVersion, readiness.RiskPolicyHash, readiness.EntryPolicyVersion, readiness.EntryPolicyHash)
}
