package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPosition_JSONMarshaling(t *testing.T) {
	position := Position{
		Symbol:        "AAPL",
		Quantity:      100,
		AvgEntryPrice: 150.25,
		CurrentPrice:  152.50,
		UnrealizedPL:  225.00,
		RealizedPL:    50.00,
		OpenedAt:      time.Date(2026, 2, 10, 9, 30, 0, 0, time.UTC),
		LastUpdated:   time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.Marshal(position)
	if err != nil {
		t.Fatalf("Failed to marshal position: %v", err)
	}

	// Unmarshal back
	var decoded Position
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal position: %v", err)
	}

	// Verify fields
	if decoded.Symbol != position.Symbol {
		t.Errorf("Symbol mismatch: got %s, want %s", decoded.Symbol, position.Symbol)
	}
	if decoded.Quantity != position.Quantity {
		t.Errorf("Quantity mismatch: got %d, want %d", decoded.Quantity, position.Quantity)
	}
	if decoded.UnrealizedPL != position.UnrealizedPL {
		t.Errorf("UnrealizedPL mismatch: got %f, want %f", decoded.UnrealizedPL, position.UnrealizedPL)
	}
}

func TestPortfolio_JSONMarshaling(t *testing.T) {
	portfolio := Portfolio{
		AccountID:   "acc-123",
		Cash:        50000.00,
		BuyingPower: 200000.00,
		Positions: []Position{
			{
				Symbol:        "AAPL",
				Quantity:      100,
				AvgEntryPrice: 150.25,
				CurrentPrice:  152.50,
				UnrealizedPL:  225.00,
				OpenedAt:      time.Date(2026, 2, 10, 9, 30, 0, 0, time.UTC),
				LastUpdated:   time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
			},
			{
				Symbol:        "TSLA",
				Quantity:      50,
				AvgEntryPrice: 200.00,
				CurrentPrice:  195.00,
				UnrealizedPL:  -250.00,
				OpenedAt:      time.Date(2026, 2, 11, 9, 30, 0, 0, time.UTC),
				LastUpdated:   time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
			},
		},
		TotalValue:  59975.00,
		TotalPL:     -25.00,
		LastUpdated: time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
	}

	// Marshal to JSON
	data, err := json.Marshal(portfolio)
	if err != nil {
		t.Fatalf("Failed to marshal portfolio: %v", err)
	}

	// Unmarshal back
	var decoded Portfolio
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal portfolio: %v", err)
	}

	// Verify fields
	if decoded.AccountID != portfolio.AccountID {
		t.Errorf("AccountID mismatch: got %s, want %s", decoded.AccountID, portfolio.AccountID)
	}
	if len(decoded.Positions) != len(portfolio.Positions) {
		t.Errorf("Positions length mismatch: got %d, want %d", len(decoded.Positions), len(portfolio.Positions))
	}
	if decoded.TotalValue != portfolio.TotalValue {
		t.Errorf("TotalValue mismatch: got %f, want %f", decoded.TotalValue, portfolio.TotalValue)
	}
}

func TestPortfolio_EmptyPositions(t *testing.T) {
	portfolio := Portfolio{
		AccountID:   "acc-456",
		Cash:        100000.00,
		BuyingPower: 400000.00,
		Positions:   []Position{},
		TotalValue:  100000.00,
		TotalPL:     0.00,
		LastUpdated: time.Date(2026, 2, 13, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(portfolio)
	if err != nil {
		t.Fatalf("Failed to marshal empty portfolio: %v", err)
	}

	var decoded Portfolio
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal empty portfolio: %v", err)
	}

	if len(decoded.Positions) != 0 {
		t.Errorf("Positions should be empty array, got length %d", len(decoded.Positions))
	}
}
