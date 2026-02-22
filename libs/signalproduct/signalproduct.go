// Package signalproduct provides a clean, read-only export surface for trading
// signals produced by the strategy layer.  It decouples downstream consumers
// (paper-trading UI, analytics, external subscribers) from internal signal
// representation details.
//
// # Overview
//
// The layer has three concerns:
//
//  1. Schema: a canonical, version-pinned [SignalProduct] (the "product"
//     published to consumers).
//
//  2. Publisher: an in-process fan-out bus ([Publisher]) that components
//     subscribe to.  Note the package is deliberately database-free — the
//     caller decides whether to persist before or after publishing.
//
//  3. Filter: a composable predicate DSL ([Filter]) so subscribers can
//     self-select the signal slice they care about without the publisher
//     needing to know about routing rules.
package signalproduct

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ─── Schema ──────────────────────────────────────────────────────────────────

// Direction is the intended trade direction.
type Direction string

const (
	DirectionBuy  Direction = "buy"
	DirectionSell Direction = "sell"
	DirectionFlat Direction = "flat" // close / reduce only
)

// Strength captures the qualitative strength of a signal.
type Strength string

const (
	StrengthWeak     Strength = "weak"
	StrengthModerate Strength = "moderate"
	StrengthStrong   Strength = "strong"
)

// SignalProduct is the canonical export record for a trading signal.
// Fields are kept flat and JSON-serialisable so any downstream consumer
// can decode the struct without depending on internal packages.
type SignalProduct struct {
	// Identity.
	ID         string    `json:"id"`          // stable unique ID (e.g. UUID or strategy+ts hash)
	Version    string    `json:"version"`     // schema version, semver-ish, e.g. "1.0"
	ProducedAt time.Time `json:"produced_at"` // wall-clock production time (UTC)

	// Signal core.
	Symbol     string    `json:"symbol"`
	Direction  Direction `json:"direction"`
	Strength   Strength  `json:"strength"`
	Confidence float64   `json:"confidence"` // [0,1]

	// Price targets (optional; zero means not set).
	EntryPrice float64 `json:"entry_price,omitempty"`
	StopLoss   float64 `json:"stop_loss,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`

	// Attribution.
	StrategyID   string `json:"strategy_id"`
	ArtifactID   string `json:"artifact_id,omitempty"`
	ExperimentID string `json:"experiment_id,omitempty"`

	// Lifecycle.
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // nil = no expiry
	Cancelled bool       `json:"cancelled,omitempty"`

	// Opaque metadata — consumers should not key logic on these.
	Meta map[string]any `json:"meta,omitempty"`
}

// IsExpired reports whether the signal has passed its expiry time.
// Signals without an expiry time are never expired.
func (s *SignalProduct) IsExpired(now time.Time) bool {
	return s.ExpiresAt != nil && now.After(*s.ExpiresAt)
}

// Validate performs basic sanity checks.
func (s *SignalProduct) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("signalproduct: ID is required")
	}
	if s.Symbol == "" {
		return fmt.Errorf("signalproduct: Symbol is required")
	}
	switch s.Direction {
	case DirectionBuy, DirectionSell, DirectionFlat:
	default:
		return fmt.Errorf("signalproduct: unknown Direction %q", s.Direction)
	}
	switch s.Strength {
	case StrengthWeak, StrengthModerate, StrengthStrong:
	default:
		return fmt.Errorf("signalproduct: unknown Strength %q", s.Strength)
	}
	if s.Confidence < 0 || s.Confidence > 1 {
		return fmt.Errorf("signalproduct: Confidence must be in [0,1], got %f", s.Confidence)
	}
	if s.StrategyID == "" {
		return fmt.Errorf("signalproduct: StrategyID is required")
	}
	return nil
}

// ─── Filter ──────────────────────────────────────────────────────────────────

// Filter is a predicate that decides whether a subscriber should receive a
// given [SignalProduct].  Compose multiple filters with [AllOf] / [AnyOf].
type Filter func(sp *SignalProduct) bool

// NoFilter accepts every signal.
func NoFilter() Filter { return func(_ *SignalProduct) bool { return true } }

// BySymbol accepts signals whose symbol matches any of the given values.
func BySymbol(symbols ...string) Filter {
	set := make(map[string]struct{}, len(symbols))
	for _, s := range symbols {
		set[s] = struct{}{}
	}
	return func(sp *SignalProduct) bool {
		_, ok := set[sp.Symbol]
		return ok
	}
}

// ByStrategy accepts signals from specific strategy IDs.
func ByStrategy(ids ...string) Filter {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return func(sp *SignalProduct) bool {
		_, ok := set[sp.StrategyID]
		return ok
	}
}

// ByDirection accepts signals with a specific direction.
func ByDirection(d Direction) Filter {
	return func(sp *SignalProduct) bool { return sp.Direction == d }
}

// ByMinConfidence accepts signals at or above a confidence threshold.
func ByMinConfidence(min float64) Filter {
	return func(sp *SignalProduct) bool { return sp.Confidence >= min }
}

// ByMinStrength accepts signals at or above a strength tier.
func ByMinStrength(min Strength) Filter {
	order := map[Strength]int{StrengthWeak: 0, StrengthModerate: 1, StrengthStrong: 2}
	return func(sp *SignalProduct) bool {
		return order[sp.Strength] >= order[min]
	}
}

// ExcludeExpired drops signals that are past their expiry time.
func ExcludeExpired(now func() time.Time) Filter {
	return func(sp *SignalProduct) bool { return !sp.IsExpired(now()) }
}

// ExcludeCancelled drops cancelled signals.
func ExcludeCancelled() Filter {
	return func(sp *SignalProduct) bool { return !sp.Cancelled }
}

// AllOf returns a filter that passes only when all given filters pass (AND).
func AllOf(filters ...Filter) Filter {
	return func(sp *SignalProduct) bool {
		for _, f := range filters {
			if !f(sp) {
				return false
			}
		}
		return true
	}
}

// AnyOf returns a filter that passes when at least one filter passes (OR).
func AnyOf(filters ...Filter) Filter {
	return func(sp *SignalProduct) bool {
		for _, f := range filters {
			if f(sp) {
				return true
			}
		}
		return false
	}
}

// ─── Publisher ───────────────────────────────────────────────────────────────

// Handler is called once per signal product received by a subscription.
type Handler func(ctx context.Context, sp *SignalProduct)

// Subscription represents a registered handler that will receive signals
// matching its filter.
type Subscription struct {
	id     uint64
	filter Filter
	fn     Handler
}

// Publisher is a synchronous in-process fan-out bus.  Publish is called by the
// strategy layer; Subscribe is called by downstream consumers (UI, analytics,
// risk pre-checks, etc.).
//
// Publisher is safe for concurrent use.
type Publisher struct {
	mu   sync.RWMutex
	next uint64
	subs []*Subscription

	// Optional middleware applied to every published signal before fan-out.
	middleware []func(*SignalProduct) *SignalProduct
}

// NewPublisher creates an empty publisher.
func NewPublisher() *Publisher { return &Publisher{} }

// Use registers a middleware transform applied to every signal before fan-out.
// Transforms are applied in registration order.  A transform may return nil to
// drop the signal entirely.
func (p *Publisher) Use(t func(*SignalProduct) *SignalProduct) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.middleware = append(p.middleware, t)
}

// Subscribe registers a handler with an optional filter.  Returns an
// unsubscribe function.  If filter is nil, NoFilter is used.
func (p *Publisher) Subscribe(f Filter, fn Handler) func() {
	if f == nil {
		f = NoFilter()
	}
	p.mu.Lock()
	id := p.next
	p.next++
	sub := &Subscription{id: id, filter: f, fn: fn}
	p.subs = append(p.subs, sub)
	p.mu.Unlock()

	return func() { p.unsubscribe(id) }
}

// Publish validates sp and fans it out to all matching subscribers.
// Returns the number of subscribers the signal was delivered to.
// Returns an error only on validation failure — subscriber panics are not
// caught (callers may wrap in a recover middleware if desired).
func (p *Publisher) Publish(ctx context.Context, sp *SignalProduct) (int, error) {
	if err := sp.Validate(); err != nil {
		return 0, err
	}

	// Apply middleware chain (a nil return drops the signal).
	p.mu.RLock()
	mw := append([]func(*SignalProduct) *SignalProduct(nil), p.middleware...)
	subs := append([]*Subscription(nil), p.subs...)
	p.mu.RUnlock()

	cur := sp
	for _, t := range mw {
		cur = t(cur)
		if cur == nil {
			return 0, nil
		}
	}

	delivered := 0
	for _, sub := range subs {
		if sub.filter(cur) {
			sub.fn(ctx, cur)
			delivered++
		}
	}
	return delivered, nil
}

// Len returns the current number of active subscriptions.
func (p *Publisher) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subs)
}

func (p *Publisher) unsubscribe(id uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, s := range p.subs {
		if s.id == id {
			p.subs = append(p.subs[:i], p.subs[i+1:]...)
			return
		}
	}
}

// ─── StrengthFromConfidence ───────────────────────────────────────────────────

// StrengthFromConfidence maps a numeric confidence to a qualitative [Strength].
//   - [0, 0.50) → weak
//   - [0.50, 0.75) → moderate
//   - [0.75, 1.0] → strong
func StrengthFromConfidence(c float64) Strength {
	switch {
	case c >= 0.75:
		return StrengthStrong
	case c >= 0.50:
		return StrengthModerate
	default:
		return StrengthWeak
	}
}
