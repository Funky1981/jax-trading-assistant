package papertrading

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/workflow"
)

type CreateOrderRequest struct {
	Workflow       workflow.Workflow    `json:"workflow"`
	PaperIntent    workflow.PaperIntent `json:"paper_intent"`
	Venue          CapabilityContract   `json:"venue"`
	CostModel      CostModel            `json:"cost_model"`
	InstrumentID   string               `json:"instrument_id"`
	Quantity       float64              `json:"quantity"`
	ReferencePrice float64              `json:"reference_price"`
	OrderType      OrderType            `json:"order_type"`
	LimitPrice     float64              `json:"limit_price,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	IdempotencyKey string               `json:"idempotency_key"`
}

type PaperOrderStatus string

const (
	OrderNew             PaperOrderStatus = "NEW"
	OrderActive          PaperOrderStatus = "ACTIVE"
	OrderPartiallyFilled PaperOrderStatus = "PARTIALLY_FILLED"
	OrderFilled          PaperOrderStatus = "FILLED"
	OrderCancelled       PaperOrderStatus = "CANCELLED"
	OrderRejected        PaperOrderStatus = "REJECTED"
)

type PaperOrder struct {
	OrderID           string           `json:"order_id"`
	ContractVersion   string           `json:"contract_version"`
	VenueID           string           `json:"venue_id"`
	Environment       Environment      `json:"environment"`
	PaperIntentID     string           `json:"paper_intent_id"`
	WorkflowID        string           `json:"workflow_id"`
	RecommendationID  string           `json:"recommendation_id"`
	RiskDecisionID    string           `json:"risk_decision_id"`
	ConfirmationID    string           `json:"confirmation_id"`
	InstrumentID      string           `json:"instrument_id"`
	Direction         string           `json:"direction"`
	Quantity          float64          `json:"quantity"`
	RemainingQuantity float64          `json:"remaining_quantity"`
	FilledQuantity    float64          `json:"filled_quantity"`
	OrderType         OrderType        `json:"order_type"`
	LimitPrice        float64          `json:"limit_price,omitempty"`
	CostModelID       string           `json:"cost_model_id"`
	CreatedAt         time.Time        `json:"created_at"`
	ActivatesAt       time.Time        `json:"activates_at"`
	Status            PaperOrderStatus `json:"status"`
}

func CreatePaperOrder(request CreateOrderRequest) (PaperOrder, error) {
	if err := request.Venue.Validate(); err != nil {
		return PaperOrder{}, err
	}
	if request.Venue.Environment != EnvironmentPaper {
		return PaperOrder{}, ErrLiveExecutionDisabled
	}
	if err := request.CostModel.Validate(); err != nil {
		return PaperOrder{}, err
	}
	if err := request.Workflow.Validate(); err != nil {
		return PaperOrder{}, fmt.Errorf("workflow: %w", err)
	}
	if err := request.PaperIntent.Validate(); err != nil {
		return PaperOrder{}, fmt.Errorf("paper intent: %w", err)
	}
	if request.Workflow.State != workflow.StatePaperIntentCreated || request.Workflow.PaperIntentID != request.PaperIntent.IntentID || request.PaperIntent.WorkflowID != request.Workflow.WorkflowID || request.Workflow.Confirmation == nil || request.Workflow.Confirmation.Decision != workflow.ConfirmationApprove {
		return PaperOrder{}, ErrInvalidArtifact
	}
	if request.Workflow.Confirmation.ConfirmationID == "" || request.Workflow.Confirmation.BrokerExecutionAllowed || request.Workflow.Confirmation.LiveTradingAllowed {
		return PaperOrder{}, ErrLiveExecutionDisabled
	}
	if request.InstrumentID == "" || request.InstrumentID != request.PaperIntent.InstrumentID || request.InstrumentID != request.Workflow.Confirmation.InstrumentID || (request.PaperIntent.Direction != "LONG" && request.PaperIntent.Direction != "SHORT") || !finitePositive(request.Quantity) || !finitePositive(request.ReferencePrice) || request.Quantity*request.ReferencePrice > request.PaperIntent.DescriptiveValue {
		return PaperOrder{}, ErrInvalidArtifact
	}
	if request.Quantity < request.Venue.MinimumQuantity || (!request.Venue.SupportsFractionalQty && request.Quantity != float64(int64(request.Quantity))) {
		return PaperOrder{}, ErrUnsupportedCapability
	}
	if err := request.OrderType.Validate(); err != nil || !request.Venue.Supports(request.OrderType) {
		return PaperOrder{}, ErrUnsupportedCapability
	}
	if request.OrderType == OrderLimit && (!finitePositive(request.LimitPrice) || request.LimitPrice < request.ReferencePrice/1000) {
		return PaperOrder{}, ErrInvalidArtifact
	}
	if request.CreatedAt.IsZero() || request.CreatedAt.Location() != time.UTC || request.IdempotencyKey == "" {
		return PaperOrder{}, ErrInvalidArtifact
	}
	if err := request.Workflow.Confirmation.ValidateFor(request.Workflow, request.CreatedAt); err != nil {
		return PaperOrder{}, err
	}
	order := PaperOrder{ContractVersion: OrderContractVersion, VenueID: request.Venue.VenueID, Environment: EnvironmentPaper, PaperIntentID: request.PaperIntent.IntentID, WorkflowID: request.Workflow.WorkflowID, RecommendationID: request.PaperIntent.RecommendationID, RiskDecisionID: request.PaperIntent.RiskDecisionID, ConfirmationID: request.Workflow.Confirmation.ConfirmationID, InstrumentID: request.InstrumentID, Direction: request.PaperIntent.Direction, Quantity: request.Quantity, RemainingQuantity: request.Quantity, OrderType: request.OrderType, LimitPrice: request.LimitPrice, CostModelID: request.CostModel.ModelID, CreatedAt: request.CreatedAt, ActivatesAt: request.CreatedAt.Add(request.CostModel.Latency), Status: OrderNew}
	order.OrderID = orderIdentity(order)
	return order, order.Validate()
}

func (order PaperOrder) Validate() error {
	if order.ContractVersion != OrderContractVersion || order.Environment != EnvironmentPaper || order.VenueID == "" || order.PaperIntentID == "" || order.WorkflowID == "" || order.RecommendationID == "" || order.RiskDecisionID == "" || order.ConfirmationID == "" || order.InstrumentID == "" || (order.Direction != "LONG" && order.Direction != "SHORT") || !finitePositive(order.Quantity) || !finiteNonNegative(order.RemainingQuantity) || !finiteNonNegative(order.FilledQuantity) || order.FilledQuantity+order.RemainingQuantity != order.Quantity || order.OrderType.Validate() != nil || order.CreatedAt.IsZero() || order.ActivatesAt.IsZero() || order.CreatedAt.Location() != time.UTC || order.ActivatesAt.Location() != time.UTC || order.ActivatesAt.Before(order.CreatedAt) || order.CostModelID == "" {
		return ErrInvalidArtifact
	}
	if order.Status == OrderNew && order.FilledQuantity != 0 || order.Status == OrderFilled && order.RemainingQuantity != 0 || order.Status == OrderPartiallyFilled && (order.FilledQuantity == 0 || order.RemainingQuantity == 0) {
		return ErrInvalidArtifact
	}
	switch order.Status {
	case OrderNew, OrderActive, OrderPartiallyFilled, OrderFilled, OrderCancelled, OrderRejected:
	default:
		return ErrInvalidArtifact
	}
	if order.OrderID != orderIdentity(order) {
		return ErrInvalidArtifact
	}
	return nil
}

func orderIdentity(order PaperOrder) string {
	copyOrder := order
	copyOrder.OrderID = ""
	copyOrder.RemainingQuantity = 0
	copyOrder.FilledQuantity = 0
	copyOrder.Status = ""
	data, _ := json.Marshal(copyOrder)
	digest := sha256.Sum256(data)
	return "pord_" + hex.EncodeToString(digest[:])
}

func directionMultiplier(direction string) float64 {
	if strings.EqualFold(direction, "SHORT") {
		return -1
	}
	return 1
}
