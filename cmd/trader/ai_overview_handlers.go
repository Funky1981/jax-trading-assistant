package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type aiScannerSentimentState struct {
	Enabled                  bool    `json:"enabled"`
	SourceScope              string  `json:"sourceScope"`
	Window                   string  `json:"window"`
	Threshold                float64 `json:"threshold"`
	MinimumSourceCount       int     `json:"minimumSourceCount"`
	SourceTrustWeightingMode string  `json:"sourceTrustWeightingMode"`
	Mode                     string  `json:"mode"`
}

type aiScannerChannels struct {
	InApp      bool `json:"inApp"`
	DesktopWeb bool `json:"desktopWeb"`
	MobilePush bool `json:"mobilePush"`
}

type aiScannerPolicy struct {
	ManualRouteEnabled    bool   `json:"manualRouteEnabled"`
	ApprovalRouteEnabled  bool   `json:"approvalRouteEnabled"`
	BlockedReason         string `json:"blockedReason,omitempty"`
	RequiresHumanApproval bool   `json:"requiresHumanApproval"`
}

type aiScannerState struct {
	Enabled             bool                    `json:"enabled"`
	AssetScope          string                  `json:"assetScope"`
	Symbols             []string                `json:"symbols"`
	UniversePreset      string                  `json:"universePreset"`
	IntervalSeconds     int                     `json:"intervalSeconds"`
	MinimumConfidence   float64                 `json:"minimumConfidence"`
	Sentiment           aiScannerSentimentState `json:"sentiment"`
	Status              string                  `json:"status"`
	LastScanCompletedAt *time.Time              `json:"lastScanCompletedAt,omitempty"`
	NextScanAt          *time.Time              `json:"nextScanAt,omitempty"`
	Channels            aiScannerChannels       `json:"channels"`
	Policy              aiScannerPolicy         `json:"policy"`
}

type aiScannerStateStore struct {
	mu    sync.RWMutex
	state aiScannerState
}

func newAIScannerStateStore() *aiScannerStateStore {
	return &aiScannerStateStore{state: defaultAIScannerState()}
}

func (s *aiScannerStateStore) get() aiScannerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *aiScannerStateStore) set(state aiScannerState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *aiScannerStateStore) reset() {
	s.set(defaultAIScannerState())
}

var globalAIScannerStore = newAIScannerStateStore()

func defaultAIScannerState() aiScannerState {
	now := time.Now().UTC()
	next := now.Add(5 * time.Minute)
	last := now.Add(-5 * time.Minute)
	return aiScannerState{
		Enabled:           true,
		AssetScope:        "etf",
		Symbols:           []string{"SPY", "QQQ", "IWM"},
		UniversePreset:    "etf-core",
		IntervalSeconds:   300,
		MinimumConfidence: 0.7,
		Sentiment: aiScannerSentimentState{
			Enabled:                  false,
			SourceScope:              "news",
			Window:                   "24h",
			Threshold:                0.6,
			MinimumSourceCount:       3,
			SourceTrustWeightingMode: "equal",
			Mode:                     "filter",
		},
		Status:              "ready",
		LastScanCompletedAt: &last,
		NextScanAt:          &next,
		Channels: aiScannerChannels{
			InApp:      true,
			DesktopWeb: false,
			MobilePush: false,
		},
		Policy: aiScannerPolicy{
			ManualRouteEnabled:    true,
			ApprovalRouteEnabled:  true,
			RequiresHumanApproval: true,
		},
	}
}

func aiScannerHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			state, err := loadAIScannerState(r.Context(), pool)
			if err != nil {
				http.Error(w, "failed to load scanner settings", http.StatusInternalServerError)
				return
			}
			jsonOK(w, state)
		case http.MethodPut:
			var req aiScannerState
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid JSON payload", http.StatusBadRequest)
				return
			}
			if errs := validateAIScannerState(req); len(errs) > 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":   "invalid scanner settings",
					"details": errs,
				})
				return
			}
			req.Status = deriveScannerStatus(req)
			now := time.Now().UTC()
			next := now.Add(time.Duration(req.IntervalSeconds) * time.Second)
			req.LastScanCompletedAt = &now
			req.NextScanAt = &next
			if err := saveAIScannerState(r.Context(), pool, req); err != nil {
				http.Error(w, "failed to persist scanner settings", http.StatusInternalServerError)
				return
			}
			jsonOK(w, req)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func loadAIScannerState(ctx context.Context, pool *pgxpool.Pool) (aiScannerState, error) {
	if pool == nil {
		return globalAIScannerStore.get(), nil
	}

	var (
		state       aiScannerState
		symbolsRaw  []byte
		sentRaw     []byte
		channelsRaw []byte
		policyRaw   []byte
		lastScan    sql.NullTime
		nextScan    sql.NullTime
	)

	err := pool.QueryRow(ctx, `
		SELECT enabled, asset_scope, symbols, universe_preset, interval_seconds,
		       minimum_confidence, sentiment, status, last_scan_completed_at,
		       next_scan_at, channels, policy
		FROM ai_scanner_settings
		WHERE id = 1
	`).Scan(
		&state.Enabled,
		&state.AssetScope,
		&symbolsRaw,
		&state.UniversePreset,
		&state.IntervalSeconds,
		&state.MinimumConfidence,
		&sentRaw,
		&state.Status,
		&lastScan,
		&nextScan,
		&channelsRaw,
		&policyRaw,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			defaultState := defaultAIScannerState()
			if saveErr := saveAIScannerState(ctx, pool, defaultState); saveErr != nil {
				return aiScannerState{}, saveErr
			}
			return defaultState, nil
		}
		return aiScannerState{}, fmt.Errorf("query scanner settings: %w", err)
	}

	if err := json.Unmarshal(symbolsRaw, &state.Symbols); err != nil {
		return aiScannerState{}, fmt.Errorf("decode symbols: %w", err)
	}
	if err := json.Unmarshal(sentRaw, &state.Sentiment); err != nil {
		return aiScannerState{}, fmt.Errorf("decode sentiment: %w", err)
	}
	if err := json.Unmarshal(channelsRaw, &state.Channels); err != nil {
		return aiScannerState{}, fmt.Errorf("decode channels: %w", err)
	}
	if err := json.Unmarshal(policyRaw, &state.Policy); err != nil {
		return aiScannerState{}, fmt.Errorf("decode policy: %w", err)
	}

	if lastScan.Valid {
		t := lastScan.Time.UTC()
		state.LastScanCompletedAt = &t
	}
	if nextScan.Valid {
		t := nextScan.Time.UTC()
		state.NextScanAt = &t
	}

	return state, nil
}

func saveAIScannerState(ctx context.Context, pool *pgxpool.Pool, state aiScannerState) error {
	if pool == nil {
		globalAIScannerStore.set(state)
		return nil
	}

	symbolsRaw, err := json.Marshal(state.Symbols)
	if err != nil {
		return fmt.Errorf("encode symbols: %w", err)
	}
	sentRaw, err := json.Marshal(state.Sentiment)
	if err != nil {
		return fmt.Errorf("encode sentiment: %w", err)
	}
	channelsRaw, err := json.Marshal(state.Channels)
	if err != nil {
		return fmt.Errorf("encode channels: %w", err)
	}
	policyRaw, err := json.Marshal(state.Policy)
	if err != nil {
		return fmt.Errorf("encode policy: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO ai_scanner_settings (
			id, enabled, asset_scope, symbols, universe_preset, interval_seconds,
			minimum_confidence, sentiment, status, last_scan_completed_at,
			next_scan_at, channels, policy
		)
		VALUES (
			1, $1, $2, $3::jsonb, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb, $12::jsonb
		)
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			asset_scope = EXCLUDED.asset_scope,
			symbols = EXCLUDED.symbols,
			universe_preset = EXCLUDED.universe_preset,
			interval_seconds = EXCLUDED.interval_seconds,
			minimum_confidence = EXCLUDED.minimum_confidence,
			sentiment = EXCLUDED.sentiment,
			status = EXCLUDED.status,
			last_scan_completed_at = EXCLUDED.last_scan_completed_at,
			next_scan_at = EXCLUDED.next_scan_at,
			channels = EXCLUDED.channels,
			policy = EXCLUDED.policy,
			updated_at = NOW()
	`,
		state.Enabled,
		state.AssetScope,
		symbolsRaw,
		state.UniversePreset,
		state.IntervalSeconds,
		state.MinimumConfidence,
		sentRaw,
		state.Status,
		state.LastScanCompletedAt,
		state.NextScanAt,
		channelsRaw,
		policyRaw,
	)
	if err != nil {
		return fmt.Errorf("upsert scanner settings: %w", err)
	}

	return nil
}

func deriveScannerStatus(state aiScannerState) string {
	if !state.Enabled {
		return "disabled"
	}
	return "ready"
}

func validateAIScannerState(state aiScannerState) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(state.AssetScope) == "" {
		errs["assetScope"] = "assetScope is required"
	}
	if strings.EqualFold(strings.TrimSpace(state.AssetScope), "symbols") && len(state.Symbols) == 0 {
		errs["symbols"] = "symbols must include at least one symbol when assetScope is symbols"
	}
	if strings.TrimSpace(state.UniversePreset) == "" {
		errs["universePreset"] = "universePreset is required"
	}
	if state.IntervalSeconds <= 0 || state.IntervalSeconds > 86400 {
		errs["intervalSeconds"] = "intervalSeconds must be between 1 and 86400"
	}
	if state.MinimumConfidence < 0 || state.MinimumConfidence > 1 {
		errs["minimumConfidence"] = "minimumConfidence must be between 0 and 1"
	}
	if strings.TrimSpace(state.Sentiment.SourceScope) == "" {
		errs["sentiment.sourceScope"] = "sentiment.sourceScope is required"
	}
	if strings.TrimSpace(state.Sentiment.Window) == "" {
		errs["sentiment.window"] = "sentiment.window is required"
	}
	if state.Sentiment.Threshold < 0 || state.Sentiment.Threshold > 1 {
		errs["sentiment.threshold"] = "sentiment.threshold must be between 0 and 1"
	}
	if state.Sentiment.MinimumSourceCount <= 0 {
		errs["sentiment.minimumSourceCount"] = "sentiment.minimumSourceCount must be greater than 0"
	}
	if strings.TrimSpace(state.Sentiment.SourceTrustWeightingMode) == "" {
		errs["sentiment.sourceTrustWeightingMode"] = "sentiment.sourceTrustWeightingMode is required"
	}
	if strings.TrimSpace(state.Sentiment.Mode) == "" {
		errs["sentiment.mode"] = "sentiment.mode is required"
	}
	return errs
}

type aiOverviewResponse struct {
	CheckedAt         time.Time      `json:"checkedAt"`
	Scanner           aiScannerState `json:"scanner"`
	OpportunityCounts map[string]int `json:"opportunityCounts"`
	PolicySummary     map[string]any `json:"policySummary"`
	ChannelSummary    map[string]any `json:"channelSummary"`
}

func aiOverviewHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		scanner, err := loadAIScannerState(r.Context(), pool)
		if err != nil {
			log.Printf("warn: failed to load ai scanner settings for overview: %v", err)
			scanner = defaultAIScannerState()
		}
		counts := loadAIOpportunityCounts(r.Context(), pool)
		jsonOK(w, aiOverviewResponse{
			CheckedAt:         time.Now().UTC(),
			Scanner:           scanner,
			OpportunityCounts: counts,
			PolicySummary: map[string]any{
				"requiresHumanApproval": scanner.Policy.RequiresHumanApproval,
				"manualRouteEnabled":    scanner.Policy.ManualRouteEnabled,
				"approvalRouteEnabled":  scanner.Policy.ApprovalRouteEnabled,
			},
			ChannelSummary: map[string]any{
				"inApp":      scanner.Channels.InApp,
				"desktopWeb": scanner.Channels.DesktopWeb,
				"mobilePush": scanner.Channels.MobilePush,
			},
		})
	}
}

func loadAIOpportunityCounts(ctx context.Context, pool *pgxpool.Pool) map[string]int {
	counts := map[string]int{
		"signalsPending": 0,
		"candidates":     0,
		"approvals":      0,
	}
	if pool == nil {
		return counts
	}

	var signalsPending int
	var candidates int
	var approvals int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM strategy_signals WHERE status='pending'`).Scan(&signalsPending)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_trades WHERE status IN ('detected','awaiting_approval')`).Scan(&candidates)
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM candidate_trades WHERE status='awaiting_approval'`).Scan(&approvals)
	counts["signalsPending"] = signalsPending
	counts["candidates"] = candidates
	counts["approvals"] = approvals
	return counts
}
