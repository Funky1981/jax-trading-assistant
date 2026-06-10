package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type macroEventDTO struct {
	ID              string           `json:"id"`
	Source          string           `json:"source"`
	SourceEventID   string           `json:"sourceEventId"`
	EventType       string           `json:"eventType"`
	Region          string           `json:"region"`
	EventTimeUTC    time.Time        `json:"eventTimeUtc"`
	Headline        string           `json:"headline"`
	Summary         string           `json:"summary,omitempty"`
	ActualValue     any              `json:"actualValue,omitempty"`
	ExpectedValue   any              `json:"expectedValue,omitempty"`
	PreviousValue   any              `json:"previousValue,omitempty"`
	Unit            string           `json:"unit,omitempty"`
	SurpriseValue   any              `json:"surpriseValue,omitempty"`
	SurprisePercent any              `json:"surprisePercent,omitempty"`
	Direction       string           `json:"direction"`
	Confidence      float64          `json:"confidence"`
	Status          string           `json:"status"`
	RawPayload      json.RawMessage  `json:"rawPayload,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	ETFMappings     []macroETFMapDTO `json:"etfMappings,omitempty"`
	CandidateCount  int              `json:"candidateCount"`
	EvidenceCount   int              `json:"evidenceCount"`
}

type macroETFMapDTO struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Theme         string    `json:"theme"`
	MappingReason string    `json:"mappingReason"`
	Confidence    float64   `json:"confidence"`
	CreatedAt     time.Time `json:"createdAt"`
}

type macroReactionDTO struct {
	ID            string          `json:"id"`
	Symbol        string          `json:"symbol"`
	Timeframe     string          `json:"timeframe"`
	PrePrice      float64         `json:"prePrice"`
	PostPrice     float64         `json:"postPrice"`
	ChangeAbs     float64         `json:"changeAbs"`
	ChangePercent float64         `json:"changePercent"`
	HighAfter     any             `json:"highAfter,omitempty"`
	LowAfter      any             `json:"lowAfter,omitempty"`
	VolumeRatio   any             `json:"volumeRatio,omitempty"`
	ATRRatio      any             `json:"atrRatio,omitempty"`
	Direction     string          `json:"direction"`
	ConfirmsEvent bool            `json:"confirmsEvent"`
	TooExtended   bool            `json:"tooExtended"`
	Noisy         bool            `json:"noisy"`
	Reason        string          `json:"reason"`
	RawCandles    json.RawMessage `json:"rawCandles,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type macroTechnicalDTO struct {
	ID                string          `json:"id"`
	Symbol            string          `json:"symbol"`
	Timeframe         string          `json:"timeframe"`
	AnalysisTimeUTC   time.Time       `json:"analysisTimeUtc"`
	TrendState        string          `json:"trendState"`
	StructureState    string          `json:"structureState"`
	KeyLevels         json.RawMessage `json:"keyLevels"`
	EventReaction     json.RawMessage `json:"eventReaction"`
	VolumeVolatility  json.RawMessage `json:"volumeVolatility"`
	RelativeStrength  json.RawMessage `json:"relativeStrength"`
	TechnicalScore    float64         `json:"technicalScore"`
	Verdict           string          `json:"verdict"`
	Reasons           []string        `json:"reasons"`
	InvalidationRules []string        `json:"invalidationRules"`
	CreatedAt         time.Time       `json:"createdAt"`
}

type macroScenarioDTO struct {
	ID                    string          `json:"id"`
	ScenarioKey           string          `json:"scenarioKey"`
	CandidateBias         string          `json:"candidateBias"`
	PrimarySymbols        []string        `json:"primarySymbols"`
	SecondarySymbols      []string        `json:"secondarySymbols"`
	RequiredConfirmations []string        `json:"requiredConfirmations"`
	ExpectedReactions     json.RawMessage `json:"expectedReactions"`
	Result                string          `json:"result"`
	Reason                string          `json:"reason"`
	CreatedAt             time.Time       `json:"createdAt"`
}

type macroPricedInDTO struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Verdict   string    `json:"verdict"`
	Score     float64   `json:"score"`
	Reasons   []string  `json:"reasons"`
	CreatedAt time.Time `json:"createdAt"`
}

type macroConfounderDTO struct {
	ID             string    `json:"id"`
	ConfounderType string    `json:"confounderType"`
	Headline       string    `json:"headline"`
	Source         string    `json:"source,omitempty"`
	Severity       string    `json:"severity"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"createdAt"`
}

type macroEvidenceBundleDTO struct {
	ID              string          `json:"id"`
	Symbol          string          `json:"symbol"`
	Status          string          `json:"status"`
	Verdict         string          `json:"verdict"`
	Summary         string          `json:"summary"`
	Evidence        json.RawMessage `json:"evidence"`
	MissingEvidence []string        `json:"missingEvidence"`
	WalkawayReasons []string        `json:"walkawayReasons"`
	CreatedAt       time.Time       `json:"createdAt"`
}

type macroCandidateDTO struct {
	ID                    string    `json:"id"`
	MacroEventID          string    `json:"macroEventId"`
	EvidenceBundleID      string    `json:"evidenceBundleId"`
	Symbol                string    `json:"symbol"`
	Side                  string    `json:"side"`
	Bias                  string    `json:"bias"`
	EntryType             string    `json:"entryType"`
	EntryReferencePrice   float64   `json:"entryReferencePrice"`
	StopReferencePrice    float64   `json:"stopReferencePrice"`
	TargetReferencePrice  float64   `json:"targetReferencePrice"`
	RiskPercent           float64   `json:"riskPercent"`
	TimeLimit             string    `json:"timeLimit"`
	Status                string    `json:"status"`
	CreatedReason         string    `json:"createdReason"`
	RejectionReason       string    `json:"rejectionReason,omitempty"`
	WalkawayReasons       []string  `json:"walkawayReasons"`
	CreatedAt             time.Time `json:"createdAt"`
	HumanApprovalRequired bool      `json:"humanApprovalRequired"`
}

type macroEventDetailDTO struct {
	Event             macroEventDTO            `json:"event"`
	Reactions         []macroReactionDTO       `json:"reactions"`
	TechnicalAnalysis []macroTechnicalDTO      `json:"technicalAnalysis"`
	Scenarios         []macroScenarioDTO       `json:"scenarios"`
	PricedInScores    []macroPricedInDTO       `json:"pricedInScores"`
	Confounders       []macroConfounderDTO     `json:"confounders"`
	EvidenceBundles   []macroEvidenceBundleDTO `json:"evidenceBundles"`
	Candidates        []macroCandidateDTO      `json:"candidates"`
}

func macroEventsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if pool == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		limit := parseIntParam(r.URL.Query().Get("limit"), 50)
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		offset := parseIntParam(r.URL.Query().Get("offset"), 0)
		if offset < 0 {
			offset = 0
		}
		rows, err := pool.Query(r.Context(), `
			SELECT
				me.id::text, me.source, me.source_event_id, me.event_type, me.region, me.event_time_utc,
				me.headline, COALESCE(me.summary,''), me.actual_value::float8, me.expected_value::float8,
				me.previous_value::float8, COALESCE(me.unit,''), me.surprise_value::float8,
				me.surprise_percent::float8, me.direction, me.confidence::float8, me.status,
				me.raw_payload::text, me.created_at, me.updated_at,
				COUNT(DISTINCT eb.id)::int AS evidence_count,
				COUNT(DISTINCT ct.id)::int AS candidate_count
			FROM macro_events me
			LEFT JOIN macro_evidence_bundles eb ON eb.macro_event_id = me.id
			LEFT JOIN macro_candidate_trades ct ON ct.macro_event_id = me.id
			GROUP BY me.id
			ORDER BY me.event_time_utc DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		events := make([]macroEventDTO, 0, limit)
		for rows.Next() {
			event, err := scanMacroEvent(rows)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			events = append(events, event)
		}
		var total int
		if err := pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM macro_events`).Scan(&total); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := attachMacroETFMappings(r.Context(), pool, events); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]any{"events": events, "total": total, "limit": limit, "offset": offset})
	}
}

func macroEventDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/macro/events/"), "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		eventID := parts[0]
		if _, err := uuid.Parse(eventID); err != nil {
			http.Error(w, "invalid macro event id", http.StatusBadRequest)
			return
		}
		action := ""
		if len(parts) > 1 {
			action = parts[1]
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch action {
		case "":
			detail, err := loadMacroEventDetail(r, pool, eventID)
			if err != nil {
				if err == pgx.ErrNoRows {
					http.NotFound(w, r)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, detail)
		case "reactions":
			reactions, err := loadMacroReactions(r, pool, eventID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, map[string]any{"macroEventId": eventID, "reactions": reactions})
		case "evidence":
			evidence, err := loadMacroEvidenceBundles(r, pool, eventID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			jsonOK(w, map[string]any{"macroEventId": eventID, "evidenceBundles": evidence})
		default:
			http.NotFound(w, r)
		}
	}
}

func macroCandidateDetailHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if pool == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/macro/candidates/"), "/"), "/")
		if len(parts) < 2 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		candidateID := parts[0]
		if _, err := uuid.Parse(candidateID); err != nil {
			http.Error(w, "invalid macro candidate id", http.StatusBadRequest)
			return
		}
		action := parts[1]
		if r.Method != http.MethodPost || (action != "approve" && action != "reject") {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Notes string `json:"notes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		status := "watch_only"
		reason := strings.TrimSpace(req.Notes)
		if action == "reject" {
			status = "rejected"
			if reason == "" {
				reason = "Rejected by operator"
			}
		}
		row := pool.QueryRow(r.Context(), `
			UPDATE macro_candidate_trades
			SET status = $2, rejection_reason = NULLIF($3, '')
			WHERE id = $1::uuid
			RETURNING id::text, macro_event_id::text, evidence_bundle_id::text, symbol, side, bias,
				entry_type, entry_reference_price::float8, stop_reference_price::float8,
				target_reference_price::float8, risk_percent::float8, time_limit, status,
				created_reason, COALESCE(rejection_reason,''), walkaway_reasons, created_at
		`, candidateID, status, reason)
		candidate, err := scanMacroCandidate(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		jsonOK(w, candidate)
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMacroEvent(row rowScanner) (macroEventDTO, error) {
	var event macroEventDTO
	var actual, expected, previous, surprise, surprisePct sql.NullFloat64
	var raw string
	err := row.Scan(
		&event.ID, &event.Source, &event.SourceEventID, &event.EventType, &event.Region, &event.EventTimeUTC,
		&event.Headline, &event.Summary, &actual, &expected, &previous, &event.Unit, &surprise,
		&surprisePct, &event.Direction, &event.Confidence, &event.Status, &raw, &event.CreatedAt,
		&event.UpdatedAt, &event.EvidenceCount, &event.CandidateCount,
	)
	if err != nil {
		return event, err
	}
	event.ActualValue = nullableFloat(actual)
	event.ExpectedValue = nullableFloat(expected)
	event.PreviousValue = nullableFloat(previous)
	event.SurpriseValue = nullableFloat(surprise)
	event.SurprisePercent = nullableFloat(surprisePct)
	event.RawPayload = json.RawMessage(defaultJSON(raw, "{}"))
	return event, nil
}

func scanMacroCandidate(row rowScanner) (macroCandidateDTO, error) {
	var candidate macroCandidateDTO
	err := row.Scan(
		&candidate.ID, &candidate.MacroEventID, &candidate.EvidenceBundleID, &candidate.Symbol,
		&candidate.Side, &candidate.Bias, &candidate.EntryType, &candidate.EntryReferencePrice,
		&candidate.StopReferencePrice, &candidate.TargetReferencePrice, &candidate.RiskPercent,
		&candidate.TimeLimit, &candidate.Status, &candidate.CreatedReason, &candidate.RejectionReason,
		&candidate.WalkawayReasons, &candidate.CreatedAt,
	)
	candidate.HumanApprovalRequired = candidate.Status == "awaiting_human_approval"
	return candidate, err
}

func loadMacroEventDetail(r *http.Request, pool *pgxpool.Pool, eventID string) (macroEventDetailDTO, error) {
	var detail macroEventDetailDTO
	row := pool.QueryRow(r.Context(), `
		SELECT
			me.id::text, me.source, me.source_event_id, me.event_type, me.region, me.event_time_utc,
			me.headline, COALESCE(me.summary,''), me.actual_value::float8, me.expected_value::float8,
			me.previous_value::float8, COALESCE(me.unit,''), me.surprise_value::float8,
			me.surprise_percent::float8, me.direction, me.confidence::float8, me.status,
			me.raw_payload::text, me.created_at, me.updated_at,
			(SELECT COUNT(*) FROM macro_evidence_bundles eb WHERE eb.macro_event_id = me.id)::int,
			(SELECT COUNT(*) FROM macro_candidate_trades ct WHERE ct.macro_event_id = me.id)::int
		FROM macro_events me
		WHERE me.id = $1::uuid
	`, eventID)
	event, err := scanMacroEvent(row)
	if err != nil {
		return detail, err
	}
	detail.Event = event
	events := []macroEventDTO{detail.Event}
	if err := attachMacroETFMappings(r.Context(), pool, events); err != nil {
		return detail, err
	}
	detail.Event = events[0]
	if detail.Reactions, err = loadMacroReactions(r, pool, eventID); err != nil {
		return detail, err
	}
	if detail.TechnicalAnalysis, err = loadMacroTechnicalAnalysis(r, pool, eventID); err != nil {
		return detail, err
	}
	if detail.Scenarios, err = loadMacroScenarios(r, pool, eventID); err != nil {
		return detail, err
	}
	if detail.PricedInScores, err = loadMacroPricedInScores(r, pool, eventID); err != nil {
		return detail, err
	}
	if detail.Confounders, err = loadMacroConfounders(r, pool, eventID); err != nil {
		return detail, err
	}
	if detail.EvidenceBundles, err = loadMacroEvidenceBundles(r, pool, eventID); err != nil {
		return detail, err
	}
	detail.Candidates, err = loadMacroCandidates(r, pool, eventID)
	return detail, err
}

func attachMacroETFMappings(ctx context.Context, pool *pgxpool.Pool, events []macroEventDTO) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]string, 0, len(events))
	byID := map[string]int{}
	for i := range events {
		ids = append(ids, events[i].ID)
		byID[events[i].ID] = i
	}
	rows, err := pool.Query(ctx, `
		SELECT macro_event_id::text, id::text, symbol, theme, mapping_reason, confidence::float8, created_at
		FROM macro_event_etf_map
		WHERE macro_event_id = ANY($1::uuid[])
		ORDER BY symbol
	`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID string
		var mapping macroETFMapDTO
		if err := rows.Scan(&eventID, &mapping.ID, &mapping.Symbol, &mapping.Theme, &mapping.MappingReason, &mapping.Confidence, &mapping.CreatedAt); err != nil {
			return err
		}
		if idx, ok := byID[eventID]; ok {
			events[idx].ETFMappings = append(events[idx].ETFMappings, mapping)
		}
	}
	return rows.Err()
}

func loadMacroReactions(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroReactionDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, symbol, timeframe, pre_price::float8, post_price::float8, change_abs::float8,
			change_percent::float8, high_after::float8, low_after::float8, volume_ratio::float8,
			atr_ratio::float8, direction, confirms_event, too_extended, noisy, reason,
			raw_candles::text, created_at
		FROM macro_reaction_snapshots
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC, symbol
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroReactionDTO{}
	for rows.Next() {
		var item macroReactionDTO
		var high, low, volume, atr sql.NullFloat64
		var raw string
		if err := rows.Scan(&item.ID, &item.Symbol, &item.Timeframe, &item.PrePrice, &item.PostPrice, &item.ChangeAbs,
			&item.ChangePercent, &high, &low, &volume, &atr, &item.Direction, &item.ConfirmsEvent,
			&item.TooExtended, &item.Noisy, &item.Reason, &raw, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.HighAfter = nullableFloat(high)
		item.LowAfter = nullableFloat(low)
		item.VolumeRatio = nullableFloat(volume)
		item.ATRRatio = nullableFloat(atr)
		item.RawCandles = json.RawMessage(defaultJSON(raw, "[]"))
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroTechnicalAnalysis(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroTechnicalDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, symbol, timeframe, analysis_time_utc, trend_state, structure_state,
			key_levels::text, event_reaction::text, volume_volatility::text, relative_strength::text,
			technical_score::float8, verdict, reasons, invalidation_rules, created_at
		FROM technical_analysis_snapshots
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC, symbol, timeframe
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroTechnicalDTO{}
	for rows.Next() {
		var item macroTechnicalDTO
		var keyLevels, eventReaction, volumeVolatility, relativeStrength string
		if err := rows.Scan(&item.ID, &item.Symbol, &item.Timeframe, &item.AnalysisTimeUTC, &item.TrendState, &item.StructureState,
			&keyLevels, &eventReaction, &volumeVolatility, &relativeStrength, &item.TechnicalScore, &item.Verdict,
			&item.Reasons, &item.InvalidationRules, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.KeyLevels = json.RawMessage(defaultJSON(keyLevels, "{}"))
		item.EventReaction = json.RawMessage(defaultJSON(eventReaction, "{}"))
		item.VolumeVolatility = json.RawMessage(defaultJSON(volumeVolatility, "{}"))
		item.RelativeStrength = json.RawMessage(defaultJSON(relativeStrength, "{}"))
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroScenarios(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroScenarioDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, scenario_key, candidate_bias, primary_symbols, secondary_symbols,
			required_confirmations, expected_reactions::text, result, reason, created_at
		FROM macro_scenario_results
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroScenarioDTO{}
	for rows.Next() {
		var item macroScenarioDTO
		var expected string
		if err := rows.Scan(&item.ID, &item.ScenarioKey, &item.CandidateBias, &item.PrimarySymbols, &item.SecondarySymbols,
			&item.RequiredConfirmations, &expected, &item.Result, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.ExpectedReactions = json.RawMessage(defaultJSON(expected, "{}"))
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroPricedInScores(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroPricedInDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, symbol, verdict, score::float8, reasons, created_at
		FROM macro_priced_in_scores
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC, symbol
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroPricedInDTO{}
	for rows.Next() {
		var item macroPricedInDTO
		if err := rows.Scan(&item.ID, &item.Symbol, &item.Verdict, &item.Score, &item.Reasons, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroConfounders(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroConfounderDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, confounder_type, headline, COALESCE(source,''), severity, reason, created_at
		FROM macro_confounders
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroConfounderDTO{}
	for rows.Next() {
		var item macroConfounderDTO
		if err := rows.Scan(&item.ID, &item.ConfounderType, &item.Headline, &item.Source, &item.Severity, &item.Reason, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroEvidenceBundles(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroEvidenceBundleDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, symbol, status, verdict, summary, evidence::text, missing_evidence,
			walkaway_reasons, created_at
		FROM macro_evidence_bundles
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC, symbol
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroEvidenceBundleDTO{}
	for rows.Next() {
		var item macroEvidenceBundleDTO
		var evidence string
		if err := rows.Scan(&item.ID, &item.Symbol, &item.Status, &item.Verdict, &item.Summary, &evidence,
			&item.MissingEvidence, &item.WalkawayReasons, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Evidence = json.RawMessage(defaultJSON(evidence, "{}"))
		out = append(out, item)
	}
	return out, rows.Err()
}

func loadMacroCandidates(r *http.Request, pool *pgxpool.Pool, eventID string) ([]macroCandidateDTO, error) {
	rows, err := pool.Query(r.Context(), `
		SELECT id::text, macro_event_id::text, evidence_bundle_id::text, symbol, side, bias,
			entry_type, entry_reference_price::float8, stop_reference_price::float8,
			target_reference_price::float8, risk_percent::float8, time_limit, status,
			created_reason, COALESCE(rejection_reason,''), walkaway_reasons, created_at
		FROM macro_candidate_trades
		WHERE macro_event_id = $1::uuid
		ORDER BY created_at DESC, symbol
	`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []macroCandidateDTO{}
	for rows.Next() {
		item, err := scanMacroCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func defaultJSON(raw string, fallback string) string {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return raw
}
