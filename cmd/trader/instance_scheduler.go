package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
)

// instanceRecord is a minimal view of a strategy_instance row needed by the watcher.
type instanceRecord struct {
	ID                 uuid.UUID
	Name               string
	StrategyTypeID     string
	Enabled            bool
	SessionTimezone    string
	FlattenByCloseTime string
}

// loadEnabledInstances reads all enabled strategy instances from the DB.
func loadEnabledInstances(ctx context.Context, pool *pgxpool.Pool) ([]instanceRecord, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, strategy_type_id, enabled, session_timezone, flatten_by_close_time
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
		if err := rows.Scan(&r.ID, &r.Name, &r.StrategyTypeID, &r.Enabled, &r.SessionTimezone, &r.FlattenByCloseTime); err != nil {
			return nil, err
		}
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

// scanInstance evaluates a single strategy instance and proposes a candidate trade
// if the conditions are met.  In production this would call the signal generator;
// here we define the contract so the watcher can plug real logic in.
func scanInstance(ctx context.Context, svc *candidatesmod.Service, inst instanceRecord) {
	// TODO: integrate real signal-generation logic per strategy type.
	// For now this stub logs the scan attempt and returns without creating candidates.
	// Replace with: outcome := sigGen.EvaluateInstance(ctx, inst.ID)
	log.Printf("watcher: scanning instance %q (%s)", inst.Name, inst.StrategyTypeID)
	_ = svc // svc.Propose(ctx, req) once real signal output is available
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
