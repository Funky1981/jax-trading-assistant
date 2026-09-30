package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	candidatesmod "jax-trading-assistant/internal/modules/candidates"
	"jax-trading-assistant/internal/modules/exploratorypaper"
	"jax-trading-assistant/internal/modules/papertrading"
	"jax-trading-assistant/internal/modules/portfoliorisk"
	"jax-trading-assistant/internal/modules/workflow"
)

const (
	canonicalHandoffPolicyEnv = "JAX_PORTFOLIO_RISK_POLICY_FILE"
	canonicalHandoffMaxAgeEnv = "JAX_PORTFOLIO_STATE_MAX_AGE"
)

var errCanonicalHandoffRejected = errors.New("canonical PAPER handoff rejected")

type canonicalHandoffService struct {
	pool        *pgxpool.Pool
	accountID   string
	now         func() time.Time
	market      marketDataSafetyPolicy
	marketErr   error
	policy      portfoliorisk.RiskPolicy
	policyErr   error
	economic    candidateEconomicPolicy
	economicErr error
	maxAge      time.Duration
	calendar    *exploratorypaper.SessionCalendar
}

type canonicalPrepareResult struct {
	CandidateID         string                        `json:"candidateId"`
	Symbol              string                        `json:"symbol"`
	InstrumentID        string                        `json:"instrumentId"`
	IssuerID            string                        `json:"issuerId"`
	Entry               float64                       `json:"entry"`
	Stop                float64                       `json:"stop"`
	Target              float64                       `json:"target"`
	SlippageAllowance   float64                       `json:"slippageAllowance"`
	RiskAllocation      float64                       `json:"requestedRiskAllocation"`
	RequestedLeverage   float64                       `json:"requestedLeverage"`
	RequestedNotional   float64                       `json:"requestedNotional"`
	RequestedQuantity   float64                       `json:"requestedQuantity"`
	ResultingNotional   float64                       `json:"resultingNotional"`
	Outcome             portfoliorisk.DecisionOutcome `json:"riskOutcome"`
	ReasonCodes         []portfoliorisk.ReasonCode    `json:"reasonCodes"`
	PortfolioSnapshotID string                        `json:"portfolioSnapshotId"`
	RiskDecisionID      string                        `json:"riskDecisionId"`
	WorkflowID          string                        `json:"workflowId,omitempty"`
	WorkflowState       workflow.State                `json:"workflowState,omitempty"`
	PolicyID            string                        `json:"policyId"`
	PreparedAt          time.Time                     `json:"preparedAt"`
}

type canonicalHandoffInput struct {
	Candidate  candidatesmod.Candidate
	Economic   candidatesmod.CandidateEconomicInput
	Score      candidatesmod.EvidenceScoreSummary
	Evidence   []exploratorypaper.EvidenceReference
	Assessment exploratorypaper.EvidenceAssessment
	Metadata   map[string]json.RawMessage
}

func newCanonicalHandoffService(pool *pgxpool.Pool) *canonicalHandoffService {
	service := &canonicalHandoffService{pool: pool, accountID: strings.TrimSpace(os.Getenv("PAPER_ACCOUNT_ID")), now: func() time.Time { return time.Now().UTC() }}
	service.market, service.marketErr = marketDataSafetyPolicyFromEnv()
	service.policy, service.policyErr = loadCanonicalPortfolioRiskPolicy()
	service.economic, service.economicErr = loadCandidateEconomicPolicy()
	maxAgeText := strings.TrimSpace(os.Getenv(canonicalHandoffMaxAgeEnv))
	if age, err := time.ParseDuration(maxAgeText); err != nil || age <= 0 {
		service.maxAge = 0
	} else {
		service.maxAge = age
	}
	return service
}

func (s *canonicalHandoffService) configurationError() error {
	if s == nil || s.pool == nil {
		return errors.New("canonical PAPER handoff database is unavailable")
	}
	if s.accountID == "" {
		return errors.New("canonical PAPER handoff requires PAPER_ACCOUNT_ID")
	}
	if s.marketErr != nil {
		return fmt.Errorf("canonical market-data policy unavailable: %w", s.marketErr)
	}
	if s.policyErr != nil {
		return fmt.Errorf("canonical portfolio risk policy unavailable: %w", s.policyErr)
	}
	if s.economicErr != nil {
		return fmt.Errorf("canonical economic identity policy unavailable: %w", s.economicErr)
	}
	if s.maxAge <= 0 {
		return errors.New("canonical portfolio state freshness must be positive")
	}
	return nil
}

func (s *canonicalHandoffService) sessionCalendar() (exploratorypaper.SessionCalendar, error) {
	if s.calendar != nil {
		return *s.calendar, nil
	}
	return configuredExploratoryCalendar()
}

func loadCanonicalHandoffInput(ctx context.Context, tx pgx.Tx, candidateID uuid.UUID, now time.Time, requireCurrent bool) (canonicalHandoffInput, error) {
	var input canonicalHandoffInput
	store := candidatesmod.NewStore(nil)
	candidate, err := store.GetByIDTx(ctx, tx, candidateID)
	if err != nil {
		return input, err
	}
	input.Candidate = *candidate
	if !strings.EqualFold(strings.TrimSpace(candidate.Source), "world-monitor") {
		return input, fmt.Errorf("canonical handoff only accepts source=world-monitor")
	}
	if candidate.Status != candidatesmod.StatusAwaitingApproval || candidate.ExpiresAt == nil || (requireCurrent && !candidate.ExpiresAt.After(now.UTC())) {
		return input, fmt.Errorf("candidate is not current and awaiting human approval")
	}
	if candidate.ExecutionInstructionID != nil || candidate.TradeID != nil {
		return input, fmt.Errorf("candidate already has conflicting execution or trade identity")
	}
	input.Economic, err = store.GetCandidateEconomicInputTx(ctx, tx, candidateID)
	if err != nil {
		return input, fmt.Errorf("candidate economic input is required: %w", err)
	}
	if candidate.Metadata != nil {
		if err := json.Unmarshal(*candidate.Metadata, &input.Metadata); err != nil {
			return input, fmt.Errorf("candidate metadata is malformed: %w", err)
		}
	}
	var scoredAt time.Time
	if err := tx.QueryRow(ctx, `SELECT support_score::float8,contradiction_score::float8,quality_score::float8,freshness_score::float8,overall_evidence_score::float8,evidence_item_count,supporting_item_count,contradictory_item_count,stale_item_count,evidence_status,evidence_ready,evidence_gate_ready,approval_granted,broker_execution_allowed,execution_instruction_created,scored_at FROM candidate_evidence_scores WHERE candidate_id=$1 ORDER BY scored_at DESC LIMIT 1`, candidateID).Scan(&input.Score.SupportScore, &input.Score.ContradictionScore, &input.Score.QualityScore, &input.Score.FreshnessScore, &input.Score.OverallEvidenceScore, &input.Score.EvidenceItemCount, &input.Score.SupportingItemCount, &input.Score.ContradictoryItemCount, &input.Score.StaleItemCount, &input.Score.EvidenceStatus, &input.Score.EvidenceReady, &input.Score.EvidenceGateReady, &input.Score.ApprovalGranted, &input.Score.BrokerExecutionAllowed, &input.Score.ExecutionInstructionCreated, &scoredAt); err != nil {
		return input, fmt.Errorf("canonical evidence score unavailable: %w", err)
	}
	var currentDecision string
	if err := tx.QueryRow(ctx, `SELECT decision FROM genuine_event_decisions WHERE candidate_id=$1 AND is_current ORDER BY decision_at DESC LIMIT 1`, candidateID).Scan(&currentDecision); err != nil || currentDecision != string(exploratorypaper.DecisionCandidate) {
		return input, fmt.Errorf("current canonical event decision is not CANDIDATE")
	}
	input.Score.CandidateID = candidateID
	if input.Score.BrokerExecutionAllowed || input.Score.ExecutionInstructionCreated || input.Score.ApprovalGranted {
		return input, fmt.Errorf("candidate evidence projection contains conflicting execution or approval state")
	}
	var eventSourceID string
	var eventID uuid.NullUUID
	var eventType, headline, summary, mappingReason string
	var eventAt time.Time
	var synthetic bool
	if err := tx.QueryRow(ctx, `SELECT w.source_event_id,w.normalized_event_id,w.event_type,w.headline,COALESCE(w.summary,''),w.mapping_reason,w.event_time,e.is_synthetic FROM world_monitor_research_inbox w JOIN event_normalized e ON e.id=w.normalized_event_id WHERE w.candidate_id=$1`, candidateID).Scan(&eventSourceID, &eventID, &eventType, &headline, &summary, &mappingReason, &eventAt, &synthetic); err != nil {
		return input, fmt.Errorf("durable World Monitor source/event record is unavailable: %w", err)
	}
	if input.Candidate.Metadata == nil {
		return input, fmt.Errorf("durable World Monitor candidate metadata is unavailable")
	}
	var sourceMetadata struct {
		WorldMonitor struct {
			SourceEventID     string `json:"sourceEventId"`
			NormalizedEventID string `json:"normalizedEventId"`
			EventType         string `json:"eventType"`
			Headline          string `json:"headline"`
			Summary           string `json:"summary"`
			MappingReason     string `json:"mappingReason"`
			PromotedSymbol    string `json:"promotedSymbol"`
			IsSynthetic       bool   `json:"isSynthetic"`
		} `json:"worldMonitor"`
	}
	if err := json.Unmarshal(*input.Candidate.Metadata, &sourceMetadata); err != nil {
		return input, fmt.Errorf("decode durable World Monitor metadata: %w", err)
	}
	metadata := sourceMetadata.WorldMonitor
	parsedEventID, parseErr := uuid.Parse(metadata.NormalizedEventID)
	if parseErr != nil || !eventID.Valid || parsedEventID != eventID.UUID || metadata.SourceEventID == "" || metadata.SourceEventID != eventSourceID || synthetic || metadata.IsSynthetic || metadata.EventType != eventType || metadata.Headline != headline || metadata.Summary != summary || metadata.MappingReason != mappingReason || input.Candidate.CatalystTimestamp == nil || !eventAt.UTC().Equal(input.Candidate.CatalystTimestamp.UTC()) || !strings.EqualFold(metadata.PromotedSymbol, input.Candidate.Symbol) {
		return input, fmt.Errorf("candidate World Monitor source/event metadata does not match its durable intake record")
	}
	rows, err := tx.Query(ctx, `SELECT evidence_id::text,source_ref,observed_at,quality_score::float8,freshness_status,supports_candidate,contradicts_candidate FROM candidate_evidence_items WHERE candidate_id=$1 ORDER BY observed_at,evidence_id`, candidateID)
	if err != nil {
		return input, err
	}
	defer rows.Close()
	independent := map[string]struct{}{}
	for rows.Next() {
		var ref exploratorypaper.EvidenceReference
		var freshness string
		var supports, contradicts bool
		var quality float64
		if err := rows.Scan(&ref.EvidenceID, &ref.SourceURL, &ref.ObservedAt, &quality, &freshness, &supports, &contradicts); err != nil {
			return input, err
		}
		ref.SourceID = ref.SourceURL
		ref.Quality = fmt.Sprintf("%.3f/%s", quality, freshness)
		ref.ObservedAt = ref.ObservedAt.UTC()
		if !isDurableEvidenceReference(ref.SourceURL) || ref.ObservedAt.IsZero() || freshness != string(candidatesmod.FreshnessStatusFresh) {
			return input, fmt.Errorf("candidate evidence source or freshness is not canonical")
		}
		if supports && contradicts {
			return input, fmt.Errorf("candidate evidence item is simultaneously supporting and contradictory")
		}
		independent[ref.SourceID] = struct{}{}
		input.Evidence = append(input.Evidence, ref)
		input.Assessment.Contradictory = input.Assessment.Contradictory || contradicts
	}
	if err := rows.Err(); err != nil {
		return input, err
	}
	if len(input.Evidence) != input.Score.EvidenceItemCount || input.Score.EvidenceStatus != candidatesmod.EvidenceStatusSufficient || !input.Score.EvidenceReady || !input.Score.EvidenceGateReady || input.Score.StaleItemCount != 0 || input.Score.ContradictoryItemCount != 0 || len(independent) < 2 {
		return input, fmt.Errorf("canonical reviewed evidence is incomplete, stale, contradictory, or insufficient")
	}
	input.Assessment = exploratorypaper.EvidenceAssessment{
		Provider: "candidate_evidence_scores", PolicyVersion: "candidate-evidence-scoring-v1",
		EvidenceSetFingerprint: exploratorypaper.EvidenceSetFingerprint(input.Evidence),
		ReviewedAt:             scoredAt.UTC(), SourceBacked: true, QualityState: string(input.Score.EvidenceStatus),
		QualityScore: input.Score.QualityScore, RequiredQualityScore: 0.70, EvidenceReady: input.Score.EvidenceReady,
		EvidenceGateReady: input.Score.EvidenceGateReady, Corroborated: len(independent) >= 2,
		Contradictory: input.Assessment.Contradictory, IndependentSourceGroups: len(independent),
		IssuerRelevant: true, InstrumentRelevant: true,
	}
	if err := input.Assessment.ValidateFor(input.Evidence, input.Economic.IssuerID, input.Economic.InstrumentID); err != nil {
		return input, fmt.Errorf("canonical reviewed evidence validation: %w", err)
	}
	return input, nil
}

type paperAccountRead struct {
	account papertrading.PaperAccount
	updated time.Time
}

func loadPaperAccountRead(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, accountID string) (paperAccountRead, error) {
	var result paperAccountRead
	var environment string
	err := queryer.QueryRow(ctx, `SELECT account_id,contract_version,environment,currency,initial_cash::float8,cash::float8,equity::float8,realized_pnl::float8,fees::float8,updated_at FROM paper_accounts WHERE account_id=$1`, accountID).Scan(&result.account.AccountID, &result.account.ContractVersion, &environment, &result.account.Currency, &result.account.InitialCash, &result.account.Cash, &result.account.Equity, &result.account.RealizedPnL, &result.account.Fees, &result.updated)
	if err != nil {
		return result, fmt.Errorf("load existing PAPER account %q: %w", accountID, err)
	}
	result.account.Environment = papertrading.Environment(environment)
	if result.account.Environment != papertrading.EnvironmentPaper {
		return result, fmt.Errorf("configured account is not PAPER")
	}
	result.account.Positions = map[string]papertrading.LedgerPosition{}
	rows, err := queryer.Query(ctx, `SELECT event_id,contract_version,fill_id,order_id,workflow_id,instrument_id,direction,quantity::float8,price::float8,fee::float8,cash_delta::float8,realized_pnl::float8,occurred_at FROM paper_ledger_events WHERE account_id=$1 ORDER BY occurred_at,event_id`, accountID)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var event papertrading.LedgerEvent
		if err := rows.Scan(&event.EventID, &event.ContractVersion, &event.FillID, &event.OrderID, &event.WorkflowID, &event.InstrumentID, &event.Direction, &event.Quantity, &event.Price, &event.Fee, &event.CashDelta, &event.RealizedPnL, &event.OccurredAt); err != nil {
			return result, err
		}
		event.OccurredAt = event.OccurredAt.UTC()
		result.account.Events = append(result.account.Events, event)
		position := result.account.Positions[event.InstrumentID]
		if event.Direction == "LONG" {
			quantity := position.Quantity + event.Quantity
			position = papertrading.LedgerPosition{InstrumentID: event.InstrumentID, Quantity: quantity, AverageCost: (position.Quantity*position.AverageCost + event.Quantity*event.Price) / quantity}
		} else if event.Direction == "SHORT" && position.Quantity >= event.Quantity {
			position.Quantity -= event.Quantity
			if position.Quantity > 1e-9 {
				result.account.Positions[event.InstrumentID] = position
				continue
			}
			delete(result.account.Positions, event.InstrumentID)
			continue
		} else {
			return result, fmt.Errorf("PAPER ledger has unsupported or unmatched short opening semantics")
		}
		result.account.Positions[event.InstrumentID] = position
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if _, err := papertrading.RestorePaperLedger(result.account); err != nil {
		return result, fmt.Errorf("PAPER account and append-only ledger do not reconcile: %w", err)
	}
	return result, nil
}

func (s *canonicalHandoffService) instrumentSymbol(instrumentID string) (string, error) {
	if s.economicErr != nil {
		return "", s.economicErr
	}
	var matches []string
	for symbol, identity := range s.economic.Instruments {
		if strings.TrimSpace(identity.InstrumentID) == instrumentID {
			matches = append(matches, strings.ToUpper(strings.TrimSpace(symbol)))
		}
	}
	sort.Strings(matches)
	if len(matches) != 1 {
		return "", fmt.Errorf("instrument %q does not have one explicit symbol mapping", instrumentID)
	}
	return matches[0], nil
}

func (s *canonicalHandoffService) buildPortfolioSnapshot(ctx context.Context, queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, now time.Time) (portfoliorisk.PortfolioSnapshot, paperAccountRead, error) {
	account, err := loadPaperAccountRead(ctx, queryer, s.accountID)
	if err != nil {
		return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, err
	}
	if !portfoliorisk.SupportedCurrency(account.account.Currency) || !strings.EqualFold(account.account.Currency, s.policy.Currency) {
		return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, fmt.Errorf("PAPER account currency is unsupported or differs from explicit risk policy")
	}
	positions := make([]portfoliorisk.Position, 0, len(account.account.Positions))
	markedValue := 0.0
	provenance := []string{"paper_accounts:" + account.account.AccountID, "paper_account_updated_at:" + account.updated.UTC().Format(time.RFC3339Nano)}
	asOf := account.updated.UTC()
	capturedAt := account.updated.UTC()
	for _, position := range account.account.Positions {
		if position.Quantity <= 0 {
			return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, fmt.Errorf("PAPER account contains unsupported non-positive open quantity")
		}
		symbol, err := s.instrumentSymbol(position.InstrumentID)
		if err != nil {
			return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, err
		}
		observation, err := loadCanonicalQuoteObservationFrom(ctx, queryer, symbol, now, s.market)
		if err != nil {
			return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, fmt.Errorf("cannot value material PAPER position %s: %w", position.InstrumentID, err)
		}
		marketValue := position.Quantity * observation.Last
		if !finitePositive(marketValue) {
			return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, fmt.Errorf("PAPER position %s market value is invalid", position.InstrumentID)
		}
		positions = append(positions, portfoliorisk.Position{InstrumentID: position.InstrumentID, InstrumentResolved: true,
			Currency: account.account.Currency, SignedQuantity: position.Quantity,
			Price: portfoliorisk.KnownNumber(observation.Last, observation.Source), MarketValue: portfoliorisk.KnownNumber(marketValue, observation.Source),
			CostBasis:     portfoliorisk.KnownNumber(position.Quantity*position.AverageCost, "paper_ledger_events"),
			ValuationAsOf: observation.ProviderAt.UTC(), PriceSource: observation.Source,
			Provenance: []string{"quotes:" + observation.Source + ":" + observation.ProviderAt.UTC().Format(time.RFC3339Nano), "quote_last_trade_at:" + observation.LastProviderAt.UTC().Format(time.RFC3339Nano), "quote_received_at:" + observation.ReceivedAt.UTC().Format(time.RFC3339Nano), "instrument_mapping:" + position.InstrumentID + "=" + symbol}})
		markedValue += marketValue
		if observation.ProviderAt.After(asOf) {
			asOf = observation.ProviderAt.UTC()
		}
		if observation.LastProviderAt.After(asOf) {
			asOf = observation.LastProviderAt.UTC()
		}
		if observation.ReceivedAt.After(capturedAt) {
			capturedAt = observation.ReceivedAt.UTC()
		}
		provenance = append(provenance, "open_position:"+position.InstrumentID)
	}
	for _, event := range account.account.Events {
		provenance = append(provenance, "paper_ledger_event:"+event.EventID)
	}
	if len(positions) == 0 {
		provenance = append(provenance, "paper_ledger_events:known-empty")
	}
	snapshot, err := portfoliorisk.BuildSnapshot(portfoliorisk.PortfolioSnapshot{
		AccountID: s.accountID, AsOf: asOf, CapturedAt: capturedAt, Provider: "paper_accounts+paper_ledger_events+persisted-provider-quotes",
		Currency: strings.ToUpper(account.account.Currency), Synthetic: false, KnownEmpty: len(positions) == 0,
		Cash:      portfoliorisk.KnownNumber(account.account.Cash, "paper_accounts.cash"),
		Equity:    portfoliorisk.KnownNumber(account.account.Cash+markedValue, "derived:paper_accounts.cash+current-position-market-value"),
		Positions: positions, Provenance: provenance, ValuationBasis: "cash-plus-fresh-provider-market-value",
	})
	if err != nil {
		return portfoliorisk.PortfolioSnapshot{}, paperAccountRead{}, err
	}
	return snapshot, account, nil
}

func finitePositive(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
}

func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func canonicalDigest(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
}

func nullableCanonicalString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func retryCanonicalTransaction(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
}

func (s *canonicalHandoffService) prepare(ctx context.Context, candidateID uuid.UUID) (canonicalPrepareResult, error) {
	for attempt := 0; attempt < 3; attempt++ {
		result, err := s.prepareOnce(ctx, candidateID)
		if err == nil || !retryCanonicalTransaction(err) || attempt == 2 {
			return result, err
		}
	}
	return canonicalPrepareResult{}, fmt.Errorf("canonical PREPARE transaction retries exhausted")
}

func (s *canonicalHandoffService) prepareOnce(ctx context.Context, candidateID uuid.UUID) (canonicalPrepareResult, error) {
	var response canonicalPrepareResult
	if err := s.configurationError(); err != nil {
		return response, err
	}
	if candidateID == uuid.Nil {
		return response, fmt.Errorf("candidate id is required")
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
	var lockedID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM candidate_trades WHERE id=$1 FOR UPDATE`, candidateID).Scan(&lockedID); err != nil {
		return response, fmt.Errorf("load and lock candidate: %w", err)
	}
	input, err := loadCanonicalHandoffInput(ctx, tx, candidateID, now, true)
	if err != nil {
		return response, err
	}
	if err := s.validateCanonicalCandidate(input); err != nil {
		return response, err
	}
	if err := ensureCandidateNotConsumed(ctx, tx, candidateID); err != nil {
		return response, err
	}
	snapshot, _, err := s.buildPortfolioSnapshot(ctx, tx, now)
	if err != nil {
		return response, err
	}
	freshness := snapshot.AssessFreshness(now, s.maxAge)
	if freshness.Status != portfoliorisk.FreshnessFresh {
		return response, fmt.Errorf("PAPER portfolio state is %s: %s", freshness.Status, freshness.Reason)
	}
	entry, stop, target := *input.Candidate.EntryPrice, *input.Candidate.StopLoss, *input.Candidate.TakeProfit
	if !finitePositive(entry) || !finitePositive(stop) || !finitePositive(target) || !finitePositive(snapshot.Equity.Value) || input.Economic.SlippageAllowance == nil || !finiteNonNegative(*input.Economic.SlippageAllowance) {
		return response, fmt.Errorf("candidate entry, stop, target, equity, and explicit slippage allowance must be valid")
	}
	slippage := *input.Economic.SlippageAllowance
	direction, err := canonicalCandidateDirection(input.Candidate.Direction, input.Candidate.SignalType)
	if err != nil {
		return response, err
	}
	if direction != "LONG" {
		return response, fmt.Errorf("short opening is unsupported by the current PAPER ledger")
	}
	if !(stop < entry && target > entry) {
		return response, fmt.Errorf("LONG candidate stop/entry/target geometry is invalid")
	}
	if input.Economic.RequestedLeverage > *s.policy.MaximumLeverage {
		return response, fmt.Errorf("candidate requested leverage exceeds explicit portfolio policy")
	}
	if s.policy.MaximumRiskAllocation != nil && input.Economic.RiskAllocation > *s.policy.MaximumRiskAllocation {
		return response, fmt.Errorf("candidate risk allocation exceeds explicit portfolio policy")
	}
	riskBudget := snapshot.Equity.Value * input.Economic.RiskAllocation
	perUnitRisk := math.Abs(entry-stop) + slippage
	if !finitePositive(riskBudget) || !finitePositive(perUnitRisk) {
		return response, fmt.Errorf("candidate risk budget or per-unit risk is invalid")
	}
	requestedQuantity := riskBudget / perUnitRisk
	rawNotional := requestedQuantity * entry
	leverageCap := snapshot.Equity.Value * input.Economic.RequestedLeverage
	requestedNotional := math.Min(rawNotional, leverageCap)
	if !finitePositive(requestedQuantity) || !finitePositive(requestedNotional) {
		return response, fmt.Errorf("requested economic exposure is invalid")
	}
	analytics, err := portfoliorisk.CalculateExposure(snapshot, snapshot.AsOf, s.maxAge)
	if err != nil {
		return response, fmt.Errorf("calculate canonical exposure: %w", err)
	}
	signedValue := requestedNotional
	recommendation := portfoliorisk.RecommendationRiskInput{
		RecommendationID: candidateID.String(), InstrumentID: input.Economic.InstrumentID,
		Currency: snapshot.Currency, SignedMarketValue: signedValue,
		RiskAllocation:    portfoliorisk.Limit(input.Economic.RiskAllocation),
		RequestedLeverage: portfoliorisk.Limit(input.Economic.RequestedLeverage), ExecutionAuthority: "NONE",
	}
	decision := portfoliorisk.EvaluateRecommendation(recommendation, snapshot, analytics, s.policy, snapshot.AsOf, s.maxAge)
	var priorWorkflowID string
	var priorWorkflowPayload []byte
	priorErr := tx.QueryRow(ctx, `SELECT workflow_id,payload FROM workflow_instances WHERE recommendation_id=$1 ORDER BY created_at DESC LIMIT 1`, candidateID.String()).Scan(&priorWorkflowID, &priorWorkflowPayload)
	if priorErr == nil {
		var prior struct {
			Workflow workflow.Workflow `json:"workflow"`
		}
		if err := json.Unmarshal(priorWorkflowPayload, &prior); err != nil {
			return response, fmt.Errorf("existing candidate workflow is malformed: %w", err)
		}
		if prior.Workflow.State != workflow.StateAwaitingHumanConfirmation || prior.Workflow.RiskDecisionID != decision.DecisionID || prior.Workflow.PortfolioSnapshotID != snapshot.SnapshotID || prior.Workflow.WorkflowID != priorWorkflowID {
			return response, fmt.Errorf("candidate already has a different or final canonical workflow; explicit supersession is required")
		}
		response = canonicalPrepareResult{
			CandidateID: candidateID.String(), Symbol: input.Candidate.Symbol, InstrumentID: input.Economic.InstrumentID,
			IssuerID: input.Economic.IssuerID, Entry: entry, Stop: stop, Target: target,
			RiskAllocation: input.Economic.RiskAllocation, RequestedLeverage: input.Economic.RequestedLeverage, SlippageAllowance: slippage,
			RequestedNotional: requestedNotional, RequestedQuantity: math.Min(requestedQuantity, requestedNotional/entry),
			ResultingNotional: math.Abs(decision.ResultingValue), Outcome: decision.Outcome, ReasonCodes: decision.ReasonCodes,
			PortfolioSnapshotID: snapshot.SnapshotID, RiskDecisionID: decision.DecisionID, WorkflowID: priorWorkflowID,
			WorkflowState: prior.Workflow.State, PolicyID: s.policy.PolicyID, PreparedAt: prior.Workflow.CreatedAt,
		}
		if err := tx.Commit(ctx); err != nil {
			return canonicalPrepareResult{}, err
		}
		return response, nil
	}
	if !errors.Is(priorErr, pgx.ErrNoRows) {
		return response, fmt.Errorf("load prior candidate workflow: %w", priorErr)
	}
	if err := savePortfolioSnapshotTx(ctx, tx, snapshot); err != nil {
		return response, err
	}
	if err := saveRiskDecisionTx(ctx, tx, decision); err != nil {
		return response, err
	}
	response = canonicalPrepareResult{
		CandidateID: candidateID.String(), Symbol: input.Candidate.Symbol, InstrumentID: input.Economic.InstrumentID,
		IssuerID: input.Economic.IssuerID, Entry: entry, Stop: stop, Target: target,
		RiskAllocation: input.Economic.RiskAllocation, RequestedLeverage: input.Economic.RequestedLeverage, SlippageAllowance: slippage,
		RequestedNotional: requestedNotional, RequestedQuantity: math.Min(requestedQuantity, requestedNotional/entry),
		ResultingNotional: math.Abs(decision.ResultingValue), Outcome: decision.Outcome, ReasonCodes: decision.ReasonCodes,
		PortfolioSnapshotID: snapshot.SnapshotID, RiskDecisionID: decision.DecisionID, PolicyID: s.policy.PolicyID, PreparedAt: now,
	}
	if decision.Outcome == portfoliorisk.DecisionReject {
		if err := tx.Commit(ctx); err != nil {
			return canonicalPrepareResult{}, err
		}
		return response, nil
	}
	store := workflow.NewStore()
	wf, err := store.Create(ctx, workflow.CreateRequest{RiskDecision: decision, Now: now, IdempotencyKey: "canonical-prepare:" + decision.DecisionID})
	if err != nil {
		return canonicalPrepareResult{}, err
	}
	wf, err = store.RequestHumanConfirmation(ctx, workflow.TransitionRequest{WorkflowID: wf.WorkflowID, Actor: "canonical-paper-handoff", ActorRole: workflow.ActorSystem, IdempotencyKey: "canonical-await-human:" + decision.DecisionID, Now: now, Reason: "portfolio risk decision is durably ready for human review"})
	if err != nil {
		return canonicalPrepareResult{}, err
	}
	workflowBytes, err := store.ExportSnapshot()
	if err != nil {
		return canonicalPrepareResult{}, err
	}
	var durable workflow.DurableSnapshot
	if err := json.Unmarshal(workflowBytes, &durable); err != nil {
		return canonicalPrepareResult{}, err
	}
	if err := persistCanonicalWorkflowSnapshot(ctx, tx, durable, wf.WorkflowID, workflow.PaperIntent{}); err != nil {
		return canonicalPrepareResult{}, err
	}
	response.WorkflowID, response.WorkflowState = wf.WorkflowID, wf.State
	if err := tx.Commit(ctx); err != nil {
		return canonicalPrepareResult{}, err
	}
	return response, nil
}

func (s *canonicalHandoffService) validateCanonicalCandidate(input canonicalHandoffInput) error {
	candidate, economic := input.Candidate, input.Economic
	if candidate.EntryPrice == nil || candidate.StopLoss == nil || candidate.TakeProfit == nil || candidate.Direction == "" || candidate.Confidence == nil || candidate.ExpiresAt == nil {
		return fmt.Errorf("candidate durable entry/stop/target/direction/confidence/expiry/slippage facts are incomplete")
	}
	if economic.CandidateID != candidate.ID || strings.TrimSpace(economic.InstrumentID) == "" || strings.TrimSpace(economic.IssuerID) == "" || economic.IdentitySource == "" || economic.IdentityPolicyVersion == "" || economic.SizingPolicyID == "" || economic.SizingPolicyVersion == "" || economic.ContentIdentity == "" || economic.SlippageAllowance == nil || !finiteNonNegative(*economic.SlippageAllowance) {
		return fmt.Errorf("candidate economic input provenance is incomplete or mismatched")
	}
	if candidate.SlippageAllowance != nil && *candidate.SlippageAllowance != *economic.SlippageAllowance {
		return fmt.Errorf("candidate and canonical economic slippage allowance conflict")
	}
	if candidate.Metadata == nil {
		return fmt.Errorf("candidate metadata is unavailable for normalized event and chart provenance")
	}
	var wm struct {
		WorldMonitor struct {
			NormalizedEventID string `json:"normalizedEventId"`
			IsSynthetic       bool   `json:"isSynthetic"`
		} `json:"worldMonitor"`
	}
	if err := json.Unmarshal(*candidate.Metadata, &wm); err != nil || wm.WorldMonitor.NormalizedEventID == "" || wm.WorldMonitor.IsSynthetic {
		return fmt.Errorf("candidate lacks genuine normalized World Monitor event provenance")
	}
	if _, err := s.instrumentSymbol(economic.InstrumentID); err != nil {
		return err
	}
	return nil
}

func canonicalCandidateDirection(direction, signalType string) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(direction))
	if value == "" {
		value = strings.ToUpper(strings.TrimSpace(signalType))
	}
	switch value {
	case "LONG", "BUY":
		return "LONG", nil
	case "SHORT", "SELL":
		return "SHORT", nil
	default:
		return "", fmt.Errorf("candidate direction is unresolved")
	}
}

func ensureCandidateNotConsumed(ctx context.Context, tx pgx.Tx, candidateID uuid.UUID) error {
	var approved, rejected, tickets, instructions, lifecycles, queue int
	queries := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM candidate_approvals WHERE candidate_id=$1 AND decision IN ('approved','rejected')`, &approved},
		{`SELECT COUNT(*) FROM candidate_paper_tickets WHERE candidate_id=$1`, &tickets},
		{`SELECT COUNT(*) FROM execution_instructions WHERE candidate_id=$1`, &instructions},
		{`SELECT COUNT(*) FROM exploratory_paper_lifecycles WHERE candidate_id=$1::text`, &lifecycles},
		{`SELECT COUNT(*) FROM exploratory_paper_entry_queue WHERE candidate_id=$1::text`, &queue},
	}
	for _, query := range queries {
		if err := tx.QueryRow(ctx, query.query, candidateID).Scan(query.dest); err != nil {
			return fmt.Errorf("check candidate prior consumption: %w", err)
		}
	}
	if approved+rejected+tickets+instructions+lifecycles+queue > 0 {
		return fmt.Errorf("candidate has a prior final decision or downstream PAPER/execution artifact")
	}
	return nil
}

func saveRiskDecisionTx(ctx context.Context, tx pgx.Tx, decision portfoliorisk.RiskDecision) error {
	if err := decision.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	reasons := make([]string, len(decision.ReasonCodes))
	for index, reason := range decision.ReasonCodes {
		reasons[index] = string(reason)
	}
	_, err = tx.Exec(ctx, `INSERT INTO portfolio_risk_decisions(decision_id,recommendation_id,portfolio_snapshot_id,analytics_id,policy_id,algorithm,outcome,reason_codes,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(decision_id) DO NOTHING`, decision.DecisionID, decision.RecommendationID, decision.PortfolioSnapshotID, decision.AnalyticsID, decision.PolicyID, decision.Algorithm, string(decision.Outcome), reasons, payload)
	if err != nil {
		return fmt.Errorf("persist portfolio risk decision: %w", err)
	}
	var existing []byte
	if err := tx.QueryRow(ctx, `SELECT payload FROM portfolio_risk_decisions WHERE decision_id=$1`, decision.DecisionID).Scan(&existing); err != nil {
		return fmt.Errorf("portfolio risk decision replay content conflicts")
	}
	var replay portfoliorisk.RiskDecision
	if json.Unmarshal(existing, &replay) != nil || !reflect.DeepEqual(replay, decision) {
		return fmt.Errorf("portfolio risk decision replay content conflicts")
	}
	return nil
}

func savePortfolioSnapshotTx(ctx context.Context, tx pgx.Tx, snapshot portfoliorisk.PortfolioSnapshot) error {
	canonical, err := portfoliorisk.BuildSnapshot(snapshot)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO portfolio_snapshots(snapshot_id,contract_version,identity_algorithm,account_id,as_of,captured_at,provider,currency,synthetic,payload) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT(snapshot_id) DO NOTHING`, canonical.SnapshotID, canonical.ContractVersion, canonical.IdentityAlgorithm, canonical.AccountID, canonical.AsOf, canonical.CapturedAt, canonical.Provider, canonical.Currency, canonical.Synthetic, payload)
	if err != nil {
		return fmt.Errorf("persist portfolio snapshot: %w", err)
	}
	var existing []byte
	if err := tx.QueryRow(ctx, `SELECT payload FROM portfolio_snapshots WHERE snapshot_id=$1`, canonical.SnapshotID).Scan(&existing); err != nil {
		return fmt.Errorf("portfolio snapshot replay content conflicts")
	}
	var replay portfoliorisk.PortfolioSnapshot
	if json.Unmarshal(existing, &replay) != nil {
		return fmt.Errorf("portfolio snapshot replay content conflicts")
	}
	replayed, err := portfoliorisk.BuildSnapshot(replay)
	if err != nil || !reflect.DeepEqual(replayed, canonical) {
		return fmt.Errorf("portfolio snapshot replay content conflicts")
	}
	return nil
}

func persistCanonicalWorkflowSnapshot(ctx context.Context, tx pgx.Tx, snapshot workflow.DurableSnapshot, workflowID string, intent workflow.PaperIntent) error {
	wf, ok := snapshot.Workflows[workflowID]
	if !ok || workflow.ValidateAuditEvents(snapshot.Events[workflowID]) != nil {
		return fmt.Errorf("canonical workflow or audit stream is incomplete")
	}
	payload, err := json.Marshal(struct {
		Workflow    workflow.Workflow    `json:"workflow"`
		PaperIntent workflow.PaperIntent `json:"paperIntent"`
	}{wf, intent})
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `INSERT INTO workflow_instances(workflow_id,contract_version,algorithm,state,revision,recommendation_id,risk_decision_id,portfolio_snapshot_id,analytics_id,policy_id,proposal_id,payload,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(workflow_id) DO UPDATE SET state=EXCLUDED.state,revision=EXCLUDED.revision,payload=EXCLUDED.payload,updated_at=EXCLUDED.updated_at WHERE workflow_instances.contract_version=EXCLUDED.contract_version AND workflow_instances.algorithm=EXCLUDED.algorithm AND workflow_instances.recommendation_id=EXCLUDED.recommendation_id AND workflow_instances.risk_decision_id=EXCLUDED.risk_decision_id AND workflow_instances.portfolio_snapshot_id=EXCLUDED.portfolio_snapshot_id AND workflow_instances.analytics_id=EXCLUDED.analytics_id AND workflow_instances.policy_id=EXCLUDED.policy_id AND workflow_instances.proposal_id IS NOT DISTINCT FROM EXCLUDED.proposal_id`, wf.WorkflowID, wf.ContractVersion, wf.Algorithm, string(wf.State), wf.Revision, wf.RecommendationID, wf.RiskDecisionID, wf.PortfolioSnapshotID, wf.AnalyticsID, wf.PolicyID, nullableCanonicalString(wf.ProposalID), payload, wf.CreatedAt, wf.UpdatedAt)
	if err != nil {
		return fmt.Errorf("persist canonical workflow: %w", err)
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("persist canonical workflow binding conflict")
	}
	for _, event := range snapshot.Events[workflowID] {
		if err := event.Validate(); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `INSERT INTO workflow_audit_events(event_id,workflow_id,sequence,previous_state,next_state,action,actor,actor_role,idempotency_key,input_fingerprint,content_identity,reason,occurred_at,transition_version) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(event_id) DO NOTHING`, event.EventID, event.WorkflowID, event.Sequence, string(event.PreviousState), string(event.NextState), string(event.Action), event.Actor, string(event.ActorRole), event.IdempotencyKey, event.InputFingerprint, event.ContentIdentity, nullableCanonicalString(event.Reason), event.OccurredAt, event.TransitionVersion)
		if err != nil {
			return fmt.Errorf("persist workflow audit: %w", err)
		}
		if result.RowsAffected() == 0 {
			var stored workflow.TransitionEvent
			var sequence int64
			var previous, next, action, actorRole, reason string
			if err := tx.QueryRow(ctx, `SELECT event_id,workflow_id,sequence,previous_state,next_state,action,actor,actor_role,idempotency_key,input_fingerprint,content_identity,COALESCE(reason,''),occurred_at,transition_version FROM workflow_audit_events WHERE event_id=$1`, event.EventID).Scan(&stored.EventID, &stored.WorkflowID, &sequence, &previous, &next, &action, &stored.Actor, &actorRole, &stored.IdempotencyKey, &stored.InputFingerprint, &stored.ContentIdentity, &reason, &stored.OccurredAt, &stored.TransitionVersion); err != nil {
				return fmt.Errorf("verify replayed workflow audit event: %w", err)
			}
			stored.Sequence, stored.PreviousState, stored.NextState = uint64(sequence), workflow.State(previous), workflow.State(next)
			stored.Action, stored.ActorRole, stored.Reason = workflow.Action(action), workflow.ActorRole(actorRole), reason
			stored.OccurredAt = stored.OccurredAt.UTC()
			event.OccurredAt = event.OccurredAt.UTC()
			if !reflect.DeepEqual(stored, event) {
				return fmt.Errorf("workflow audit event replay content conflicts")
			}
		}
	}
	return nil
}

var _ = savePortfolioSnapshotTx
