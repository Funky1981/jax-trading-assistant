package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const maxEvaluationCoverageSymbols = 25

type evaluationCoverageRequest struct {
	DecisionRuleset string `json:"decisionRuleset"`
	ResolverRuleset string `json:"resolverRuleset"`
	MaxSymbols      int    `json:"maxSymbols"`
}

type evaluationCoverageResult struct {
	Mode      string                          `json:"mode"`
	Symbols   []string                        `json:"symbols"`
	Attempts  int                             `json:"attempts"`
	Completed []genuineCandleCollectionResult `json:"completed"`
	Errors    []string                        `json:"errors,omitempty"`
}

func evaluationMarketCoverageHandler(pool *pgxpool.Pool, fetcher sourcedCandleFetcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req evaluationCoverageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid evaluation coverage request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.DecisionRuleset) == "" {
			req.DecisionRuleset = "genuine-event-decision-v2"
		}
		if strings.TrimSpace(req.ResolverRuleset) == "" {
			req.ResolverRuleset = "event-asset-resolution-v1"
		}
		if req.MaxSymbols <= 0 {
			req.MaxSymbols = maxEvaluationCoverageSymbols
		}
		if req.MaxSymbols > maxEvaluationCoverageSymbols {
			http.Error(w, "maxSymbols must not exceed 25", http.StatusBadRequest)
			return
		}
		result, err := collectEvaluationMarketCoverage(r.Context(), pool, fetcher, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		jsonOK(w, result)
	}
}

func collectEvaluationMarketCoverage(ctx context.Context, pool *pgxpool.Pool, fetcher sourcedCandleFetcher, req evaluationCoverageRequest) (evaluationCoverageResult, error) {
	rows, err := pool.Query(ctx, `SELECT r.resolved_symbol,r.benchmark_symbol,MIN(d.event_receipt_at)
		FROM event_asset_resolutions r JOIN genuine_event_decisions d ON d.id=r.decision_id
		WHERE d.ruleset_version=$1 AND d.is_initial AND d.decision_origin='historical_backfill'
		  AND r.resolver_ruleset_version=$2 AND r.resolution_status='resolved'
		GROUP BY r.resolved_symbol,r.benchmark_symbol`, req.DecisionRuleset, req.ResolverRuleset)
	if err != nil {
		return evaluationCoverageResult{}, fmt.Errorf("load accepted evaluation assets: %w", err)
	}
	defer rows.Close()
	earliest := map[string]time.Time{}
	for rows.Next() {
		var symbol string
		var benchmark *string
		var at time.Time
		if err := rows.Scan(&symbol, &benchmark, &at); err != nil {
			return evaluationCoverageResult{}, err
		}
		for _, candidate := range []string{symbol, stringValue(benchmark)} {
			candidate = strings.ToUpper(strings.TrimSpace(candidate))
			if !safeMarketSymbol.MatchString(candidate) {
				continue
			}
			if prior, ok := earliest[candidate]; !ok || at.Before(prior) {
				earliest[candidate] = at.UTC()
			}
		}
	}
	if err := rows.Err(); err != nil {
		return evaluationCoverageResult{}, err
	}
	symbols := make([]string, 0, len(earliest))
	for symbol := range earliest {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	if len(symbols) > req.MaxSymbols {
		symbols = symbols[:req.MaxSymbols]
	}
	result := evaluationCoverageResult{Mode: "paper_read_only", Symbols: symbols}
	for _, symbol := range symbols {
		from := earliest[symbol].Add(-72 * time.Hour)
		for _, item := range []struct {
			timeframe string
			limit     int
		}{{"1h", 1000}, {"1d", 400}} {
			var collected genuineCandleCollectionResult
			var collectErr error
			for attempt := 1; attempt <= 2; attempt++ {
				result.Attempts++
				attemptCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
				collected, collectErr = collectGenuineCandles(attemptCtx, pool, fetcher, symbol, item.timeframe, from, item.limit)
				cancel()
				if collectErr == nil {
					result.Completed = append(result.Completed, collected)
					break
				}
			}
			if collectErr != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: %v", symbol, item.timeframe, collectErr))
			}
		}
	}
	return result, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
