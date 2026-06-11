package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type opportunityScanner struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

type opportunityScannerResult struct {
	Promoted int
	Skipped  int
	Disabled bool
}

func newOpportunityScanner(pool *pgxpool.Pool) *opportunityScanner {
	return &opportunityScanner{
		pool: pool,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func startOpportunityScanner(ctx context.Context, pool *pgxpool.Pool) {
	scanner := newOpportunityScanner(pool)
	for {
		state, err := loadAIScannerState(ctx, pool)
		interval := 5 * time.Minute
		if err == nil && state.IntervalSeconds > 0 {
			interval = time.Duration(state.IntervalSeconds) * time.Second
		}

		if _, err := scanner.ScanOnce(ctx); err != nil {
			log.Printf("opportunity scanner: %v", err)
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *opportunityScanner) ScanOnce(ctx context.Context) (opportunityScannerResult, error) {
	if s.pool == nil {
		return opportunityScannerResult{Disabled: true}, nil
	}
	state, err := loadAIScannerState(ctx, s.pool)
	if err != nil {
		return opportunityScannerResult{}, err
	}
	if !state.Enabled || !opportunityScannerRuntimeEnabled() {
		state.Status = deriveScannerStatus(state)
		if !opportunityScannerRuntimeEnabled() {
			state.Status = "disabled"
		}
		return opportunityScannerResult{Disabled: true}, saveAIScannerState(ctx, s.pool, state)
	}

	promoter := newWorldMonitorOpportunityPromoter(s.pool)
	rows, err := promoter.loadPromotionRows(ctx, worldMonitorPromoterMaxLimit)
	if err != nil {
		return opportunityScannerResult{}, err
	}

	result := opportunityScannerResult{}
	allowedSymbols := symbolSet(state.Symbols)
	for _, row := range rows {
		if row.Confidence < state.MinimumConfidence {
			result.Skipped++
			continue
		}
		if len(allowedSymbols) > 0 && !rowMatchesAllowedSymbols(row, allowedSymbols) {
			result.Skipped++
			continue
		}
		if _, err := promoter.promoteRow(ctx, row); err != nil {
			result.Skipped++
			continue
		}
		result.Promoted++
	}

	now := s.now()
	next := now.Add(time.Duration(state.IntervalSeconds) * time.Second)
	state.Status = "ready"
	state.LastScanCompletedAt = &now
	state.NextScanAt = &next
	if err := saveAIScannerState(ctx, s.pool, state); err != nil {
		return result, err
	}
	return result, nil
}

func opportunityScannerRuntimeEnabled() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_LIVE_TRADING")), "true") {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("JAX_RUNTIME_MODE")))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("JAX_TRADER_RUNTIME_MODE")))
	}
	return mode == "paper"
}

func symbolSet(symbols []string) map[string]bool {
	out := map[string]bool{}
	for _, symbol := range symbols {
		trimmed := strings.ToUpper(strings.TrimSpace(symbol))
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func rowMatchesAllowedSymbols(row worldMonitorInboxPromotionRow, allowed map[string]bool) bool {
	for _, symbol := range normalizeSymbols("", row.PossibleAffectedETFs) {
		if allowed[symbol] {
			return true
		}
	}
	return false
}
