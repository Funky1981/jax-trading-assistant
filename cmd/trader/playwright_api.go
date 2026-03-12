package main

// playwright_api.go — API handlers for running the Playwright E2E test suite
// from the frontend UI.
//
// Endpoints (registered in codex_api.go):
//
//	POST /api/v1/e2e/run                 – run browser tests (optionally scoped
//	                                        to a single spec via ?spec= query param)
//	GET  /api/v1/e2e/results             – last run result (cached in memory)
//
// The handler shells out to `npx playwright test` from the frontend/ directory.
// It is intentionally disabled in live-trading mode (ALLOW_LIVE_TRADING=true).
// Results are kept in an in-memory struct — suitable for local dev use.

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── In-memory result store ─────────────────────────────────────────────────

type playwrightRunResult struct {
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	DurationMs  int64  `json:"durationMs"`
	ExitCode    int    `json:"exitCode"`
	Status      string `json:"status"` // "passed" | "failed" | "running"
	Spec        string `json:"spec"`   // "" means full suite
	Output      string `json:"output"`
}

var (
	pwMu      sync.RWMutex
	pwResult  *playwrightRunResult // nil = no run yet
	pwRunning bool
)

// ── Handler: POST /api/v1/e2e/run ─────────────────────────────────────────

func playwrightRunHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.EqualFold(os.Getenv("ALLOW_LIVE_TRADING"), "true") {
			http.Error(w, "e2e tests are disabled in live-trading mode", http.StatusForbidden)
			return
		}

		pwMu.Lock()
		if pwRunning {
			pwMu.Unlock()
			jsonOK(w, map[string]any{"status": "running", "message": "A run is already in progress"})
			return
		}
		pwRunning = true
		spec := r.URL.Query().Get("spec") // optional: e.g. "auth" → runs e2e/auth.spec.ts
		pwMu.Unlock()

		go func() {
			result := execPlaywright(spec)
			pwMu.Lock()
			pwResult = result
			pwRunning = false
			pwMu.Unlock()
		}()

		jsonOK(w, map[string]any{"status": "triggered", "spec": spec, "message": "Playwright run started"})
	}
}

// ── Handler: GET /api/v1/e2e/results ──────────────────────────────────────

func playwrightResultsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pwMu.RLock()
		running := pwRunning
		result := pwResult
		pwMu.RUnlock()

		if running {
			jsonOK(w, map[string]any{"status": "running"})
			return
		}
		if result == nil {
			jsonOK(w, map[string]any{"status": "idle"})
			return
		}
		jsonOK(w, result)
	}
}

// ── Subprocess execution ───────────────────────────────────────────────────

func execPlaywright(spec string) *playwrightRunResult {
	started := time.Now().UTC()

	// Resolve the frontend/ directory relative to the repo root.
	frontendDir := filepath.Join(repoRoot(), "frontend")

	// Build the command: npx playwright test [e2e/<spec>.spec.ts] --reporter=list
	args := []string{"playwright", "test", "--reporter=list"}
	if spec != "" {
		// Sanitize: spec must be a bare name like "auth", "trading" — no path separators.
		cleaned := filepath.Base(spec)
		cleaned = strings.TrimSuffix(cleaned, ".spec.ts")
		cleaned = strings.ReplaceAll(cleaned, "..", "")
		if cleaned != "" {
			args = append(args, "e2e/"+cleaned+".spec.ts")
		}
	}

	var outBuf bytes.Buffer
	cmd := exec.Command("npx", args...) //nolint:gosec // args sanitized above
	cmd.Dir = frontendDir
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err := cmd.Run()
	completed := time.Now().UTC()
	durationMs := completed.Sub(started).Milliseconds()

	exitCode := 0
	status := "passed"
	if err != nil {
		status = "failed"
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &playwrightRunResult{
		StartedAt:   started.Format(time.RFC3339),
		CompletedAt: completed.Format(time.RFC3339),
		DurationMs:  durationMs,
		ExitCode:    exitCode,
		Status:      status,
		Spec:        spec,
		Output:      truncateForArtifact(outBuf.String(), 24_000),
	}
}
