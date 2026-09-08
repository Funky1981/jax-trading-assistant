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
	Contract  CapabilityContract    `json:"contract"`
	CostModel CostModel             `json:"cost_model"`
	Orders    map[string]PaperOrder `json:"orders"`
	Fills     map[string]PaperFill  `json:"fills"`
}

type PaperVenue struct {
	mu             sync.Mutex
	contract       CapabilityContract
	costModel      CostModel
	orders         map[string]PaperOrder
	fills          map[string]PaperFill
	intentOrders   map[string]string
	processedTicks map[string]string
	lastTick       time.Time
}

func NewPaperVenue(contract CapabilityContract, costModel CostModel) (*PaperVenue, error) {
	if err := contract.Validate(); err != nil {
		return nil, err
	}
	if err := costModel.Validate(); err != nil {
		return nil, err
	}
	return &PaperVenue{contract: contract, costModel: costModel, orders: map[string]PaperOrder{}, fills: map[string]PaperFill{}, intentOrders: map[string]string{}, processedTicks: map[string]string{}}, nil
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
	if err := tick.Validate(venue.contract.MaxQuoteAge); err != nil {
		return nil, err
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
	for _, id := range ids {
		order := venue.orders[id]
		if order.Status != OrderActive && order.Status != OrderPartiallyFilled {
			continue
		}
		quantity := min(order.RemainingQuantity, tick.AvailableQuantity)
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
	}
	return fills, nil
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
	return VenueSnapshot{Contract: venue.contract, CostModel: venue.costModel, Orders: orders, Fills: fills}
}

func fillIdentity(fill PaperFill) string {
	copyFill := fill
	copyFill.FillID = ""
	data, _ := json.Marshal(copyFill)
	digest := sha256.Sum256(data)
	return "pfl_" + hex.EncodeToString(digest[:])
}

func tickFingerprint(tick MarketTick) string {
	data, _ := json.Marshal(tick)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
