package macroevents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) FindBySourceEventID(ctx context.Context, source, sourceEventID string) (StoredEvent, bool, error) {
	var event StoredEvent
	if s == nil || s.pool == nil {
		return StoredEvent{}, false, nil
	}
	var rejectionReason *string
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, status, rejection_reason
		FROM macro_events
		WHERE source = $1 AND source_event_id = $2
	`, source, sourceEventID).Scan(&event.ID, &event.Status, &rejectionReason)
	if err != nil {
		if err == pgx.ErrNoRows {
			return StoredEvent{}, false, nil
		}
		return StoredEvent{}, false, fmt.Errorf("lookup macro event duplicate: %w", err)
	}
	if rejectionReason != nil {
		event.RejectionReason = *rejectionReason
	}
	mappings, err := s.loadMappings(ctx, event.ID)
	if err != nil {
		return StoredEvent{}, false, err
	}
	event.Mappings = mappings
	return event, true, nil
}

func (s *Store) Save(ctx context.Context, event StoredEvent) (StoredEvent, error) {
	if s == nil || s.pool == nil {
		if event.ID == "" {
			event.ID = uuid.NewString()
		}
		return event, nil
	}

	rawPayload := event.Input.RawPayload
	if rawPayload == nil {
		rawPayload = map[string]any{}
	}
	rawPayloadJSON, err := json.Marshal(rawPayload)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("marshal macro event raw payload: %w", err)
	}
	surpriseValue, surprisePercent := ComputeSurprise(event.Input.ActualValue, event.Input.ExpectedValue)
	id := uuid.NewString()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("begin macro event insert: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO macro_events (
			id, source, source_event_id, event_type, region, event_time_utc,
			headline, summary, actual_value, expected_value, previous_value, unit,
			surprise_value, surprise_percent, direction, confidence, raw_payload,
			status, rejection_reason
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6,
			$7, NULLIF($8, ''), $9, $10, $11, NULLIF($12, ''),
			$13, $14, $15, $16, $17::jsonb,
			$18, NULLIF($19, '')
		)
		ON CONFLICT (source, source_event_id) DO UPDATE SET
			updated_at = NOW()
		RETURNING id::text, status, COALESCE(rejection_reason, '')
	`, id, event.Input.Source, event.Input.SourceEventID, event.Input.EventType, event.Input.Region, event.Input.EventTimeUTC.UTC(),
		event.Input.Headline, event.Input.Summary, event.Input.ActualValue, event.Input.ExpectedValue, event.Input.PreviousValue, event.Input.Unit,
		surpriseValue, surprisePercent, event.Input.Direction, event.Input.Confidence, string(rawPayloadJSON), event.Status, event.RejectionReason,
	).Scan(&event.ID, &event.Status, &event.RejectionReason)
	if err != nil {
		return StoredEvent{}, fmt.Errorf("insert macro event: %w", err)
	}

	for _, mapping := range event.Mappings {
		if _, err := tx.Exec(ctx, `
			INSERT INTO macro_event_etf_map (
				macro_event_id, symbol, theme, mapping_reason, confidence
			)
			VALUES ($1::uuid, $2, $3, $4, $5)
			ON CONFLICT (macro_event_id, symbol) DO UPDATE SET
				theme = EXCLUDED.theme,
				mapping_reason = EXCLUDED.mapping_reason,
				confidence = EXCLUDED.confidence
		`, event.ID, mapping.Symbol, mapping.Theme, mapping.MappingReason, mapping.Confidence); err != nil {
			return StoredEvent{}, fmt.Errorf("insert macro ETF mapping: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return StoredEvent{}, fmt.Errorf("commit macro event insert: %w", err)
	}
	return event, nil
}

func (s *Store) loadMappings(ctx context.Context, macroEventID string) ([]ETFMapping, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT symbol, theme, mapping_reason, confidence::float8
		FROM macro_event_etf_map
		WHERE macro_event_id = $1::uuid
		ORDER BY symbol
	`, macroEventID)
	if err != nil {
		return nil, fmt.Errorf("load macro ETF mappings: %w", err)
	}
	defer rows.Close()

	mappings := []ETFMapping{}
	for rows.Next() {
		var mapping ETFMapping
		if err := rows.Scan(&mapping.Symbol, &mapping.Theme, &mapping.MappingReason, &mapping.Confidence); err != nil {
			return nil, fmt.Errorf("scan macro ETF mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate macro ETF mappings: %w", err)
	}
	return mappings, nil
}

func (s *Store) SaveReactionSnapshot(ctx context.Context, snapshot ReactionSnapshot) (ReactionSnapshot, error) {
	if s == nil || s.pool == nil {
		if snapshot.ID == "" {
			snapshot.ID = uuid.NewString()
		}
		return snapshot, nil
	}
	rawCandlesJSON, err := json.Marshal(snapshot.RawCandles)
	if err != nil {
		return ReactionSnapshot{}, fmt.Errorf("marshal macro reaction raw candles: %w", err)
	}
	id := uuid.NewString()
	err = s.pool.QueryRow(ctx, `
		INSERT INTO macro_reaction_snapshots (
			id, macro_event_id, symbol, timeframe, pre_price, post_price,
			change_abs, change_percent, high_after, low_after, volume_ratio,
			atr_ratio, direction, confirms_event, too_extended, noisy, reason,
			raw_candles
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18::jsonb
		)
		ON CONFLICT (macro_event_id, symbol, timeframe) DO UPDATE SET
			pre_price = EXCLUDED.pre_price,
			post_price = EXCLUDED.post_price,
			change_abs = EXCLUDED.change_abs,
			change_percent = EXCLUDED.change_percent,
			high_after = EXCLUDED.high_after,
			low_after = EXCLUDED.low_after,
			volume_ratio = EXCLUDED.volume_ratio,
			atr_ratio = EXCLUDED.atr_ratio,
			direction = EXCLUDED.direction,
			confirms_event = EXCLUDED.confirms_event,
			too_extended = EXCLUDED.too_extended,
			noisy = EXCLUDED.noisy,
			reason = EXCLUDED.reason,
			raw_candles = EXCLUDED.raw_candles
		RETURNING id::text
	`, id, snapshot.MacroEventID, snapshot.Symbol, snapshot.Timeframe, snapshot.PrePrice, snapshot.PostPrice,
		snapshot.ChangeAbs, snapshot.ChangePercent, snapshot.HighAfter, snapshot.LowAfter, snapshot.VolumeRatio,
		snapshot.ATRRatio, snapshot.Direction, snapshot.ConfirmsEvent, snapshot.TooExtended, snapshot.Noisy, snapshot.Reason,
		string(rawCandlesJSON)).Scan(&snapshot.ID)
	if err != nil {
		return ReactionSnapshot{}, fmt.Errorf("insert macro reaction snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) SaveTechnicalAnalysisSnapshot(ctx context.Context, snapshot TechnicalSnapshot) (TechnicalSnapshot, error) {
	if s == nil || s.pool == nil {
		if snapshot.ID == "" {
			snapshot.ID = uuid.NewString()
		}
		return snapshot, nil
	}
	keyLevelsJSON, err := json.Marshal(snapshot.KeyLevels)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("marshal technical key levels: %w", err)
	}
	eventReactionJSON, err := json.Marshal(snapshot.EventReaction)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("marshal technical event reaction: %w", err)
	}
	volumeVolatilityJSON, err := json.Marshal(snapshot.VolumeVolatility)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("marshal technical volume volatility: %w", err)
	}
	relativeStrengthJSON, err := json.Marshal(snapshot.RelativeStrength)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("marshal technical relative strength: %w", err)
	}
	id := uuid.NewString()
	var macroEventID any
	if snapshot.MacroEventID != "" {
		macroEventID = snapshot.MacroEventID
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO technical_analysis_snapshots (
			id, macro_event_id, symbol, analysis_time_utc, timeframe,
			trend_state, structure_state, key_levels, event_reaction,
			volume_volatility, relative_strength, technical_score,
			verdict, reasons, invalidation_rules
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5,
			$6, $7, $8::jsonb, $9::jsonb,
			$10::jsonb, $11::jsonb, $12,
			$13, $14, $15
		)
		ON CONFLICT (macro_event_id, symbol, timeframe) DO UPDATE SET
			analysis_time_utc = EXCLUDED.analysis_time_utc,
			trend_state = EXCLUDED.trend_state,
			structure_state = EXCLUDED.structure_state,
			key_levels = EXCLUDED.key_levels,
			event_reaction = EXCLUDED.event_reaction,
			volume_volatility = EXCLUDED.volume_volatility,
			relative_strength = EXCLUDED.relative_strength,
			technical_score = EXCLUDED.technical_score,
			verdict = EXCLUDED.verdict,
			reasons = EXCLUDED.reasons,
			invalidation_rules = EXCLUDED.invalidation_rules
		RETURNING id::text
	`, id, macroEventID, snapshot.Symbol, snapshot.AnalysisTimeUTC, snapshot.Timeframe,
		snapshot.TrendState, snapshot.StructureState, string(keyLevelsJSON), string(eventReactionJSON),
		string(volumeVolatilityJSON), string(relativeStrengthJSON), snapshot.TechnicalScore,
		snapshot.Verdict, snapshot.Reasons, snapshot.InvalidationRules).Scan(&snapshot.ID)
	if err != nil {
		return TechnicalSnapshot{}, fmt.Errorf("insert technical analysis snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) SaveFundamentalAnalysisSnapshot(ctx context.Context, snapshot FundamentalSnapshot) (FundamentalSnapshot, error) {
	if s == nil || s.pool == nil {
		if snapshot.ID == "" {
			snapshot.ID = uuid.NewString()
		}
		return snapshot, nil
	}
	crossMarketJSON, err := json.Marshal(snapshot.CrossMarketChecks)
	if err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("marshal fundamental cross-market checks: %w", err)
	}
	confoundersJSON, err := json.Marshal(snapshot.Confounders)
	if err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("marshal fundamental confounders: %w", err)
	}
	id := uuid.NewString()
	var macroEventID any
	if snapshot.MacroEventID != "" {
		macroEventID = snapshot.MacroEventID
	}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO fundamental_analysis_snapshots (
			id, macro_event_id, symbol, analysis_time_utc, event_summary,
			expected_market_impact, affected_themes, cross_market_checks,
			confounders, fundamental_score, verdict, reasons, missing_evidence
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5,
			$6, $7, $8::jsonb,
			$9::jsonb, $10, $11, $12, $13
		)
		ON CONFLICT (macro_event_id, symbol) DO UPDATE SET
			analysis_time_utc = EXCLUDED.analysis_time_utc,
			event_summary = EXCLUDED.event_summary,
			expected_market_impact = EXCLUDED.expected_market_impact,
			affected_themes = EXCLUDED.affected_themes,
			cross_market_checks = EXCLUDED.cross_market_checks,
			confounders = EXCLUDED.confounders,
			fundamental_score = EXCLUDED.fundamental_score,
			verdict = EXCLUDED.verdict,
			reasons = EXCLUDED.reasons,
			missing_evidence = EXCLUDED.missing_evidence
		RETURNING id::text
	`, id, macroEventID, snapshot.Symbol, snapshot.AnalysisTimeUTC, snapshot.EventSummary,
		snapshot.ExpectedMarketImpact, snapshot.AffectedThemes, string(crossMarketJSON), string(confoundersJSON),
		snapshot.FundamentalScore, snapshot.Verdict, snapshot.Reasons, snapshot.MissingEvidence).Scan(&snapshot.ID)
	if err != nil {
		return FundamentalSnapshot{}, fmt.Errorf("insert fundamental analysis snapshot: %w", err)
	}
	return snapshot, nil
}

func (s *Store) SaveScenarioResult(ctx context.Context, result ScenarioEvaluation) (ScenarioEvaluation, error) {
	if s == nil || s.pool == nil {
		if result.ID == "" {
			result.ID = uuid.NewString()
		}
		return result, nil
	}
	expectedReactionsJSON, err := MarshalExpectedReactions(result.ExpectedReactions)
	if err != nil {
		return ScenarioEvaluation{}, err
	}
	id := uuid.NewString()
	err = s.pool.QueryRow(ctx, `
		INSERT INTO macro_scenario_results (
			id, macro_event_id, scenario_key, candidate_bias, primary_symbols,
			secondary_symbols, required_confirmations, expected_reactions,
			result, reason
		)
		VALUES (
			$1::uuid, $2::uuid, $3, $4, $5,
			$6, $7, $8::jsonb,
			$9, $10
		)
		ON CONFLICT (macro_event_id, scenario_key) DO UPDATE SET
			candidate_bias = EXCLUDED.candidate_bias,
			primary_symbols = EXCLUDED.primary_symbols,
			secondary_symbols = EXCLUDED.secondary_symbols,
			required_confirmations = EXCLUDED.required_confirmations,
			expected_reactions = EXCLUDED.expected_reactions,
			result = EXCLUDED.result,
			reason = EXCLUDED.reason
		RETURNING id::text
	`, id, result.MacroEventID, result.ScenarioKey, result.CandidateBias, result.PrimarySymbols,
		result.SecondarySymbols, result.RequiredConfirmations, string(expectedReactionsJSON),
		result.Result, result.Reason).Scan(&result.ID)
	if err != nil {
		return ScenarioEvaluation{}, fmt.Errorf("insert macro scenario result: %w", err)
	}
	return result, nil
}

func (s *Store) SavePricedInScore(ctx context.Context, score PricedInScore) (PricedInScore, error) {
	if s == nil || s.pool == nil {
		if score.ID == "" {
			score.ID = uuid.NewString()
		}
		return score, nil
	}
	id := uuid.NewString()
	err := s.pool.QueryRow(ctx, `
		INSERT INTO macro_priced_in_scores (
			id, macro_event_id, symbol, verdict, score, reasons
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		ON CONFLICT (macro_event_id, symbol) DO UPDATE SET
			verdict = EXCLUDED.verdict,
			score = EXCLUDED.score,
			reasons = EXCLUDED.reasons
		RETURNING id::text
	`, id, score.MacroEventID, score.Symbol, score.Verdict, score.Score, score.Reasons).Scan(&score.ID)
	if err != nil {
		return PricedInScore{}, fmt.Errorf("insert macro priced-in score: %w", err)
	}
	return score, nil
}

func (s *Store) SaveConfounders(ctx context.Context, confounders []Confounder) ([]Confounder, error) {
	if s == nil || s.pool == nil {
		out := append([]Confounder(nil), confounders...)
		for i := range out {
			if out[i].ID == "" {
				out[i].ID = uuid.NewString()
			}
		}
		return out, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin macro confounder insert: %w", err)
	}
	defer tx.Rollback(ctx)

	out := make([]Confounder, 0, len(confounders))
	for _, confounder := range confounders {
		id := uuid.NewString()
		err := tx.QueryRow(ctx, `
			INSERT INTO macro_confounders (
				id, macro_event_id, confounder_type, headline, source, severity, reason
			)
			VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5, ''), $6, $7)
			RETURNING id::text
		`, id, confounder.MacroEventID, confounder.Type, confounder.Headline,
			confounder.Source, confounder.Severity, confounder.Reason).Scan(&confounder.ID)
		if err != nil {
			return nil, fmt.Errorf("insert macro confounder: %w", err)
		}
		out = append(out, confounder)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit macro confounder insert: %w", err)
	}
	return out, nil
}

func (s *Store) SaveEvidenceBundle(ctx context.Context, bundle EvidenceBundle) (EvidenceBundle, error) {
	if s == nil || s.pool == nil {
		if bundle.ID == "" {
			bundle.ID = uuid.NewString()
		}
		return bundle, nil
	}
	evidenceJSON, err := MarshalEvidence(bundle.Evidence)
	if err != nil {
		return EvidenceBundle{}, fmt.Errorf("marshal macro evidence bundle: %w", err)
	}
	id := uuid.NewString()
	err = s.pool.QueryRow(ctx, `
		INSERT INTO macro_evidence_bundles (
			id, macro_event_id, symbol, status, verdict, summary, evidence,
			missing_evidence, walkaway_reasons
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7::jsonb, $8, $9)
		ON CONFLICT (macro_event_id, symbol) DO UPDATE SET
			status = EXCLUDED.status,
			verdict = EXCLUDED.verdict,
			summary = EXCLUDED.summary,
			evidence = EXCLUDED.evidence,
			missing_evidence = EXCLUDED.missing_evidence,
			walkaway_reasons = EXCLUDED.walkaway_reasons
		RETURNING id::text
	`, id, bundle.MacroEventID, bundle.Symbol, bundle.Status, bundle.Verdict, bundle.Summary,
		string(evidenceJSON), bundle.MissingEvidence, bundle.WalkawayReasons).Scan(&bundle.ID)
	if err != nil {
		return EvidenceBundle{}, fmt.Errorf("insert macro evidence bundle: %w", err)
	}
	return bundle, nil
}

func (s *Store) SaveMacroCandidate(ctx context.Context, candidate MacroCandidate) (MacroCandidate, error) {
	if s == nil || s.pool == nil {
		if candidate.ID == "" {
			candidate.ID = uuid.NewString()
		}
		return candidate, nil
	}
	id := uuid.NewString()
	err := s.pool.QueryRow(ctx, `
		INSERT INTO macro_candidate_trades (
			id, macro_event_id, evidence_bundle_id, symbol, side, bias, entry_type,
			entry_reference_price, stop_reference_price, target_reference_price,
			risk_percent, time_limit, status, created_reason, rejection_reason,
			walkaway_reasons
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13, $14, NULLIF($15, ''),
			$16
		)
		ON CONFLICT (macro_event_id, evidence_bundle_id, symbol) DO UPDATE SET
			side = EXCLUDED.side,
			bias = EXCLUDED.bias,
			entry_type = EXCLUDED.entry_type,
			entry_reference_price = EXCLUDED.entry_reference_price,
			stop_reference_price = EXCLUDED.stop_reference_price,
			target_reference_price = EXCLUDED.target_reference_price,
			risk_percent = EXCLUDED.risk_percent,
			time_limit = EXCLUDED.time_limit,
			status = EXCLUDED.status,
			created_reason = EXCLUDED.created_reason,
			rejection_reason = EXCLUDED.rejection_reason,
			walkaway_reasons = EXCLUDED.walkaway_reasons
		RETURNING id::text
	`, id, candidate.MacroEventID, candidate.EvidenceBundleID, candidate.Symbol, candidate.Side, candidate.Bias, candidate.EntryType,
		candidate.EntryReferencePrice, candidate.StopReferencePrice, candidate.TargetReferencePrice,
		candidate.RiskPercent, candidate.TimeLimit, candidate.Status, candidate.CreatedReason, candidate.RejectionReason,
		candidate.WalkawayReasons).Scan(&candidate.ID)
	if err != nil {
		return MacroCandidate{}, fmt.Errorf("insert macro candidate: %w", err)
	}
	return candidate, nil
}
