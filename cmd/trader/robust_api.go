package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type robustPerformanceDTO struct {
	Funnel     robustEventFunnelDTO      `json:"funnel"`
	Strategies []robustStrategyMetricDTO `json:"strategies"`
}

type robustEventFunnelDTO struct {
	EventsAnalyzed    int `json:"eventsAnalyzed"`
	CandidatesCreated int `json:"candidatesCreated"`
	BlockingWalkaways int `json:"blockingWalkaways"`
	ReviewedTrades    int `json:"reviewedTrades"`
}

type robustStrategyMetricDTO struct {
	StrategyKey string  `json:"strategyKey"`
	Trades      int     `json:"trades"`
	AverageR    float64 `json:"averageR"`
	WinRate     float64 `json:"winRate"`
}

func robustPerformanceHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if pool == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}

		var payload robustPerformanceDTO
		if err := pool.QueryRow(r.Context(), `
			SELECT events_analyzed, candidates_created, blocking_walkaways, reviewed_trades
			FROM robust_event_funnel_summary
		`).Scan(
			&payload.Funnel.EventsAnalyzed,
			&payload.Funnel.CandidatesCreated,
			&payload.Funnel.BlockingWalkaways,
			&payload.Funnel.ReviewedTrades,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT strategy_key, trades, COALESCE(avg_r, 0)::float8, COALESCE(win_rate, 0)::float8
			FROM strategy_performance_summary
			ORDER BY strategy_key
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		payload.Strategies = []robustStrategyMetricDTO{}
		for rows.Next() {
			var row robustStrategyMetricDTO
			if err := rows.Scan(&row.StrategyKey, &row.Trades, &row.AverageR, &row.WinRate); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			payload.Strategies = append(payload.Strategies, row)
		}
		if err := rows.Err(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, payload)
	}
}
