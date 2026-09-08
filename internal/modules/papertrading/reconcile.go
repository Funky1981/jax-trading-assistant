package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

type ReconciliationStatus string

const (
	ReconciliationClean    ReconciliationStatus = "CLEAN"
	ReconciliationRequired ReconciliationStatus = "RECONCILIATION_REQUIRED"
)

type ReasonCode string

const (
	ReasonMissingOrder       ReasonCode = "MISSING_ORDER"
	ReasonDuplicateOrder     ReasonCode = "DUPLICATE_ORDER"
	ReasonMissingFill        ReasonCode = "MISSING_FILL"
	ReasonDuplicateFill      ReasonCode = "DUPLICATE_FILL"
	ReasonQuantityMismatch   ReasonCode = "QUANTITY_MISMATCH"
	ReasonLedgerMismatch     ReasonCode = "LEDGER_MISMATCH"
	ReasonInvalidMarketData  ReasonCode = "INVALID_MARKET_REFERENCE"
	ReasonUnknownEnvironment ReasonCode = "UNKNOWN_ENVIRONMENT"
	ReasonWorkflowMismatch   ReasonCode = "WORKFLOW_MISMATCH"
)

type ReconciliationInput struct {
	VenueSnapshot VenueSnapshot `json:"venue_snapshot"`
	Account       PaperAccount  `json:"account"`
	MarketTick    *MarketTick   `json:"market_tick,omitempty"`
}

type ReconciliationResult struct {
	Identity        string               `json:"identity"`
	ContractVersion string               `json:"contract_version"`
	Status          ReconciliationStatus `json:"status"`
	ReasonCodes     []ReasonCode         `json:"reason_codes"`
	Details         []string             `json:"details,omitempty"`
	CheckedAt       time.Time            `json:"checked_at"`
}

func Reconcile(input ReconciliationInput, checkedAt time.Time) ReconciliationResult {
	result := ReconciliationResult{ContractVersion: ReconciliationVersion, Status: ReconciliationClean, CheckedAt: checkedAt.UTC()}
	add := func(code ReasonCode, detail string) {
		result.Status = ReconciliationRequired
		result.ReasonCodes = append(result.ReasonCodes, code)
		result.Details = append(result.Details, detail)
	}
	if input.VenueSnapshot.Contract.Environment != EnvironmentPaper {
		add(ReasonUnknownEnvironment, "venue environment is not PAPER")
	}
	if input.VenueSnapshot.Contract.Validate() != nil || input.VenueSnapshot.CostModel.Validate() != nil {
		add(ReasonUnknownEnvironment, "venue or cost contract is invalid")
	}
	if input.Account.Environment != EnvironmentPaper || input.Account.ContractVersion != LedgerContractVersion {
		add(ReasonUnknownEnvironment, "account environment or contract is invalid")
	}
	if input.MarketTick != nil {
		if err := input.MarketTick.Validate(input.VenueSnapshot.Contract.MaxQuoteAge); err != nil {
			add(ReasonInvalidMarketData, err.Error())
		}
	}
	if checkedAt.IsZero() || checkedAt.Location() != time.UTC {
		add(ReasonInvalidMarketData, "reconciliation timestamp must be UTC")
	}
	fillByOrder := map[string]float64{}
	seenFills := map[string]bool{}
	for fillID, fill := range input.VenueSnapshot.Fills {
		if fillID != fill.FillID || fill.Validate() != nil {
			add(ReasonMissingFill, "fill identity or contract is invalid")
			continue
		}
		if seenFills[fill.FillID] {
			add(ReasonDuplicateFill, "fill appears more than once")
		}
		seenFills[fill.FillID] = true
		if _, ok := input.VenueSnapshot.Orders[fill.OrderID]; !ok {
			add(ReasonMissingOrder, "fill references a missing order")
		} else {
			order := input.VenueSnapshot.Orders[fill.OrderID]
			if fill.PaperIntentID != order.PaperIntentID || fill.WorkflowID != order.WorkflowID || fill.InstrumentID != order.InstrumentID || fill.Direction != order.Direction {
				add(ReasonWorkflowMismatch, "fill provenance does not match its order")
			}
		}
		fillByOrder[fill.OrderID] += fill.Quantity
	}
	for orderID, order := range input.VenueSnapshot.Orders {
		if orderID != order.OrderID || order.Validate() != nil {
			add(ReasonMissingOrder, "order identity or contract is invalid")
			continue
		}
		filled := fillByOrder[orderID]
		if math.Abs(filled-order.FilledQuantity) > 1e-9 || math.Abs(order.FilledQuantity+order.RemainingQuantity-order.Quantity) > 1e-9 {
			add(ReasonQuantityMismatch, "order quantity does not reconcile to fills")
		}
		if order.Status == OrderFilled && order.RemainingQuantity != 0 || order.Status == OrderPartiallyFilled && order.RemainingQuantity == 0 {
			add(ReasonQuantityMismatch, "order status does not match remaining quantity")
		}
	}
	if input.Account.AccountID == "" || input.Account.Currency == "" || !finitePositive(input.Account.InitialCash) || !finiteNonNegative(input.Account.Cash) {
		add(ReasonLedgerMismatch, "account identity or balances are invalid")
	} else {
		ledger, err := NewPaperLedger(input.Account.AccountID, input.Account.Currency, input.Account.InitialCash)
		if err != nil {
			add(ReasonLedgerMismatch, "paper ledger cannot be reconstructed")
		} else {
			fills := make([]PaperFill, 0, len(input.VenueSnapshot.Fills))
			for _, fill := range input.VenueSnapshot.Fills {
				fills = append(fills, fill)
			}
			sort.Slice(fills, func(i, j int) bool {
				if fills[i].FilledAt.Equal(fills[j].FilledAt) {
					return fills[i].FillID < fills[j].FillID
				}
				return fills[i].FilledAt.Before(fills[j].FilledAt)
			})
			for _, fill := range fills {
				if _, err := ledger.ApplyFill(fill); err != nil {
					add(ReasonLedgerMismatch, fmt.Sprintf("fill %s cannot be posted: %v", fill.FillID, err))
				}
			}
			expected := ledger.Snapshot()
			if !accountsMatch(expected, input.Account) {
				add(ReasonLedgerMismatch, "derived account differs from supplied account")
			}
		}
	}
	result.Identity = reconciliationIdentity(input, result)
	return result
}

func accountsMatch(expected, actual PaperAccount) bool {
	if math.Abs(expected.Cash-actual.Cash) > 1e-9 || math.Abs(expected.Equity-actual.Equity) > 1e-9 || math.Abs(expected.RealizedPnL-actual.RealizedPnL) > 1e-9 || math.Abs(expected.Fees-actual.Fees) > 1e-9 || len(expected.Positions) != len(actual.Positions) || len(expected.Events) != len(actual.Events) {
		return false
	}
	for key, position := range expected.Positions {
		other, ok := actual.Positions[key]
		if !ok || math.Abs(position.Quantity-other.Quantity) > 1e-9 || math.Abs(position.AverageCost-other.AverageCost) > 1e-9 {
			return false
		}
	}
	for index, event := range expected.Events {
		if event != actual.Events[index] {
			return false
		}
	}
	return true
}

func reconciliationIdentity(input ReconciliationInput, result ReconciliationResult) string {
	data, _ := json.Marshal(struct {
		Input   ReconciliationInput  `json:"input"`
		Status  ReconciliationStatus `json:"status"`
		Reasons []ReasonCode         `json:"reasons"`
	}{input, result.Status, result.ReasonCodes})
	digest := sha256.Sum256(data)
	return "prec_" + hex.EncodeToString(digest[:])
}
