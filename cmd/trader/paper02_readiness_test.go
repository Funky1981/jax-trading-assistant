package main

import (
	"testing"
	"time"

	"jax-trading-assistant/internal/modules/exploratorypaper"
)

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

func TestPaper02ActualCalendarUsesExchangeSessions(t *testing.T) {
	calendar, err := configuredExploratoryCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, _ := time.LoadLocation("America/New_York")
	cases := []struct {
		name string
		at   time.Time
		want exploratorypaper.MarketSession
	}{
		{"October 12", time.Date(2026, 10, 12, 12, 0, 0, 0, location), exploratorypaper.SessionOpen},
		{"November 11", time.Date(2026, 11, 11, 12, 0, 0, 0, location), exploratorypaper.SessionOpen},
		{"Thanksgiving", time.Date(2026, 11, 26, 12, 0, 0, 0, location), exploratorypaper.SessionClosed},
		{"November 27 before close", time.Date(2026, 11, 27, 12, 59, 0, 0, location), exploratorypaper.SessionOpen},
		{"November 27 after close", time.Date(2026, 11, 27, 13, 1, 0, 0, location), exploratorypaper.SessionClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := calendar.SessionState(tc.at.UTC())
			if err != nil || got != tc.want {
				t.Fatalf("session=%s err=%v want=%s", got, err, tc.want)
			}
		})
	}
}
