package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	signalgenerator "jax-trading-assistant/internal/trader/signalgenerator"
)

// instanceRecord is a minimal view of a strategy_instance row needed by the watcher.
type instanceRecord struct {
	ID                 uuid.UUID
	Name               string
	StrategyTypeID     string
	Enabled            bool
	SessionTimezone    string
	FlattenByCloseTime string
	Symbols            []string // extracted from config JSONB
}

// loadEnabledInstances reads all enabled strategy instances from the DB.
func loadEnabledInstances(ctx context.Context, pool *pgxpool.Pool) ([]instanceRecord, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, strategy_type_id, enabled, session_timezone, flatten_by_close_time,
		       COALESCE(config->>'symbols', '[]') AS symbols_json
		FROM strategy_instances
		WHERE enabled = TRUE
	`)
	if err != nil {
		return nil, fmt.Errorf("loadEnabledInstances: %w", err)
	}
	defer rows.Close()
	var out []instanceRecord
	for rows.Next() {
		var r instanceRecord
		var symbolsJSON string
		if err := rows.Scan(&r.ID, &r.Name, &r.StrategyTypeID, &r.Enabled, &r.SessionTimezone, &r.FlattenByCloseTime, &symbolsJSON); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(symbolsJSON), &r.Symbols)
		out = append(out, r)
	}
	return out, rows.Err()
}

// checkKillSwitch returns true when a global kill-switch flag is set, preventing
// any new candidates from being generated.
func checkKillSwitch(ctx context.Context, pool *pgxpool.Pool) bool {
	var val string
	err := pool.QueryRow(ctx, `
		SELECT value FROM config_flags WHERE key = 'global_kill_switch' LIMIT 1
	`).Scan(&val)
	if err != nil {
		// table may not exist yet — treat as switch OFF
		return false
	}
	return strings.EqualFold(strings.TrimSpace(val), "true")
}

// scanInstance evaluates a single strategy instance and proposes candidate trades
// for any signals the signal generator produces with sufficient confidence.
// Duplicate candidates are silently skipped (ErrDuplicateCandidate).
func scanInstance(ctx context.Context, svc *candidatesmod.Service, sigGen *signalgenerator.InProcessSignalGenerator, inst instanceRecord) {
	if len(inst.Symbols) == 0 {
		log.Printf("watcher: instance %q has no symbols configured, skipping", inst.Name)
		return
	}

	// Generate signals for this instance's symbols using the in-process signal generator.
	signals, err := sigGen.GenerateSignals(ctx, inst.Symbols)
	if err != nil {
		log.Printf("watcher: GenerateSignals error for instance %q: %v", inst.Name, err)
		return
	}

	proposeCount := 0
	for _, sig := range signals {
		// Only include signals from the strategy type that matches this instance.
		if !strings.EqualFold(sig.StrategyID, inst.StrategyTypeID) {
			continue
		}
		// Require a minimum confidence threshold.
		if sig.Confidence < 0.60 {
			continue
		}
		// Convert signal type ("buy"/"sell") to uppercase convention.
		signalType := strings.ToUpper(sig.Type)
		if signalType == "HOLD" || signalType == "" {
			continue
		}

		var entryPrice, stopLoss *float64
		if sig.EntryPrice > 0 {
			v := sig.EntryPrice
			entryPrice = &v
		}
		if sig.StopLoss > 0 {
			v := sig.StopLoss
			stopLoss = &v
		}
		var takeProfit *float64
		if len(sig.TakeProfit) > 0 && sig.TakeProfit[0] > 0 {
			v := sig.TakeProfit[0]
			takeProfit = &v
		}
		conf := sig.Confidence
		reasoning := sig.Reason

		req := candidatesmod.ProposalRequest{
			StrategyInstanceID: inst.ID,
			Symbol:             sig.Symbol,
			SignalType:         signalType,
			EntryPrice:         entryPrice,
			StopLoss:           stopLoss,
			TakeProfit:         takeProfit,
			Confidence:         &conf,
			Reasoning:          &reasoning,
			DataProvenance:     "signal-generator",
			TTL:                4 * time.Hour,
		}
		candidate, err := svc.Propose(ctx, req)
		if err != nil {
			if errors.Is(err, candidatesmod.ErrDuplicateCandidate) {
				log.Printf("watcher: duplicate candidate skipped for %s/%s", inst.Name, sig.Symbol)
				continue
			}
			log.Printf("watcher: propose error for %s/%s: %v", inst.Name, sig.Symbol, err)
			continue
		}

		// Qualify immediately — moves to awaiting_approval.
		if err := svc.Qualify(ctx, candidate.ID); err != nil {
			log.Printf("watcher: qualify error for candidate %s: %v", candidate.ID, err)
			continue
		}
		proposeCount++
		log.Printf("watcher: proposed candidate %s for %s/%s (conf=%.2f)",
			candidate.ID, inst.Name, sig.Symbol, sig.Confidence)
	}
	if proposeCount > 0 {
		log.Printf("watcher: instance %q proposed %d candidate(s)", inst.Name, proposeCount)
	}
}

// instanceScheduler tracks per-instance scan state and throttles calls.
type instanceScheduler struct {
	lastScan map[uuid.UUID]time.Time
	interval time.Duration
}

func newInstanceScheduler(interval time.Duration) *instanceScheduler {
	return &instanceScheduler{
		lastScan: make(map[uuid.UUID]time.Time),
		interval: interval,
	}
}

// due returns true when the instance is overdue for a scan.
func (s *instanceScheduler) due(id uuid.UUID) bool {
	last, ok := s.lastScan[id]
	return !ok || time.Since(last) >= s.interval
}

// mark records the current time as the last scan time for an instance.
func (s *instanceScheduler) mark(id uuid.UUID) {
	s.lastScan[id] = time.Now()
}
