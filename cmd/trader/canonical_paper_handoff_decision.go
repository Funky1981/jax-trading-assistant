package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	approvalsmod "jax-trading-assistant/internal/modules/approvals"
	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
	"jax-trading-assistant/libs/marketdata"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type canonicalWorkflowRecord struct {
	workflow workflow.Workflow
	intent   workflow.PaperIntent
	events   []workflow.TransitionEvent
}

func loadCanonicalWorkflowTx(ctx context.Context, tx pgx.Tx, candidateID string) (canonicalWorkflowRecord, error) {
	var record canonicalWorkflowRecord
	var payload []byte
	err := tx.QueryRow(ctx, `SELECT payload FROM workflow_instances WHERE recommendation_id=$1 ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, candidateID).Scan(&payload)
	if err != nil {
		return record, err
	}
	var envelope struct {
		Workflow    workflow.Workflow    `json:"workflow"`
		PaperIntent workflow.PaperIntent `json:"paperIntent"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return record, fmt.Errorf("decode canonical workflow payload: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT event_id,workflow_id,sequence,previous_state,next_state,action,actor,actor_role,idempotency_key,input_fingerprint,content_identity,COALESCE(reason,''),occurred_at,transition_version FROM workflow_audit_events WHERE workflow_id=$1 ORDER BY sequence`, envelope.Workflow.WorkflowID)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	for rows.Next() {
		var event workflow.TransitionEvent
		var sequence int64
		var previous, next, action, actorRole string
		if err := rows.Scan(&event.EventID, &event.WorkflowID, &sequence, &previous, &next, &action, &event.Actor, &actorRole, &event.IdempotencyKey, &event.InputFingerprint, &event.ContentIdentity, &event.Reason, &event.OccurredAt, &event.TransitionVersion); err != nil {
			return record, err
		}
		event.Sequence, event.PreviousState, event.NextState = uint64(sequence), workflow.State(previous), workflow.State(next)
		event.Action, event.ActorRole, event.OccurredAt = workflow.Action(action), workflow.ActorRole(actorRole), event.OccurredAt.UTC()
		record.events = append(record.events, event)
	}
	if err := rows.Err(); err != nil {
		return record, err
	}
	if err := envelope.Workflow.Validate(); err != nil {
		return record, err
	}
	if err := workflow.ValidateAuditEvents(record.events); err != nil || len(record.events) == 0 || record.events[len(record.events)-1].NextState != envelope.Workflow.State || record.events[len(record.events)-1].Sequence != envelope.Workflow.Revision {
		return record, fmt.Errorf("canonical workflow durable audit does not match state: %v", err)
	}
	record.workflow, record.intent = envelope.Workflow, envelope.PaperIntent
	return record, nil
}

func loadCanonicalRiskTx(ctx context.Context, tx pgx.Tx, riskID string) (portfoliorisk.RiskDecision, error) {
	var payload []byte
	if err := tx.QueryRow(ctx, `SELECT payload FROM portfolio_risk_decisions WHERE decision_id=$1`, riskID).Scan(&payload); err != nil {
		return portfoliorisk.RiskDecision{}, err
	}
	var decision portfoliorisk.RiskDecision
	if err := json.Unmarshal(payload, &decision); err != nil {
		return portfoliorisk.RiskDecision{}, err
	}
	return decision, decision.Validate()
}

func loadCanonicalSnapshotTx(ctx context.Context, tx pgx.Tx, snapshotID string) (portfoliorisk.PortfolioSnapshot, error) {
	var payload []byte
	if err := tx.QueryRow(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, snapshotID).Scan(&payload); err != nil {
		return portfoliorisk.PortfolioSnapshot{}, err
	}
	var snapshot portfoliorisk.PortfolioSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return portfoliorisk.PortfolioSnapshot{}, err
	}
	canonical, err := portfoliorisk.BuildSnapshot(snapshot)
	if err != nil || canonical.SnapshotID != snapshotID {
		return portfoliorisk.PortfolioSnapshot{}, fmt.Errorf("persisted portfolio snapshot identity is invalid")
	}
	return canonical, nil
}

func (s *canonicalHandoffService) reject(ctx context.Context, candidateID uuid.UUID, actor string) (canonicalWorkflowRecord, error) {
	for attempt := 0; attempt < 3; attempt++ {
		result, err := s.rejectOnce(ctx, candidateID, actor)
		if err == nil || !retryCanonicalTransaction(err) || attempt == 2 {
			return result, err
		}
	}
	return canonicalWorkflowRecord{}, fmt.Errorf("canonical REJECT transaction retries exhausted")
}

func (s *canonicalHandoffService) rejectOnce(ctx context.Context, candidateID uuid.UUID, actor string) (canonicalWorkflowRecord, error) {
	if s == nil || s.pool == nil {
		return canonicalWorkflowRecord{}, fmt.Errorf("canonical PAPER workflow database is unavailable")
	}
	if candidateID == uuid.Nil || strings.TrimSpace(actor) == "" {
		return canonicalWorkflowRecord{}, fmt.Errorf("candidate and authenticated JWT actor are required")
	}
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return canonicalWorkflowRecord{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, candidateID.String()); err != nil {
		return canonicalWorkflowRecord{}, err
	}
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM candidate_trades WHERE id=$1 FOR UPDATE`, candidateID).Scan(&locked); err != nil {
		return canonicalWorkflowRecord{}, err
	}
	record, err := loadCanonicalWorkflowTx(ctx, tx, candidateID.String())
	if err != nil {
		return record, err
	}
	decision, err := loadCanonicalRiskTx(ctx, tx, record.workflow.RiskDecisionID)
	if err != nil || decision.RecommendationID != candidateID.String() || (decision.Outcome != portfoliorisk.DecisionAccept && decision.Outcome != portfoliorisk.DecisionAmend) {
		return record, fmt.Errorf("pending workflow has no matching accepted portfolio risk decision")
	}
	if record.workflow.State == workflow.StateHumanRejected && record.workflow.Confirmation != nil && record.workflow.Confirmation.Actor == actor {
		if err := tx.Commit(ctx); err != nil {
			return canonicalWorkflowRecord{}, err
		}
		return record, nil
	}
	if record.workflow.State != workflow.StateAwaitingHumanConfirmation {
		return record, fmt.Errorf("canonical rejection requires AWAITING_HUMAN_CONFIRMATION")
	}
	input, err := loadCanonicalHandoffInput(ctx, tx, candidateID, now, false)
	if err != nil {
		return record, err
	}
	if input.Economic.CandidateID != candidateID || strings.TrimSpace(input.Economic.InstrumentID) == "" {
		return record, fmt.Errorf("durable candidate economic instrument identity is unavailable for rejection provenance")
	}
	expiresAt := input.Candidate.ExpiresAt.UTC()
	if !expiresAt.After(now) {
		expiresAt = now.Add(time.Minute)
	}
	direction, directionErr := canonicalCandidateDirection(input.Candidate.Direction, input.Candidate.SignalType)
	if directionErr != nil {
		return record, directionErr
	}
	confirmation, err := workflow.NewConfirmation(record.workflow, input.Economic.InstrumentID, direction, now, expiresAt)
	if err != nil {
		return record, err
	}
	confirmation, err = confirmation.WithDecision(actor, workflow.ConfirmationReject)
	if err != nil {
		return record, err
	}
	store, err := workflow.RestoreWorkflowState(record.workflow, record.events, nil)
	if err != nil {
		return record, err
	}
	final, err := store.Confirm(ctx, workflow.HumanConfirmationRequest{WorkflowID: record.workflow.WorkflowID, Confirmation: confirmation, Actor: actor,
		ActorRole: workflow.ActorHuman, IdempotencyKey: "canonical-human-reject:" + record.workflow.WorkflowID, Now: now})
	if err != nil {
		return record, err
	}
	bytes, err := store.ExportSnapshot()
	if err != nil {
		return record, err
	}
	var durable workflow.DurableSnapshot
	if err := json.Unmarshal(bytes, &durable); err != nil {
		return record, err
	}
	if err := approvalsmod.PersistCanonicalWorldMonitorDecision(ctx, tx, candidateID, actor, input.Economic.ContentIdentity, decision, final, now); err != nil {
		return record, err
	}
	if err := persistCanonicalWorkflowSnapshot(ctx, tx, durable, final.WorkflowID, workflow.PaperIntent{}); err != nil {
		return record, err
	}
	if err := tx.Commit(ctx); err != nil {
		return canonicalWorkflowRecord{}, err
	}
	return canonicalWorkflowRecord{workflow: final, events: durable.Events[final.WorkflowID]}, nil
}

func (s *canonicalHandoffService) approve(ctx context.Context, candidateID uuid.UUID, actor string) (canonicalPrepareResult, error) {
	for attempt := 0; attempt < 3; attempt++ {
		result, err := s.approveOnce(ctx, candidateID, actor)
		if err == nil || !retryCanonicalTransaction(err) || attempt == 2 {
			return result, err
		}
	}
	return canonicalPrepareResult{}, fmt.Errorf("canonical APPROVE transaction retries exhausted")
}

func (s *canonicalHandoffService) approveOnce(ctx context.Context, candidateID uuid.UUID, actor string) (canonicalPrepareResult, error) {
	var response canonicalPrepareResult
	if err := s.configurationError(); err != nil {
		return response, err
	}
	if candidateID == uuid.Nil || strings.TrimSpace(actor) == "" {
		return response, fmt.Errorf("candidate and authenticated JWT actor are required")
	}
	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return response, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, candidateID.String()); err != nil {
		return response, err
	}
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM candidate_trades WHERE id=$1 FOR UPDATE`, candidateID).Scan(&locked); err != nil {
		return response, err
	}
	var candidateStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, candidateID).Scan(&candidateStatus); err != nil {
		return response, err
	}
	if candidateStatus == candidatesmod.StatusApproved {
		return s.canonicalApprovalReplay(ctx, tx, candidateID, actor)
	}
	if candidateStatus != candidatesmod.StatusAwaitingApproval {
		return response, fmt.Errorf("candidate is not awaiting canonical PAPER approval")
	}
	input, err := loadCanonicalHandoffInput(ctx, tx, candidateID, now, true)
	if err != nil {
		return response, err
	}
	if err := s.validateCanonicalCandidate(input); err != nil {
		return response, err
	}
	record, err := loadCanonicalWorkflowTx(ctx, tx, candidateID.String())
	if err != nil {
		return response, fmt.Errorf("canonical PREPARE workflow is required: %w", err)
	}
	if record.workflow.State == workflow.StatePaperIntentCreated {
		return s.canonicalApprovalReplay(ctx, tx, candidateID, actor)
	}
	if record.workflow.State != workflow.StateAwaitingHumanConfirmation {
		return response, fmt.Errorf("canonical approval requires AWAITING_HUMAN_CONFIRMATION")
	}
	risk, err := loadCanonicalRiskTx(ctx, tx, record.workflow.RiskDecisionID)
	if err != nil || risk.RecommendationID != candidateID.String() || risk.PortfolioSnapshotID != record.workflow.PortfolioSnapshotID || (risk.Outcome != portfoliorisk.DecisionAccept && risk.Outcome != portfoliorisk.DecisionAmend) {
		return response, fmt.Errorf("pending workflow risk decision is missing or mismatched")
	}
	preparedSnapshot, err := loadCanonicalSnapshotTx(ctx, tx, risk.PortfolioSnapshotID)
	if err != nil {
		return response, err
	}
	currentSnapshot, account, err := s.buildPortfolioSnapshot(ctx, tx, now)
	if err != nil {
		return response, err
	}
	if currentSnapshot.SnapshotID != preparedSnapshot.SnapshotID || currentSnapshot.AssessFreshness(now, s.maxAge).Status != portfoliorisk.FreshnessFresh {
		return response, fmt.Errorf("PAPER account or material position valuation changed after PREPARE; prepare again under explicit supersession")
	}
	entry, stop, target := *input.Candidate.EntryPrice, *input.Candidate.StopLoss, *input.Candidate.TakeProfit
	slippage := *input.Economic.SlippageAllowance
	if !finitePositive(entry) || !finitePositive(stop) || !finitePositive(target) || !finiteNonNegative(slippage) {
		return response, fmt.Errorf("persisted candidate economic levels are invalid")
	}
	direction, err := canonicalCandidateDirection(input.Candidate.Direction, input.Candidate.SignalType)
	if err != nil || direction != "LONG" {
		return response, fmt.Errorf("short opening is unsupported by the current PAPER ledger")
	}
	if stop >= entry || target <= entry {
		return response, fmt.Errorf("candidate LONG stop/entry/target geometry is invalid")
	}
	quote, err := loadCanonicalQuoteObservationFrom(ctx, tx, input.Candidate.Symbol, now, s.market)
	if err != nil {
		return response, fmt.Errorf("fresh canonical entry observation unavailable: %w", err)
	}
	if math.Abs(quote.Last-entry) > slippage+1e-9 {
		return response, fmt.Errorf("fresh entry price differs from prepared reference beyond explicit slippage allowance; prepare again")
	}
	calendar, err := s.sessionCalendar()
	if err != nil {
		return response, fmt.Errorf("PAPER session calendar unavailable: %w", err)
	}
	session, err := calendar.SessionState(quote.ProviderAt.UTC())
	if err != nil || session == exploratorypaper.SessionUnknown {
		return response, fmt.Errorf("fresh entry observation has unknown PAPER session")
	}
	store, err := workflow.RestoreWorkflowState(record.workflow, record.events, nil)
	if err != nil {
		return response, err
	}
	confirmation, err := workflow.NewConfirmation(record.workflow, input.Economic.InstrumentID, direction, now, input.Candidate.ExpiresAt.UTC())
	if err != nil {
		return response, err
	}
	confirmation, err = confirmation.WithDecision(actor, workflow.ConfirmationApprove)
	if err != nil {
		return response, err
	}
	approved, err := store.Confirm(ctx, workflow.HumanConfirmationRequest{WorkflowID: record.workflow.WorkflowID, Confirmation: confirmation, Actor: actor,
		ActorRole: workflow.ActorHuman, IdempotencyKey: "canonical-human-approve:" + record.workflow.WorkflowID, Now: now})
	if err != nil {
		return response, err
	}
	approved, intent, err := store.CreatePaperIntent(ctx, workflow.PaperIntentRequest{WorkflowID: approved.WorkflowID, Actor: "canonical-paper-handoff", ActorRole: workflow.ActorSystem,
		IdempotencyKey: "canonical-paper-intent:" + approved.WorkflowID, Now: now})
	if err != nil {
		return response, err
	}
	thesisInput, thesis, result, err := buildCanonicalThesis(input, risk, preparedSnapshot.Equity.Value, now)
	if err != nil {
		return response, err
	}
	if result.Decision != exploratorypaper.DecisionCandidate {
		return response, fmt.Errorf("canonical thesis candidate projection failed: %s", result.Reason)
	}
	venue := papertrading.DefaultPaperCapabilityContract()
	costModel := papertrading.DefaultCostModel()
	quantity := math.Min(math.Abs(risk.ResultingValue)/quote.Last, thesisInput.RequestedQuantity)
	quantity = math.Min(quantity, account.account.Cash/quote.Last)
	quantity = math.Floor(quantity*1e6) / 1e6
	if !finitePositive(quantity) || quantity < venue.MinimumQuantity || quantity*quote.Last > account.account.Cash+1e-9 || quantity*quote.Last > math.Abs(risk.ResultingValue)+1e-6 {
		return response, fmt.Errorf("risk-approved quantity cannot satisfy venue, capital, and exposure limits")
	}
	tickID := "tick_" + canonicalDigest(input.Candidate.Symbol, quote.Source, quote.ProviderAt.UTC().Format(time.RFC3339Nano), quote.LastProviderAt.UTC().Format(time.RFC3339Nano), quote.ReceivedAt.UTC().Format(time.RFC3339Nano), fmt.Sprintf("%.12g|%.12g|%.12g", quote.Bid, quote.Ask, quote.Last))
	tick := papertrading.MarketTick{TickID: tickID, InstrumentID: input.Economic.InstrumentID, Bid: quote.Bid, Ask: quote.Ask, Last: quote.Last,
		AvailableQuantity: quantity, Timestamp: quote.ProviderAt.UTC(), LastProviderAt: quote.LastProviderAt.UTC(), ReceivedAt: quote.ReceivedAt.UTC(), AsOf: now.UTC(), RequireTemporalProvenance: true, Session: papertrading.Session(session), Source: quote.Source}
	if err := tick.Validate(venue.MaxQuoteAge); err != nil {
		return response, fmt.Errorf("fresh PAPER market tick is invalid: %w", err)
	}
	positionID := "paper-pos-" + canonicalDigest(candidateID.String(), risk.DecisionID, approved.WorkflowID)[:40]
	approvalBytes, err := store.ExportSnapshot()
	if err != nil {
		return response, err
	}
	var durable workflow.DurableSnapshot
	if err := json.Unmarshal(approvalBytes, &durable); err != nil {
		return response, err
	}
	entryRequest := exploratorypaper.EntryRequest{CandidateID: candidateID.String(), Candidate: thesisInput.Candidate,
		Evidence: input.Assessment, Thesis: thesis, RiskDecision: risk,
		Approval:   exploratorypaper.ApprovalSnapshot{Workflow: approved, Events: durable.Events[approved.WorkflowID], PaperIntent: intent},
		PositionID: positionID, Quantity: quantity, Tick: tick, Venue: venue, CostModel: costModel, Ledger: account.account,
		Calendar: calendar, Now: now}
	if err := approvalsmod.PersistCanonicalWorldMonitorDecision(ctx, tx, candidateID, actor, input.Economic.ContentIdentity, risk, approved, now); err != nil {
		return response, err
	}
	if err := persistCanonicalWorkflowSnapshot(ctx, tx, durable, approved.WorkflowID, intent); err != nil {
		return response, err
	}
	queue := exploratorypaper.NewPostgresStoreForAccount(s.pool, s.accountID)
	if err := queue.QueueApprovedEntryTx(ctx, tx, entryRequest); err != nil {
		return response, err
	}
	if err := tx.Commit(ctx); err != nil {
		return canonicalPrepareResult{}, err
	}
	response = canonicalPrepareResult{CandidateID: candidateID.String(), Symbol: input.Candidate.Symbol, InstrumentID: input.Economic.InstrumentID,
		IssuerID: input.Economic.IssuerID, Entry: entry, Stop: stop, Target: target, RiskAllocation: input.Economic.RiskAllocation,
		RequestedLeverage: input.Economic.RequestedLeverage, RequestedNotional: math.Abs(risk.RequestedValue), ResultingNotional: math.Abs(risk.ResultingValue),
		RequestedQuantity: quantity, Outcome: risk.Outcome, ReasonCodes: risk.ReasonCodes, PortfolioSnapshotID: risk.PortfolioSnapshotID,
		RiskDecisionID: risk.DecisionID, WorkflowID: approved.WorkflowID, WorkflowState: approved.State, PolicyID: risk.PolicyID, PreparedAt: approved.CreatedAt}
	return response, nil
}

type canonicalThesisInput struct {
	Candidate         exploratorypaper.CandidateInput
	RequestedQuantity float64
}

func buildCanonicalThesis(input canonicalHandoffInput, risk portfoliorisk.RiskDecision, equity float64, now time.Time) (canonicalThesisInput, exploratorypaper.TradeThesis, exploratorypaper.CandidateResult, error) {
	var out canonicalThesisInput
	var thesis exploratorypaper.TradeThesis
	var result exploratorypaper.CandidateResult
	var metadata struct {
		WorldMonitor struct {
			NormalizedEventID string   `json:"normalizedEventId"`
			EventType         string   `json:"eventType"`
			Headline          string   `json:"headline"`
			Summary           string   `json:"summary"`
			SourceURLs        []string `json:"sourceURLs"`
			MappingReason     string   `json:"mappingReason"`
			IsSynthetic       bool     `json:"isSynthetic"`
		} `json:"worldMonitor"`
		Chart struct {
			Reason           string    `json:"reason"`
			CandleCount      int       `json:"candleCount"`
			LastClose        float64   `json:"lastClose"`
			SMA20            float64   `json:"sma20"`
			CheckedAt        time.Time `json:"checkedAt"`
			Source           string    `json:"source"`
			Timeframe        string    `json:"timeframe"`
			EntryObservation struct {
				Symbol         string    `json:"symbol"`
				Source         string    `json:"source"`
				Last           float64   `json:"last"`
				ProviderAt     time.Time `json:"providerAt"`
				LastProviderAt time.Time `json:"lastProviderAt"`
				ReceivedAt     time.Time `json:"receivedAt"`
			} `json:"entryObservation"`
		} `json:"chartConfirmation"`
		Horizon struct {
			HoldTargetDays int `json:"holdTargetDays"`
			MaxHoldDays    int `json:"maxHoldDays"`
		} `json:"horizonPolicy"`
	}
	if input.Candidate.Metadata == nil || json.Unmarshal(*input.Candidate.Metadata, &metadata) != nil {
		return out, thesis, result, fmt.Errorf("missing durable candidate metadata for thesis projection")
	}
	candidate := input.Candidate
	entryObservation := marketdata.EconomicObservation{ProviderAt: metadata.Chart.EntryObservation.ProviderAt, LastProviderAt: metadata.Chart.EntryObservation.LastProviderAt, ReceivedAt: metadata.Chart.EntryObservation.ReceivedAt}
	if metadata.WorldMonitor.NormalizedEventID == "" || metadata.WorldMonitor.IsSynthetic || metadata.WorldMonitor.EventType == "" || metadata.WorldMonitor.MappingReason == "" || metadata.Chart.Reason == "" || metadata.Chart.Source == "" || metadata.Chart.Timeframe == "" || metadata.Chart.CandleCount < 20 || !finitePositive(metadata.Chart.LastClose) || !finitePositive(metadata.Chart.SMA20) || metadata.Chart.CheckedAt.IsZero() || metadata.Chart.EntryObservation.Source == "" || !strings.EqualFold(metadata.Chart.EntryObservation.Symbol, candidate.Symbol) || !finitePositive(metadata.Chart.EntryObservation.Last) || entryObservation.ValidateQuoteTemporal(metadata.Chart.CheckedAt, 60*time.Second, marketdata.MaxQuoteClockSkew) != nil || candidate.EntryPrice == nil || math.Abs(metadata.Chart.EntryObservation.Last-*candidate.EntryPrice) > 1e-8 {
		return out, thesis, result, fmt.Errorf("missing durable thesis provenance: normalized event, event category, source-backed mapping reason, or chart confirmation")
	}
	eventID := metadata.WorldMonitor.NormalizedEventID
	if _, err := uuid.Parse(eventID); err != nil {
		return out, thesis, result, fmt.Errorf("normalized event identity is invalid")
	}
	if candidate.CatalystTimestamp == nil || candidate.DetectedAt.IsZero() || candidate.Confidence == nil || candidate.CatalystSummary == "" || candidate.InvalidationReason == "" {
		return out, thesis, result, fmt.Errorf("missing durable thesis provenance: catalyst time, generated time, confidence, causal summary, or invalidation reason")
	}
	horizon := metadata.Horizon.HoldTargetDays
	if horizon < 1 || horizon > 5 {
		return out, thesis, result, fmt.Errorf("missing durable thesis provenance: horizonPolicy.holdTargetDays must be 1-5")
	}
	entryRationale := ""
	if candidate.CandidateReasonSummary != nil {
		entryRationale = strings.TrimSpace(*candidate.CandidateReasonSummary)
	}
	if entryRationale == "" && candidate.Reasoning != nil {
		entryRationale = strings.TrimSpace(*candidate.Reasoning)
	}
	if entryRationale == "" {
		return out, thesis, result, fmt.Errorf("missing durable thesis provenance: candidate rationale")
	}
	quantContext := fmt.Sprintf("%s; timeframe=%s; provider=%s; candles=%d; close=%.8g; SMA20=%.8g; checked_at=%s", metadata.Chart.Reason, metadata.Chart.Timeframe, metadata.Chart.Source, metadata.Chart.CandleCount, metadata.Chart.LastClose, metadata.Chart.SMA20, metadata.Chart.CheckedAt.UTC().Format(time.RFC3339Nano))
	riskAssessment := fmt.Sprintf("policy=%s; outcome=%s; requested_value=%.8g; resulting_value=%.8g; reasons=%s", risk.PolicyID, risk.Outcome, risk.RequestedValue, risk.ResultingValue, strings.Join(reasonCodeValues(risk.ReasonCodes), ","))
	inputCandidate := exploratorypaper.CandidateInput{EventID: eventID, IssuerID: input.Economic.IssuerID, InstrumentID: input.Economic.InstrumentID,
		EventCategory: metadata.WorldMonitor.EventType, EventTimestamp: candidate.CatalystTimestamp.UTC(), GeneratedAt: candidate.DetectedAt.UTC(),
		Evidence: input.Evidence, EvidenceQuality: input.Score.QualityScore, Corroborated: true, ReviewedEvidence: &input.Assessment,
		CandidatePolicyVersion: input.Assessment.PolicyVersion, CausalMechanism: metadata.WorldMonitor.MappingReason, QuantContext: quantContext,
		RiskAssessment: riskAssessment, TechnicalConfirmation: metadata.Chart.Reason, Direction: exploratorypaper.DirectionLong}
	for _, rawURL := range metadata.WorldMonitor.SourceURLs {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(rawURL)), "https://") {
			inputCandidate.SourceURL = strings.TrimSpace(rawURL)
			break
		}
	}
	if inputCandidate.SourceURL == "" && len(input.Evidence) > 0 && (strings.HasPrefix(input.Evidence[0].SourceURL, "https://") || strings.Contains(input.Evidence[0].SourceURL, ":")) {
		inputCandidate.SourceURL = input.Evidence[0].SourceURL
	}
	if inputCandidate.SourceURL == "" {
		return out, thesis, result, fmt.Errorf("missing durable thesis provenance: source URL")
	}
	result, err := exploratorypaper.GenerateCandidate(inputCandidate)
	if err != nil || result.Decision != exploratorypaper.DecisionCandidate {
		return out, thesis, result, fmt.Errorf("candidate thesis gate failed: %s (%v)", result.Reason, err)
	}
	entry, stop, target := *candidate.EntryPrice, *candidate.StopLoss, *candidate.TakeProfit
	thesis = exploratorypaper.TradeThesis{ThesisID: "thesis-" + canonicalDigest(input.Candidate.ID.String(), risk.DecisionID)[:40], ContractVersion: exploratorypaper.ContractVersion, Mode: exploratorypaper.ExploratoryPaperMode,
		EventID: eventID, IssuerID: input.Economic.IssuerID, InstrumentID: input.Economic.InstrumentID, Direction: exploratorypaper.DirectionLong,
		Exposure: "LONG", EventCategory: metadata.WorldMonitor.EventType, EventTimestamp: candidate.CatalystTimestamp.UTC(), CandidateGeneratedAt: candidate.DetectedAt.UTC(),
		CausalReason: strings.TrimSpace(candidate.CatalystSummary), Evidence: input.Evidence, ExpectedMechanism: metadata.WorldMonitor.MappingReason,
		ExpectedHorizonSessions: horizon, EntryRationale: entryRationale, QuantTechnicalContext: quantContext, RiskAssessment: riskAssessment,
		EntryPolicyVersion: input.Economic.SizingPolicyVersion, ProtectiveStop: stop, Target: target,
		InvalidationConditions: []string{strings.TrimSpace(candidate.InvalidationReason)}, Confidence: *candidate.Confidence,
		Uncertainty:   fmt.Sprintf("Recorded World Monitor confidence=%.6f; reviewed evidence quality=%.6f.", *candidate.Confidence, input.Score.QualityScore),
		PolicyVersion: input.Economic.SizingPolicyID, CreatedAt: now.UTC()}
	if err := thesis.Validate(); err != nil {
		return out, thesis, result, fmt.Errorf("durable candidate facts cannot satisfy exploratory thesis contract: %w", err)
	}
	perUnitRisk := math.Abs(entry - stop)
	if input.Economic.SlippageAllowance == nil {
		return out, thesis, result, fmt.Errorf("missing explicit candidate slippage allowance")
	}
	perUnitRisk += *input.Economic.SlippageAllowance
	if perUnitRisk <= 0 || !finitePositive(perUnitRisk) {
		return out, thesis, result, fmt.Errorf("candidate quantity ceiling cannot be reconstructed")
	}
	if !finitePositive(equity) || !finitePositive(input.Economic.RiskAllocation) {
		return out, thesis, result, fmt.Errorf("canonical equity and explicit risk allocation are required to reconstruct quantity ceiling")
	}
	requestedQuantity := equity * input.Economic.RiskAllocation / perUnitRisk
	if requestedQuantity <= 0 || !finitePositive(requestedQuantity) {
		return out, thesis, result, fmt.Errorf("requested quantity ceiling is invalid")
	}
	out = canonicalThesisInput{Candidate: inputCandidate, RequestedQuantity: requestedQuantity}
	return out, thesis, result, nil
}

func reasonCodeValues(codes []portfoliorisk.ReasonCode) []string {
	values := make([]string, len(codes))
	for index, code := range codes {
		values[index] = string(code)
	}
	return values
}

func (s *canonicalHandoffService) canonicalApprovalReplay(ctx context.Context, tx pgx.Tx, candidateID uuid.UUID, actor string) (canonicalPrepareResult, error) {
	var response canonicalPrepareResult
	var payload []byte
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM candidate_trades WHERE id=$1`, candidateID).Scan(&status); err != nil {
		return response, err
	}
	if status != candidatesmod.StatusApproved {
		return response, fmt.Errorf("canonical approval workflow is not durably approved")
	}
	var queuePayload []byte
	if err := tx.QueryRow(ctx, `SELECT payload FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, candidateID.String()).Scan(&queuePayload); err != nil {
		return response, fmt.Errorf("canonical approval replay has no durable entry queue: %w", err)
	}
	var entry exploratorypaper.EntryRequest
	if err := json.Unmarshal(queuePayload, &entry); err != nil {
		return response, fmt.Errorf("decode canonical queued entry replay: %w", err)
	}
	if entry.CandidateID != candidateID.String() || entry.Approval.Workflow.State != workflow.StatePaperIntentCreated || entry.Approval.Workflow.Confirmation == nil || entry.Approval.Workflow.Confirmation.Actor != actor {
		return response, fmt.Errorf("canonical approval retry identity or JWT actor does not match durable queue")
	}
	if err := entry.Approval.Workflow.Validate(); err != nil || entry.Approval.PaperIntent.Validate() != nil || entry.Approval.Workflow.Confirmation.ValidateFor(entry.Approval.Workflow, entry.Now) != nil {
		return response, fmt.Errorf("canonical approval retry queue artifacts are invalid")
	}
	var decision portfoliorisk.RiskDecision
	if err := tx.QueryRow(ctx, `SELECT payload FROM portfolio_risk_decisions WHERE decision_id=$1`, entry.RiskDecision.DecisionID).Scan(&payload); err != nil || json.Unmarshal(payload, &decision) != nil || decision.Validate() != nil {
		return response, fmt.Errorf("canonical approval retry risk decision is unavailable")
	}
	if err := tx.Commit(ctx); err != nil {
		return canonicalPrepareResult{}, err
	}
	return canonicalPrepareResult{CandidateID: candidateID.String(), Symbol: entry.Candidate.InstrumentID, InstrumentID: entry.Candidate.InstrumentID,
		IssuerID: entry.Candidate.IssuerID, ResultingNotional: math.Abs(decision.ResultingValue), Outcome: decision.Outcome,
		ReasonCodes: decision.ReasonCodes, PortfolioSnapshotID: decision.PortfolioSnapshotID, RiskDecisionID: decision.DecisionID,
		WorkflowID: entry.Approval.Workflow.WorkflowID, WorkflowState: entry.Approval.Workflow.State, PolicyID: decision.PolicyID, PreparedAt: entry.Approval.Workflow.CreatedAt}, nil
}
