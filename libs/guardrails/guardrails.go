// Package guardrails implements L14 + L18:
//
//   - L14: Platform guardrails — feed, broker, and execution integrity
//     monitoring. A HealthMonitor periodically checks registered probes
//     and trips a SystemHalt when critical failures are detected.
//   - L18: Incident handling + override controls. An IncidentLog records
//     operational events; an OverrideController allows operators to
//     manually pause/resume trading or force-halt the system.
package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── CheckResult (L14) ────────────────────────────────────────────────────────

// CheckStatus is the result of a single health probe.
type CheckStatus string

const (
	StatusOK       CheckStatus = "ok"
	StatusDegraded CheckStatus = "degraded"
	StatusFailed   CheckStatus = "failed"
)

// CheckResult holds the outcome of one health probe.
type CheckResult struct {
	Name      string
	Status    CheckStatus
	Message   string
	CheckedAt time.Time
}

// HealthProbe is any component that can report its own health.
type HealthProbe interface {
	// ProbeName returns the display name for this probe.
	ProbeName() string
	// Check performs a health check and returns the result.
	Check(ctx context.Context) CheckResult
}

// ─── HealthMonitor (L14) ─────────────────────────────────────────────────────

// HaltCallback is called when the monitor decides the system must halt.
// Implementations should trigger an emergency close-all-positions sequence.
type HaltCallback func(reason string)

// MonitorConfig controls the health monitor's polling and escalation logic.
type MonitorConfig struct {
	// Interval between checks (default 30s).
	Interval time.Duration
	// FailuresBeforeHalt: how many consecutive cycles where ≥1 critical
	// probe fails before triggering a system halt (default 3).
	FailuresBeforeHalt int
	// CriticalProbes lists ProbeName values that are escalated to halt.
	// An empty list means every probe is critical.
	CriticalProbes []string
}

// DefaultMonitorConfig returns sensible defaults.
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		Interval:           30 * time.Second,
		FailuresBeforeHalt: 3,
	}
}

// HealthMonitor runs periodic integrity checks and escalates to halt.
type HealthMonitor struct {
	cfg            MonitorConfig
	probes         []HealthProbe
	haltCb         HaltCallback
	mu             sync.RWMutex
	latest         map[string]CheckResult
	failStreak     int
	halted         bool
	criticalSet    map[string]bool
}

// NewHealthMonitor creates a HealthMonitor.
// haltCb may be nil (no halt escalation, useful for tests/monitoring-only mode).
func NewHealthMonitor(cfg MonitorConfig, haltCb HaltCallback, probes ...HealthProbe) *HealthMonitor {
	cs := make(map[string]bool, len(cfg.CriticalProbes))
	for _, name := range cfg.CriticalProbes {
		cs[name] = true
	}
	return &HealthMonitor{
		cfg:         cfg,
		probes:      probes,
		haltCb:      haltCb,
		latest:      make(map[string]CheckResult),
		criticalSet: cs,
	}
}

// RegisterProbe adds a probe to the monitor at runtime.
func (m *HealthMonitor) RegisterProbe(p HealthProbe) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.probes = append(m.probes, p)
}

// RunOnce performs a single round of checks synchronously and returns the results.
func (m *HealthMonitor) RunOnce(ctx context.Context) []CheckResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	var results []CheckResult
	criticalFailed := false

	for _, probe := range m.probes {
		r := probe.Check(ctx)
		m.latest[r.Name] = r
		results = append(results, r)

		if r.Status == StatusFailed {
			isCritical := len(m.criticalSet) == 0 || m.criticalSet[r.Name]
			if isCritical {
				criticalFailed = true
				log.Printf("[guardrail] CRITICAL probe=%q status=failed msg=%q", r.Name, r.Message)
			} else {
				log.Printf("[guardrail] probe=%q status=failed msg=%q", r.Name, r.Message)
			}
		}
	}

	if criticalFailed {
		m.failStreak++
		if !m.halted && m.failStreak >= m.cfg.FailuresBeforeHalt && m.haltCb != nil {
			m.halted = true
			reason := fmt.Sprintf("health monitor: %d consecutive critical failures", m.failStreak)
			log.Printf("[guardrail] HALT triggered: %s", reason)
			go m.haltCb(reason)
		}
	} else {
		m.failStreak = 0
	}

	return results
}

// Latest returns the most recent check result for each probe.
func (m *HealthMonitor) Latest() map[string]CheckResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]CheckResult, len(m.latest))
	for k, v := range m.latest {
		out[k] = v
	}
	return out
}

// IsHalted returns true if the monitor has triggered a system halt.
func (m *HealthMonitor) IsHalted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.halted
}

// ResetHalt clears the halt state (operator override after manual review).
func (m *HealthMonitor) ResetHalt() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.halted = false
	m.failStreak = 0
	log.Printf("[guardrail] halt reset by operator")
}

// Run starts the periodic check loop. It blocks until ctx is cancelled.
func (m *HealthMonitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.RunOnce(ctx)
		}
	}
}

// ─── FuncProbe: convenience wrapper ──────────────────────────────────────────

// FuncProbe wraps a function as a HealthProbe.
type FuncProbe struct {
	name string
	fn   func(ctx context.Context) CheckResult
}

// NewFuncProbe creates a HealthProbe from a function.
func NewFuncProbe(name string, fn func(ctx context.Context) CheckResult) *FuncProbe {
	return &FuncProbe{name: name, fn: fn}
}

func (f *FuncProbe) ProbeName() string { return f.name }
func (f *FuncProbe) Check(ctx context.Context) CheckResult {
	r := f.fn(ctx)
	if r.Name == "" {
		r.Name = f.name
	}
	if r.CheckedAt.IsZero() {
		r.CheckedAt = time.Now().UTC()
	}
	return r
}

// ─── Incident (L18) ──────────────────────────────────────────────────────────

// Severity classifies an incident by operational urgency.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// IncidentStatus tracks the lifecycle of an incident.
type IncidentStatus string

const (
	IncidentOpen     IncidentStatus = "open"
	IncidentAcked    IncidentStatus = "acknowledged"
	IncidentResolved IncidentStatus = "resolved"
)

// Incident represents an operational event that requires attention.
type Incident struct {
	ID         string         `json:"id"`
	Title      string         `json:"title"`
	Severity   Severity       `json:"severity"`
	Status     IncidentStatus `json:"status"`
	Source     string         `json:"source"`   // e.g. "health_monitor", "operator"
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Notes      []string       `json:"notes,omitempty"`
}

const incidentFile = "incidents.json"

// IncidentLog is a JSON-persisted log of operational incidents.
type IncidentLog struct {
	mu        sync.Mutex
	dir       string
	incidents map[string]*Incident
}

// OpenIncidentLog loads (or creates) an incident log in dir.
func OpenIncidentLog(dir string) (*IncidentLog, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("guardrails.OpenIncidentLog: mkdir: %w", err)
	}
	il := &IncidentLog{dir: dir, incidents: make(map[string]*Incident)}
	return il, il.load()
}

// Open creates a new incident and persists it.
func (il *IncidentLog) Open(title string, severity Severity, source string) (*Incident, error) {
	il.mu.Lock()
	defer il.mu.Unlock()

	now := time.Now().UTC()
	id := fmt.Sprintf("INC-%d", now.UnixMilli())
	inc := &Incident{
		ID:        id,
		Title:     title,
		Severity:  severity,
		Status:    IncidentOpen,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
	il.incidents[id] = inc
	log.Printf("[guardrail] incident opened: id=%s severity=%s title=%q", id, severity, title)
	return inc, il.save()
}

// Acknowledge marks an incident as acknowledged by an operator.
func (il *IncidentLog) Acknowledge(id, note string) error {
	return il.transition(id, IncidentAcked, note)
}

// Resolve marks an incident as resolved.
func (il *IncidentLog) Resolve(id, note string) error {
	return il.transition(id, IncidentResolved, note)
}

func (il *IncidentLog) transition(id string, status IncidentStatus, note string) error {
	il.mu.Lock()
	defer il.mu.Unlock()
	inc, ok := il.incidents[id]
	if !ok {
		return fmt.Errorf("guardrails.IncidentLog: incident %q not found", id)
	}
	inc.Status = status
	inc.UpdatedAt = time.Now().UTC()
	if note != "" {
		inc.Notes = append(inc.Notes, note)
	}
	log.Printf("[guardrail] incident %s: status=%s", id, status)
	return il.save()
}

// AddNote appends a note to an incident.
func (il *IncidentLog) AddNote(id, note string) error {
	il.mu.Lock()
	defer il.mu.Unlock()
	inc, ok := il.incidents[id]
	if !ok {
		return fmt.Errorf("guardrails.IncidentLog: incident %q not found", id)
	}
	inc.Notes = append(inc.Notes, note)
	inc.UpdatedAt = time.Now().UTC()
	return il.save()
}

// Get returns a copy of the incident or an error if not found.
func (il *IncidentLog) Get(id string) (Incident, error) {
	il.mu.Lock()
	defer il.mu.Unlock()
	inc, ok := il.incidents[id]
	if !ok {
		return Incident{}, fmt.Errorf("guardrails.IncidentLog: incident %q not found", id)
	}
	return *inc, nil
}

// List returns all incidents sorted by CreatedAt descending.
func (il *IncidentLog) List(statusFilter IncidentStatus) []Incident {
	il.mu.Lock()
	defer il.mu.Unlock()
	var out []Incident
	for _, inc := range il.incidents {
		if statusFilter != "" && inc.Status != statusFilter {
			continue
		}
		out = append(out, *inc)
	}
	// Sort descending by creation time.
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].CreatedAt.After(out[i].CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (il *IncidentLog) load() error {
	p := filepath.Join(il.dir, incidentFile)
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("guardrails: load incidents: %w", err)
	}
	var list []Incident
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("guardrails: decode incidents: %w", err)
	}
	for i := range list {
		il.incidents[list[i].ID] = &list[i]
	}
	return nil
}

func (il *IncidentLog) save() error {
	list := make([]Incident, 0, len(il.incidents))
	for _, inc := range il.incidents {
		list = append(list, *inc)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("guardrails: encode incidents: %w", err)
	}
	p := filepath.Join(il.dir, incidentFile)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("guardrails: write incidents: %w", err)
	}
	return os.Rename(tmp, p)
}

// ─── OverrideController (L18) ────────────────────────────────────────────────

// OverrideState represents the current operator-controlled trading state.
type OverrideState string

const (
	// OverrideNone means no override is active; the system runs normally.
	OverrideNone OverrideState = "none"
	// OverridePause means new order entry is paused; existing positions remain.
	OverridePause OverrideState = "pause"
	// OverrideHalt means all activity is halted; liquidation may follow.
	OverrideHalt OverrideState = "halt"
)

// OverrideController lets operators manually pause or halt trading.
// All checks are non-blocking and safe for concurrent use.
type OverrideController struct {
	mu     sync.RWMutex
	state  OverrideState
	reason string
	since  time.Time
}

// NewOverrideController creates a controller in the OverrideNone state.
func NewOverrideController() *OverrideController {
	return &OverrideController{state: OverrideNone}
}

// Pause sets the override to OverridePause.
func (c *OverrideController) Pause(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = OverridePause
	c.reason = reason
	c.since = time.Now().UTC()
	log.Printf("[guardrail] override=pause reason=%q", reason)
}

// Halt sets the override to OverrideHalt (stronger than Pause).
func (c *OverrideController) Halt(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = OverrideHalt
	c.reason = reason
	c.since = time.Now().UTC()
	log.Printf("[guardrail] override=halt reason=%q", reason)
}

// Resume clears any active override.
func (c *OverrideController) Resume(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.state
	c.state = OverrideNone
	c.reason = ""
	c.since = time.Time{}
	log.Printf("[guardrail] override cleared (was %s) reason=%q", prev, reason)
}

// State returns the current override state and reason.
func (c *OverrideController) State() (OverrideState, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state, c.reason
}

// AllowEntry returns true if new order entry is permitted.
func (c *OverrideController) AllowEntry() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == OverrideNone
}

// AllowAnyActivity returns true if any trading activity is permitted.
func (c *OverrideController) AllowAnyActivity() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state != OverrideHalt
}

// Since returns when the current override was set (zero if none active).
func (c *OverrideController) Since() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.since
}

// ─── Integrated guard check ───────────────────────────────────────────────────

// GuardReport is a summary of the current system safety state.
type GuardReport struct {
	Override    OverrideState
	OverrideReason string
	IsHalted    bool
	FailStreak  int
	ProbeStates map[string]CheckResult
	OpenIncidents int
}

// BuildReport assembles a GuardReport from the monitor, controller, and log.
func BuildReport(m *HealthMonitor, c *OverrideController, il *IncidentLog) GuardReport {
	state, reason := c.State()
	m.mu.RLock()
	streak := m.failStreak
	halted := m.halted
	m.mu.RUnlock()

	open := len(il.List(IncidentOpen))

	return GuardReport{
		Override:       state,
		OverrideReason: reason,
		IsHalted:       halted,
		FailStreak:     streak,
		ProbeStates:    m.Latest(),
		OpenIncidents:  open,
	}
}

// TradingAllowed returns true when all guardrails permit new entry.
func TradingAllowed(m *HealthMonitor, c *OverrideController) bool {
	return !m.IsHalted() && c.AllowEntry()
}

// ─── stringer helpers ─────────────────────────────────────────────────────────

func (r GuardReport) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "override=%s halted=%v failStreak=%d openIncidents=%d probes=[",
		r.Override, r.IsHalted, r.FailStreak, r.OpenIncidents)
	first := true
	for name, cr := range r.ProbeStates {
		if !first {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%s:%s", name, cr.Status)
		first = false
	}
	sb.WriteString("]")
	return sb.String()
}
