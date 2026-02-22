package guardrails

import (
	"context"
	"testing"
	"time"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func okProbe(name string) *FuncProbe {
	return NewFuncProbe(name, func(_ context.Context) CheckResult {
		return CheckResult{Name: name, Status: StatusOK, Message: "healthy"}
	})
}

func failProbe(name string) *FuncProbe {
	return NewFuncProbe(name, func(_ context.Context) CheckResult {
		return CheckResult{Name: name, Status: StatusFailed, Message: "connection refused"}
	})
}

func newLog(t *testing.T) *IncidentLog {
	t.Helper()
	il, err := OpenIncidentLog(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return il
}

// ─── HealthMonitor ────────────────────────────────────────────────────────────

func TestHealthMonitor_AllOK(t *testing.T) {
	cfg := DefaultMonitorConfig()
	m := NewHealthMonitor(cfg, nil, okProbe("feed"), okProbe("broker"))
	results := m.RunOnce(context.Background())
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != StatusOK {
			t.Errorf("probe %q: want OK, got %q", r.Name, r.Status)
		}
	}
	if m.IsHalted() {
		t.Error("monitor should not be halted when all probes pass")
	}
}

func TestHealthMonitor_FailStreakBelowThreshold(t *testing.T) {
	cfg := DefaultMonitorConfig()
	cfg.FailuresBeforeHalt = 3

	haltCalled := false
	m := NewHealthMonitor(cfg, func(_ string) { haltCalled = true }, failProbe("feed"))

	m.RunOnce(context.Background())
	m.RunOnce(context.Background())
	// failStreak = 2, threshold = 3 → should not halt yet
	if m.IsHalted() || haltCalled {
		t.Error("should not halt after 2 failures when threshold is 3")
	}
}

func TestHealthMonitor_HaltAfterConsecutiveFailures(t *testing.T) {
	cfg := DefaultMonitorConfig()
	cfg.FailuresBeforeHalt = 3

	haltReasonCh := make(chan string, 1)
	m := NewHealthMonitor(cfg, func(reason string) {
		haltReasonCh <- reason
	}, failProbe("broker"))

	for range 3 {
		m.RunOnce(context.Background())
	}

	select {
	case <-haltReasonCh:
		// ok
	case <-time.After(time.Second):
		t.Fatal("halt callback not called after 3 failures")
	}
	if !m.IsHalted() {
		t.Error("monitor should be halted")
	}
}

func TestHealthMonitor_ResetHalt(t *testing.T) {
	cfg := DefaultMonitorConfig()
	cfg.FailuresBeforeHalt = 1

	called := make(chan string, 1)
	m := NewHealthMonitor(cfg, func(r string) { called <- r }, failProbe("feed"))
	m.RunOnce(context.Background())
	<-called // drain halt callback

	if !m.IsHalted() {
		t.Fatal("should be halted")
	}
	m.ResetHalt()
	if m.IsHalted() {
		t.Error("should not be halted after reset")
	}
}

func TestHealthMonitor_PartialFailure_NonCritical(t *testing.T) {
	cfg := DefaultMonitorConfig()
	cfg.FailuresBeforeHalt = 1
	cfg.CriticalProbes = []string{"broker"} // only "broker" is critical

	haltCalled := false
	m := NewHealthMonitor(cfg, func(_ string) { haltCalled = true },
		failProbe("feed"), // not critical
		okProbe("broker")) // critical, passes

	m.RunOnce(context.Background())
	if haltCalled {
		t.Error("non-critical probe failure should not trigger halt")
	}
}

func TestHealthMonitor_RegisterProbeAtRuntime(t *testing.T) {
	cfg := DefaultMonitorConfig()
	m := NewHealthMonitor(cfg, nil, okProbe("base"))
	m.RegisterProbe(okProbe("dynamic"))

	results := m.RunOnce(context.Background())
	if len(results) != 2 {
		t.Fatalf("want 2 probes, got %d", len(results))
	}
}

func TestHealthMonitor_LatestResults(t *testing.T) {
	cfg := DefaultMonitorConfig()
	m := NewHealthMonitor(cfg, nil, okProbe("feed"), failProbe("exec"))
	m.RunOnce(context.Background())

	latest := m.Latest()
	if len(latest) != 2 {
		t.Fatalf("want 2 latest results, got %d", len(latest))
	}
	if latest["feed"].Status != StatusOK {
		t.Errorf("feed should be OK")
	}
	if latest["exec"].Status != StatusFailed {
		t.Errorf("exec should be Failed")
	}
}

// ─── IncidentLog ─────────────────────────────────────────────────────────────

func TestIncidentLog_OpenAndGet(t *testing.T) {
	il := newLog(t)
	inc, err := il.Open("Feed disconnected", SeverityCritical, "health_monitor")
	if err != nil {
		t.Fatal(err)
	}
	if inc.Status != IncidentOpen {
		t.Fatalf("new incident should be open, got %q", inc.Status)
	}

	got, err := il.Get(inc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Feed disconnected" {
		t.Fatalf("unexpected title: %q", got.Title)
	}
}

func TestIncidentLog_Acknowledge(t *testing.T) {
	il := newLog(t)
	inc, _ := il.Open("Broker spike", SeverityWarning, "monitor")
	if err := il.Acknowledge(inc.ID, "ops-team aware"); err != nil {
		t.Fatal(err)
	}
	got, _ := il.Get(inc.ID)
	if got.Status != IncidentAcked {
		t.Fatalf("want acked, got %q", got.Status)
	}
	if len(got.Notes) == 0 || got.Notes[0] != "ops-team aware" {
		t.Fatalf("note not recorded: %+v", got.Notes)
	}
}

func TestIncidentLog_Resolve(t *testing.T) {
	il := newLog(t)
	inc, _ := il.Open("Exec latency spike", SeverityWarning, "monitor")
	_ = il.Acknowledge(inc.ID, "")
	if err := il.Resolve(inc.ID, "latency back to normal"); err != nil {
		t.Fatal(err)
	}
	got, _ := il.Get(inc.ID)
	if got.Status != IncidentResolved {
		t.Fatalf("want resolved, got %q", got.Status)
	}
}

func TestIncidentLog_Persistence(t *testing.T) {
	dir := t.TempDir()
	il, _ := OpenIncidentLog(dir)
	inc, _ := il.Open("Persist test", SeverityInfo, "test")

	il2, err := OpenIncidentLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := il2.Get(inc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Persist test" {
		t.Fatalf("persistence failed: %+v", got)
	}
}

func TestIncidentLog_ListFilter(t *testing.T) {
	il := newLog(t)
	inc1, _ := il.Open("A", SeverityWarning, "t")
	_, _ = il.Open("B", SeverityCritical, "t")
	_ = il.Resolve(inc1.ID, "")

	open := il.List(IncidentOpen)
	if len(open) != 1 {
		t.Fatalf("want 1 open incident, got %d", len(open))
	}
	all := il.List("")
	if len(all) != 2 {
		t.Fatalf("want 2 total, got %d", len(all))
	}
}

func TestIncidentLog_NotFound(t *testing.T) {
	il := newLog(t)
	if err := il.Acknowledge("nonexistent-id", ""); err == nil {
		t.Fatal("expected error for unknown incident ID")
	}
}

// ─── OverrideController ───────────────────────────────────────────────────────

func TestOverrideController_DefaultAllowsEntry(t *testing.T) {
	c := NewOverrideController()
	if !c.AllowEntry() {
		t.Error("default state should allow entry")
	}
	if !c.AllowAnyActivity() {
		t.Error("default state should allow any activity")
	}
}

func TestOverrideController_Pause(t *testing.T) {
	c := NewOverrideController()
	c.Pause("earnings blackout")
	if c.AllowEntry() {
		t.Error("paused controller should not allow entry")
	}
	if !c.AllowAnyActivity() {
		t.Error("paused controller should still allow existing positions")
	}
	state, reason := c.State()
	if state != OverridePause || reason != "earnings blackout" {
		t.Fatalf("state=%q reason=%q", state, reason)
	}
}

func TestOverrideController_Halt(t *testing.T) {
	c := NewOverrideController()
	c.Halt("critical drawdown")
	if c.AllowEntry() {
		t.Error("halted controller should not allow entry")
	}
	if c.AllowAnyActivity() {
		t.Error("halted controller should not allow any activity")
	}
}

func TestOverrideController_Resume(t *testing.T) {
	c := NewOverrideController()
	c.Pause("scheduled maintenance")
	c.Resume("maintenance complete")
	if !c.AllowEntry() {
		t.Error("resumed controller should allow entry")
	}
	state, _ := c.State()
	if state != OverrideNone {
		t.Fatalf("want OverrideNone after resume, got %q", state)
	}
}

func TestOverrideController_Since(t *testing.T) {
	c := NewOverrideController()
	before := time.Now().UTC().Add(-time.Second)
	c.Halt("test")
	after := time.Now().UTC().Add(time.Second)
	since := c.Since()
	if since.Before(before) || since.After(after) {
		t.Fatalf("Since out of range: %v", since)
	}
}

// ─── BuildReport ─────────────────────────────────────────────────────────────

func TestBuildReport(t *testing.T) {
	cfg := DefaultMonitorConfig()
	m := NewHealthMonitor(cfg, nil, okProbe("broker"), okProbe("feed"))
	m.RunOnce(context.Background())

	c := NewOverrideController()
	c.Pause("test")

	il := newLog(t)
	_, _ = il.Open("Test incident", SeverityInfo, "test")

	report := BuildReport(m, c, il)
	if report.Override != OverridePause {
		t.Errorf("want OverridePause, got %q", report.Override)
	}
	if report.OpenIncidents != 1 {
		t.Errorf("want 1 open incident, got %d", report.OpenIncidents)
	}
	if len(report.ProbeStates) != 2 {
		t.Errorf("want 2 probe states, got %d", len(report.ProbeStates))
	}
	_ = report.String() // smoke-test stringer
}

func TestTradingAllowed(t *testing.T) {
	cfg := DefaultMonitorConfig()
	m := NewHealthMonitor(cfg, nil, okProbe("ok"))
	c := NewOverrideController()

	if !TradingAllowed(m, c) {
		t.Error("trading should be allowed in default state")
	}

	c.Halt("test")
	if TradingAllowed(m, c) {
		t.Error("trading should not be allowed when controller is halted")
	}
}
