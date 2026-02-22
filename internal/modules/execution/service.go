package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

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
	StoreTrade(ctx context.Context, trade *TradeResult) error
	UpdateTradeApproval(ctx context.Context, signalID uuid.UUID, orderID string) error
	UpdateTradeStatus(ctx context.Context, tradeID uuid.UUID, status *BrokerOrderStatus) error
	GetDailyRisk(ctx context.Context) (float64, error)
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

// ExecuteTrade executes a trade for an approved signal
func (s *Service) ExecuteTrade(ctx context.Context, signalID uuid.UUID, approvedBy string) (*TradeResult, error) {
	// 1. Fetch signal from database
	signal, err := s.store.GetSignal(ctx, signalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal: %w", err)
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

	// 11. Store trade in database
	if err := s.store.StoreTrade(ctx, trade); err != nil {
		log.Printf("Warning: Failed to store trade record: %v", err)
		// Don't fail - order was placed successfully
	}

	// 12. Update trade approval with order ID
	if err := s.store.UpdateTradeApproval(ctx, signalID, trade.OrderID); err != nil {
		log.Printf("Warning: Failed to update trade approval: %v", err)
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
		       s.strategy_id, s.confidence, s.artifact_id, 
		       a.hash AS artifact_hash
		FROM strategy_signals s
		LEFT JOIN strategy_artifacts a ON s.artifact_id = a.id
		WHERE s.id = $1 AND s.status = 'approved'
	`

	var signal Signal
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
		&artifactIDPtr,
		&artifactHashPtr,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found or not approved")
	}
	if err != nil {
		return nil, err
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

// StoreTrade stores a trade record in database
func (s *PostgresTradeStore) StoreTrade(ctx context.Context, trade *TradeResult) error {
	// ADR-0012 Phase 4: Include artifact_id and artifact_hash
	query := `
		INSERT INTO trades (id, signal_id, symbol, direction, entry, stop, targets, strategy_id, notes, risk, order_status, filled_qty, avg_fill_price, created_at, artifact_id, artifact_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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
