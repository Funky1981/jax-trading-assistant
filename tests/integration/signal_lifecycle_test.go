//go:build integration
// +build integration

// signal_lifecycle_test.go verifies the full lifecycle of a strategy signal:
//
//	DB insert → GET /api/v1/signals → POST approve → verify DB state
//
// Requirements:
//   - A running jax-trader instance (default: http://localhost:8081)
//   - A reachable PostgreSQL database
//
// Environment variables:
//
//	TEST_TRADER_URL    – override trader API base URL (default: http://localhost:8081)
//	TEST_DATABASE_URL  – override PG DSN (default: postgresql://jax:jax@localhost:5433/jax)
//	SKIP_INTEGRATION   – set to "1" to skip all integration tests
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	defaultTraderURL = "http://localhost:8081"
	defaultDBURL     = "postgresql://jax:jax@localhost:5433/jax"
)

func traderURL() string {
	if u := os.Getenv("TEST_TRADER_URL"); u != "" {
		return u
	}
	return defaultTraderURL
}

func databaseURL() string {
	if u := os.Getenv("TEST_DATABASE_URL"); u != "" {
		return u
	}
	return defaultDBURL
}

// TestSignalLifecycle exercises the full approved-signal path end-to-end.
func TestSignalLifecycle(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") == "1" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION=1)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── 1. Connect to database ─────────────────────────────────────────────────
	conn, err := pgx.Connect(ctx, databaseURL())
	if err != nil {
		t.Fatalf("connect to DB: %v (set TEST_DATABASE_URL or ensure docker-compose is up)", err)
	}
	defer conn.Close(ctx)

	// ── 2. Insert a test signal directly into strategy_signals ─────────────────
	testTag := fmt.Sprintf("inttest_%d", time.Now().UnixNano())
	var signalID string

	err = conn.QueryRow(ctx, `
		INSERT INTO strategy_signals
			(strategy_id, symbol, signal_type, confidence, status, entry_price, reasoning, generated_at, expires_at)
		VALUES
			($1, $2, $3, $4, 'pending', $5, $6, NOW(), NOW() + INTERVAL '1 hour')
		RETURNING id::text`,
		"test-strategy",
		"AAPL",
		"BUY",
		0.85,
		190.00,
		testTag,
	).Scan(&signalID)
	if err != nil {
		t.Fatalf("insert test signal: %v", err)
	}
	t.Logf("inserted test signal id=%s tag=%s", signalID, testTag)

	// Cleanup — always try to remove the test signal.
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn.Exec(cleanCtx, `DELETE FROM strategy_signals WHERE id = $1`, signalID) //nolint:errcheck
	})

	// ── 3. GET /api/v1/signals?status=pending – signal should appear ────────────
	authHeader := integrationAuthHeader(ctx, t)
	listURL := fmt.Sprintf("%s/api/v1/signals?status=pending", traderURL())
	listResp, err := doGET(ctx, listURL, authHeader)
	if err != nil {
		t.Fatalf("GET signals list: %v (is jax-trader running at %s?)", err, traderURL())
	}
	defer listResp.Body.Close()

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/signals status = %d; want 200", listResp.StatusCode)
	}

	var listPayload struct {
		Signals []map[string]any `json:"signals"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode signals list: %v", err)
	}

	found := false
	for _, s := range listPayload.Signals {
		if id, _ := s["id"].(string); id == signalID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("inserted signal %s not found in pending list (got %d signals)", signalID, len(listPayload.Signals))
	}

	// ── 4. POST /api/v1/signals/{id}/approve ──────────────────────────────────
	approveURL := fmt.Sprintf("%s/api/v1/signals/%s/approve", traderURL(), signalID)
	approveBody, _ := json.Marshal(map[string]any{
		"approved_by": "integration-test",
	})
	approveResp, err := doPOST(ctx, approveURL, approveBody, authHeader)
	if err != nil {
		t.Fatalf("POST approve: %v", err)
	}
	defer approveResp.Body.Close()

	if approveResp.StatusCode != http.StatusOK {
		t.Fatalf("POST approve status = %d; want 200", approveResp.StatusCode)
	}
	t.Logf("signal %s approved via API", signalID)

	// ── 5. GET /api/v1/signals/{id} – verify status is now approved ─────────────
	detailURL := fmt.Sprintf("%s/api/v1/signals/%s", traderURL(), signalID)
	detailResp, err := doGET(ctx, detailURL, authHeader)
	if err != nil {
		t.Fatalf("GET signal detail: %v", err)
	}
	defer detailResp.Body.Close()

	if detailResp.StatusCode != http.StatusOK {
		t.Fatalf("GET signal detail status = %d; want 200", detailResp.StatusCode)
	}

	var detailPayload map[string]any
	if err := json.NewDecoder(detailResp.Body).Decode(&detailPayload); err != nil {
		t.Fatalf("decode signal detail: %v", err)
	}

	apiStatus, _ := detailPayload["status"].(string)
	if apiStatus != "approved" {
		t.Errorf("API signal status = %q; want \"approved\"", apiStatus)
	}

	// ── 6. Confirm directly in DB ──────────────────────────────────────────────
	var dbStatus string
	err = conn.QueryRow(ctx,
		`SELECT status FROM strategy_signals WHERE id = $1`, signalID,
	).Scan(&dbStatus)
	if err != nil {
		t.Fatalf("read signal from DB: %v", err)
	}
	if dbStatus != "approved" {
		t.Errorf("DB signal status = %q; want \"approved\"", dbStatus)
	}
	t.Logf("✓ signal %s lifecycle verified: pending → approved", signalID)
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func integrationAuthHeader(ctx context.Context, t *testing.T) string {
	t.Helper()

	statusResp, err := doGET(ctx, traderURL()+"/auth/status", "")
	if err != nil {
		t.Fatalf("GET auth status: %v (is jax-trader running at %s?)", err, traderURL())
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /auth/status status = %d; want 200", statusResp.StatusCode)
	}
	var status struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatalf("decode auth status: %v", err)
	}
	if !status.Enabled {
		return ""
	}

	if token := strings.TrimSpace(os.Getenv("TEST_AUTH_TOKEN")); token != "" {
		return "Bearer " + token
	}
	username := strings.TrimSpace(os.Getenv("TEST_AUTH_USERNAME"))
	password := strings.TrimSpace(os.Getenv("TEST_AUTH_PASSWORD"))
	if username == "" || password == "" {
		t.Skip("trader API auth is enabled; set TEST_AUTH_TOKEN or TEST_AUTH_USERNAME/TEST_AUTH_PASSWORD")
	}

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	loginResp, err := doPOST(ctx, traderURL()+"/auth/login", body, "")
	if err != nil {
		t.Fatalf("POST auth login: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /auth/login status = %d; want 200", loginResp.StatusCode)
	}
	var login struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&login); err != nil {
		t.Fatalf("decode auth login: %v", err)
	}
	if login.AccessToken == "" {
		t.Fatal("auth login returned empty access token")
	}
	if strings.EqualFold(login.TokenType, "bearer") || login.TokenType == "" {
		return "Bearer " + login.AccessToken
	}
	return login.TokenType + " " + login.AccessToken
}

func doGET(ctx context.Context, url string, authHeader string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return http.DefaultClient.Do(req)
}

func doPOST(ctx context.Context, url string, body []byte, authHeader string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return http.DefaultClient.Do(req)
}
