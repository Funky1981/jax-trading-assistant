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

type LedgerPosition struct {
	InstrumentID string  `json:"instrument_id"`
	Quantity     float64 `json:"quantity"`
	AverageCost  float64 `json:"average_cost"`
}

type LedgerEvent struct {
	EventID         string    `json:"event_id"`
	ContractVersion string    `json:"contract_version"`
	FillID          string    `json:"fill_id"`
	OrderID         string    `json:"order_id"`
	WorkflowID      string    `json:"workflow_id"`
	InstrumentID    string    `json:"instrument_id"`
	Direction       string    `json:"direction"`
	Quantity        float64   `json:"quantity"`
	Price           float64   `json:"price"`
	Fee             float64   `json:"fee"`
	CashDelta       float64   `json:"cash_delta"`
	RealizedPnL     float64   `json:"realized_pnl"`
	OccurredAt      time.Time `json:"occurred_at"`
}

func (event LedgerEvent) Validate() error {
	if event.ContractVersion != LedgerContractVersion || event.EventID == "" || event.FillID == "" || event.OrderID == "" || event.WorkflowID == "" || event.InstrumentID == "" || (event.Direction != "LONG" && event.Direction != "SHORT") || !finitePositive(event.Quantity) || !finitePositive(event.Price) || !finiteNonNegative(event.Fee) || !finite(event.CashDelta) || !finite(event.RealizedPnL) || event.OccurredAt.IsZero() || event.OccurredAt.Location() != time.UTC || event.EventID != ledgerEventIdentity(event) {
		return ErrInvalidArtifact
	}
	return nil
}

type PaperAccount struct {
	AccountID       string                    `json:"account_id"`
	ContractVersion string                    `json:"contract_version"`
	Environment     Environment               `json:"environment"`
	Currency        string                    `json:"currency"`
	InitialCash     float64                   `json:"initial_cash"`
	Cash            float64                   `json:"cash"`
	Equity          float64                   `json:"equity"`
	RealizedPnL     float64                   `json:"realized_pnl"`
	Fees            float64                   `json:"fees"`
	Positions       map[string]LedgerPosition `json:"positions"`
	Events          []LedgerEvent             `json:"events"`
}

type PaperLedger struct {
	mu      sync.Mutex
	account PaperAccount
	applied map[string]LedgerEvent
}

func NewPaperLedger(accountID, currency string, initialCash float64) (*PaperLedger, error) {
	if accountID == "" || currency == "" || !finitePositive(initialCash) {
		return nil, ErrInvalidArtifact
	}
	return &PaperLedger{account: PaperAccount{AccountID: accountID, ContractVersion: LedgerContractVersion, Environment: EnvironmentPaper, Currency: currency, InitialCash: initialCash, Cash: initialCash, Equity: initialCash, Positions: map[string]LedgerPosition{}, Events: []LedgerEvent{}}, applied: map[string]LedgerEvent{}}, nil
}

func RestorePaperLedger(account PaperAccount) (*PaperLedger, error) {
	ledger, err := NewPaperLedger(account.AccountID, account.Currency, account.InitialCash)
	if err != nil || account.ContractVersion != LedgerContractVersion || account.Environment != EnvironmentPaper || !finiteNonNegative(account.Cash) || !finite(account.Equity) || !finite(account.RealizedPnL) || !finiteNonNegative(account.Fees) || account.Positions == nil {
		return nil, ErrInvalidArtifact
	}
	previous := time.Time{}
	for _, event := range account.Events {
		if err := event.Validate(); err != nil || (!previous.IsZero() && event.OccurredAt.Before(previous)) {
			return nil, ErrInvalidArtifact
		}
		previous = event.OccurredAt
		ledger.applied[event.FillID] = event
	}
	replayed := PaperAccount{AccountID: account.AccountID, ContractVersion: LedgerContractVersion, Environment: EnvironmentPaper, Currency: account.Currency, InitialCash: account.InitialCash, Cash: account.InitialCash, Equity: account.InitialCash, Positions: map[string]LedgerPosition{}, Events: []LedgerEvent{}}
	for _, event := range account.Events {
		position := replayed.Positions[event.InstrumentID]
		notional := event.Price * event.Quantity
		var cashDelta, realized float64
		if event.Direction == "LONG" {
			cashDelta = -(notional + event.Fee)
			if replayed.Cash+cashDelta < 0 {
				return nil, ErrInvalidArtifact
			}
			newQuantity := position.Quantity + event.Quantity
			position = LedgerPosition{InstrumentID: event.InstrumentID, Quantity: newQuantity, AverageCost: (position.Quantity*position.AverageCost + notional) / newQuantity}
			replayed.Positions[event.InstrumentID] = position
		} else {
			if position.Quantity < event.Quantity {
				return nil, ErrInvalidArtifact
			}
			cashDelta = notional - event.Fee
			realized = (event.Price - position.AverageCost) * event.Quantity
			position.Quantity -= event.Quantity
			if position.Quantity == 0 {
				delete(replayed.Positions, event.InstrumentID)
			} else {
				replayed.Positions[event.InstrumentID] = position
			}
		}
		if math.Abs(cashDelta-event.CashDelta) > quantityTolerance || math.Abs(realized-event.RealizedPnL) > quantityTolerance {
			return nil, ErrInvalidArtifact
		}
		replayed.Cash += cashDelta
		replayed.RealizedPnL += realized
		replayed.Fees += event.Fee
		replayed.Equity = replayed.Cash + positionValue(replayed.Positions)
		replayed.Events = append(replayed.Events, event)
	}
	if !accountsMatch(replayed, account) {
		return nil, ErrInvalidArtifact
	}
	ledger.account = cloneAccount(account)
	return ledger, nil
}

func (ledger *PaperLedger) ApplyFill(fill PaperFill) (PaperAccount, error) {
	if err := fill.Validate(); err != nil {
		return PaperAccount{}, err
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.applied[fill.FillID]; ok {
		if existing.EventID != ledgerEventIdentityForFill(existing, fill) {
			return PaperAccount{}, ErrIdempotencyConflict
		}
		return cloneAccount(ledger.account), nil
	}
	account := cloneAccount(ledger.account)
	position := account.Positions[fill.InstrumentID]
	notional := fill.Price * fill.Quantity
	fee := fill.Costs.Commission
	if !finitePositive(notional) || !finiteNonNegative(fee) {
		return PaperAccount{}, ErrInvalidArtifact
	}
	var cashDelta, realized float64
	if fill.Direction == "LONG" {
		cashDelta = -(notional + fee)
		if account.Cash+cashDelta < 0 {
			return PaperAccount{}, ErrInsufficientCapital
		}
		newQuantity := position.Quantity + fill.Quantity
		average := (position.Quantity*position.AverageCost + notional) / newQuantity
		position = LedgerPosition{InstrumentID: fill.InstrumentID, Quantity: newQuantity, AverageCost: average}
		account.Positions[fill.InstrumentID] = position
	} else {
		if position.Quantity < fill.Quantity {
			return PaperAccount{}, ErrInsufficientCapital
		}
		cashDelta = notional - fee
		realized = (fill.Price - position.AverageCost) * fill.Quantity
		position.Quantity -= fill.Quantity
		if position.Quantity == 0 {
			delete(account.Positions, fill.InstrumentID)
		} else {
			account.Positions[fill.InstrumentID] = position
		}
	}
	account.Cash += cashDelta
	account.RealizedPnL += realized
	account.Fees += fee
	account.Equity = account.Cash + positionValue(account.Positions)
	event := LedgerEvent{ContractVersion: LedgerContractVersion, FillID: fill.FillID, OrderID: fill.OrderID, WorkflowID: fill.WorkflowID, InstrumentID: fill.InstrumentID, Direction: fill.Direction, Quantity: fill.Quantity, Price: fill.Price, Fee: fee, CashDelta: cashDelta, RealizedPnL: realized, OccurredAt: fill.FilledAt}
	event.EventID = ledgerEventIdentity(event)
	if err := event.Validate(); err != nil {
		return PaperAccount{}, err
	}
	account.Events = append(account.Events, event)
	ledger.account = account
	ledger.applied[fill.FillID] = event
	return cloneAccount(account), nil
}

func (ledger *PaperLedger) MarkToMarket(prices map[string]float64) (PaperAccount, error) {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	account := cloneAccount(ledger.account)
	value := 0.0
	for instrument, position := range account.Positions {
		price, ok := prices[instrument]
		if !ok || !finitePositive(price) {
			return PaperAccount{}, fmt.Errorf("%w: missing price for %s", ErrInvalidMarketData, instrument)
		}
		value += price * position.Quantity
	}
	account.Equity = account.Cash + value
	return account, nil
}

func (ledger *PaperLedger) Snapshot() PaperAccount {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	return cloneAccount(ledger.account)
}

func cloneAccount(account PaperAccount) PaperAccount {
	copyAccount := account
	copyAccount.Positions = make(map[string]LedgerPosition, len(account.Positions))
	for key, position := range account.Positions {
		copyAccount.Positions[key] = position
	}
	copyAccount.Events = append([]LedgerEvent(nil), account.Events...)
	return copyAccount
}

func positionValue(positions map[string]LedgerPosition) float64 {
	keys := make([]string, 0, len(positions))
	for key := range positions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	value := 0.0
	for _, key := range keys {
		value += positions[key].Quantity * positions[key].AverageCost
	}
	return value
}

func ledgerEventIdentity(event LedgerEvent) string {
	data, _ := json.Marshal(struct {
		ContractVersion string    `json:"contract_version"`
		FillID          string    `json:"fill_id"`
		OrderID         string    `json:"order_id"`
		WorkflowID      string    `json:"workflow_id"`
		InstrumentID    string    `json:"instrument_id"`
		Direction       string    `json:"direction"`
		Quantity        float64   `json:"quantity"`
		Price           float64   `json:"price"`
		Fee             float64   `json:"fee"`
		CashDelta       float64   `json:"cash_delta"`
		RealizedPnL     float64   `json:"realized_pnl"`
		OccurredAt      time.Time `json:"occurred_at"`
	}{event.ContractVersion, event.FillID, event.OrderID, event.WorkflowID, event.InstrumentID, event.Direction, event.Quantity, event.Price, event.Fee, event.CashDelta, event.RealizedPnL, event.OccurredAt})
	digest := sha256.Sum256(data)
	return "ple_" + hex.EncodeToString(digest[:])
}

func ledgerEventIdentityForFill(event LedgerEvent, fill PaperFill) string {
	copyEvent := event
	copyEvent.EventID = ""
	return ledgerEventIdentity(copyEvent)
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
