package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOrder_JSONMarshaling(t *testing.T) {
	filledAt := time.Date(2026, 2, 13, 10, 35, 0, 0, time.UTC)
	order := Order{
		ID:             "ord-123",
		Symbol:         "AAPL",
		Type:           "limit",
		Side:           "buy",
		Quantity:       100,
		Price:          150.25,
		TimeInForce:    "day",
		Status:         "filled",
		SubmittedAt:    time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		FilledAt:       &filledAt,
		FilledQuantity: 100,
		FilledPrice:    150.20,
		Commission:     1.50,
	}

	// Marshal to JSON
	data, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("Failed to marshal order: %v", err)
	}

	// Unmarshal back
	var decoded Order
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal order: %v", err)
	}

	// Verify fields
	if decoded.ID != order.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, order.ID)
	}
	if decoded.Symbol != order.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", decoded.Symbol, order.Symbol)
	}
	if decoded.Quantity != order.Quantity {
		t.Errorf("Quantity mismatch: got %d, want %d", decoded.Quantity, order.Quantity)
	}
	if decoded.FilledAt == nil {
		t.Errorf("FilledAt should not be nil")
	} else if !decoded.FilledAt.Equal(*order.FilledAt) {
		t.Errorf("FilledAt mismatch: got %v, want %v", *decoded.FilledAt, *order.FilledAt)
	}
}

func TestOrder_RejectedOrder(t *testing.T) {
	order := Order{
		ID:              "ord-456",
		Symbol:          "TSLA",
		Type:            "market",
		Side:            "sell",
		Quantity:        50,
		Status:          "rejected",
		SubmittedAt:     time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
		FilledQuantity:  0,
		RejectionReason: "Insufficient buying power",
	}

	data, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("Failed to marshal rejected order: %v", err)
	}

	var decoded Order
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal rejected order: %v", err)
	}

	if decoded.Status != "rejected" {
		t.Errorf("Status mismatch: got %s, want rejected", decoded.Status)
	}
	if decoded.RejectionReason != order.RejectionReason {
		t.Errorf("RejectionReason mismatch: got %s, want %s", decoded.RejectionReason, order.RejectionReason)
	}
	if decoded.FilledAt != nil {
		t.Errorf("FilledAt should be nil for rejected order")
	}
}
