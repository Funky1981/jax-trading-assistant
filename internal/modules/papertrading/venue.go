package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

type PaperFill struct {
	FillID          string        `json:"fill_id"`
	ContractVersion string        `json:"contract_version"`
	OrderID         string        `json:"order_id"`
	PaperIntentID   string        `json:"paper_intent_id"`
	WorkflowID      string        `json:"workflow_id"`
	InstrumentID    string        `json:"instrument_id"`
	Direction       string        `json:"direction"`
	Quantity        float64       `json:"quantity"`
	Price           float64       `json:"price"`
	Costs           CostBreakdown `json:"costs"`
	FilledAt        time.Time     `json:"filled_at"`
	TickID          string        `json:"tick_id"`
}

func (fill PaperFill) Validate() error {
	if fill.ContractVersion != FillContractVersion || fill.OrderID == "" || fill.PaperIntentID == "" || fill.WorkflowID == "" || fill.InstrumentID == "" || (fill.Direction != "LONG" && fill.Direction != "SHORT") || !finitePositive(fill.Quantity) || !finitePositive(fill.Price) || fill.TickID == "" || fill.FilledAt.IsZero() || fill.FilledAt.Location() != time.UTC || fill.Costs.ModelID == "" || fill.FillID != fillIdentity(fill) {
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
	lastTick       time.Time
}

func NewPaperVenue(contract CapabilityContract, costModel CostModel) (*PaperVenue, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}
	if err := costModel.Validate(); err != nil {
		return nil, err
	}
	return &PaperVenue{contract: contract, costModel: costModel, orders: map[string]PaperOrder{}, fills: map[string]PaperFill{}, intentOrders: map[string]string{}, processedTicks: map[string]string{}, commandKeys: map[string]string{}}, nil
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
		if id != order.OrderID || order.Validate() != nil {
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
		if !exists || fill.Quantity > order.Quantity {
			return nil, ErrInvalidArtifact
		}
		venue.fills[id] = fill
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
	if !snapshot.LastTick.IsZero() && snapshot.LastTick.Location() != time.UTC {
		return nil, ErrInvalidArtifact
	}
	venue.lastTick = snapshot.LastTick
	return venue, nil
}

func (venue *PaperVenue) Submit(request CreateOrderRequest) (PaperOrder, error) {
	if request.Venue.VenueID != venue.contract.VenueID || request.CostModel.ModelID != venue.costModel.ModelID {
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

func (venue *PaperVenue) ProcessTick(tick MarketTick) ([]PaperFill, error) {
	return venue.ProcessTickWithSafety(tick, false)
}

func (venue *PaperVenue) ProcessTickWithSafety(tick MarketTick, breakerTripped bool) ([]PaperFill, error) {
	if breakerTripped {
		return nil, ErrPaperExecutionBlocked
	}
	if err := tick.Validate(venue.contract.MaxQuoteAge); err != nil {
		return nil, err
	}
	if tick.Session == SessionUnknown {
		return nil, ErrMarketUnknown
	}
	venue.mu.Lock()
	defer venue.mu.Unlock()
	if prior, ok := venue.processedTicks[tick.TickID]; ok {
		if prior != tickFingerprint(tick) {
			return nil, ErrIdempotencyConflict
		}
		return nil, nil
	}
	if !venue.lastTick.IsZero() && !tick.Timestamp.After(venue.lastTick) {
		return nil, fmt.Errorf("%w: market ticks must be strictly ordered", ErrInvalidMarketData)
	}
	if tick.Session == SessionUnknown {
		return nil, ErrMarketUnknown
	}
	venue.processedTicks[tick.TickID] = tickFingerprint(tick)
	venue.lastTick = tick.Timestamp
	if tick.Session == SessionClosed {
		venue.activateOrdersLocked(tick.Timestamp)
		return nil, nil
	}
	venue.activateOrdersLocked(tick.Timestamp)
	ids := make([]string, 0, len(venue.orders))
	for id := range venue.orders {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var fills []PaperFill
	available := tick.AvailableQuantity
	for _, id := range ids {
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
		fill := PaperFill{ContractVersion: FillContractVersion, OrderID: order.OrderID, PaperIntentID: order.PaperIntentID, WorkflowID: order.WorkflowID, InstrumentID: order.InstrumentID, Direction: order.Direction, Quantity: quantity, Price: costs.ExecutedPrice, Costs: costs, FilledAt: tick.Timestamp, TickID: tick.TickID}
		fill.FillID = fillIdentity(fill)
		if err := fill.Validate(); err != nil {
			return fills, err
		}
		order.FilledQuantity += quantity
		order.RemainingQuantity -= quantity
		if order.RemainingQuantity == 0 {
			order.Status = OrderFilled
		} else {
			order.Status = OrderPartiallyFilled
		}
		venue.orders[id] = order
		venue.fills[fill.FillID] = fill
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
	return VenueSnapshot{Contract: venue.contract, CostModel: venue.costModel, Orders: orders, Fills: fills, ProcessedTicks: processed, CommandKeys: commands, LastTick: venue.lastTick}
}

func fillIdentity(fill PaperFill) string {
	copyFill := fill
	copyFill.FillID = ""
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
