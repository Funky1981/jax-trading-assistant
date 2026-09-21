package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/libs/runtimepolicy"

	"github.com/jackc/pgx/v5/pgxpool"
)

// startExploratoryPaperReviewWorker runs the real PAPER-01 runtime seam. Entry
// requests are loaded from the idempotent approved-entry queue, while due
// reviews obtain canonical evidence and persisted market observations. Missing
// inputs remain explicit; this worker never fabricates a review or approval.
func startExploratoryPaperReviewWorker(ctx context.Context, pool *pgxpool.Pool) {
	runtime, err := newExploratoryPaperRuntime(pool)
	if err != nil {
		log.Printf("exploratory_paper_runtime_initialization_failed error=%q", err)
		return
	}
	health := exploratoryWorkerHealth{}
	run := func() {
		now := time.Now().UTC()
		if err := runtime.RunEntryCycle(ctx); err != nil {
			health.entryFailure(now, err)
			log.Printf("exploratory_paper_cycle_failed operation=entry timestamp=%s category=entry_cycle retryable=true consecutive_failures=%d error=%q", now.Format(time.RFC3339Nano), health.entryConsecutiveFailures, err)
		} else {
			health.entrySuccess(now)
			log.Printf("exploratory_paper_cycle_succeeded operation=entry timestamp=%s last_success=%s consecutive_failures=0", now.Format(time.RFC3339Nano), health.lastEntrySuccess.Format(time.RFC3339Nano))
		}
		if err := runtime.RunDueReviews(ctx); err != nil {
			health.reviewFailure(now, err)
			log.Printf("exploratory_paper_cycle_failed operation=review timestamp=%s category=review_cycle retryable=true consecutive_failures=%d error=%q", now.Format(time.RFC3339Nano), health.reviewConsecutiveFailures, err)
		} else {
			health.reviewSuccess(now)
			log.Printf("exploratory_paper_cycle_succeeded operation=review timestamp=%s last_success=%s consecutive_failures=0", now.Format(time.RFC3339Nano), health.lastReviewSuccess.Format(time.RFC3339Nano))
		}
	}
	run()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

func newExploratoryPaperRuntime(pool *pgxpool.Pool) (*exploratorypaper.Runtime, error) {
	store := exploratorypaper.NewPostgresStore(pool)
	venue, err := papertrading.NewPaperVenue(papertrading.DefaultPaperCapabilityContract(), papertrading.DefaultCostModel())
	if err != nil {
		return nil, err
	}
	return &exploratorypaper.Runtime{
		Mode:         runtimepolicy.CurrentMode().String(),
		Store:        store,
		Entries:      store,
		Reviews:      &postgresExploratoryReviewSource{pool: pool},
		ExitApprover: store,
		Venue:        venue,
		CostModel:    papertrading.DefaultCostModel(),
		Now:          func() time.Time { return time.Now().UTC() },
	}, nil
}

type exploratoryWorkerHealth struct {
	lastEntrySuccess          time.Time
	lastReviewSuccess         time.Time
	entryConsecutiveFailures  int
	reviewConsecutiveFailures int
}

func (h *exploratoryWorkerHealth) entrySuccess(at time.Time) {
	h.lastEntrySuccess = at
	h.entryConsecutiveFailures = 0
}

func (h *exploratoryWorkerHealth) reviewSuccess(at time.Time) {
	h.lastReviewSuccess = at
	h.reviewConsecutiveFailures = 0
}

func (h *exploratoryWorkerHealth) entryFailure(_ time.Time, _ error) {
	h.entryConsecutiveFailures++
}

func (h *exploratoryWorkerHealth) reviewFailure(_ time.Time, _ error) {
	h.reviewConsecutiveFailures++
}

type postgresExploratoryReviewSource struct{ pool *pgxpool.Pool }

func (s *postgresExploratoryReviewSource) LoadReviewObservation(ctx context.Context, record exploratorypaper.LifecycleRecord, review exploratorypaper.Review) (exploratorypaper.ReviewObservation, error) {
	calendar, err := configuredExploratoryCalendar()
	if err != nil {
		return exploratorypaper.ReviewObservation{}, err
	}
	var observedAt time.Time
	var price float64
	var source string
	err = s.pool.QueryRow(ctx, `SELECT timestamp,close::float8,source FROM candles WHERE UPPER(symbol)=UPPER($1) AND timestamp >= $2 ORDER BY timestamp LIMIT 1`, record.Thesis.Thesis.InstrumentID, review.ScheduledAt).Scan(&observedAt, &price, &source)
	if err != nil {
		return exploratorypaper.ReviewObservation{}, fmt.Errorf("market observation unavailable: %w", err)
	}
	observedAt = observedAt.UTC()
	if price <= 0 || strings.TrimSpace(source) == "" {
		return exploratorypaper.ReviewObservation{}, fmt.Errorf("market observation provenance is incomplete")
	}
	state, err := calendar.SessionState(observedAt)
	if err != nil {
		return exploratorypaper.ReviewObservation{}, err
	}
	items, err := s.pool.Query(ctx, `SELECT evidence_id::text,source_ref,observed_at,quality_score::float8,supports_candidate,contradicts_candidate FROM candidate_evidence_items WHERE candidate_id=$1::uuid ORDER BY observed_at,evidence_id`, record.CandidateID)
	if err != nil {
		return exploratorypaper.ReviewObservation{}, fmt.Errorf("review evidence unavailable: %w", err)
	}
	defer items.Close()
	var evidence []exploratorypaper.RelevantEvidence
	for items.Next() {
		var item exploratorypaper.EvidenceReference
		var sourceRef string
		var quality float64
		var supports, contradicts bool
		if err := items.Scan(&item.EvidenceID, &sourceRef, &item.ObservedAt, &quality, &supports, &contradicts); err != nil {
			return exploratorypaper.ReviewObservation{}, err
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(sourceRef)), "http") {
			return exploratorypaper.ReviewObservation{}, fmt.Errorf("review evidence source provenance is incomplete")
		}
		item.SourceID, item.SourceURL, item.Quality = sourceRef, sourceRef, fmt.Sprintf("%.3f", quality)
		signal := exploratorypaper.EvidenceIrrelevant
		if contradicts {
			signal = exploratorypaper.EvidenceInvalidates
		} else if supports {
			signal = exploratorypaper.EvidenceStrengthens
		}
		evidence = append(evidence, exploratorypaper.RelevantEvidence{Reference: item, IssuerID: record.Thesis.Thesis.IssuerID, InstrumentID: record.Thesis.Thesis.InstrumentID, Signal: signal, Reason: "canonical candidate evidence reassessment"})
	}
	if err := items.Err(); err != nil || len(evidence) == 0 {
		if err != nil {
			return exploratorypaper.ReviewObservation{}, err
		}
		return exploratorypaper.ReviewObservation{}, fmt.Errorf("no canonical review evidence available")
	}
	ledger, err := loadPaperLedger(ctx, s.pool, record.EntryFill.FillID)
	if err != nil {
		return exploratorypaper.ReviewObservation{}, err
	}
	receivedAt := time.Now().UTC()
	tick := papertrading.MarketTick{TickID: fmt.Sprintf("review-%s-%d", record.Position.PositionID, review.SessionNumber), InstrumentID: record.Thesis.Thesis.InstrumentID, Bid: price, Ask: price, Last: price, AvailableQuantity: record.EntryFill.Quantity, Timestamp: observedAt, ReceivedAt: receivedAt, Session: papertrading.Session(state), Source: source}
	return exploratorypaper.ReviewObservation{
		Tick: tick, Price: price, PriceSource: source, Evidence: evidence, Calendar: calendar, Ledger: ledger,
		QuoteMode: "MODELED_CANDLE_CLOSE", LiquidityMode: "MODELED_POSITION_CAPACITY", ActualQuoteAvailable: false,
		ObservedAt: observedAt, ReceivedAt: receivedAt,
		Excursion: exploratorypaper.ExcursionCoverage{Status: "INCOMPLETE", WindowStart: record.Position.EntryAt, WindowEnd: observedAt, ExpectedObservationCount: review.SessionNumber, Cadence: "one_observation_per_review", SourceProvenance: true, KnownGaps: []string{"candle-close-only-review-observation"}},
		Path:      []exploratorypaper.PriceObservation{{ObservationID: tick.TickID, At: observedAt, Price: price, Source: source}},
	}, nil
}

func configuredExploratoryCalendar() (exploratorypaper.SessionCalendar, error) {
	manifestPath := strings.TrimSpace(os.Getenv("PAPER_SESSION_CALENDAR_FILE"))
	if manifestPath == "" {
		manifestPath = "config/paper-02/session-calendar-us-equities-2026-v2.json"
	}
	manifestPath = resolveRuntimePath(manifestPath)
	if manifest, err := exploratorypaper.LoadVersionedSessionCalendar(manifestPath); err == nil {
		return manifest.SessionCalendar()
	}
	raw := strings.TrimSpace(os.Getenv("PAPER_SESSION_CALENDAR_JSON"))
	if raw == "" {
		return exploratorypaper.SessionCalendar{}, fmt.Errorf("versioned PAPER session calendar is unavailable at %q and PAPER_SESSION_CALENDAR_JSON is not configured; calendar state is unknown", manifestPath)
	}
	var calendar exploratorypaper.SessionCalendar
	if err := json.Unmarshal([]byte(raw), &calendar); err != nil {
		return exploratorypaper.SessionCalendar{}, err
	}
	return calendar, calendar.Validate()
}

func loadPaperLedger(ctx context.Context, pool *pgxpool.Pool, fillID string) (papertrading.PaperAccount, error) {
	var accountID string
	if err := pool.QueryRow(ctx, `SELECT account_id FROM paper_ledger_events WHERE fill_id=$1`, fillID).Scan(&accountID); err != nil {
		return papertrading.PaperAccount{}, fmt.Errorf("paper ledger identity unavailable: %w", err)
	}
	var account papertrading.PaperAccount
	var environment string
	var updatedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT account_id,contract_version,environment,currency,initial_cash::float8,cash::float8,equity::float8,realized_pnl::float8,fees::float8,updated_at FROM paper_accounts WHERE account_id=$1`, accountID).Scan(&account.AccountID, &account.ContractVersion, &environment, &account.Currency, &account.InitialCash, &account.Cash, &account.Equity, &account.RealizedPnL, &account.Fees, &updatedAt); err != nil {
		return papertrading.PaperAccount{}, err
	}
	account.Environment = papertrading.Environment(environment)
	rows, err := pool.Query(ctx, `SELECT event_id,contract_version,fill_id,order_id,workflow_id,instrument_id,direction,quantity::float8,price::float8,fee::float8,cash_delta::float8,realized_pnl::float8,occurred_at FROM paper_ledger_events WHERE account_id=$1 ORDER BY occurred_at,event_id`, accountID)
	if err != nil {
		return papertrading.PaperAccount{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var event papertrading.LedgerEvent
		if err := rows.Scan(&event.EventID, &event.ContractVersion, &event.FillID, &event.OrderID, &event.WorkflowID, &event.InstrumentID, &event.Direction, &event.Quantity, &event.Price, &event.Fee, &event.CashDelta, &event.RealizedPnL, &event.OccurredAt); err != nil {
			return papertrading.PaperAccount{}, err
		}
		account.Events = append(account.Events, event)
	}
	if err := rows.Err(); err != nil {
		return papertrading.PaperAccount{}, err
	}
	return account, nil
}
