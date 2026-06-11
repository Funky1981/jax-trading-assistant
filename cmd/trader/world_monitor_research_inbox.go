package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	worldMonitorInboxStatusNew              = "new"
	worldMonitorInboxStatusIgnored          = "ignored"
	worldMonitorInboxStatusCandidateCreated = "candidate_created"
	worldMonitorInboxStatusRejected         = "rejected"
)

type worldMonitorResearchInboxService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func newWorldMonitorResearchInboxService(pool *pgxpool.Pool) *worldMonitorResearchInboxService {
	return &worldMonitorResearchInboxService{
		pool: pool,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *worldMonitorResearchInboxService) Validate(trigger worldMonitorResearchTrigger, now time.Time) worldMonitorValidationResult {
	return validateWorldMonitorResearchTrigger(trigger, now)
}

func (s *worldMonitorResearchInboxService) Ingest(ctx context.Context, trigger worldMonitorResearchTrigger) (worldMonitorResearchReceipt, error) {
	trigger = normalizeWorldMonitorResearchTrigger(trigger)
	validation := s.Validate(trigger, s.now())
	if !validation.Valid {
		receipt := worldMonitorResearchReceipt{
			Status:          worldMonitorInboxStatusRejected,
			RejectionReason: validation.Reason,
		}
		if s.pool == nil {
			return receipt, nil
		}
		inboxID, err := s.insertInboxRow(ctx, trigger, worldMonitorInboxStatusRejected, validation.Reason, "")
		if err != nil {
			return worldMonitorResearchReceipt{}, err
		}
		receipt.InboxID = inboxID
		return receipt, nil
	}

	if s.pool == nil {
		return worldMonitorResearchReceipt{Status: s.statusForAcceptedTrigger(trigger)}, nil
	}

	existing, found, err := s.findExistingReceipt(ctx, trigger)
	if err != nil {
		return worldMonitorResearchReceipt{}, err
	}
	if found {
		existing.Duplicate = true
		return existing, nil
	}

	status := s.statusForAcceptedTrigger(trigger)
	var eventID string
	if s.shouldPersistAcceptedTrigger(trigger) {
		ref, err := newEventStore(s.pool).persistEventWithRef(ctx, s.toPersistEventInput(trigger))
		if err != nil {
			return worldMonitorResearchReceipt{}, err
		}
		eventID = ref.NormalizedID
	}

	inboxID, err := s.insertInboxRow(ctx, trigger, status, "", eventID)
	if err != nil {
		if isUniqueViolation(err) {
			existing, found, lookupErr := s.findExistingReceipt(ctx, trigger)
			if lookupErr != nil {
				return worldMonitorResearchReceipt{}, lookupErr
			}
			if found {
				existing.Duplicate = true
				return existing, nil
			}
		}
		return worldMonitorResearchReceipt{}, err
	}

	return worldMonitorResearchReceipt{
		InboxID: inboxID,
		EventID: eventID,
		Status:  status,
	}, nil
}

func (s *worldMonitorResearchInboxService) rejectedReceipt(trigger worldMonitorResearchTrigger) worldMonitorResearchReceipt {
	validation := s.Validate(trigger, s.now())
	if validation.Valid {
		return worldMonitorResearchReceipt{Status: s.statusForAcceptedTrigger(trigger)}
	}
	return worldMonitorResearchReceipt{
		Status:          worldMonitorInboxStatusRejected,
		RejectionReason: validation.Reason,
	}
}

func (s *worldMonitorResearchInboxService) statusForAcceptedTrigger(trigger worldMonitorResearchTrigger) string {
	if strings.EqualFold(strings.TrimSpace(trigger.Severity), "low") {
		return worldMonitorInboxStatusIgnored
	}
	return worldMonitorInboxStatusNew
}

func (s *worldMonitorResearchInboxService) shouldPersistAcceptedTrigger(trigger worldMonitorResearchTrigger) bool {
	return !strings.EqualFold(strings.TrimSpace(trigger.Severity), "low")
}

func (s *worldMonitorResearchInboxService) toPersistEventInput(trigger worldMonitorResearchTrigger) persistEventInput {
	symbols := normalizeSymbols("", trigger.PossibleAffectedETFs)
	primary := ""
	if len(symbols) > 0 {
		primary = symbols[0]
	}
	payload := map[string]any{
		"source":                 trigger.Source,
		"source_event_id":        trigger.SourceEventID,
		"event_type":             trigger.EventType,
		"headline":               trigger.Headline,
		"summary":                trigger.Summary,
		"source_urls":            trigger.SourceURLs,
		"source_count":           trigger.SourceCount,
		"timestamp_utc":          trigger.TimestampUTC.UTC().Format(time.RFC3339),
		"region":                 trigger.Region,
		"possible_affected_etfs": trigger.PossibleAffectedETFs,
		"asset_themes":           trigger.AssetThemes,
		"severity":               trigger.Severity,
		"source_tier":            trigger.SourceTier,
		"confidence":             trigger.Confidence,
		"confidence_reasons":     trigger.ConfidenceReasons,
		"reason":                 trigger.Reason,
		"raw_payload":            trigger.RawPayload,
	}
	attrs := map[string]any{
		"eventType":           trigger.EventType,
		"sourceUrls":          trigger.SourceURLs,
		"sourceCount":         trigger.SourceCount,
		"assetThemes":         trigger.AssetThemes,
		"confidenceReasons":   trigger.ConfidenceReasons,
		"worldMonitorEventId": trigger.SourceEventID,
		"mappingReason":       trigger.Reason,
		"region":              trigger.Region,
		"sourceTier":          trigger.SourceTier,
	}
	return persistEventInput{
		SourceID:      "world-monitor",
		SourceName:    "World Monitor",
		ProviderType:  "external",
		SourceEventID: trigger.SourceEventID,
		EventKind:     "research_trigger",
		EventTime:     trigger.TimestampUTC.UTC(),
		PrimarySymbol: primary,
		Title:         strings.TrimSpace(trigger.Headline),
		Summary:       strings.TrimSpace(trigger.Summary),
		Severity:      strings.ToLower(strings.TrimSpace(trigger.Severity)),
		Confidence:    trigger.Confidence,
		Payload:       payload,
		Attributes:    attrs,
		Symbols:       symbols,
	}
}

func (s *worldMonitorResearchInboxService) findExistingReceipt(ctx context.Context, trigger worldMonitorResearchTrigger) (worldMonitorResearchReceipt, bool, error) {
	var receipt worldMonitorResearchReceipt
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(normalized_event_id::text, ''), status, COALESCE(rejection_reason, '')
		FROM world_monitor_research_inbox
		WHERE source = $1 AND (source_event_id = $2 OR dedupe_key = $3)
		ORDER BY received_at DESC
		LIMIT 1
	`, normalizedWorldMonitorSource(trigger.Source), trigger.SourceEventID, worldMonitorDedupeKey(trigger)).Scan(
		&receipt.InboxID,
		&receipt.EventID,
		&receipt.Status,
		&receipt.RejectionReason,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return worldMonitorResearchReceipt{}, false, nil
		}
		return worldMonitorResearchReceipt{}, false, fmt.Errorf("lookup world monitor inbox duplicate: %w", err)
	}
	return receipt, true, nil
}

func (s *worldMonitorResearchInboxService) insertInboxRow(ctx context.Context, trigger worldMonitorResearchTrigger, status, rejectionReason, normalizedEventID string) (string, error) {
	sourceURLs, _ := json.Marshal(trigger.SourceURLs)
	etfs, _ := json.Marshal(trigger.PossibleAffectedETFs)
	themes, _ := json.Marshal(trigger.AssetThemes)
	confidenceReasons, _ := json.Marshal(trigger.ConfidenceReasons)
	rawPayload := trigger.RawPayload
	if rawPayload == nil {
		rawPayload = map[string]any{}
	}
	rawPayloadJSON, _ := json.Marshal(rawPayload)

	inboxID := uuid.NewString()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO world_monitor_research_inbox (
			id, source, world_monitor_event_id, source_event_id, status, rejection_reason,
			event_type, headline, summary, source_urls, source_count, event_time, region,
			possible_affected_etfs, asset_themes, severity, source_tier, confidence,
			confidence_reasons, mapping_reason, dedupe_key, raw_payload, normalized_event_id
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, NULLIF($6, ''),
			$7, $8, NULLIF($9, ''), $10::jsonb, $11, $12, NULLIF($13, ''),
			$14::jsonb, $15::jsonb, $16, $17, $18,
			$19::jsonb, $20, $21, $22::jsonb, NULLIF($23, '')::uuid
		)
	`, inboxID, normalizedWorldMonitorSource(trigger.Source), trigger.SourceEventID, trigger.SourceEventID, status, rejectionReason,
		strings.ToLower(strings.TrimSpace(trigger.EventType)), strings.TrimSpace(trigger.Headline), strings.TrimSpace(trigger.Summary),
		string(sourceURLs), trigger.SourceCount, trigger.TimestampUTC.UTC(), strings.TrimSpace(trigger.Region),
		string(etfs), string(themes), strings.ToLower(strings.TrimSpace(trigger.Severity)),
		strings.ToLower(strings.TrimSpace(trigger.SourceTier)), trigger.Confidence, string(confidenceReasons),
		strings.TrimSpace(trigger.Reason), worldMonitorDedupeKey(trigger), string(rawPayloadJSON), normalizedEventID)
	if err != nil {
		return "", fmt.Errorf("insert world monitor research inbox: %w", err)
	}
	return inboxID, nil
}

func worldMonitorDedupeKey(trigger worldMonitorResearchTrigger) string {
	return deterministicEventID(
		normalizedWorldMonitorSource(trigger.Source),
		trigger.SourceEventID,
		strings.ToLower(strings.TrimSpace(trigger.Headline)),
	)
}

func normalizedWorldMonitorSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return "world-monitor"
	}
	return strings.ToLower(trimmed)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
