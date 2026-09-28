package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/tradingmodes"
	"jax-trading-assistant/libs/marketdata"
)

const (
	worldMonitorPromoterDefaultLimit  = 10
	worldMonitorPromoterMaxLimit      = 50
	worldMonitorPromoterMinConfidence = 0.55
	worldMonitorCandidateTTL          = 45 * time.Minute
)

var errWorldMonitorNoUsableQuote = errors.New("world monitor quote has no usable price")

type worldMonitorOpportunityPromoter struct {
	pool           *pgxpool.Pool
	now            func() time.Time
	marketPolicy   *marketDataSafetyPolicy  // non-nil only for explicit isolated test injection
	economicPolicy *candidateEconomicPolicy // non-nil only for explicit policy injection
}

type worldMonitorPromotionResult struct {
	Promoted            []worldMonitorPromotedOpportunity `json:"promoted"`
	PromotedCount       int                               `json:"promotedCount"`
	BlockedSkippedCount int                               `json:"blockedSkippedCount"`
	Skipped             int                               `json:"skipped"`
	Outcomes            []worldMonitorPromotionOutcome    `json:"outcomes"`
}

type worldMonitorPromotionOutcome struct {
	InboxID     string                               `json:"inboxId"`
	EventID     string                               `json:"eventId,omitempty"`
	Symbol      string                               `json:"symbol,omitempty"`
	Status      string                               `json:"status"`
	ReasonCode  string                               `json:"reasonCode,omitempty"`
	Reason      string                               `json:"reason,omitempty"`
	CandidateID string                               `json:"candidateId,omitempty"`
	RiskReview  *candidatesmod.RiskReviewPersistence `json:"riskReview,omitempty"`
}

type worldMonitorPromotedOpportunity struct {
	InboxID     string `json:"inboxId"`
	EventID     string `json:"eventId,omitempty"`
	SignalID    string `json:"signalId"`
	CandidateID string `json:"candidateId"`
	Symbol      string `json:"symbol"`
	Route       string `json:"route"`
}

type worldMonitorInboxPromotionRow struct {
	ID                   uuid.UUID
	SourceEventID        string
	NormalizedEventID    *uuid.UUID
	RawEventID           *uuid.UUID
	NormalizedSummary    string
	EventType            string
	Headline             string
	Summary              string
	SourceURLs           []string
	SourceCount          int
	PossibleAffectedETFs []string
	AssetThemes          []string
	Confidence           float64
	ConfidenceReasons    []string
	MappingReason        string
	EventTime            time.Time
	IsSynthetic          bool
	SyntheticReason      string
}

type worldMonitorChartConfirmation struct {
	Confirmed           bool                           `json:"confirmed"`
	ReasonCode          string                         `json:"reasonCode"`
	Reason              string                         `json:"reason"`
	CandleCount         int                            `json:"candleCount"`
	LastClose           float64                        `json:"lastClose,omitempty"`
	SMA20               float64                        `json:"sma20,omitempty"`
	FiveCandleChangePct float64                        `json:"fiveCandleChangePct,omitempty"`
	CheckedAt           time.Time                      `json:"checkedAt"`
	Source              string                         `json:"source,omitempty"`
	Timeframe           string                         `json:"timeframe,omitempty"`
	AsOf                time.Time                      `json:"asOf,omitempty"`
	EarliestCandleAt    time.Time                      `json:"earliestCandleAt,omitempty"`
	LatestCandleAt      time.Time                      `json:"latestCandleAt,omitempty"`
	LatestReceivedAt    time.Time                      `json:"latestReceivedAt,omitempty"`
	EntryObservation    marketdata.EconomicObservation `json:"entryObservation"`
}

func newWorldMonitorOpportunityPromoter(pool *pgxpool.Pool) *worldMonitorOpportunityPromoter {
	return &worldMonitorOpportunityPromoter{
		pool: pool,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (p *worldMonitorOpportunityPromoter) PromotePending(ctx context.Context, limit int) (worldMonitorPromotionResult, error) {
	if p.pool == nil {
		return worldMonitorPromotionResult{}, fmt.Errorf("world monitor promoter requires database pool")
	}
	if limit <= 0 {
		limit = worldMonitorPromoterDefaultLimit
	}
	if limit > worldMonitorPromoterMaxLimit {
		limit = worldMonitorPromoterMaxLimit
	}

	result, err := p.reviewPendingRiskCandidates(ctx, limit)
	if err != nil {
		return worldMonitorPromotionResult{}, err
	}
	rows, err := p.loadPromotionRows(ctx, limit)
	if err != nil {
		return worldMonitorPromotionResult{}, err
	}
	promoted, err := p.promoteRows(ctx, rows)
	if err != nil {
		return result, err
	}
	result.Promoted = append(result.Promoted, promoted.Promoted...)
	result.Outcomes = append(result.Outcomes, promoted.Outcomes...)
	result.PromotedCount += promoted.PromotedCount
	result.BlockedSkippedCount += promoted.BlockedSkippedCount
	result.Skipped = result.BlockedSkippedCount
	return result, nil
}

func (p *worldMonitorOpportunityPromoter) reviewPendingRiskCandidates(ctx context.Context, limit int) (worldMonitorPromotionResult, error) {
	result := worldMonitorPromotionResult{Promoted: []worldMonitorPromotedOpportunity{}, Outcomes: []worldMonitorPromotionOutcome{}}
	rows, err := p.pool.Query(ctx, `
		SELECT ct.id, w.id, w.normalized_event_id
		FROM candidate_trades ct JOIN world_monitor_research_inbox w ON w.candidate_id=ct.id
		WHERE ct.source='world-monitor' AND ct.status='awaiting_approval'
		  AND ct.gate_status=$1
		  AND (ct.expires_at IS NULL OR ct.expires_at >= NOW())
		ORDER BY ct.created_at LIMIT $2
	`, candidatesmod.GateStatusReadyForRiskReview, limit)
	if err != nil {
		return result, fmt.Errorf("load pending World Monitor risk candidates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var candidateID, inboxID uuid.UUID
		var eventID uuid.NullUUID
		if err := rows.Scan(&candidateID, &inboxID, &eventID); err != nil {
			return result, err
		}
		promoted, outcome, err := p.reviewCandidateRisk(ctx, candidateID, inboxID, eventID)
		if err != nil {
			return result, err
		}
		result.Outcomes = append(result.Outcomes, outcome)
		if promoted != nil {
			result.Promoted = append(result.Promoted, *promoted)
			result.PromotedCount++
		}
		if outcome.Status != "promoted" {
			result.BlockedSkippedCount++
		}
	}
	result.Skipped = result.BlockedSkippedCount
	return result, rows.Err()
}

func (p *worldMonitorOpportunityPromoter) reviewCandidateRisk(ctx context.Context, candidateID, inboxID uuid.UUID, eventID uuid.NullUUID) (*worldMonitorPromotedOpportunity, worldMonitorPromotionOutcome, error) {
	store := candidatesmod.NewStore(p.pool)
	candidate, err := store.GetByID(ctx, candidateID)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, fmt.Errorf("load risk candidate: %w", err)
	}
	policy := p.economicPolicy
	if policy == nil {
		loaded, loadErr := loadCandidateEconomicPolicy()
		if loadErr != nil {
			if err := store.MarkEconomicInputUnavailable(ctx, candidateID, "canonical_economic_inputs_unavailable"); err != nil {
				return nil, worldMonitorPromotionOutcome{}, err
			}
			return nil, promotionOutcomeIDs(inboxID, eventID, candidate.Symbol, "blocked", "canonical_economic_inputs_unavailable", loadErr.Error(), candidateID.String(), nil), nil
		}
		policy = &loaded
	}
	input, err := policy.build(candidate.Symbol, candidateID, p.now())
	if err != nil {
		if persistErr := store.MarkEconomicInputUnavailable(ctx, candidateID, "canonical_economic_inputs_unavailable"); persistErr != nil {
			return nil, worldMonitorPromotionOutcome{}, persistErr
		}
		return nil, promotionOutcomeIDs(inboxID, eventID, candidate.Symbol, "blocked", "canonical_economic_inputs_unavailable", err.Error(), candidateID.String(), nil), nil
	}
	if err := store.PersistCandidateEconomicInput(ctx, input); err != nil {
		if persistErr := store.MarkEconomicInputUnavailable(ctx, candidateID, "candidate_economic_input_persistence_failed"); persistErr != nil {
			return nil, worldMonitorPromotionOutcome{}, persistErr
		}
		return nil, promotionOutcomeIDs(inboxID, eventID, candidate.Symbol, "blocked", "candidate_economic_input_persistence_failed", err.Error(), candidateID.String(), nil), nil
	}
	// 02A3 persists only the explicit exposure request. Account-dependent
	// portfolio risk and the durable entry handoff are intentionally deferred.
	if err := store.MarkEconomicInputUnavailable(ctx, candidateID, "portfolio_risk_not_performed"); err != nil {
		return nil, worldMonitorPromotionOutcome{}, err
	}
	return nil, promotionOutcomeIDs(inboxID, eventID, candidate.Symbol, "blocked", "portfolio_risk_not_performed", "Canonical economic inputs were persisted; current PAPER account portfolio risk and the 02A4 handoff have not been performed.", candidateID.String(), nil), nil
}

func promotionOutcomeIDs(inboxID uuid.UUID, eventID uuid.NullUUID, symbol, status, reasonCode, reason, candidateID string, risk *candidatesmod.RiskReviewPersistence) worldMonitorPromotionOutcome {
	out := worldMonitorPromotionOutcome{InboxID: inboxID.String(), Symbol: symbol, Status: status, ReasonCode: reasonCode, Reason: reason, CandidateID: candidateID, RiskReview: risk}
	if eventID.Valid {
		out.EventID = eventID.UUID.String()
	}
	return out
}

func (p *worldMonitorOpportunityPromoter) promoteRows(ctx context.Context, rows []worldMonitorInboxPromotionRow) (worldMonitorPromotionResult, error) {
	result := worldMonitorPromotionResult{
		Promoted: []worldMonitorPromotedOpportunity{},
		Outcomes: []worldMonitorPromotionOutcome{},
	}
	for _, row := range rows {
		promoted, outcomes, err := p.promoteRow(ctx, row)
		if err != nil {
			return result, err
		}
		result.Outcomes = append(result.Outcomes, outcomes...)
		if promoted != nil {
			result.Promoted = append(result.Promoted, *promoted)
			result.PromotedCount++
		}
		for _, outcome := range outcomes {
			if outcome.Status != "promoted" {
				result.BlockedSkippedCount++
			}
		}
	}
	result.Skipped = result.BlockedSkippedCount
	return result, nil
}

func (p *worldMonitorOpportunityPromoter) loadPromotionRows(ctx context.Context, limit int) ([]worldMonitorInboxPromotionRow, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT
			w.id,
			w.source_event_id,
			w.normalized_event_id,
			e.raw_event_id,
			COALESCE(e.summary, ''),
			w.event_type,
			w.headline,
			COALESCE(w.summary, ''),
			COALESCE(w.source_urls, '[]'::jsonb),
			w.source_count,
			w.possible_affected_etfs,
			w.asset_themes,
			w.confidence,
			w.confidence_reasons,
			w.mapping_reason,
			w.event_time,
			e.is_synthetic,
			COALESCE(e.synthetic_reason, '')
		FROM world_monitor_research_inbox w
		JOIN event_normalized e ON e.id = w.normalized_event_id
		WHERE w.status = $1
		  AND w.candidate_id IS NULL
		  AND w.normalized_event_id IS NOT NULL
		  AND w.confidence >= $2
		ORDER BY w.received_at ASC
		LIMIT $3
	`, worldMonitorInboxStatusNew, worldMonitorPromoterMinConfidence, limit)
	if err != nil {
		return nil, fmt.Errorf("load world monitor promotion rows: %w", err)
	}
	defer rows.Close()

	out := []worldMonitorInboxPromotionRow{}
	for rows.Next() {
		var row worldMonitorInboxPromotionRow
		var sourceURLsRaw, etfsRaw, themesRaw, confidenceReasonsRaw []byte
		var normalizedEventID uuid.NullUUID
		var rawEventID uuid.NullUUID
		if err := rows.Scan(
			&row.ID,
			&row.SourceEventID,
			&normalizedEventID,
			&rawEventID,
			&row.NormalizedSummary,
			&row.EventType,
			&row.Headline,
			&row.Summary,
			&sourceURLsRaw,
			&row.SourceCount,
			&etfsRaw,
			&themesRaw,
			&row.Confidence,
			&confidenceReasonsRaw,
			&row.MappingReason,
			&row.EventTime,
			&row.IsSynthetic,
			&row.SyntheticReason,
		); err != nil {
			return nil, fmt.Errorf("scan world monitor promotion row: %w", err)
		}
		if normalizedEventID.Valid {
			v := normalizedEventID.UUID
			row.NormalizedEventID = &v
		}
		if rawEventID.Valid {
			v := rawEventID.UUID
			row.RawEventID = &v
		}
		_ = json.Unmarshal(sourceURLsRaw, &row.SourceURLs)
		_ = json.Unmarshal(etfsRaw, &row.PossibleAffectedETFs)
		_ = json.Unmarshal(themesRaw, &row.AssetThemes)
		_ = json.Unmarshal(confidenceReasonsRaw, &row.ConfidenceReasons)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (p *worldMonitorOpportunityPromoter) promoteRow(ctx context.Context, row worldMonitorInboxPromotionRow) (*worldMonitorPromotedOpportunity, []worldMonitorPromotionOutcome, error) {
	symbols := normalizeWorldMonitorPromotionSymbols(row.PossibleAffectedETFs)
	if len(symbols) == 0 {
		return nil, []worldMonitorPromotionOutcome{promotionOutcome(row, "", "skipped", "no_symbols", "No possible affected ETF symbols were supplied.", "")}, nil
	}
	outcomes := make([]worldMonitorPromotionOutcome, 0, len(symbols))
	for _, symbol := range symbols {
		promoted, outcome, usable, err := p.promoteSymbol(ctx, row, symbol)
		if err != nil {
			return nil, outcomes, err
		}
		outcomes = append(outcomes, outcome)
		if usable {
			return promoted, outcomes, nil
		}
	}
	return nil, outcomes, nil
}

func normalizeWorldMonitorPromotionSymbols(symbols []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(symbols))
	for _, raw := range symbols {
		symbol := strings.ToUpper(strings.TrimSpace(raw))
		if symbol == "" {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		out = append(out, symbol)
	}
	return out
}

func (p *worldMonitorOpportunityPromoter) promoteSymbol(ctx context.Context, row worldMonitorInboxPromotionRow, symbol string) (*worldMonitorPromotedOpportunity, worldMonitorPromotionOutcome, bool, error) {
	instanceID, strategyID, err := p.findStrategyInstance(ctx, symbol)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, promotionOutcome(row, symbol, "skipped", "no_enabled_strategy_instance", fmt.Sprintf("No compatible enabled ETF strategy instance is configured for %s.", symbol), ""), false, nil
		}
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	entryObservation, err := p.latestEntryPrice(ctx, symbol)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, errWorldMonitorNoUsableQuote) {
			return nil, promotionOutcome(row, symbol, "skipped", "no_quote", fmt.Sprintf("No usable quote is available for %s.", symbol), ""), false, nil
		}
		return nil, worldMonitorPromotionOutcome{}, false, err
	}

	entry := entryObservation.Last
	stop := roundPrice(entry * 0.98)
	target := roundPrice(entry * 1.04)
	confidence := row.Confidence
	expiresAt := p.now().Add(worldMonitorCandidateTTL)
	reasoning := p.reasoning(row, symbol)
	chart, err := p.confirmChart(ctx, symbol)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	if chart.ReasonCode == "insufficient_candles" {
		return nil, promotionOutcome(row, symbol, "skipped", chart.ReasonCode, chart.Reason, ""), false, nil
	}
	chart.EntryObservation = entryObservation

	candidateSvc := candidatesmod.NewService(candidatesmod.NewStore(p.pool))
	horizonPolicy := tradingmodes.SwingHorizonPolicy(3, 10)
	strategyTypeID, err := p.strategyTypeID(ctx, instanceID)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	structuredFields := worldMonitorStructuredCandidateFields(row, symbol, strategyTypeID, stop, chart)
	if !chart.Confirmed {
		candidate, err := candidateSvc.CreateBlocked(ctx, candidatesmod.BlockRequest{
			StrategyInstanceID:        instanceID,
			StructuredCandidateFields: structuredFields,
			StrategyID:                strategyID,
			Symbol:                    symbol,
			SignalType:                "BUY",
			EntryPrice:                &entry,
			StopLoss:                  &stop,
			TakeProfit:                &target,
			Confidence:                &confidence,
			Reasoning:                 &reasoning,
			DataProvenance:            "world-monitor",
			ReasonCode:                chart.ReasonCode,
			Reason:                    chart.Reason,
			TTL:                       worldMonitorCandidateTTL,
			HorizonPolicy:             &horizonPolicy,
			PaperOnly:                 true,
			ApprovalRequired:          true,
		})
		if err != nil {
			if errors.Is(err, candidatesmod.ErrDuplicateCandidate) || errors.Is(err, candidatesmod.ErrInstrumentPolicy) {
				return nil, promotionOutcome(row, symbol, "blocked", "candidate_validation_failed", err.Error(), ""), true, nil
			}
			return nil, worldMonitorPromotionOutcome{}, false, err
		}
		if err := p.attachCandidateMetadata(ctx, candidate.ID, row, uuid.Nil, symbol, strategyID, "blocked", chart, entry, stop, target); err != nil {
			return nil, worldMonitorPromotionOutcome{}, false, err
		}
		if err := p.markInboxCandidateCreated(ctx, row.ID, candidate.ID); err != nil {
			return nil, worldMonitorPromotionOutcome{}, false, err
		}
		eventID := ""
		if row.NormalizedEventID != nil {
			eventID = row.NormalizedEventID.String()
		}
		promoted := &worldMonitorPromotedOpportunity{
			InboxID:     row.ID.String(),
			EventID:     eventID,
			CandidateID: candidate.ID.String(),
			Symbol:      symbol,
			Route:       "blocked",
		}
		return promoted, promotionOutcome(row, symbol, "blocked", "chart_confirmation_failed", chart.Reason, candidate.ID.String()), true, nil
	}

	signalID, err := p.createStrategySignal(ctx, instanceID, strategyID, symbol, confidence, entry, stop, target, reasoning, expiresAt)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}

	candidate, err := candidateSvc.Propose(ctx, candidatesmod.ProposalRequest{
		StrategyInstanceID:        instanceID,
		StructuredCandidateFields: structuredFields,
		SignalID:                  signalID.String(),
		StrategyID:                strategyID,
		Symbol:                    symbol,
		SignalType:                "BUY",
		EntryPrice:                &entry,
		StopLoss:                  &stop,
		TakeProfit:                &target,
		Confidence:                &confidence,
		Reasoning:                 &reasoning,
		DataProvenance:            "world-monitor",
		TTL:                       worldMonitorCandidateTTL,
		HorizonPolicy:             &horizonPolicy,
		PaperOnly:                 true,
		ApprovalRequired:          true,
	})
	if err != nil {
		if errors.Is(err, candidatesmod.ErrDuplicateCandidate) || errors.Is(err, candidatesmod.ErrInstrumentPolicy) {
			return nil, promotionOutcome(row, symbol, "blocked", "candidate_validation_failed", err.Error(), ""), true, nil
		}
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	if err := candidateSvc.Qualify(ctx, candidate.ID); err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	qualified, err := candidateSvc.GetByID(ctx, candidate.ID)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	route := "evidence_review"
	if qualified.Status == candidatesmod.StatusBlocked {
		route = "blocked"
	}
	var evidence candidatesmod.EvidenceScoreSummary
	var gate candidatesmod.GateResult
	if route != "blocked" {
		evidence, gate, err = p.scoreAndPersistEvidence(ctx, *qualified, row, chart)
		if err != nil {
			return nil, worldMonitorPromotionOutcome{}, false, err
		}
		if gate.GateReady && gate.NextRequiredPhase == candidatesmod.NextPhaseRiskReview {
			route = "risk_review"
		}
	}
	if err := p.attachCandidateMetadata(ctx, candidate.ID, row, signalID, symbol, strategyID, route, chart, entry, stop, target); err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	eventID := ""
	if row.NormalizedEventID != nil {
		eventID = row.NormalizedEventID.String()
	}
	promoted := &worldMonitorPromotedOpportunity{
		InboxID:     row.ID.String(),
		EventID:     eventID,
		SignalID:    signalID.String(),
		CandidateID: candidate.ID.String(),
		Symbol:      symbol,
		Route:       route,
	}
	if route == "blocked" {
		reason := "Candidate validation blocked the candidate."
		if qualified.BlockReason != nil && strings.TrimSpace(*qualified.BlockReason) != "" {
			reason = *qualified.BlockReason
		}
		return promoted, promotionOutcome(row, symbol, "blocked", "candidate_validation_failed", reason, candidate.ID.String()), true, nil
	}
	if route != "risk_review" {
		reasonCode := "evidence_" + string(evidence.EvidenceStatus)
		reason := fmt.Sprintf("Candidate evidence scored %s (overall %.3f); trust gate requires %s.", evidence.EvidenceStatus, evidence.OverallEvidenceScore, gate.NextRequiredPhase)
		return promoted, promotionOutcome(row, symbol, "blocked", reasonCode, reason, candidate.ID.String()), true, nil
	}
	normalizedEventID := uuid.NullUUID{}
	if row.NormalizedEventID != nil {
		normalizedEventID = uuid.NullUUID{UUID: *row.NormalizedEventID, Valid: true}
	}
	riskPromoted, riskOutcome, err := p.reviewCandidateRisk(ctx, candidate.ID, row.ID, normalizedEventID)
	if err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	if riskPromoted != nil {
		promoted.Route = riskPromoted.Route
	}
	if err := p.markInboxCandidateCreated(ctx, row.ID, candidate.ID); err != nil {
		return nil, worldMonitorPromotionOutcome{}, false, err
	}
	return promoted, riskOutcome, true, nil
}

func (p *worldMonitorOpportunityPromoter) scoreAndPersistEvidence(ctx context.Context, candidate candidatesmod.Candidate, row worldMonitorInboxPromotionRow, chart worldMonitorChartConfirmation) (candidatesmod.EvidenceScoreSummary, candidatesmod.GateResult, error) {
	items := worldMonitorEvidenceItems(candidate.ID, candidate.Symbol, row, chart, p.now())
	score := candidatesmod.ScoreEvidenceForCandidate(candidate, items, p.now())
	gate := candidatesmod.EvaluateCandidateGate(candidate, score, p.now())
	if err := candidatesmod.NewStore(p.pool).PersistEvidenceEvaluation(ctx, items, score, gate); err != nil {
		return candidatesmod.EvidenceScoreSummary{}, candidatesmod.GateResult{}, err
	}
	return score, gate, nil
}

func worldMonitorEvidenceItems(candidateID uuid.UUID, symbol string, row worldMonitorInboxPromotionRow, chart worldMonitorChartConfirmation, now time.Time) []candidatesmod.EvidenceItem {
	monitorRef := "world_monitor_research_inbox:" + row.ID.String()
	if row.NormalizedEventID != nil {
		monitorRef = "event_normalized:" + row.NormalizedEventID.String()
	}
	monitorSummary := strings.TrimSpace(row.NormalizedSummary)
	if monitorSummary == "" {
		monitorSummary = strings.TrimSpace(row.Summary)
	}
	monitorObservedAt := row.EventTime
	monitorFreshness := worldMonitorEvidenceFreshness(monitorObservedAt, now)
	monitorQuality := math.Min(0.95, 0.60+float64(row.SourceCount)*0.10)
	if row.SourceCount <= 0 {
		monitorQuality = 0.50
	}
	items := []candidatesmod.EvidenceItem{{
		EvidenceID:        uuid.New(),
		CandidateID:       candidateID,
		SourceType:        "world_monitor",
		SourceRef:         monitorRef,
		ObservedAt:        monitorObservedAt,
		Summary:           monitorSummary,
		EvidenceKind:      "normalized_market_event",
		SupportsCandidate: true,
		Confidence:        row.Confidence,
		ImpactScore:       0.80,
		QualityScore:      monitorQuality,
		FreshnessStatus:   monitorFreshness,
	}}
	items = append(items, candidatesmod.EvidenceItem{
		EvidenceID:        uuid.New(),
		CandidateID:       candidateID,
		SourceType:        "market_candles",
		SourceRef:         fmt.Sprintf("candles:%s:%s", strings.ToUpper(strings.TrimSpace(symbol)), chart.CheckedAt.UTC().Format(time.RFC3339Nano)),
		ObservedAt:        chart.CheckedAt,
		Summary:           chart.Reason,
		EvidenceKind:      "chart_confirmation",
		SupportsCandidate: chart.Confirmed,
		Confidence:        0.90,
		ImpactScore:       0.80,
		QualityScore:      0.90,
		FreshnessStatus:   candidatesmod.FreshnessStatusFresh,
	})
	return items
}

func worldMonitorEvidenceFreshness(observedAt, now time.Time) candidatesmod.FreshnessStatus {
	if observedAt.IsZero() {
		return candidatesmod.FreshnessStatusStale
	}
	age := now.Sub(observedAt)
	if age <= 24*time.Hour {
		return candidatesmod.FreshnessStatusFresh
	}
	if age <= 48*time.Hour {
		return candidatesmod.FreshnessStatusStale
	}
	return candidatesmod.FreshnessStatusCriticalStale
}

func (p *worldMonitorOpportunityPromoter) strategyTypeID(ctx context.Context, instanceID uuid.UUID) (string, error) {
	var strategyTypeID string
	if err := p.pool.QueryRow(ctx, `SELECT strategy_type_id FROM strategy_instances WHERE id = $1`, instanceID).Scan(&strategyTypeID); err != nil {
		return "", fmt.Errorf("load strategy type for world monitor promotion: %w", err)
	}
	return strings.TrimSpace(strategyTypeID), nil
}

func worldMonitorStructuredCandidateFields(row worldMonitorInboxPromotionRow, symbol, strategyTypeID string, stop float64, chart worldMonitorChartConfirmation) candidatesmod.StructuredCandidateFields {
	catalystSummary := strings.TrimSpace(row.NormalizedSummary)
	if catalystSummary == "" {
		catalystSummary = strings.TrimSpace(row.Summary)
	}
	setupType := worldMonitorSetupType(strategyTypeID)
	invalidationReason := ""
	if setupType != "" && chart.Confirmed && stop > 0 {
		invalidationReason = fmt.Sprintf("%s trades at or below the candidate stop level %.2f, invalidating the confirmed sector-news momentum setup.", symbol, stop)
	}
	catalystType := strings.TrimSpace(row.EventType)
	catalystSource := "world-monitor"
	strategyFamily := strings.TrimSpace(strategyTypeID)
	rawSourceRef := ""
	if row.RawEventID != nil {
		rawSourceRef = "event_raw:" + row.RawEventID.String()
	}
	sourcePayloadRef := "world_monitor_research_inbox:" + row.ID.String()
	decisionLogRef := ""
	if row.NormalizedEventID != nil {
		decisionLogRef = "event_normalized:" + row.NormalizedEventID.String()
	}
	return candidatesmod.StructuredCandidateFields{
		Source:              "world-monitor",
		InstrumentType:      "etf",
		SetupType:           setupType,
		TimeHorizon:         "swing",
		StrategyFamily:      optionalWorldMonitorString(strategyFamily),
		CatalystType:        optionalWorldMonitorString(catalystType),
		CatalystSummary:     catalystSummary,
		CatalystSource:      &catalystSource,
		CatalystTimestamp:   &row.EventTime,
		CatalystConfidence:  &row.Confidence,
		EvidenceSourceCount: &row.SourceCount,
		InvalidationReason:  invalidationReason,
		RawSourceRef:        optionalWorldMonitorString(rawSourceRef),
		SourcePayloadRef:    &sourcePayloadRef,
		DecisionLogRef:      optionalWorldMonitorString(decisionLogRef),
		RejectReasons:       []string{},
	}
}

func worldMonitorSetupType(strategyTypeID string) string {
	if strings.EqualFold(strings.TrimSpace(strategyTypeID), "etf_news_sector_momentum_v1") {
		return "sector_news_momentum"
	}
	return ""
}

func optionalWorldMonitorString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func promotionOutcome(row worldMonitorInboxPromotionRow, symbol, status, reasonCode, reason, candidateID string) worldMonitorPromotionOutcome {
	eventID := ""
	if row.NormalizedEventID != nil {
		eventID = row.NormalizedEventID.String()
	}
	return worldMonitorPromotionOutcome{InboxID: row.ID.String(), EventID: eventID, Symbol: symbol, Status: status, ReasonCode: reasonCode, Reason: reason, CandidateID: candidateID}
}

func (p *worldMonitorOpportunityPromoter) findStrategyInstance(ctx context.Context, symbol string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var strategyID string
	err := p.pool.QueryRow(ctx, `
		SELECT id, COALESCE(NULLIF(strategy_id, ''), strategy_type_id)
		FROM strategy_instances
		WHERE enabled = TRUE
		  AND strategy_type_id LIKE 'etf_%'
		  AND config->'symbols' ? $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, symbol).Scan(&id, &strategyID)
	if err == nil {
		return id, strategyID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", fmt.Errorf("find matching strategy instance for %s: %w", symbol, err)
	}

	return uuid.Nil, "", pgx.ErrNoRows
}

func (p *worldMonitorOpportunityPromoter) confirmChart(ctx context.Context, symbol string) (worldMonitorChartConfirmation, error) {
	checkedAt := p.now()
	policy, err := p.marketDataPolicy()
	if err != nil {
		return worldMonitorChartConfirmation{}, fmt.Errorf("chart market-data policy unavailable: %w", err)
	}
	observations, err := loadBoundedCandles(ctx, p.pool, symbol, checkedAt, policy, 30)
	if err != nil {
		return worldMonitorChartConfirmation{}, fmt.Errorf("load chart candles for %s: %w", symbol, err)
	}
	closesDesc := make([]float64, 0, len(observations))
	for _, observation := range observations {
		if len(closesDesc) > 0 && observations[len(closesDesc)-1].Source != observation.Source {
			return worldMonitorChartConfirmation{}, fmt.Errorf("chart confirmation cannot mix market-data providers")
		}
		closesDesc = append(closesDesc, observation.Last)
	}
	if len(closesDesc) < 20 {
		return worldMonitorChartConfirmation{
			Confirmed:   false,
			ReasonCode:  "insufficient_candles",
			Reason:      fmt.Sprintf("Needs chart confirmation: only %d recent candles are available for %s; at least 20 are required.", len(closesDesc), symbol),
			CandleCount: len(closesDesc),
			CheckedAt:   checkedAt,
			Timeframe:   policy.Timeframe,
			AsOf:        checkedAt,
		}, nil
	}

	closes := make([]float64, len(closesDesc))
	for i := range closesDesc {
		closes[len(closesDesc)-1-i] = closesDesc[i]
	}
	last := closes[len(closes)-1]
	window := closes[len(closes)-20:]
	var sum float64
	for _, close := range window {
		sum += close
	}
	sma20 := sum / 20
	fiveStart := closes[len(closes)-5]
	fiveChange := 0.0
	if fiveStart > 0 {
		fiveChange = ((last - fiveStart) / fiveStart) * 100
	}

	out := worldMonitorChartConfirmation{
		CandleCount:         len(closes),
		LastClose:           roundPrice(last),
		SMA20:               roundPrice(sma20),
		FiveCandleChangePct: math.Round(fiveChange*100) / 100,
		CheckedAt:           checkedAt,
		Source:              observations[0].Source,
		Timeframe:           policy.Timeframe,
		AsOf:                checkedAt,
		EarliestCandleAt:    observations[len(observations)-1].ProviderAt,
		LatestCandleAt:      observations[0].ProviderAt,
		LatestReceivedAt:    observations[0].ReceivedAt,
	}
	if last < sma20 {
		out.Confirmed = false
		out.ReasonCode = "no_chart_confirmation"
		out.Reason = fmt.Sprintf("Needs chart confirmation: %s latest close %.2f is below its 20-candle average %.2f.", symbol, last, sma20)
		return out, nil
	}
	if fiveChange < -0.5 {
		out.Confirmed = false
		out.ReasonCode = "no_chart_confirmation"
		out.Reason = fmt.Sprintf("Needs chart confirmation: %s has fallen %.2f%% over the last five candles.", symbol, math.Abs(fiveChange))
		return out, nil
	}
	out.Confirmed = true
	out.ReasonCode = "chart_confirmed"
	out.Reason = fmt.Sprintf("Chart confirmed: %s latest close %.2f is above its 20-candle average %.2f and recent momentum is not materially negative.", symbol, last, sma20)
	return out, nil
}

func (p *worldMonitorOpportunityPromoter) latestEntryPrice(ctx context.Context, symbol string) (marketdata.EconomicObservation, error) {
	policy, err := p.marketDataPolicy()
	if err != nil {
		return marketdata.EconomicObservation{}, fmt.Errorf("entry market-data policy unavailable: %w", err)
	}
	observation, err := loadCanonicalQuoteObservation(ctx, p.pool, symbol, p.now().UTC(), policy)
	if err != nil {
		return marketdata.EconomicObservation{}, fmt.Errorf("latest quote for %s: %w", symbol, err)
	}
	return observation, nil
}

func (p *worldMonitorOpportunityPromoter) marketDataPolicy() (marketDataSafetyPolicy, error) {
	if p.marketPolicy != nil {
		return *p.marketPolicy, nil
	}
	return marketDataSafetyPolicyFromEnv()
}

func (p *worldMonitorOpportunityPromoter) createStrategySignal(ctx context.Context, instanceID uuid.UUID, strategyID, symbol string, confidence, entry, stop, target float64, reasoning string, expiresAt time.Time) (uuid.UUID, error) {
	signalID := uuid.New()
	_, err := p.pool.Exec(ctx, `
		INSERT INTO strategy_signals (
			id, symbol, strategy_id, signal_type, confidence, entry_price, stop_loss,
			take_profit, reasoning, generated_at, expires_at, status, instance_id
		)
		VALUES ($1, $2, $3, 'BUY', $4, $5, $6, $7, $8, NOW(), $9, 'pending', $10)
	`, signalID, symbol, strategyID, confidence, entry, stop, target, reasoning, expiresAt.UTC(), instanceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create world monitor strategy signal: %w", err)
	}
	return signalID, nil
}

func (p *worldMonitorOpportunityPromoter) attachCandidateMetadata(ctx context.Context, candidateID uuid.UUID, row worldMonitorInboxPromotionRow, signalID uuid.UUID, symbol, strategyID, route string, chart worldMonitorChartConfirmation, entry, stop, target float64) error {
	eventID := ""
	if row.NormalizedEventID != nil {
		eventID = row.NormalizedEventID.String()
	}
	signalIDValue := ""
	if signalID != uuid.Nil {
		signalIDValue = signalID.String()
	}
	metadata := map[string]any{
		"source":            "world-monitor",
		"sourceEventId":     row.SourceEventID,
		"normalizedEventId": eventID,
		"eventType":         row.EventType,
		"headline":          row.Headline,
		"summary":           row.Summary,
		"sourceURLs":        row.SourceURLs,
		"sourceCount":       row.SourceCount,
		"assetThemes":       row.AssetThemes,
		"confidenceReasons": row.ConfidenceReasons,
		"mappingReason":     row.MappingReason,
		"promotedSymbol":    symbol,
		"strategyId":        strategyID,
		"signalId":          signalIDValue,
		"route":             route,
		"isSynthetic":       row.IsSynthetic,
		"syntheticReason":   row.SyntheticReason,
	}
	payload, _ := json.Marshal(map[string]any{
		"worldMonitor":      metadata,
		"chartConfirmation": chart,
	})
	_, err := p.pool.Exec(ctx, `
		UPDATE candidate_trades
		SET metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb,
		    updated_at = NOW()
		WHERE id = $1
	`, candidateID, string(payload))
	if err != nil {
		return fmt.Errorf("attach world monitor candidate metadata: %w", err)
	}
	return nil
}

func (p *worldMonitorOpportunityPromoter) markInboxCandidateCreated(ctx context.Context, inboxID, candidateID uuid.UUID) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE world_monitor_research_inbox
		SET status = $2,
		    candidate_id = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, inboxID, worldMonitorInboxStatusCandidateCreated, candidateID)
	if err != nil {
		return fmt.Errorf("mark world monitor inbox candidate created: %w", err)
	}
	return nil
}

func (p *worldMonitorOpportunityPromoter) reasoning(row worldMonitorInboxPromotionRow, symbol string) string {
	parts := []string{
		fmt.Sprintf("World Monitor highlighted %s for %s.", symbol, strings.TrimSpace(row.Headline)),
		strings.TrimSpace(row.MappingReason),
	}
	if strings.TrimSpace(row.Summary) != "" {
		parts = append(parts, strings.TrimSpace(row.Summary))
	}
	return strings.Join(nonEmptyWorldMonitorStrings(parts), " ")
}

func roundPrice(value float64) float64 {
	return math.Round(value*100) / 100
}
