package eventdecisions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"

	"github.com/jackc/pgx/v5"
)

type Replayer struct {
	Store     *Store
	Evaluator Evaluator
	Resolver  *assetresolution.Resolver
	Origin    DecisionOrigin
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
		resolution  *assetresolution.Result
		fingerprint string
		replayID    string
	}
	ready := []evaluated{}
	for _, event := range events {
		event, resolution := r.prepareEvent(event)
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
		summary.Outcomes = append(summary.Outcomes, EventOutcome{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Proposed: result, AssetResolution: resolution})
		ready = append(ready, evaluated{event: event, result: result, resolution: resolution, fingerprint: fingerprint, replayID: replayID})
	}
	if len(summary.Failures) > 0 {
		summary.CompletedAt = now().UTC()
		return summary, fmt.Errorf("replay evaluation failed closed with %d failure(s)", len(summary.Failures))
	}
	decisionAt := now().UTC()
	for index, item := range ready {
		persisted, reused, err := persistDecision(ctx, tx, item.event, item.result, r.Evaluator.Ruleset, decisionAt, item.fingerprint, item.replayID, r.decisionOrigin(), r.decisionContext())
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("transactional event replay failed: %w", err)
		}
		summary.Outcomes[index].Persisted = &persisted
		summary.Outcomes[index].Reused = reused
		if item.resolution != nil {
			if err := persistAssetResolution(ctx, tx, item.event, persisted, *item.resolution, decisionAt); err != nil {
				return summary, fmt.Errorf("persist event asset resolution: %w", err)
			}
			summary.Outcomes[index].AssetResolution = item.resolution
		}
		if reused {
			summary.DecisionsReused++
		} else {
			summary.DecisionsCreated++
		}
		if item.result.Decision == DecisionCandidate {
			summary.CandidatesReused++
		}
		subject, err := r.Store.PersistSubjectEvaluation(ctx, tx, item.event, r.Evaluator.Ruleset, decisionAt)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("transactional subject replay failed: %w", err)
		}
		summary.Outcomes[index].Subject = &subject
		incrementSubjectPersistence(&summary, subject)
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
		resolution  *assetresolution.Result
		fingerprint string
		replayID    string
	}
	ready := []evaluated{}
	for _, event := range events {
		event, resolution := r.prepareEvent(event)
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
		summary.Outcomes = append(summary.Outcomes, EventOutcome{InboxID: event.InboxID, SourceEventID: event.SourceEventID, Proposed: result, AssetResolution: resolution})
		ready = append(ready, evaluated{event: event, result: result, resolution: resolution, fingerprint: fingerprint, replayID: replayID})
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
		persisted, reused, err := persistDecision(ctx, tx, item.event, item.result, r.Evaluator.Ruleset, decisionAt, item.fingerprint, item.replayID, r.decisionOrigin(), r.decisionContext())
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("persisted replay rolled back: %w", err)
		}
		summary.Outcomes[index].Persisted = &persisted
		summary.Outcomes[index].Reused = reused
		if item.resolution != nil {
			if err := persistAssetResolution(ctx, tx, item.event, persisted, *item.resolution, decisionAt); err != nil {
				return summary, fmt.Errorf("persist event asset resolution: %w", err)
			}
			summary.Outcomes[index].AssetResolution = item.resolution
		}
		if reused {
			summary.DecisionsReused++
		} else {
			summary.DecisionsCreated++
		}
		if item.result.Decision == DecisionCandidate {
			summary.CandidatesReused++
		}
		subject, err := r.Store.PersistSubjectEvaluation(ctx, tx, item.event, r.Evaluator.Ruleset, decisionAt)
		if err != nil {
			summary.Failures = append(summary.Failures, Failure{InboxID: item.event.InboxID, SourceEventID: item.event.SourceEventID, Error: err.Error()})
			summary.CompletedAt = now().UTC()
			return summary, fmt.Errorf("persisted subject replay rolled back: %w", err)
		}
		summary.Outcomes[index].Subject = &subject
		incrementSubjectPersistence(&summary, subject)
	}
	if err := tx.Commit(ctx); err != nil {
		return summary, fmt.Errorf("commit event decision replay: %w", err)
	}
	summary.CompletedAt = now().UTC()
	return summary, nil
}

func (r Replayer) decisionOrigin() DecisionOrigin {
	if r.Origin == DecisionOriginLive || r.Origin == DecisionOriginBackfill || r.Origin == DecisionOriginReplay {
		return r.Origin
	}
	return DecisionOriginBackfill
}

func (r Replayer) decisionContext() string {
	if r.decisionOrigin() == DecisionOriginLive {
		return "continuous_world_monitor_ingestion"
	}
	return "bounded_operator_backfill"
}

func (r Replayer) prepareEvent(event Event) (Event, *assetresolution.Result) {
	if r.Resolver == nil {
		return event, nil
	}
	sourceURL := ""
	if len(event.SourceURLs) > 0 {
		sourceURL = event.SourceURLs[0]
	}
	resolution := r.Resolver.Resolve(assetresolution.Input{
		EventID: event.InboxID.String(), Headline: event.Headline, Summary: event.Summary,
		SourceName: event.SourceName, SourceURL: sourceURL, EventType: event.EventType,
		PublicationAt: event.PublicationAt, ReceiptAt: event.ReceiptAt,
		ExplicitSymbols: event.AffectedAssets, ExplicitReason: event.MappingReason, ExplicitMethods: event.MappingMethods,
	})
	if resolution.Status == assetresolution.StatusResolved {
		event.AffectedAssets = []string{resolution.Symbol}
		event.MappingReason = resolution.Reason
		event.MappingMethods = append([]string{resolution.MappingType, resolution.RulesetVersion}, resolution.SourceFields...)
	} else {
		event.AffectedAssets = nil
		event.MappingReason = resolution.Reason
		event.MappingMethods = append([]string{resolution.Status, resolution.RulesetVersion}, resolution.SourceFields...)
	}
	if strings.EqualFold(r.Evaluator.Ruleset.Version, "genuine-event-decision-v2") {
		if resolution.MaterialEvent {
			event.Confidence = 0.7
			event.ConfidenceReasons = []string{"deterministic material-event term in source content"}
		} else {
			event.Confidence = 0.3
			event.ConfidenceReasons = []string{"no bounded material-event term in source content"}
		}
		if strings.EqualFold(event.SourceName, "Federal Reserve") && resolution.MaterialEvent {
			event.Confidence = 0.75
			event.ConfidenceReasons = []string{"official Federal Reserve source and bounded monetary-policy term"}
		}
	}
	return event, &resolution
}

func incrementSubjectPersistence(summary *Summary, outcome SubjectPersistenceOutcome) {
	if outcome.SubjectCreated {
		summary.SubjectsCreated++
	} else {
		summary.SubjectsReused++
	}
	if outcome.LinkCreated {
		summary.LinksCreated++
	} else {
		summary.LinksReused++
	}
	if outcome.EvaluationCreated {
		summary.SubjectEvaluationsCreated++
	} else {
		summary.SubjectEvaluationsReused++
	}
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
