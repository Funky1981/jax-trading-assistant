package app

import (
	"context"
	"testing"
)

func TestTradingGuardRecordFailureHalts(t *testing.T) {
	guard := NewTradingGuard(TradingGuardConfig{MaxConsecutiveLosses: 3}, nil)
	guard.RecordFailure(context.Background(), "test_stage", nil)

	allowed, reason := guard.CanTrade()
	if allowed {
		t.Fatalf("expected trading to be halted")
	}
	if reason == "" {
		t.Fatalf("expected halt reason to be set")
	}
}

func TestTradingGuardLossStreak(t *testing.T) {
	guard := NewTradingGuard(TradingGuardConfig{MaxConsecutiveLosses: 3}, nil)

	guard.RecordOutcome(context.Background(), -1)
	guard.RecordOutcome(context.Background(), -1)
	guard.RecordOutcome(context.Background(), -1)

	allowed, _ := guard.CanTrade()
	if !allowed {
		t.Fatalf("expected trading to continue before exceeding loss threshold")
	}

	guard.RecordOutcome(context.Background(), -1)
	allowed, _ = guard.CanTrade()
	if allowed {
		t.Fatalf("expected trading to halt after loss threshold exceeded")
	}
}

func TestTradingGuardWinResetsLosses(t *testing.T) {
	guard := NewTradingGuard(TradingGuardConfig{MaxConsecutiveLosses: 3}, nil)

	guard.RecordOutcome(context.Background(), -1)
	guard.RecordOutcome(context.Background(), -1)
	guard.RecordOutcome(context.Background(), 5)

	status := guard.Status()
	if status.ConsecutiveLosses != 0 {
		t.Fatalf("expected losses to reset after win, got %d", status.ConsecutiveLosses)
	}
}
