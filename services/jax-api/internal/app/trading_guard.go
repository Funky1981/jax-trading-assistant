package app

import (
	"context"
	"fmt"
	"log"
	"sync"

	"jax-trading-assistant/services/jax-api/internal/domain"
)

const defaultMaxConsecutiveLosses = 3

type TradingGuardConfig struct {
	MaxConsecutiveLosses int
}

type TradingGuardStatus struct {
	Halted               bool   `json:"halted"`
	Reason               string `json:"reason,omitempty"`
	ConsecutiveLosses    int    `json:"consecutiveLosses"`
	MaxConsecutiveLosses int    `json:"maxConsecutiveLosses"`
}

// TradingGuard halts trading when failures or loss streaks exceed limits.
type TradingGuard struct {
	mu                   sync.Mutex
	halted               bool
	haltReason           string
	consecutiveLosses    int
	maxConsecutiveLosses int
	audit                *AuditLogger
}

func NewTradingGuard(cfg TradingGuardConfig, audit *AuditLogger) *TradingGuard {
	maxLosses := cfg.MaxConsecutiveLosses
	if maxLosses <= 0 {
		maxLosses = defaultMaxConsecutiveLosses
	}
	return &TradingGuard{
		maxConsecutiveLosses: maxLosses,
		audit:                audit,
	}
}

func (g *TradingGuard) CanTrade() (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.halted {
		return false, g.haltReason
	}
	return true, ""
}

func (g *TradingGuard) Status() TradingGuardStatus {
	g.mu.Lock()
	defer g.mu.Unlock()
	return TradingGuardStatus{
		Halted:               g.halted,
		Reason:               g.haltReason,
		ConsecutiveLosses:    g.consecutiveLosses,
		MaxConsecutiveLosses: g.maxConsecutiveLosses,
	}
}

func (g *TradingGuard) RecordFailure(ctx context.Context, stage string, err error) {
	if g == nil {
		return
	}
	g.mu.Lock()
	message := fmt.Sprintf("failure in %s", stage)
	if err != nil {
		message = fmt.Sprintf("failure in %s: %v", stage, err)
	}
	g.halted = true
	g.haltReason = message
	g.mu.Unlock()

	if g.audit != nil {
		if err := g.audit.LogDecision(ctx, "trading_guard_halt", domain.AuditOutcomeError, map[string]any{
			"stage":  stage,
			"reason": message,
		}, err); err != nil {
			// Log the audit error but don't fail the operation
			log.Printf("warning: failed to log trading guard halt to audit: %v", err)
		}
	}
}

func (g *TradingGuard) RecordOutcome(ctx context.Context, pnl float64) {
	if g == nil {
		return
	}
	g.mu.Lock()
	if pnl < 0 {
		g.consecutiveLosses++
	} else {
		g.consecutiveLosses = 0
	}
	shouldHalt := g.consecutiveLosses > g.maxConsecutiveLosses
	if shouldHalt {
		g.halted = true
		g.haltReason = fmt.Sprintf("loss streak exceeded: %d losses", g.consecutiveLosses)
	}
	status := g.halted
	reason := g.haltReason
	losses := g.consecutiveLosses
	g.mu.Unlock()

	if g.audit != nil && status {
		_ = g.audit.LogDecision(ctx, "trading_guard_halt", domain.AuditOutcomeError, map[string]any{
			"stage":             "loss_streak",
			"reason":            reason,
			"consecutiveLosses": losses,
		}, nil)
	}
}
