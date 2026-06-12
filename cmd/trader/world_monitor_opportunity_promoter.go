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
)

const (
	worldMonitorPromoterDefaultLimit  = 10
	worldMonitorPromoterMaxLimit      = 50
	worldMonitorPromoterMinConfidence = 0.55
	worldMonitorCandidateTTL          = 45 * time.Minute
)

type worldMonitorOpportunityPromoter struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

type worldMonitorPromotionResult struct {
	Promoted []worldMonitorPromotedOpportunity `json:"promoted"`
	Skipped  int                               `json:"skipped"`
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
	EventType            string
	Headline             string
	Summary              string
	PossibleAffectedETFs []string
	AssetThemes          []string
	Confidence           float64
	ConfidenceReasons    []string
	MappingReason        string
	EventTime            time.Time
}

type worldMonitorChartConfirmation struct {
	Confirmed           bool      `json:"confirmed"`
	ReasonCode          string    `json:"reasonCode"`
	Reason              string    `json:"reason"`
	CandleCount         int       `json:"candleCount"`
	LastClose           float64   `json:"lastClose,omitempty"`
	SMA20               float64   `json:"sma20,omitempty"`
	FiveCandleChangePct float64   `json:"fiveCandleChangePct,omitempty"`
	CheckedAt           time.Time `json:"checkedAt"`
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

	rows, err := p.loadPromotionRows(ctx, limit)
	if err != nil {
		return worldMonitorPromotionResult{}, err
	}

	result := worldMonitorPromotionResult{Promoted: []worldMonitorPromotedOpportunity{}}
	for _, row := range rows {
		promoted, err := p.promoteRow(ctx, row)
		if err != nil {
			if errors.Is(err, candidatesmod.ErrDuplicateCandidate) || errors.Is(err, pgx.ErrNoRows) {
				result.Skipped++
				continue
			}
			return result, err
		}
		result.Promoted = append(result.Promoted, promoted)
	}
	return result, nil
}

func (p *worldMonitorOpportunityPromoter) loadPromotionRows(ctx context.Context, limit int) ([]worldMonitorInboxPromotionRow, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT
			id,
			source_event_id,
			normalized_event_id,
			event_type,
			headline,
			COALESCE(summary, ''),
			possible_affected_etfs,
			asset_themes,
			confidence,
			confidence_reasons,
			mapping_reason,
			event_time
		FROM world_monitor_research_inbox
		WHERE status = $1
		  AND candidate_id IS NULL
		  AND normalized_event_id IS NOT NULL
		  AND confidence >= $2
		ORDER BY received_at ASC
		LIMIT $3
	`, worldMonitorInboxStatusNew, worldMonitorPromoterMinConfidence, limit)
	if err != nil {
		return nil, fmt.Errorf("load world monitor promotion rows: %w", err)
	}
	defer rows.Close()

	out := []worldMonitorInboxPromotionRow{}
	for rows.Next() {
		var row worldMonitorInboxPromotionRow
		var etfsRaw, themesRaw, confidenceReasonsRaw []byte
		var normalizedEventID uuid.NullUUID
		if err := rows.Scan(
			&row.ID,
			&row.SourceEventID,
			&normalizedEventID,
			&row.EventType,
			&row.Headline,
			&row.Summary,
			&etfsRaw,
			&themesRaw,
			&row.Confidence,
			&confidenceReasonsRaw,
			&row.MappingReason,
			&row.EventTime,
		); err != nil {
			return nil, fmt.Errorf("scan world monitor promotion row: %w", err)
		}
		if normalizedEventID.Valid {
			v := normalizedEventID.UUID
			row.NormalizedEventID = &v
		}
		_ = json.Unmarshal(etfsRaw, &row.PossibleAffectedETFs)
		_ = json.Unmarshal(themesRaw, &row.AssetThemes)
		_ = json.Unmarshal(confidenceReasonsRaw, &row.ConfidenceReasons)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (p *worldMonitorOpportunityPromoter) promoteRow(ctx context.Context, row worldMonitorInboxPromotionRow) (worldMonitorPromotedOpportunity, error) {
	symbols := normalizeSymbols("", row.PossibleAffectedETFs)
	if len(symbols) == 0 {
		return worldMonitorPromotedOpportunity{}, pgx.ErrNoRows
	}
	symbol := symbols[0]

	instanceID, strategyID, err := p.findStrategyInstance(ctx, symbol)
	if err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}
	entry, err := p.latestEntryPrice(ctx, symbol)
	if err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}

	stop := roundPrice(entry * 0.98)
	target := roundPrice(entry * 1.04)
	confidence := row.Confidence
	expiresAt := p.now().Add(worldMonitorCandidateTTL)
	reasoning := p.reasoning(row, symbol)
	chart, err := p.confirmChart(ctx, symbol)
	if err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}

	candidateSvc := candidatesmod.NewService(candidatesmod.NewStore(p.pool))
	if !chart.Confirmed {
		candidate, err := candidateSvc.CreateBlocked(ctx, candidatesmod.BlockRequest{
			StrategyInstanceID: instanceID,
			StrategyID:         strategyID,
			Symbol:             symbol,
			SignalType:         "BUY",
			EntryPrice:         &entry,
			StopLoss:           &stop,
			TakeProfit:         &target,
			Confidence:         &confidence,
			Reasoning:          &reasoning,
			DataProvenance:     "world-monitor",
			ReasonCode:         chart.ReasonCode,
			Reason:             chart.Reason,
			TTL:                worldMonitorCandidateTTL,
		})
		if err != nil {
			return worldMonitorPromotedOpportunity{}, err
		}
		if err := p.attachCandidateMetadata(ctx, candidate.ID, row, uuid.Nil, symbol, strategyID, "blocked", chart); err != nil {
			return worldMonitorPromotedOpportunity{}, err
		}
		if err := p.markInboxCandidateCreated(ctx, row.ID, candidate.ID); err != nil {
			return worldMonitorPromotedOpportunity{}, err
		}
		eventID := ""
		if row.NormalizedEventID != nil {
			eventID = row.NormalizedEventID.String()
		}
		return worldMonitorPromotedOpportunity{
			InboxID:     row.ID.String(),
			EventID:     eventID,
			CandidateID: candidate.ID.String(),
			Symbol:      symbol,
			Route:       "blocked",
		}, nil
	}

	signalID, err := p.createStrategySignal(ctx, instanceID, strategyID, symbol, confidence, entry, stop, target, reasoning, expiresAt)
	if err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}

	candidate, err := candidateSvc.Propose(ctx, candidatesmod.ProposalRequest{
		StrategyInstanceID: instanceID,
		SignalID:           signalID.String(),
		StrategyID:         strategyID,
		Symbol:             symbol,
		SignalType:         "BUY",
		EntryPrice:         &entry,
		StopLoss:           &stop,
		TakeProfit:         &target,
		Confidence:         &confidence,
		Reasoning:          &reasoning,
		DataProvenance:     "world-monitor",
		TTL:                worldMonitorCandidateTTL,
	})
	if err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}
	if err := candidateSvc.Qualify(ctx, candidate.ID); err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}
	if err := p.attachCandidateMetadata(ctx, candidate.ID, row, signalID, symbol, strategyID, "approval_required", chart); err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}
	if err := p.markInboxCandidateCreated(ctx, row.ID, candidate.ID); err != nil {
		return worldMonitorPromotedOpportunity{}, err
	}

	eventID := ""
	if row.NormalizedEventID != nil {
		eventID = row.NormalizedEventID.String()
	}
	return worldMonitorPromotedOpportunity{
		InboxID:     row.ID.String(),
		EventID:     eventID,
		SignalID:    signalID.String(),
		CandidateID: candidate.ID.String(),
		Symbol:      symbol,
		Route:       "approval_required",
	}, nil
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

	preferred := []string{
		"etf-news-sector-momentum-paper-v1",
		"etf-news-rates-rotation-paper-v1",
		"etf-news-market-panic-paper-v1",
		"or-spy-paper-v1",
	}
	for _, name := range preferred {
		err = p.pool.QueryRow(ctx, `
			SELECT id, COALESCE(NULLIF(strategy_id, ''), strategy_type_id)
			FROM strategy_instances
			WHERE name = $1
			  AND enabled = TRUE
			  AND (config->'symbols' ? $2 OR name = 'or-spy-paper-v1')
			LIMIT 1
		`, name, symbol).Scan(&id, &strategyID)
		if err == nil {
			return id, strategyID, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, "", fmt.Errorf("find strategy instance: %w", err)
		}
	}

	err = p.pool.QueryRow(ctx, `
		SELECT id, COALESCE(NULLIF(strategy_id, ''), strategy_type_id)
		FROM strategy_instances
		WHERE strategy_type_id LIKE 'etf_%'
		ORDER BY updated_at DESC
		LIMIT 1
	`).Scan(&id, &strategyID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("find fallback strategy instance for %s: %w", symbol, err)
	}
	return id, strategyID, nil
}

func (p *worldMonitorOpportunityPromoter) confirmChart(ctx context.Context, symbol string) (worldMonitorChartConfirmation, error) {
	checkedAt := p.now()
	rows, err := p.pool.Query(ctx, `
		SELECT close::float8
		FROM candles
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT 30
	`, symbol)
	if err != nil {
		return worldMonitorChartConfirmation{}, fmt.Errorf("load chart candles for %s: %w", symbol, err)
	}
	defer rows.Close()

	closesDesc := []float64{}
	for rows.Next() {
		var close float64
		if err := rows.Scan(&close); err != nil {
			return worldMonitorChartConfirmation{}, fmt.Errorf("scan chart candle for %s: %w", symbol, err)
		}
		if close > 0 {
			closesDesc = append(closesDesc, close)
		}
	}
	if err := rows.Err(); err != nil {
		return worldMonitorChartConfirmation{}, err
	}
	if len(closesDesc) < 20 {
		return worldMonitorChartConfirmation{
			Confirmed:   false,
			ReasonCode:  "no_chart_confirmation",
			Reason:      fmt.Sprintf("Needs chart confirmation: only %d recent candles are available for %s; at least 20 are required.", len(closesDesc), symbol),
			CandleCount: len(closesDesc),
			CheckedAt:   checkedAt,
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

func (p *worldMonitorOpportunityPromoter) latestEntryPrice(ctx context.Context, symbol string) (float64, error) {
	var price float64
	err := p.pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(price, 0), NULLIF((COALESCE(bid, 0) + COALESCE(ask, 0)) / 2, 0))
		FROM quotes
		WHERE symbol = $1
	`, symbol).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("latest quote for %s: %w", symbol, err)
	}
	if price <= 0 {
		return 0, fmt.Errorf("latest quote for %s has no usable price", symbol)
	}
	return roundPrice(price), nil
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

func (p *worldMonitorOpportunityPromoter) attachCandidateMetadata(ctx context.Context, candidateID uuid.UUID, row worldMonitorInboxPromotionRow, signalID uuid.UUID, symbol, strategyID, route string, chart worldMonitorChartConfirmation) error {
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
		"assetThemes":       row.AssetThemes,
		"confidenceReasons": row.ConfidenceReasons,
		"mappingReason":     row.MappingReason,
		"promotedSymbol":    symbol,
		"strategyId":        strategyID,
		"signalId":          signalIDValue,
		"route":             route,
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
