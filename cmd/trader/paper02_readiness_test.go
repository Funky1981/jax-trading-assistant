package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	if readiness.EventSourceIdentity != worldMonitorPullConsumer || readiness.EventSourceContract != worldMonitorProviderContract {
		t.Fatalf("readiness did not expose the canonical disabled source identity: %+v", readiness)
	}
	if readiness.BrokerExecutionAllowed || readiness.ExecutionAuthority != "NONE" || readiness.RuntimeMode != "PAPER" {
		t.Fatalf("unsafe runtime projection: %+v", readiness)
	}
	t.Logf("calendar=%s calendarHash=%s universe=%s universeHash=%s risk=%s riskHash=%s entry=%s entryHash=%s", readiness.CalendarVersion, readiness.CalendarHash, readiness.EligibleUniverseVersion, readiness.EligibleUniverseHash, readiness.RiskPolicyVersion, readiness.RiskPolicyHash, readiness.EntryPolicyVersion, readiness.EntryPolicyHash)
}

func TestPaper02ConfiguredSourceRequiresOperationalHealthAndUsesSafeIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("health probe path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config, err := loadWorldMonitorPullConfig(func(key string) (string, bool) {
		values := map[string]string{"WORLD_MONITOR_PULL_ENABLED": "true", "WORLD_MONITOR_EVENTS_URL": server.URL + worldMonitorEventsPath}
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	ready := assessWorldMonitorPullSource(config, server.Client())
	if !ready.Ready || ready.SourceIdentity != worldMonitorPullConsumer || ready.ProviderContract != worldMonitorProviderContract || ready.EndpointIdentity == "" || ready.Reason != "" {
		t.Fatalf("healthy source was not represented as ready: %+v", ready)
	}
	encoded, err := json.Marshal(ready)
	if err != nil || strings.Contains(string(encoded), server.URL) {
		t.Fatalf("readiness leaked raw endpoint: %s err=%v", encoded, err)
	}
	server.Close()
	blocked := assessWorldMonitorPullSource(config, &http.Client{Timeout: 100 * time.Millisecond})
	if blocked.Ready || !strings.Contains(blocked.Reason, "unavailable") {
		t.Fatalf("unavailable source was not blocked: %+v", blocked)
	}
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
