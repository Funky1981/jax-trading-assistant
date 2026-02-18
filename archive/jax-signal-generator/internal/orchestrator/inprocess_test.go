package orchestrator

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewInProcessTrigger(t *testing.T) {
	_, err := NewInProcessTrigger(nil)
	if err == nil {
		t.Fatalf("expected error when trigger func is nil")
	}

	expected := uuid.New()
	tr, err := NewInProcessTrigger(func(_ context.Context, _ uuid.UUID, _ string, _ string) (uuid.UUID, error) {
		return expected, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := tr.TriggerOrchestration(context.Background(), uuid.New(), "AAPL", "ctx")
	if err != nil {
		t.Fatalf("unexpected trigger error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}
