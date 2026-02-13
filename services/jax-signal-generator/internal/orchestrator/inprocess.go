package orchestrator

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// TriggerFunc allows wiring an in-process orchestration path without HTTP.
// This is used by phase-1 migration to keep signal generation logic unchanged
// while replacing transport.
type TriggerFunc func(ctx context.Context, signalID uuid.UUID, symbol, context string) (uuid.UUID, error)

// InProcessTrigger implements the same TriggerOrchestration contract as the
// HTTP client, but delegates directly to an in-process function.
type InProcessTrigger struct {
	trigger TriggerFunc
}

func NewInProcessTrigger(trigger TriggerFunc) (*InProcessTrigger, error) {
	if trigger == nil {
		return nil, fmt.Errorf("in-process trigger: trigger func is required")
	}
	return &InProcessTrigger{trigger: trigger}, nil
}

func (t *InProcessTrigger) TriggerOrchestration(ctx context.Context, signalID uuid.UUID, symbol, contextStr string) (uuid.UUID, error) {
	return t.trigger(ctx, signalID, symbol, contextStr)
}
