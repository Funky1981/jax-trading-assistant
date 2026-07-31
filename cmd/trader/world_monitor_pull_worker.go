package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"jax-trading-assistant/internal/modules/assetresolution"
	"jax-trading-assistant/internal/modules/eventdecisions"
	"jax-trading-assistant/internal/modules/instruments"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const worldMonitorPullConsumer = "jax-genuine-event-pull-v1"

type worldMonitorPullConfig struct {
	Enabled  bool
	Endpoint string
	Interval time.Duration
	Timeout  time.Duration
	PageSize int
}

type worldMonitorPullEvent struct {
	EventID          string         `json:"event_id"`
	PersistenceSeq   string         `json:"persistence_sequence"`
	SourceID         string         `json:"source_id"`
	SourceName       string         `json:"source_name"`
	FeedURL          string         `json:"feed_url"`
	ArticleURL       *string        `json:"article_url"`
	SourceNativeID   *string        `json:"source_native_id"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	PublicationTime  *time.Time     `json:"publication_time"`
	SourceTimestamp  *time.Time     `json:"source_timestamp"`
	CollectedAt      time.Time      `json:"collected_at"`
	FirstSeenAt      time.Time      `json:"first_seen_at"`
	LastSeenAt       time.Time      `json:"last_seen_at"`
	ContentHash      string         `json:"content_hash"`
	RawSourcePayload map[string]any `json:"raw_source_payload"`
	SchemaVersion    int            `json:"schema_version"`
	Provenance       map[string]any `json:"provenance"`
}

type worldMonitorPullPage struct {
	Events     []worldMonitorPullEvent `json:"events"`
	NextCursor string                  `json:"next_cursor"`
	Count      int                     `json:"count"`
}

type worldMonitorPullWorker struct {
	pool       *pgxpool.Pool
	config     worldMonitorPullConfig
	httpClient *http.Client
	replayer   eventdecisions.Replayer
	now        func() time.Time
}

type worldMonitorPullResult struct {
	Cursor           int64
	Fetched          int
	Ingested         int
	Duplicates       int
	DecisionsCreated int
	DecisionsReused  int
}

func loadWorldMonitorPullConfig(lookup func(string) (string, bool)) (worldMonitorPullConfig, error) {
	value := func(key string) string { raw, _ := lookup(key); return strings.TrimSpace(raw) }
	enabled, err := strconv.ParseBool(worldMonitorDefaultString(value("WORLD_MONITOR_PULL_ENABLED"), "false"))
	if err != nil {
		return worldMonitorPullConfig{}, fmt.Errorf("WORLD_MONITOR_PULL_ENABLED must be true or false")
	}
	config := worldMonitorPullConfig{Enabled: enabled}
	if !enabled {
		return config, nil
	}
	config.Endpoint = strings.TrimRight(value("WORLD_MONITOR_EVENTS_URL"), "/")
	parsed, err := url.Parse(config.Endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return worldMonitorPullConfig{}, fmt.Errorf("WORLD_MONITOR_EVENTS_URL must be an absolute HTTP(S) URL")
	}
	intervalSeconds, err := boundedEnvironmentInt(value("WORLD_MONITOR_PULL_INTERVAL_SECONDS"), 30, 5, 3600)
	if err != nil {
		return worldMonitorPullConfig{}, err
	}
	timeoutSeconds, err := boundedEnvironmentInt(value("WORLD_MONITOR_PULL_TIMEOUT_SECONDS"), 10, 1, 30)
	if err != nil {
		return worldMonitorPullConfig{}, err
	}
	config.PageSize, err = boundedEnvironmentInt(value("WORLD_MONITOR_PULL_PAGE_SIZE"), 100, 1, 250)
	if err != nil {
		return worldMonitorPullConfig{}, err
	}
	config.Interval = time.Duration(intervalSeconds) * time.Second
	config.Timeout = time.Duration(timeoutSeconds) * time.Second
	return config, nil
}

func worldMonitorDefaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func boundedEnvironmentInt(raw string, fallback, minimum, maximum int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("configuration value %q must be an integer between %d and %d", raw, minimum, maximum)
	}
	return value, nil
}

func startWorldMonitorPullWorker(ctx context.Context, pool *pgxpool.Pool) {
	config, err := loadWorldMonitorPullConfig(os.LookupEnv)
	if err != nil {
		log.Printf("world monitor pull worker disabled: invalid configuration: %v", err)
		return
	}
	if !config.Enabled {
		log.Printf("world monitor pull worker disabled")
		return
	}
	if _, err := eventdecisions.ReadEnvironmentSafetyState(); err != nil {
		log.Printf("world monitor pull worker failed closed: %v", err)
		return
	}
	rules, err := eventdecisions.LoadRuleset("config/genuine-event-decision-v2.json")
	if err != nil {
		log.Printf("world monitor pull worker failed closed: %v", err)
		return
	}
	catalog, err := instruments.LoadDefaultCatalog()
	if err != nil {
		log.Printf("world monitor pull worker failed closed: %v", err)
		return
	}
	assetRules, err := assetresolution.LoadRuleset("config/event-asset-resolution-v1.json")
	if err != nil {
		log.Printf("world monitor pull worker failed closed: %v", err)
		return
	}
	resolver := assetresolution.Resolver{Rules: assetRules}
	worker := &worldMonitorPullWorker{
		pool: pool, config: config, httpClient: &http.Client{Timeout: config.Timeout},
		replayer: eventdecisions.Replayer{Store: eventdecisions.NewStore(pool), Evaluator: eventdecisions.Evaluator{Ruleset: rules, Catalog: catalog}, Resolver: &resolver, Origin: eventdecisions.DecisionOriginLive},
		now:      func() time.Time { return time.Now().UTC() },
	}
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		result, err := worker.cycle(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("world_monitor_pull cycle failed endpoint=%q error=%q", config.Endpoint, err)
			}
		} else {
			log.Printf("world_monitor_pull cycle committed cursor=%d fetched=%d ingested=%d duplicates=%d decisions_created=%d decisions_reused=%d", result.Cursor, result.Fetched, result.Ingested, result.Duplicates, result.DecisionsCreated, result.DecisionsReused)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *worldMonitorPullWorker) cycle(ctx context.Context) (worldMonitorPullResult, error) {
	position, err := loadWorldMonitorCursor(ctx, w.pool, w.config.Endpoint, false)
	if err != nil {
		return worldMonitorPullResult{}, err
	}
	page, err := w.fetchPage(ctx, position)
	if err != nil {
		return worldMonitorPullResult{Cursor: position}, err
	}
	result := worldMonitorPullResult{Cursor: position, Fetched: len(page.Events)}
	if len(page.Events) == 0 {
		return result, nil
	}

	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return result, fmt.Errorf("begin World Monitor pull page: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, worldMonitorPullConsumer+":"+w.config.Endpoint); err != nil {
		return result, fmt.Errorf("lock World Monitor pull cursor: %w", err)
	}
	lockedPosition, err := loadWorldMonitorCursor(ctx, tx, w.config.Endpoint, true)
	if err != nil {
		return result, err
	}
	if lockedPosition != position {
		return result, fmt.Errorf("World Monitor cursor changed concurrently from %d to %d", position, lockedPosition)
	}

	inbox := newWorldMonitorResearchInboxService(w.pool)
	decisionStore := eventdecisions.NewStore(w.pool)
	for _, item := range page.Events {
		trigger, err := worldMonitorPullTrigger(item)
		if err != nil {
			return result, err
		}
		receipt, err := inbox.IngestTx(ctx, tx, trigger)
		if err != nil {
			return result, fmt.Errorf("ingest World Monitor sequence %s: %w", item.PersistenceSeq, err)
		}
		if receipt.Status == worldMonitorInboxStatusRejected || receipt.InboxID == "" {
			return result, fmt.Errorf("World Monitor sequence %s rejected: %s", item.PersistenceSeq, receipt.RejectionReason)
		}
		if receipt.Duplicate {
			result.Duplicates++
		} else {
			result.Ingested++
		}
		events, err := decisionStore.LoadSelectedEventsTx(ctx, tx, receipt.InboxID, 1)
		if err != nil {
			return result, fmt.Errorf("load ingested World Monitor sequence %s for decision: %w", item.PersistenceSeq, err)
		}
		if len(events) != 1 {
			return result, fmt.Errorf("load ingested World Monitor sequence %s for decision: rows=%d", item.PersistenceSeq, len(events))
		}
		summary, err := w.replayer.RunTx(ctx, tx, events)
		if err != nil {
			return result, fmt.Errorf("decide World Monitor sequence %s: %w", item.PersistenceSeq, err)
		}
		if summary.Eligible != 1 {
			return result, fmt.Errorf("decide World Monitor sequence %s: eligible=%d", item.PersistenceSeq, summary.Eligible)
		}
		result.DecisionsCreated += summary.DecisionsCreated
		result.DecisionsReused += summary.DecisionsReused
	}

	finalPosition, _ := strconv.ParseInt(page.NextCursor, 10, 64)
	metadata, _ := json.Marshal(map[string]any{"page_size": len(page.Events), "committed_at": w.now().UTC(), "first_sequence": page.Events[0].PersistenceSeq, "last_sequence": page.Events[len(page.Events)-1].PersistenceSeq})
	if _, err := tx.Exec(ctx, `
		INSERT INTO world_monitor_pull_cursors (consumer_name,source_endpoint_identity,last_committed_position,diagnostic_metadata)
		VALUES ($1,$2,$3,$4::jsonb)
		ON CONFLICT (consumer_name,source_endpoint_identity) DO UPDATE SET
			last_committed_position=EXCLUDED.last_committed_position,
			diagnostic_metadata=EXCLUDED.diagnostic_metadata,
			updated_at=NOW()
	`, worldMonitorPullConsumer, w.config.Endpoint, finalPosition, metadata); err != nil {
		return result, fmt.Errorf("persist World Monitor cursor: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit World Monitor pull page: %w", err)
	}
	result.Cursor = finalPosition
	return result, nil
}

type worldMonitorCursorDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadWorldMonitorCursor(ctx context.Context, db worldMonitorCursorDB, endpoint string, forUpdate bool) (int64, error) {
	query := `SELECT last_committed_position FROM world_monitor_pull_cursors WHERE consumer_name=$1 AND source_endpoint_identity=$2`
	if forUpdate {
		query += ` FOR UPDATE`
	}
	var position int64
	err := db.QueryRow(ctx, query, worldMonitorPullConsumer, endpoint).Scan(&position)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load World Monitor pull cursor: %w", err)
	}
	return position, nil
}

func (w *worldMonitorPullWorker) fetchPage(ctx context.Context, position int64) (worldMonitorPullPage, error) {
	requestURL, _ := url.Parse(w.config.Endpoint)
	query := requestURL.Query()
	query.Set("after", strconv.FormatInt(position, 10))
	query.Set("limit", strconv.Itoa(w.config.PageSize))
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return worldMonitorPullPage{}, fmt.Errorf("build World Monitor request: %w", err)
	}
	response, err := w.httpClient.Do(req)
	if err != nil {
		return worldMonitorPullPage{}, fmt.Errorf("fetch World Monitor page: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return worldMonitorPullPage{}, fmt.Errorf("read World Monitor page: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return worldMonitorPullPage{}, fmt.Errorf("World Monitor returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body[:min(len(body), 512)])))
	}
	var page worldMonitorPullPage
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&page); err != nil {
		return worldMonitorPullPage{}, fmt.Errorf("decode World Monitor page: %w", err)
	}
	if err := validateWorldMonitorPage(page, position, w.config.PageSize); err != nil {
		return worldMonitorPullPage{}, err
	}
	return page, nil
}

func validateWorldMonitorPage(page worldMonitorPullPage, after int64, pageSize int) error {
	if page.Count != len(page.Events) || len(page.Events) > pageSize {
		return fmt.Errorf("invalid World Monitor page count")
	}
	previous := after
	for _, event := range page.Events {
		sequence, err := strconv.ParseInt(event.PersistenceSeq, 10, 64)
		if err != nil || sequence <= previous || strings.TrimSpace(event.EventID) == "" {
			return fmt.Errorf("invalid or non-monotonic World Monitor persistence sequence %q", event.PersistenceSeq)
		}
		previous = sequence
	}
	next, err := strconv.ParseInt(page.NextCursor, 10, 64)
	if err != nil || next < after || (len(page.Events) > 0 && next != previous) || (len(page.Events) == 0 && next != after) {
		return fmt.Errorf("invalid World Monitor next cursor %q", page.NextCursor)
	}
	return nil
}

func worldMonitorPullTrigger(item worldMonitorPullEvent) (worldMonitorResearchTrigger, error) {
	if item.CollectedAt.IsZero() || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.EventID) == "" {
		return worldMonitorResearchTrigger{}, fmt.Errorf("World Monitor event %q is missing required persisted fields", item.EventID)
	}
	publication := item.CollectedAt
	publicationSupplied := false
	if item.PublicationTime != nil {
		publication = item.PublicationTime.UTC()
		publicationSupplied = true
	}
	sourceURL := item.FeedURL
	if item.ArticleURL != nil && strings.TrimSpace(*item.ArticleURL) != "" {
		sourceURL = strings.TrimSpace(*item.ArticleURL)
	}
	eventType, _ := item.Provenance["event_type"].(string)
	if !allowedWorldMonitorEventType(eventType) {
		eventType = "unknown"
	}
	isSynthetic := false
	raw := map[string]any{
		"kind": "world_monitor_persisted_rss_item", "world_monitor_event_id": item.EventID,
		"persistence_sequence": item.PersistenceSeq, "source_id": item.SourceID, "source_name": item.SourceName,
		"feed_url": item.FeedURL, "article_url": item.ArticleURL, "source_native_id": item.SourceNativeID,
		"publication_time_supplied": publicationSupplied, "publication_time": item.PublicationTime,
		"source_timestamp": item.SourceTimestamp, "first_seen_at": item.FirstSeenAt, "last_seen_at": item.LastSeenAt,
		"content_hash": item.ContentHash, "schema_version": item.SchemaVersion, "provenance": item.Provenance,
		"raw_source_payload": item.RawSourcePayload,
	}
	collected := item.CollectedAt.UTC()
	return worldMonitorResearchTrigger{
		Source: "world-monitor", SourceEventID: item.EventID, EventType: eventType, Headline: item.Title,
		Summary: item.Summary, SourceURLs: []string{sourceURL}, SourceCount: 1, TimestampUTC: publication,
		PossibleAffectedETFs: []string{}, AssetThemes: []string{}, Severity: "medium", SourceTier: "tier1",
		Confidence: 0.5, ConfidenceReasons: []string{"Persisted by the continuous World Monitor RSS/Atom collector"},
		Reason:     "Deterministic World Monitor pull ingestion; asset mapping remains unknown unless supplied by genuine evidence.",
		RawPayload: raw, IsSynthetic: &isSynthetic, CollectionTimestamp: &collected, DiscoveryMethod: "rss",
		DeterministicAnalysis: "world-monitor-pull-v1", AllowStalePublication: true, AllowNewsTradeLanguage: true,
	}, nil
}
