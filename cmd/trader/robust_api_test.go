package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRobustPerformanceHandlerReturnsReadOnlyMetrics(t *testing.T) {
	pool := testFrontendAPIPool(t)
	ensureRobustAPISchema(t, pool)

	candidateID := uuid.NewString()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO trade_reviews (
			candidate_id, symbol, strategy_key, entry_price, exit_price, stop_price,
			target_price, mfe_r, mae_r, final_r, outcome
		) VALUES (
			$1::uuid, 'QQQ', 'cpi_rates_shock', 100, 103, 98,
			104, 2.5, -1.0, 1.5, 'win'
		)
	`, candidateID)
	if err != nil {
		t.Fatalf("insert trade review: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(t.Context(), `DELETE FROM trade_reviews WHERE candidate_id=$1::uuid`, candidateID)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/robust/performance", nil)
	rec := httptest.NewRecorder()
	robustPerformanceHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var payload robustPerformanceDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode performance response: %v", err)
	}
	if len(payload.Strategies) == 0 {
		t.Fatal("expected strategy performance rows")
	}
	if payload.Strategies[0].StrategyKey == "" {
		t.Fatalf("strategy row missing key: %#v", payload.Strategies[0])
	}
}

func TestRobustPerformanceHandlerIsReadOnly(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/robust/performance", nil)
	rec := httptest.NewRecorder()
	robustPerformanceHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func ensureRobustAPISchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	up, err := os.ReadFile("../../db/postgres/migrations/000042_robust_profitability_layer.up.sql")
	if err != nil {
		t.Fatalf("read robust migration: %v", err)
	}
	if _, err := pool.Exec(t.Context(), string(up)); err != nil {
		t.Fatalf("apply robust migration: %v", err)
	}
}
