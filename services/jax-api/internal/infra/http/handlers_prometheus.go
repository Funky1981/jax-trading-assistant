package httpapi

import (
	"database/sql"
	"fmt"
	"net/http"
)

func (s *Server) RegisterPrometheusMetrics(db *sql.DB) {
	s.mux.HandleFunc("/metrics/prometheus", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		if db == nil {
			fmt.Fprintf(w, "# HELP jax_api_metrics_db_connected Database connection available\n")
			fmt.Fprintf(w, "# TYPE jax_api_metrics_db_connected gauge\n")
			fmt.Fprintf(w, "jax_api_metrics_db_connected 0\n")
			return
		}

		fmt.Fprintf(w, "# HELP jax_api_metrics_db_connected Database connection available\n")
		fmt.Fprintf(w, "# TYPE jax_api_metrics_db_connected gauge\n")
		fmt.Fprintf(w, "jax_api_metrics_db_connected 1\n")

		var signals, approvals, runs, trades int
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM strategy_signals").Scan(&signals)
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM trade_approvals WHERE approved = true").Scan(&approvals)
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM orchestration_runs").Scan(&runs)
		_ = db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM trades").Scan(&trades)

		fmt.Fprintf(w, "# HELP jax_api_signals_total Total strategy signals\n")
		fmt.Fprintf(w, "# TYPE jax_api_signals_total counter\n")
		fmt.Fprintf(w, "jax_api_signals_total %d\n", signals)

		fmt.Fprintf(w, "# HELP jax_api_approvals_total Total approved signals\n")
		fmt.Fprintf(w, "# TYPE jax_api_approvals_total counter\n")
		fmt.Fprintf(w, "jax_api_approvals_total %d\n", approvals)

		fmt.Fprintf(w, "# HELP jax_api_orchestration_runs_total Total orchestration runs\n")
		fmt.Fprintf(w, "# TYPE jax_api_orchestration_runs_total counter\n")
		fmt.Fprintf(w, "jax_api_orchestration_runs_total %d\n", runs)

		fmt.Fprintf(w, "# HELP jax_api_trades_total Total trades\n")
		fmt.Fprintf(w, "# TYPE jax_api_trades_total counter\n")
		fmt.Fprintf(w, "jax_api_trades_total %d\n", trades)
	})
}
