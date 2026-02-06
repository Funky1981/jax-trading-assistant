package app

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Signal represents a trading signal from the database
type Signal struct {
	ID                 uuid.UUID  `json:"id"`
	Symbol             string     `json:"symbol"`
	StrategyID         string     `json:"strategy_id"`
	SignalType         string     `json:"signal_type"`
	Confidence         float64    `json:"confidence"`
	EntryPrice         *float64   `json:"entry_price,omitempty"`
	StopLoss           *float64   `json:"stop_loss,omitempty"`
	TakeProfit         *float64   `json:"take_profit,omitempty"`
	Reasoning          *string    `json:"reasoning,omitempty"`
	GeneratedAt        time.Time  `json:"generated_at"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	Status             string     `json:"status"`
	OrchestrationRunID *uuid.UUID `json:"orchestration_run_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// SignalApproval represents an approval/rejection action on a signal
type SignalApproval struct {
	SignalID          uuid.UUID `json:"signal_id"`
	Approved          bool      `json:"approved"`
	ApprovedBy        string    `json:"approved_by"`
	ModificationNotes *string   `json:"modification_notes,omitempty"`
	RejectionReason   *string   `json:"rejection_reason,omitempty"`
}

// SignalListResponse contains the list of signals with pagination metadata
type SignalListResponse struct {
	Signals []Signal `json:"signals"`
	Total   int      `json:"total"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
}

// SignalStore defines the interface for signal data access
type SignalStore interface {
	// ListSignals returns a list of signals with filtering and pagination
	ListSignals(ctx context.Context, status, symbol, strategy string, limit, offset int) (*SignalListResponse, error)

	// GetSignal returns a single signal by ID
	GetSignal(ctx context.Context, id uuid.UUID) (*Signal, error)

	// ApproveSignal approves a signal and creates a trade_approvals record
	ApproveSignal(ctx context.Context, id uuid.UUID, approvedBy, modificationNotes string) (*Signal, error)

	// RejectSignal rejects a signal and creates a trade_approvals record with approved=false
	RejectSignal(ctx context.Context, id uuid.UUID, approvedBy, rejectionReason string) (*Signal, error)
}
