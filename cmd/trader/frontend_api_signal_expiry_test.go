package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testFrontendAPIPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgresql://jax:jax@localhost:5433/jax?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip DB-backed frontend API test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skip DB-backed frontend API test: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func insertSignalForExpiryTest(t *testing.T, pool *pgxpool.Pool, generatedAt time.Time, expiresAt *time.Time, status string) (string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	signalID := uuid.NewString()
	strategyID := fmt.Sprintf("signal-expiry-test-%d", time.Now().UnixNano())
	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_signals
			(id, symbol, strategy_id, signal_type, confidence, generated_at, expires_at, status)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)`,
		signalID,
		"AAPL",
		strategyID,
		"BUY",
		0.75,
		generatedAt.UTC(),
		expiresAt,
		status,
	)
	if err != nil {
		t.Fatalf("insert test signal: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM strategy_signals WHERE id=$1`, signalID)
	})

	return signalID, strategyID
}

func loadExpiryTestSignal(t *testing.T, pool *pgxpool.Pool, signalID string) (string, *time.Time) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var status string
	var expiresAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT status, expires_at FROM strategy_signals WHERE id=$1`, signalID).Scan(&status, &expiresAt); err != nil {
		t.Fatalf("load test signal: %v", err)
	}
	return status, expiresAt
}

func requireTimeClose(t *testing.T, got, want time.Time, tolerance time.Duration) {
	t.Helper()

	diff := got.UTC().Sub(want.UTC())
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Fatalf("time = %s, want %s within %s", got.UTC().Format(time.RFC3339Nano), want.UTC().Format(time.RFC3339Nano), tolerance)
	}
}

func TestSignalsListHandlerExpiresLegacyPendingSignals(t *testing.T) {
	pool := testFrontendAPIPool(t)
	generatedAt := time.Now().UTC().Add(-48 * time.Hour)
	signalID, strategyID := insertSignalForExpiryTest(t, pool, generatedAt, nil, "pending")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signals?status=pending&strategy="+strategyID, nil)
	rec := httptest.NewRecorder()
	signalsListHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("signals list status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Signals []Signal `json:"signals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode signals list: %v", err)
	}
	if len(payload.Signals) != 0 {
		t.Fatalf("expected stale pending signal to be excluded, got %d signals", len(payload.Signals))
	}

	status, expiresAt := loadExpiryTestSignal(t, pool, signalID)
	if status != "expired" {
		t.Fatalf("signal status = %q, want expired", status)
	}
	if expiresAt == nil {
		t.Fatal("expected expires_at to be backfilled for legacy pending signal")
	}
	expected := generatedAt.Add(24 * time.Hour)
	requireTimeClose(t, *expiresAt, expected, time.Second)
}

func TestRecommendationsListHandlerExcludesExpiredPendingSignals(t *testing.T) {
	pool := testFrontendAPIPool(t)
	generatedAt := time.Now().UTC()
	expiredAt := generatedAt.Add(-1 * time.Minute)
	signalID, _ := insertSignalForExpiryTest(t, pool, generatedAt, &expiredAt, "pending")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/recommendations?limit=20&offset=0", nil)
	rec := httptest.NewRecorder()
	recommendationsListHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("recommendations list status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Recommendations []RecommendationRow `json:"recommendations"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode recommendations list: %v", err)
	}
	for _, row := range payload.Recommendations {
		if row.Signal != nil && row.Signal.ID == signalID {
			t.Fatalf("expired pending signal %s should not appear in recommendations", signalID)
		}
	}

	status, expiresAt := loadExpiryTestSignal(t, pool, signalID)
	if status != "expired" {
		t.Fatalf("signal status = %q, want expired", status)
	}
	if expiresAt == nil {
		t.Fatal("expected expires_at to remain populated for expired signal")
	}
	requireTimeClose(t, *expiresAt, expiredAt, time.Second)
}

func TestSignalApproveRejectsExpiredPendingSignal(t *testing.T) {
	pool := testFrontendAPIPool(t)
	generatedAt := time.Now().UTC().Add(-72 * time.Hour)
	signalID, _ := insertSignalForExpiryTest(t, pool, generatedAt, nil, "pending")

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/signals/"+signalID+"/approve",
		strings.NewReader(`{"approved_by":"tester@local"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	signalApprove(rec, req, pool, signalID)

	if rec.Code != http.StatusConflict {
		t.Fatalf("approve status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(strings.ToLower(rec.Body.String()), "expired") {
		t.Fatalf("approve error body = %q, want expiry message", rec.Body.String())
	}

	status, expiresAt := loadExpiryTestSignal(t, pool, signalID)
	if status != "expired" {
		t.Fatalf("signal status = %q, want expired", status)
	}
	if expiresAt == nil {
		t.Fatal("expected expires_at to be backfilled before approve conflict")
	}
}
