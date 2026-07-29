package eventdecisions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Replayer struct {
	Store     *Store
	Evaluator Evaluator
	Now       func() time.Time
}

// RunTx evaluates and persists events in a caller-owned transaction. The caller
// is responsible for committing or rolling back the transaction together with
// any upstream ingestion and consumer-cursor update.
func (r Replayer) RunTx(ctx context.Context, tx pgx.Tx, events []Event) (Summary, error) {
	if tx == nil {
		return Summary{}, fmt.Errorf("transactional event replay requires a transaction")
	}
	now := r.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	started := now().UTC()
	summary := Summary{DryRun: false, RulesetVersion: r.Evaluator.Ruleset.Version, Selected: len(events), StartedAt: started, Excluded: []Exclusion{}, Failures: []Failure{}, Outcomes: []EventOutcome{}}
	type evaluated struct {
		event       Event
		result      Result
		fingerprint string
		replayID    string
	}
	ready := []evaluated{}
	for _, event := range events {
		if ok, reason := Eligible(event); !ok {
			summary.Excluded = append(summary.Excluded, Exclusion{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Reason: reason})
			continue
		}
		result, err := r.Evaluator.Evaluate(event)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Error: err.Error()})
			continue
		}
		fingerprint, replayID, err := replayIdentity(event, r.Evaluator.Ruleset.Version)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Error: err.Error()})
			continue
		}
		summary.Eligible++
		incrementDecision(&summary, result.Decision)
		summary.Outcomes = append(summary.Outcomes, EventOutcome{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Proposed: result})
		ready = append(ready, evaluated{event: event, result: result, fingerprint: fingerprint, replayID: replayID})
	}
	if len(summary.Failures) > 0 {
		summary.CompletedAt = now().UTC()
		return summary, fmt.Errorf("replay evaluation failed closed with %d failure(s)", len(summary.Failures))
	}
	decisionAt := now().UTC()
	for index, item := range ready {
		persisted, reused, err := persistDecision(ctx, tx, item.event, item.result, r.Evaluator.Ruleset, decisionAt, item.fingerprint, item.replayID)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("transactional event replay failed: %w", err)
		}
		summary.Outcomes[index].Persisted = &persisted
		summary.Outcomes[index].Reused = reused
		if reused {
			summary.DecisionsReused++
		} else {
			summary.DecisionsCreated++
		}
		if item.result.Decision == DecisionCandidate {
			summary.CandidatesReused++
		}
	}
	summary.CompletedAt = now().UTC()
	return summary, nil
}

func (r Replayer) Run(ctx context.Context, events []Event, dryRun bool) (Summary, error) {
	now := r.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	started := now().UTC()
	summary := Summary{DryRun: dryRun, RulesetVersion: r.Evaluator.Ruleset.Version, Selected: len(events), StartedAt: started, Excluded: []Exclusion{}, Failures: []Failure{}, Outcomes: []EventOutcome{}}
	type evaluated struct {
		event       Event
		result      Result
		fingerprint string
		replayID    string
	}
	ready := []evaluated{}
	for _, event := range events {
		if ok, reason := Eligible(event); !ok {
			summary.Excluded = append(summary.Excluded, Exclusion{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Reason: reason})
			continue
		}
		result, err := r.Evaluator.Evaluate(event)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Error: err.Error()})
			continue
		}
		fingerprint, replayID, err := replayIdentity(event, r.Evaluator.Ruleset.Version)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Error: err.Error()})
			continue
		}
		summary.Eligible++
		incrementDecision(&summary, result.Decision)
		summary.Outcomes = append(summary.Outcomes, EventOutcome{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Proposed: result})
		ready = append(ready, evaluated{event: event, result: result, fingerprint: fingerprint, replayID: replayID})
	}
	if len(summary.Failures) > 0 {
		summary.CompletedAt = now().UTC()
		return summary, fmt.Errorf("replay evaluation failed closed with %d failure(s)", len(summary.Failures))
	}
	if dryRun {
		summary.CompletedAt = now().UTC()
		return summary, nil
	}
	if r.Store == nil {
		return summary, fmt.Errorf("persisted replay requires a store")
	}
	tx, err := r.Store.Begin(ctx)
	if err != nil {
		return summary, fmt.Errorf("begin event decision replay: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	decisionAt := now().UTC()
	for index, item := range ready {
		persisted, reused, err := persistDecision(ctx, tx, item.event, item.result, r.Evaluator.Ruleset, decisionAt, item.fingerprint, item.replayID)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("persisted replay rolled back: %w", err)
		}
		summary.Outcomes[index].Persisted = &persisted
		summary.Outcomes[index].Reused = reused
		if reused {
			summary.DecisionsReused++
		} else {
			summary.DecisionsCreated++
		}
		if item.result.Decision == DecisionCandidate {
			summary.CandidatesReused++
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return summary, fmt.Errorf("commit event decision replay: %w", err)
	}
	summary.CompletedAt = now().UTC()
	return summary, nil
}

func replayIdentity(event Event, rulesetVersion string) (string, string, error) {
	input := struct {
		Event   Event  `json:"event"`
		Ruleset string `json:"ruleset"`
	}{Event: event, Ruleset: rulesetVersion}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", "", fmt.Errorf("fingerprint replay input: %w", err)
	}
	sum := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(sum[:])
	replayRaw := sha256.Sum256([]byte(rulesetVersion + ":" + event.InboxID.String() + ":" + fingerprint))
	return fingerprint, "gedr_" + hex.EncodeToString(replayRaw[:]), nil
}

func incrementDecision(summary *Summary, decision Decision) {
	switch decision {
	case DecisionNoTrade:
		summary.NoTrade++
	case DecisionWatch:
		summary.Watch++
	case DecisionCandidate:
		summary.Candidate++
	}
}
