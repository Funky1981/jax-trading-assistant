package chattools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HandlerFunc func(context.Context, *pgxpool.Pool, json.RawMessage) (json.RawMessage, error)

type ToolSpec struct {
	Name                 string
	Description          string
	ArgKey               string
	ArgLabel             string
	ReadOnly             bool
	EvidenceLevel        string
	FreshnessExpectation string
	Handler              HandlerFunc
}

var defaultTools = []ToolSpec{
	{
		Name:                 "get_candidate_trade",
		Description:          "Retrieve a candidate trade by ID",
		ArgKey:               "candidateId",
		ArgLabel:             "Candidate ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              GetCandidateTrade,
	},
	{
		Name:                 "get_signal",
		Description:          "Retrieve a strategy signal by ID",
		ArgKey:               "signalId",
		ArgLabel:             "Signal ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              GetSignal,
	},
	{
		Name:                 "get_trade",
		Description:          "Retrieve an executed trade by ID",
		ArgKey:               "tradeId",
		ArgLabel:             "Trade ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "historical_snapshot",
		Handler:              GetTrade,
	},
	{
		Name:                 "get_strategy",
		Description:          "Retrieve strategy definition by ID",
		ArgKey:               "strategyId",
		ArgLabel:             "Strategy ID",
		ReadOnly:             true,
		EvidenceLevel:        "weak_inference",
		FreshnessExpectation: "reference_data",
		Handler:              GetStrategy,
	},
	{
		Name:                 "get_strategy_instance",
		Description:          "Retrieve strategy instance by ID",
		ArgKey:               "instanceId",
		ArgLabel:             "Instance ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "configuration_snapshot",
		Handler:              GetStrategyInstance,
	},
	{
		Name:                 "get_orchestration_run",
		Description:          "Retrieve an orchestration run by ID",
		ArgKey:               "runId",
		ArgLabel:             "Run ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "historical_snapshot",
		Handler:              GetOrchestrationRun,
	},
	{
		Name:                 "search_research_runs",
		Description:          "Search recent orchestration/research runs",
		ArgKey:               "symbol",
		ArgLabel:             "Symbol (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "historical_snapshot",
		Handler:              SearchResearchRuns,
	},
	{
		Name:                 "explain_trade_blockers",
		Description:          "Explain why a candidate was blocked",
		ArgKey:               "candidateId",
		ArgLabel:             "Candidate ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              ExplainTradeBlockers,
	},
	{
		Name:                 "list_pending_approvals",
		Description:          "List candidates currently awaiting approval",
		ArgKey:               "limit",
		ArgLabel:             "Limit (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              ListPendingApprovals,
	},
	{
		Name:                 "list_recent_blocked_candidates",
		Description:          "List recently blocked candidates",
		ArgKey:               "limit",
		ArgLabel:             "Limit (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              ListRecentBlockedCandidates,
	},
	{
		Name:                 "search_candidates",
		Description:          "Search recent candidates by symbol or status",
		ArgKey:               "query",
		ArgLabel:             "Symbol or status",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              SearchCandidates,
	},
	{
		Name:                 "query_knowledge",
		Description:          "Search local knowledge markdown when configured",
		ArgKey:               "query",
		ArgLabel:             "Knowledge query",
		ReadOnly:             true,
		EvidenceLevel:        "derived_internal_data",
		FreshnessExpectation: "local_docs_snapshot",
		Handler:              QueryKnowledge,
	},
	{
		Name:                 "compare_runs",
		Description:          "Compare two or more run records by ID",
		ArgKey:               "runIds",
		ArgLabel:             "Run IDs (comma-separated)",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "historical_snapshot",
		Handler:              CompareRuns,
	},
	{
		Name:                 "strategy_instance_summary",
		Description:          "Summarise one strategy instance with recent signals, trades, and runs",
		ArgKey:               "instanceId",
		ArgLabel:             "Instance ID",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              StrategyInstanceSummary,
	},
	{
		Name:                 "blocked_candidate_analysis",
		Description:          "Aggregate recent blocked candidates by symbol and blocker code",
		ArgKey:               "symbol",
		ArgLabel:             "Symbol (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "hard_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              BlockedCandidateAnalysis,
	},
	{
		Name:                 "recent_research_narrative",
		Description:          "Summarise recent research and orchestration activity",
		ArgKey:               "symbol",
		ArgLabel:             "Symbol (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "derived_internal_data",
		FreshnessExpectation: "historical_snapshot",
		Handler:              RecentResearchNarrative,
	},
	{
		Name:                 "confidence_drift_summary",
		Description:          "Summarise recent confidence drift across signals and candidates",
		ArgKey:               "symbol",
		ArgLabel:             "Symbol (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "derived_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              ConfidenceDriftSummary,
	},
	{
		Name:                 "signal_clustering_overview",
		Description:          "Group recent signals into symbol and strategy clusters",
		ArgKey:               "symbol",
		ArgLabel:             "Symbol (optional)",
		ReadOnly:             true,
		EvidenceLevel:        "derived_internal_data",
		FreshnessExpectation: "near_real_time",
		Handler:              SignalClusteringOverview,
	},
}

func DefaultTools() []ToolSpec {
	out := make([]ToolSpec, len(defaultTools))
	copy(out, defaultTools)
	return out
}

func Lookup(name string) (ToolSpec, bool) {
	for _, tool := range defaultTools {
		if tool.Name == name {
			return tool, true
		}
	}
	return ToolSpec{}, false
}

func GetCandidateTrade(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		CandidateID string `json:"candidateId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.CandidateID == "" {
		return nil, fmt.Errorf("candidateId required")
	}
	id, err := uuid.Parse(p.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("invalid candidateId")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT ct.id::text, ct.signal_id::text, ct.strategy_id, ct.artifact_id::text, ct.symbol, ct.signal_type, ct.status, ct.confidence, ct.entry_price,
			       ct.stop_loss, ct.take_profit, ct.reasoning, ct.block_reason, ct.blocked_reason_code, ct.detected_at,
			       ei.id::text AS execution_instruction_id, ei.trade_id
			FROM candidate_trades ct
			LEFT JOIN LATERAL (
				SELECT id, trade_id
				FROM execution_instructions
				WHERE candidate_id = ct.id
				ORDER BY created_at DESC
				LIMIT 1
			) ei ON TRUE
			WHERE ct.id = $1) t`, id)
}

func GetSignal(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		SignalID string `json:"signalId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.SignalID == "" {
		return nil, fmt.Errorf("signalId required")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, strategy_id, signal_type, confidence, status, generated_at
			FROM strategy_signals WHERE id = $1::uuid) t`, p.SignalID)
}

func GetTrade(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		TradeID string `json:"tradeId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.TradeID == "" {
		return nil, fmt.Errorf("tradeId required")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, direction, quantity, entry_price, status, created_at
			FROM trades WHERE id = $1::uuid) t`, p.TradeID)
}

func GetStrategy(_ context.Context, _ *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		StrategyID string `json:"strategyId"`
	}
	_ = json.Unmarshal(args, &p)
	return marshalJSON(map[string]any{
		"strategyId": p.StrategyID,
		"note":       "strategy details available via /api/v1/strategies/{id}",
	})
}

func GetStrategyInstance(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		InstanceID string `json:"instanceId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.InstanceID == "" {
		return nil, fmt.Errorf("instanceId required")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, name, strategy_type_id, enabled, session_timezone, flatten_by_close_time
			FROM strategy_instances WHERE id = $1::uuid) t`, p.InstanceID)
}

func GetOrchestrationRun(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.RunID == "" {
		return nil, fmt.Errorf("runId required")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, symbol, trigger_type, status, started_at, completed_at
			FROM orchestration_runs WHERE id = $1::uuid) t`, p.RunID)
}

func SearchResearchRuns(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Symbol string `json:"symbol"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	rows, err := pool.Query(ctx,
		`SELECT id::text, symbol, trigger_type, status, started_at
		   FROM orchestration_runs
		  WHERE ($1 = '' OR symbol = $1)
		  ORDER BY started_at DESC LIMIT $2`, p.Symbol, p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, sym, trig, status string
		var startedAt any
		if err := rows.Scan(&id, &sym, &trig, &status, &startedAt); err != nil {
			continue
		}
		runs = append(runs, map[string]any{
			"id":          id,
			"symbol":      sym,
			"triggerType": trig,
			"status":      status,
			"startedAt":   startedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(runs)
}

func ExplainTradeBlockers(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		CandidateID string `json:"candidateId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || p.CandidateID == "" {
		return nil, fmt.Errorf("candidateId required")
	}
	return rowQueryJSON(ctx, pool,
		`SELECT row_to_json(t) FROM (
			SELECT id::text, status, block_reason, blocked_reason_code, reasoning, detected_at
			FROM candidate_trades WHERE id = $1::uuid) t`, p.CandidateID)
}

func ListPendingApprovals(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Limit int `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, status, confidence, detected_at, expires_at,
		       COALESCE(strategy_id,''), COALESCE(signal_id::text,''), COALESCE(artifact_id::text,'')
		FROM candidate_trades
		WHERE status = 'awaiting_approval'
		ORDER BY detected_at ASC
		LIMIT $1
	`, p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType, status, strategyID, signalID, artifactID string
		var confidence *float64
		var detectedAt time.Time
		var expiresAt *time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &status, &confidence, &detectedAt, &expiresAt, &strategyID, &signalID, &artifactID); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":         id,
			"symbol":     symbol,
			"signalType": signalType,
			"status":     status,
			"confidence": confidence,
			"detectedAt": detectedAt,
			"expiresAt":  expiresAt,
			"strategyId": strategyID,
			"signalId":   signalID,
			"artifactId": artifactID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(items)
}

func ListRecentBlockedCandidates(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Limit int `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, blocked_reason_code, block_reason, confidence, blocked_at, detected_at
		FROM candidate_trades
		WHERE status = 'blocked'
		ORDER BY COALESCE(blocked_at, detected_at) DESC
		LIMIT $1
	`, p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType string
		var reasonCode, reason *string
		var confidence *float64
		var blockedAt, detectedAt time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &reasonCode, &reason, &confidence, &blockedAt, &detectedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":                id,
			"symbol":            symbol,
			"signalType":        signalType,
			"blockedReasonCode": reasonCode,
			"blockReason":       reason,
			"confidence":        confidence,
			"blockedAt":         blockedAt,
			"detectedAt":        detectedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(items)
}

func SearchCandidates(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 25 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}
	rows, err := pool.Query(ctx, `
		SELECT id::text, symbol, signal_type, status, blocked_reason_code, detected_at
		FROM candidate_trades
		WHERE ($1 = '' OR symbol ILIKE '%' || $1 || '%' OR status ILIKE '%' || $1 || '%')
		ORDER BY detected_at DESC
		LIMIT $2
	`, strings.TrimSpace(p.Query), p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, symbol, signalType, status string
		var reasonCode *string
		var detectedAt time.Time
		if err := rows.Scan(&id, &symbol, &signalType, &status, &reasonCode, &detectedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":                id,
			"symbol":            symbol,
			"signalType":        signalType,
			"status":            status,
			"blockedReasonCode": reasonCode,
			"detectedAt":        detectedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(items)
}

func QueryKnowledge(_ context.Context, _ *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Query) == "" {
		return nil, fmt.Errorf("query required")
	}
	if p.Limit <= 0 || p.Limit > 10 {
		p.Limit = 5
	}

	root := os.Getenv("JAX_KNOWLEDGE_ROOT")
	if strings.TrimSpace(root) == "" {
		root = filepath.Join("knowledge", "md")
	}
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("knowledge root not configured")
	}

	query := strings.ToLower(strings.TrimSpace(p.Query))
	matches := make([]map[string]any, 0, p.Limit)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() || len(matches) >= p.Limit {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(raw)
		idx := strings.Index(strings.ToLower(text), query)
		if idx < 0 {
			return nil
		}
		start := idx - 120
		if start < 0 {
			start = 0
		}
		end := idx + len(query) + 160
		if end > len(text) {
			end = len(text)
		}
		matches = append(matches, map[string]any{
			"path":    path,
			"excerpt": strings.TrimSpace(text[start:end]),
		})
		return nil
	})
	return marshalJSON(matches)
}

func CompareRuns(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		RunIDs []string `json:"runIds"`
	}
	var rawMap map[string]any
	_ = json.Unmarshal(args, &rawMap)
	_ = json.Unmarshal(args, &p)
	if len(p.RunIDs) == 0 {
		switch value := rawMap["runIds"].(type) {
		case string:
			for _, part := range strings.Split(value, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					p.RunIDs = append(p.RunIDs, part)
				}
			}
		case []any:
			for _, item := range value {
				if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
					p.RunIDs = append(p.RunIDs, strings.TrimSpace(text))
				}
			}
		}
	}
	if len(p.RunIDs) < 2 {
		return nil, fmt.Errorf("at least two runIds required")
	}
	if len(p.RunIDs) > 5 {
		p.RunIDs = p.RunIDs[:5]
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.run_type, r.status, COALESCE(r.flow_id, ''), COALESCE(r.source, ''), COALESCE(r.summary, '{}'::jsonb),
		       r.started_at, r.completed_at, COALESCE(r.error, ''), COALESCE(si.name, '')
		FROM runs r
		LEFT JOIN strategy_instances si ON si.id = r.instance_id
		WHERE r.id::text = ANY($1)
		ORDER BY r.created_at DESC
	`, p.RunIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, len(p.RunIDs))
	for rows.Next() {
		var id, runType, status, flowID, source, errText, instanceName string
		var summary map[string]any
		var startedAt time.Time
		var completedAt *time.Time
		if err := rows.Scan(&id, &runType, &status, &flowID, &source, &summary, &startedAt, &completedAt, &errText, &instanceName); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":           id,
			"runType":      runType,
			"status":       status,
			"flowId":       flowID,
			"source":       source,
			"summary":      summary,
			"startedAt":    startedAt,
			"completedAt":  completedAt,
			"error":        errText,
			"instanceName": instanceName,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(map[string]any{
		"requestedRunIds": p.RunIDs,
		"runs":            items,
	})
}

func StrategyInstanceSummary(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		InstanceID string `json:"instanceId"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.InstanceID) == "" {
		return nil, fmt.Errorf("instanceId required")
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	var summary []byte
	err := pool.QueryRow(ctx, `
		WITH instance_core AS (
			SELECT si.id::text, si.name, si.strategy_type_id, si.strategy_id, si.enabled,
			       si.session_timezone, si.flatten_by_close_time, si.updated_at
			FROM strategy_instances si
			WHERE si.id = $1::uuid
		),
		signal_stats AS (
			SELECT COUNT(*)::int AS total_signals,
			       COUNT(*) FILTER (WHERE status = 'pending')::int AS pending_signals,
			       ROUND(AVG(confidence)::numeric, 4) AS avg_signal_confidence,
			       MAX(generated_at) AS last_signal_at
			FROM strategy_signals
			WHERE instance_id = $1::uuid
		),
		trade_stats AS (
			SELECT COUNT(*)::int AS total_trades,
			       COUNT(*) FILTER (WHERE status = 'filled')::int AS filled_trades,
			       MAX(created_at) AS last_trade_at
			FROM trades
			WHERE instance_id = $1::uuid
		),
		run_stats AS (
			SELECT COUNT(*)::int AS total_runs,
			       COUNT(*) FILTER (WHERE status = 'completed')::int AS completed_runs,
			       MAX(created_at) AS last_run_at
			FROM runs
			WHERE instance_id = $1::uuid
		)
		SELECT row_to_json(t)
		FROM (
			SELECT instance_core.*,
			       signal_stats.total_signals, signal_stats.pending_signals, signal_stats.avg_signal_confidence, signal_stats.last_signal_at,
			       trade_stats.total_trades, trade_stats.filled_trades, trade_stats.last_trade_at,
			       run_stats.total_runs, run_stats.completed_runs, run_stats.last_run_at
			FROM instance_core, signal_stats, trade_stats, run_stats
		) t
	`, p.InstanceID).Scan(&summary)
	if err != nil {
		return nil, fmt.Errorf("not found: %v", err)
	}
	return json.RawMessage(summary), nil
}

func BlockedCandidateAnalysis(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Symbol string `json:"symbol"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	rows, err := pool.Query(ctx, `
		SELECT symbol,
		       COALESCE(blocked_reason_code, 'unspecified') AS reason_code,
		       COUNT(*)::int AS blocked_count,
		       ROUND(AVG(confidence)::numeric, 4) AS avg_confidence,
		       MAX(COALESCE(blocked_at, detected_at)) AS latest_blocked_at
		FROM candidate_trades
		WHERE status = 'blocked'
		  AND ($1 = '' OR symbol = $1)
		GROUP BY symbol, COALESCE(blocked_reason_code, 'unspecified')
		ORDER BY blocked_count DESC, latest_blocked_at DESC
		LIMIT $2
	`, strings.TrimSpace(p.Symbol), p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var symbol, reasonCode string
		var blockedCount int
		var avgConfidence *float64
		var latestBlockedAt time.Time
		if err := rows.Scan(&symbol, &reasonCode, &blockedCount, &avgConfidence, &latestBlockedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"symbol":          symbol,
			"reasonCode":      reasonCode,
			"blockedCount":    blockedCount,
			"avgConfidence":   avgConfidence,
			"latestBlockedAt": latestBlockedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(items)
}

func RecentResearchNarrative(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Symbol string `json:"symbol"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 10 {
		p.Limit = 5
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.run_type, r.status, COALESCE(r.source, ''), COALESCE(si.name, ''),
		       r.created_at, COALESCE(r.summary, '{}'::jsonb)
		FROM runs r
		LEFT JOIN strategy_instances si ON si.id = r.instance_id
		LEFT JOIN orchestration_runs orun ON orun.id = r.orchestration_run_id
		WHERE ($1 = '' OR orun.symbol = $1 OR (r.summary->>'symbol') = $1)
		ORDER BY r.created_at DESC
		LIMIT $2
	`, strings.TrimSpace(p.Symbol), p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var id, runType, status, source, instanceName string
		var createdAt time.Time
		var summary map[string]any
		if err := rows.Scan(&id, &runType, &status, &source, &instanceName, &createdAt, &summary); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":           id,
			"runType":      runType,
			"status":       status,
			"source":       source,
			"instanceName": instanceName,
			"createdAt":    createdAt,
			"summary":      summary,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(map[string]any{
		"symbol":     strings.TrimSpace(p.Symbol),
		"recentRuns": items,
	})
}

func ConfidenceDriftSummary(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Symbol string `json:"symbol"`
	}
	_ = json.Unmarshal(args, &p)
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	var result []byte
	err := pool.QueryRow(ctx, `
		WITH signal_window AS (
			SELECT confidence::float8 AS confidence, generated_at
			FROM strategy_signals
			WHERE ($1 = '' OR symbol = $1)
			ORDER BY generated_at DESC
			LIMIT 20
		),
		candidate_window AS (
			SELECT confidence::float8 AS confidence, detected_at
			FROM candidate_trades
			WHERE ($1 = '' OR symbol = $1)
			  AND confidence IS NOT NULL
			ORDER BY detected_at DESC
			LIMIT 20
		)
		SELECT row_to_json(t)
		FROM (
			SELECT $1 AS symbol,
			       (SELECT COUNT(*)::int FROM signal_window) AS signalPoints,
			       (SELECT ROUND(AVG(confidence)::numeric, 4) FROM signal_window) AS avgSignalConfidence,
			       (SELECT ROUND(MAX(confidence)::numeric, 4) FROM signal_window) AS maxSignalConfidence,
			       (SELECT ROUND(MIN(confidence)::numeric, 4) FROM signal_window) AS minSignalConfidence,
			       (SELECT COUNT(*)::int FROM candidate_window) AS candidatePoints,
			       (SELECT ROUND(AVG(confidence)::numeric, 4) FROM candidate_window) AS avgCandidateConfidence,
			       (SELECT ROUND(MAX(confidence)::numeric, 4) FROM candidate_window) AS maxCandidateConfidence,
			       (SELECT ROUND(MIN(confidence)::numeric, 4) FROM candidate_window) AS minCandidateConfidence
		) t
	`, strings.TrimSpace(p.Symbol)).Scan(&result)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
}

func SignalClusteringOverview(ctx context.Context, pool *pgxpool.Pool, args json.RawMessage) (json.RawMessage, error) {
	var p struct {
		Symbol string `json:"symbol"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal(args, &p)
	if p.Limit <= 0 || p.Limit > 20 {
		p.Limit = 10
	}
	if pool == nil {
		return nil, fmt.Errorf("database unavailable")
	}

	rows, err := pool.Query(ctx, `
		SELECT symbol, strategy_id, signal_type,
		       COUNT(*)::int AS signal_count,
		       ROUND(AVG(confidence)::numeric, 4) AS avg_confidence,
		       MAX(generated_at) AS latest_generated_at
		FROM strategy_signals
		WHERE ($1 = '' OR symbol = $1)
		GROUP BY symbol, strategy_id, signal_type
		ORDER BY signal_count DESC, latest_generated_at DESC
		LIMIT $2
	`, strings.TrimSpace(p.Symbol), p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]any, 0, p.Limit)
	for rows.Next() {
		var symbol, strategyID, signalType string
		var signalCount int
		var avgConfidence *float64
		var latestGeneratedAt time.Time
		if err := rows.Scan(&symbol, &strategyID, &signalType, &signalCount, &avgConfidence, &latestGeneratedAt); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"symbol":            symbol,
			"strategyId":        strategyID,
			"signalType":        signalType,
			"signalCount":       signalCount,
			"avgConfidence":     avgConfidence,
			"latestGeneratedAt": latestGeneratedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return marshalJSON(items)
}
