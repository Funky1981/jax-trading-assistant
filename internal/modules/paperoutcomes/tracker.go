package paperoutcomes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTicketNotFound       = errors.New("paper ticket not found")
	ErrExecutionLinkedState = errors.New("paper outcome safety verification failed: execution-linked state exists")
)

type checkpointDefinition struct {
	Name  string
	After time.Duration
}

var checkpointDefinitions = []checkpointDefinition{{"1h", time.Hour}, {"1d", 24 * time.Hour}, {"1w", 7 * 24 * time.Hour}}

type ticket struct {
	ID, Symbol, Direction, Status                  string
	CreatedAt                                      time.Time
	ApprovalAt, CandidateApprovalAt                *time.Time
	Entry, Stop, Target, Size                      float64
	PaperOnly                                      bool
	BrokerAllowed, InstructionCreated, LiveAllowed bool
}

type observation struct {
	At               time.Time
	High, Low, Close float64
}

type Checkpoint struct {
	PaperTicketID              string     `json:"paperTicketId"`
	CheckpointName             string     `json:"checkpointName"`
	TrackingStartedAt          time.Time  `json:"trackingStartedAt"`
	TrackingStartSource        string     `json:"trackingStartSource"`
	ScheduledAt                time.Time  `json:"scheduledAt"`
	ObservationAt              *time.Time `json:"observationAt,omitempty"`
	ObservationDelaySeconds    *int64     `json:"observationDelaySeconds,omitempty"`
	HypotheticalEntryPrice     float64    `json:"hypotheticalEntryPrice"`
	CheckpointPrice            *float64   `json:"checkpointPrice,omitempty"`
	PriceChange                *float64   `json:"priceChange,omitempty"`
	PercentageReturn           *float64   `json:"percentageReturn,omitempty"`
	HypotheticalPnL            *float64   `json:"hypotheticalPnl,omitempty"`
	HighestObservedPrice       *float64   `json:"highestObservedPrice,omitempty"`
	LowestObservedPrice        *float64   `json:"lowestObservedPrice,omitempty"`
	MaximumFavourableExcursion *float64   `json:"maximumFavourableExcursion,omitempty"`
	MaximumAdverseExcursion    *float64   `json:"maximumAdverseExcursion,omitempty"`
	TargetTouched              bool       `json:"targetTouched"`
	StopTouched                bool       `json:"stopTouched"`
	FirstTargetTouchAt         *time.Time `json:"firstTargetTouchAt,omitempty"`
	FirstStopTouchAt           *time.Time `json:"firstStopTouchAt,omitempty"`
	CheckpointStatus           string     `json:"checkpointStatus"`
	DataQualityStatus          string     `json:"dataQualityStatus"`
	MarketDataSource           string     `json:"marketDataSource,omitempty"`
	MarketDataClassification   string     `json:"marketDataClassification,omitempty"`
	CandleInterval             string     `json:"candleInterval,omitempty"`
	ObservationCount           int        `json:"observationCount"`
	EarliestObservationAt      *time.Time `json:"earliestObservationAt,omitempty"`
	LatestObservationAt        *time.Time `json:"latestObservationAt,omitempty"`
	stopPrice                  float64
	targetPrice                float64
	positionSize               float64
	direction                  string
	usedObservations           []observation
}

type Result struct {
	PaperTicketID         string       `json:"paperTicketId"`
	Nature                string       `json:"nature"`
	EntryAssumption       string       `json:"entryAssumption"`
	Checkpoints           []Checkpoint `json:"checkpoints"`
	ExecutionInstructions int          `json:"executionInstructions"`
	OrderIntents          int          `json:"orderIntents"`
	BrokerOrders          int          `json:"brokerOrders"`
	Trades                int          `json:"trades"`
}

type Tracker struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func New(pool *pgxpool.Pool) *Tracker {
	return &Tracker{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (t *Tracker) Track(ctx context.Context, paperTicketID string) (Result, error) {
	ticket, err := t.loadTicket(ctx, paperTicketID)
	if err != nil {
		return Result{}, err
	}
	start, source := trackingStart(ticket)
	now := t.now()
	observations, dataSource, classification, interval, err := t.loadCandles(ctx, ticket.Symbol, start, now)
	if err != nil {
		return Result{}, err
	}
	if len(observations) == 0 {
		if quote, quoteErr := t.loadQuote(ctx, ticket.Symbol, start, now); quoteErr != nil {
			return Result{}, quoteErr
		} else if quote != nil {
			observations = []observation{*quote}
			dataSource = "persisted_quote"
			interval = "point_in_time"
		} else {
			dataSource = "persisted_candles_and_quotes_checked"
			interval = "unavailable"
		}
	}

	result := Result{PaperTicketID: paperTicketID, Nature: "hypothetical_research_outcome", EntryAssumption: "ticket entry price; no broker fill occurred"}
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("begin paper outcome transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, def := range checkpointDefinitions {
		cp := calculateCheckpoint(ticket, def, start, source, now, observations, dataSource, classification, interval)
		if err := t.persist(ctx, tx, cp); err != nil {
			return Result{}, err
		}
		result.Checkpoints = append(result.Checkpoints, cp)
	}
	if err := tx.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=p.candidate_id),
		(SELECT COUNT(*) FROM order_intents WHERE signal_id=c.signal_id),
		(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=p.candidate_id AND broker_order_id IS NOT NULL),
		(SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=p.candidate_id AND trade_id IS NOT NULL)
		FROM candidate_paper_tickets p JOIN candidate_trades c ON c.id=p.candidate_id WHERE p.paper_ticket_id=$1`, paperTicketID).Scan(
		&result.ExecutionInstructions, &result.OrderIntents, &result.BrokerOrders, &result.Trades); err != nil {
		return Result{}, fmt.Errorf("verify execution boundary: %w", err)
	}
	if result.ExecutionInstructions != 0 || result.OrderIntents != 0 || result.BrokerOrders != 0 || result.Trades != 0 {
		return Result{}, ErrExecutionLinkedState
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, fmt.Errorf("commit paper outcomes: %w", err)
	}
	return result, nil
}

func trackingStart(v ticket) (time.Time, string) {
	if !v.CreatedAt.IsZero() {
		return v.CreatedAt.UTC(), "paper_ticket_created_at"
	}
	if v.ApprovalAt != nil {
		return v.ApprovalAt.UTC(), "human_approval_decided_at"
	}
	return v.CandidateApprovalAt.UTC(), "candidate_approved_at"
}

func calculateCheckpoint(v ticket, def checkpointDefinition, start time.Time, source string, now time.Time, all []observation, dataSource, classification, interval string) Checkpoint {
	cp := Checkpoint{PaperTicketID: v.ID, CheckpointName: def.Name, TrackingStartedAt: start, TrackingStartSource: source,
		ScheduledAt: start.Add(def.After), HypotheticalEntryPrice: v.Entry, CheckpointStatus: "pending_not_due", DataQualityStatus: "not_due",
		MarketDataSource: dataSource, MarketDataClassification: classification, CandleInterval: interval,
		stopPrice: v.Stop, targetPrice: v.Target, positionSize: v.Size, direction: v.Direction}
	if !v.PaperOnly || v.BrokerAllowed || v.InstructionCreated || v.LiveAllowed || v.Direction != "long" || v.Entry <= 0 || v.Size <= 0 {
		cp.CheckpointStatus = "invalid_ticket"
		cp.DataQualityStatus = "invalid_ticket"
		return cp
	}
	if v.Status == "paper_ticket_cancelled" {
		cp.CheckpointStatus = "cancelled"
		cp.DataQualityStatus = "cancelled"
		return cp
	}
	if now.Before(cp.ScheduledAt) {
		return cp
	}
	var window []observation
	var chosen *observation
	for i := range all {
		o := all[i]
		if o.At.Before(start) {
			continue
		}
		if chosen == nil && !o.At.Before(cp.ScheduledAt) {
			copy := o
			chosen = &copy
		}
		if chosen == nil || !o.At.After(chosen.At) {
			window = append(window, o)
		}
	}
	if chosen == nil {
		cp.CheckpointStatus = "pending_market_data"
		cp.DataQualityStatus = "missing_observation_at_or_after_checkpoint"
		return cp
	}
	// Trim observations after the selected checkpoint observation.
	window = window[:0]
	for _, o := range all {
		if !o.At.Before(start) && !o.At.After(chosen.At) {
			window = append(window, o)
		}
	}
	cp.MarketDataSource = dataSource
	cp.MarketDataClassification = classification
	cp.CandleInterval = interval
	cp.ObservationAt = &chosen.At
	delay := int64(chosen.At.Sub(cp.ScheduledAt).Seconds())
	cp.ObservationDelaySeconds = &delay
	cp.CheckpointPrice = ptr(chosen.Close)
	change := chosen.Close - v.Entry
	cp.PriceChange = ptr(change)
	cp.PercentageReturn = ptr(change / v.Entry * 100)
	cp.HypotheticalPnL = ptr(change * v.Size)
	cp.ObservationCount = len(window)
	cp.usedObservations = append([]observation(nil), window...)
	earliest, latest := window[0].At, window[len(window)-1].At
	cp.EarliestObservationAt = &earliest
	cp.LatestObservationAt = &latest
	high, low := window[0].High, window[0].Low
	for _, o := range window {
		if o.High > high {
			high = o.High
		}
		if o.Low < low {
			low = o.Low
		}
		target := o.High >= v.Target
		stop := o.Low <= v.Stop
		if target && cp.FirstTargetTouchAt == nil {
			at := o.At
			cp.FirstTargetTouchAt = &at
			cp.TargetTouched = true
		}
		if stop && cp.FirstStopTouchAt == nil {
			at := o.At
			cp.FirstStopTouchAt = &at
			cp.StopTouched = true
		}
	}
	cp.HighestObservedPrice = ptr(high)
	cp.LowestObservedPrice = ptr(low)
	cp.MaximumFavourableExcursion = ptr(high - v.Entry)
	cp.MaximumAdverseExcursion = ptr(low - v.Entry)
	cp.CheckpointStatus = "completed"
	cp.DataQualityStatus = "complete_candle_window"
	if dataSource == "persisted_quote" {
		cp.DataQualityStatus = "limited_quote_only"
	}
	if cp.TargetTouched && cp.StopTouched && cp.FirstTargetTouchAt.Equal(*cp.FirstStopTouchAt) {
		cp.CheckpointStatus = "ambiguous_same_candle"
	} else if cp.TargetTouched && (!cp.StopTouched || cp.FirstTargetTouchAt.Before(*cp.FirstStopTouchAt)) {
		cp.CheckpointStatus = "target_touched"
	} else if cp.StopTouched {
		cp.CheckpointStatus = "stop_touched"
	}
	return cp
}

func ptr(v float64) *float64 { return &v }

func (t *Tracker) loadTicket(ctx context.Context, id string) (ticket, error) {
	var v ticket
	err := t.pool.QueryRow(ctx, `SELECT p.paper_ticket_id,p.symbol,p.direction,p.status,p.created_at,a.decided_at,
		CASE WHEN c.status='approved' THEN c.updated_at END,p.entry_price::float8,p.stop_loss_price::float8,p.target_price::float8,p.position_size::float8,
		p.paper_only,p.broker_execution_allowed,p.execution_instruction_created,p.live_trading_allowed
		FROM candidate_paper_tickets p LEFT JOIN candidate_approvals a ON a.id=p.source_approval_id JOIN candidate_trades c ON c.id=p.candidate_id WHERE p.paper_ticket_id=$1`, id).Scan(
		&v.ID, &v.Symbol, &v.Direction, &v.Status, &v.CreatedAt, &v.ApprovalAt, &v.CandidateApprovalAt, &v.Entry, &v.Stop, &v.Target, &v.Size, &v.PaperOnly, &v.BrokerAllowed, &v.InstructionCreated, &v.LiveAllowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrTicketNotFound
	}
	if err != nil {
		return v, fmt.Errorf("load paper ticket: %w", err)
	}
	return v, nil
}

func (t *Tracker) loadCandles(ctx context.Context, symbol string, start, end time.Time) ([]observation, string, string, string, error) {
	rows, err := t.pool.Query(ctx, `WITH selected AS (
		SELECT source,timeframe FROM candles
		WHERE symbol=$1 AND timestamp >= $2 AND timestamp <= $3
		  AND source <> 'unknown' AND UPPER(source) NOT IN ('TEST','SYNTHETIC','FIXTURE')
		ORDER BY timestamp DESC LIMIT 1
	) SELECT c.timestamp,c.high::float8,c.low::float8,c.close::float8,c.source,c.timeframe,c.market_data_classification
		FROM candles c JOIN selected s ON s.source=c.source AND s.timeframe=c.timeframe
		WHERE c.symbol=$1 AND c.timestamp >= $2 AND c.timestamp <= $3 ORDER BY c.timestamp`, symbol, start, end)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("load persisted candles: %w", err)
	}
	defer rows.Close()
	var out []observation
	var source, timeframe, classification string
	for rows.Next() {
		var o observation
		if err := rows.Scan(&o.At, &o.High, &o.Low, &o.Close, &source, &timeframe, &classification); err != nil {
			return nil, "", "", "", err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, "", "", "", err
	}
	if len(out) == 0 {
		return out, "persisted_candles_and_quotes_checked", "unknown", "unavailable", nil
	}
	return out, "persisted_candles:" + source, classification, timeframe, nil
}
func (t *Tracker) loadQuote(ctx context.Context, symbol string, start, end time.Time) (*observation, error) {
	var o observation
	err := t.pool.QueryRow(ctx, `SELECT timestamp,price,price,price FROM quotes
		WHERE symbol=$1 AND timestamp >= $2 AND timestamp <= $3
		  AND UPPER(COALESCE(exchange, '')) NOT IN ('TEST', 'SYNTHETIC', 'FIXTURE')`, symbol, start, end).Scan(&o.At, &o.High, &o.Low, &o.Close)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load persisted quote: %w", err)
	}
	return &o, nil
}

func (t *Tracker) persist(ctx context.Context, tx pgx.Tx, cp Checkpoint) error {
	auditObservations := make([]map[string]any, 0, len(cp.usedObservations))
	for _, o := range cp.usedObservations {
		auditObservations = append(auditObservations, map[string]any{"timestamp": o.At, "high": o.High, "low": o.Low, "close": o.Close})
	}
	inputs, _ := json.Marshal(map[string]any{
		"nature": "hypothetical_research_outcome", "no_actual_fill": true,
		"direction": cp.direction, "entry_price_assumption": cp.HypotheticalEntryPrice,
		"stop_price": cp.stopPrice, "target_price": cp.targetPrice, "position_size": cp.positionSize,
		"observations": auditObservations,
	})
	_, err := tx.Exec(ctx, `INSERT INTO paper_ticket_outcome_checkpoints (paper_ticket_id,checkpoint_name,tracking_started_at,tracking_start_source,scheduled_at,observation_at,observation_delay_seconds,hypothetical_entry_price,checkpoint_price,price_change,percentage_return,hypothetical_pnl,highest_observed_price,lowest_observed_price,maximum_favourable_excursion,maximum_adverse_excursion,target_touched,stop_touched,first_target_touch_at,first_stop_touch_at,checkpoint_status,data_quality_status,market_data_source,market_data_classification,candle_interval,observation_count,earliest_observation_at,latest_observation_at,calculation_inputs)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
	ON CONFLICT(paper_ticket_id,checkpoint_name) DO UPDATE SET tracking_started_at=EXCLUDED.tracking_started_at,tracking_start_source=EXCLUDED.tracking_start_source,scheduled_at=EXCLUDED.scheduled_at,observation_at=EXCLUDED.observation_at,observation_delay_seconds=EXCLUDED.observation_delay_seconds,checkpoint_price=EXCLUDED.checkpoint_price,price_change=EXCLUDED.price_change,percentage_return=EXCLUDED.percentage_return,hypothetical_pnl=EXCLUDED.hypothetical_pnl,highest_observed_price=EXCLUDED.highest_observed_price,lowest_observed_price=EXCLUDED.lowest_observed_price,maximum_favourable_excursion=EXCLUDED.maximum_favourable_excursion,maximum_adverse_excursion=EXCLUDED.maximum_adverse_excursion,target_touched=EXCLUDED.target_touched,stop_touched=EXCLUDED.stop_touched,first_target_touch_at=EXCLUDED.first_target_touch_at,first_stop_touch_at=EXCLUDED.first_stop_touch_at,checkpoint_status=EXCLUDED.checkpoint_status,data_quality_status=EXCLUDED.data_quality_status,market_data_source=EXCLUDED.market_data_source,market_data_classification=EXCLUDED.market_data_classification,candle_interval=EXCLUDED.candle_interval,observation_count=EXCLUDED.observation_count,earliest_observation_at=EXCLUDED.earliest_observation_at,latest_observation_at=EXCLUDED.latest_observation_at,calculation_inputs=EXCLUDED.calculation_inputs,updated_at=NOW()
	WHERE paper_ticket_outcome_checkpoints.checkpoint_status IN ('pending_not_due','pending_market_data','insufficient_data')`, cp.PaperTicketID, cp.CheckpointName, cp.TrackingStartedAt, cp.TrackingStartSource, cp.ScheduledAt, cp.ObservationAt, cp.ObservationDelaySeconds, cp.HypotheticalEntryPrice, cp.CheckpointPrice, cp.PriceChange, cp.PercentageReturn, cp.HypotheticalPnL, cp.HighestObservedPrice, cp.LowestObservedPrice, cp.MaximumFavourableExcursion, cp.MaximumAdverseExcursion, cp.TargetTouched, cp.StopTouched, cp.FirstTargetTouchAt, cp.FirstStopTouchAt, cp.CheckpointStatus, cp.DataQualityStatus, nullString(cp.MarketDataSource), nullString(cp.MarketDataClassification), nullString(cp.CandleInterval), cp.ObservationCount, cp.EarliestObservationAt, cp.LatestObservationAt, inputs)
	if err != nil {
		return fmt.Errorf("persist paper checkpoint %s: %w", cp.CheckpointName, err)
	}
	return nil
}
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
