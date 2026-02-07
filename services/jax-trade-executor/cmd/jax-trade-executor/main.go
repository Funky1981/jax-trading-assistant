package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"jax-trading-assistant/libs/database"
	"jax-trading-assistant/libs/marketdata/ib"
	"jax-trading-assistant/libs/trading/executor"
)

type Config struct {
	Port           string
	PostgresDSN    string
	IBBridgeURL    string
	RiskPerTrade   float64
	MaxPositionPct float64
	OrderType      string // MKT, LMT, STP
}

func loadConfig() Config {
	return Config{
		Port:           getEnv("PORT", "8097"),
		PostgresDSN:    getEnv("POSTGRES_DSN", ""),
		IBBridgeURL:    getEnv("IB_BRIDGE_URL", "http://localhost:8092"),
		RiskPerTrade:   getEnvFloat("RISK_PER_TRADE", 0.01),   // 1% default
		MaxPositionPct: getEnvFloat("MAX_POSITION_PCT", 0.20), // 20% default
		OrderType:      getEnv("ORDER_TYPE", "LMT"),           // Limit orders by default
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		var f float64
		if _, err := fmt.Sscanf(value, "%f", &f); err == nil {
			return f
		}
	}
	return defaultValue
}

type Server struct {
	config   Config
	db       *sql.DB
	ibClient *ib.Client
	executor *executor.Executor
}

func main() {
	config := loadConfig()

	// Connect to database
	dbConfig := database.Config{
		DSN:             config.PostgresDSN,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
	}

	db, err := database.Connect(context.Background(), &dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create IB Bridge client
	ibClient := ib.NewClient(ib.Config{
		BaseURL: config.IBBridgeURL,
		Timeout: 30 * time.Second,
	})

	// Create executor with risk parameters
	exec := executor.NewExecutor(executor.RiskParameters{
		MaxRiskPerTrade:     config.RiskPerTrade,
		MinPositionSize:     1,
		MaxPositionSize:     0, // No hard limit
		MaxPositionValuePct: config.MaxPositionPct,
	})

	server := &Server{
		config:   config,
		db:       db.DB,
		ibClient: ibClient,
		executor: exec,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/api/v1/execute", server.handleExecute)
	mux.HandleFunc("/api/v1/trades", server.handleGetTrades)
	mux.HandleFunc("/api/v1/trades/", server.handleGetTrade)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    ":" + config.Port,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Trade Executor service starting on port %s", config.Port)
	log.Printf("IB Bridge: %s", config.IBBridgeURL)
	log.Printf("Risk per trade: %.2f%%", config.RiskPerTrade*100)
	log.Printf("Max position size: %.2f%%", config.MaxPositionPct*100)
	log.Printf("Order type: %s", config.OrderType)

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		http.Error(w, fmt.Sprintf("database unhealthy: %v", err), http.StatusServiceUnavailable)
		return
	}

	// Check IB Bridge
	_, err := s.ibClient.Health(ctx)
	ibHealthy := err == nil

	response := map[string]interface{}{
		"status":   "healthy",
		"service":  "jax-trade-executor",
		"database": "connected",
		"ib_bridge": map[string]interface{}{
			"connected": ibHealthy,
			"url":       s.config.IBBridgeURL,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ExecuteRequest struct {
	SignalID   uuid.UUID `json:"signal_id"`
	ApprovedBy string    `json:"approved_by"`
}

type ExecuteResponse struct {
	Success bool          `json:"success"`
	TradeID string        `json:"trade_id,omitempty"`
	OrderID string        `json:"order_id,omitempty"`
	Message string        `json:"message"`
	Trade   *TradeDetails `json:"trade,omitempty"`
	Error   string        `json:"error,omitempty"`
}

type TradeDetails struct {
	ID            string    `json:"id"`
	Symbol        string    `json:"symbol"`
	Direction     string    `json:"direction"`
	Quantity      int       `json:"quantity"`
	EntryPrice    float64   `json:"entry_price"`
	StopLoss      float64   `json:"stop_loss"`
	TakeProfit    float64   `json:"take_profit"`
	RiskAmount    float64   `json:"risk_amount"`
	RiskPercent   float64   `json:"risk_percent"`
	PositionValue float64   `json:"position_value"`
	RRRatio       float64   `json:"rr_ratio"`
	Status        string    `json:"status"`
	SubmittedAt   time.Time `json:"submitted_at"`
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ExecuteResponse{
			Success: false,
			Message: "invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if req.ApprovedBy == "" {
		respondJSON(w, http.StatusBadRequest, ExecuteResponse{
			Success: false,
			Message: "approved_by is required",
		})
		return
	}

	// Execute the trade
	result, err := s.executeTrade(r.Context(), req.SignalID, req.ApprovedBy)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, ExecuteResponse{
			Success: false,
			Message: "trade execution failed",
			Error:   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, ExecuteResponse{
		Success: true,
		TradeID: result.TradeID.String(),
		OrderID: result.OrderID,
		Message: fmt.Sprintf("Trade executed: %s %d shares of %s",
			result.Direction, result.Quantity, result.Symbol),
		Trade: &TradeDetails{
			ID:            result.TradeID.String(),
			Symbol:        result.Symbol,
			Direction:     result.Direction,
			Quantity:      result.Quantity,
			EntryPrice:    result.EntryPrice,
			StopLoss:      result.StopLoss,
			TakeProfit:    result.TakeProfit,
			RiskAmount:    result.RiskAmount,
			RiskPercent:   result.RiskPercent,
			PositionValue: result.PositionValue,
			RRRatio: s.executor.CalculateRiskRewardRatio(executor.Signal{
				EntryPrice: result.EntryPrice,
				StopLoss:   result.StopLoss,
				TakeProfit: result.TakeProfit,
			}),
			Status:      result.Status,
			SubmittedAt: result.SubmittedAt,
		},
	})
}

func (s *Server) executeTrade(ctx context.Context, signalID uuid.UUID, approvedBy string) (*executor.TradeResult, error) {
	// 1. Fetch signal from database
	signal, err := s.getSignal(ctx, signalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal: %w", err)
	}

	// 2. Validate signal
	if err := s.executor.ValidateSignal(*signal); err != nil {
		return nil, fmt.Errorf("signal validation failed: %w", err)
	}

	// 3. Get account info from IB
	account, err := s.ibClient.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	accountInfo := executor.AccountInfo{
		NetLiquidation: account.NetLiquidation,
		BuyingPower:    account.BuyingPower,
		Currency:       account.Currency,
	}

	// 4. Calculate position size
	quantity, err := s.executor.CalculatePositionSize(*signal, accountInfo)
	if err != nil {
		return nil, fmt.Errorf("position size calculation failed: %w", err)
	}

	// 5. Create order request
	orderReq := s.executor.CreateOrderRequest(*signal, quantity, s.config.OrderType)

	// Convert to IB order request
	ibOrderReq := &ib.OrderRequest{
		Symbol:     orderReq.Symbol,
		Action:     orderReq.Action,
		Quantity:   orderReq.Quantity,
		OrderType:  orderReq.OrderType,
		LimitPrice: orderReq.LimitPrice,
		StopPrice:  orderReq.StopPrice,
	}

	// 6. Place order with IB
	orderResp, err := s.ibClient.PlaceOrder(ctx, ibOrderReq)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	if !orderResp.Success {
		return nil, fmt.Errorf("order rejected: %s", orderResp.Message)
	}

	// 7. Create trade record
	tradeID := uuid.New()
	riskAmount := float64(quantity) * (signal.EntryPrice - signal.StopLoss)
	if signal.SignalType == "SELL" {
		riskAmount = float64(quantity) * (signal.StopLoss - signal.EntryPrice)
	}
	riskAmount = riskAmount * -1 // Make positive
	riskPercent := riskAmount / accountInfo.NetLiquidation
	positionValue := float64(quantity) * signal.EntryPrice

	trade := executor.TradeResult{
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
		Status:        "pending",
		RiskAmount:    riskAmount,
		RiskPercent:   riskPercent,
		PositionValue: positionValue,
		SubmittedAt:   time.Now().UTC(),
	}

	// 8. Store trade in database
	if err := s.storeTrade(ctx, &trade); err != nil {
		log.Printf("Warning: Failed to store trade record: %v", err)
		// Don't fail the request - order was placed successfully
	}

	// 9. Update trade approval with order ID
	if err := s.updateTradeApproval(ctx, signalID, fmt.Sprintf("%d", orderResp.OrderID)); err != nil {
		log.Printf("Warning: Failed to update trade approval: %v", err)
	}

	return &trade, nil
}

func (s *Server) getSignal(ctx context.Context, signalID uuid.UUID) (*executor.Signal, error) {
	query := `
		SELECT id, symbol, signal_type, entry_price, stop_loss, take_profit, 
		       strategy_id, confidence
		FROM strategy_signals
		WHERE id = $1 AND status = 'approved'
	`

	var signal executor.Signal
	err := s.db.QueryRowContext(ctx, query, signalID).Scan(
		&signal.ID,
		&signal.Symbol,
		&signal.SignalType,
		&signal.EntryPrice,
		&signal.StopLoss,
		&signal.TakeProfit,
		&signal.StrategyID,
		&signal.Confidence,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("signal not found or not approved")
	}
	if err != nil {
		return nil, err
	}

	return &signal, nil
}

func (s *Server) storeTrade(ctx context.Context, trade *executor.TradeResult) error {
	query := `
		INSERT INTO trades (id, symbol, direction, entry, stop, targets, strategy_id, notes, risk, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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

	_, err := s.db.ExecContext(ctx, query,
		trade.TradeID.String(),
		trade.Symbol,
		trade.Direction,
		trade.EntryPrice,
		trade.StopLoss,
		targetsJSON,
		trade.StrategyID,
		notes,
		riskJSON,
		trade.SubmittedAt,
	)

	return err
}

func (s *Server) updateTradeApproval(ctx context.Context, signalID uuid.UUID, orderID string) error {
	query := `
		UPDATE trade_approvals
		SET order_id = $1
		WHERE signal_id = $2
	`

	_, err := s.db.ExecContext(ctx, query, orderID, signalID)
	return err
}

func (s *Server) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := `
		SELECT id, symbol, direction, entry, stop, targets, strategy_id, notes, risk, created_at
		FROM trades
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var trades []map[string]interface{}
	for rows.Next() {
		var (
			id, symbol, direction, strategyID string
			entry, stop                       float64
			notes                             sql.NullString
			targets, risk                     []byte
			createdAt                         time.Time
		)

		if err := rows.Scan(&id, &symbol, &direction, &entry, &stop, &targets, &strategyID, &notes, &risk, &createdAt); err != nil {
			continue
		}

		trade := map[string]interface{}{
			"id":          id,
			"symbol":      symbol,
			"direction":   direction,
			"entry":       entry,
			"stop":        stop,
			"strategy_id": strategyID,
			"created_at":  createdAt,
		}

		if notes.Valid {
			trade["notes"] = notes.String
		}

		var targetsArray []float64
		json.Unmarshal(targets, &targetsArray)
		trade["targets"] = targetsArray

		var riskMap map[string]interface{}
		json.Unmarshal(risk, &riskMap)
		trade["risk"] = riskMap

		trades = append(trades, trade)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"trades": trades,
		"count":  len(trades),
	})
}

func (s *Server) handleGetTrade(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get single trade by ID
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
