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

	// LinkOrchestration links a signal to an orchestration run
	LinkOrchestration(ctx context.Context, signalID, runID uuid.UUID) error

	// GetRecommendations returns signals with orchestration analysis (pending recommendations)
	GetRecommendations(ctx context.Context, limit, offset int) (*RecommendationListResponse, error)

	// GetRecommendation returns a single recommendation with full details
	GetRecommendation(ctx context.Context, signalID uuid.UUID) (*Recommendation, error)
}

// OrchestrationRun represents an AI orchestration run from the database
type OrchestrationRun struct {
	ID               uuid.UUID  `json:"id"`
	Symbol           string     `json:"symbol"`
	TriggerType      string     `json:"trigger_type"`
	TriggerID        *uuid.UUID `json:"trigger_id,omitempty"`
	AgentSuggestion  *string    `json:"agent_suggestion,omitempty"`
	Confidence       *float64   `json:"confidence,omitempty"`
	Reasoning        *string    `json:"reasoning,omitempty"`
	MemoriesRecalled int        `json:"memories_recalled"`
	Status           string     `json:"status"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	Error            *string    `json:"error,omitempty"`
}

// Recommendation combines a signal with its AI analysis
type Recommendation struct {
	Signal   Signal            `json:"signal"`
	Analysis *OrchestrationRun `json:"ai_analysis,omitempty"`
}

// RecommendationListResponse contains the list of recommendations with pagination metadata
type RecommendationListResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
	Total           int              `json:"total"`
	Limit           int              `json:"limit"`
	Offset          int              `json:"offset"`
}
