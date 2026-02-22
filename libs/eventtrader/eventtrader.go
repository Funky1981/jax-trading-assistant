// Package eventtrader implements L10 + L11:
//
//   - L10: EventPhaseDetector – classifies the current moment relative to
//     scheduled economic events (Normal / PreEvent / Blackout / PostEvent).
//   - L11: EventGate – a strategy guard that blocks signal generation or
//     order entry during blackout/pre-event windows, and an EventBucket
//     that groups signals by the nearest relevant economic event.
package eventtrader

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"jax-trading-assistant/libs/calendar"
)

// ─── Phase ────────────────────────────────────────────────────────────────────

// Phase represents a trader's operational stance relative to a calendar event.
type Phase string

const (
	// PhaseNormal means no high-impact event is nearby; trade freely.
	PhaseNormal Phase = "normal"
	// PhasePreEvent means a high-impact event is within the pre-event window;
	// new entries should be avoided unless the strategy explicitly handles it.
	PhasePreEvent Phase = "pre_event"
	// PhaseBlackout means an event is imminent (within the blackout window);
	// all new order entry is blocked.
	PhaseBlackout Phase = "blackout"
	// PhasePostEvent means a high-impact event fired recently; the market may
	// still be digesting the release.
	PhasePostEvent Phase = "post_event"
)

// PhaseResult contains the detected phase plus the triggering event if any.
type PhaseResult struct {
	Phase      Phase
	Event      *calendar.EconEvent // nil for PhaseNormal
	TimeToGo   time.Duration       // positive = event in future, negative = past (post-event)
	DetectedAt time.Time
}

// ─── PhaseDetectorConfig ──────────────────────────────────────────────────────

// PhaseDetectorConfig controls the window sizes for the phase detector.
type PhaseDetectorConfig struct {
	// BlackoutBefore is how long before the event the blackout phase begins.
	// Default: 10 minutes.
	BlackoutBefore time.Duration
	// PreEventBefore is how long before the event the pre-event phase begins.
	// Must be >= BlackoutBefore. Default: 60 minutes.
	PreEventBefore time.Duration
	// PostEventAfter is how long after the event the post-event phase lasts.
	// Default: 30 minutes.
	PostEventAfter time.Duration
	// Currencies filters which currencies trigger phase changes.
	// Empty = all currencies.
	Currencies []string
	// MinImpact is the minimum event impact level to consider. Default: High.
	MinImpact calendar.Impact
}

// DefaultPhaseDetectorConfig returns conservative defaults.
func DefaultPhaseDetectorConfig() PhaseDetectorConfig {
	return PhaseDetectorConfig{
		BlackoutBefore: 10 * time.Minute,
		PreEventBefore: 60 * time.Minute,
		PostEventAfter: 30 * time.Minute,
		MinImpact:      calendar.ImpactHigh,
	}
}

// ─── PhaseDetector ────────────────────────────────────────────────────────────

// PhaseDetector classifies the current moment relative to calendar events.
// It is safe for concurrent use.
type PhaseDetector struct {
	store  *calendar.Store
	cfg    PhaseDetectorConfig
	mu     sync.Mutex
	lastAt time.Time
	cached *PhaseResult
}

// NewPhaseDetector creates a PhaseDetector backed by the given calendar store.
func NewPhaseDetector(store *calendar.Store, cfg PhaseDetectorConfig) *PhaseDetector {
	return &PhaseDetector{store: store, cfg: cfg}
}

// Detect returns the current phase for a set of currency symbols at the given
// time. The result is computed fresh on every call (no internal caching that
// crosses calls; callers may cache externally).
func (d *PhaseDetector) Detect(now time.Time, currencies ...string) PhaseResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Widen the query window to cover pre-event + post-event.
	from := now.Add(-d.cfg.PostEventAfter)
	to := now.Add(d.cfg.PreEventBefore)

	curs := d.cfg.Currencies
	if len(currencies) > 0 {
		curs = currencies
	}

	// Query all high(er)-impact events in the relevant window.
	// Store.Query accepts one currency; we fan out if needed.
	var events []calendar.EconEvent
	seen := make(map[string]bool)

	addEvents := func(cur string) {
		for _, e := range d.store.Query(from, to, "", cur, d.cfg.MinImpact) {
			if !seen[e.ID] {
				seen[e.ID] = true
				events = append(events, e)
			}
		}
	}

	if len(curs) == 0 {
		events = d.store.Query(from, to, "", "", d.cfg.MinImpact)
		for _, e := range events {
			seen[e.ID] = true
		}
	} else {
		for _, cur := range curs {
			addEvents(cur)
		}
	}

	return d.classify(now, events)
}

func (d *PhaseDetector) classify(now time.Time, events []calendar.EconEvent) PhaseResult {
	best := PhaseResult{Phase: PhaseNormal, DetectedAt: now}

	for i := range events {
		e := events[i]
		ttg := e.ScheduledAt.Sub(now) // positive = future

		var phase Phase
		switch {
		case ttg >= 0 && ttg <= d.cfg.BlackoutBefore:
			phase = PhaseBlackout
		case ttg > d.cfg.BlackoutBefore && ttg <= d.cfg.PreEventBefore:
			phase = PhasePreEvent
		case ttg < 0 && -ttg <= d.cfg.PostEventAfter:
			phase = PhasePostEvent
		default:
			continue
		}

		// Priority order: blackout > post-event > pre-event > normal.
		if phaseRank(phase) > phaseRank(best.Phase) {
			eCopy := e
			best = PhaseResult{
				Phase:      phase,
				Event:      &eCopy,
				TimeToGo:   ttg,
				DetectedAt: now,
			}
		}
	}
	return best
}

func phaseRank(p Phase) int {
	switch p {
	case PhaseBlackout:
		return 4
	case PhasePostEvent:
		return 3
	case PhasePreEvent:
		return 2
	default:
		return 0
	}
}

// ─── EventGate (L11) ──────────────────────────────────────────────────────────

// GateDecision is the verdict returned by EventGate.
type GateDecision string

const (
	// GateAllow means the strategy may generate/submit the signal.
	GateAllow GateDecision = "allow"
	// GateHold means the strategy should hold (pre-event caution).
	GateHold GateDecision = "hold"
	// GateBlock means new order entry is blocked (blackout).
	GateBlock GateDecision = "block"
)

// GateResult is the full verdict from EventGate.
type GateResult struct {
	Decision     GateDecision
	Phase        Phase
	Reason       string
	TriggerEvent *calendar.EconEvent
}

// EventGateConfig configures the gate's blocking policy.
type EventGateConfig struct {
	// BlockOnBlackout enables blocking during blackout windows.
	BlockOnBlackout bool
	// HoldOnPreEvent enables hold during pre-event windows.
	HoldOnPreEvent bool
	// AllowStrategies is a list of strategy IDs that are allowed through even
	// during pre-event/blackout windows (event-specialist strategies).
	AllowStrategies []string
}

// DefaultEventGateConfig returns sensible blocking defaults.
func DefaultEventGateConfig() EventGateConfig {
	return EventGateConfig{
		BlockOnBlackout: true,
		HoldOnPreEvent:  true,
	}
}

// EventGate wraps a PhaseDetector and returns allow/hold/block decisions.
type EventGate struct {
	detector *PhaseDetector
	cfg      EventGateConfig
}

// NewEventGate creates an EventGate.
func NewEventGate(detector *PhaseDetector, cfg EventGateConfig) *EventGate {
	return &EventGate{detector: detector, cfg: cfg}
}

// Check returns a GateResult for the given (strategyID, currencies, time).
func (g *EventGate) Check(strategyID string, now time.Time, currencies ...string) GateResult {
	// Specialist strategies bypass blocking.
	for _, id := range g.cfg.AllowStrategies {
		if id == strategyID {
			return GateResult{Decision: GateAllow, Phase: PhaseNormal,
				Reason: "allow-listed strategy"}
		}
	}

	pr := g.detector.Detect(now, currencies...)

	switch pr.Phase {
	case PhaseBlackout:
		if g.cfg.BlockOnBlackout {
			reason := "blackout window"
			if pr.Event != nil {
				reason = fmt.Sprintf("blackout: %s in %s",
					pr.Event.Title, pr.TimeToGo.Round(time.Second))
			}
			log.Printf("[eventgate] strategy=%s decision=block phase=blackout reason=%q",
				strategyID, reason)
			return GateResult{Decision: GateBlock, Phase: PhaseBlackout,
				Reason: reason, TriggerEvent: pr.Event}
		}
	case PhasePreEvent:
		if g.cfg.HoldOnPreEvent {
			reason := "pre-event window"
			if pr.Event != nil {
				reason = fmt.Sprintf("pre-event: %s in %s",
					pr.Event.Title, pr.TimeToGo.Round(time.Minute))
			}
			log.Printf("[eventgate] strategy=%s decision=hold phase=pre_event reason=%q",
				strategyID, reason)
			return GateResult{Decision: GateHold, Phase: PhasePreEvent,
				Reason: reason, TriggerEvent: pr.Event}
		}
	}

	return GateResult{Decision: GateAllow, Phase: pr.Phase, Reason: "clear"}
}

// ─── EventBucket (L11) ───────────────────────────────────────────────────────

// Signal is a lightweight representation of a trade signal for bucketing.
// Callers attach their own full signal and embed it via UserData.
type Signal struct {
	ID          string
	StrategyID  string
	Symbol      string
	Currency    string
	GeneratedAt time.Time
	UserData    interface{}
}

// Bucket groups signals that share a triggering calendar event.
type Bucket struct {
	// EventID is the EventID of the anchoring calendar event, or "baseline"
	// when no event is nearby.
	EventID string
	Event   *calendar.EconEvent
	Phase   Phase
	Signals []Signal
}

// EventBucketizer assigns incoming signals to event buckets based on proximity
// to the nearest calendar event at the time of signal generation.
type EventBucketizer struct {
	detector *PhaseDetector
	mu       sync.Mutex
	buckets  map[string]*Bucket // keyed by EventID or "baseline"
}

// NewEventBucketizer creates an EventBucketizer.
func NewEventBucketizer(detector *PhaseDetector) *EventBucketizer {
	return &EventBucketizer{
		detector: detector,
		buckets:  make(map[string]*Bucket),
	}
}

// Classify assigns signal s to an event bucket and records it.
func (b *EventBucketizer) Classify(_ context.Context, s Signal) *Bucket {
	pr := b.detector.Detect(s.GeneratedAt, s.Currency)

	var key string
	var eventCopy *calendar.EconEvent
	if pr.Event != nil {
		key = pr.Event.ID
		eCopy := *pr.Event
		eventCopy = &eCopy
	} else {
		key = "baseline"
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	bk, ok := b.buckets[key]
	if !ok {
		bk = &Bucket{EventID: key, Event: eventCopy, Phase: pr.Phase}
		b.buckets[key] = bk
	}
	bk.Signals = append(bk.Signals, s)
	return bk
}

// Buckets returns a snapshot of all current buckets.
func (b *EventBucketizer) Buckets() []*Bucket {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*Bucket, 0, len(b.buckets))
	for _, bk := range b.buckets {
		out = append(out, bk)
	}
	return out
}

// Reset clears all buckets.
func (b *EventBucketizer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buckets = make(map[string]*Bucket)
}
