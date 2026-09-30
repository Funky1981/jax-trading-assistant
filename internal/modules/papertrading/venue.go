package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type PaperFill struct {
	FillID           string            `json:"fill_id"`
	ContractVersion  string            `json:"contract_version"`
	OrderID          string            `json:"order_id"`
	PaperIntentID    string            `json:"paper_intent_id"`
	WorkflowID       string            `json:"workflow_id"`
	InstrumentID     string            `json:"instrument_id"`
	Direction        string            `json:"direction"`
	Quantity         float64           `json:"quantity"`
	Price            float64           `json:"price"`
	Costs            CostBreakdown     `json:"costs"`
	FilledAt         time.Time         `json:"filled_at"`
	TickID           string            `json:"tick_id"`
	MarketProvenance *MarketProvenance `json:"market_provenance,omitempty"`
}

func (fill PaperFill) Validate() error {
	if fill.ContractVersion != FillContractVersion || fill.OrderID == "" || fill.PaperIntentID == "" || fill.WorkflowID == "" || fill.InstrumentID == "" || (fill.Direction != "LONG" && fill.Direction != "SHORT") || !finitePositive(fill.Quantity) || !finitePositive(fill.Price) || fill.TickID == "" || fill.FilledAt.IsZero() || fill.FilledAt.Location() != time.UTC || fill.Costs.ModelID == "" || fill.FillID != fillIdentity(fill) {
		return ErrInvalidArtifact
	}
	if fill.MarketProvenance != nil && (!fill.MarketProvenance.StrictTemporal || fill.MarketProvenance.Provider == "" || fill.MarketProvenance.QuoteProviderAt.IsZero() || fill.MarketProvenance.TradeProviderAt.IsZero() || fill.MarketProvenance.ReceivedAt.IsZero() || fill.MarketProvenance.AsOf.IsZero() || fill.MarketProvenance.AvailableAt.IsZero() || fill.MarketProvenance.AvailableAt.After(fill.FilledAt)) {
		return ErrInvalidArtifact
	}
	return nil
}

type VenueSnapshot struct {
	Contract       CapabilityContract    `json:"contract"`
	CostModel      CostModel             `json:"cost_model"`
	Orders         map[string]PaperOrder `json:"orders"`
	Fills          map[string]PaperFill  `json:"fills"`
	ProcessedTicks map[string]string     `json:"processed_ticks"`
	CommandKeys    map[string]string     `json:"command_keys"`
	ConsumedByTick map[string]float64    `json:"consumed_by_tick,omitempty"`
	LastTick       time.Time             `json:"last_tick"`
}

type PaperVenue struct {
	mu             sync.Mutex
	contract       CapabilityContract
	costModel      CostModel
	orders         map[string]PaperOrder
	fills          map[string]PaperFill
	intentOrders   map[string]string
	processedTicks map[string]string
	commandKeys    map[string]string
	consumedByTick map[string]float64
	lastTick       time.Time
}

func NewPaperVenue(contract CapabilityContract, costModel CostModel) (*PaperVenue, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}
	if err := costModel.Validate(); err != nil {
		return nil, err
	}
	return &PaperVenue{contract: contract, costModel: costModel, orders: map[string]PaperOrder{}, fills: map[string]PaperFill{}, intentOrders: map[string]string{}, processedTicks: map[string]string{}, commandKeys: map[string]string{}, consumedByTick: map[string]float64{}}, nil
}

func RestorePaperVenue(snapshot VenueSnapshot) (*PaperVenue, error) {
	venue, err := NewPaperVenue(snapshot.Contract, snapshot.CostModel)
	if err != nil {
		return nil, err
	}
	if snapshot.Orders == nil || snapshot.Fills == nil || snapshot.ProcessedTicks == nil || snapshot.CommandKeys == nil {
		return nil, ErrInvalidArtifact
	}
	for id, order := range snapshot.Orders {
		if id != order.OrderID || order.Validate() != nil || order.CostModelID != snapshot.CostModel.ModelID || !order.ActivatesAt.Equal(order.CreatedAt.Add(snapshot.CostModel.Latency)) {
			return nil, ErrInvalidArtifact
		}
		if prior, exists := venue.intentOrders[order.PaperIntentID]; exists && prior != id {
			return nil, ErrIdempotencyConflict
		}
		venue.orders[id] = order
		venue.intentOrders[order.PaperIntentID] = id
	}
	for id, fill := range snapshot.Fills {
		if id != fill.FillID || fill.Validate() != nil {
			return nil, ErrInvalidArtifact
		}
		order, exists := venue.orders[fill.OrderID]
		if !exists || fill.Quantity > order.Quantity || fill.FilledAt.Before(order.ActivatesAt) {
			return nil, ErrInvalidArtifact
		}
		venue.fills[id] = fill
		venue.consumedByTick[fill.TickID] += fill.Quantity
	}
	filledByOrder := make(map[string]float64)
	for _, fill := range snapshot.Fills {
		filledByOrder[fill.OrderID] += fill.Quantity
	}
	for id, order := range snapshot.Orders {
		if filledByOrder[id] > order.Quantity || math.Abs(filledByOrder[id]-order.FilledQuantity) > quantityTolerance {
			return nil, ErrInvalidArtifact
		}
	}
	for key, fingerprint := range snapshot.ProcessedTicks {
		if key == "" || fingerprint == "" {
			return nil, ErrInvalidArtifact
		}
		venue.processedTicks[key] = fingerprint
	}
	for key, fingerprint := range snapshot.CommandKeys {
		if key == "" || fingerprint == "" {
			return nil, ErrInvalidArtifact
		}
		venue.commandKeys[key] = fingerprint
	}
	for tickID, quantity := range snapshot.ConsumedByTick {
		if tickID == "" || !finiteNonNegative(quantity) {
			return nil, ErrInvalidArtifact
		}
		if quantity > venue.consumedByTick[tickID] {
			venue.consumedByTick[tickID] = quantity
		}
	}
	if !snapshot.LastTick.IsZero() && snapshot.LastTick.Location() != time.UTC {
		return nil, ErrInvalidArtifact
	}
	venue.lastTick = snapshot.LastTick
	return venue, nil
}

func (venue *PaperVenue) Submit(request CreateOrderRequest) (PaperOrder, error) {
	if request.Venue.VenueID != venue.contract.VenueID || request.CostModel != venue.costModel {
		return PaperOrder{}, ErrInvalidContract
	}
	if request.Venue.Environment != EnvironmentPaper {
		return PaperOrder{}, ErrLiveExecutionDisabled
	}
	order, err := CreatePaperOrder(request)
	if err != nil {
		return PaperOrder{}, err
	}
	venue.mu.Lock()
	defer venue.mu.Unlock()
	if existingID, ok := venue.intentOrders[order.PaperIntentID]; ok {
		if existingID != order.OrderID {
			return PaperOrder{}, ErrIdempotencyConflict
		}
		return venue.orders[existingID], nil
	}
	venue.orders[order.OrderID] = order
	venue.intentOrders[order.PaperIntentID] = order.OrderID
	return order, nil
}

// OrderForIntent returns the durable-in-process order identity for safe runtime
// retries after restoration. It does not mutate order state.
func (venue *PaperVenue) OrderForIntent(paperIntentID string) (PaperOrder, bool) {
	venue.mu.Lock()
	defer venue.mu.Unlock()
	id, ok := venue.intentOrders[paperIntentID]
	if !ok {
		return PaperOrder{}, false
	}
	order, ok := venue.orders[id]
	return order, ok
}

func (venue *PaperVenue) ProcessTick(tick MarketTick) ([]PaperFill, error) {
	return venue.ProcessTickWithSafety(tick, false)
}

func (venue *PaperVenue) ValidateMarketTick(tick MarketTick) error {
	if err := tick.Validate(venue.contract.MaxQuoteAge); err != nil {
		return err
	}
	if tick.Session == SessionUnknown {
		return ErrMarketUnknown
	}
	return nil
}

func (venue *PaperVenue) ProcessTickWithSafety(tick MarketTick, breakerTripped bool) ([]PaperFill, error) {
	return venue.processTick(tick, breakerTripped, "")
}

// ProcessOrderTickWithSafety limits one observation to one durable order. This
// lets the account-scoped runtime persist that order's fill and ledger in one
// transaction without mutating unrelated pending orders in memory.
func (venue *PaperVenue) ProcessOrderTickWithSafety(tick MarketTick, orderID string, breakerTripped bool) ([]PaperFill, error) {
	if orderID == "" {
		return nil, ErrOrderNotFound
	}
	return venue.processTick(tick, breakerTripped, orderID)
}

func (venue *PaperVenue) processTick(tick MarketTick, breakerTripped bool, onlyOrderID string) ([]PaperFill, error) {
	if breakerTripped {
		return nil, ErrPaperExecutionBlocked
	}
	if err := tick.Validate(venue.contract.MaxQuoteAge); err != nil {
		return nil, err
	}
	if tick.Session == SessionUnknown {
		return nil, ErrMarketUnknown
	}
	availableAt, err := tick.AvailableAt()
	if err != nil {
		return nil, err
	}
	venue.mu.Lock()
	defer venue.mu.Unlock()
	orderProcessedKey := tick.TickID + "|" + onlyOrderID
	if onlyOrderID != "" {
		if prior, ok := venue.processedTicks[orderProcessedKey]; ok {
			if prior != tickFingerprint(tick) {
				return nil, ErrIdempotencyConflict
			}
			return nil, nil
		}
	}
	prior, tickSeen := venue.processedTicks[tick.TickID]
	if tickSeen {
		if prior != tickFingerprint(tick) {
			return nil, ErrIdempotencyConflict
		}
		if onlyOrderID == "" {
			return nil, nil
		}
		if !venue.lastTick.Equal(availableAt) {
			return nil, fmt.Errorf("%w: repeated market observation at %s is no longer the latest tick (last=%s)", ErrInvalidMarketData, availableAt, venue.lastTick)
		}
	} else if !venue.lastTick.IsZero() && !availableAt.After(venue.lastTick) {
		return nil, fmt.Errorf("%w: market tick %s must be after last tick %s", ErrInvalidMarketData, availableAt, venue.lastTick)
	}
	if tick.Session == SessionUnknown {
		return nil, ErrMarketUnknown
	}
	if !tickSeen {
		venue.processedTicks[tick.TickID] = tickFingerprint(tick)
		venue.lastTick = availableAt
	}
	if onlyOrderID != "" {
		venue.processedTicks[orderProcessedKey] = tickFingerprint(tick)
	}
	if tick.Session == SessionClosed {
		venue.activateOrdersLocked(availableAt)
		return nil, nil
	}
	venue.activateOrdersLocked(availableAt)
	ids := make([]string, 0, len(venue.orders))
	for id := range venue.orders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var fills []PaperFill
	available := tick.AvailableQuantity - venue.consumedByTick[tick.TickID]
	if available < 0 {
		available = 0
	}
	for _, id := range ids {
		if onlyOrderID != "" && id != onlyOrderID {
			continue
		}
		order := venue.orders[id]
		if order.Status != OrderActive && order.Status != OrderPartiallyFilled {
			continue
		}
		quantity := min(order.RemainingQuantity, available)
		if quantity <= 0 || !venue.matches(order, tick) {
			continue
		}
		costs, err := venue.costModel.Price(tick, order.Direction, quantity)
		if err != nil {
			return fills, err
		}
		if order.OrderType == OrderLimit && ((order.Direction == "LONG" && costs.ExecutedPrice > order.LimitPrice) || (order.Direction == "SHORT" && costs.ExecutedPrice < order.LimitPrice)) {
			continue
		}
		provenance, err := tick.Provenance()
		if err != nil && tick.RequireTemporalProvenance {
			return fills, err
		}
		fill := PaperFill{ContractVersion: FillContractVersion, OrderID: order.OrderID, PaperIntentID: order.PaperIntentID, WorkflowID: order.WorkflowID, InstrumentID: order.InstrumentID, Direction: order.Direction, Quantity: quantity, Price: costs.ExecutedPrice, Costs: costs, FilledAt: availableAt, TickID: tick.TickID}
		if provenance.StrictTemporal {
			fill.MarketProvenance = &provenance
		}
		fill.FillID = fillIdentity(fill)
		if err := fill.Validate(); err != nil {
			return fills, err
		}
		order.FilledQuantity += quantity
		order.RemainingQuantity -= quantity
		if order.RemainingQuantity <= quantityTolerance {
			order.RemainingQuantity = 0
		}
		if order.RemainingQuantity == 0 {
			order.Status = OrderFilled
		} else {
			order.Status = OrderPartiallyFilled
		}
		venue.orders[id] = order
		venue.fills[fill.FillID] = fill
		venue.consumedByTick[tick.TickID] += quantity
		fills = append(fills, fill)
		available -= quantity
	}
	return fills, nil
}

type CancelRequest struct {
	OrderID        string    `json:"order_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Now            time.Time `json:"now"`
}

func (venue *PaperVenue) Cancel(request CancelRequest) (PaperOrder, error) {
	if request.OrderID == "" || request.IdempotencyKey == "" || request.Now.IsZero() || request.Now.Location() != time.UTC {
		return PaperOrder{}, ErrInvalidArtifact
	}
	venue.mu.Lock()
	defer venue.mu.Unlock()
	fingerprint := valueFingerprint(struct {
		OrderID string
		Now     time.Time
	}{request.OrderID, request.Now})
	if prior, ok := venue.commandKeys[request.IdempotencyKey]; ok {
		if prior != fingerprint {
			return PaperOrder{}, ErrIdempotencyConflict
		}
		return venue.orders[request.OrderID], nil
	}
	order, ok := venue.orders[request.OrderID]
	if !ok {
		return PaperOrder{}, ErrOrderNotFound
	}
	if order.Status == OrderFilled || order.Status == OrderCancelled || order.Status == OrderRejected {
		return PaperOrder{}, ErrInvalidOrderTransition
	}
	if request.Now.Before(order.CreatedAt) {
		return PaperOrder{}, ErrInvalidOrderTransition
	}
	order.Status = OrderCancelled
	venue.orders[request.OrderID] = order
	venue.commandKeys[request.IdempotencyKey] = fingerprint
	return order, nil
}

func (venue *PaperVenue) activateOrdersLocked(at time.Time) {
	for id, order := range venue.orders {
		if order.Status == OrderNew && !at.Before(order.ActivatesAt) {
			order.Status = OrderActive
			venue.orders[id] = order
		}
	}
}

func (venue *PaperVenue) matches(order PaperOrder, tick MarketTick) bool {
	if tick.InstrumentID != order.InstrumentID {
		return false
	}
	if order.OrderType == OrderMarket {
		return true
	}
	mid := (tick.Bid + tick.Ask) / 2
	if order.Direction == "LONG" {
		return tick.Ask <= order.LimitPrice && mid > 0
	}
	return tick.Bid >= order.LimitPrice && mid > 0
}

func (venue *PaperVenue) Snapshot() VenueSnapshot {
	venue.mu.Lock()
	defer venue.mu.Unlock()
	orders := make(map[string]PaperOrder, len(venue.orders))
	for id, order := range venue.orders {
		orders[id] = order
	}
	fills := make(map[string]PaperFill, len(venue.fills))
	for id, fill := range venue.fills {
		fills[id] = fill
	}
	processed := make(map[string]string, len(venue.processedTicks))
	for key, value := range venue.processedTicks {
		processed[key] = value
	}
	commands := make(map[string]string, len(venue.commandKeys))
	for key, value := range venue.commandKeys {
		commands[key] = value
	}
	consumed := make(map[string]float64, len(venue.consumedByTick))
	for key, value := range venue.consumedByTick {
		consumed[key] = value
	}
	return VenueSnapshot{Contract: venue.contract, CostModel: venue.costModel, Orders: orders, Fills: fills, ProcessedTicks: processed, CommandKeys: commands, ConsumedByTick: consumed, LastTick: venue.lastTick}
}

func fillIdentity(fill PaperFill) string {
	copyFill := fill
	copyFill.FillID = ""
	// Keep artifact identity stable across PostgreSQL's microsecond timestamp
	// precision when fills are restored and validated after a restart.
	copyFill.FilledAt = canonicalPersistedTime(copyFill.FilledAt)
	data, _ := json.Marshal(copyFill)
	digest := sha256.Sum256(data)
	return "pfl_" + hex.EncodeToString(digest[:])
}

func tickFingerprint(tick MarketTick) string {
	return valueFingerprint(tick)
}

func valueFingerprint(value any) string {
	data, _ := json.Marshal(value)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
