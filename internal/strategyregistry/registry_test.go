package strategyregistry_test

import (
	"context"
	"testing"
	"time"

	"jax-trading-assistant/internal/strategyregistry"

	"github.com/google/uuid"
)

func TestDocumentStruct(t *testing.T) {
	// Basic test that Document struct can be instantiated correctly
	doc := strategyregistry.Document{
		DocID:      uuid.New(),
		DocType:    strategyregistry.DocTypeStrategy,
		RelPath:    "strategies/test.md",
		Title:      "Test Strategy",
		Version:    "1.0",
		Status:     strategyregistry.StatusApproved,
		CreatedUTC: time.Now().UTC(),
		UpdatedUTC: time.Now().UTC(),
		Tags:       []string{"test", "example"},
		Sha256:     "abc123",
		Markdown:   "# Test\n\nContent here",
	}

	if doc.DocType != "strategy" {
		t.Errorf("expected doc_type 'strategy', got %s", doc.DocType)
	}

	if doc.Status != "approved" {
		t.Errorf("expected status 'approved', got %s", doc.Status)
	}

	if len(doc.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(doc.Tags))
	}
}

func TestConstants(t *testing.T) {
	// Verify constants match expected values
	if strategyregistry.DocTypeStrategy != "strategy" {
		t.Error("DocTypeStrategy mismatch")
	}
	if strategyregistry.DocTypeAntiPattern != "anti-pattern" {
		t.Error("DocTypeAntiPattern mismatch")
	}
	if strategyregistry.StatusApproved != "approved" {
		t.Error("StatusApproved mismatch")
	}
}

// Integration tests require a running database.
// Run with: go test -tags=integration ./internal/strategyregistry/...

func TestRegistry_RequiresPool(t *testing.T) {
	// Registry with nil pool should not panic on creation
	reg := strategyregistry.New(nil)
	if reg == nil {
		t.Error("New(nil) should return a non-nil Registry")
	}
}

func TestRegistry_HealthCheck_NoPool(t *testing.T) {
	// This tests graceful handling when pool is nil
	// In real usage, pool would never be nil, but we verify no panic
	ctx := context.Background()

	// Note: Can't easily test without a real pool
	// This is a placeholder for when we add mock support
	_ = ctx
}
