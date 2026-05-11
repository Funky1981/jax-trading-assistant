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
	"jax-trading-assistant/libs/contracts/domain"
)

// instanceRecord is a minimal view of a strategy_instance row needed by the watcher.
type instanceRecord struct {
	ID                 uuid.UUID
	Name               string
	StrategyTypeID     string
	Enabled            bool
	SessionTimezone    string
	FlattenByCloseTime string
	Symbols            []string
}

// loadEnabledInstances reads all enabled strategy instances from the DB.
func loadEnabledInstances(ctx context.Context, pool *pgxpool.Pool) ([]instanceRecord, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, strategy_type_id, enabled, session_timezone, flatten_by_close_time,
		       COALESCE(config::text, '{}') AS config_json
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
		var configJSON string
		if err := rows.Scan(&r.ID, &r.Name, &r.StrategyTypeID, &r.Enabled, &r.SessionTimezone, &r.FlattenByCloseTime, &configJSON); err != nil {
			return nil, err
		}
		normalizedConfig, err := normalizeInstanceConfig(json.RawMessage(configJSON))
		if err != nil {
			log.Printf("watcher: normalize config failed for %s: %v", r.Name, err)
			continue
		}
		r.Symbols = normalizedSymbolListFromRaw(normalizedConfig)
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
	blockedCount := 0
	postFlatten := isPastInstanceFlatten(inst, time.Now().UTC())
	etfPolicy, _, _ := loadActiveETFInstrumentPolicy()
	for _, sig := range signals {
		// Only include signals from the strategy type that matches this instance.
		if !strings.EqualFold(sig.StrategyID, inst.StrategyTypeID) {
			continue
		}
		if decision := evaluateETFPhase1Eligibility(etfPolicy, sig.Symbol, "paper"); decision.IsETF && !decision.Allowed {
			if blocked := persistBlockedSignal(ctx, svc, inst, sig, decision.ReasonCode, decision.Reason); blocked != nil {
				blockedCount++
			}
			continue
		}
		signalType := strings.ToUpper(strings.TrimSpace(sig.Type))
		switch {
		case sig.Confidence < 0.60:
			if blocked := persistBlockedSignal(ctx, svc, inst, sig, "low_confidence", fmt.Sprintf("signal confidence %.2f below 0.60 threshold", sig.Confidence)); blocked != nil {
				blockedCount++
			}
			continue
		case signalType == "HOLD" || signalType == "":
			if blocked := persistBlockedSignal(ctx, svc, inst, sig, "non_actionable_signal", "signal is HOLD or missing actionable direction"); blocked != nil {
				blockedCount++
			}
			continue
		}
		if postFlatten {
			if blocked := persistBlockedSignal(ctx, svc, inst, sig, "post_flatten", fmt.Sprintf("instance flatten window %s %s already passed", inst.SessionTimezone, inst.FlattenByCloseTime)); blocked != nil {
				blockedCount++
			}
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
			SignalID:           sig.ID,
			StrategyID:         sig.StrategyID,
			ArtifactID:         sig.ArtifactID,
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
				if blocked := persistBlockedSignal(ctx, svc, inst, sig, "duplicate_candidate", "an open candidate already exists for this instance, symbol, and session"); blocked != nil {
					blockedCount++
				}
				continue
			}
			log.Printf("watcher: propose error for %s/%s: %v", inst.Name, sig.Symbol, err)
			continue
		}

		publishCandidateEvent("candidate.detected", candidate)
		if err := svc.Qualify(ctx, candidate.ID); err != nil {
			log.Printf("watcher: qualify error for candidate %s: %v", candidate.ID, err)
			continue
		}
		if qualified, err := svc.GetByID(ctx, candidate.ID); err == nil {
			publishCandidateEvent("candidate.qualified", qualified)
		}
		proposeCount++
		log.Printf("watcher: proposed candidate %s for %s/%s (conf=%.2f)",
			candidate.ID, inst.Name, sig.Symbol, sig.Confidence)
	}
	if proposeCount > 0 {
		log.Printf("watcher: instance %q proposed %d candidate(s)", inst.Name, proposeCount)
	}
	if blockedCount > 0 {
		log.Printf("watcher: instance %q recorded %d blocked candidate(s)", inst.Name, blockedCount)
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

func persistBlockedSignal(ctx context.Context, svc *candidatesmod.Service, inst instanceRecord, sig domain.Signal, reasonCode, reason string) *candidatesmod.Candidate {
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
	reasoning := strings.TrimSpace(sig.Reason)
	if reasoning == "" {
		reasoning = reason
	}
	blocked, err := svc.CreateBlocked(ctx, candidatesmod.BlockRequest{
		StrategyInstanceID: inst.ID,
		SignalID:           sig.ID,
		StrategyID:         sig.StrategyID,
		ArtifactID:         sig.ArtifactID,
		Symbol:             sig.Symbol,
		SignalType:         strings.ToUpper(strings.TrimSpace(sig.Type)),
		EntryPrice:         entryPrice,
		StopLoss:           stopLoss,
		TakeProfit:         takeProfit,
		Confidence:         &conf,
		Reasoning:          &reasoning,
		DataProvenance:     "signal-generator",
		ReasonCode:         reasonCode,
		Reason:             reason,
		TTL:                4 * time.Hour,
	})
	if err != nil {
		log.Printf("watcher: blocked candidate persist failed for %s/%s (%s): %v", inst.Name, sig.Symbol, reasonCode, err)
		return nil
	}
	publishCandidateEvent("candidate.blocked", blocked)
	return blocked
}

func publishCandidateEvent(eventType string, candidate *candidatesmod.Candidate) {
	if candidate == nil {
		return
	}
	publishEvent(eventType, candidate)
}

func isPastInstanceFlatten(inst instanceRecord, now time.Time) bool {
	tz := strings.TrimSpace(inst.SessionTimezone)
	if tz == "" {
		tz = "America/New_York"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	current := now.In(loc)
	flattenAt := strings.TrimSpace(inst.FlattenByCloseTime)
	if flattenAt == "" {
		flattenAt = "15:55"
	}
	var hh, mm int
	if _, err := fmt.Sscanf(flattenAt, "%d:%d", &hh, &mm); err != nil {
		return false
	}
	flattenTs := time.Date(current.Year(), current.Month(), current.Day(), hh, mm, 0, 0, loc)
	return !current.Before(flattenTs)
}
