package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"jax-trading-assistant/internal/modules/audit"
	"jax-trading-assistant/libs/observability"
	"jax-trading-assistant/libs/risk"

	"github.com/google/uuid"
)

// BrokerClient interface for broker operations
type BrokerClient interface {
	GetAccount(ctx context.Context) (*BrokerAccountInfo, error)
	PlaceOrder(ctx context.Context, order *BrokerOrderRequest) (*BrokerOrderResponse, error)
	GetOrderStatus(ctx context.Context, orderID int) (*BrokerOrderStatus, error)
	GetPositions(ctx context.Context) (*BrokerPositionsResponse, error)
}

// TradeStore interface for database operations
type TradeStore interface {
	GetSignal(ctx context.Context, signalID uuid.UUID) (*Signal, error)
	GetTradeBySignal(ctx context.Context, signalID uuid.UUID) (*TradeResult, error)
	GetLatestOrderIntentBySignal(ctx context.Context, signalID uuid.UUID) (*OrderIntentSummary, error)
	StoreTrade(ctx context.Context, trade *TradeResult) error
	UpdateTradeApproval(ctx context.Context, signalID uuid.UUID, orderID string) error
	UpdateTradeStatus(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error
	UpdateOrderIntentStatusByTrade(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error
	GetDailyRisk(ctx context.Context) (float64, error)
	StoreOrderIntent(ctx context.Context, intent *OrderIntent) (string, error)
	UpdateOrderIntentStatus(ctx context.Context, signalID uuid.UUID, status *BrokerOrderStatus) error
}

// BrokerAccountInfo represents broker account information
type BrokerAccountInfo struct {
	NetLiquidation float64
	BuyingPower    float64
	Currency       string
}

// BrokerOrderRequest represents a broker order request
type BrokerOrderRequest struct {
	Symbol     string
	Action     string
	Quantity   int
	OrderType  string
	LimitPrice *float64
	StopPrice  *float64
}

// BrokerOrderResponse represents a broker order response
type BrokerOrderResponse struct {
	OrderID int
	Success bool
	Message string
}

// BrokerOrderStatus represents broker order status
type BrokerOrderStatus struct {
	OrderID      int
	Status       string // Submitted, Filled, Cancelled, etc.
	FilledQty    int
	AvgFillPrice float64
}

// BrokerPositionsResponse represents broker positions
type BrokerPositionsResponse struct {
	Positions []BrokerPosition
}

// BrokerPosition represents a single position
type BrokerPosition struct {
	Symbol   string
	Quantity int
}

// Service provides trade execution functionality
type Service struct {
	engine     *Engine
	broker     BrokerClient
	store      TradeStore
	orderType  string
	riskParams RiskParameters
	// enforcer applies versioned policy constraints (L16) on top of the
	// engine's built-in position-level checks. May be nil (no-op).
	enforcer *risk.Enforcer
	audit    *audit.Service
}

// OrderIntent records an execution request before/while the broker order is live.
type OrderIntent struct {
	SignalID      uuid.UUID
	InstanceID    string
	Symbol        string
	Side          string
	Quantity      int
	OrderType     string
	LimitPrice    *float64
	StopPrice     *float64
	Status        string
	BrokerOrderID string
}

// OrderIntentSummary is a minimal view for idempotency checks.
type OrderIntentSummary struct {
	ID            string
	Status        string
	BrokerOrderID string
}

// NewService creates a new execution service.
// enforcer may be nil; when non-nil, portfolio-level policy gates (L16) are
// applied before every order submission.
func NewService(engine *Engine, broker BrokerClient, store TradeStore, orderType string, riskParams RiskParameters, enforcer *risk.Enforcer) *Service {
	return &Service{
		engine:     engine,
		broker:     broker,
		store:      store,
		orderType:  orderType,
		riskParams: riskParams,
		enforcer:   enforcer,
	}
}

func (s *Service) WithAudit(auditSvc *audit.Service) *Service {
	s.audit = auditSvc
	return s
}

// ExecuteTrade executes a trade for an approved signal
func (s *Service) ExecuteTrade(ctx context.Context, signalID uuid.UUID, approvedBy string) (*TradeResult, error) {
	if existing, err := s.store.GetTradeBySignal(ctx, signalID); err == nil && existing != nil {
		return existing, nil
	}

	// 1. Fetch signal from database
	signal, err := s.store.GetSignal(ctx, signalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal: %w", err)
	}

	// Codex v1: per-instance guardrails + flatten-by-close enforcement.
	if signal.InstanceID != "" {
		if checker, ok := s.store.(interface {
			CheckInstanceExecutionAllowed(context.Context, string) error
		}); ok {
			if err := checker.CheckInstanceExecutionAllowed(ctx, signal.InstanceID); err != nil {
				return nil, fmt.Errorf("instance guardrail blocked execution: %w", err)
			}
		}
	}

	// 2. Validate signal
	if err := s.engine.ValidateSignal(*signal); err != nil {
		return nil, fmt.Errorf("signal validation failed: %w", err)
	}

	// 3. Get account info from broker
	brokerAccount, err := s.broker.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	accountInfo := AccountInfo{
		NetLiquidation: brokerAccount.NetLiquidation,
		BuyingPower:    brokerAccount.BuyingPower,
		Currency:       brokerAccount.Currency,
	}

	// 4. Check risk gates
	if err := s.checkRiskGates(ctx); err != nil {
		return nil, err
	}

	// 5. Calculate position size
	quantity, err := s.engine.CalculatePositionSize(*signal, accountInfo)
	if err != nil {
		return nil, fmt.Errorf("position size calculation failed: %w", err)
	}

	// 6. Create order request
	orderReq := s.engine.CreateOrderRequest(*signal, quantity, s.orderType)

	// Convert to broker order request
	brokerOrderReq := &BrokerOrderRequest{
		Symbol:     orderReq.Symbol,
		Action:     orderReq.Action,
		Quantity:   orderReq.Quantity,
		OrderType:  orderReq.OrderType,
		LimitPrice: orderReq.LimitPrice,
		StopPrice:  orderReq.StopPrice,
	}

	// 6b. Idempotency guard: block duplicate broker submissions if an intent already exists.
	if intent, err := s.store.GetLatestOrderIntentBySignal(ctx, signalID); err == nil && intent != nil {
		if intent.BrokerOrderID != "" && !isTerminalOrderStatus(intent.Status) {
			if s.audit != nil {
				flowID := observability.FlowIDFromContext(ctx)
				_ = s.audit.LogAuditEvent(ctx, flowID, "execution", "idempotency_guard", "blocked",
					"duplicate broker submission blocked",
					map[string]any{
						"signal_id":       signalID.String(),
						"intent_id":       intent.ID,
						"intent_status":   intent.Status,
						"broker_order_id": intent.BrokerOrderID,
					})
			}
			return nil, fmt.Errorf("order already submitted for signal %s (intent %s, status %s)", signalID, intent.ID, intent.Status)
		}
	}

	// 7. Place order with broker
	orderResp, err := s.broker.PlaceOrder(ctx, brokerOrderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	if !orderResp.Success {
		return nil, fmt.Errorf("order rejected: %s", orderResp.Message)
	}

	// 8. Get initial order status
	orderStatus, _ := s.broker.GetOrderStatus(ctx, orderResp.OrderID)

	// 8b. Record order intent
	intentStatus := "submitted"
	if orderStatus != nil && orderStatus.Status != "" {
		intentStatus = orderStatus.Status
	}
	intentID, err := s.store.StoreOrderIntent(ctx, &OrderIntent{
		SignalID:      signalID,
		InstanceID:    signal.InstanceID,
		Symbol:        signal.Symbol,
		Side:          orderReq.Action,
		Quantity:      orderReq.Quantity,
		OrderType:     orderReq.OrderType,
		LimitPrice:    orderReq.LimitPrice,
		StopPrice:     orderReq.StopPrice,
		Status:        intentStatus,
		BrokerOrderID: fmt.Sprintf("%d", orderResp.OrderID),
	})
	if err != nil {
		log.Printf("Warning: Failed to store order intent: %v", err)
	}

	// 9. Calculate risk metrics
	riskAmount := float64(quantity) * math.Abs(signal.EntryPrice-signal.StopLoss)
	riskPercent := riskAmount / accountInfo.NetLiquidation
	positionValue := float64(quantity) * signal.EntryPrice

	// 10. Create trade record
	tradeID := uuid.New()
	trade := &TradeResult{
		TradeID:       tradeID,
		SignalID:      signalID,
		OrderID:       fmt.Sprintf("%d", orderResp.OrderID),
		Symbol:        signal.Symbol,
		Direction:     signal.SignalType,
		Quantity:      quantity,
		EntryPrice:    signal.EntryPrice,
		StopLoss:      signal.StopLoss,
		TakeProfit:    signal.TakeProfit,
		StrategyID:    signal.StrategyID,
		InstanceID:    signal.InstanceID,
		Status:        "submitted",
		RiskAmount:    riskAmount,
		RiskPercent:   riskPercent,
		PositionValue: positionValue,
		SubmittedAt:   time.Now().UTC(),
		ArtifactID:    signal.ArtifactID,   // ADR-0012 Phase 4: Propagate artifact tracking
		ArtifactHash:  signal.ArtifactHash, // ADR-0012 Phase 4: Immutability guarantee
	}

	if orderStatus != nil {
		trade.Status = orderStatus.Status
		trade.FilledQty = orderStatus.FilledQty
		trade.AvgFillPrice = orderStatus.AvgFillPrice
	}

	if intentID == "" {
		log.Printf("Warning: order intent ID missing for trade %s", tradeID)
	}

	// 11. Store trade in database
	if err := s.store.StoreTrade(ctx, trade); err != nil {
		log.Printf("Warning: Failed to store trade record: %v", err)
		// Don't fail - order was placed successfully
	}

	// 12. Update trade approval with order ID
	if err := s.store.UpdateTradeApproval(ctx, signalID, trade.OrderID); err != nil {
		log.Printf("Warning: Failed to update trade approval: %v", err)
	}

	// 12b. Update order intent status (if available)
	if orderStatus != nil {
		if err := s.store.UpdateOrderIntentStatus(ctx, signalID, orderStatus); err != nil {
			log.Printf("Warning: Failed to update order intent status: %v", err)
		}
	}

	// 13. Poll for status updates in background
	go s.pollOrderStatus(trade.TradeID, orderResp.OrderID)

	return trade, nil
}

// checkRiskGates verifies risk constraints before execution
func (s *Service) checkRiskGates(ctx context.Context) error {
	// Fetch current positions (needed by both engine and enforcer checks)
	var openPositions int
	if s.riskParams.MaxOpenPositions > 0 || s.enforcer != nil {
		positions, err := s.broker.GetPositions(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch positions for risk gate: %w", err)
		}
		for _, pos := range positions.Positions {
			if pos.Quantity != 0 {
				openPositions++
			}
		}
		// Engine-level max-positions check (backwards-compat with RiskParameters)
		if s.riskParams.MaxOpenPositions > 0 && openPositions >= s.riskParams.MaxOpenPositions {
			return fmt.Errorf("risk gate: open positions %d exceeds max %d", openPositions, s.riskParams.MaxOpenPositions)
		}
	}

	// Fetch daily risk (needed by both engine and enforcer checks)
	var dailyRiskDollar float64
	if s.riskParams.MaxDailyLoss > 0 || s.enforcer != nil {
		totalRisk, err := s.store.GetDailyRisk(ctx)
		if err != nil {
			return fmt.Errorf("failed to calculate daily risk: %w", err)
		}
		dailyRiskDollar = totalRisk
		// Engine-level dollar daily-loss check
		if s.riskParams.MaxDailyLoss > 0 && totalRisk >= s.riskParams.MaxDailyLoss {
			return fmt.Errorf("risk gate: daily risk %.2f exceeds max %.2f", totalRisk, s.riskParams.MaxDailyLoss)
		}
	}

	// L16: Policy-level portfolio gate (versioned, logged with violation codes)
	if s.enforcer != nil {
		account, err := s.broker.GetAccount(ctx)
		netLiq := 0.0
		if err == nil {
			netLiq = account.NetLiquidation
		}
		state := risk.PortfolioState{
			NetLiquidation:  netLiq,
			OpenPositions:   openPositions,
			DailyLossDollar: dailyRiskDollar,
			// CurrentDrawdown: supplied by L08 edge-stability monitor once implemented
		}
		if vs := s.enforcer.CheckPortfolio(state); !vs.IsEmpty() {
			log.Printf("[RISK VIOLATION] policy=%s violations=%s",
				s.enforcer.Policy().Version, vs.Error())
			return fmt.Errorf("risk policy violation: %w", vs)
		}
	}

	return nil
}

// pollOrderStatus polls broker for order status updates
func (s *Service) pollOrderStatus(tradeID uuid.UUID, orderID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status, err := s.broker.GetOrderStatus(ctx, orderID)
			if err != nil {
				log.Printf("order status poll failed for %d: %v", orderID, err)
				continue
			}
			if err := s.store.UpdateOrderIntentStatusByTrade(context.Background(), tradeID, status); err != nil {
				log.Printf("failed to update order intent status for %s: %v", tradeID, err)
			}
			if err := s.store.UpdateTradeStatus(context.Background(), tradeID, status); err != nil {
				log.Printf("failed to update trade status for %s: %v", tradeID, err)
			}
			if status.Status == "Filled" || status.Status == "Cancelled" {
				return
			}
		}
	}
}

// PostgresTradeStore implements TradeStore interface
type PostgresTradeStore struct {
	db *sql.DB
}

// NewPostgresTradeStore creates a new PostgreSQL trade store
func NewPostgresTradeStore(db *sql.DB) *PostgresTradeStore {
	return &PostgresTradeStore{db: db}
}

// GetSignal retrieves an approved signal from database
func (s *PostgresTradeStore) GetSignal(ctx context.Context, signalID uuid.UUID) (*Signal, error) {
	// ADR-0012 Phase 4: Include artifact_id in signal retrieval
	query := `
		SELECT s.id, s.symbol, s.signal_type, s.entry_price, s.stop_loss, s.take_profit,
		       s.strategy_id, s.confidence, s.instance_id, s.artifact_id,
		       a.hash AS artifact_hash
		FROM strategy_signals s
		LEFT JOIN strategy_artifacts a ON s.artifact_id = a.id
		WHERE s.id = $1
		  AND s.status = 'approved'
		  AND s.artifact_id IS NOT NULL
		  AND EXISTS (
		      SELECT 1
		      FROM artifact_approvals ap
		      WHERE ap.artifact_id = s.artifact_id
		        AND ap.state IN ('APPROVED', 'ACTIVE')
		  )
	`

	var signal Signal
	var instanceIDPtr *uuid.UUID
	var artifactIDPtr *uuid.UUID
	var artifactHashPtr *string

	err := s.db.QueryRowContext(ctx, query, signalID).Scan(
		&signal.ID,
		&signal.Symbol,
		&signal.SignalType,
		&signal.EntryPrice,
		&signal.StopLoss,
		&signal.TakeProfit,
		&signal.StrategyID,
		&signal.Confidence,
		&instanceIDPtr,
		&artifactIDPtr,
		&artifactHashPtr,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found or not approved")
	}
	if err != nil {
		return nil, err
	}

	if instanceIDPtr != nil {
		signal.InstanceID = instanceIDPtr.String()
	}
	// Convert nullable artifact_id/hash to strings
	if artifactIDPtr != nil {
		signal.ArtifactID = artifactIDPtr.String()
	}
	if artifactHashPtr != nil {
		signal.ArtifactHash = *artifactHashPtr
	}

	return &signal, nil
}

// GetTradeBySignal returns the latest trade for a signal, if one exists.
func (s *PostgresTradeStore) GetTradeBySignal(ctx context.Context, signalID uuid.UUID) (*TradeResult, error) {
	var (
		idStr           string
		symbol          string
		direction       string
		entry           float64
		stop            float64
		targetsRaw      []byte
		strategyID      string
		statusRaw       sql.NullString
		filledQtyRaw    sql.NullInt64
		avgFillRaw      sql.NullFloat64
		createdAt       time.Time
		riskRaw         []byte
		instanceIDPtr   *uuid.UUID
		artifactIDPtr   *uuid.UUID
		artifactHashPtr *string
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, symbol, direction, entry, stop, targets, strategy_id,
		       order_status, filled_qty, avg_fill_price, created_at, risk,
		       instance_id, artifact_id, artifact_hash
		FROM trades
		WHERE signal_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, signalID).Scan(
		&idStr,
		&symbol,
		&direction,
		&entry,
		&stop,
		&targetsRaw,
		&strategyID,
		&statusRaw,
		&filledQtyRaw,
		&avgFillRaw,
		&createdAt,
		&riskRaw,
		&instanceIDPtr,
		&artifactIDPtr,
		&artifactHashPtr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	tradeID, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid trade id %q: %w", idStr, err)
	}

	var targets []float64
	if len(targetsRaw) > 0 {
		_ = json.Unmarshal(targetsRaw, &targets)
	}
	if len(targets) == 0 {
		targets = []float64{0}
	}

	var (
		quantity      int
		orderID       string
		riskAmount    float64
		riskPercent   float64
		positionValue float64
	)
	if len(riskRaw) > 0 {
		var riskMap map[string]any
		if json.Unmarshal(riskRaw, &riskMap) == nil {
			quantity = intFromAny(riskMap["quantity"], 0)
			orderID, _ = riskMap["order_id"].(string)
			riskAmount = floatFromAny(riskMap["amount"], 0)
			riskPercent = floatFromAny(riskMap["percent"], 0)
			positionValue = floatFromAny(riskMap["position_value"], 0)
		}
	}

	trade := &TradeResult{
		TradeID:       tradeID,
		SignalID:      signalID,
		OrderID:       orderID,
		Symbol:        symbol,
		Direction:     direction,
		Quantity:      quantity,
		EntryPrice:    entry,
		StopLoss:      stop,
		TakeProfit:    targets[0],
		StrategyID:    strategyID,
		Status:        statusRaw.String,
		FilledQty:     int(filledQtyRaw.Int64),
		AvgFillPrice:  avgFillRaw.Float64,
		RiskAmount:    riskAmount,
		RiskPercent:   riskPercent,
		PositionValue: positionValue,
		SubmittedAt:   createdAt,
	}

	if instanceIDPtr != nil {
		trade.InstanceID = instanceIDPtr.String()
	}
	if artifactIDPtr != nil {
		trade.ArtifactID = artifactIDPtr.String()
	}
	if artifactHashPtr != nil {
		trade.ArtifactHash = *artifactHashPtr
	}

	return trade, nil
}

// GetLatestOrderIntentBySignal fetches the most recent order intent for a signal.
func (s *PostgresTradeStore) GetLatestOrderIntentBySignal(ctx context.Context, signalID uuid.UUID) (*OrderIntentSummary, error) {
	var (
		intentID      string
		status        sql.NullString
		brokerOrderID sql.NullString
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id::text, status, metadata->>'brokerOrderId'
		FROM order_intents
		WHERE signal_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, signalID).Scan(&intentID, &status, &brokerOrderID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &OrderIntentSummary{
		ID:            intentID,
		Status:        status.String,
		BrokerOrderID: brokerOrderID.String,
	}, nil
}

// StoreTrade stores a trade record in database
func (s *PostgresTradeStore) StoreTrade(ctx context.Context, trade *TradeResult) error {
	// ADR-0012 Phase 4: Include artifact_id and artifact_hash
	query := `
		INSERT INTO trades (id, signal_id, instance_id, symbol, direction, entry, stop, targets, strategy_id, notes, risk, order_status, filled_qty, avg_fill_price, created_at, artifact_id, artifact_hash)
		VALUES ($1, $2, NULLIF($3,'')::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	targetsJSON, _ := json.Marshal([]float64{trade.TakeProfit})
	riskJSON, _ := json.Marshal(map[string]interface{}{
		"amount":         trade.RiskAmount,
		"percent":        trade.RiskPercent,
		"position_value": trade.PositionValue,
		"quantity":       trade.Quantity,
		"order_id":       trade.OrderID,
		"status":         trade.Status,
	})

	notes := fmt.Sprintf("Signal ID: %s, Order ID: %s, Approved trade execution at %s",
		trade.SignalID, trade.OrderID, trade.SubmittedAt.Format(time.RFC3339))

	// Convert artifact_id to UUID pointer (nullable)
	var artifactIDPtr *uuid.UUID
	if trade.ArtifactID != "" {
		if artifactUUID, err := uuid.Parse(trade.ArtifactID); err == nil {
			artifactIDPtr = &artifactUUID
		}
	}

	// artifact_hash is nullable string
	var artifactHashPtr *string
	if trade.ArtifactHash != "" {
		artifactHashPtr = &trade.ArtifactHash
	}

	_, err := s.db.ExecContext(ctx, query,
		trade.TradeID.String(),
		trade.SignalID.String(),
		trade.InstanceID,
		trade.Symbol,
		trade.Direction,
		trade.EntryPrice,
		trade.StopLoss,
		targetsJSON,
		trade.StrategyID,
		notes,
		riskJSON,
		trade.Status,
		trade.FilledQty,
		trade.AvgFillPrice,
		trade.SubmittedAt,
		artifactIDPtr,
		artifactHashPtr,
	)

	return err
}

// UpdateTradeApproval updates trade approval with order ID
func (s *PostgresTradeStore) UpdateTradeApproval(ctx context.Context, signalID uuid.UUID, orderID string) error {
	query := `
		UPDATE trade_approvals
		SET order_id = $1
		WHERE signal_id = $2
	`

	_, err := s.db.ExecContext(ctx, query, orderID, signalID)
	return err
}

// UpdateTradeStatus updates trade status from broker
func (s *PostgresTradeStore) UpdateTradeStatus(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error {
	query := `
		UPDATE trades
		SET order_status = $1,
		    filled_qty = $2,
		    avg_fill_price = $3
		WHERE id = $4
	`
	_, err := s.db.ExecContext(ctx, query, status.Status, status.FilledQty, status.AvgFillPrice, tradeID.String())
	if err == nil {
		_ = s.updateIntentAndFill(ctx, tradeID, status)
	}
	return err
}

// GetDailyRisk calculates total risk for today
func (s *PostgresTradeStore) GetDailyRisk(ctx context.Context) (float64, error) {
	query := `
		SELECT COALESCE(SUM((risk->>'amount')::numeric), 0)
		FROM trades
		WHERE created_at >= date_trunc('day', NOW())
	`
	var totalRisk float64
	err := s.db.QueryRowContext(ctx, query).Scan(&totalRisk)
	return totalRisk, err
}

// StoreOrderIntent writes to order_intents for the current signal/order.
func (s *PostgresTradeStore) StoreOrderIntent(ctx context.Context, intent *OrderIntent) (string, error) {
	if intent == nil {
		return "", fmt.Errorf("order intent is nil")
	}
	if intent.BrokerOrderID != "" {
		var existingID string
		err := s.db.QueryRowContext(ctx, `
			SELECT id::text
			FROM order_intents
			WHERE signal_id = $1
			  AND metadata->>'brokerOrderId' = $2
			ORDER BY created_at DESC
			LIMIT 1
		`, intent.SignalID, intent.BrokerOrderID).Scan(&existingID)
		if err == nil {
			return existingID, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	metadata, _ := json.Marshal(map[string]any{
		"brokerOrderId": intent.BrokerOrderID,
	})
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO order_intents (
			instance_id, signal_id, flow_id, symbol, side, quantity, order_type,
			limit_price, stop_price, status, metadata
		)
		VALUES (
			NULLIF($1,'')::uuid, $2, NULL, $3, $4, $5, $6,
			$7, $8, $9, $10::jsonb
		)
		RETURNING id::text
	`, intent.InstanceID, intent.SignalID, intent.Symbol, intent.Side, intent.Quantity, intent.OrderType,
		intent.LimitPrice, intent.StopPrice, intent.Status, string(metadata)).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateOrderIntentStatus updates the most recent intent for a signal.
func (s *PostgresTradeStore) UpdateOrderIntentStatus(ctx context.Context, signalID uuid.UUID, status *BrokerOrderStatus) error {
	if status == nil {
		return nil
	}
	metadata, _ := json.Marshal(map[string]any{
		"filledQty":    status.FilledQty,
		"avgFillPrice": status.AvgFillPrice,
	})
	_, err := s.db.ExecContext(ctx, `
		UPDATE order_intents
		SET status = $2,
		    metadata = metadata || $3::jsonb,
		    updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM order_intents
			WHERE signal_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		)
	`, signalID, status.Status, string(metadata))
	return err
}

// UpdateOrderIntentStatusByTrade updates the most recent intent using a trade lookup.
func (s *PostgresTradeStore) UpdateOrderIntentStatusByTrade(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error {
	if status == nil {
		return nil
	}
	var signalID uuid.UUID
	if err := s.db.QueryRowContext(ctx, `
		SELECT signal_id
		FROM trades
		WHERE id = $1
	`, tradeID).Scan(&signalID); err != nil {
		return err
	}
	return s.UpdateOrderIntentStatus(ctx, signalID, status)
}

func (s *PostgresTradeStore) updateIntentAndFill(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error {
	if status == nil {
		return nil
	}
	var (
		signalID uuid.UUID
		symbol   string
		side     string
	)
	if err := s.db.QueryRowContext(ctx, `
		SELECT signal_id, symbol, direction
		FROM trades
		WHERE id = $1
	`, tradeID).Scan(&signalID, &symbol, &side); err != nil {
		return err
	}
	_ = s.UpdateOrderIntentStatus(ctx, signalID, status)

	if status.FilledQty <= 0 && status.AvgFillPrice <= 0 {
		return nil
	}
	// Insert fill if not already recorded.
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO fills (
			intent_id, trade_id, broker_order_id, symbol, side, filled_qty, avg_fill_price, status, metadata
		)
		SELECT
			oi.id, $1, $2, $3, $4, $5, $6, $7, $8::jsonb
		FROM order_intents oi
		WHERE oi.signal_id = $9
		AND NOT EXISTS (
			SELECT 1 FROM fills f
			WHERE f.trade_id = $1
			  AND f.broker_order_id = $2
		)
		ORDER BY oi.created_at DESC
		LIMIT 1
	`, tradeID.String(), fmt.Sprintf("%d", status.OrderID), symbol, side, status.FilledQty, status.AvgFillPrice, status.Status,
		`{}`, signalID)
	return err
}

// CheckInstanceExecutionAllowed enforces per-instance limits and flatten window.
func (s *PostgresTradeStore) CheckInstanceExecutionAllowed(ctx context.Context, instanceID string) error {
	var (
		enabled    bool
		tz         string
		flattenAt  string
		configText string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, session_timezone, flatten_by_close_time, config::text
		FROM strategy_instances
		WHERE id = $1::uuid
	`, instanceID).Scan(&enabled, &tz, &flattenAt, &configText)
	if err != nil {
		return fmt.Errorf("load instance %s: %w", instanceID, err)
	}
	if !enabled {
		return fmt.Errorf("instance is disabled")
	}
	if tz == "" {
		tz = "America/New_York"
	}
	if flattenAt == "" {
		flattenAt = "15:55"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	var hh, mm int
	if _, err := fmt.Sscanf(flattenAt, "%d:%d", &hh, &mm); err == nil {
		flattenTs := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, loc)
		if !now.Before(flattenTs) {
			return fmt.Errorf("post-flatten execution rejected at %s", flattenAt)
		}
	}

	cfg := map[string]any{}
	if configText != "" {
		_ = json.Unmarshal([]byte(configText), &cfg)
	}
	maxTrades := intFromAny(cfg["maxTradesPerDay"], 1000)
	maxOpenPositions := intFromAny(cfg["maxOpenPositions"], 1000)
	maxDailyLoss := floatFromAny(cfg["maxDailyLoss"], 0)

	var tradesToday int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM trades
		WHERE instance_id = $1::uuid
		  AND created_at >= date_trunc('day', NOW())
	`, instanceID).Scan(&tradesToday)
	if err == nil && maxTrades > 0 && tradesToday >= maxTrades {
		return fmt.Errorf("max trades/day reached (%d)", maxTrades)
	}

	var openPositions int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT symbol)
		FROM trades
		WHERE instance_id = $1::uuid
		  AND COALESCE(order_status, '') NOT IN ('Filled', 'Cancelled')
		  AND created_at >= NOW() - INTERVAL '1 day'
	`, instanceID).Scan(&openPositions)
	if err == nil && maxOpenPositions > 0 && openPositions >= maxOpenPositions {
		return fmt.Errorf("max open positions reached (%d)", maxOpenPositions)
	}

	if maxDailyLoss > 0 {
		var riskAtStake float64
		_ = s.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM((risk->>'amount')::numeric), 0)
			FROM trades
			WHERE instance_id = $1::uuid
			  AND created_at >= date_trunc('day', NOW())
		`, instanceID).Scan(&riskAtStake)
		if riskAtStake >= maxDailyLoss {
			return fmt.Errorf("max daily loss/risk reached (%.2f)", maxDailyLoss)
		}
	}
	return nil
}

func intFromAny(v any, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return def
	}
}

func floatFromAny(v any, def float64) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return def
	}
}

func isTerminalOrderStatus(status string) bool {
	switch status {
	case "Filled", "Cancelled", "Rejected":
		return true
	default:
		return false
	}
}
