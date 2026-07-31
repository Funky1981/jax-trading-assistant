package evidencequality

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type rowQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) LoadSnapshot(ctx context.Context) (Snapshot, error) {
	if s.pool == nil {
		return Snapshot{}, fmt.Errorf("evidence quality store requires a database")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin read-only evaluation snapshot: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	before, err := loadSafetyCounts(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	events, err := loadEvents(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	candles, err := loadCandles(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	outcomes, err := loadExistingOutcomes(ctx, tx)
	if err != nil {
		return Snapshot{}, err
	}
	if err := tx.Rollback(ctx); err != nil {
		return Snapshot{}, fmt.Errorf("close read-only evaluation snapshot: %w", err)
	}
	closed = true
	after, err := loadSafetyCounts(ctx, s.pool)
	if err != nil {
		return Snapshot{}, err
	}
	if before != after {
		return Snapshot{}, fmt.Errorf("prohibited record counts changed inside read-only evaluation snapshot")
	}
	return Snapshot{Events: events, Candles: candles, SafetyBefore: before, SafetyAfter: after, ExistingOutcomes: outcomes}, nil
}

func loadEvents(ctx context.Context, tx pgx.Tx) ([]Event, error) {
	rows, err := tx.Query(ctx, `
		SELECT d.id::text,d.source_inbox_event_id::text,COALESCE(d.normalized_event_id::text,''),
			d.source_event_identity,d.decision,d.ruleset_version,d.decision_at,d.event_publication_at,
			d.event_collection_at,d.event_receipt_at,d.source,d.source_url,d.event_type,d.severity,
			d.confidence::float8,d.affected_assets,d.unknown_assets,d.reasons,d.missing_evidence,
			w.headline,COALESCE(w.summary,''),COALESCE(en.primary_symbol,''),
			COALESCE(er.payload->'raw_payload'->>'source_name',''),
			COALESCE(er.payload->'raw_payload'->>'feed_url',''),
			COALESCE(er.payload->'raw_payload'->>'source_native_id',''),
			COALESCE(er.payload->'raw_payload'->>'content_hash',er.content_hash,''),
			COALESCE(er.data_source_type,''),COALESCE(er.source_provider,''),
			COALESCE(er.is_synthetic,false),COALESCE(er.synthetic_reason,''),
			COALESCE(subject.subject_id,''),COALESCE(subject.subject_type,''),
			COALESCE(subject.current_decision,''),COALESCE(subject.event_count,0),
			COALESCE(subject.source_group_count,0),COALESCE(subject.independent_count,0),
			COALESCE(subject.primary_count,0),COALESCE(subject.repeated_count,0)
		FROM genuine_event_decisions d
		JOIN world_monitor_research_inbox w ON w.id=d.source_inbox_event_id
		LEFT JOIN event_normalized en ON en.id=d.normalized_event_id
		LEFT JOIN LATERAL (
			SELECT candidate.* FROM event_raw candidate
			WHERE candidate.id=en.raw_event_id OR (candidate.source_event_id=w.source_event_id AND candidate.source_id=w.source)
			ORDER BY (candidate.id=en.raw_event_id) DESC,candidate.received_at DESC LIMIT 1
		) er ON true
		LEFT JOIN LATERAL (
			SELECT s.public_id AS subject_id,s.subject_type,s.current_decision,
				COUNT(*)::int AS event_count,COUNT(DISTINCT l2.source_group_key)::int AS source_group_count,
				COUNT(DISTINCT l2.source_group_key) FILTER (WHERE l2.source_independence IN ('primary','independent'))::int AS independent_count,
				COUNT(DISTINCT l2.source_group_key) FILTER (WHERE l2.source_independence='primary')::int AS primary_count,
				COUNT(*) FILTER (WHERE l2.source_independence='not_independent')::int AS repeated_count
			FROM evidence_subject_events own
			JOIN evidence_subjects s ON s.id=own.subject_id
			JOIN evidence_subject_events l2 ON l2.subject_id=s.id
			WHERE own.genuine_event_id=d.source_inbox_event_id
			GROUP BY s.public_id,s.subject_type,s.current_decision
		) subject ON true
		WHERE d.is_current
		ORDER BY d.event_receipt_at,d.source_event_identity`)
	if err != nil {
		return nil, fmt.Errorf("load current genuine decisions: %w", err)
	}
	defer rows.Close()
	result := []Event{}
	for rows.Next() {
		var event Event
		var collection sql.NullTime
		if err := rows.Scan(
			&event.DecisionID, &event.InboxID, &event.NormalizedEventID, &event.SourceEventIdentity,
			&event.Decision, &event.RulesetVersion, &event.DecisionAt, &event.PublicationAt,
			&collection, &event.ReceiptAt, &event.Source, &event.SourceURL, &event.EventType,
			&event.Severity, &event.Confidence, &event.AffectedAssets, &event.UnknownAssets,
			&event.Reasons, &event.MissingEvidence, &event.Headline, &event.Summary,
			&event.PrimarySymbol, &event.SourceName, &event.FeedURL, &event.SourceNativeID,
			&event.ContentHash, &event.DataSourceType, &event.SourceProvider, &event.IsSynthetic,
			&event.SyntheticReason, &event.SubjectID, &event.SubjectType, &event.SubjectCurrentDecision,
			&event.SubjectEventCount, &event.SourceGroupCount, &event.IndependentSourceCount,
			&event.PrimarySourceCount, &event.RepeatedSourceCount,
		); err != nil {
			return nil, fmt.Errorf("scan genuine decision for evaluation: %w", err)
		}
		if collection.Valid {
			value := collection.Time
			event.CollectionAt = &value
		}
		result = append(result, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("current genuine decision rows: %w", err)
	}
	return result, nil
}

func loadCandles(ctx context.Context, tx pgx.Tx) ([]Candle, error) {
	rows, err := tx.Query(ctx, `
		SELECT symbol,timestamp,open::float8,high::float8,low::float8,close::float8,
			timeframe,source,timestamp_semantics,regular_trading_hours,market_data_classification
		FROM candles
		WHERE timeframe IN ('1h','1d')
		  AND source NOT IN ('unknown','TEST','SYNTHETIC','FIXTURE')
		  AND timestamp_semantics='interval_start'
		  AND UPPER(market_data_classification) NOT IN ('TEST','SYNTHETIC','FIXTURE')
		ORDER BY symbol,timeframe,source,timestamp`)
	if err != nil {
		return nil, fmt.Errorf("load persisted genuine candles: %w", err)
	}
	defer rows.Close()
	result := []Candle{}
	for rows.Next() {
		var candle Candle
		var rth sql.NullBool
		if err := rows.Scan(&candle.Symbol, &candle.Timestamp, &candle.Open, &candle.High, &candle.Low,
			&candle.Close, &candle.Timeframe, &candle.Source, &candle.TimestampSemantics,
			&rth, &candle.MarketDataClassification); err != nil {
			return nil, fmt.Errorf("scan persisted candle: %w", err)
		}
		if rth.Valid {
			value := rth.Bool
			candle.RegularTradingHours = &value
		}
		result = append(result, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("persisted candle rows: %w", err)
	}
	return result, nil
}

func loadSafetyCounts(ctx context.Context, db rowQueryer) (SafetyCounts, error) {
	var counts SafetyCounts
	err := db.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM trade_approvals),
		(SELECT COUNT(*) FROM candidate_approvals),
		(SELECT COUNT(*) FROM candidate_paper_tickets),
		(SELECT COUNT(*) FROM execution_instructions),
		(SELECT COUNT(*) FROM order_intents),
		(SELECT COUNT(*) FROM execution_instructions WHERE NULLIF(BTRIM(broker_order_id),'') IS NOT NULL),
		(SELECT COUNT(*) FROM trades),
		(SELECT COUNT(*) FROM fills)`).Scan(
		&counts.Approvals, &counts.CandidateApprovals, &counts.PaperTickets,
		&counts.ExecutionInstructions, &counts.OrderIntents, &counts.BrokerOrders,
		&counts.Trades, &counts.Fills,
	)
	if err != nil {
		return SafetyCounts{}, fmt.Errorf("read prohibited record counts: %w", err)
	}
	return counts, nil
}

func loadExistingOutcomes(ctx context.Context, tx pgx.Tx) (ExistingOutcomeSummary, error) {
	var result ExistingOutcomeSummary
	var sources, horizons string
	err := tx.QueryRow(ctx, `SELECT COUNT(*),
		COALESCE(string_agg(DISTINCT market_data_source,',' ORDER BY market_data_source),''),
		COALESCE(string_agg(DISTINCT checkpoint_name,',' ORDER BY checkpoint_name),'')
		FROM paper_ticket_outcome_checkpoints`).Scan(&result.RecordCount, &sources, &horizons)
	if err != nil {
		return ExistingOutcomeSummary{}, fmt.Errorf("read existing outcome records: %w", err)
	}
	if sources != "" {
		result.Sources = strings.Split(sources, ",")
	}
	if horizons != "" {
		result.Horizons = strings.Split(horizons, ",")
	}
	return result, nil
}
