package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"jax-trading-assistant/libs/runtimepolicy"
)

type ops01WorkerHealthState struct {
	Running             bool       `json:"running"`
	Status              string     `json:"status"`
	LastAttempt         *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess         *time.Time `json:"lastSuccess,omitempty"`
	LastFailure         *time.Time `json:"lastFailure,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
}

var ops01WorkerHealth = struct {
	sync.RWMutex
	workers map[string]ops01WorkerHealthState
}{workers: map[string]ops01WorkerHealthState{}}

func ops01WorkerConfigured(name, status string) {
	ops01WorkerHealth.Lock()
	defer ops01WorkerHealth.Unlock()
	state := ops01WorkerHealth.workers[name]
	state.Status = status
	ops01WorkerHealth.workers[name] = state
}

func ops01WorkerStarted(name string) {
	ops01WorkerHealth.Lock()
	defer ops01WorkerHealth.Unlock()
	state := ops01WorkerHealth.workers[name]
	state.Running = true
	if state.Status == "" {
		state.Status = "RUNNING"
	}
	ops01WorkerHealth.workers[name] = state
}

func ops01WorkerStopped(name string) {
	ops01WorkerHealth.Lock()
	defer ops01WorkerHealth.Unlock()
	state := ops01WorkerHealth.workers[name]
	state.Running = false
	if state.Status == "RUNNING" || state.Status == "HEALTHY" {
		state.Status = "STOPPED"
	}
	ops01WorkerHealth.workers[name] = state
}

func ops01WorkerRun(name string, at time.Time, runErr error) {
	at = at.UTC()
	ops01WorkerHealth.Lock()
	defer ops01WorkerHealth.Unlock()
	state := ops01WorkerHealth.workers[name]
	state.LastAttempt = timePointer(at)
	if runErr == nil {
		state.LastSuccess = timePointer(at)
		state.LastError = ""
		state.ConsecutiveFailures = 0
		state.Status = "HEALTHY"
	} else {
		state.LastFailure = timePointer(at)
		state.LastError = runErr.Error()
		state.ConsecutiveFailures++
		state.Status = "FAILED"
	}
	ops01WorkerHealth.workers[name] = state
}

func timePointer(at time.Time) *time.Time {
	value := at.UTC()
	return &value
}

func ops01WorkerSnapshot(name string) ops01WorkerHealthState {
	ops01WorkerHealth.RLock()
	defer ops01WorkerHealth.RUnlock()
	return ops01WorkerHealth.workers[name]
}

func ops01ReadinessHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonOK(w, buildOps01Readiness(r.Context(), pool))
	}
}

func buildOps01Readiness(ctx context.Context, pool *pgxpool.Pool) map[string]any {
	checkedAt := time.Now().UTC()
	model := map[string]any{
		"checkedAt": checkedAt,
		"process":   map[string]any{"running": true, "service": "jax-trader-api"},
		"runtime": map[string]any{
			"mode":                   strings.ToUpper(runtimepolicy.CurrentMode().String()),
			"executionAuthority":     "NONE",
			"executionEnabled":       false,
			"brokerExecutionAllowed": false,
			"maximumLeverage":        1.0,
		},
	}

	databaseConnected := pool != nil && pool.Ping(ctx) == nil
	model["database"] = map[string]any{"connected": databaseConnected}

	executionEnabled, executionErr := ops01BoolEnv("EXECUTION_ENABLED", false)
	brokerAllowed, brokerErr := ops01BoolEnv("BROKER_EXECUTION_ALLOWED", false)
	maximumLeverage, leverageErr := ops01FloatEnv("MAX_LEVERAGE", 1)
	runtimeModel := model["runtime"].(map[string]any)
	runtimeModel["executionEnabled"] = executionEnabled
	runtimeModel["brokerExecutionAllowed"] = brokerAllowed
	runtimeModel["maximumLeverage"] = maximumLeverage
	if configErr := firstError(executionErr, brokerErr, leverageErr); configErr != nil {
		runtimeModel["safetyConfigurationError"] = configErr.Error()
	}

	worldMonitor := map[string]any{}
	config, configErr := loadWorldMonitorPullConfig(os.LookupEnv)
	if configErr != nil {
		worldMonitor["status"] = "CONFIG_INVALID"
		worldMonitor["configurationError"] = configErr.Error()
	} else {
		worldMonitor["enabled"] = config.Enabled
		worldMonitor["endpointIdentity"] = config.EndpointIdentity
		worldMonitor["sourceIdentity"] = config.SourceIdentity
		worldMonitor["providerContract"] = config.ProviderContract
		if !config.Enabled {
			worldMonitor["status"] = "DISABLED"
		}
		if config.Enabled && databaseConnected {
			position, metadata, err := loadWorldMonitorPullDiagnostics(ctx, pool, config.Endpoint)
			if err != nil {
				worldMonitor["status"] = "PERSISTENCE_UNAVAILABLE"
				worldMonitor["diagnosticError"] = err.Error()
			} else {
				worldMonitor["lastCommittedCursor"] = position
				for key, value := range metadata {
					worldMonitor[key] = value
				}
				if _, ok := worldMonitor["status"]; !ok {
					worldMonitor["status"] = "NOT_ATTEMPTED"
				}
			}
		}
	}
	worldMonitor["worker"] = ops01WorkerSnapshot("world_monitor")
	model["worldMonitor"] = worldMonitor

	entryWorker := ops01WorkerSnapshot("entry_worker")
	reviewWorker := ops01WorkerSnapshot("review_worker")
	model["workers"] = map[string]any{"entry": entryWorker, "review": reviewWorker}

	pendingExitRecommendations := any(nil)
	if databaseConnected {
		var pending int
		err := pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM exploratory_paper_reviews WHERE status='EXIT_RECOMMENDED'`).Scan(&pending)
		if err == nil {
			pendingExitRecommendations = pending
		} else {
			pendingExitRecommendations = map[string]any{"error": err.Error()}
		}
	}
	model["pendingExitRecommendations"] = pendingExitRecommendations

	worldHealth := ops01WorkerSnapshot("world_monitor")
	ready := databaseConnected && strings.EqualFold(runtimepolicy.CurrentMode().String(), "paper") && !executionEnabled && !brokerAllowed && leverageErr == nil && maximumLeverage <= 1 && configErr == nil && config.Enabled && worldHealth.Running && entryWorker.Running && reviewWorker.Running
	if configErr == nil && config.Enabled {
		if failures, ok := worldMonitor["consecutive_failures"].(float64); ok && failures > 0 {
			ready = false
		}
		if failures, ok := worldMonitor["consecutiveFailures"].(int); ok && failures > 0 {
			ready = false
		}
	}
	model["ready"] = ready
	if !ready {
		model["status"] = "NOT_READY"
	} else {
		model["status"] = "READY"
	}
	return model
}

func loadWorldMonitorPullDiagnostics(ctx context.Context, pool *pgxpool.Pool, endpoint string) (int64, map[string]any, error) {
	var position int64
	var raw []byte
	err := pool.QueryRow(ctx, `SELECT last_committed_position,diagnostic_metadata FROM world_monitor_pull_cursors WHERE consumer_name=$1 AND source_endpoint_identity=$2`, worldMonitorPullConsumer, endpoint).Scan(&position, &raw)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, map[string]any{}, nil
		}
		return 0, nil, err
	}
	metadata := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return 0, nil, fmt.Errorf("decode World Monitor pull diagnostics: %w", err)
		}
	}
	return position, metadata, nil
}

func ops01BoolEnv(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseBool(raw)
}

func ops01FloatEnv(key string, fallback float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func firstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
