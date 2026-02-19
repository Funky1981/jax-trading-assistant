package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type OrchestrationV1Handler struct {
	db              *sql.DB
	orchestratorURL string
	httpClient      *http.Client
}

func (s *Server) RegisterOrchestrationV1(db *sql.DB, orchestratorURL string) {
	h := &OrchestrationV1Handler{
		db:              db,
		orchestratorURL: orchestratorURL,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
	s.mux.HandleFunc("/api/v1/orchestrate", s.protect(h.handleOrchestrate))
	s.mux.HandleFunc("/api/v1/orchestrate/runs", s.protect(h.handleRuns))
	s.mux.HandleFunc("/api/v1/orchestrate/runs/", s.protect(h.handleRun))
}

type orchestrationRequest struct {
	Bank            string                 `json:"bank"`
	Symbol          string                 `json:"symbol"`
	Strategy        string                 `json:"strategy,omitempty"`
	Constraints     map[string]interface{} `json:"constraints"`
	UserContext     string                 `json:"userContext"`
	Tags            []string               `json:"tags"`
	ResearchQueries []string               `json:"researchQueries"`
}

type orchestratorTriggerRequest struct {
	SignalID    string `json:"signal_id"`
	Symbol      string `json:"symbol"`
	TriggerType string `json:"trigger_type"`
	Context     string `json:"context"`
}

type orchestratorTriggerResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

func (h *OrchestrationV1Handler) handleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.db == nil {
		http.Error(w, "orchestration store not configured", http.StatusServiceUnavailable)
		return
	}

	var req orchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Symbol == "" {
		http.Error(w, "symbol is required", http.StatusBadRequest)
		return
	}

	contextStr := buildOrchestrationContext(req)
	trigger := orchestratorTriggerRequest{
		SignalID:    "",
		Symbol:      req.Symbol,
		TriggerType: "manual",
		Context:     contextStr,
	}

	if h.orchestratorURL == "" {
		h.orchestratorURL = getenvDefault("JAX_ORCHESTRATOR_URL", "http://jax-orchestrator:8091")
	}

	body, _ := json.Marshal(trigger)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, h.orchestratorURL+"/orchestrate", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		http.Error(w, "failed to trigger orchestration", http.StatusBadGateway)
		return
	}

	var orchResp orchestratorTriggerResponse
	if err := json.NewDecoder(resp.Body).Decode(&orchResp); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	result := map[string]interface{}{
		"plan": map[string]interface{}{
			"summary":        "Orchestration queued",
			"steps":          []string{},
			"action":         "PENDING",
			"confidence":     0.0,
			"reasoningNotes": "",
		},
		"tools":    []map[string]interface{}{},
		"runId":    orchResp.RunID,
		"status":   orchResp.Status,
		"duration": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *OrchestrationV1Handler) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.db == nil {
		http.Error(w, "orchestration store not configured", http.StatusServiceUnavailable)
		return
	}

	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}

	query := `
		SELECT id, symbol, status, started_at
		FROM orchestration_runs
		ORDER BY started_at DESC
		LIMIT $1
	`
	rows, err := h.db.QueryContext(r.Context(), query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var runs []map[string]interface{}
	for rows.Next() {
		var id, symbol, status string
		var startedAt time.Time
		if err := rows.Scan(&id, &symbol, &status, &startedAt); err != nil {
			continue
		}
		runs = append(runs, map[string]interface{}{
			"runId":     id,
			"symbol":    symbol,
			"timestamp": startedAt.UTC().Format(time.RFC3339),
			"success":   status == "completed",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runs)
}

func (h *OrchestrationV1Handler) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.db == nil {
		http.Error(w, "orchestration store not configured", http.StatusServiceUnavailable)
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/orchestrate/runs/")
	if runID == "" {
		http.Error(w, "run ID required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, symbol, status, started_at, completed_at, agent_suggestion, confidence, reasoning, agent_response
		FROM orchestration_runs
		WHERE id = $1
	`
	var (
		id, symbol, status string
		startedAt          time.Time
		completedAt        sql.NullTime
		agentSuggestion    sql.NullString
		confidence         sql.NullFloat64
		reasoning          sql.NullString
		agentResponse      []byte
	)
	if err := h.db.QueryRowContext(r.Context(), query, runID).Scan(
		&id, &symbol, &status, &startedAt, &completedAt, &agentSuggestion, &confidence, &reasoning, &agentResponse,
	); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "run not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	plan := map[string]interface{}{
		"summary":        agentSuggestion.String,
		"steps":          []string{},
		"action":         agentSuggestion.String,
		"confidence":     normalizeConfidence(confidence.Float64),
		"reasoningNotes": reasoning.String,
	}

	if len(agentResponse) > 0 {
		var resp map[string]interface{}
		if err := json.Unmarshal(agentResponse, &resp); err == nil {
			if v, ok := resp["action"].(string); ok && v != "" {
				plan["action"] = v
			}
			if v, ok := resp["reasoning"].(string); ok && v != "" {
				plan["summary"] = v
				plan["reasoningNotes"] = v
			}
			if v, ok := resp["key_factors"].([]interface{}); ok && len(v) > 0 {
				steps := make([]string, 0, len(v))
				for _, item := range v {
					if s, ok := item.(string); ok {
						steps = append(steps, s)
					}
				}
				plan["steps"] = steps
			}
			if v, ok := resp["confidence"].(float64); ok {
				plan["confidence"] = normalizeConfidence(v)
			}
		}
	}

	duration := 0.0
	if completedAt.Valid {
		duration = completedAt.Time.Sub(startedAt).Seconds()
	}

	result := map[string]interface{}{
		"plan":     plan,
		"tools":    []map[string]interface{}{},
		"runId":    id,
		"duration": duration,
		"status":   status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func buildOrchestrationContext(req orchestrationRequest) string {
	var sb strings.Builder
	if req.UserContext != "" {
		sb.WriteString(req.UserContext)
	}
	if req.Strategy != "" {
		sb.WriteString("\n\nStrategy: ")
		sb.WriteString(req.Strategy)
	}
	if len(req.Tags) > 0 {
		sb.WriteString("\n\nTags: ")
		sb.WriteString(strings.Join(req.Tags, ", "))
	}
	if len(req.ResearchQueries) > 0 {
		sb.WriteString("\n\nResearch queries:\n- ")
		sb.WriteString(strings.Join(req.ResearchQueries, "\n- "))
	}
	if len(req.Constraints) > 0 {
		encoded, _ := json.MarshalIndent(req.Constraints, "", "  ")
		sb.WriteString("\n\nConstraints:\n")
		sb.Write(encoded)
	}
	return sb.String()
}

func normalizeConfidence(v float64) float64 {
	if v > 1 {
		return v / 100
	}
	return v
}

func getenvDefault(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
