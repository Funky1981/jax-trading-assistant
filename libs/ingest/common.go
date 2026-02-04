package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"jax-trading-assistant/libs/contracts"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/utcp"
)

// DexterPayload represents the structure from Dexter ingestion
type DexterPayload struct {
	GeneratedAt  time.Time           `json:"generated_at"`
	Observations []DexterObservation `json:"observations"`
}

// DexterObservation represents a single observation from Dexter
type DexterObservation struct {
	Type           string    `json:"type"`
	Timestamp      time.Time `json:"timestamp"`
	Symbol         string    `json:"symbol,omitempty"`
	Price          float64   `json:"price,omitempty"`
	Volume         int64     `json:"volume,omitempty"`
	ImpactEstimate float64   `json:"impact_estimate,omitempty"`
	Confidence     float64   `json:"confidence,omitempty"`
	Bookmarked     bool      `json:"bookmarked,omitempty"`
	Data           any       `json:"data,omitempty"`
}

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

// ReadPayload reads and parses a Dexter payload from file or stdin
func ReadPayload(path string) (DexterPayload, error) {
	var payload DexterPayload
	reader, err := OpenInput(path)
	if err != nil {
		return payload, err
	}
	defer reader.Close()

	raw, err := io.ReadAll(reader)
	if err != nil {
		return payload, err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return payload, errors.New("dexter payload is empty")
	}

	// Try full payload format first
	if err := json.Unmarshal(raw, &payload); err == nil && len(payload.Observations) > 0 {
		return payload, nil
	}

	// Fall back to array of observations
	var observations []DexterObservation
	if err := json.Unmarshal(raw, &observations); err != nil {
		return payload, fmt.Errorf("parse dexter payload: %w", err)
	}
	payload.Observations = observations
	return payload, nil
}

// OpenInput opens a file for reading, or returns stdin if path is empty
func OpenInput(path string) (io.ReadCloser, error) {
	if path == "" {
		return io.NopCloser(os.Stdin), nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// NormalizeTimestamps ensures all observations have valid timestamps
func NormalizeTimestamps(payload *DexterPayload) {
	if payload.GeneratedAt.IsZero() {
		payload.GeneratedAt = time.Now().UTC()
	}
	for i := range payload.Observations {
		if payload.Observations[i].Timestamp.IsZero() {
			payload.Observations[i].Timestamp = payload.GeneratedAt
		}
	}
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
