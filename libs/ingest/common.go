package ingest

import (
	"context"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
)

// RetentionConfig configures observation retention logic
type RetentionConfig struct {
	SignificanceThreshold float64
}

// RetentionResult contains statistics about retention operation
type RetentionResult struct {
	Retained int
	Skipped  int
}

// MemoryStore defines the interface for storing observations
type MemoryStore interface {
	Ping(context.Context) error
	Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error)
}

// MemoryAdapter wraps UTCP memory service for ingestion
type MemoryAdapter struct {
	svc *utcp.MemoryService
}

// NewMemoryAdapter creates a new memory adapter
func NewMemoryAdapter(client *utcp.UTCPClient) *MemoryAdapter {
	return &MemoryAdapter{
		svc: utcp.NewMemoryService(client),
	}
}

// Ping checks connectivity
func (m *MemoryAdapter) Ping(ctx context.Context) error {
	// UTCP doesn't have a ping, just return success
	return nil
}

// Retain stores a memory item
func (m *MemoryAdapter) Retain(ctx context.Context, bank string, item contracts.MemoryItem) (contracts.MemoryID, error) {
	out, err := m.svc.Retain(ctx, contracts.MemoryRetainRequest{Bank: bank, Item: item})
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

// Recall retrieves memory items matching the query
func (m *MemoryAdapter) Recall(ctx context.Context, bank string, query contracts.MemoryQuery) ([]contracts.MemoryItem, error) {
	out, err := m.svc.Recall(ctx, contracts.MemoryRecallRequest{Bank: bank, Query: query})
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

// Reflect generates reflection items based on params
func (m *MemoryAdapter) Reflect(ctx context.Context, bank string, params contracts.ReflectionParams) ([]contracts.MemoryItem, error) {
	out, err := m.svc.Reflect(ctx, contracts.MemoryReflectRequest{Bank: bank, Params: params})
	if err != nil {
		return nil, err
	}
	return out.Items, nil
}

// CreateContext creates a context with observability metadata
func CreateContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	ctx = observability.WithRunInfo(ctx, observability.RunInfo{
		RunID:  observability.NewRunID(),
		TaskID: "ingest",
	})
	return ctx, cancel
}
