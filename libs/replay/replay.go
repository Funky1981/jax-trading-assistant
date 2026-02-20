// Package replay implements L12 + L13:
//
//   - L12: an append-only DecisionTrace store that records all signal/order
//     decisions with enough context to replay them deterministically.
//   - L13: a pipeline Simulator with a SimBroker that replays a trace against
//     historical candle data, producing fills and P&L.
package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// ─── TraceEntry (L12) ─────────────────────────────────────────────────────────

// Decision is the outcome of a signal-to-order evaluation.
type Decision string

const (
	DecisionEmit   Decision = "emit"   // signal accepted, order placed
	DecisionHold   Decision = "hold"   // pre-event gate: signal held
	DecisionBlock  Decision = "block"  // blackout gate: signal blocked
	DecisionReject Decision = "reject" // risk policy rejection
	DecisionCancel Decision = "cancel" // order cancelled post-placement
)

// TraceEntry is an immutable record of one trader decision.
// All fields except Notes are mandatory for deterministic replay.
type TraceEntry struct {
	// Sequence is the 1-based, monotonically increasing entry number.
	Sequence uint64 `json:"seq"`
	// RecordedAt is the wall-clock time the entry was written.
	RecordedAt time.Time `json:"recorded_at"`
	// StrategyID identifies the originating strategy.
	StrategyID string `json:"strategy_id"`
	// Symbol is the traded instrument (e.g. "AAPL", "EURUSD").
	Symbol string `json:"symbol"`
	// EventPhase is the phase reported by PhaseDetector at decision time.
	EventPhase string `json:"event_phase,omitempty"`
	// SignalPrice is the market price at signal generation.
	SignalPrice float64 `json:"signal_price"`
	// StopLoss is the SL level that informed risk sizing.
	StopLoss float64 `json:"stop_loss"`
	// PositionSize is the computed lot/share size.
	PositionSize float64 `json:"position_size"`
	// Decision is the outcome.
	Decision Decision `json:"decision"`
	// Reason is a short human-readable explanation.
	Reason string `json:"reason,omitempty"`
	// OrderID is set when Decision=emit.
	OrderID string `json:"order_id,omitempty"`
	// PnL is set when the associated order has been closed (may be 0 initially).
	PnL float64 `json:"pnl,omitempty"`
	// Notes is optional free-form debugging text.
	Notes string `json:"notes,omitempty"`
}

// ─── TraceStore (L12) ─────────────────────────────────────────────────────────

// TraceStore is an append-only, JSON-line-backed decision trace.
// All writes are atomic per-entry. The store is safe for concurrent use.
type TraceStore struct {
	mu   sync.Mutex
	path string
	seq  uint64
}

const traceFile = "decisions.jsonl"

// OpenTraceStore opens (or creates) a trace store in dir.
func OpenTraceStore(dir string) (*TraceStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("replay.OpenTraceStore: mkdir: %w", err)
	}
	ts := &TraceStore{path: filepath.Join(dir, traceFile)}
	// Count existing lines to initialise sequence.
	entries, err := ts.ReadAll()
	if err != nil {
		return nil, err
	}
	ts.seq = uint64(len(entries))
	return ts, nil
}

// Append records a decision. Sequence is auto-assigned; RecordedAt is set to now.
func (ts *TraceStore) Append(entry TraceEntry) (TraceEntry, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.seq++
	entry.Sequence = ts.seq
	entry.RecordedAt = time.Now().UTC()

	data, err := json.Marshal(entry)
	if err != nil {
		ts.seq-- // rollback
		return TraceEntry{}, fmt.Errorf("replay.TraceStore.Append: marshal: %w", err)
	}

	f, err := os.OpenFile(ts.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		ts.seq--
		return TraceEntry{}, fmt.Errorf("replay.TraceStore.Append: open: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
		ts.seq--
		return TraceEntry{}, fmt.Errorf("replay.TraceStore.Append: write: %w", err)
	}
	return entry, nil
}

// ReadAll reads all entries from the store in append order.
func (ts *TraceStore) ReadAll() ([]TraceEntry, error) {
	data, err := os.ReadFile(ts.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("replay.TraceStore.ReadAll: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	entries := make([]TraceEntry, 0, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e TraceEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("replay.TraceStore.ReadAll: line %d: %w", i+1, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Filter returns trace entries matching all non-zero predicates.
func (ts *TraceStore) Filter(strategyID, symbol string, decision Decision) ([]TraceEntry, error) {
	all, err := ts.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []TraceEntry
	for _, e := range all {
		if strategyID != "" && e.StrategyID != strategyID {
			continue
		}
		if symbol != "" && e.Symbol != symbol {
			continue
		}
		if decision != "" && e.Decision != decision {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// ─── Candle ───────────────────────────────────────────────────────────────────

// Candle is an OHLCV bar used for simulation.
type Candle struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// ─── SimBroker (L13) ─────────────────────────────────────────────────────────

// OrderSide is buy or sell.
type OrderSide string

const (
	SideBuy  OrderSide = "buy"
	SideSell OrderSide = "sell"
)

// OrderType for simulation.
type OrderType string

const (
	OrderMarket OrderType = "market"
	OrderLimit  OrderType = "limit"
	OrderStop   OrderType = "stop"
)

// SimOrder is an order submitted to the SimBroker.
type SimOrder struct {
	ID           string
	StrategyID   string
	Symbol       string
	Side         OrderSide
	Type         OrderType
	Quantity     float64
	LimitPrice   float64 // for limit orders
	StopPrice    float64 // for stop/stop-limit orders
	SubmittedAt  time.Time
}

// Fill represents a completed execution.
type Fill struct {
	OrderID     string
	Symbol      string
	Side        OrderSide
	Quantity    float64
	FillPrice   float64
	FilledAt    time.Time
	Slippage    float64 // in price units
	Commission  float64
}

// Position tracks an open position in the SimBroker.
type Position struct {
	Symbol       string
	Side         OrderSide
	Quantity     float64
	EntryPrice   float64
	OpenedAt     time.Time
	RelatedOrderID string
}

// SimBrokerConfig controls broker simulation parameters.
type SimBrokerConfig struct {
	// SlippageBps is slippage in basis points applied to market orders.
	SlippageBps float64
	// CommissionPerShare is a flat per-share/lot commission.
	CommissionPerShare float64
	// FillOnNextOpen: when true market orders fill at the next candle's open.
	// When false they fill at current close.
	FillOnNextOpen bool
}

// DefaultSimBrokerConfig returns a default cost structure.
func DefaultSimBrokerConfig() SimBrokerConfig {
	return SimBrokerConfig{
		SlippageBps:        5,
		CommissionPerShare: 0.005,
		FillOnNextOpen:     true,
	}
}

// SimBroker is a deterministic simulated broker.
type SimBroker struct {
	cfg       SimBrokerConfig
	mu        sync.Mutex
	pending   map[string]SimOrder // awaiting next candle
	positions map[string]*Position
	fills     []Fill
	equity    float64
	cash      float64
}

// NewSimBroker creates a SimBroker with initialCapital.
func NewSimBroker(cfg SimBrokerConfig, initialCapital float64) *SimBroker {
	return &SimBroker{
		cfg:       cfg,
		pending:   make(map[string]SimOrder),
		positions: make(map[string]*Position),
		cash:      initialCapital,
		equity:    initialCapital,
	}
}

// SubmitOrder enqueues an order for the next candle's processing.
func (b *SimBroker) SubmitOrder(o SimOrder) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending[o.ID] = o
}

// ProcessCandle applies all pending orders against the candle's prices and
// marks to market any open positions.
func (b *SimBroker) ProcessCandle(c Candle) []Fill {
	b.mu.Lock()
	defer b.mu.Unlock()

	var newFills []Fill

	for id, o := range b.pending {
		// If the candle has no symbol set, it matches all pending orders
		// (useful for single-symbol simulations).
		if c.Symbol != "" && o.Symbol != c.Symbol {
			continue
		}
		fill, ok := b.tryFill(o, c)
		if !ok {
			// Limit/stop not triggered yet.
			continue
		}
		b.fills = append(b.fills, fill)
		newFills = append(newFills, fill)
		delete(b.pending, id)

		// Update cash and position.
		cost := fill.FillPrice * fill.Quantity
		if fill.Side == SideBuy {
			b.cash -= cost + fill.Commission
			b.positions[o.ID] = &Position{
				Symbol:         o.Symbol,
				Side:           SideBuy,
				Quantity:       fill.Quantity,
				EntryPrice:     fill.FillPrice,
				OpenedAt:       fill.FilledAt,
				RelatedOrderID: o.ID,
			}
		} else {
			b.cash += cost - fill.Commission
			// Remove matching long position (simplified: latest LIFO).
			for k, pos := range b.positions {
				if pos.Symbol == o.Symbol && pos.Side == SideBuy {
					delete(b.positions, k)
					break
				}
			}
		}
	}

	// Mark positions to current close.
	b.equity = b.cash
	for _, pos := range b.positions {
		if c.Symbol == "" || pos.Symbol == c.Symbol {
			b.equity += pos.Quantity * c.Close
		} else {
			b.equity += pos.Quantity * pos.EntryPrice // flat for other symbols
		}
	}

	return newFills
}

func (b *SimBroker) tryFill(o SimOrder, c Candle) (Fill, bool) {
	slipBps := b.cfg.SlippageBps / 10_000

	var fillPrice float64
	switch o.Type {
	case OrderMarket:
		if b.cfg.FillOnNextOpen {
			fillPrice = c.Open
		} else {
			fillPrice = c.Close
		}
		if o.Side == SideBuy {
			fillPrice *= (1 + slipBps)
		} else {
			fillPrice *= (1 - slipBps)
		}

	case OrderLimit:
		if o.Side == SideBuy && c.Low <= o.LimitPrice {
			fillPrice = math.Min(o.LimitPrice, c.Open)
		} else if o.Side == SideSell && c.High >= o.LimitPrice {
			fillPrice = math.Max(o.LimitPrice, c.Open)
		} else {
			return Fill{}, false
		}

	case OrderStop:
		if o.Side == SideBuy && c.High >= o.StopPrice {
			fillPrice = math.Max(o.StopPrice, c.Open)
		} else if o.Side == SideSell && c.Low <= o.StopPrice {
			fillPrice = math.Min(o.StopPrice, c.Open)
		} else {
			return Fill{}, false
		}
	}

	commission := o.Quantity * b.cfg.CommissionPerShare
	slippage := math.Abs(fillPrice-c.Open) * o.Quantity

	return Fill{
		OrderID:   o.ID,
		Symbol:    o.Symbol,
		Side:      o.Side,
		Quantity:  o.Quantity,
		FillPrice: fillPrice,
		FilledAt:  c.Timestamp,
		Slippage:  slippage,
		Commission: commission,
	}, true
}

// Equity returns the current portfolio equity.
func (b *SimBroker) Equity() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.equity
}

// Fills returns all fills recorded so far.
func (b *SimBroker) Fills() []Fill {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Fill, len(b.fills))
	copy(out, b.fills)
	return out
}

// OpenPositions returns the current open positions.
func (b *SimBroker) OpenPositions() []*Position {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*Position, 0, len(b.positions))
	for _, p := range b.positions {
		out = append(out, p)
	}
	return out
}

// ─── Simulator (L13) ─────────────────────────────────────────────────────────

// PlaybackResult summarises a simulation run.
type PlaybackResult struct {
	Fills         []Fill
	FinalEquity   float64
	InitialEquity float64
	TotalReturn   float64
	TotalTrades   int
	WinTrades     int
	LossTrades    int
	WinRate       float64
	// TotalSlippage is the sum of all fill slippages in price units × qty.
	TotalSlippage  float64
	TotalCommission float64
}

// PlaybackSignal is a minimal representation of a signal for the simulator.
type PlaybackSignal struct {
	OrderID    string
	StrategyID string
	Symbol     string
	Side       OrderSide
	Quantity   float64
	StopLoss   float64 // used as stop price for protective stop placement
	EventAt    time.Time
}

// Simulator feeds candles through a SimBroker and optionally records to a TraceStore.
type Simulator struct {
	broker *SimBroker
	trace  *TraceStore // may be nil
}

// NewSimulator creates a Simulator.
// trace may be nil — in that case no trace entries are written.
func NewSimulator(broker *SimBroker, trace *TraceStore) *Simulator {
	return &Simulator{broker: broker, trace: trace}
}

// Run feeds a sequence of candles through the broker, placing signals as market
// orders at their EventAt timestamps and processing them on the next candle.
// Candles must be time-sorted ascending for a single symbol.
func (s *Simulator) Run(_ context.Context, signals []PlaybackSignal, candles []Candle) (PlaybackResult, error) {
	if len(candles) == 0 {
		return PlaybackResult{}, fmt.Errorf("replay.Simulator.Run: no candles")
	}

	initEquity := s.broker.Equity()

	// Sort candles ascending by timestamp.
	slices.SortFunc(candles, func(a, b Candle) int {
		return a.Timestamp.Compare(b.Timestamp)
	})

	// Index signals by EventAt for quick lookup.
	// A signal is submitted when we see the candle whose timestamp >= signal.EventAt.
	sigsByTime := make(map[time.Time][]PlaybackSignal)
	for _, sig := range signals {
		sigsByTime[sig.EventAt] = append(sigsByTime[sig.EventAt], sig)
	}

	// Build ordered signal timestamps.
	var sigTimes []time.Time
	for t := range sigsByTime {
		sigTimes = append(sigTimes, t)
	}
	slices.SortFunc(sigTimes, func(a, b time.Time) int { return a.Compare(b) })
	sigIdx := 0

	for _, c := range candles {
		// Submit any signals whose time has arrived.
		for sigIdx < len(sigTimes) && !c.Timestamp.Before(sigTimes[sigIdx]) {
			for _, sig := range sigsByTime[sigTimes[sigIdx]] {
				order := SimOrder{
					ID:          sig.OrderID,
					StrategyID:  sig.StrategyID,
					Symbol:      sig.Symbol,
					Side:        sig.Side,
					Type:        OrderMarket,
					Quantity:    sig.Quantity,
					StopPrice:   sig.StopLoss,
					SubmittedAt: c.Timestamp,
				}
				s.broker.SubmitOrder(order)

				if s.trace != nil {
					_, _ = s.trace.Append(TraceEntry{
						StrategyID:   sig.StrategyID,
						Symbol:       sig.Symbol,
						SignalPrice:   c.Open,
						StopLoss:     sig.StopLoss,
						PositionSize: sig.Quantity,
						Decision:     DecisionEmit,
						OrderID:      sig.OrderID,
						Reason:       "simulator playback",
					})
				}
			}
			sigIdx++
		}
		s.broker.ProcessCandle(c)
	}

	fills := s.broker.Fills()
	result := buildResult(fills, initEquity, s.broker.Equity())
	return result, nil
}

func buildResult(fills []Fill, initEquity, finalEquity float64) PlaybackResult {
	r := PlaybackResult{
		Fills:         fills,
		InitialEquity: initEquity,
		FinalEquity:   finalEquity,
		TotalTrades:   len(fills),
	}
	if initEquity > 0 {
		r.TotalReturn = (finalEquity - initEquity) / initEquity
	}

	// Pair fills as buy→sell for P&L win/loss counting (simplified).
	// For each sell fill, find the most recent preceding buy fill.
	openBuys := make(map[string]float64) // symbol → buy price
	for _, f := range fills {
		switch f.Side {
		case SideBuy:
			openBuys[f.Symbol] = f.FillPrice
		case SideSell:
			if buyPrice, ok := openBuys[f.Symbol]; ok {
				pnl := (f.FillPrice - buyPrice) * f.Quantity
				if pnl >= 0 {
					r.WinTrades++
				} else {
					r.LossTrades++
				}
				delete(openBuys, f.Symbol)
			}
		}
		r.TotalSlippage += f.Slippage
		r.TotalCommission += f.Commission
	}
	if r.TotalTrades > 0 && r.WinTrades+r.LossTrades > 0 {
		r.WinRate = float64(r.WinTrades) / float64(r.WinTrades+r.LossTrades)
	}
	return r
}
