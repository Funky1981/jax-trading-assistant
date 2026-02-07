package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"jax-trading-assistant/services/jax-orchestrator/internal/app"

	"github.com/google/uuid"
)

// Server is the HTTP server for jax-orchestrator
type Server struct {
	orchestrator *app.Orchestrator
	db           *sql.DB
	mux          *http.ServeMux
}

// NewServer creates a new HTTP server
func NewServer(orchestrator *app.Orchestrator, db *sql.DB) *Server {
	s := &Server{
		orchestrator: orchestrator,
		db:           db,
		mux:          http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/orchestrate", s.handleOrchestrate)
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// OrchestrateRequest represents a request to trigger orchestration
type OrchestrateRequest struct {
	SignalID    string `json:"signal_id"`
	Symbol      string `json:"symbol"`
	TriggerType string `json:"trigger_type"`
	Context     string `json:"context"`
}

// OrchestrateResponse represents the orchestration response
type OrchestrateResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"service": "jax-orchestrator",
		"status":  "healthy",
	})
}

func (s *Server) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrchestrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}
	if req.TriggerType == "" {
		req.TriggerType = "manual"
	}

	// Parse signal ID if provided
	var signalID uuid.UUID
	if req.SignalID != "" {
		var err error
		signalID, err = uuid.Parse(req.SignalID)
		if err != nil {
			http.Error(w, "invalid signal_id", http.StatusBadRequest)
			return
		}
	}

	// Create orchestration run record
	runID := uuid.New()
	if err := s.createOrchestrationRun(r.Context(), runID, req.Symbol, req.TriggerType, signalID); err != nil {
		log.Printf("failed to create orchestration run: %v", err)
		http.Error(w, "failed to create orchestration run", http.StatusInternalServerError)
		return
	}

	// Fetch signal details and build enhanced context
	enhancedContext, err := s.buildEnhancedContext(r.Context(), req.Symbol, signalID, req.Context)
	if err != nil {
		log.Printf("failed to build enhanced context: %v", err)
		// Continue with provided context
		enhancedContext = req.Context
	}

	// Run orchestration asynchronously
	go s.runOrchestration(runID, req.Symbol, enhancedContext, signalID)

	// Return immediate response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(OrchestrateResponse{
		RunID:  runID.String(),
		Status: "running",
	})
}

func (s *Server) createOrchestrationRun(reqCtx context.Context, runID uuid.UUID, symbol, triggerType string, triggerID uuid.UUID) error {
	query := `
		INSERT INTO orchestration_runs (id, symbol, trigger_type, trigger_id, status, started_at)
		VALUES ($1, $2, $3, $4, 'running', NOW())
	`
	var triggerIDPtr *uuid.UUID
	if triggerID != uuid.Nil {
		triggerIDPtr = &triggerID
	}
	_, err := s.db.ExecContext(reqCtx, query, runID, symbol, triggerType, triggerIDPtr)
	return err
}

func (s *Server) buildEnhancedContext(reqCtx context.Context, symbol string, signalID uuid.UUID, baseContext string) (string, error) {
	var contextBuilder strings.Builder
	contextBuilder.WriteString(baseContext)

	// Fetch signal details if signal ID is provided
	if signalID != uuid.Nil {
		signalDetails, err := s.fetchSignalDetails(reqCtx, signalID)
		if err != nil {
			log.Printf("failed to fetch signal details: %v", err)
		} else {
			contextBuilder.WriteString(fmt.Sprintf("\n\nSignal Details (from database):\n%s", signalDetails))
		}
	}

	// TODO: Query jax-memory for similar past trades (future enhancement)
	// memories, err := s.recallMemories(ctx, symbol)

	return contextBuilder.String(), nil
}

func (s *Server) fetchSignalDetails(reqCtx context.Context, signalID uuid.UUID) (string, error) {
	query := `
		SELECT symbol, strategy_id, signal_type, confidence, entry_price, stop_loss, take_profit, reasoning
		FROM strategy_signals
		WHERE id = $1
	`
	var symbol, strategyID, signalType, reasoning string
	var confidence, entryPrice, stopLoss, takeProfit float64

	err := s.db.QueryRowContext(reqCtx, query, signalID).Scan(
		&symbol, &strategyID, &signalType, &confidence, &entryPrice, &stopLoss, &takeProfit, &reasoning,
	)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`Symbol: %s
Strategy: %s
Type: %s
Confidence: %.2f%%
Entry: $%.2f
Stop Loss: $%.2f
Take Profit: $%.2f
Reasoning: %s`,
		symbol, strategyID, signalType, confidence*100, entryPrice, stopLoss, takeProfit, reasoning,
	), nil
}

func (s *Server) runOrchestration(runID uuid.UUID, symbol, contextStr string, signalID uuid.UUID) {
	ctx := context.Background()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("orchestration run %s panicked: %v", runID, r)
			s.updateOrchestrationError(ctx, runID, fmt.Sprintf("panic: %v", r))
		}
	}()

	// Call Agent0 service
	agent0Response, err := s.callAgent0(ctx, symbol, contextStr)
	if err != nil {
		log.Printf("agent0 call failed for run %s: %v", runID, err)
		s.updateOrchestrationError(ctx, runID, err.Error())
		return
	}

	// Update orchestration run with results
	if err := s.updateOrchestrationComplete(ctx, runID, agent0Response); err != nil {
		log.Printf("failed to update orchestration run %s: %v", runID, err)
		return
	}

	log.Printf("orchestration run %s completed successfully", runID)
}

// Agent0Response represents the response from agent0-service
type Agent0Response struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

func (s *Server) callAgent0(ctx context.Context, symbol, contextStr string) (Agent0Response, error) {
	// TODO: Get agent0 URL from config
	agent0URL := "http://agent0-service:8093/suggest"

	requestBody := map[string]interface{}{
		"symbol":  symbol,
		"context": contextStr,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return Agent0Response{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agent0URL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return Agent0Response{}, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Agent0Response{}, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Agent0Response{}, fmt.Errorf("agent0 returned status %d", resp.StatusCode)
	}

	var result Agent0Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Agent0Response{}, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}

func (s *Server) updateOrchestrationComplete(ctx context.Context, runID uuid.UUID, response Agent0Response) error {
	query := `
		UPDATE orchestration_runs
		SET agent_suggestion = $1,
			confidence = $2,
			status = 'completed',
			completed_at = NOW()
		WHERE id = $3
	`
	_, err := s.db.ExecContext(ctx, query, response.Suggestion, response.Confidence, runID)
	return err
}

func (s *Server) updateOrchestrationError(ctx context.Context, runID uuid.UUID, errorMsg string) error {
	query := `
		UPDATE orchestration_runs
		SET error = $1,
			status = 'failed',
			completed_at = NOW()
		WHERE id = $2
	`
	_, err := s.db.ExecContext(ctx, query, errorMsg, runID)
	return err
}
